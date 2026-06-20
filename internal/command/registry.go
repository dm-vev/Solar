package command

import (
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/internal/blockdef"
)

// Position is a coarse world position used by chat commands.
type Position struct {
	X int
	Y int
	Z int
}

// Authority checks command permissions.
type Authority interface {
	CanAdmin() bool
}

// WorldService exposes world mutations used by commands.
type WorldService interface {
	SetBlock(x, y, z int, block byte) bool
	MovePlayer(x, y, z int, yaw, pitch byte) bool
	SetSpawn(x, y, z int, yaw, pitch byte) bool
	GenerateWorld(name, theme string, width, height, length int, seed string) bool
}

// PersistenceService persists runtime state.
type PersistenceService interface {
	SaveState() bool
}

// ModerationService exposes player policy commands.
type ModerationService interface {
	KickPlayer(name, reason string) bool
	BanPlayer(name, reason string) bool
	UnbanPlayer(name string) bool
	WhitelistEnabled() bool
	SetWhitelistEnabled(enabled bool) bool
	WhitelistAdd(name string) bool
	WhitelistRemove(name string) bool
}

// PlayerDirectory exposes player listings.
type PlayerDirectory interface {
	ListPlayers() []string
	ListWhitelisted() []string
}

// BlockDefService exposes custom block definition operations.
type BlockDefService interface {
	AddBlockDef(def blockdef.BlockDefinition) bool
	RemoveBlockDef(id byte) bool
	GetBlockDef(id byte) (blockdef.BlockDefinition, bool)
	ListBlockDefs() []blockdef.BlockDefinition
	FreeBlockID() byte
}

// BlockDBEntry is a single block change record.
type BlockDBEntry struct {
	PlayerName string
	Time       time.Time
	X, Y, Z    int
	OldBlock   byte
	NewBlock   byte
	Flags      uint16
}

// BlockDBService exposes block change history for the player's current level.
type BlockDBService interface {
	ChangesAt(x, y, z int) []BlockDBEntry
	ChangesBy(playerName string, since time.Time, maxResults int) []BlockDBEntry
	Count() int64
	Enabled() bool
	SetEnabled(bool)
	Clear() error
	RevertBlock(x, y, z int, block byte) bool
}

// LevelService exposes multi-level operations for commands.
type LevelService interface {
	Goto(levelName string) bool
	MainLevel() string
	LoadLevel(name string) bool
	UnloadLevel(name string) bool
	ReloadLevel() bool
	ListLevels() []string
	ListLevelFiles() []string
	PhysicsMode() int
	SetPhysicsMode(mode int)
}

// LevelEnvService exposes per-level environment properties for commands.
type LevelEnvService interface {
	// GetEnvColor returns the color for the given slot (0=sky, 1=cloud,
	// 2=fog, 3=ambient, 4=diffuse). Returns r, g, b, set.
	GetEnvColor(slot int) (r, g, b byte, set bool)
	// SetEnvColor sets the color for the given slot.
	SetEnvColor(slot int, r, g, b byte)
	// Weather returns the current weather (0=sunny, 1=rain, 2=snow).
	Weather() int
	// SetWeather sets the weather and broadcasts to level players.
	SetWeather(weather int)
	// MOTD returns the per-level MOTD.
	MOTD() string
	// SetMOTD sets the per-level MOTD.
	SetMOTD(motd string)
}

// Context carries command execution state.
type Context struct {
	Username    string
	Position    Position
	Yaw         byte
	Pitch       byte
	Authority   Authority
	World       WorldService
	Persistence PersistenceService
	Moderation  ModerationService
	Players     PlayerDirectory
	BlockDefs   BlockDefService
	BlockDB     BlockDBService
	Levels      LevelService
	LevelEnv    LevelEnvService
	Tr          func(string, ...any) string
}

// tr translates a message key for the current player. Falls back to
// the key itself if no translator is configured.
func (ctx Context) tr(key string, args ...any) string {
	if ctx.Tr != nil {
		return ctx.Tr(key, args...)
	}
	return key
}

// Handler processes a command and returns a user-facing response.
type Handler func(Context, []string) (string, bool)

// Registry stores the available chat commands.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
	admin    map[string]struct{}
}

// NewRegistry creates the command registry with the built-in commands.
func NewRegistry() *Registry {
	registry := &Registry{
		handlers: make(map[string]Handler),
		admin:    make(map[string]struct{}),
	}
	for _, cmd := range []string{"tp", "setspawn", "setspawnpoint", "save", "kick", "ban", "unban", "whitelist", "newlvl", "gb", "lb", "blockdb", "load", "unload", "reload", "physics", "map"} {
		registry.admin[cmd] = struct{}{}
	}
	registry.Register("help", helpCommand(registry))
	registry.Register("where", whereCommand)
	registry.Register("setblock", setBlockCommand)
	registry.Register("tp", teleportCommand)
	registry.Register("setspawn", setSpawnCommand)
	registry.Register("setspawnpoint", setSpawnCommand)
	registry.Register("save", saveCommand)
	registry.Register("kick", kickCommand)
	registry.Register("ban", banCommand)
	registry.Register("unban", unbanCommand)
	registry.Register("whitelist", whitelistCommand)
	registry.Register("players", playersCommand)
	registry.Register("newlvl", newLevelCommand)
	registry.Register("gb", globalBlockCommand)
	registry.Register("lb", levelBlockCommand)
	registry.Register("about", aboutCommand)
	registry.Register("b", aboutCommand)
	registry.Register("undo", undoCommand)
	registry.Register("blockdb", blockDBCommand)
	registry.Register("goto", gotoCommand)
	registry.Register("main", mainCommand)
	registry.Register("load", loadCommand)
	registry.Register("unload", unloadCommand)
	registry.Register("reload", reloadCommand)
	registry.Register("levels", levelsCommand)
	registry.Register("physics", physicsCommand)
	registry.Register("map", mapCommand)
	return registry
}

// SetAdminCommands replaces the set of commands requiring operator privileges.
func (r *Registry) SetAdminCommands(cmds []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.admin = make(map[string]struct{}, len(cmds))
	for _, cmd := range cmds {
		r.admin[strings.ToLower(cmd)] = struct{}{}
	}
}

// Register adds or replaces a command handler.
func (r *Registry) Register(name string, handler Handler) {
	if name == "" || handler == nil {
		return
	}

	r.mu.Lock()
	r.handlers[strings.ToLower(name)] = handler
	r.mu.Unlock()
}

// Unregister removes a command handler. Returns false if not found.
func (r *Registry) Unregister(name string) bool {
	if name == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.ToLower(name)
	if _, ok := r.handlers[key]; !ok {
		return false
	}
	delete(r.handlers, key)
	return true
}

// Execute runs a command line and returns a response when handled.
func (r *Registry) Execute(ctx Context, line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	line = strings.TrimPrefix(line, "/")

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return "", false
	}

	name := strings.ToLower(fields[0])
	args := fields[1:]

	r.mu.RLock()
	handler := r.handlers[name]
	r.mu.RUnlock()

	if handler == nil {
		return ctx.tr("command.shared.unknown", name), true
	}

	if r.requiresAdmin(name) && (ctx.Authority == nil || !ctx.Authority.CanAdmin()) {
		return ctx.tr("command.shared.permission_denied"), true
	}

	return handler(ctx, args)
}

func (r *Registry) requiresAdmin(name string) bool {
	r.mu.RLock()
	_, ok := r.admin[name]
	r.mu.RUnlock()
	return ok
}
