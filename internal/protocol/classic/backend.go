// backend.go defines the SessionBackend interface.
//
// SessionBackend is the contract between the protocol layer and the
// command/adapter layer. It exposes session operations needed by chat
// commands without coupling the command package to the session struct.
//
// The interface is implemented by *Session in admin.go. The app package
// (adapters.go) wraps SessionBackend methods into command.Context fields
// (ModerationService, LevelService, BlockDBService, etc.).
//
// This separation follows the Go best practice of defining interfaces
// on the consumer side (Effective Go: "Interfaces belong in the package
// that uses values of the interface type"). SessionBackend is defined
// here because it bridges two internal packages.

package classic

import (
	"time"

	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin/playerdb"
)

type SessionBackend interface {
	CurrentUsername() string
	CurrentLocation() (world.Spawn, byte, byte)
	IsOperator() bool
	Translate(key string, args ...any) string

	ApplyBlockChange(x, y, z int, blockID byte, echo bool) error
	TeleportSelf(x, y, z int, yaw, pitch byte) bool
	SetSpawn(spawn world.Spawn) bool
	GenerateWorld(name, theme string, width, height, length int, seed string) bool
	SaveState() bool
	PersistPlayerPolicy() bool

	SpawnPoint() (x, y, z int, yaw, pitch byte)
	TeleportToPlayer(name string) bool
	SummonPlayer(name string) bool
	BackToLastPos() bool

	MeAction(action string)
	WhisperTo(targetName, msg string) bool
	IgnorePlayer(name string) (bool, bool)

	StartSelection(markCount int, callback func(marks [][3]int)) bool
	GetBlockAt(x, y, z int) (byte, bool)
	PlaceBlock(x, y, z int, block byte) bool
	LevelDims() (width, height, length int)
	CopyRegion(min, max [3]int) bool
	HasClipboard() bool
	PasteAt(origin [3]int, pasteAir bool) int

	KickPlayer(name, reason string) bool
	BanPlayer(name, reason string) bool
	UnbanPlayer(name string) bool
	WhitelistEnabled() bool
	WhitelistAdd(name string) bool
	WhitelistRemove(name string) bool
	SetWhitelistEnabled(enabled bool) bool

	MutePlayer(name string) bool
	UnmutePlayer(name string) bool
	FreezePlayer(name string) bool
	UnfreezePlayer(name string) bool
	ToggleAFK(name string) (afk bool, ok bool)
	ToggleHide(name string) (hidden bool, ok bool)

	OnlineNames() []string
	WhitelistNames() []string

	AddBlockDef(def blockdef.BlockDefinition) bool
	RemoveBlockDef(id byte) bool
	GetBlockDef(id byte) (blockdef.BlockDefinition, bool)
	ListBlockDefs() []blockdef.BlockDefinition
	FreeBlockID() byte

	BlockDBChangesAt(x, y, z int) []command.BlockDBEntry
	BlockDBChangesBy(playerName string, since time.Time, max int) []command.BlockDBEntry
	BlockDBCount() int64
	BlockDBEnabled() bool
	BlockDBSetEnabled(bool)
	BlockDBClear() error
	BlockDBRevertBlock(x, y, z int, block byte) bool

	GotoLevel(name string) bool
	MainLevelName() string
	LoadLevel(name string) bool
	UnloadLevel(name string) bool
	ReloadCurrentLevel() bool
	ListLoadedLevels() []string
	ListLevelFiles() []string
	CurrentPhysicsMode() int
	SetCurrentPhysicsMode(mode int)

	GetEnvColor(slot int) (r, g, b byte, set bool)
	SetLevelEnvColor(slot int, r, g, b byte)
	GetWeather() int
	SetLevelWeather(weather int)
	GetLevelMOTD() string
	SetLevelMOTD(motd string)

	PlayerDBLookup(name string) *playerdb.PlayerEntry
	ServerName() string
	ServerMOTD() string
	OnlinePlayerCount() int
	MaxPlayersCount() int
	LoadedLevelCount() int
	ServerUptime() time.Duration
}

// --- SessionBackend implementation on *session ---
