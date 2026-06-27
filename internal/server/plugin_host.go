package server

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/playerdb"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

// pluginServer implements plugin.Server for the Solar server.
type pluginServer struct {
	codec      *classic.Codec
	worlds     *world.Manager
	multiMgr   *world.MultiManager
	commands   *command.Registry
	server     *Server
	sched      plugin.Scheduler
	entityMgr  plugin.EntityManager
	playerDB   plugin.PlayerDB
	blockDBs   map[string]plugin.BlockDB
	blockDBsMu sync.Mutex
}

// NewPluginServer creates a plugin.Server handle from the server's subsystems.
func NewPluginServer(codec *classic.Codec, worlds *world.Manager, commands *command.Registry, srv *Server) *pluginServer {
	mm := world.NewMultiManager()
	mm.SetMain(srv.cfg.Storage.MainWorldName, worlds, srv.worldSavePath())

	dbPath := filepath.Join(srv.store.PlayersDir(), "playerdb.json")
	pdb, err := playerdb.New(dbPath)
	if err != nil {
		srv.logger.Error("load playerdb", "path", dbPath, "error", err)
	}
	srv.playerDB = pdb

	return &pluginServer{
		codec:     codec,
		worlds:    worlds,
		multiMgr:  mm,
		commands:  commands,
		server:    srv,
		sched:     plugin.DefaultScheduler,
		entityMgr: newEntityManager(codec, srv.entities),
		playerDB:  pdb,
		blockDBs:  make(map[string]plugin.BlockDB),
	}
}

// PostInit wires callbacks that need the pluginServer pointer.
// Called by bootstrap after NewPluginServer.
func (p *pluginServer) PostInit() {
	p.server.SetFlushBlockDBsFn(p.flushAllBlockDBs)
}

func (p *pluginServer) BroadcastMessage(msg string) {
	p.codec.BroadcastMessage(msg)
}

func (p *pluginServer) BroadcastMessageTo(scope string, source plugin.Player, msg string) {
	switch scope {
	case "all":
		p.codec.BroadcastMessage(msg)
	case "level":
		if source == nil {
			p.codec.BroadcastMessage(msg)
			return
		}
		// Find the source player's level and message only players on it.
		sourceWorld := p.codec.PlayerWorldManager(source)
		if sourceWorld == nil {
			p.codec.BroadcastMessage(msg)
			return
		}
		for _, pl := range p.codec.PlayersOnLevel(sourceWorld) {
			pl.Message(msg)
		}
	case "ops":
		for _, pl := range p.codec.OnlinePlayers() {
			if pl.IsOperator() {
				pl.Message(msg)
			}
		}
	default:
		p.codec.BroadcastMessage(msg)
	}
}

func (p *pluginServer) OnlinePlayers() []plugin.Player {
	return p.codec.OnlinePlayers()
}

func (p *pluginServer) OnlineCount() int {
	return p.server.players.Count()
}

func (p *pluginServer) MaxPlayers() int {
	return p.server.cfg.MaxPlayers
}

func (p *pluginServer) ServerName() string {
	return p.server.cfg.Name
}

func (p *pluginServer) MOTD() string {
	return p.server.cfg.MOTD
}

func (p *pluginServer) FindPlayer(name string) plugin.Player {
	return p.codec.FindPlayer(name)
}

func (p *pluginServer) World() plugin.World {
	return &pluginWorld{
		mgr:       p.worlds,
		codec:     p.codec,
		server:    p.server,
		worldPath: p.server.worldSavePath(),
	}
}

func (p *pluginServer) Levels() plugin.LevelManager {
	return &pluginLevelManager{srv: p}
}

func (p *pluginServer) ChangeMap(pl plugin.Player, levelName string) bool {
	mgr := p.multiMgr.Get(levelName)
	if mgr == nil {
		return false
	}
	return p.codec.ChangeMap(pl, mgr) == nil
}

func (p *pluginServer) Physics() plugin.Physics {
	return p.server.physics
}

func (p *pluginServer) RegisterCommand(name string, help string, handler plugin.CommandHandler) bool {
	if name == "" {
		return false
	}
	p.commands.Register(name, func(ctx command.Context, args []string) (string, bool) {
		player := p.codec.FindPlayer(ctx.Username)
		if player == nil {
			return "player not found", true
		}
		return handler(player, args), true
	})
	return true
}

func (p *pluginServer) UnregisterCommand(name string) bool {
	return p.commands.Unregister(name)
}

func (p *pluginServer) BanPlayer(name, reason string) bool {
	return p.server.players.Ban(name, reason)
}

func (p *pluginServer) UnbanPlayer(name string) bool {
	return p.server.players.Unban(name)
}

func (p *pluginServer) IsWhitelistEnabled() bool {
	return p.server.players.WhitelistEnabled()
}

func (p *pluginServer) SetWhitelistEnabled(enabled bool) {
	p.server.players.SetWhitelistEnabled(enabled)
}

func (p *pluginServer) WhitelistAdd(name string) bool {
	return p.server.players.WhitelistAdd(name)
}

func (p *pluginServer) WhitelistRemove(name string) bool {
	return p.server.players.WhitelistRemove(name)
}

func (p *pluginServer) IsOperator(name string) bool {
	return p.server.players.IsOperator(name)
}

func (p *pluginServer) AddOperators(names ...string) bool {
	return p.server.players.AddOperators(names...)
}

func (p *pluginServer) OperatorNames() []string {
	return p.server.players.OperatorNames()
}

func (p *pluginServer) SaveState() bool {
	p.server.SaveStateNow()
	return true
}

func (p *pluginServer) Scheduler() plugin.Scheduler {
	return p.sched
}

func (p *pluginServer) Stop() {
	if p.server.cancel != nil {
		p.server.cancel()
	}
}

func (p *pluginServer) Entities() plugin.EntityManager {
	return p.entityMgr
}

func (p *pluginServer) Config() plugin.Config {
	return &pluginConfig{server: p.server}
}

func (p *pluginServer) PlayerDB() plugin.PlayerDB {
	return p.playerDB
}

func (p *pluginServer) BlockDB(levelName string) plugin.BlockDB {
	p.blockDBsMu.Lock()
	defer p.blockDBsMu.Unlock()

	key := strings.ToLower(levelName)
	if db, ok := p.blockDBs[key]; ok {
		return db
	}
	mgr := p.multiMgr.Get(levelName)
	if mgr == nil {
		return nil
	}
	lvl := mgr.Current()
	path := p.server.store.BlockDBFile(levelName)
	db, err := blocks.New(path, lvl.Width, lvl.Height, lvl.Length)
	if err != nil {
		p.server.logger.Error("load blockdb", "level", levelName, "error", err)
		return nil
	}
	p.blockDBs[key] = db
	return db
}

func (p *pluginServer) flushAllBlockDBs() {
	p.blockDBsMu.Lock()
	dbs := make([]plugin.BlockDB, 0, len(p.blockDBs))
	for _, db := range p.blockDBs {
		dbs = append(dbs, db)
	}
	p.blockDBsMu.Unlock()
	for _, db := range dbs {
		if err := db.Flush(); err != nil {
			p.server.logger.Error("flush blockdb", "error", err)
		}
	}
}

// ─── Level callback helpers for codec ───

func (p *pluginServer) MainLevelName() string {
	return p.multiMgr.MainName()
}

func (p *pluginServer) LoadLevelByName(name string) bool {
	_, err := p.Levels().Load(name)
	return err == nil
}

func (p *pluginServer) UnloadLevelByName(name string) bool {
	return p.Levels().Unload(name)
}

func (p *pluginServer) ListLoadedLevels() []string {
	return p.multiMgr.Names()
}

func (p *pluginServer) ListLevelFiles() []string {
	return p.Levels().ListFiles()
}

// pluginConfig implements plugin.Config by reading/writing the server's live
// config. Most setters mutate cfg in place; SetName also syncs the codec's
// advertised server name, and operators/whitelist delegate to the player
// registry. SetMaxPlayers only takes effect on restart: the connection
// semaphore cannot be safely resized while connections are in flight.
type pluginConfig struct {
	server *Server
}

func (c *pluginConfig) fireConfigUpdated() {
	if plugin.OnConfigUpdated.HasHandlers() {
		plugin.OnConfigUpdated.Fire(plugin.ConfigUpdatedData{})
	}
}

func (c *pluginConfig) Name() string { return c.server.cfg.Name }

func (c *pluginConfig) SetName(name string) {
	c.server.cfg.Name = name
	c.server.codec.SetServerName(name)
	c.fireConfigUpdated()
}

func (c *pluginConfig) MOTD() string { return c.server.cfg.MOTD }

func (c *pluginConfig) SetMOTD(motd string) {
	c.server.cfg.MOTD = motd
	c.fireConfigUpdated()
}

func (c *pluginConfig) MaxPlayers() int { return c.server.cfg.MaxPlayers }

// SetMaxPlayers takes effect on restart; the live semaphore is not resized.
// ponytail: semaphore resize needs a coordinated swap; restart covers it.
func (c *pluginConfig) SetMaxPlayers(n int) {
	if n >= 1 && n <= int(entity.MaxClassicEntityID) {
		c.server.cfg.MaxPlayers = n
		c.fireConfigUpdated()
	}
}

func (c *pluginConfig) ConnectRate() int { return c.server.cfg.ConnectRate }

func (c *pluginConfig) SetConnectRate(rate int) {
	if rate >= 1 {
		c.server.cfg.ConnectRate = rate
		c.fireConfigUpdated()
	}
}

func (c *pluginConfig) ReadTimeout() time.Duration  { return c.server.cfg.Network.ReadTimeout }
func (c *pluginConfig) WriteTimeout() time.Duration { return c.server.cfg.Network.WriteTimeout }

func (c *pluginConfig) TCPNoDelay() bool { return c.server.cfg.Network.TCPNoDelay }

func (c *pluginConfig) SetTCPNoDelay(enable bool) {
	c.server.cfg.Network.TCPNoDelay = enable
	c.fireConfigUpdated()
}

func (c *pluginConfig) TickInterval() time.Duration { return c.server.cfg.Simulation.TickInterval }

func (c *pluginConfig) SetTickInterval(d time.Duration) {
	if d >= 0 {
		c.server.cfg.Simulation.TickInterval = d
		c.fireConfigUpdated()
	}
}

func (c *pluginConfig) DefaultWidth() int  { return c.server.cfg.World.DefaultWidth }
func (c *pluginConfig) DefaultHeight() int { return c.server.cfg.World.DefaultHeight }
func (c *pluginConfig) DefaultLength() int { return c.server.cfg.World.DefaultLength }
func (c *pluginConfig) MaxBlocks() int     { return c.server.cfg.World.MaxBlocks }

func (c *pluginConfig) AutosaveInterval() time.Duration { return c.server.cfg.Autosave }

func (c *pluginConfig) SetAutosaveInterval(d time.Duration) {
	if d >= 0 {
		c.server.cfg.Autosave = d
		c.fireConfigUpdated()
	}
}

func (c *pluginConfig) Operators() []string             { return c.server.players.OperatorNames() }
func (c *pluginConfig) AddOperator(name string) bool    { return c.server.players.AddOperators(name) }
func (c *pluginConfig) RemoveOperator(name string) bool { return c.server.players.RemoveOperator(name) }
func (c *pluginConfig) WhitelistEnabled() bool          { return c.server.players.WhitelistEnabled() }
func (c *pluginConfig) SetWhitelistEnabled(enabled bool) {
	c.server.players.SetWhitelistEnabled(enabled)
}

// pluginWorld implements plugin.World wrapping world.Manager.
// It also satisfies plugin.Level for the single-level LevelManager.
type pluginWorld struct {
	mgr       *world.Manager
	codec     *classic.Codec
	server    *Server
	worldPath string
}

func (w *pluginWorld) GetBlock(x, y, z int) (byte, bool) {
	return w.mgr.BlockAt(x, y, z)
}

func (w *pluginWorld) SetBlock(x, y, z int, block byte) bool {
	return w.mgr.SetBlock(x, y, z, block)
}

func (w *pluginWorld) Spawn() (int, int, int, byte, byte) {
	s := w.mgr.Spawn()
	return s.X, s.Y, s.Z, s.Yaw, s.Pitch
}

func (w *pluginWorld) SetSpawn(x, y, z int, yaw, pitch byte) {
	w.mgr.SetSpawn(world.Spawn{X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch})
}

func (w *pluginWorld) Dimensions() (int, int, int) {
	l := w.mgr.Current()
	return l.Width, l.Height, l.Length
}

func (w *pluginWorld) Save() error {
	if plugin.OnLevelSave.HasHandlers() {
		if ctx := plugin.OnLevelSave.Fire(plugin.LevelSaveData{}); ctx.Cancelled() {
			return nil
		}
	}
	return w.mgr.Save(w.worldPath)
}

// ─── plugin.Level methods ───

func (w *pluginWorld) Name() string {
	return w.mgr.Current().Name
}

func (w *pluginWorld) PlayerCount() int {
	if w.codec == nil || w.mgr == nil {
		return 0
	}
	return len(w.codec.PlayersOnLevel(w.mgr))
}

func (w *pluginWorld) Players() []plugin.Player {
	if w.codec == nil || w.mgr == nil {
		return nil
	}
	return w.codec.PlayersOnLevel(w.mgr)
}

// levelPath derives the on-disk path for a named level from the current
// level's directory and file extension.
func (w *pluginWorld) levelPath(name string) string {
	return filepath.Join(filepath.Dir(w.worldPath), name+filepath.Ext(w.worldPath))
}

func (w *pluginWorld) Message(msg string) {
	if w.codec == nil || w.mgr == nil {
		return
	}
	for _, p := range w.codec.PlayersOnLevel(w.mgr) {
		p.Message(msg)
	}
}

func (w *pluginWorld) Rename(newName string) error {
	if err := validateLevelName(newName); err != nil {
		return err
	}
	oldName := w.Name()
	dest := w.levelPath(newName)
	if err := renameFileSync(w.worldPath, dest); err != nil {
		return err
	}
	w.worldPath = dest
	lvl := w.mgr.Current()
	lvl.Name = newName
	w.mgr.SetCurrent(lvl)
	if plugin.OnLevelRenamed.HasHandlers() {
		plugin.OnLevelRenamed.Fire(plugin.LevelRenamedData{Source: oldName, Dest: newName})
	}
	return nil
}

func (w *pluginWorld) Copy(destName string) error {
	if err := validateLevelName(destName); err != nil {
		return err
	}
	if err := copyFile(w.worldPath, w.levelPath(destName)); err != nil {
		return err
	}
	if plugin.OnLevelCopied.HasHandlers() {
		plugin.OnLevelCopied.Fire(plugin.LevelCopiedData{Source: w.Name(), Dest: destName})
	}
	return nil
}

func (w *pluginWorld) Backup(backupName string) error {
	if err := validateLevelName(backupName); err != nil {
		return err
	}
	return copyFile(w.worldPath, w.levelPath(backupName))
}

func (w *pluginWorld) Delete() error {
	name := w.Name()
	if err := removeFileSync(w.worldPath); err != nil {
		return err
	}
	if plugin.OnLevelDeleted.HasHandlers() {
		plugin.OnLevelDeleted.Fire(plugin.LevelDeletedData{Name: name})
	}
	return nil
}

// Resize rebuilds the block array with the new dimensions, copying the
// overlapping region. ponytail: connected clients are not re-sent the world;
// a dimension change requires a reload/reconnect to update them.
func (w *pluginWorld) Resize(width, height, length int) error {
	if width < 1 || height < 1 || length < 1 {
		return fmt.Errorf("dimensions must be positive")
	}
	cur := w.mgr.Current()
	blocks := make([]byte, width*height*length)
	for y := 0; y < height && y < cur.Height; y++ {
		for z := 0; z < length && z < cur.Length; z++ {
			for x := 0; x < width && x < cur.Width; x++ {
				srcIdx := x + cur.Width*(z+cur.Length*y)
				dstIdx := x + width*(z+length*y)
				blocks[dstIdx] = cur.Blocks[srcIdx]
			}
		}
	}
	cur.Width, cur.Height, cur.Length = width, height, length
	cur.Blocks = blocks
	w.mgr.SetCurrent(cur)
	if w.server != nil {
		w.server.RegisterBlockPhysics(w.mgr)
	}
	return nil
}

func (w *pluginWorld) Reload() error {
	if err := w.mgr.Load(w.worldPath); err != nil {
		return fmt.Errorf("reload level %q: %w", w.Name(), err)
	}
	if w.server != nil {
		w.server.RegisterBlockPhysics(w.mgr)
	}
	return nil
}

// copyFile copies src to dst using a single io.Copy.
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := out.Name()
	defer func() {
		_ = out.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Sync(); err != nil {
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(dst))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func syncDir(path string) error {
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func renameFileSync(src, dst string) error {
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	return syncDir(dst)
}

func removeFileSync(path string) error {
	if err := os.Remove(path); err != nil {
		return err
	}
	return syncDir(path)
}

// pluginLevel implements plugin.Level for a loaded level in the MultiManager.
// Unlike pluginWorld (which assumes all players are on the same level),
// pluginLevel tracks players per-level via codec.PlayersOnLevel.
type pluginLevel struct {
	mgr    *world.Manager
	name   string
	path   string
	codec  *classic.Codec
	server *Server
}

func (l *pluginLevel) Name() string {
	return l.name
}

func (l *pluginLevel) GetBlock(x, y, z int) (byte, bool) {
	return l.mgr.BlockAt(x, y, z)
}

func (l *pluginLevel) SetBlock(x, y, z int, block byte) bool {
	if !l.mgr.SetBlock(x, y, z, block) {
		return false
	}
	l.codec.BroadcastSetBlockToLevel(l.mgr, x, y, z, block)
	return true
}

func (l *pluginLevel) Spawn() (int, int, int, byte, byte) {
	s := l.mgr.Spawn()
	return s.X, s.Y, s.Z, s.Yaw, s.Pitch
}

func (l *pluginLevel) SetSpawn(x, y, z int, yaw, pitch byte) {
	l.mgr.SetSpawn(world.Spawn{X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch})
}

func (l *pluginLevel) Dimensions() (int, int, int) {
	lvl := l.mgr.Current()
	return lvl.Width, lvl.Height, lvl.Length
}

func (l *pluginLevel) Save() error {
	if plugin.OnLevelSave.HasHandlers() {
		if ctx := plugin.OnLevelSave.Fire(plugin.LevelSaveData{}); ctx.Cancelled() {
			return nil
		}
	}
	return l.mgr.Save(l.path)
}

func (l *pluginLevel) PlayerCount() int {
	return len(l.codec.PlayersOnLevel(l.mgr))
}

func (l *pluginLevel) Players() []plugin.Player {
	return l.codec.PlayersOnLevel(l.mgr)
}

func (l *pluginLevel) levelPath(name string) string {
	return filepath.Join(filepath.Dir(l.path), name+filepath.Ext(l.path))
}

func (l *pluginLevel) Message(msg string) {
	for _, p := range l.codec.PlayersOnLevel(l.mgr) {
		p.Message(msg)
	}
}

func (l *pluginLevel) Rename(newName string) error {
	if err := validateLevelName(newName); err != nil {
		return err
	}
	oldName := l.name
	dest := l.levelPath(newName)
	if err := renameFileSync(l.path, dest); err != nil {
		return err
	}
	l.path = dest
	l.name = newName
	lvl := l.mgr.Current()
	lvl.Name = newName
	l.mgr.SetCurrent(lvl)
	if plugin.OnLevelRenamed.HasHandlers() {
		plugin.OnLevelRenamed.Fire(plugin.LevelRenamedData{Source: oldName, Dest: newName})
	}
	return nil
}

func (l *pluginLevel) Copy(destName string) error {
	if err := validateLevelName(destName); err != nil {
		return err
	}
	if err := copyFile(l.path, l.levelPath(destName)); err != nil {
		return err
	}
	if plugin.OnLevelCopied.HasHandlers() {
		plugin.OnLevelCopied.Fire(plugin.LevelCopiedData{Source: l.name, Dest: destName})
	}
	return nil
}

func (l *pluginLevel) Backup(backupName string) error {
	if err := validateLevelName(backupName); err != nil {
		return err
	}
	return copyFile(l.path, l.levelPath(backupName))
}

func (l *pluginLevel) Delete() error {
	name := l.name
	if err := removeFileSync(l.path); err != nil {
		return err
	}
	if plugin.OnLevelDeleted.HasHandlers() {
		plugin.OnLevelDeleted.Fire(plugin.LevelDeletedData{Name: name})
	}
	return nil
}

// ponytail: connected clients are not re-sent the world; a dimension change
// requires a reload/reconnect to update them.
func (l *pluginLevel) Resize(width, height, length int) error {
	if width < 1 || height < 1 || length < 1 {
		return fmt.Errorf("dimensions must be positive")
	}
	cur := l.mgr.Current()
	blocks := make([]byte, width*height*length)
	for y := 0; y < height && y < cur.Height; y++ {
		for z := 0; z < length && z < cur.Length; z++ {
			for x := 0; x < width && x < cur.Width; x++ {
				srcIdx := x + cur.Width*(z+cur.Length*y)
				dstIdx := x + width*(z+length*y)
				blocks[dstIdx] = cur.Blocks[srcIdx]
			}
		}
	}
	cur.Width, cur.Height, cur.Length = width, height, length
	cur.Blocks = blocks
	l.mgr.SetCurrent(cur)
	if l.server != nil {
		l.server.RegisterBlockPhysics(l.mgr)
	}
	return nil
}

func (l *pluginLevel) Reload() error {
	if err := l.mgr.Load(l.path); err != nil {
		return fmt.Errorf("reload level %q: %w", l.name, err)
	}
	if l.server != nil {
		l.server.RegisterBlockPhysics(l.mgr)
	}
	return nil
}

// pluginLevelManager implements plugin.LevelManager for multi-level mode.
type pluginLevelManager struct {
	srv *pluginServer
}

func (m *pluginLevelManager) Current() plugin.Level {
	name := m.srv.multiMgr.MainName()
	mgr := m.srv.multiMgr.MainManager()
	if mgr == nil {
		mgr = m.srv.worlds
		name = m.srv.server.cfg.Storage.MainWorldName
	}
	return &pluginLevel{
		mgr:    mgr,
		name:   name,
		path:   m.srv.multiMgr.Path(name),
		codec:  m.srv.codec,
		server: m.srv.server,
	}
}

func (m *pluginLevelManager) Find(name string) plugin.Level {
	mgr := m.srv.multiMgr.Get(name)
	if mgr == nil {
		return nil
	}
	return &pluginLevel{
		mgr:    mgr,
		name:   name,
		path:   m.srv.multiMgr.Path(name),
		codec:  m.srv.codec,
		server: m.srv.server,
	}
}

func (m *pluginLevelManager) Create(name string, width, height, length int, generatorName, seed string) (plugin.Level, error) {
	if err := validateLevelName(name); err != nil {
		return nil, err
	}
	gen, ok := generator.Find(generatorName)
	if !ok {
		return nil, fmt.Errorf("generator %q not found", generatorName)
	}
	args := generator.Args{Raw: seed, RandomSeed: seed == ""}
	lvl, err := generator.Generate(gen, name, width, height, length, args)
	if err != nil {
		return nil, err
	}
	level := world.FromGeneratorLevel(lvl)
	mgr := world.NewManager()
	mgr.SetCurrent(level)
	path := m.srv.server.store.WorldFile(name)
	if err := mgr.Save(path); err != nil {
		return nil, fmt.Errorf("save level %q: %w", name, err)
	}
	m.srv.multiMgr.Add(name, mgr, path)
	m.srv.server.RegisterBlockPhysics(mgr)
	if plugin.OnLevelAdded.HasHandlers() {
		plugin.OnLevelAdded.Fire(plugin.LevelAddedData{Name: name})
	}
	return &pluginLevel{
		mgr:    mgr,
		name:   name,
		path:   path,
		codec:  m.srv.codec,
		server: m.srv.server,
	}, nil
}

func (m *pluginLevelManager) Load(name string) (plugin.Level, error) {
	if err := validateLevelName(name); err != nil {
		return nil, err
	}
	if m.srv.multiMgr.Has(name) {
		return nil, fmt.Errorf("level %q already loaded", name)
	}
	if plugin.OnLevelLoad.HasHandlers() {
		ctx := plugin.OnLevelLoad.Fire(plugin.LevelLoadData{Name: name})
		if ctx.Cancelled() {
			return nil, fmt.Errorf("level load cancelled")
		}
	}
	path := m.srv.server.store.WorldFile(name)
	mgr := world.NewManager()
	if err := mgr.Load(path); err != nil {
		return nil, fmt.Errorf("load level %q: %w", name, err)
	}
	m.srv.multiMgr.Add(name, mgr, path)
	m.srv.server.RegisterBlockPhysics(mgr)
	if plugin.OnLevelLoaded.HasHandlers() {
		plugin.OnLevelLoaded.Fire(plugin.LevelLoadedData{Name: name})
	}
	return &pluginLevel{
		mgr:    mgr,
		name:   name,
		path:   path,
		codec:  m.srv.codec,
		server: m.srv.server,
	}, nil
}

func (m *pluginLevelManager) Unload(name string) bool {
	if strings.EqualFold(name, m.srv.multiMgr.MainName()) {
		return false
	}
	mgr := m.srv.multiMgr.Get(name)
	if mgr == nil {
		return false
	}
	if len(m.srv.codec.PlayersOnLevel(mgr)) > 0 {
		return false
	}
	if plugin.OnLevelUnload.HasHandlers() {
		plugin.OnLevelUnload.Fire(plugin.LevelUnloadData{Name: name})
	}
	removed := m.srv.multiMgr.Remove(name)
	if !removed {
		return false
	}
	m.srv.server.UnregisterBlockPhysics(mgr)
	if plugin.OnLevelRemoved.HasHandlers() {
		plugin.OnLevelRemoved.Fire(plugin.LevelRemovedData{Name: name})
	}
	return true
}

func (m *pluginLevelManager) SaveAll() error {
	for _, name := range m.srv.multiMgr.Names() {
		mgr := m.srv.multiMgr.Get(name)
		path := m.srv.multiMgr.Path(name)
		if mgr == nil || path == "" {
			continue
		}
		if err := mgr.Save(path); err != nil {
			return fmt.Errorf("save level %q: %w", name, err)
		}
	}
	if !m.srv.SaveState() {
		return fmt.Errorf("save player policy failed")
	}
	return nil
}

func (m *pluginLevelManager) List() []string {
	return m.srv.multiMgr.Names()
}

func (m *pluginLevelManager) ListFiles() []string {
	files, err := os.ReadDir(m.srv.server.store.WorldsDir())
	if err != nil {
		return nil
	}
	ext := m.srv.server.cfg.Storage.WorldFileExt
	var names []string
	for _, f := range files {
		name := strings.TrimSuffix(f.Name(), ext)
		names = append(names, name)
	}
	return names
}

func (m *pluginLevelManager) RenameLevel(oldName, newName string) error {
	if err := validateLevelName(oldName); err != nil {
		return err
	}
	if err := validateLevelName(newName); err != nil {
		return err
	}
	dest := m.srv.server.store.WorldFile(newName)
	if err := renameFileSync(m.srv.server.store.WorldFile(oldName), dest); err != nil {
		return err
	}
	m.srv.multiMgr.Rename(oldName, newName, dest)
	if plugin.OnLevelRenamed.HasHandlers() {
		plugin.OnLevelRenamed.Fire(plugin.LevelRenamedData{Source: oldName, Dest: newName})
	}
	return nil
}

func (m *pluginLevelManager) DeleteLevel(name string) error {
	if err := validateLevelName(name); err != nil {
		return err
	}
	if err := removeFileSync(m.srv.server.store.WorldFile(name)); err != nil {
		return err
	}
	if !strings.EqualFold(name, m.srv.multiMgr.MainName()) {
		m.srv.multiMgr.Remove(name)
	}
	if plugin.OnLevelRemoved.HasHandlers() {
		plugin.OnLevelRemoved.Fire(plugin.LevelRemovedData{Name: name})
	}
	return nil
}

func (m *pluginLevelManager) CopyLevel(srcName, destName string) error {
	if err := validateLevelName(srcName); err != nil {
		return err
	}
	if err := validateLevelName(destName); err != nil {
		return err
	}
	return copyFile(m.srv.server.store.WorldFile(srcName), m.srv.server.store.WorldFile(destName))
}

func (m *pluginLevelManager) BackupLevel(name, backupName string) error {
	if err := validateLevelName(name); err != nil {
		return err
	}
	if err := validateLevelName(backupName); err != nil {
		return err
	}
	return copyFile(m.srv.server.store.WorldFile(name), m.srv.server.store.WorldFile(backupName))
}

func validateLevelName(name string) error {
	if !storage.ValidName(name) {
		return fmt.Errorf("invalid level name %q", name)
	}
	return nil
}

// pluginPhysics implements plugin.Physics. Scheduled blocks are pumped
// through registered handlers each tick; blocks any handler votes false
// for are dropped. ponytail: single-level, O(n) per tick — per-region
// queues if throughput matters.
type pluginPhysics struct {
	mu        sync.Mutex
	mode      plugin.PhysicsMode
	scheduled []physicsBlock
	handlers  []plugin.PhysicsHandler
	worlds    *world.Manager
}

type physicsBlock struct {
	X, Y, Z int
}

func newPluginPhysics(worlds *world.Manager) *pluginPhysics {
	return &pluginPhysics{worlds: worlds}
}

func (ph *pluginPhysics) Mode() plugin.PhysicsMode {
	ph.mu.Lock()
	defer ph.mu.Unlock()
	return ph.mode
}

func (ph *pluginPhysics) SetMode(mode plugin.PhysicsMode) {
	ph.mu.Lock()
	ph.mode = mode
	ph.mu.Unlock()
	if plugin.OnPhysicsStateChanged.HasHandlers() {
		level := ""
		if ph.worlds != nil {
			level = ph.worlds.Current().Name
		}
		plugin.OnPhysicsStateChanged.Fire(plugin.PhysicsStateChangedData{Level: level, Mode: int(mode)})
	}
}

func (ph *pluginPhysics) Schedule(x, y, z int) {
	ph.mu.Lock()
	ph.scheduled = append(ph.scheduled, physicsBlock{X: x, Y: y, Z: z})
	ph.mu.Unlock()
}

func (ph *pluginPhysics) RegisterHandler(handler plugin.PhysicsHandler) {
	ph.mu.Lock()
	ph.handlers = append(ph.handlers, handler)
	ph.mu.Unlock()
}

// Tick processes every scheduled block through the registered handlers,
// fires OnPhysicsUpdate, and keeps blocks that no handler votes to remove.
func (ph *pluginPhysics) Tick() {
	ph.mu.Lock()
	scheduled := ph.scheduled
	handlers := ph.handlers
	ph.mu.Unlock()

	if len(scheduled) == 0 {
		return
	}

	level := ""
	if ph.worlds != nil {
		level = ph.worlds.Current().Name
	}

	keep := make([]physicsBlock, 0, len(scheduled))
	for _, b := range scheduled {
		var block byte
		if ph.worlds != nil {
			block, _ = ph.worlds.BlockAt(b.X, b.Y, b.Z)
		}
		if plugin.OnPhysicsUpdate.HasHandlers() {
			plugin.OnPhysicsUpdate.Fire(plugin.PhysicsUpdateData{X: b.X, Y: b.Y, Z: b.Z, Block: block, Level: level})
		}
		pb := plugin.PhysicsBlock{X: b.X, Y: b.Y, Z: b.Z, Block: block, Level: level}
		stay := true
		for _, h := range handlers {
			if !h(pb) {
				stay = false
			}
		}
		if stay {
			keep = append(keep, b)
		}
	}

	ph.mu.Lock()
	ph.scheduled = keep
	ph.mu.Unlock()
}

// entityManager implements plugin.EntityManager, broadcasting plugin-owned
// entities with IDs allocated by the shared Classic entity manager.
type entityManager struct {
	codec    *classic.Codec
	entities *entity.Manager
	mu       sync.Mutex
	slots    map[byte]entitySlot
}

type entitySlot struct {
	internalID uint32
	info       plugin.EntityInfo
}

func newEntityManager(codec *classic.Codec, entities *entity.Manager) *entityManager {
	return &entityManager{codec: codec, entities: entities, slots: make(map[byte]entitySlot)}
}

func (e *entityManager) Spawn(info plugin.EntityInfo) byte {
	id, ok := e.entities.Add(info.Name, entity.Position{X: info.X, Y: info.Y, Z: info.Z})
	if !ok {
		return 0
	}
	slot := byte(id)
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, exists := e.slots[slot]; exists {
		e.entities.Remove(id)
		return 0
	}
	e.slots[slot] = entitySlot{internalID: id, info: info}
	mgr := e.codec.MainWorldManager()
	e.codec.BroadcastPacketToLevel(mgr,
		e.codec.EncodeAddEntity(slot, info.Name, info.X, info.Y, info.Z, info.Yaw, info.Pitch))
	if info.Model != "" && info.Model != "humanoid" {
		if plugin.OnSendingModel.HasHandlers() {
			m := info.Model
			plugin.OnSendingModel.Fire(plugin.SendingModelData{Model: &m})
			info.Model = m
		}
		e.codec.BroadcastPacketToLevel(mgr, e.codec.EncodeChangeModel(slot, info.Model))
	}
	if plugin.OnEntitySpawned.HasHandlers() {
		name := info.Name
		model := info.Model
		plugin.OnEntitySpawned.Fire(plugin.EntitySpawnedData{Name: &name, Model: &model})
	}
	return slot
}

func (e *entityManager) Despawn(entityID byte) bool {
	e.mu.Lock()
	slot, ok := e.slots[entityID]
	if !ok {
		e.mu.Unlock()
		return false
	}
	delete(e.slots, entityID)
	e.mu.Unlock()
	e.entities.Remove(slot.internalID)
	mgr := e.codec.MainWorldManager()
	e.codec.BroadcastPacketToLevel(mgr, e.codec.EncodeRemoveEntity(entityID))
	if plugin.OnEntityDespawned.HasHandlers() {
		plugin.OnEntityDespawned.Fire(plugin.EntityDespawnedData{EntityID: entityID})
	}
	return true
}

func (e *entityManager) Teleport(entityID byte, x, y, z int, yaw, pitch byte) bool {
	e.mu.Lock()
	slot, ok := e.slots[entityID]
	if !ok {
		e.mu.Unlock()
		return false
	}
	slot.info.X, slot.info.Y, slot.info.Z = x, y, z
	slot.info.Yaw, slot.info.Pitch = yaw, pitch
	e.slots[entityID] = slot
	e.mu.Unlock()
	if !e.entities.SetLocation(slot.internalID, entity.Position{X: x, Y: y, Z: z}, yaw, pitch) {
		return false
	}
	mgr := e.codec.MainWorldManager()
	e.codec.BroadcastPacketToLevel(mgr, e.codec.EncodeEntityTeleport(entityID, x, y, z, yaw, pitch))
	return true
}

func (e *entityManager) Get(entityID byte) (plugin.EntityInfo, bool) {
	e.mu.Lock()
	slot, ok := e.slots[entityID]
	e.mu.Unlock()
	if !ok {
		return plugin.EntityInfo{}, false
	}
	info := slot.info
	if ent, ok := e.entities.Get(slot.internalID); ok {
		info.X, info.Y, info.Z = ent.Pos.X, ent.Pos.Y, ent.Pos.Z
		info.Yaw, info.Pitch = ent.Yaw, ent.Pitch
	}
	return info, true
}

func (e *entityManager) Count() int {
	return e.entities.Count()
}
