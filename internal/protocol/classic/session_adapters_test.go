package classic

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/ranks"
	roompkg "github.com/solar-mc/solar/internal/session"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
	playerdbplugin "github.com/solar-mc/solar/plugin/playerdb"
)

func TestSessionDrawClipboardUndoAndEnvAdapters(t *testing.T) {
	s := newCoverageSession(t, "alice")

	if err := s.ApplyBlockChange(1, 1, 1, 2, true); err != nil {
		t.Fatalf("ApplyBlockChange: %v", err)
	}
	if got, ok := s.worlds.BlockAt(1, 1, 1); !ok || got != 2 {
		t.Fatalf("world block = %d ok=%v", got, ok)
	}
	assertPacketCount(t, s, 1)

	if !s.PlaceBlock(1, 1, 2, 3) {
		t.Fatal("PlaceBlock returned false")
	}
	if got, ok := s.GetBlockAt(1, 1, 2); !ok || got != 3 {
		t.Fatalf("GetBlockAt = %d ok=%v", got, ok)
	}
	if width, height, length := s.LevelDims(); width == 0 || height == 0 || length == 0 {
		t.Fatalf("LevelDims = %d %d %d", width, height, length)
	}

	if !s.CopyRegion([3]int{1, 1, 1}, [3]int{1, 1, 2}) || !s.HasClipboard() {
		t.Fatal("CopyRegion did not create clipboard")
	}
	s.BeginBatch()
	if pasted := s.PasteAt([3]int{2, 1, 1}, true); pasted != 2 {
		t.Fatalf("PasteAt = %d, want 2", pasted)
	}
	s.CommitBatch()
	undo := s.UndoBatch()
	if len(undo) != 2 {
		t.Fatalf("UndoBatch = %d changes, want 2", len(undo))
	}
	redo := s.RedoBatch()
	if len(redo) != 2 {
		t.Fatalf("RedoBatch = %d changes, want 2", len(redo))
	}
	assertPacketCount(t, s, 3)

	if s.SetSpecialBlock(1, 2, 3, command.SpecialBlockEntry{Type: int(blocks.SpecialDoor)}) {
		t.Fatal("SetSpecialBlock succeeded without registry")
	}
	s.specialBlocks = blocks.NewSpecialRegistry()
	if !s.SetSpecialBlock(1, 2, 3, command.SpecialBlockEntry{Type: int(blocks.SpecialMessage), Message: "hello"}) {
		t.Fatal("SetSpecialBlock returned false")
	}
	if got := s.specialBlocks.Get(1, 2, 3); got == nil || got.Message != "hello" {
		t.Fatalf("special block = %+v", got)
	}

	s.SetLevelEnvColor(0, 1, 2, 3)
	if r, g, b, set := s.GetEnvColor(0); !set || r != 1 || g != 2 || b != 3 {
		t.Fatalf("env color = %d %d %d set=%v", r, g, b, set)
	}
	s.SetLevelWeather(2)
	if got := s.GetWeather(); got != 2 {
		t.Fatalf("weather = %d, want 2", got)
	}
	s.SetLevelMOTD("level motd")
	if got := s.GetLevelMOTD(); got != "level motd" {
		t.Fatalf("motd = %q", got)
	}
	assertPacketCount(t, s, 2)
}

func TestSessionRankPolicyBlockDefAndBlockDBAdapters(t *testing.T) {
	s := newCoverageSession(t, "alice")

	s.players.AddOperators("alice")
	if got := s.PlayerRank(); got != ranks.PermOperator {
		t.Fatalf("operator fallback PlayerRank = %d", got)
	}
	if got := s.RankGetPlayer("alice"); got != ranks.PermOperator {
		t.Fatalf("operator fallback RankGetPlayer = %d", got)
	}

	playerDB := newMemoryPlayerDB()
	s.players.RemoveOperator("alice")
	s.rankRegistry = ranks.NewRegistry()
	s.rankRegistry.SetPlayerDB(playerDB)
	if !s.RankSetPlayer("bob", ranks.PermBuilder) {
		t.Fatal("RankSetPlayer returned false")
	}
	if got := s.RankGetPlayer("bob"); got != ranks.PermBuilder {
		t.Fatalf("bob rank = %d, want builder", got)
	}
	if s.RankGet("builder") == nil || s.RankGetByPerm(ranks.PermBuilder) == nil || len(s.RankAll()) == 0 {
		t.Fatal("rank lookup returned nil")
	}
	if got := s.DrawLimit(); got != 1 {
		t.Fatalf("alice draw limit = %d, want guest limit 1", got)
	}
	if s.CanPlaceBlock(182) {
		t.Fatal("guest/operator fallback with rank registry should not place operator-only TNT")
	}
	if !s.CanDeleteBlock(182) {
		t.Fatal("CanDeleteBlock returned false")
	}

	s.blockDefs = blocks.NewRegistry(t.TempDir())
	def := blocks.Default(66)
	def.Name = "unit"
	if !s.AddBlockDef(def) || s.FreeBlockID() == 66 {
		t.Fatal("AddBlockDef did not store custom block")
	}
	if got, ok := s.GetBlockDef(66); !ok || got.Name != "unit" {
		t.Fatalf("GetBlockDef = %+v ok=%v", got, ok)
	}
	if defs := s.ListBlockDefs(); len(defs) != 1 || defs[0].ID != 66 {
		t.Fatalf("ListBlockDefs = %+v", defs)
	}
	if !s.RemoveBlockDef(66) || s.RemoveBlockDef(66) {
		t.Fatal("RemoveBlockDef returned unexpected values")
	}
	assertPacketCount(t, s, 0)

	if s.WhitelistEnabled() {
		t.Fatal("whitelist should start disabled")
	}
	if !s.WhitelistAdd("bob") || !s.SetWhitelistEnabled(true) || !s.WhitelistEnabled() {
		t.Fatal("whitelist add/enable failed")
	}
	if names := s.WhitelistNames(); len(names) != 1 || names[0] != "bob" {
		t.Fatalf("WhitelistNames = %v", names)
	}
	if !s.WhitelistRemove("bob") {
		t.Fatal("WhitelistRemove returned false")
	}
	s.players.Ban("bob", "test")
	if !s.UnbanPlayer("bob") {
		t.Fatal("UnbanPlayer returned false")
	}

	db := &coveragePluginBlockDB{
		entries: []plugin.BlockEntry{{
			PlayerID: 7,
			Time:     time.Now(),
			X:        1,
			Y:        2,
			Z:        3,
			OldBlock: 1,
			NewBlock: 2,
			Flags:    plugin.BlockManualPlace,
		}},
		enabled: true,
	}
	s.blockDB = db
	s.nameConv = blocks.NewNameConverter()
	if got := s.BlockDBChangesAt(1, 2, 3); len(got) != 1 || got[0].PlayerName != "ID:7" {
		t.Fatalf("BlockDBChangesAt = %+v", got)
	}
	if got := s.BlockDBChangesBy("alice", time.Time{}, 1); len(got) != 1 {
		t.Fatalf("BlockDBChangesBy = %+v", got)
	}
	if s.BlockDBCount() != int64(len(db.entries)) || !s.BlockDBEnabled() {
		t.Fatal("BlockDB count/enabled returned unexpected values")
	}
	s.BlockDBSetEnabled(false)
	if db.enabled {
		t.Fatal("BlockDBSetEnabled(false) did not update db")
	}
	if err := s.BlockDBClear(); err != nil || !db.cleared {
		t.Fatalf("BlockDBClear err=%v cleared=%v", err, db.cleared)
	}
	if !s.BlockDBRevertBlock(1, 1, 1, 4) {
		t.Fatal("BlockDBRevertBlock returned false")
	}
	assertPacketCount(t, s, 1)
}

func TestSessionModerationChatTeleportAndInfoAdapters(t *testing.T) {
	alice := newCoverageSession(t, "alice")
	bob := newCoverageSession(t, "bob")
	bob.worlds = alice.worlds
	bobID, ok := bob.entities.Add("bob", entity.Position{X: 5 * coordScale, Y: 6 * coordScale, Z: 7 * coordScale})
	if !ok {
		t.Fatal("add second bob entity failed")
	}
	bob.setIdentity("bob", bobID, true)
	bob.players.Remove("bob")
	bob.players.Add("bob", bobID)
	if !bob.entities.SetLocation(bob.currentEntityID(), entity.Position{X: 5 * coordScale, Y: 6 * coordScale, Z: 7 * coordScale}, 9, 10) {
		t.Fatal("set bob location failed")
	}
	room := roompkg.NewRoom[*session]()
	alice.room = room
	bob.room = room
	room.Join(alice)
	room.Join(bob)

	if !alice.TeleportToPlayer("bob") {
		t.Fatal("TeleportToPlayer returned false")
	}
	if snap, ok := alice.entities.Get(alice.currentEntityID()); !ok ||
		snap.Pos.X != 5*coordScale || snap.Pos.Y != 6*coordScale || snap.Pos.Z != 7*coordScale {
		t.Fatalf("alice position after TeleportToPlayer = %+v ok=%v", snap, ok)
	}
	if !alice.SummonPlayer("bob") {
		t.Fatal("SummonPlayer returned false")
	}
	if snap, ok := bob.entities.Get(bob.currentEntityID()); !ok ||
		snap.Pos.X != 5*coordScale || snap.Pos.Y != 6*coordScale || snap.Pos.Z != 7*coordScale {
		t.Fatalf("bob position after SummonPlayer = %+v ok=%v", snap, ok)
	}
	if !alice.BackToLastPos() {
		t.Fatal("BackToLastPos returned false")
	}
	if snap, ok := alice.entities.Get(alice.currentEntityID()); !ok ||
		snap.Pos.X != coordScale || snap.Pos.Y != 2*coordScale || snap.Pos.Z != 3*coordScale {
		t.Fatalf("alice position after BackToLastPos = %+v ok=%v", snap, ok)
	}
	assertPacketCount(t, alice, 2)
	assertPacketCount(t, bob, 1)

	alice.MeAction("waves")
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)

	if !alice.WhisperTo("bob", "secret") {
		t.Fatal("WhisperTo returned false")
	}
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)
	if ignored, ok := bob.IgnorePlayer("alice"); !ok || !ignored || !bob.isIgnoring("alice") {
		t.Fatalf("IgnorePlayer ignored=%v ok=%v", ignored, ok)
	}
	if alice.WhisperTo("bob", "blocked") {
		t.Fatal("WhisperTo succeeded while target ignored sender")
	}

	if !alice.MutePlayer("bob") || !bob.IsMuted() {
		t.Fatal("MutePlayer failed")
	}
	if !alice.UnmutePlayer("bob") || bob.IsMuted() {
		t.Fatal("UnmutePlayer failed")
	}
	if !alice.FreezePlayer("bob") || !bob.IsFrozen() {
		t.Fatal("FreezePlayer failed")
	}
	if !alice.UnfreezePlayer("bob") || bob.IsFrozen() {
		t.Fatal("UnfreezePlayer failed")
	}
	if afk, ok := alice.ToggleAFK("bob"); !ok || !afk || !bob.IsAfk() {
		t.Fatalf("ToggleAFK afk=%v ok=%v", afk, ok)
	}
	if hidden, ok := alice.ToggleHide("bob"); !ok || !hidden || !bob.IsHidden() {
		t.Fatalf("ToggleHide hidden=%v ok=%v", hidden, ok)
	}

	playerDB := newMemoryPlayerDB()
	playerDB.Save(&playerdbplugin.PlayerEntry{Name: "alice", LoginCount: 2})
	alice.playerDB = playerDB
	if got := alice.PlayerDBLookup("alice"); got == nil || got.LoginCount != 2 {
		t.Fatalf("PlayerDBLookup = %+v", got)
	}
	alice.serverName = "SolarTest"
	alice.motd = "MOTD"
	alice.maxPlayers = 42
	alice.listLoadedLevels = func() []string { return []string{"main", "test"} }
	if alice.ServerName() != "SolarTest" || alice.ServerMOTD() != "MOTD" || alice.MaxPlayersCount() != 42 {
		t.Fatal("server info adapters returned unexpected values")
	}
	if alice.OnlinePlayerCount() != 1 || alice.LoadedLevelCount() != 2 || alice.ServerUptime() <= 0 {
		t.Fatal("count/uptime adapters returned unexpected values")
	}
	if names := alice.OnlineNames(); len(names) != 1 || names[0] != "alice" {
		t.Fatalf("OnlineNames = %v", names)
	}
}

func TestSessionLevelCallbacksAndPluginPlayerMethods(t *testing.T) {
	s := newCoverageSession(t, "alice")

	if x, y, z, yaw, pitch := s.SpawnPoint(); x == 0 && y == 0 && z == 0 && yaw == 0 && pitch == 0 {
		t.Fatal("SpawnPoint returned empty spawn")
	}
	if !s.StartSelection(2, func(marks [][3]int) {}) || s.StartSelection(0, func(marks [][3]int) {}) {
		t.Fatal("StartSelection returned unexpected values")
	}
	generator.RegisterDefaults()
	if !s.GenerateWorld("flatunit", "Flat", 4, 4, 4, "2") {
		t.Fatal("GenerateWorld returned false")
	}
	assertPacketCountAtLeast(t, s, 1)
	if !s.PersistPlayerPolicy() {
		t.Fatal("PersistPlayerPolicy returned false")
	}
	path := t.TempDir() + "/world.swld"
	if err := s.worlds.Save(path); err != nil {
		t.Fatalf("save world: %v", err)
	}
	s.worldPath = path
	if !s.ReloadCurrentLevel() {
		t.Fatal("ReloadCurrentLevel returned false")
	}
	s.SetCurrentPhysicsMode(2)

	s.gotoLevel = func(p plugin.Player, name string) bool { return p.Name() == "alice" && name == "test" }
	s.mainLevelName = func() string { return "main" }
	s.loadLevel = func(name string) bool { return name == "test" }
	s.unloadLevel = func(name string) bool { return name == "test" }
	s.listLoadedLevels = func() []string { return []string{"main", "test"} }
	s.listLevelFiles = func() []string { return []string{"main", "test"} }

	if !s.GotoLevel("test") || s.MainLevelName() != "main" || !s.LoadLevel("test") || !s.UnloadLevel("test") {
		t.Fatal("level callbacks returned unexpected values")
	}
	if !slicesEqual(s.ListLoadedLevels(), []string{"main", "test"}) || !slicesEqual(s.ListLevelFiles(), []string{"main", "test"}) {
		t.Fatalf("level lists = %v / %v", s.ListLoadedLevels(), s.ListLevelFiles())
	}
	if s.CurrentPhysicsMode() != 0 {
		t.Fatal("CurrentPhysicsMode should be no-op zero")
	}

	if s.Name() != "alice" || s.EntityID() == 0 {
		t.Fatal("Name/EntityID returned unexpected values")
	}
	s.SendCpeMessage(1, "status")
	s.SetColor("&e")
	if s.Color() != "&e" {
		t.Fatalf("Color = %q", s.Color())
	}
	s.SetModel("chicken")
	if s.Model() != "chicken" {
		t.Fatalf("Model = %q", s.Model())
	}
	s.SetSkin("skin-url")
	s.SetHidden(true)
	if !s.IsHidden() {
		t.Fatal("SetHidden(true) failed")
	}
	s.SetMuted(true)
	s.SetFrozen(true)
	if !s.IsMuted() || !s.IsFrozen() {
		t.Fatal("SetMuted/SetFrozen failed")
	}
	s.SetAfk(true)
	if !s.IsAfk() || s.AfkSince().IsZero() {
		t.Fatal("SetAfk(true) failed")
	}
	s.touchLastAction()
	if s.LastAction().IsZero() {
		t.Fatal("touchLastAction failed")
	}
	s.SetAllowBuild(true)
	if !s.AllowBuild() {
		t.Fatal("SetAllowBuild failed")
	}
	if !s.Teleport(2, 3, 4, 5, 6) {
		t.Fatal("Teleport returned false")
	}
	if !s.SetBlock(1, 1, 2, 6) {
		t.Fatal("SetBlock returned false")
	}
	s.setCPESupport(map[string]uint32{cpeExtSelectionCuboid: 1})
	if !s.SupportsCPE(cpeExtSelectionCuboid) {
		t.Fatal("SupportsCPE returned false")
	}
	s.SendRawPacket([]byte{0x99})
	if !s.ChangeBlock(1, 1, 1, 5) {
		t.Fatal("ChangeBlock returned false")
	}
	s.RevertBlock(1, 1, 1)
	if !s.MakeSelection(1, "sel", 0, 0, 0, 1, 1, 1, 255, 0, 0) {
		t.Fatal("MakeSelection returned false")
	}
	if !s.ClearSelection(1) {
		t.Fatal("ClearSelection returned false")
	}
	s.SetLanguage("ru")
	if s.Language() != "ru" || s.Tr("missing.key") != "missing.key" || s.Translate("missing.key") != "missing.key" {
		t.Fatal("language/translation helpers returned unexpected values")
	}
	if !s.Kill(0) {
		t.Fatal("Kill returned false")
	}

	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()
	s.conn = server
	if !strings.Contains(s.IP(), "pipe") {
		t.Fatalf("IP = %q", s.IP())
	}
	assertPacketCountAtLeast(t, s, 10)

	kicked := newCoverageSession(t, "kickme")
	kicked.Kick("bye")
	assertPacketCount(t, kicked, 1)
}

func TestSessionCPESenders(t *testing.T) {
	s := newCoverageSession(t, "alice")
	if s.CPE() != s {
		t.Fatal("CPE did not return session")
	}
	s.SetEnvColor(0, 1, 2, 3)
	assertPacketCount(t, s, 0)

	exts := make(map[string]uint32)
	for _, ext := range []string{
		cpeExtEnvColors,
		cpeExtEnvWeatherType,
		cpeExtHackControl,
		cpeExtClickDistance,
		cpeExtTextHotkey,
		cpeExtHeldBlock,
		cpeExtBlockPermissions,
		cpeExtChangeModel,
		cpeExtEnvMapAppearance,
		cpeExtInventoryOrder,
		cpeExtSetHotbar,
		cpeExtSetSpawnpoint,
		cpeExtCustomParticles,
		cpeExtSelectionCuboid,
		cpeExtEnvMapAspect,
		cpeExtEntityProperty,
		cpeExtPluginMessages,
		cpeExtLightingMode,
	} {
		exts[ext] = 1
	}
	s.setCPESupport(exts)

	s.SetEnvColor(0, 1, 2, 3)
	s.SetWeather(1)
	s.SetHackControl(true, false, true, false, true)
	s.SetClickDistance(10)
	s.SetTextHotkey('F', "/spawn", 0)
	s.HoldThis(1, true)
	s.SetBlockPermission(1, true, false)
	s.ChangeModel(1, "humanoid")
	s.SetMapAppearance("texture.png", 4)
	s.SetInventoryOrder(1, 2)
	s.SetHotbar(0, 1)
	s.SetSpawnpoint(1, 2, 3, 4, 5)
	s.SendEffect(1, 2, 3, 4)
	s.SendSelection(1, "sel", 0, 0, 0, 1, 1, 1, 255, 0, 0)
	s.RemoveSelection(1)
	s.SetMapProperty(0, 10)
	s.SetEntityProperty(1, 0, 20)
	s.SendPluginMessage(1, []byte{1, 2, 3})
	s.SetLightingMode(1, true)

	assertPacketCount(t, s, 19)
}

func TestCodecSettersBroadcastAndMapChange(t *testing.T) {
	codec := NewCodec("Solar", "MOTD", world.NewManager(), player.NewRegistry(), entity.NewManager(), command.NewRegistry())
	codec.SetLogger(nil)
	codec.SetOutboxSize(8)
	codec.SetOutboxSize(0)
	codec.SetWriteBatchSize(4, 5)
	codec.SetTCPNoDelay(false)
	codec.SetBlockDefinitions(blocks.NewRegistry(t.TempDir()))
	codec.SetSendTimeout("adaptive", time.Millisecond)
	codec.SetSendTimeout("fixed", 2*time.Millisecond)
	codec.SetServerName("Solar2")
	codec.SetPlayerDatabase(newMemoryPlayerDB())
	codec.SetI18n(nil)
	codec.SetBlockDBLookup(func(levelName string) plugin.BlockDB { return &coveragePluginBlockDB{} })
	codec.SetNameConverter(blocks.NewNameConverter())
	codec.SetLevelCallbacks(
		func(player plugin.Player, name string) bool { return name == "main" },
		func() string { return "main" },
		func(name string) bool { return true },
		func(name string) bool { return true },
		func() []string { return []string{"main"} },
		func() []string { return []string{"main"} },
	)
	codec.SetQueuePhysics(func(mgr *world.Manager, x, y, z int) {})
	codec.SetMaxPlayers(64)
	codec.SetSpamChecker(player.NewChecker(player.SpamConfig{Enabled: true}))
	codec.SetRankRegistry(ranks.NewRegistry())
	if codec.I18nGet("missing") != "missing" {
		t.Fatal("I18nGet fallback returned unexpected value")
	}

	alice := newCoverageSession(t, "alice")
	bob := newCoverageSession(t, "bob")
	bob.worlds = alice.worlds
	bobID, ok := bob.entities.Add("bob", entity.Position{X: 2 * coordScale, Y: 2 * coordScale, Z: 2 * coordScale})
	if !ok {
		t.Fatal("add bob entity failed")
	}
	bob.setIdentity("bob", bobID, true)
	alice.room = codec.room
	bob.room = codec.room
	codec.room.Join(alice)
	codec.room.Join(bob)
	codec.worlds = alice.worlds

	if len(codec.OnlinePlayers()) != 2 || codec.FindPlayer("alice") == nil {
		t.Fatal("online player lookup failed")
	}
	if codec.GetPlayerRank("alice") != ranks.PermGuest {
		t.Fatal("GetPlayerRank should default to guest")
	}
	codec.BroadcastPacket([]byte{0x01})
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)
	codec.BroadcastPacketToLevel(alice.worlds, []byte{0x02})
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)
	codec.BroadcastAddEntity(9, "mob", 1, 2, 3, 4, 5)
	codec.BroadcastRemoveEntity(9)
	codec.BroadcastEntityTeleport(9, 1, 2, 3, 4, 5)
	codec.BroadcastChangeModel(9, "humanoid")
	assertPacketCount(t, alice, 4)
	assertPacketCount(t, bob, 4)
	if len(codec.EncodeAddEntity(1, "x", 1, 2, 3, 4, 5)) == 0 ||
		len(codec.EncodeRemoveEntity(1)) == 0 ||
		len(codec.EncodeEntityTeleport(1, 1, 2, 3, 4, 5)) == 0 ||
		len(codec.EncodeChangeModel(1, "x")) == 0 {
		t.Fatal("encode wrappers returned empty packets")
	}
	if len(codec.PlayersOnLevel(alice.worlds)) != 2 {
		t.Fatal("PlayersOnLevel returned wrong count")
	}
	if codec.PlayerWorldManager(alice) != alice.worlds || codec.MainWorldManager() != alice.worlds {
		t.Fatal("world manager lookup returned wrong manager")
	}
	codec.BroadcastSetBlockToLevel(alice.worlds, 1, 2, 3, 4)
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)
	codec.BroadcastMessage("hello")
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)
	codec.sendTimeoutMode = "adaptive"
	codec.BroadcastEntityUpdates()
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)

	newWorld := world.NewManager()
	newWorld.SetCurrent(world.Level{Name: "next", Width: 4, Height: 4, Length: 4, Blocks: make([]byte, 64), Spawn: world.Spawn{X: 1, Y: 2, Z: 3}})
	alice.room = codec.room
	alice.markJoined(true)
	if err := codec.ChangeMap(alice, newWorld); err != nil {
		t.Fatalf("ChangeMap: %v", err)
	}
	assertPacketCountAtLeast(t, alice, 1)

	kickCodec := NewCodec("Solar", "MOTD", world.NewManager(), player.NewRegistry(), entity.NewManager(), command.NewRegistry())
	target := newCoverageSession(t, "target")
	target.room = kickCodec.room
	kickCodec.room.Join(target)
	kickCodec.KickAll("shutdown")
	assertPacketCount(t, target, 1)
}

func TestSessionRelativeCPEAndEncodingHandlers(t *testing.T) {
	s := newCoverageSession(t, "alice")

	s.reader = bufio.NewReader(bytes.NewReader([]byte{selfID, 1, 254, 3, 11, 12}))
	if err := s.handleRelativePosition(true, true); err != nil {
		t.Fatalf("handleRelativePosition pos+rot: %v", err)
	}
	snap, _ := s.entities.Get(s.currentEntityID())
	if snap.Pos.X != 33 || snap.Pos.Y != 62 || snap.Pos.Z != 99 || snap.Yaw != 11 || snap.Pitch != 12 {
		t.Fatalf("relative pos+rot snap = %+v", snap)
	}
	s.reader = bufio.NewReader(bytes.NewReader([]byte{selfID, 1, 1, 1}))
	if err := s.handleRelativePosition(true, false); err != nil {
		t.Fatalf("handleRelativePosition pos: %v", err)
	}
	s.reader = bufio.NewReader(bytes.NewReader([]byte{selfID, 21, 22}))
	if err := s.handleRelativePosition(false, true); err != nil {
		t.Fatalf("handleRelativePosition orientation: %v", err)
	}
	if err := s.handleRelativePosition(false, false); err != nil {
		t.Fatalf("handleRelativePosition noop: %v", err)
	}
	if s.resolveEntityID(selfID) != s.currentEntityID() || s.resolveEntityID(3) != 3 {
		t.Fatal("resolveEntityID returned unexpected values")
	}
	if decodeClassicDelta(255) != -1 || decodeClassicDelta(2) != 2 {
		t.Fatal("decodeClassicDelta returned unexpected values")
	}
	if specialBlockType(blocks.MBWhite) != blocks.SpecialMessage ||
		specialBlockType(blocks.PortalAir) != blocks.SpecialPortal ||
		specialBlockType(blocks.DoorLogAir) != blocks.SpecialDoor ||
		specialBlockType(1) != blocks.SpecialNone {
		t.Fatal("specialBlockType returned unexpected values")
	}

	s.handleSpamResult(player.SpamResult{Action: player.SpamActionWarn, Count: 2, Max: 1})
	s.handleSpamResult(player.SpamResult{Action: player.SpamActionMute})
	assertPacketCount(t, s, 2)
	kick := newCoverageSession(t, "spam")
	kick.handleSpamResult(player.SpamResult{Action: player.SpamActionKick})
	assertPacketCount(t, kick, 1)

	if len(encodeRelPosAndOrient(1, 1, 2, 3, 4, 5)) == 0 ||
		len(encodeRelPos(1, 1, 2, 3)) == 0 ||
		len(encodeOrientation(1, 4, 5)) == 0 ||
		!fitsRelDelta(127) ||
		fitsRelDelta(128) {
		t.Fatal("relative encoders returned unexpected values")
	}
	if len(encodeUndefineBlock(66)) == 0 {
		t.Fatal("encodeUndefineBlock returned empty packet")
	}
	def := blocks.Default(66)
	s.setCPESupport(map[string]uint32{cpeExtBlockDefinitions: 1, cpeExtBlockDefinitionsExt: 1, cpeExtExtTextures: 1})
	if len(s.encodeBlockDef(def)) == 0 {
		t.Fatal("encodeBlockDef returned empty packet")
	}

	exts := map[string]uint32{
		cpeExtPlayerClick:    1,
		cpeExtPluginMessages: 1,
		cpeExtNotifyAction:   1,
	}
	s.setCPESupport(exts)
	s.reader = bufio.NewReader(bytes.NewReader(make([]byte, 14)))
	if err := s.handleCPEPacket(opcodePlayerClick); err != nil {
		t.Fatalf("handle player click: %v", err)
	}
	s.reader = bufio.NewReader(bytes.NewReader(make([]byte, 65)))
	if err := s.handleCPEPacket(opcodePluginMessage); err != nil {
		t.Fatalf("handle plugin message: %v", err)
	}
	s.reader = bufio.NewReader(bytes.NewReader(make([]byte, 4)))
	if err := s.handleCPEPacket(opcodeNotifyAction); err != nil {
		t.Fatalf("handle notify action: %v", err)
	}
	s.reader = bufio.NewReader(bytes.NewReader(make([]byte, 8)))
	if err := s.handleCPEPacket(opcodeNotifyPositionAction); err != nil {
		t.Fatalf("handle notify position action: %v", err)
	}
	if err := s.handleCPEPacket(0xff); err != nil {
		t.Fatalf("handle unknown cpe packet: %v", err)
	}
	loggedOut := newCoverageSession(t, "loggedout")
	loggedOut.loggedIn = false
	if err := loggedOut.handleCPEPacket(opcodePlayerClick); err != nil {
		t.Fatalf("logged out cpe packet: %v", err)
	}
}

func newCoverageSession(t *testing.T, name string) *session {
	t.Helper()

	worlds := world.NewManager()
	entities := entity.NewManager()
	entityID, ok := entities.Add(name, entity.Position{X: 32, Y: 64, Z: 96})
	if !ok {
		t.Fatalf("add entity %s failed", name)
	}
	entities.SetLocation(entityID, entity.Position{X: 32, Y: 64, Z: 96}, 7, 8)

	players := player.NewRegistry()
	players.Add(name, entityID)
	var sendTimeout atomic.Int64
	sendTimeout.Store(int64(time.Second))

	s := &session{
		serverName:          "Solar",
		motd:                "MOTD",
		worlds:              worlds,
		players:             players,
		entities:            entities,
		commands:            command.NewRegistry(),
		logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		outbox:              make(chan []byte, 128),
		stop:                make(chan struct{}),
		sendTimeoutVal:      &sendTimeout,
		writeBatchSize:      16,
		shutdownBatchSize:   16,
		buildCommandContext: testBuildContext,
		allowBuild:          true,
	}
	s.setIdentity(name, entityID, true)
	s.markLoggedIn()
	return s
}

func assertPacketCount(t *testing.T, s *session, want int) {
	t.Helper()
	got := drainPackets(s)
	if got != want {
		t.Fatalf("drained packets = %d, want %d", got, want)
	}
}

func assertPacketCountAtLeast(t *testing.T, s *session, want int) {
	t.Helper()
	got := drainPackets(s)
	if got < want {
		t.Fatalf("drained packets = %d, want at least %d", got, want)
	}
}

func drainPackets(s *session) int {
	count := 0
	for {
		select {
		case <-s.outbox:
			count++
		default:
			return count
		}
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type coveragePluginBlockDB struct {
	entries []plugin.BlockEntry
	enabled bool
	cleared bool
	added   []plugin.BlockEntry
}

func (db *coveragePluginBlockDB) Add(e plugin.BlockEntry) {
	db.added = append(db.added, e)
}

func (db *coveragePluginBlockDB) ChangesAt(x, y, z int) []plugin.BlockEntry {
	return append([]plugin.BlockEntry(nil), db.entries...)
}

func (db *coveragePluginBlockDB) ChangesBy(playerID int32, since, until time.Time, maxResults int) []plugin.BlockEntry {
	return append([]plugin.BlockEntry(nil), db.entries...)
}

func (db *coveragePluginBlockDB) Count() int64 { return int64(len(db.entries)) }
func (db *coveragePluginBlockDB) Flush() error { return nil }
func (db *coveragePluginBlockDB) Clear() error {
	db.cleared = true
	return nil
}
func (db *coveragePluginBlockDB) Enabled() bool { return db.enabled }
func (db *coveragePluginBlockDB) SetEnabled(enabled bool) {
	db.enabled = enabled
}

type memoryPlayerDB struct {
	entries map[string]*playerdbplugin.PlayerEntry
}

func newMemoryPlayerDB() *memoryPlayerDB {
	return &memoryPlayerDB{entries: make(map[string]*playerdbplugin.PlayerEntry)}
}

func (db *memoryPlayerDB) Get(name string) *playerdbplugin.PlayerEntry {
	e := db.entries[strings.ToLower(name)]
	if e == nil {
		return nil
	}
	cp := *e
	return &cp
}

func (db *memoryPlayerDB) Save(entry *playerdbplugin.PlayerEntry) {
	cp := *entry
	db.entries[strings.ToLower(entry.Name)] = &cp
}

func (db *memoryPlayerDB) Delete(name string) bool {
	key := strings.ToLower(name)
	if _, ok := db.entries[key]; !ok {
		return false
	}
	delete(db.entries, key)
	return true
}

func (db *memoryPlayerDB) Search(prefix string) []*playerdbplugin.PlayerEntry {
	var out []*playerdbplugin.PlayerEntry
	for _, e := range db.entries {
		if strings.HasPrefix(strings.ToLower(e.Name), strings.ToLower(prefix)) {
			cp := *e
			out = append(out, &cp)
		}
	}
	return out
}

func (db *memoryPlayerDB) List() []*playerdbplugin.PlayerEntry {
	out := make([]*playerdbplugin.PlayerEntry, 0, len(db.entries))
	for _, e := range db.entries {
		cp := *e
		out = append(out, &cp)
	}
	return out
}

func (db *memoryPlayerDB) Count() int { return len(db.entries) }
func (db *memoryPlayerDB) Flush() error {
	return nil
}
