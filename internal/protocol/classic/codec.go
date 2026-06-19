package classic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
	sess "github.com/solar-mc/solar/internal/session"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
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

// ServeConn handles a single client connection until it closes, sends bad
// data, or ctx is canceled.
func (c *Codec) ServeConn(ctx context.Context, conn net.Conn) {
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

type session struct {
	conn                net.Conn
	reader              *bufio.Reader
	writer              *bufio.Writer
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
	sendTimeoutVal      *atomic.Int64
	outboxSize          int
	writeBatchSize      int
	shutdownBatchSize   int
	tcpNoDelay          bool
	blockDefs           *blockdef.Registry
	outbox              chan []byte
	stop                chan struct{}
	writerDone          chan struct{}
	stateMu             sync.RWMutex
	stopOnce            sync.Once
	connOnce            sync.Once
	username            string
	entityID            uint32
	tracked             bool
	loggedIn            bool
	joined              bool
	lastPos             entity.Position
	lastYaw             byte
	lastPitch           byte
	cpeExts             map[string]uint32
	buildCommandContext func(SessionBackend) command.Context
}

func (s *session) RoomEntityID() uint32 {
	return s.currentEntityID()
}

func (s *session) RoomUsername() string {
	return s.currentUsername()
}

func (s *session) run() error {
	defer s.cleanup()

	for {
		if s.readDeadline > 0 {
			if err := s.conn.SetReadDeadline(time.Now().Add(s.readDeadline)); err != nil {
				return fmt.Errorf("set read deadline: %w", err)
			}
		}

		opcode, err := s.reader.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read opcode: %w", err)
		}

		switch opcode {
		case opcodeHandshake:
			if err := s.handleHandshake(); err != nil {
				return err
			}
		case opcodePing:
			continue
		case opcodeSetBlockClient:
			if !s.loggedIn {
				continue
			}
			if err := s.handleSetBlock(); err != nil {
				return err
			}
		case opcodeEntityTeleport:
			if !s.loggedIn {
				continue
			}
			if err := s.handleEntityTeleport(); err != nil {
				return err
			}
		case opcodeRelPosAndOrientation:
			if !s.loggedIn {
				continue
			}
			if err := s.handleRelativePosition(true, true); err != nil {
				return err
			}
		case opcodeRelPos:
			if !s.loggedIn {
				continue
			}
			if err := s.handleRelativePosition(true, false); err != nil {
				return err
			}
		case opcodeOrientation:
			if !s.loggedIn {
				continue
			}
			if err := s.handleRelativePosition(false, true); err != nil {
				return err
			}
		case opcodeMessage:
			if !s.loggedIn {
				continue
			}
			if err := s.handleMessage(); err != nil {
				return err
			}
		case opcodeTwoWayPing:
			if err := s.handleLoggedInPacket(opcode, s.handleTwoWayPing); err != nil {
				return err
			}
		case opcodePlayerClick, opcodePluginMessage, opcodeNotifyAction, opcodeNotifyPositionAction:
			if err := s.handleCPEPacket(opcode); err != nil {
				return err
			}
		default:
			if s.loggedIn {
				_ = s.writeKick("unsupported packet")
			}
			return fmt.Errorf("unsupported opcode %d", opcode)
		}
	}
}

func (s *session) cleanup() {
	s.leaveRoom()
	username, entityID, tracked := s.sessionIdentity()
	if s.players != nil && tracked && username != "" {
		s.players.Remove(username)
	}
	if s.entities != nil && entityID != 0 {
		s.entities.Remove(entityID)
	}
}

// BroadcastEntityUpdates runs the per-tick entity position broadcast.
// It snapshots all sessions, finds entities whose position/rotation changed
// since the last tick, and sends each recipient a single concatenated buffer.
// Uses delta encoding: small movements go as RelPos/RelPosAndOrient packets
// (which the client interpolates smoothly); large jumps use EntityTeleport.
// Modeled on MCGalaxy's BroadcastEntityPositions.
func (c *Codec) BroadcastEntityUpdates() {
	peers := c.room.Snapshot()

	if c.sendTimeoutMode == "adaptive" {
		n := len(peers)
		t := adaptiveBase + time.Duration(n)*adaptivePerPlayer
		if t > adaptiveMax {
			t = adaptiveMax
		}
		c.sendTimeoutVal.Store(int64(t))
	}

	if len(peers) < 2 {
		return
	}

	type change struct {
		sess  *session
		pkt   []byte
		pos   entity.Position
		yaw   byte
		pitch byte
	}

	changes := make([]change, 0, len(peers))
	for _, s := range peers {
		eid := s.currentEntityID()
		if eid == 0 {
			continue
		}
		snap, ok := s.entitySnapshot()
		if !ok {
			continue
		}

		lastPos, lastYaw, lastPitch := s.lastBroadcast()
		posChanged := snap.Pos != lastPos
		oriChanged := snap.Yaw != lastYaw || snap.Pitch != lastPitch
		if !posChanged && !oriChanged {
			continue
		}

		var pkt []byte
		id := byte(eid)
		dx, dy, dz := snap.Pos.X-lastPos.X, snap.Pos.Y-lastPos.Y, snap.Pos.Z-lastPos.Z

		switch {
		case posChanged && fitsRelDelta(dx) && fitsRelDelta(dy) && fitsRelDelta(dz) && oriChanged:
			pkt = encodeRelPosAndOrient(id, dx, dy, dz, snap.Yaw, snap.Pitch)
		case posChanged && fitsRelDelta(dx) && fitsRelDelta(dy) && fitsRelDelta(dz):
			pkt = encodeRelPos(id, dx, dy, dz)
		case !posChanged && oriChanged:
			pkt = encodeOrientation(id, snap.Yaw, snap.Pitch)
		default:
			pkt = encodeEntityTeleport(id, snap.Pos, snap.Yaw, snap.Pitch)
		}

		changes = append(changes, change{
			sess:  s,
			pkt:   pkt,
			pos:   snap.Pos,
			yaw:   snap.Yaw,
			pitch: snap.Pitch,
		})
	}

	if len(changes) == 0 {
		return
	}

	for _, dst := range peers {
		var buf []byte
		for _, ch := range changes {
			if ch.sess == dst {
				continue
			}
			buf = append(buf, ch.pkt...)
		}
		if len(buf) > 0 {
			_ = dst.writePacketNoCopy(buf)
		}
	}

	for _, ch := range changes {
		ch.sess.setLastBroadcast(ch.pos, ch.yaw, ch.pitch)
	}
}

type packetHandler func() error

func (s *session) handleLoggedInPacket(_ byte, handler packetHandler) error {
	if !s.loggedIn {
		return nil
	}
	return handler()
}

func (s *session) handleCPEPacket(opcode byte) error {
	if !s.loggedIn {
		return nil
	}
	switch opcode {
	case opcodePlayerClick:
		return s.handlePlayerClick()
	case opcodePluginMessage:
		return s.handlePluginMessage()
	case opcodeNotifyAction:
		return s.handleNotifyAction()
	case opcodeNotifyPositionAction:
		return s.handleNotifyPositionAction()
	default:
		return nil
	}
}
