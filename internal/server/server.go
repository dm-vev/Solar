package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

// Server wires the CLI, network, and core subsystems together.
type Server struct {
	cfg       config.Config
	listener  *network.Listener
	codec     *classic.Codec
	worlds    *world.Manager
	players   *player.Registry
	entities  *entity.Manager
	store     *storage.LocalStore
	workers   *worker.Pool
	logger    *slog.Logger
	sema      chan struct{}
	pprofAddr string
	cancel    context.CancelFunc
	physics   *pluginPhysics
	playerDB  plugin.PlayerDB
}

// New creates the bootstrap server.
func New(
	cfg config.Config,
	listener *network.Listener,
	codec *classic.Codec,
	worlds *world.Manager,
	players *player.Registry,
	entities *entity.Manager,
	store *storage.LocalStore,
	workers *worker.Pool,
	logger *slog.Logger,
) *Server {
	if workers == nil {
		workers = worker.NewPool(context.Background(), 0)
	}
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Server{
		cfg:       cfg,
		listener:  listener,
		codec:     codec,
		worlds:    worlds,
		players:   players,
		entities:  entities,
		store:     store,
		workers:   workers,
		logger:    logger,
		sema:      make(chan struct{}, cfg.MaxPlayers),
		pprofAddr: "",
		physics:   newPluginPhysics(worlds),
	}
}

// SetLogger configures the server logger.
func (s *Server) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s.logger = logger
}

// SetPprofAddress configures the pprof/health HTTP server address.
// Empty string disables the debug server. Prefer a localhost address
// (e.g. "127.0.0.1:6060") to avoid exposing pprof endpoints to the network.
func (s *Server) SetPprofAddress(addr string) {
	s.pprofAddr = addr
}

// Run starts the server and blocks until ctx is canceled or a fatal error
// occurs. It guarantees graceful shutdown: background goroutines are stopped
// and state is persisted before returning.
func (s *Server) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	defer cancel()

	worldPath, policyPath, err := s.loadState()
	if err != nil {
		return err
	}

	autosaveMsg := s.cfg.Autosave.String()
	if s.cfg.Autosave <= 0 {
		autosaveMsg = "disabled"
	}
	s.logger.Info("starting server",
		"listen", s.cfg.ListenAddress,
		"workers", s.workers.Size,
		"max_players", s.cfg.MaxPlayers,
		"connect_rate", s.cfg.ConnectRate,
		"autosave", autosaveMsg,
	)

	var debugWG sync.WaitGroup
	if s.pprofAddr != "" {
		debugWG.Add(1)
		go func() {
			defer debugWG.Done()
			s.runDebugServer(ctx)
		}()
	}

	tickCtx, stopTicks := context.WithCancel(ctx)

	var tickWG sync.WaitGroup
	tickWG.Add(1)
	go func() {
		defer tickWG.Done()
		s.runTicks(tickCtx)
	}()
	if s.cfg.Autosave > 0 {
		tickWG.Add(1)
		go func() {
			defer tickWG.Done()
			s.autosaveLoop(tickCtx, worldPath, policyPath)
		}()
	}

	serveErr := s.listener.Serve(ctx, func(conn net.Conn) {
		select {
		case s.sema <- struct{}{}:
			go func() {
				defer func() { <-s.sema }()
				s.codec.ServeConn(ctx, conn)
			}()
		default:
			s.logger.Warn("connection rejected", "remote", conn.RemoteAddr().String(), "reason", "server full")
			_ = conn.Close()
		}
	})

	// Graceful shutdown: stop background goroutines, kick all players, save.
	stopTicks()
	tickWG.Wait()
	debugWG.Wait()

	s.codec.BroadcastMessage(s.codec.I18nGet("server.shutdown.msg"))
	s.codec.KickAll(s.codec.I18nGet("server.shutdown.kick"))

	if plugin.OnShutdown.HasHandlers() {
		plugin.OnShutdown.Fire(plugin.ShutdownData{Reason: "server stopping"})
	}
	plugin.UnloadAll(s.logger)

	s.saveState(worldPath, policyPath)
	s.workers.Close()

	return serveErr
}

func (s *Server) runDebugServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"status":"ok","players":%d,"entities":%d}`, s.players.Count(), s.entities.Count())
	})
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	srv := &http.Server{
		Addr:    s.pprofAddr,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		timeout := s.cfg.Debug.PprofShutdownTimeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if isNonLocalAddress(s.pprofAddr) {
		s.logger.Warn("debug server bound to a non-local address; pprof endpoints may be exposed to the network",
			"addr", s.pprofAddr)
	}

	s.logger.Info("debug server listening", "addr", s.pprofAddr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.logger.Error("debug server error", "error", err)
	}
}

// isNonLocalAddress reports whether addr is not a loopback address.
// It treats an empty or unresolvable host as non-local.
func isNonLocalAddress(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return true
	}
	if host == "" {
		return true
	}
	if strings.EqualFold(host, "localhost") {
		return false
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return true
	}
	return !ip.IsLoopback()
}

func (s *Server) runTicks(ctx context.Context) {
	interval := s.cfg.Simulation.TickInterval
	if interval <= 0 {
		interval = 50 * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.worlds.Tick()
			if s.entities != nil {
				s.entities.Tick()
			}
			s.codec.BroadcastEntityUpdates()
			s.physics.Tick()
			plugin.DefaultScheduler.Tick()
			if plugin.OnTick.HasHandlers() {
				plugin.OnTick.Fire(plugin.TickData{Tick: s.worlds.TickCount()})
			}
		}
	}
}
