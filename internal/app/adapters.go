package app

import (
	"time"

	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin/playerdb"
)

// sessionAuthority implements command.Authority.
type sessionAuthority struct {
	backend classic.SessionBackend
}

func (a sessionAuthority) CanAdmin() bool {
	return a.backend.IsOperator()
}

// sessionWorld implements command.WorldService.
type sessionWorld struct {
	backend classic.SessionBackend
}

func (w sessionWorld) SetBlock(x, y, z int, blockID byte) bool {
	return w.backend.ApplyBlockChange(x, y, z, blockID, true) == nil
}

func (w sessionWorld) MovePlayer(x, y, z int, yaw, pitch byte) bool {
	return w.backend.TeleportSelf(x, y, z, yaw, pitch)
}

func (w sessionWorld) SetSpawn(x, y, z int, yaw, pitch byte) bool {
	return w.backend.SetSpawn(world.Spawn{X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch})
}

func (w sessionWorld) GenerateWorld(name, theme string, width, height, length int, seed string) bool {
	return w.backend.GenerateWorld(name, theme, width, height, length, seed)
}

// sessionPersistence implements command.PersistenceService.
type sessionPersistence struct {
	backend classic.SessionBackend
}

func (p sessionPersistence) SaveState() bool {
	return p.backend.SaveState()
}

// sessionModeration implements command.ModerationService.
type sessionModeration struct {
	backend classic.SessionBackend
}

func (m sessionModeration) KickPlayer(name, reason string) bool {
	return m.backend.KickPlayer(name, reason)
}

func (m sessionModeration) BanPlayer(name, reason string) bool {
	return m.backend.BanPlayer(name, reason)
}

func (m sessionModeration) UnbanPlayer(name string) bool {
	return m.backend.UnbanPlayer(name)
}

func (m sessionModeration) WhitelistEnabled() bool {
	return m.backend.WhitelistEnabled()
}

func (m sessionModeration) SetWhitelistEnabled(enabled bool) bool {
	return m.backend.SetWhitelistEnabled(enabled)
}

func (m sessionModeration) WhitelistAdd(name string) bool {
	return m.backend.WhitelistAdd(name)
}

func (m sessionModeration) WhitelistRemove(name string) bool {
	return m.backend.WhitelistRemove(name)
}

func (m sessionModeration) MutePlayer(name string) bool {
	return m.backend.MutePlayer(name)
}

func (m sessionModeration) UnmutePlayer(name string) bool {
	return m.backend.UnmutePlayer(name)
}

func (m sessionModeration) FreezePlayer(name string) bool {
	return m.backend.FreezePlayer(name)
}

func (m sessionModeration) UnfreezePlayer(name string) bool {
	return m.backend.UnfreezePlayer(name)
}

func (m sessionModeration) ToggleAFK(name string) (bool, bool) {
	return m.backend.ToggleAFK(name)
}

func (m sessionModeration) ToggleHide(name string) (bool, bool) {
	return m.backend.ToggleHide(name)
}

// sessionDirectory implements command.PlayerDirectory.
type sessionDirectory struct {
	backend classic.SessionBackend
}

func (d sessionDirectory) ListPlayers() []string {
	return d.backend.OnlineNames()
}

func (d sessionDirectory) ListWhitelisted() []string {
	return d.backend.WhitelistNames()
}

// sessionBlockDefs implements command.BlockDefService.
type sessionBlockDefs struct {
	backend classic.SessionBackend
}

func (b sessionBlockDefs) AddBlockDef(def blockdef.BlockDefinition) bool {
	return b.backend.AddBlockDef(def)
}

func (b sessionBlockDefs) RemoveBlockDef(id byte) bool {
	return b.backend.RemoveBlockDef(id)
}

func (b sessionBlockDefs) GetBlockDef(id byte) (blockdef.BlockDefinition, bool) {
	return b.backend.GetBlockDef(id)
}

func (b sessionBlockDefs) ListBlockDefs() []blockdef.BlockDefinition {
	return b.backend.ListBlockDefs()
}

func (b sessionBlockDefs) FreeBlockID() byte {
	return b.backend.FreeBlockID()
}

// buildCommandContext assembles a command.Context from a SessionBackend.
// This function is injected into the codec via SetCommandContextBuilder,
// keeping the protocol layer decoupled from the command adapter types.
func buildCommandContext(backend classic.SessionBackend) command.Context {
	position, yaw, pitch := backend.CurrentLocation()

	return command.Context{
		Username: backend.CurrentUsername(),
		Position: command.Position{
			X: position.X,
			Y: position.Y,
			Z: position.Z,
		},
		Yaw:         yaw,
		Pitch:       pitch,
		Authority:   sessionAuthority{backend},
		World:       sessionWorld{backend},
		Teleport:    sessionTeleport{backend},
		Chat:        sessionChat{backend},
		Draw:        sessionDraw{backend},
		Persistence: sessionPersistence{backend},
		Moderation:  sessionModeration{backend},
		Players:     sessionDirectory{backend},
		BlockDefs:   sessionBlockDefs{backend},
		BlockDB:     sessionBlockDB{backend},
		Levels:      sessionLevels{backend},
		LevelEnv:    sessionLevelEnv{backend},
		PlayerDB:    sessionPlayerDB{backend},
		ServerInfo:  sessionSrvInfo{backend},
		Tr:          backend.Translate,
	}
}

// ─── TeleportService adapter ───

type sessionTeleport struct{ backend classic.SessionBackend }

func (s sessionTeleport) SpawnPoint() (int, int, int, byte, byte) {
	return s.backend.SpawnPoint()
}
func (s sessionTeleport) TeleportToPlayer(name string) bool {
	return s.backend.TeleportToPlayer(name)
}
func (s sessionTeleport) SummonPlayer(name string) bool {
	return s.backend.SummonPlayer(name)
}
func (s sessionTeleport) Back() bool {
	return s.backend.BackToLastPos()
}

// ─── ChatService adapter ───

type sessionChat struct{ backend classic.SessionBackend }

func (s sessionChat) Me(action string)                { s.backend.MeAction(action) }
func (s sessionChat) Whisper(target, msg string) bool { return s.backend.WhisperTo(target, msg) }
func (s sessionChat) Ignore(name string) (bool, bool) { return s.backend.IgnorePlayer(name) }

// ─── DrawService adapter ───

type sessionDraw struct{ backend classic.SessionBackend }

func (s sessionDraw) StartSelection(markCount int, callback func([][3]int)) bool {
	return s.backend.StartSelection(markCount, callback)
}
func (s sessionDraw) GetBlockAt(x, y, z int) (byte, bool) {
	return s.backend.GetBlockAt(x, y, z)
}
func (s sessionDraw) PlaceBlock(x, y, z int, block byte) bool {
	return s.backend.PlaceBlock(x, y, z, block)
}
func (s sessionDraw) LevelDims() (int, int, int) {
	return s.backend.LevelDims()
}
func (s sessionDraw) CopyRegion(min, max [3]int) bool {
	return s.backend.CopyRegion(min, max)
}
func (s sessionDraw) HasClipboard() bool {
	return s.backend.HasClipboard()
}
func (s sessionDraw) PasteAt(origin [3]int, pasteAir bool) int {
	return s.backend.PasteAt(origin, pasteAir)
}
func (s sessionDraw) SetSpecialBlock(x, y, z int, entry command.SpecialBlockEntry) bool {
	return s.backend.SetSpecialBlock(x, y, z, entry)
}

// ─── BlockDB adapter ───

type sessionBlockDB struct{ backend classic.SessionBackend }

func (s sessionBlockDB) ChangesAt(x, y, z int) []command.BlockDBEntry {
	return s.backend.BlockDBChangesAt(x, y, z)
}

func (s sessionBlockDB) ChangesBy(playerName string, since time.Time, max int) []command.BlockDBEntry {
	return s.backend.BlockDBChangesBy(playerName, since, max)
}

func (s sessionBlockDB) Count() int64 {
	return s.backend.BlockDBCount()
}

func (s sessionBlockDB) Enabled() bool {
	return s.backend.BlockDBEnabled()
}

func (s sessionBlockDB) SetEnabled(enabled bool) {
	s.backend.BlockDBSetEnabled(enabled)
}

func (s sessionBlockDB) Clear() error {
	return s.backend.BlockDBClear()
}

func (s sessionBlockDB) RevertBlock(x, y, z int, block byte) bool {
	return s.backend.BlockDBRevertBlock(x, y, z, block)
}

// ─── LevelService adapter ───

type sessionLevels struct{ backend classic.SessionBackend }

func (s sessionLevels) Goto(name string) bool {
	return s.backend.GotoLevel(name)
}

func (s sessionLevels) MainLevel() string {
	return s.backend.MainLevelName()
}

func (s sessionLevels) LoadLevel(name string) bool {
	return s.backend.LoadLevel(name)
}

func (s sessionLevels) UnloadLevel(name string) bool {
	return s.backend.UnloadLevel(name)
}

func (s sessionLevels) ReloadLevel() bool {
	return s.backend.ReloadCurrentLevel()
}

func (s sessionLevels) ListLevels() []string {
	return s.backend.ListLoadedLevels()
}

func (s sessionLevels) ListLevelFiles() []string {
	return s.backend.ListLevelFiles()
}

func (s sessionLevels) PhysicsMode() int {
	return s.backend.CurrentPhysicsMode()
}

func (s sessionLevels) SetPhysicsMode(mode int) {
	s.backend.SetCurrentPhysicsMode(mode)
}

// ─── LevelEnvService adapter ───

type sessionLevelEnv struct{ backend classic.SessionBackend }

func (s sessionLevelEnv) GetEnvColor(slot int) (r, g, b byte, set bool) {
	return s.backend.GetEnvColor(slot)
}

func (s sessionLevelEnv) SetEnvColor(slot int, r, g, b byte) {
	s.backend.SetLevelEnvColor(slot, r, g, b)
}

func (s sessionLevelEnv) Weather() int {
	return s.backend.GetWeather()
}

func (s sessionLevelEnv) SetWeather(weather int) {
	s.backend.SetLevelWeather(weather)
}

func (s sessionLevelEnv) MOTD() string {
	return s.backend.GetLevelMOTD()
}

func (s sessionLevelEnv) SetMOTD(motd string) {
	s.backend.SetLevelMOTD(motd)
}

// ─── PlayerLookup adapter ───

type sessionPlayerDB struct{ backend classic.SessionBackend }

func (s sessionPlayerDB) Lookup(name string) *playerdb.PlayerEntry {
	return s.backend.PlayerDBLookup(name)
}

// ─── ServerInfo adapter ───

type sessionSrvInfo struct{ backend classic.SessionBackend }

func (s sessionSrvInfo) ServerName() string    { return s.backend.ServerName() }
func (s sessionSrvInfo) MOTD() string          { return s.backend.ServerMOTD() }
func (s sessionSrvInfo) OnlineCount() int      { return s.backend.OnlinePlayerCount() }
func (s sessionSrvInfo) MaxPlayers() int       { return s.backend.MaxPlayersCount() }
func (s sessionSrvInfo) LevelCount() int       { return s.backend.LoadedLevelCount() }
func (s sessionSrvInfo) Uptime() time.Duration { return s.backend.ServerUptime() }
