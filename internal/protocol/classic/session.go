// session.go defines the Session type: per-connection state and lifecycle.
//
// A Session is created by Codec.ServeConn and owns:
//   - The TCP connection (bufio reader/writer)
//   - The packet outbox (buffered channel for async writes)
//   - Player identity (username, entity ID, tracked flag)
//   - Plugin state (color, model, frozen, muted, afk, hidden, allow_build)
//   - CPE extension support map
//   - References to world, players, entities, commands, blockDB, i18n
//
// The session runs a read loop (session.run) that dispatches incoming
// packets to handlers, and a write loop (session.writeLoop) that drains
// the outbox and flushes to the TCP connection. Both loops terminate
// when the session's stop channel is closed.
//
// All plugin.Player and plugin.CPE methods are implemented on *Session
// in plugin_player.go and cpe_plugin.go respectively.

package classic

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/i18n"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/ranks"
	sess "github.com/solar-mc/solar/internal/session"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/playerdb"
)

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
	blockDefs           *blocks.Registry
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
	playerDB            playerdb.PlayerDB
	loginTime           time.Time
	i18n                *i18n.I18n
	blockDB             plugin.BlockDB
	playerDBID          int32
	blockDBForLevel     func(levelName string) plugin.BlockDB
	nameConv            *blocks.NameConverter
	gotoLevel           func(p plugin.Player, name string) bool
	mainLevelName       func() string
	loadLevel           func(name string) bool
	unloadLevel         func(name string) bool
	listLoadedLevels    func() []string
	listLevelFiles      func() []string
	queuePhysics        func(*world.Manager, int, int, int)
	maxPlayers          int

	// ponytail: plugin.Player stub state, guarded by stateMu
	color      string
	model      string
	hidden     bool
	muted      bool
	frozen     bool
	afk        bool
	allowBuild bool
	lang       string

	// lastTeleportPos tracks the position before the last teleport
	// for /back. Updated by TeleportSelf, changeMap, and Respawn.
	lastTeleportPos   [3]int
	lastTeleportValid bool

	// ignoredPlayers tracks chat ignores for this session.
	ignoredPlayers map[string]bool

	// markState holds the active block selection for drawing commands.
	// When non-nil, block placements are intercepted as marks instead
	// of being applied to the world.
	markState *markSelection

	// clipboard holds the last /copy region for /paste.
	clipboard *blocks.CopyState

	// specialBlocks holds per-level interactive blocks (doors, portals, MBs).
	specialBlocks    *blocks.SpecialRegistry
	spamChecker      *player.SpamChecker
	undoStack        *player.UndoStack
	batchChanges     []player.BlockChange
	rankRegistry     *ranks.Registry
	authEnabled      bool
	authSalt         string
	lastSpecialBlock [3]int    // last block coords checked for special blocks
	lastAction       time.Time // last player activity (for AFK detection)
	afkSince         time.Time // when the player became AFK (for AFK kick timing)
}

// markSelection tracks a multi-click block selection for drawing commands.
type markSelection struct {
	marks    []markPos
	index    int
	callback func(marks []markPos) // called when all marks are collected
}

type markPos struct{ X, Y, Z int }

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
	// Reset anti-spam state.
	if s.spamChecker != nil {
		s.spamChecker.Reset(s.currentUsername())
	}

	// Save player props before disconnecting.
	if s.players != nil {
		s.stateMu.RLock()
		props := player.PlayerProps{
			Color:  s.color,
			Model:  s.model,
			Frozen: s.frozen,
			Muted:  s.muted,
			AFK:    s.afk,
		}
		ab := s.allowBuild
		props.AllowBuild = &ab
		s.stateMu.RUnlock()
		s.players.SetProps(s.currentUsername(), props)
	}

	// Update PlayerDB with playtime.
	if s.playerDB != nil && !s.loginTime.IsZero() {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			e.TotalTime += time.Since(s.loginTime)
			s.playerDB.Save(e)
		}
	}

	if plugin.OnPlayerDisconnect.HasHandlers() {
		plugin.OnPlayerDisconnect.Fire(plugin.PlayerDisconnectData{
			Player: s,
			Reason: "disconnected",
		})
	}
	s.leaveRoom()
	username, entityID, tracked := s.sessionIdentity()
	if s.players != nil && tracked && username != "" {
		s.players.Remove(username)
	}
	if s.entities != nil && entityID != 0 {
		s.entities.Remove(entityID)
	}
}

// ─── plugin.Player implementation on *session ───

func (s *session) Name() string { return s.currentUsername() }

func (s *session) Message(msg string) {
	if plugin.OnMessageReceived.HasHandlers() {
		m := msg
		ctx := plugin.OnMessageReceived.Fire(plugin.MessageReceivedData{
			Player:  s,
			Message: &m,
		})
		if ctx.Cancelled() {
			return
		}
		msg = m
	}
	_ = s.writePacket(encodeMessage(selfID, msg))
}

func (s *session) Teleport(x, y, z int, yaw, pitch byte) bool {
	return s.teleportSelf(x, y, z, yaw, pitch)
}

func (s *session) Kick(reason string) {
	if s.playerDB != nil {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			e.Kicks++
			s.playerDB.Save(e)
		}
	}
	s.disconnect(reason)
}

func (s *session) Position() (int, int, int) {
	eid := s.currentEntityID()
	if s.entities != nil && eid != 0 {
		e, ok := s.entities.Get(eid)
		if ok {
			return e.Pos.X, e.Pos.Y, e.Pos.Z
		}
	}
	return 0, 0, 0
}

func (s *session) SetBlock(x, y, z int, block byte) bool {
	return s.applyBlockChange(x, y, z, block, false) == nil
}

func (s *session) SupportsCPE(extName string) bool { return s.supportsExt(extName) }

// ─── Codec broadcast helpers for plugin API ───

// KickAll sends a kick packet to every online player.
