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
	"github.com/solar-mc/solar/internal/i18n"
	"github.com/solar-mc/solar/internal/player"
	sess "github.com/solar-mc/solar/internal/session"
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
		color:               "&e",
		model:               "humanoid",
		allowBuild:          true,
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
	playerDB            playerdb.PlayerDB
	loginTime           time.Time
	i18n                *i18n.I18n
	blockDB             plugin.BlockDB
	playerDBID          int32
	blockDBForLevel     func(levelName string) plugin.BlockDB

	// ponytail: plugin.Player stub state, guarded by stateMu
	color      string
	model      string
	hidden     bool
	muted      bool
	frozen     bool
	afk        bool
	allowBuild bool
	lang       string
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
func (c *Codec) KickAll(reason string) {
	pkt := encodeKick(reason)
	c.room.ForEachPeerExcept(0, func(peer *session) {
		_ = peer.writePacket(pkt)
		peer.fail()
	})
}

// BroadcastMessage sends a chat message to all online players.
func (c *Codec) BroadcastMessage(msg string) {
	if plugin.OnChatSys.HasHandlers() {
		m := msg
		plugin.OnChatSys.Fire(plugin.ChatSysData{Message: &m})
		msg = m
	}
	if plugin.OnChat.HasHandlers() {
		m := msg
		plugin.OnChat.Fire(plugin.ChatData{Source: nil, Message: &m})
		msg = m
	}
	for _, p := range c.OnlinePlayers() {
		p.Message(msg)
	}
}

// OnlinePlayers returns all currently connected sessions as plugin.Player.
func (c *Codec) OnlinePlayers() []plugin.Player {
	peers := c.room.Snapshot()
	out := make([]plugin.Player, len(peers))
	for i, p := range peers {
		out[i] = p
	}
	return out
}

// FindPlayer returns the online session with the given name (case-insensitive).
func (c *Codec) FindPlayer(name string) plugin.Player {
	p, ok := c.room.FindByName(name)
	if !ok {
		return nil
	}
	return p
}

// BroadcastPacket sends a raw packet to all online players.
func (c *Codec) BroadcastPacket(packet []byte) {
	c.room.ForEachPeerExcept(0, func(peer *session) {
		_ = peer.writePacket(packet)
	})
}

// BroadcastPacketToLevel sends a raw packet only to players on the given level.
func (c *Codec) BroadcastPacketToLevel(mgr *world.Manager, packet []byte) {
	c.room.ForEachPeerExcept(0, func(peer *session) {
		if peer.CurrentWorldManager() == mgr {
			_ = peer.writePacket(packet)
		}
	})
}

// BroadcastAddEntity broadcasts an add-entity packet to all players.
func (c *Codec) BroadcastAddEntity(id byte, name string, x, y, z int, yaw, pitch byte) {
	c.BroadcastPacket(encodeAddEntity(id, name, entity.Position{X: x, Y: y, Z: z}, yaw, pitch))
}

// BroadcastRemoveEntity broadcasts a remove-entity packet to all players.
func (c *Codec) BroadcastRemoveEntity(id byte) {
	c.BroadcastPacket(encodeRemoveEntity(id))
}

// BroadcastEntityTeleport broadcasts an entity-teleport packet to all players.
func (c *Codec) BroadcastEntityTeleport(id byte, x, y, z int, yaw, pitch byte) {
	c.BroadcastPacket(encodeEntityTeleport(id, entity.Position{X: x, Y: y, Z: z}, yaw, pitch))
}

// BroadcastChangeModel broadcasts a change-model packet to all players.
func (c *Codec) BroadcastChangeModel(entityID byte, model string) {
	c.BroadcastPacket(encodeChangeModel(entityID, model))
}

// EncodeAddEntity returns an add-entity packet. Exported for callers
// that need to batch or route packets themselves (e.g. per-level).
func (c *Codec) EncodeAddEntity(id byte, name string, x, y, z int, yaw, pitch byte) []byte {
	return encodeAddEntity(id, name, entity.Position{X: x, Y: y, Z: z}, yaw, pitch)
}

// EncodeRemoveEntity returns a remove-entity packet.
func (c *Codec) EncodeRemoveEntity(id byte) []byte { return encodeRemoveEntity(id) }

// EncodeEntityTeleport returns an entity-teleport packet.
func (c *Codec) EncodeEntityTeleport(id byte, x, y, z int, yaw, pitch byte) []byte {
	return encodeEntityTeleport(id, entity.Position{X: x, Y: y, Z: z}, yaw, pitch)
}

// EncodeChangeModel returns a change-model packet.
func (c *Codec) EncodeChangeModel(entityID byte, model string) []byte {
	return encodeChangeModel(entityID, model)
}

// ChangeMap sends a different level to the player and switches their active
// world Manager. The player must be online.
func (c *Codec) ChangeMap(p plugin.Player, mgr *world.Manager) error {
	s, ok := p.(*session)
	if !ok {
		return fmt.Errorf("player is not a classic session")
	}
	return s.changeMap(mgr)
}

// PlayersOnLevel returns all online sessions whose active world Manager
// matches mgr (by pointer identity).
func (c *Codec) PlayersOnLevel(mgr *world.Manager) []plugin.Player {
	peers := c.room.Snapshot()
	var out []plugin.Player
	for _, s := range peers {
		if s.CurrentWorldManager() == mgr {
			out = append(out, s)
		}
	}
	return out
}

// PlayerWorldManager returns the active world Manager for the given player,
// or nil if the player is not found.
func (c *Codec) PlayerWorldManager(p plugin.Player) *world.Manager {
	s, ok := c.room.FindByName(p.Name())
	if !ok {
		return nil
	}
	return s.CurrentWorldManager()
}

// MainWorldManager returns the codec's default world Manager.
func (c *Codec) MainWorldManager() *world.Manager {
	return c.worlds
}

// BroadcastSetBlockToLevel sends a set-block packet to all players on the
// level whose Manager matches mgr (by pointer identity).
func (c *Codec) BroadcastSetBlockToLevel(mgr *world.Manager, x, y, z int, block byte) {
	packet := encodeSetBlock(x, y, z, block)
	c.room.ForEachPeerExcept(0, func(peer *session) {
		peer.stateMu.RLock()
		w := peer.worlds
		peer.stateMu.RUnlock()
		if w == mgr {
			_ = peer.writePacket(packet)
		}
	})
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
		dstWorld := dst.CurrentWorldManager()
		var buf []byte
		for _, ch := range changes {
			if ch.sess == dst {
				continue
			}
			// Only send entity updates to peers on the same level.
			if ch.sess.CurrentWorldManager() != dstWorld {
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
