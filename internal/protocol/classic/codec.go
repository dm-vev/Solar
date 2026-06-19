package classic

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
	sess "github.com/solar-mc/solar/internal/session"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
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
	return &Codec{
		serverName: serverName,
		motd:       motd,
		worlds:     worlds,
		players:    players,
		entities:   entities,
		commands:   commands,
		room:       sess.NewRoom[*session](),
		logger:     slog.Default(),
	}
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
		outbox:              make(chan []byte, 64),
		stop:                make(chan struct{}),
		writerDone:          make(chan struct{}),
		buildCommandContext: c.buildCommandContext,
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
	conn                  net.Conn
	reader                *bufio.Reader
	writer                *bufio.Writer
	serverName            string
	motd                  string
	worlds                *world.Manager
	players               *player.Registry
	entities              *entity.Manager
	commands              *command.Registry
	room                  *sess.Room[*session]
	logger                *slog.Logger
	worldPath             string
	policyPath            string
	workers               *worker.Pool
	outbox                chan []byte
	stop                  chan struct{}
	writerDone            chan struct{}
	stateMu               sync.RWMutex
	stopOnce              sync.Once
	connOnce              sync.Once
	username              string
	entityID              uint32
	tracked               bool
	loggedIn              bool
	joined                bool
	supportsExtPlayerList bool
	supportsTwoWayPing    bool
	supportsFastMap       bool
	buildCommandContext   func(SessionBackend) command.Context
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
			if !s.loggedIn {
				continue
			}
			if err := s.handleTwoWayPing(); err != nil {
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
