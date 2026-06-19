package server

import (
	"github.com/solar-mc/solar/plugin/command"
	"github.com/solar-mc/solar/plugin/config"
	"github.com/solar-mc/solar/plugin/entity"
	"github.com/solar-mc/solar/plugin/level"
	"github.com/solar-mc/solar/plugin/physics"
	"github.com/solar-mc/solar/plugin/player"
	"github.com/solar-mc/solar/plugin/scheduler"
	"github.com/solar-mc/solar/plugin/world"
)

// Server is the interface plugins use to interact with the server.
// An instance is passed to Plugin.Enable.
type Server interface {
	// BroadcastMessage sends a chat message to all online players.
	BroadcastMessage(msg string)

	// OnlinePlayers returns all currently connected players.
	OnlinePlayers() []player.Player

	// FindPlayer returns the online player with the given name
	// (case-insensitive), or nil if not found.
	FindPlayer(name string) player.Player

	// World returns the current world.
	World() world.World

	// RegisterCommand adds a custom command.
	// name is the command without the leading "/".
	// help is shown by /help.
	// handler is called with the player and parsed args.
	// Returns false if the name is already taken.
	RegisterCommand(name string, help string, handler command.CommandHandler) bool

	// UnregisterCommand removes a custom command.
	// Returns false if not found or if it's a built-in command.
	UnregisterCommand(name string) bool

	// BanPlayer bans a player by name.
	BanPlayer(name, reason string) bool

	// UnbanPlayer removes a ban.
	UnbanPlayer(name string) bool

	// Stop shuts down the server.
	Stop()

	// BroadcastMessageTo sends a message to a specific scope.
	// scope: "all", "level" (same level as source), "ops" (operators only)
	BroadcastMessageTo(scope string, source player.Player, msg string)

	// OnlineCount returns the number of online players.
	OnlineCount() int

	// MaxPlayers returns the max player cap.
	MaxPlayers() int

	// ServerName returns the configured server name.
	ServerName() string

	// MOTD returns the configured MOTD.
	MOTD() string

	// IsWhitelistEnabled reports whether whitelist is active.
	IsWhitelistEnabled() bool

	// SetWhitelistEnabled enables/disables the whitelist.
	SetWhitelistEnabled(enabled bool)

	// WhitelistAdd adds a player to the whitelist.
	WhitelistAdd(name string) bool

	// WhitelistRemove removes a player from the whitelist.
	WhitelistRemove(name string) bool

	// IsOperator reports whether the named player is an operator.
	IsOperator(name string) bool

	// AddOperators adds operator names.
	AddOperators(names ...string) bool

	// OperatorNames returns all operator names.
	OperatorNames() []string

	// SaveState saves world and policy to disk.
	SaveState() bool

	// Scheduler returns the task scheduler for timed/tick tasks.
	Scheduler() scheduler.Scheduler

	// Levels returns the level manager for multi-level operations.
	Levels() level.LevelManager

	// ChangeMap teleports a player to another level by name.
	// Returns false if the level is not found.
	ChangeMap(p player.Player, levelName string) bool

	// Physics returns the physics controller for the main level.
	Physics() physics.Physics

	// Entities returns the entity manager for spawning/despawning non-player entities.
	Entities() entity.EntityManager

	// Config returns the server configuration handle.
	Config() config.Config
}
