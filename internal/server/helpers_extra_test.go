package server

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
)

func TestQueueAndTickBlockPhysics(t *testing.T) {
	manager := managerWithLevel("main", 4, 4, 4)
	manager.SetBlock(1, 1, 1, blocks.Sand)
	srv := &Server{
		worlds:       manager,
		blockPhysics: make(map[*world.Manager]*blocks.PhysicsEngine),
	}
	engine := srv.RegisterBlockPhysics(manager)
	engine.SetMode(blocks.ModeNormal)
	srv.QueueBlockPhysics(manager, 1, 1, 1)
	srv.QueueBlockPhysics(nil, 1, 1, 1)
	srv.tickBlockPhysics()
}

func TestServerSettersAndAddressHelpers(t *testing.T) {
	srv := &Server{}
	srv.SetLogger(nil)
	if srv.logger == nil {
		t.Fatal("SetLogger(nil) left logger nil")
	}
	srv.SetPprofAddress("127.0.0.1:6060")
	if srv.pprofAddr != "127.0.0.1:6060" {
		t.Fatalf("pprofAddr = %q", srv.pprofAddr)
	}

	cases := map[string]bool{
		"127.0.0.1:6060": false,
		"localhost:6060": false,
		"0.0.0.0:6060":   true,
		":6060":          true,
		"bad":            true,
	}
	for addr, want := range cases {
		if got := isNonLocalAddress(addr); got != want {
			t.Fatalf("isNonLocalAddress(%q) = %v, want %v", addr, got, want)
		}
	}
}

func TestRunTicksStopsOnContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	manager := managerWithLevel("main", 4, 4, 4)
	srv := &Server{
		cfg:      config.Config{Simulation: config.SimConfig{TickInterval: time.Millisecond}},
		worlds:   manager,
		entities: entity.NewManager(),
		codec:    classic.NewCodec("Solar", "MOTD", manager, nil, nil, nil),
	}
	done := make(chan struct{})
	go func() {
		srv.runTicks(ctx)
		close(done)
	}()
	time.Sleep(5 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("runTicks did not stop")
	}
	if manager.TickCount() == 0 {
		t.Fatal("runTicks did not tick world")
	}
}

func TestHeartbeatHelpers(t *testing.T) {
	if got := parseHeartbeatError(`{"errors":[["bad name"]]}`); got != "bad name" {
		t.Fatalf("parseHeartbeatError = %q", got)
	}
	if got := parseHeartbeatError(`{"errors":[]}`); got != "unknown heartbeat error" {
		t.Fatalf("parseHeartbeatError empty = %q", got)
	}
	if got := parseHeartbeatError(`not-json`); got != "not-json" {
		t.Fatalf("parseHeartbeatError invalid = %q", got)
	}
	if salt := generateSalt(); len(salt) != 16 || strings.Trim(salt, "0123456789abcdef") != "" {
		t.Fatalf("generateSalt = %q", salt)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	StartHeartbeat(ctx, HeartbeatConfig{
		Port:        25565,
		MaxPlayers:  8,
		Name:        "Solar",
		Public:      false,
		Software:    "Solar",
		OnlineCount: func() int { return 1 },
	}, slog.New(slog.NewTextHandler(io.Discard, nil)), make(chan string, 1))
	time.Sleep(5 * time.Millisecond)
}

func TestAutosaveLoopPersistsUntilContextCancel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	flushed := make(chan struct{}, 1)
	srv := &Server{
		cfg:     config.Config{Autosave: time.Millisecond},
		worlds:  managerWithLevel("main", 4, 4, 4),
		players: player.NewRegistry(),
		logger:  testLogger,
		flushBlockDBsFn: func() {
			select {
			case flushed <- struct{}{}:
			default:
			}
		},
	}
	worldPath := filepath.Join(dir, "worlds", "main.swld")
	policyPath := filepath.Join(dir, "players", "policy.json")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		srv.autosaveLoop(ctx, worldPath, policyPath)
	}()

	select {
	case <-flushed:
	case <-time.After(time.Second):
		cancel()
		t.Fatal("autosaveLoop did not save before timeout")
	}

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("autosaveLoop did not stop after context cancel")
	}
	if _, err := os.Stat(worldPath); err != nil {
		t.Fatalf("autosave world file not written: %v", err)
	}
	if _, err := os.Stat(policyPath); err != nil {
		t.Fatalf("autosave policy file not written: %v", err)
	}
}

func TestRunReturnsLoadStateError(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	worlds := world.NewManager()
	players := player.NewRegistry()
	entities := entity.NewManager()
	codec := classic.NewCodec("Solar", "Test", worlds, players, entities, nil)
	pool := worker.NewPool(context.Background(), 1)
	defer pool.Close()

	cfg := config.Config{
		ListenAddress:    "127.0.0.1:0",
		MaxPlayers:       1,
		ConnectRate:      1,
		DefaultGenerator: "MissingGenerator",
		Name:             "Solar",
		MOTD:             "Test",
		Storage: config.StorageConfig{
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
		},
	}
	srv := New(
		cfg,
		network.NewListener(cfg.ListenAddress),
		codec,
		worlds,
		players,
		entities,
		storage.NewLocalStore(dir),
		pool,
		testLogger,
	)

	err := srv.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unknown default generator") {
		t.Fatalf("Run error = %v, want unknown default generator", err)
	}
}
