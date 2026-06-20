// codec.go contains the Codec type: the top-level protocol coordinator.
//
// Codec is responsible for:
//   - Configuring protocol parameters (timeouts, outbox size, TCP options)
//   - Accepting incoming connections via ServeConn
//   - Creating Session instances with the correct configuration
//   - Providing broadcast helpers used by the server and plugin API
//
// Codec is safe for concurrent use. A single Codec instance is shared
// across all connections for the lifetime of the server.
package classic

import (
	"bufio"
	"context"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/solar-mc/solar/internal/antispam"
	"github.com/solar-mc/solar/internal/blockdb"
	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/i18n"
	"github.com/solar-mc/solar/internal/player"
	sess "github.com/solar-mc/solar/internal/session"
	"github.com/solar-mc/solar/internal/specialblocks"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/playerdb"
)

// Default deadlines for TCP session I/O. A read deadline protects against
// slowloris-style attacks where a client connects but never sends data; a
// write deadline protects against stuck clients that stop reading server
// output. Both can be overridden per-codec via SetConnTimeouts.
const (
	defaultReadDeadline  = 30 * time.Second
	defaultWriteDeadline = 10 * time.Second
	defaultSendTimeout   = 50 * time.Millisecond

	adaptiveBase      = 20 * time.Millisecond
	adaptivePerPlayer = 300 * time.Microsecond
	adaptiveMax       = 150 * time.Millisecond
)

// Codec owns the Classic/ClassiCube wire format.
type Codec struct {
	serverName          string
	motd                string
	worlds              *world.Manager
	players             *player.Registry
	entities            *entity.Manager
	commands            *command.Registry
	room                *sess.Room[*session]
	logger              *slog.Logger
	worldPath           string
	policyPath          string
	workers             *worker.Pool
	readDeadline        time.Duration
	writeDeadline       time.Duration
	sendTimeoutMode     string
	sendTimeoutVal      atomic.Int64
	outboxSize          int
	writeBatchSize      int
	shutdownBatchSize   int
	tcpNoDelay          bool
	blockDefs           *blockdef.Registry
	buildCommandContext func(SessionBackend) command.Context
	playerDB            playerdb.PlayerDB
	i18n                *i18n.I18n
	blockDBForLevel     func(levelName string) plugin.BlockDB
	nameConv            *blockdb.NameConverter
	gotoLevel           func(p plugin.Player, name string) bool
	mainLevelName       func() string
	loadLevel           func(name string) bool
	unloadLevel         func(name string) bool
	listLoadedLevels    func() []string
	listLevelFiles      func() []string
	queuePhysics        func(x, y, z int)
	maxPlayers          int
	spamChecker         *antispam.Checker
}

// NewCodec creates the bootstrap protocol codec.
func NewCodec(
	serverName, motd string,
	worlds *world.Manager,
	players *player.Registry,
	entities *entity.Manager,
	commands *command.Registry,
) *Codec {
	if worlds == nil {
		worlds = world.NewManager()
	}
	if players == nil {
		players = player.NewRegistry()
	}
	if entities == nil {
		entities = entity.NewManager()
	}
	if commands == nil {
		commands = command.NewRegistry()
	}
	c := &Codec{
		serverName:        serverName,
		motd:              motd,
		worlds:            worlds,
		players:           players,
		entities:          entities,
		commands:          commands,
		room:              sess.NewRoom[*session](),
		logger:            slog.Default(),
		readDeadline:      defaultReadDeadline,
		writeDeadline:     defaultWriteDeadline,
		sendTimeoutMode:   "fixed",
		outboxSize:        256,
		writeBatchSize:    32,
		shutdownBatchSize: 256,
		tcpNoDelay:        true,
	}
	c.sendTimeoutVal.Store(int64(defaultSendTimeout))
	return c
}

// SetLogger configures protocol/session logging.
func (c *Codec) SetLogger(logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	c.logger = logger
}

// SetPersistencePaths configures the default world and player policy files.
func (c *Codec) SetPersistencePaths(worldPath, policyPath string) {
	c.worldPath = worldPath
	c.policyPath = policyPath
}

// SetCommandContextBuilder configures the function that builds a
// command.Context from a SessionBackend. This decouples the protocol
// layer from the command adapter implementation.
func (c *Codec) SetCommandContextBuilder(fn func(SessionBackend) command.Context) {
	c.buildCommandContext = fn
}

// SetWorkerPool configures the background job pool for save operations.
func (c *Codec) SetWorkerPool(pool *worker.Pool) {
	c.workers = pool
}

// SetConnTimeouts configures per-session TCP read and write deadlines.
// A value of 0 disables the corresponding deadline. By default the codec
// uses defaultReadDeadline and defaultWriteDeadline.
func (c *Codec) SetConnTimeouts(read, write time.Duration) {
	c.readDeadline = read
	c.writeDeadline = write
}

// SetOutboxSize configures the per-session outbound packet queue depth.
// When the queue is full the client is disconnected.
func (c *Codec) SetOutboxSize(n int) {
	if n > 0 {
		c.outboxSize = n
	}
}

// SetWriteBatchSize configures how many packets are batched per flush
// in the write loop and during shutdown drain.
func (c *Codec) SetWriteBatchSize(batch, shutdown int) {
	if batch > 0 {
		c.writeBatchSize = batch
	}
	if shutdown > 0 {
		c.shutdownBatchSize = shutdown
	}
}

// SetTCPNoDelay controls whether TCP_NODELAY is set on accepted connections.
func (c *Codec) SetTCPNoDelay(enable bool) {
	c.tcpNoDelay = enable
}

// SetBlockDefinitions configures the custom block definition registry.
func (c *Codec) SetBlockDefinitions(reg *blockdef.Registry) {
	c.blockDefs = reg
}

// SetSendTimeout configures the send timeout mode and base duration.
// mode is "fixed" (constant timeout) or "adaptive" (scales with player count).
// In adaptive mode, timeout = min(base + players*0.3ms, 150ms).
func (c *Codec) SetSendTimeout(mode string, base time.Duration) {
	if mode == "adaptive" {
		c.sendTimeoutMode = "adaptive"
	} else {
		c.sendTimeoutMode = "fixed"
	}
	if base > 0 {
		c.sendTimeoutVal.Store(int64(base))
	}
}

// SetServerName updates the server name advertised to new connections.
func (c *Codec) SetServerName(name string) {
	c.serverName = name
}

// SetPlayerDatabase configures the persistent player database.
func (c *Codec) SetPlayerDatabase(db playerdb.PlayerDB) {
	c.playerDB = db
}

// SetI18n configures the internationalisation message store.
func (c *Codec) SetI18n(i *i18n.I18n) {
	c.i18n = i
}

// SetBlockDBLookup configures a function that returns the BlockDB for a
// given level name. Called lazily when a player's session needs to record
// a block change on their current level.
func (c *Codec) SetBlockDBLookup(fn func(levelName string) plugin.BlockDB) {
	c.blockDBForLevel = fn
}

// SetNameConverter configures the player name→integer ID converter for BlockDB.
func (c *Codec) SetNameConverter(nc *blockdb.NameConverter) {
	c.nameConv = nc
}

// SetLevelCallbacks wires multi-level operations from the server.
func (c *Codec) SetLevelCallbacks(
	gotoFn func(plugin.Player, string) bool,
	mainFn func() string,
	loadFn func(string) bool,
	unloadFn func(string) bool,
	listLoaded func() []string,
	listFiles func() []string,
) {
	c.gotoLevel = gotoFn
	c.mainLevelName = mainFn
	c.loadLevel = loadFn
	c.unloadLevel = unloadFn
	c.listLoadedLevels = listLoaded
	c.listLevelFiles = listFiles
}

// SetQueuePhysics wires the block physics queue function.
func (c *Codec) SetQueuePhysics(fn func(x, y, z int)) {
	c.queuePhysics = fn
}

// SetMaxPlayers sets the max player cap for /serverinfo.
func (c *Codec) SetMaxPlayers(n int) {
	c.maxPlayers = n
}

// SetSpamChecker configures the anti-spam rate limiter.
func (c *Codec) SetSpamChecker(sc *antispam.Checker) {
	c.spamChecker = sc
}

// StartTime records when the server started, for uptime calculation.
var StartTime time.Time

func init() {
	StartTime = time.Now()
}

// I18nGet returns a message in the server's default language.
func (c *Codec) I18nGet(key string, args ...any) string {
	if c.i18n != nil {
		return c.i18n.Get("", key, args...)
	}
	return key
}

// ServeConn handles a single client connection until it closes, sends bad
// data, or ctx is canceled.
func (c *Codec) ServeConn(ctx context.Context, conn net.Conn) {
	if plugin.OnConnectionReceived.HasHandlers() {
		ctx2 := plugin.OnConnectionReceived.Fire(plugin.ConnectionReceivedData{
			RemoteAddr: conn.RemoteAddr().String(),
		})
		if ctx2.Cancelled() {
			_ = conn.Close()
			return
		}
	}

	defer conn.Close()

	s := &session{
		conn:                conn,
		reader:              bufio.NewReader(conn),
		writer:              bufio.NewWriter(conn),
		serverName:          c.serverName,
		motd:                c.motd,
		worlds:              c.worlds,
		players:             c.players,
		entities:            c.entities,
		commands:            c.commands,
		room:                c.room,
		logger:              c.logger,
		worldPath:           c.worldPath,
		policyPath:          c.policyPath,
		workers:             c.workers,
		readDeadline:        c.readDeadline,
		writeDeadline:       c.writeDeadline,
		sendTimeoutVal:      &c.sendTimeoutVal,
		outboxSize:          c.outboxSize,
		writeBatchSize:      c.writeBatchSize,
		shutdownBatchSize:   c.shutdownBatchSize,
		tcpNoDelay:          c.tcpNoDelay,
		blockDefs:           c.blockDefs,
		outbox:              make(chan []byte, c.outboxSize),
		stop:                make(chan struct{}),
		writerDone:          make(chan struct{}),
		buildCommandContext: c.buildCommandContext,
		playerDB:            c.playerDB,
		i18n:                c.i18n,
		blockDBForLevel:     c.blockDBForLevel,
		nameConv:            c.nameConv,
		gotoLevel:           c.gotoLevel,
		mainLevelName:       c.mainLevelName,
		loadLevel:           c.loadLevel,
		unloadLevel:         c.unloadLevel,
		listLoadedLevels:    c.listLoadedLevels,
		listLevelFiles:      c.listLevelFiles,
		queuePhysics:        c.queuePhysics,
		maxPlayers:          c.maxPlayers,
		color:               "&e",
		model:               "humanoid",
		allowBuild:          true,
		specialBlocks:       specialblocks.NewRegistry(),
		spamChecker:         c.spamChecker,
	}

	if c.tcpNoDelay {
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.SetNoDelay(true)
		}
	}

	// Close the connection when ctx is canceled, unblocking the read loop.
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-s.stop:
		}
	}()

	go s.writeLoop()
	if err := s.run(); err != nil {
		c.logger.Debug("classic session closed", "remote", conn.RemoteAddr().String(), "username", s.currentUsername(), "error", err)
	}
	s.closeStop()
	<-s.writerDone
}
