package app

import (
	"errors"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/ranks"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin/playerdb"
)

func TestBuildCommandContextAdapters(t *testing.T) {
	backend := &fakeSessionBackend{}
	ctx := buildCommandContext(backend)

	if ctx.Username != "alice" || ctx.Position.X != 1 || ctx.Yaw != 4 || ctx.Pitch != 5 {
		t.Fatalf("context identity = %+v yaw=%d pitch=%d", ctx.Position, ctx.Yaw, ctx.Pitch)
	}
	if !ctx.Authority.CanAdmin() {
		t.Fatal("Authority.CanAdmin returned false")
	}
	if !ctx.World.SetBlock(1, 2, 3, 4) || !backend.blockApplied {
		t.Fatal("World.SetBlock did not call backend")
	}
	if !ctx.World.MovePlayer(1, 2, 3, 4, 5) || !ctx.World.SetSpawn(1, 2, 3, 4, 5) {
		t.Fatal("world movement/spawn adapters failed")
	}
	if !ctx.World.GenerateWorld("new", "Flat", 4, 4, 4, "1") || !ctx.Persistence.SaveState() {
		t.Fatal("generate/save adapters failed")
	}

	if x, y, z, yaw, pitch := ctx.Teleport.SpawnPoint(); x != 1 || y != 2 || z != 3 || yaw != 4 || pitch != 5 {
		t.Fatalf("SpawnPoint = %d %d %d %d %d", x, y, z, yaw, pitch)
	}
	if !ctx.Teleport.TeleportToPlayer("bob") || !ctx.Teleport.SummonPlayer("bob") || !ctx.Teleport.Back() {
		t.Fatal("teleport adapters failed")
	}
	ctx.Chat.Me("waves")
	if !ctx.Chat.Whisper("bob", "hi") {
		t.Fatal("Whisper adapter failed")
	}
	if ignored, ok := ctx.Chat.Ignore("bob"); !ignored || !ok {
		t.Fatalf("Ignore = %v %v", ignored, ok)
	}

	called := false
	if !ctx.Draw.StartSelection(2, func(marks [][3]int) { called = true }) || !called {
		t.Fatal("StartSelection adapter failed")
	}
	if block, ok := ctx.Draw.GetBlockAt(1, 2, 3); !ok || block != 7 {
		t.Fatalf("GetBlockAt = %d ok=%v", block, ok)
	}
	if !ctx.Draw.PlaceBlock(1, 2, 3, 4) || ctx.Draw.PasteAt([3]int{1, 2, 3}, true) != 1 {
		t.Fatal("draw placement adapters failed")
	}
	if width, height, length := ctx.Draw.LevelDims(); width != 8 || height != 9 || length != 10 {
		t.Fatalf("LevelDims = %d %d %d", width, height, length)
	}
	if !ctx.Draw.CopyRegion([3]int{}, [3]int{1, 1, 1}) || !ctx.Draw.HasClipboard() {
		t.Fatal("clipboard adapters failed")
	}
	if !ctx.Draw.SetSpecialBlock(1, 2, 3, command.SpecialBlockEntry{Type: 1}) {
		t.Fatal("SetSpecialBlock adapter failed")
	}
	ctx.Draw.BeginBatch()
	ctx.Draw.RecordChange(1, 2, 3, 4, 5)
	ctx.Draw.CommitBatch()
	if len(ctx.Draw.Undo()) != 1 || len(ctx.Draw.Redo()) != 1 {
		t.Fatal("undo/redo adapters failed")
	}
	if ctx.Draw.DrawLimit() != 123 || !ctx.Draw.CanPlace(1) || !ctx.Draw.CanDelete(1) {
		t.Fatal("draw permission adapters failed")
	}

	if !ctx.Moderation.KickPlayer("bob", "why") ||
		!ctx.Moderation.BanPlayer("bob", "why") ||
		!ctx.Moderation.UnbanPlayer("bob") ||
		!ctx.Moderation.WhitelistEnabled() ||
		!ctx.Moderation.SetWhitelistEnabled(true) ||
		!ctx.Moderation.WhitelistAdd("bob") ||
		!ctx.Moderation.WhitelistRemove("bob") ||
		!ctx.Moderation.MutePlayer("bob") ||
		!ctx.Moderation.UnmutePlayer("bob") ||
		!ctx.Moderation.FreezePlayer("bob") ||
		!ctx.Moderation.UnfreezePlayer("bob") {
		t.Fatal("moderation adapters failed")
	}
	if afk, ok := ctx.Moderation.ToggleAFK("bob"); !afk || !ok {
		t.Fatalf("ToggleAFK = %v %v", afk, ok)
	}
	if hidden, ok := ctx.Moderation.ToggleHide("bob"); !hidden || !ok {
		t.Fatalf("ToggleHide = %v %v", hidden, ok)
	}

	if len(ctx.Players.ListPlayers()) != 1 || len(ctx.Players.ListWhitelisted()) != 1 {
		t.Fatal("directory adapters failed")
	}
	def := blocks.Default(66)
	if !ctx.BlockDefs.AddBlockDef(def) || !ctx.BlockDefs.RemoveBlockDef(66) || ctx.BlockDefs.FreeBlockID() != 67 {
		t.Fatal("block definition mutating adapters failed")
	}
	if got, ok := ctx.BlockDefs.GetBlockDef(66); !ok || got.ID != 66 || len(ctx.BlockDefs.ListBlockDefs()) != 1 {
		t.Fatal("block definition lookup adapters failed")
	}

	if len(ctx.BlockDB.ChangesAt(1, 2, 3)) != 1 || len(ctx.BlockDB.ChangesBy("alice", time.Time{}, 1)) != 1 {
		t.Fatal("blockdb change adapters failed")
	}
	if ctx.BlockDB.Count() != 3 || !ctx.BlockDB.Enabled() {
		t.Fatal("blockdb count/enabled adapters failed")
	}
	ctx.BlockDB.SetEnabled(false)
	if err := ctx.BlockDB.Clear(); err != nil {
		t.Fatalf("BlockDB.Clear: %v", err)
	}
	if !ctx.BlockDB.RevertBlock(1, 2, 3, 4) {
		t.Fatal("BlockDB.RevertBlock adapter failed")
	}

	if !ctx.Levels.Goto("main") || ctx.Levels.MainLevel() != "main" ||
		!ctx.Levels.LoadLevel("main") || !ctx.Levels.UnloadLevel("old") ||
		!ctx.Levels.ReloadLevel() || len(ctx.Levels.ListLevels()) != 1 ||
		len(ctx.Levels.ListLevelFiles()) != 1 || ctx.Levels.PhysicsMode() != 2 {
		t.Fatal("level adapters failed")
	}
	ctx.Levels.SetPhysicsMode(3)
	if r, g, b, set := ctx.LevelEnv.GetEnvColor(0); !set || r != 1 || g != 2 || b != 3 {
		t.Fatalf("env color = %d %d %d set=%v", r, g, b, set)
	}
	ctx.LevelEnv.SetEnvColor(0, 4, 5, 6)
	if ctx.LevelEnv.Weather() != 1 {
		t.Fatal("weather adapter failed")
	}
	ctx.LevelEnv.SetWeather(2)
	if ctx.LevelEnv.MOTD() != "motd" {
		t.Fatal("motd adapter failed")
	}
	ctx.LevelEnv.SetMOTD("new motd")

	if ctx.PlayerDB.Lookup("alice").Name != "alice" ||
		ctx.ServerInfo.ServerName() != "Solar" ||
		ctx.ServerInfo.MOTD() != "MOTD" ||
		ctx.ServerInfo.OnlineCount() != 1 ||
		ctx.ServerInfo.MaxPlayers() != 64 ||
		ctx.ServerInfo.LevelCount() != 2 ||
		ctx.ServerInfo.Uptime() <= 0 {
		t.Fatal("playerdb/serverinfo adapters failed")
	}
	if ctx.RankLevel() != ranks.PermOperator ||
		ctx.Ranks.Get("operator").Permission != ranks.PermOperator ||
		ctx.Ranks.GetByPerm(ranks.PermOperator).Name != "operator" ||
		len(ctx.Ranks.All()) != 1 ||
		ctx.Ranks.GetPlayerRank("alice") != ranks.PermOperator ||
		!ctx.Ranks.SetPlayerRank("bob", ranks.PermBuilder) {
		t.Fatal("rank adapters failed")
	}
	if ctx.Ranks.Get("missing") != nil || ctx.Ranks.GetByPerm(999) != nil {
		t.Fatal("missing rank adapters should return nil")
	}
	if ctx.Tr("key") != "tr:key" {
		t.Fatal("translator adapter failed")
	}
}

type fakeSessionBackend struct {
	blockApplied bool
}

func (f *fakeSessionBackend) CurrentUsername() string { return "alice" }
func (f *fakeSessionBackend) CurrentLocation() (world.Spawn, byte, byte) {
	return world.Spawn{X: 1, Y: 2, Z: 3, Yaw: 4, Pitch: 5}, 4, 5
}
func (f *fakeSessionBackend) IsOperator() bool { return true }
func (f *fakeSessionBackend) Translate(key string, args ...any) string {
	return "tr:" + key
}
func (f *fakeSessionBackend) ApplyBlockChange(x, y, z int, blockID byte, echo bool) error {
	f.blockApplied = echo && x == 1 && y == 2 && z == 3 && blockID == 4
	if !f.blockApplied {
		return errors.New("bad block change")
	}
	return nil
}
func (f *fakeSessionBackend) TeleportSelf(x, y, z int, yaw, pitch byte) bool { return true }
func (f *fakeSessionBackend) SetSpawn(spawn world.Spawn) bool                { return true }
func (f *fakeSessionBackend) GenerateWorld(name, theme string, width, height, length int, seed string) bool {
	return true
}
func (f *fakeSessionBackend) SaveState() bool           { return true }
func (f *fakeSessionBackend) PersistPlayerPolicy() bool { return true }
func (f *fakeSessionBackend) SpawnPoint() (int, int, int, byte, byte) {
	return 1, 2, 3, 4, 5
}
func (f *fakeSessionBackend) TeleportToPlayer(name string) bool { return true }
func (f *fakeSessionBackend) SummonPlayer(name string) bool     { return true }
func (f *fakeSessionBackend) BackToLastPos() bool               { return true }
func (f *fakeSessionBackend) MeAction(action string)            {}
func (f *fakeSessionBackend) WhisperTo(targetName, msg string) bool {
	return true
}
func (f *fakeSessionBackend) IgnorePlayer(name string) (bool, bool) { return true, true }
func (f *fakeSessionBackend) StartSelection(markCount int, callback func(marks [][3]int)) bool {
	callback([][3]int{{1, 2, 3}})
	return true
}
func (f *fakeSessionBackend) GetBlockAt(x, y, z int) (byte, bool) { return 7, true }
func (f *fakeSessionBackend) PlaceBlock(x, y, z int, block byte) bool {
	return true
}
func (f *fakeSessionBackend) LevelDims() (int, int, int) { return 8, 9, 10 }
func (f *fakeSessionBackend) CopyRegion(min, max [3]int) bool {
	return true
}
func (f *fakeSessionBackend) HasClipboard() bool { return true }
func (f *fakeSessionBackend) PasteAt(origin [3]int, pasteAir bool) int {
	return 1
}
func (f *fakeSessionBackend) SetSpecialBlock(x, y, z int, entry command.SpecialBlockEntry) bool {
	return true
}
func (f *fakeSessionBackend) BeginBatch() {}
func (f *fakeSessionBackend) RecordChange(x, y, z int, oldBlock, newBlock byte) {
}
func (f *fakeSessionBackend) CommitBatch() {}
func (f *fakeSessionBackend) UndoBatch() []command.UndoChange {
	return []command.UndoChange{{X: 1}}
}
func (f *fakeSessionBackend) RedoBatch() []command.UndoChange {
	return []command.UndoChange{{X: 1}}
}
func (f *fakeSessionBackend) PlayerRank() int { return ranks.PermOperator }
func (f *fakeSessionBackend) RankGetPlayer(name string) int {
	return ranks.PermOperator
}
func (f *fakeSessionBackend) RankGet(name string) *ranks.Rank {
	if name == "missing" {
		return nil
	}
	return &ranks.Rank{Name: "operator", Permission: ranks.PermOperator}
}
func (f *fakeSessionBackend) RankGetByPerm(perm int) *ranks.Rank {
	if perm == ranks.PermOperator {
		return &ranks.Rank{Name: "operator", Permission: ranks.PermOperator}
	}
	return nil
}
func (f *fakeSessionBackend) RankAll() []*ranks.Rank {
	return []*ranks.Rank{{Name: "operator", Permission: ranks.PermOperator}}
}
func (f *fakeSessionBackend) RankSetPlayer(name string, perm int) bool { return true }
func (f *fakeSessionBackend) DrawLimit() int                           { return 123 }
func (f *fakeSessionBackend) CanPlaceBlock(block byte) bool            { return true }
func (f *fakeSessionBackend) CanDeleteBlock(block byte) bool           { return true }
func (f *fakeSessionBackend) KickPlayer(name, reason string) bool      { return true }
func (f *fakeSessionBackend) BanPlayer(name, reason string) bool       { return true }
func (f *fakeSessionBackend) UnbanPlayer(name string) bool             { return true }
func (f *fakeSessionBackend) WhitelistEnabled() bool                   { return true }
func (f *fakeSessionBackend) WhitelistAdd(name string) bool            { return true }
func (f *fakeSessionBackend) WhitelistRemove(name string) bool         { return true }
func (f *fakeSessionBackend) SetWhitelistEnabled(enabled bool) bool    { return true }
func (f *fakeSessionBackend) MutePlayer(name string) bool              { return true }
func (f *fakeSessionBackend) UnmutePlayer(name string) bool            { return true }
func (f *fakeSessionBackend) FreezePlayer(name string) bool            { return true }
func (f *fakeSessionBackend) UnfreezePlayer(name string) bool          { return true }
func (f *fakeSessionBackend) ToggleAFK(name string) (bool, bool)       { return true, true }
func (f *fakeSessionBackend) ToggleHide(name string) (bool, bool)      { return true, true }
func (f *fakeSessionBackend) OnlineNames() []string                    { return []string{"alice"} }
func (f *fakeSessionBackend) WhitelistNames() []string                 { return []string{"alice"} }
func (f *fakeSessionBackend) AddBlockDef(def blocks.BlockDefinition) bool {
	return true
}
func (f *fakeSessionBackend) RemoveBlockDef(id byte) bool { return true }
func (f *fakeSessionBackend) GetBlockDef(id byte) (blocks.BlockDefinition, bool) {
	return blocks.Default(id), true
}
func (f *fakeSessionBackend) ListBlockDefs() []blocks.BlockDefinition {
	return []blocks.BlockDefinition{blocks.Default(66)}
}
func (f *fakeSessionBackend) FreeBlockID() byte { return 67 }
func (f *fakeSessionBackend) BlockDBChangesAt(x, y, z int) []command.BlockDBEntry {
	return []command.BlockDBEntry{{X: x, Y: y, Z: z}}
}
func (f *fakeSessionBackend) BlockDBChangesBy(playerName string, since time.Time, max int) []command.BlockDBEntry {
	return []command.BlockDBEntry{{PlayerName: playerName}}
}
func (f *fakeSessionBackend) BlockDBCount() int64    { return 3 }
func (f *fakeSessionBackend) BlockDBEnabled() bool   { return true }
func (f *fakeSessionBackend) BlockDBSetEnabled(bool) {}
func (f *fakeSessionBackend) BlockDBClear() error    { return nil }
func (f *fakeSessionBackend) BlockDBRevertBlock(x, y, z int, block byte) bool {
	return true
}
func (f *fakeSessionBackend) GotoLevel(name string) bool     { return true }
func (f *fakeSessionBackend) MainLevelName() string          { return "main" }
func (f *fakeSessionBackend) LoadLevel(name string) bool     { return true }
func (f *fakeSessionBackend) UnloadLevel(name string) bool   { return true }
func (f *fakeSessionBackend) ReloadCurrentLevel() bool       { return true }
func (f *fakeSessionBackend) ListLoadedLevels() []string     { return []string{"main"} }
func (f *fakeSessionBackend) ListLevelFiles() []string       { return []string{"main"} }
func (f *fakeSessionBackend) CurrentPhysicsMode() int        { return 2 }
func (f *fakeSessionBackend) SetCurrentPhysicsMode(mode int) {}
func (f *fakeSessionBackend) GetEnvColor(slot int) (byte, byte, byte, bool) {
	return 1, 2, 3, true
}
func (f *fakeSessionBackend) SetLevelEnvColor(slot int, r, g, b byte) {}
func (f *fakeSessionBackend) GetWeather() int                         { return 1 }
func (f *fakeSessionBackend) SetLevelWeather(weather int)             {}
func (f *fakeSessionBackend) GetLevelMOTD() string                    { return "motd" }
func (f *fakeSessionBackend) SetLevelMOTD(motd string)                {}
func (f *fakeSessionBackend) PlayerDBLookup(name string) *playerdb.PlayerEntry {
	return &playerdb.PlayerEntry{Name: name}
}
func (f *fakeSessionBackend) ServerName() string          { return "Solar" }
func (f *fakeSessionBackend) ServerMOTD() string          { return "MOTD" }
func (f *fakeSessionBackend) OnlinePlayerCount() int      { return 1 }
func (f *fakeSessionBackend) MaxPlayersCount() int        { return 64 }
func (f *fakeSessionBackend) LoadedLevelCount() int       { return 2 }
func (f *fakeSessionBackend) ServerUptime() time.Duration { return time.Second }
