package command

import (
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/plugin/playerdb"
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

// TeleportService exposes teleport operations for commands.
type TeleportService interface {
	// SpawnPoint returns the world spawn coordinates.
	SpawnPoint() (x, y, z int, yaw, pitch byte)
	// TeleportToPlayer teleports the caller to the named player.
	TeleportToPlayer(name string) bool
	// SummonPlayer teleports the named player to the caller.
	SummonPlayer(name string) bool
	// Back teleports the caller to their last position.
	Back() bool
}

// ChatService exposes chat operations for commands.
type ChatService interface {
	// Me sends an IRC-style action message to all players.
	Me(action string)
	// Whisper sends a private message to the named player.
	Whisper(targetName, msg string) bool
	// Ignore toggles ignoring a player's chat.
	Ignore(name string) (ignored bool, ok bool)
}

// DrawService exposes block selection and placement for drawing commands.
type DrawService interface {
	StartSelection(markCount int, callback func(marks [][3]int)) bool
	GetBlockAt(x, y, z int) (byte, bool)
	PlaceBlock(x, y, z int, block byte) bool
	LevelDims() (width, height, length int)
	// CopyRegion captures blocks from min to max into the player's clipboard.
	CopyRegion(min, max [3]int) bool
	// HasClipboard reports whether the player has a copied region.
	HasClipboard() bool
	// PasteAt replays the clipboard at the given origin. Returns false if no clipboard.
	PasteAt(origin [3]int, pasteAir bool) int
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
	MutePlayer(name string) bool
	UnmutePlayer(name string) bool
	FreezePlayer(name string) bool
	UnfreezePlayer(name string) bool
	ToggleAFK(name string) (afk bool, ok bool)
	ToggleHide(name string) (hidden bool, ok bool)
}

// PlayerDirectory exposes player listings.
type PlayerDirectory interface {
	ListPlayers() []string
	ListWhitelisted() []string
}

// PlayerLookup exposes offline player data for info commands.
type PlayerLookup interface {
	// Lookup returns the PlayerEntry for the named player, or nil.
	Lookup(name string) *playerdb.PlayerEntry
}

// ServerInfo exposes server-wide stats for info commands.
type ServerInfo interface {
	ServerName() string
	MOTD() string
	OnlineCount() int
	MaxPlayers() int
	LevelCount() int
	Uptime() time.Duration
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
	Teleport    TeleportService
	Chat        ChatService
	Draw        DrawService
	Persistence PersistenceService
	Moderation  ModerationService
	Players     PlayerDirectory
	BlockDefs   BlockDefService
	BlockDB     BlockDBService
	Levels      LevelService
	LevelEnv    LevelEnvService
	PlayerDB    PlayerLookup
	ServerInfo  ServerInfo
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
	for _, cmd := range []string{"tp", "setspawn", "setspawnpoint", "save", "kick", "ban", "unban", "whitelist", "newlvl", "gb", "lb", "blockdb", "load", "unload", "reload", "physics", "map", "mute", "unmute", "freeze", "unfreeze", "summon"} {
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
	registry.Register("mute", muteCommand)
	registry.Register("unmute", unmuteCommand)
	registry.Register("freeze", freezeCommand)
	registry.Register("unfreeze", unfreezeCommand)
	registry.Register("afk", afkCommand)
	registry.Register("hide", hideCommand)
	registry.Register("seen", seenCommand)
	registry.Register("whois", whoisCommand)
	registry.Register("blocks", blocksCommand)
	registry.Register("mapinfo", mapinfoCommand)
	registry.Register("serverinfo", serverinfoCommand)
	registry.Register("time", timeCommand)
	registry.Register("rules", rulesCommand)
	registry.Register("spawn", spawnCommand)
	registry.Register("back", backCommand)
	registry.Register("tpa", tpaCommand)
	registry.Register("summon", summonCommand)
	registry.Register("me", meCommand)
	registry.Register("whisper", whisperCommand)
	registry.Register("ignore", ignoreCommand)
	registry.Register("cuboid", cuboidCommand)
	registry.Register("line", lineCommand)
	registry.Register("sphere", sphereCommand)
	registry.Register("fill", fillCommand)
	registry.Register("copy", copyCommand)
	registry.Register("paste", pasteCommand)
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
