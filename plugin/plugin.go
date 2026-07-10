// Package plugin defines the Solar plugin system.
//
// Plugins register at compile time via init():
//
//	package myplugin
//
//	import "github.com/solar-mc/solar/plugin"
//
//	func init() {
//	    plugin.Register("myplugin", &MyPlugin{})
//	}
//
// The server calls Init, then Enable on each registered plugin at startup.
// Plugins subscribe to events and register commands during Enable.
// On shutdown, Disable is called for cleanup.
//
// # Architecture
//
// The plugin system is split into thematic sub-packages:
//
//   - plugin/player    — Player interface
//   - plugin/world     — World interface
//   - plugin/level     — Level + LevelManager interfaces
//   - plugin/entity    — EntityInfo + EntityManager interfaces
//   - plugin/color     — Color type, constants, helpers
//   - plugin/cpe       — CPE packet sender interface
//   - plugin/physics   — Physics simulation interface
//   - plugin/command   — CommandHandler type
//   - plugin/scheduler — Scheduler, Task, Duration types
//   - plugin/server    — Server interface (aggregates all above)
//   - plugin/event     — Event system + all event types
//
// This root package re-exports the key types for ergonomic imports:
//
//	plugin.Player       // = player.Player
//	plugin.Server       // = server.Server
//	plugin.OnPlayerChat // = event.OnPlayerChat
//	plugin.PriorityNormal
package plugin

import (
	"log/slog"
	"sort"
	"sync"

	"github.com/solar-mc/solar/plugin/blockdb"
	"github.com/solar-mc/solar/plugin/color"
	"github.com/solar-mc/solar/plugin/command"
	"github.com/solar-mc/solar/plugin/config"
	"github.com/solar-mc/solar/plugin/cpe"
	"github.com/solar-mc/solar/plugin/entity"
	"github.com/solar-mc/solar/plugin/event"
	"github.com/solar-mc/solar/plugin/level"
	"github.com/solar-mc/solar/plugin/physics"
	"github.com/solar-mc/solar/plugin/player"
	"github.com/solar-mc/solar/plugin/playerdb"
	"github.com/solar-mc/solar/plugin/scheduler"
	"github.com/solar-mc/solar/plugin/server"
	"github.com/solar-mc/solar/plugin/world"
)

// ─── Re-exported API types ───

// Player is re-exported from plugin/player.
type Player = player.Player

// World is re-exported from plugin/world.
type World = world.World

// Level is re-exported from plugin/level.
type Level = level.Level

// LevelManager is re-exported from plugin/level.
type LevelManager = level.LevelManager

// Server is re-exported from plugin/server.
type Server = server.Server

// Config is re-exported from plugin/config.
type Config = config.Config

// CommandHandler is re-exported from plugin/command.
type CommandHandler = command.CommandHandler

// CommandSpec is re-exported from plugin/command.
type CommandSpec = command.CommandSpec

// BlockPos is re-exported from plugin/player.
type BlockPos = player.BlockPos

// SelectionHandler is re-exported from plugin/player.
type SelectionHandler = player.SelectionHandler

// CPE is re-exported from plugin/cpe.
type CPE = cpe.CPE

// Scheduler is re-exported from plugin/scheduler.
type Scheduler = scheduler.Scheduler

// Task is re-exported from plugin/scheduler.
type Task = scheduler.Task

// Duration is re-exported from plugin/scheduler.
type Duration = scheduler.Duration

// Physics is re-exported from plugin/physics.
type Physics = physics.Physics

// PhysicsMode is re-exported from plugin/physics.
type PhysicsMode = physics.PhysicsMode

// PhysicsBlock is re-exported from plugin/physics.
type PhysicsBlock = physics.PhysicsBlock

// PhysicsHandler is re-exported from plugin/physics.
type PhysicsHandler = physics.PhysicsHandler

// Color is re-exported from plugin/color.
type Color = color.Color

// EntityInfo is re-exported from plugin/entity.
type EntityInfo = entity.EntityInfo

// EntityManager is re-exported from plugin/entity.
type EntityManager = entity.EntityManager

// PlayerDB is re-exported from plugin/playerdb.
type PlayerDB = playerdb.PlayerDB

// PlayerEntry is re-exported from plugin/playerdb.
type PlayerEntry = playerdb.PlayerEntry

// BlockDB is re-exported from plugin/blockdb.
type BlockDB = blockdb.BlockDB

// BlockEntry is re-exported from plugin/blockdb.
type BlockEntry = blockdb.Entry

// BlockFlags is re-exported from plugin/blockdb.
type BlockFlags = blockdb.Flags

// Block change flags (re-exported from plugin/blockdb).
const (
	BlockManualPlace = blockdb.ManualPlace
	BlockPainted     = blockdb.Painted
	BlockDrawn       = blockdb.Drawn
	BlockReplaced    = blockdb.Replaced
	BlockPasted      = blockdb.Pasted
	BlockCut         = blockdb.Cut
	BlockFilled      = blockdb.Filled
	BlockRestored    = blockdb.Restored
	BlockUndoOther   = blockdb.UndoOther
	BlockUndoSelf    = blockdb.UndoSelf
	BlockRedoSelf    = blockdb.RedoSelf
	BlockFixGrass    = blockdb.FixGrass
)

// ─── Re-exported color constants ───

const (
	ColorBlack   = color.ColorBlack
	ColorNavy    = color.ColorNavy
	ColorGreen   = color.ColorGreen
	ColorTeal    = color.ColorTeal
	ColorMaroon  = color.ColorMaroon
	ColorPurple  = color.ColorPurple
	ColorGold    = color.ColorGold
	ColorSilver  = color.ColorSilver
	ColorGray    = color.ColorGray
	ColorBlue    = color.ColorBlue
	ColorLime    = color.ColorLime
	ColorAqua    = color.ColorAqua
	ColorRed     = color.ColorRed
	ColorPink    = color.ColorPink
	ColorYellow  = color.ColorYellow
	ColorWhite   = color.ColorWhite
	ColorServer  = color.ColorServer
	ColorHelp    = color.ColorHelp
	ColorSyntax  = color.ColorSyntax
	ColorIRC     = color.ColorIRC
	ColorWarning = color.ColorWarning
)

// ─── Re-exported physics constants ───

const (
	PhysicsOff      = physics.PhysicsOff
	PhysicsBasic    = physics.PhysicsBasic
	PhysicsAdvanced = physics.PhysicsAdvanced
	PhysicsCustom   = physics.PhysicsCustom
)

// Built-in rank permission levels.
const (
	RankBanned     = player.RankBanned
	RankGuest      = player.RankGuest
	RankBuilder    = player.RankBuilder
	RankAdvBuilder = player.RankAdvBuilder
	RankOperator   = player.RankOperator
	RankAdmin      = player.RankAdmin
	RankOwner      = player.RankOwner
)

// ─── Re-exported color functions ───

var (
	Colorize   = color.Colorize
	StripColor = color.StripColor
	ParseColor = color.ParseColor
	ColorName  = color.ColorName
)

// ─── Re-exported event types ───

type (
	Context  = event.Context
	Priority = event.Priority

	// Player events
	PlayerConnectData          = event.PlayerConnectData
	PlayerDisconnectData       = event.PlayerDisconnectData
	PlayerStartConnectingData  = event.PlayerStartConnectingData
	PlayerFinishConnectingData = event.PlayerFinishConnectingData
	PlayerChatData             = event.PlayerChatData
	MessageReceivedData        = event.MessageReceivedData
	PlayerMoveData             = event.PlayerMoveData
	PlayerCommandData          = event.PlayerCommandData
	PlayerHelpData             = event.PlayerHelpData
	PlayerActionData           = event.PlayerActionData
	PlayerDyingData            = event.PlayerDyingData
	PlayerDiedData             = event.PlayerDiedData
	BlockChangeData            = event.BlockChangeData
	BlockChangedData           = event.BlockChangedData
	PlayerClickData            = event.PlayerClickData
	PlayerSpawnData            = event.PlayerSpawnData
	SentMapData                = event.SentMapData
	JoiningLevelData           = event.JoiningLevelData
	JoinedLevelData            = event.JoinedLevelData
	SettingColorData           = event.SettingColorData
	GettingMotdData            = event.GettingMotdData
	SendingMotdData            = event.SendingMotdData
	GettingCanSeeData          = event.GettingCanSeeData
	NotifyActionData           = event.NotifyActionData
	NotifyPositionActionData   = event.NotifyPositionActionData

	// Server events
	ConnectionReceivedData = event.ConnectionReceivedData
	TickData               = event.TickData
	ShutdownData           = event.ShutdownData
	PluginsLoadedData      = event.PluginsLoadedData
	ConfigUpdatedData      = event.ConfigUpdatedData
	ChatSysData            = event.ChatSysData
	ChatFromData           = event.ChatFromData
	ChatData               = event.ChatData
	CpePluginMessageData   = event.CpePluginMessageData

	// Level events
	LevelSaveData           = event.LevelSaveData
	LevelLoadData           = event.LevelLoadData
	LevelLoadedData         = event.LevelLoadedData
	LevelAddedData          = event.LevelAddedData
	LevelRemovedData        = event.LevelRemovedData
	LevelUnloadData         = event.LevelUnloadData
	LevelDeletedData        = event.LevelDeletedData
	LevelCopiedData         = event.LevelCopiedData
	LevelRenamedData        = event.LevelRenamedData
	MainLevelChangingData   = event.MainLevelChangingData
	PhysicsUpdateData       = event.PhysicsUpdateData
	PhysicsStateChangedData = event.PhysicsStateChangedData

	// Entity events
	EntitySpawnedData   = event.EntitySpawnedData
	EntityDespawnedData = event.EntityDespawnedData
	SendingModelData    = event.SendingModelData
)

// ─── Re-exported event instances ───

var (
	// Player events
	OnPlayerConnect          = event.OnPlayerConnect
	OnPlayerDisconnect       = event.OnPlayerDisconnect
	OnPlayerStartConnecting  = event.OnPlayerStartConnecting
	OnPlayerFinishConnecting = event.OnPlayerFinishConnecting
	OnPlayerChat             = event.OnPlayerChat
	OnMessageReceived        = event.OnMessageReceived
	OnPlayerMove             = event.OnPlayerMove
	OnPlayerCommand          = event.OnPlayerCommand
	OnPlayerHelp             = event.OnPlayerHelp
	OnPlayerAction           = event.OnPlayerAction
	OnPlayerDying            = event.OnPlayerDying
	OnPlayerDied             = event.OnPlayerDied
	OnBlockChange            = event.OnBlockChange
	OnBlockChanged           = event.OnBlockChanged
	OnPlayerClick            = event.OnPlayerClick
	OnPlayerSpawn            = event.OnPlayerSpawn
	OnSentMap                = event.OnSentMap
	OnJoiningLevel           = event.OnJoiningLevel
	OnJoinedLevel            = event.OnJoinedLevel
	OnSettingColor           = event.OnSettingColor
	OnGettingMotd            = event.OnGettingMotd
	OnSendingMotd            = event.OnSendingMotd
	OnGettingCanSee          = event.OnGettingCanSee
	OnNotifyAction           = event.OnNotifyAction
	OnNotifyPositionAction   = event.OnNotifyPositionAction

	// Server events
	OnConnectionReceived = event.OnConnectionReceived
	OnTick               = event.OnTick
	OnShutdown           = event.OnShutdown
	OnPluginsLoaded      = event.OnPluginsLoaded
	OnConfigUpdated      = event.OnConfigUpdated
	OnChatSys            = event.OnChatSys
	OnChatFrom           = event.OnChatFrom
	OnChat               = event.OnChat
	OnPluginMessage      = event.OnPluginMessage

	// Level events
	OnLevelSave           = event.OnLevelSave
	OnLevelLoad           = event.OnLevelLoad
	OnLevelLoaded         = event.OnLevelLoaded
	OnLevelAdded          = event.OnLevelAdded
	OnLevelRemoved        = event.OnLevelRemoved
	OnLevelUnload         = event.OnLevelUnload
	OnLevelDeleted        = event.OnLevelDeleted
	OnLevelCopied         = event.OnLevelCopied
	OnLevelRenamed        = event.OnLevelRenamed
	OnMainLevelChanging   = event.OnMainLevelChanging
	OnPhysicsUpdate       = event.OnPhysicsUpdate
	OnPhysicsStateChanged = event.OnPhysicsStateChanged

	// Entity events
	OnEntitySpawned   = event.OnEntitySpawned
	OnEntityDespawned = event.OnEntityDespawned
	OnSendingModel    = event.OnSendingModel
)

// ─── Priority constants ───

const (
	PriorityLow      = event.PriorityLow
	PriorityNormal   = event.PriorityNormal
	PriorityHigh     = event.PriorityHigh
	PriorityCritical = event.PriorityCritical
)

// NewEvent creates a new event. Re-exported from plugin/event.
func NewEvent[T any]() *event.Event[T] { return event.NewEvent[T]() }

// DefaultScheduler is the default scheduler instance.
var DefaultScheduler = scheduler.Default

// ─── Plugin interface + registry ───

// Plugin is the interface every plugin implements.
type Plugin interface {
	Name() string
	Init() error
	Enable(s Server) error
	Disable() error
}

// Info holds plugin metadata.
type Info struct {
	Name    string
	Version string
	Author  string
}

var (
	mu      sync.Mutex
	plugins []Plugin
	loaded  bool
)

// Register adds a plugin to the registry. Must be called from init().
func Register(name string, p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	for _, existing := range plugins {
		if existing.Name() == name {
			return
		}
	}
	plugins = append(plugins, p)
}

// Registered returns all registered plugins sorted by name.
func Registered() []Plugin {
	mu.Lock()
	defer mu.Unlock()
	out := make([]Plugin, len(plugins))
	copy(out, plugins)
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// LoadAll initialises and enables all registered plugins.
func LoadAll(s Server, logger *slog.Logger) error {
	mu.Lock()
	if loaded {
		mu.Unlock()
		return nil
	}
	loaded = true
	toLoad := make([]Plugin, len(plugins))
	copy(toLoad, plugins)
	mu.Unlock()

	for _, p := range toLoad {
		if err := p.Init(); err != nil {
			logger.Error("plugin init failed", "plugin", p.Name(), "error", err)
			return err
		}
	}
	for _, p := range toLoad {
		if err := p.Enable(s); err != nil {
			logger.Error("plugin enable failed", "plugin", p.Name(), "error", err)
			return err
		}
		logger.Info("plugin enabled", "plugin", p.Name())
	}
	return nil
}

// UnloadAll disables all plugins in reverse order.
func UnloadAll(logger *slog.Logger) {
	mu.Lock()
	toUnload := make([]Plugin, len(plugins))
	copy(toUnload, plugins)
	mu.Unlock()

	for i := len(toUnload) - 1; i >= 0; i-- {
		p := toUnload[i]
		if err := p.Disable(); err != nil {
			logger.Error("plugin disable failed", "plugin", p.Name(), "error", err)
		}
	}
}
