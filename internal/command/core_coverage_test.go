package command

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
)

func TestBlockDefCommands(t *testing.T) {
	blockDefs := newStubBlockDefs()
	ctx := Context{BlockDefs: blockDefs, Tr: testTr}

	got, handled := globalBlockCommand(Context{}, []string{"add", "66", "glass2"})
	if !handled || got != "block definitions are unavailable" {
		t.Fatalf("globalBlockCommand without service = %q handled=%v", got, handled)
	}
	got, _ = globalBlockCommand(ctx, nil)
	if got != "usage: /gb <add|edit|remove|info|list>" {
		t.Fatalf("usage = %q", got)
	}
	got, _ = globalBlockCommand(ctx, []string{"add"})
	if !strings.Contains(got, "next free ID: 66") {
		t.Fatalf("add usage = %q", got)
	}
	got, _ = globalBlockCommand(ctx, []string{"add", "65"})
	if !strings.Contains(got, "custom block IDs must be") {
		t.Fatalf("reserved add = %q", got)
	}
	got, _ = globalBlockCommand(ctx, []string{"add", "66", "glass2"})
	if got != "added block 66 (glass2)" {
		t.Fatalf("add = %q", got)
	}
	if def, ok := blockDefs.GetBlockDef(66); !ok || def.Name != "glass2" {
		t.Fatalf("stored block = %+v ok=%v", def, ok)
	}

	got, _ = levelBlockCommand(ctx, []string{"edit", "66", "speed", "2.5"})
	if got != "updated block 66 speed = 2.5" {
		t.Fatalf("edit = %q", got)
	}
	got, _ = levelBlockCommand(ctx, []string{"info", "66"})
	if !strings.Contains(got, "block 66: name=glass2") {
		t.Fatalf("info = %q", got)
	}
	got, _ = levelBlockCommand(ctx, []string{"list"})
	if got != "blocks (1): 66:glass2" {
		t.Fatalf("list = %q", got)
	}
	got, _ = levelBlockCommand(ctx, []string{"remove", "66"})
	if got != "removed block 66" {
		t.Fatalf("remove = %q", got)
	}
	got, _ = levelBlockCommand(ctx, []string{"info", "66"})
	if got != "block 66 is not defined" {
		t.Fatalf("missing info = %q", got)
	}
}

func TestBlockDefPropertyParsing(t *testing.T) {
	def := blocks.Default(66)
	cases := []struct {
		prop string
		val  string
	}{
		{"name", "lava_glass"},
		{"collide", "1"},
		{"speed", "1.5"},
		{"toptex", "2"},
		{"bottomtex", "3"},
		{"sidetex", "4"},
		{"alltex", "5"},
		{"lefttex", "6"},
		{"righttex", "7"},
		{"fronttex", "8"},
		{"backtex", "9"},
		{"blockslight", "off"},
		{"sound", "6"},
		{"fullbright", "yes"},
		{"shape", "12"},
		{"draw", "3"},
		{"fallback", "1"},
		{"fogdensity", "10"},
		{"fogcolor", "1,2,3"},
		{"min", "1 2 3"},
		{"max", "14 15 16"},
	}
	for _, tc := range cases {
		if err := editBlockProp(&def, tc.prop, tc.val); err != nil {
			t.Fatalf("editBlockProp(%s=%s): %v", tc.prop, tc.val, err)
		}
	}

	if def.Name != "lava_glass" || def.Speed != 1.5 || def.FogR != 1 || def.FogG != 2 || def.FogB != 3 {
		t.Fatalf("edited def = %+v", def)
	}
	if def.MinX != 1 || def.MinY != 2 || def.MinZ != 3 || def.MaxX != 14 || def.MaxY != 15 || def.MaxZ != 16 {
		t.Fatalf("edited bbox = %+v", def)
	}
	if _, err := parseBlockID("256"); err == nil {
		t.Fatal("parseBlockID accepted 256")
	}
	if _, err := parseByteProp("-1", 16); err == nil {
		t.Fatal("parseByteProp accepted -1")
	}
	if err := editBlockProp(&def, "fogcolor", "1,2"); err == nil {
		t.Fatal("editFogColor accepted two components")
	}
	if err := editBlockProp(&def, "min", "1,2"); err == nil {
		t.Fatal("editBBox accepted two components")
	}
	if err := editBlockProp(&def, "unknown", "1"); err == nil {
		t.Fatal("editBlockProp accepted unknown property")
	}
}

func TestMapCommands(t *testing.T) {
	env := &stubLevelEnv{}
	ctx := Context{LevelEnv: env, Tr: testTr}

	got, _ := mapCommand(Context{Tr: testTr}, nil)
	if got != "command.level.unavailable" {
		t.Fatalf("map without env = %q", got)
	}
	got, _ = mapCommand(ctx, nil)
	if !strings.Contains(got, "command.map.info.weather") || !strings.Contains(got, "&7motd: default") {
		t.Fatalf("map info = %q", got)
	}
	got, _ = mapCommand(ctx, []string{"weather", "2"})
	if got != "command.map.weather.set" || env.weather != 2 {
		t.Fatalf("weather result = %q weather=%d", got, env.weather)
	}
	got, _ = mapCommand(ctx, []string{"sky", "1", "2", "3"})
	if got != "command.map.color.set" {
		t.Fatalf("color result = %q", got)
	}
	if r, g, b, set := env.GetEnvColor(0); !set || r != 1 || g != 2 || b != 3 {
		t.Fatalf("sky color = %d %d %d set=%v", r, g, b, set)
	}
	got, _ = mapCommand(ctx, []string{"motd", "hello", "world"})
	if got != "command.map.motd.set" || env.motd != "hello world" {
		t.Fatalf("motd result = %q motd=%q", got, env.motd)
	}
	if _, err := parseColorComp("300"); err == nil {
		t.Fatal("parseColorComp accepted 300")
	}
	if colorSlot("cloud") != 1 || colorSlot("fog") != 2 || colorSlot("ambient") != 3 || colorSlot("diffuse") != 4 {
		t.Fatal("colorSlot returned unexpected slot")
	}
}

func TestLevelCommands(t *testing.T) {
	levels := &coverageLevels{main: "main", levels: []string{"main", "test"}, files: []string{"other", "main"}}
	ctx := Context{Levels: levels, Tr: testTr}

	got, _ := mainCommand(ctx, nil)
	if got != "command.goto.done" || levels.gotoName != "main" {
		t.Fatalf("mainCommand = %q goto=%q", got, levels.gotoName)
	}
	got, _ = loadCommand(ctx, []string{"test"})
	if got != "command.load.done" || levels.loadedName != "test" {
		t.Fatalf("loadCommand = %q loaded=%q", got, levels.loadedName)
	}
	got, _ = unloadCommand(ctx, []string{"main"})
	if got != "command.unload.main" {
		t.Fatalf("unload main = %q", got)
	}
	got, _ = unloadCommand(ctx, []string{"test"})
	if got != "command.unload.done" || levels.unloadedName != "test" {
		t.Fatalf("unloadCommand = %q unloaded=%q", got, levels.unloadedName)
	}
	got, _ = reloadCommand(ctx, nil)
	if got != "command.reload.done" || !levels.reloaded {
		t.Fatalf("reloadCommand = %q reloaded=%v", got, levels.reloaded)
	}
	got, _ = levelsCommand(ctx, nil)
	if got != "command.levels.list" {
		t.Fatalf("levelsCommand = %q", got)
	}
	got, _ = physicsCommand(ctx, []string{"advanced"})
	if got != "command.blocks.set" || levels.physics != 2 {
		t.Fatalf("physics set = %q mode=%d", got, levels.physics)
	}
	got, _ = physicsCommand(ctx, nil)
	if got != "command.blocks.current" {
		t.Fatalf("physics current = %q", got)
	}
}

func TestWhitelistCommands(t *testing.T) {
	ctx := Context{
		Moderation: &stubModeration{enabled: false},
		Players:    stubDirectory{whitelisted: []string{"bob", "alice"}},
		Tr:         testTr,
	}

	got, _ := whitelistOn(ctx)
	if got != "command.whitelist.enabled" {
		t.Fatalf("whitelistOn = %q", got)
	}
	got, _ = whitelistOff(ctx)
	if got != "command.whitelist.disabled" {
		t.Fatalf("whitelistOff = %q", got)
	}
	got, _ = whitelistList(ctx)
	if got != "command.whitelist.list.items" {
		t.Fatalf("whitelistList = %q", got)
	}
}

func TestBlockDBCommands(t *testing.T) {
	entry := BlockDBEntry{
		PlayerName: "alice",
		Time:       time.Now().Add(-time.Minute),
		X:          1,
		Y:          2,
		Z:          3,
		OldBlock:   1,
		NewBlock:   2,
	}
	db := &stubBlockDB{entriesAt: []BlockDBEntry{entry}, entriesBy: []BlockDBEntry{entry}, count: 1, enabled: true}
	ctx := Context{Username: "alice", Position: Position{X: 1, Y: 2, Z: 3}, BlockDB: db, Tr: testTr}

	got, _ := aboutCommand(ctx, nil)
	if !strings.Contains(got, "alice") || !strings.Contains(got, "changed") {
		t.Fatalf("aboutCommand = %q", got)
	}
	got, _ = undoBlockDBCommand(ctx, []string{"1h"})
	if got != "command.undo.done" || db.reverted != 1 {
		t.Fatalf("undoBlockDBCommand = %q reverted=%d", got, db.reverted)
	}
	got, _ = undoBlockDBCommand(ctx, []string{"bad"})
	if got != "command.undo.invalid_duration" {
		t.Fatalf("undo invalid = %q", got)
	}
	got, _ = blockDBCommand(ctx, []string{"disable"})
	if got != "command.blockdb.disabled" || db.enabled {
		t.Fatalf("blockdb disable = %q enabled=%v", got, db.enabled)
	}
	got, _ = blockDBCommand(ctx, []string{"enable"})
	if got != "command.blockdb.enabled" || !db.enabled {
		t.Fatalf("blockdb enable = %q enabled=%v", got, db.enabled)
	}
	got, _ = blockDBCommand(ctx, []string{"stats"})
	if got != "command.blockdb.stats" {
		t.Fatalf("blockdb stats = %q", got)
	}
	got, _ = blockDBCommand(ctx, []string{"clear"})
	if got != "command.blockdb.cleared" || !db.cleared {
		t.Fatalf("blockdb clear = %q cleared=%v", got, db.cleared)
	}

	db.clearErr = errors.New("boom")
	got, _ = blockDBCommand(ctx, []string{"clear"})
	if got != "command.blockdb.clear_failed" {
		t.Fatalf("blockdb clear error = %q", got)
	}
}

func TestInfoAndMiscCommands(t *testing.T) {
	ctx := Context{
		Username:   "alice",
		PlayerDB:   stubPlayerDB{},
		Levels:     &coverageLevels{levels: []string{"main"}},
		BlockDB:    &stubBlockDB{count: 3},
		LevelEnv:   &stubLevelEnv{weather: 1},
		ServerInfo: stubServerInfo{},
		Teleport:   stubTeleport{},
		World:      stubWorld{},
		Ranks:      coverageRanks{},
		Tr:         testTr,
	}

	if got, _ := blocksCommand(ctx, nil); got != "command.blocks.stats" {
		t.Fatalf("blocksCommand = %q", got)
	}
	if got, _ := mapinfoCommand(ctx, nil); !strings.Contains(got, "command.mapinfo.levels") {
		t.Fatalf("mapinfoCommand = %q", got)
	}
	if got, _ := serverinfoCommand(ctx, nil); !strings.Contains(got, "command.serverinfo.name") {
		t.Fatalf("serverinfoCommand = %q", got)
	}
	if got, _ := summonCommand(ctx, []string{"bob"}); got != "command.summon.done" {
		t.Fatalf("summonCommand = %q", got)
	}
	if got := cuboidVolume(blocks.Vec3{0, 0, 0}, blocks.Vec3{2, 3, 4}); got != 60 {
		t.Fatalf("cuboidVolume = %d, want 60", got)
	}
	if got, _ := viewRanksCommand(ctx, nil); !strings.Contains(got, "Guest(0)") || !strings.Contains(got, "Op(100)") {
		t.Fatalf("viewRanksCommand = %q", got)
	}
}

func TestRegistryAdminAndUnregister(t *testing.T) {
	registry := NewRegistry()
	registry.SetAdminCommands([]string{"custom"})
	registry.Register("custom", func(ctx Context, args []string) (string, bool) {
		return "ok", true
	})

	got, _ := registry.Execute(Context{Tr: testTr}, "/custom")
	if got != "command.shared.permission_denied" {
		t.Fatalf("custom without admin = %q", got)
	}
	got, _ = registry.Execute(Context{Authority: testAuthority(true), Tr: testTr}, "/custom")
	if got != "ok" {
		t.Fatalf("custom with admin = %q", got)
	}
	if !registry.Unregister("custom") {
		t.Fatal("Unregister returned false for registered command")
	}
	if registry.Unregister("custom") || registry.Unregister("") {
		t.Fatal("Unregister returned true for missing command")
	}
}

type stubBlockDefs struct {
	defs map[byte]blocks.BlockDefinition
}

func newStubBlockDefs() *stubBlockDefs {
	return &stubBlockDefs{defs: make(map[byte]blocks.BlockDefinition)}
}

func (s *stubBlockDefs) AddBlockDef(def blocks.BlockDefinition) bool {
	s.defs[def.ID] = def
	return true
}

func (s *stubBlockDefs) RemoveBlockDef(id byte) bool {
	if _, ok := s.defs[id]; !ok {
		return false
	}
	delete(s.defs, id)
	return true
}

func (s *stubBlockDefs) GetBlockDef(id byte) (blocks.BlockDefinition, bool) {
	def, ok := s.defs[id]
	return def, ok
}

func (s *stubBlockDefs) ListBlockDefs() []blocks.BlockDefinition {
	out := make([]blocks.BlockDefinition, 0, len(s.defs))
	for _, def := range s.defs {
		out = append(out, def)
	}
	slices.SortFunc(out, func(a, b blocks.BlockDefinition) int {
		return int(a.ID) - int(b.ID)
	})
	return out
}

func (s *stubBlockDefs) FreeBlockID() byte {
	for id := byte(blocks.FirstCustomBlock); id <= blocks.MaxBlockID; id++ {
		if _, ok := s.defs[id]; !ok {
			return id
		}
	}
	return 0
}

type stubLevelEnv struct {
	weather int
	colors  [5][3]byte
	set     [5]bool
	motd    string
}

func (s *stubLevelEnv) GetEnvColor(slot int) (byte, byte, byte, bool) {
	return s.colors[slot][0], s.colors[slot][1], s.colors[slot][2], s.set[slot]
}

func (s *stubLevelEnv) SetEnvColor(slot int, r, g, b byte) {
	s.colors[slot] = [3]byte{r, g, b}
	s.set[slot] = true
}

func (s *stubLevelEnv) Weather() int { return s.weather }
func (s *stubLevelEnv) SetWeather(weather int) {
	s.weather = weather
}
func (s *stubLevelEnv) MOTD() string { return s.motd }
func (s *stubLevelEnv) SetMOTD(motd string) {
	s.motd = motd
}

type coverageLevels struct {
	main         string
	levels       []string
	files        []string
	physics      int
	gotoName     string
	loadedName   string
	unloadedName string
	reloaded     bool
}

func (s *coverageLevels) Goto(levelName string) bool {
	s.gotoName = levelName
	return true
}

func (s *coverageLevels) MainLevel() string { return s.main }
func (s *coverageLevels) LoadLevel(name string) bool {
	s.loadedName = name
	return true
}
func (s *coverageLevels) UnloadLevel(name string) bool {
	s.unloadedName = name
	return true
}
func (s *coverageLevels) ReloadLevel() bool {
	s.reloaded = true
	return true
}
func (s *coverageLevels) ListLevels() []string {
	return append([]string(nil), s.levels...)
}
func (s *coverageLevels) ListLevelFiles() []string {
	return append([]string(nil), s.files...)
}
func (s *coverageLevels) PhysicsMode() int { return s.physics }
func (s *coverageLevels) SetPhysicsMode(mode int) {
	s.physics = mode
}

type stubBlockDB struct {
	entriesAt []BlockDBEntry
	entriesBy []BlockDBEntry
	count     int64
	enabled   bool
	cleared   bool
	clearErr  error
	reverted  int
}

func (s *stubBlockDB) ChangesAt(x, y, z int) []BlockDBEntry {
	return append([]BlockDBEntry(nil), s.entriesAt...)
}

func (s *stubBlockDB) ChangesBy(playerName string, since time.Time, maxResults int) []BlockDBEntry {
	return append([]BlockDBEntry(nil), s.entriesBy...)
}

func (s *stubBlockDB) Count() int64 { return s.count }
func (s *stubBlockDB) Enabled() bool {
	return s.enabled
}
func (s *stubBlockDB) SetEnabled(enabled bool) {
	s.enabled = enabled
}
func (s *stubBlockDB) Clear() error {
	if s.clearErr != nil {
		return s.clearErr
	}
	s.cleared = true
	return nil
}
func (s *stubBlockDB) RevertBlock(x, y, z int, block byte) bool {
	s.reverted++
	return true
}

type coverageRanks struct{}

func (coverageRanks) Get(name string) *RankInfo {
	switch strings.ToLower(name) {
	case "guest":
		return &RankInfo{Name: "Guest", Permission: 0, Color: "&7"}
	case "op":
		return &RankInfo{Name: "Op", Permission: 100, Color: "&c"}
	default:
		return nil
	}
}

func (coverageRanks) GetByPerm(perm int) *RankInfo {
	if perm == 100 {
		return &RankInfo{Name: "Op", Permission: 100, Color: "&c"}
	}
	return &RankInfo{Name: "Guest", Permission: 0, Color: "&7"}
}

func (coverageRanks) All() []RankInfo {
	return []RankInfo{
		{Name: "Guest", Permission: 0, Color: "&7"},
		{Name: "Op", Permission: 100, Color: "&c"},
	}
}

func (coverageRanks) GetPlayerRank(name string) int {
	if strings.EqualFold(name, "op") {
		return 100
	}
	return 0
}

func (coverageRanks) SetPlayerRank(name string, perm int) bool {
	return name != "" && perm >= 0
}
