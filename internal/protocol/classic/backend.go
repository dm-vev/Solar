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
