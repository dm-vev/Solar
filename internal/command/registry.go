package command

import (
	"fmt"
	"strings"
	"sync"
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
}

// Handler processes a command and returns a user-facing response.
type Handler func(Context, []string) (string, bool)

// Registry stores the available chat commands.
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]Handler
}

// NewRegistry creates the command registry with the built-in commands.
func NewRegistry() *Registry {
	registry := &Registry{handlers: make(map[string]Handler)}
	registry.Register("help", helpCommand(registry))
	registry.Register("where", whereCommand)
	registry.Register("setblock", setBlockCommand)
	registry.Register("tp", teleportCommand)
	registry.Register("setspawn", setSpawnCommand)
	registry.Register("save", saveCommand)
	registry.Register("kick", kickCommand)
	registry.Register("ban", banCommand)
	registry.Register("unban", unbanCommand)
	registry.Register("whitelist", whitelistCommand)
	registry.Register("players", playersCommand)
	registry.Register("newlvl", newLevelCommand)
	return registry
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
		return fmt.Sprintf("unknown command: /%s", name), true
	}

	if requiresAdmin(name) && (ctx.Authority == nil || !ctx.Authority.CanAdmin()) {
		return "permission denied", true
	}

	return handler(ctx, args)
}
