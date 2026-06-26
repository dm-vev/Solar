package blocks

import (
	"path/filepath"
	"testing"
)

func TestRegistryAddGetRemove(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewRegistry(dir)
	if got := r.Dir(); got != dir {
		t.Fatalf("Dir = %q, want %q", got, dir)
	}
	def := Default(100)
	def.Name = "test_block"
	r.Add(def)

	got, ok := r.Get(100)
	if !ok {
		t.Fatal("block def not found after add")
	}
	if got.Name != "test_block" {
		t.Fatalf("name = %q, want test_block", got.Name)
	}

	if !r.Has(100) {
		t.Fatal("Has should return true")
	}
	if r.Has(101) {
		t.Fatal("Has should return false for missing block")
	}

	if !r.Remove(100) {
		t.Fatal("Remove should return true")
	}
	if r.Has(100) {
		t.Fatal("Has should return false after remove")
	}
	if r.Remove(100) {
		t.Fatal("Remove should return false for missing block")
	}
}

func TestRegistryFreeID(t *testing.T) {
	t.Parallel()

	r := NewRegistry(t.TempDir())
	if id := r.FreeID(); id != FirstCustomBlock {
		t.Fatalf("first free ID = %d, want %d", id, FirstCustomBlock)
	}

	r.Add(Default(66))
	r.Add(Default(67))
	r.Add(Default(69))

	if id := r.FreeID(); id != 68 {
		t.Fatalf("free ID = %d, want 68", id)
	}
}

func TestRegistryAll(t *testing.T) {
	t.Parallel()

	r := NewRegistry(t.TempDir())
	r.Add(Default(70))
	r.Add(Default(80))
	r.Add(Default(90))

	all := r.All()
	if len(all) != 3 {
		t.Fatalf("All() returned %d, want 3", len(all))
	}
	if all[0].ID != 70 || all[1].ID != 80 || all[2].ID != 90 {
		t.Fatalf("All() not sorted: %d %d %d", all[0].ID, all[1].ID, all[2].ID)
	}
}

func TestRegistryCount(t *testing.T) {
	t.Parallel()

	r := NewRegistry(t.TempDir())
	if r.Count() != 0 {
		t.Fatalf("Count = %d, want 0", r.Count())
	}
	r.Add(Default(66))
	r.Add(Default(67))
	if r.Count() != 2 {
		t.Fatalf("Count = %d, want 2", r.Count())
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewRegistry(dir)

	r.Add(Default(66))
	r.Add(Default(100))

	def := Default(200)
	def.Name = "special"
	def.Speed = 2.5
	def.TopTex = 10
	def.FogDensity = 5
	r.Add(def)

	if err := r.SaveGlobal(); err != nil {
		t.Fatalf("SaveGlobal: %v", err)
	}

	r2 := NewRegistry(dir)
	if err := r2.LoadGlobal(); err != nil {
		t.Fatalf("LoadGlobal: %v", err)
	}

	if r2.Count() != 3 {
		t.Fatalf("Count = %d, want 3", r2.Count())
	}

	got, ok := r2.Get(200)
	if !ok {
		t.Fatal("block 200 not found after reload")
	}
	if got.Name != "special" || got.Speed != 2.5 || got.TopTex != 10 || got.FogDensity != 5 {
		t.Fatalf("block 200 = %+v", got)
	}
}

func TestLoadGlobalMissingFile(t *testing.T) {
	t.Parallel()

	r := NewRegistry(filepath.Join(t.TempDir(), "nonexistent"))
	if err := r.LoadGlobal(); err != nil {
		t.Fatalf("LoadGlobal on missing dir should not error: %v", err)
	}
	if r.Count() != 0 {
		t.Fatalf("Count = %d, want 0", r.Count())
	}
}

func TestLevelPersistence(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := NewRegistry(dir)

	r.Add(Default(70))

	if err := r.SaveLevel("testworld"); err != nil {
		t.Fatalf("SaveLevel: %v", err)
	}

	r2 := NewRegistry(dir)
	if err := r2.LoadLevel("testworld"); err != nil {
		t.Fatalf("LoadLevel: %v", err)
	}

	if r2.Count() != 1 {
		t.Fatalf("Count = %d, want 1", r2.Count())
	}
	if _, ok := r2.Get(70); !ok {
		t.Fatal("block 70 not found after level reload")
	}
}

func TestLevelFileCannotEscapeDirectory(t *testing.T) {
	t.Parallel()

	if got := LevelFile("../evil"); got != "lvl__.json" {
		t.Fatalf("LevelFile traversal = %q, want lvl__.json", got)
	}
	if got := LevelFile("main"); got != "lvl_main.json" {
		t.Fatalf("LevelFile main = %q, want lvl_main.json", got)
	}
}

func TestRawSpeed(t *testing.T) {
	t.Parallel()

	def := Default(66)
	def.Speed = 1.0
	if got := def.RawSpeed(); got != 128 {
		t.Fatalf("RawSpeed for 1.0 = %d, want 128", got)
	}

	def.Speed = 2.0
	if got := def.RawSpeed(); got != 192 {
		t.Fatalf("RawSpeed for 2.0 = %d, want 192", got)
	}

	def.Speed = 0.5
	if got := def.RawSpeed(); got != 64 {
		t.Fatalf("RawSpeed for 0.5 = %d, want 64", got)
	}
}

func TestBlockDefinitionWireHelpers(t *testing.T) {
	t.Parallel()

	def := Default(66)
	if got := def.BrightnessByte(); got != 0 {
		t.Fatalf("default BrightnessByte = %d, want 0", got)
	}
	def.Brightness = 9
	if got := def.BrightnessByte(); got != 0x89 {
		t.Fatalf("BrightnessByte = %#x, want 0x89", got)
	}
	def.FullBright = true
	if got := def.BrightnessByte(); got != 0 {
		t.Fatalf("fullbright BrightnessByte = %d, want 0", got)
	}

	if def.IsSprite() {
		t.Fatal("default block reported sprite")
	}
	def.Shape = 0
	if !def.IsSprite() {
		t.Fatal("shape 0 block did not report sprite")
	}
}

func TestSetAllTex(t *testing.T) {
	t.Parallel()

	def := Default(66)
	def.SetAllTex(42)
	if def.TopTex != 42 || def.BottomTex != 42 || def.LeftTex != 42 || def.RightTex != 42 || def.FrontTex != 42 || def.BackTex != 42 {
		t.Fatalf("SetAllTex failed: %+v", def)
	}
}

func TestSetSideTex(t *testing.T) {
	t.Parallel()

	def := Default(66)
	def.TopTex = 1
	def.BottomTex = 2
	def.SetSideTex(99)
	if def.TopTex != 1 || def.BottomTex != 2 {
		t.Fatal("SetSideTex should not touch top/bottom")
	}
	if def.LeftTex != 99 || def.RightTex != 99 || def.FrontTex != 99 || def.BackTex != 99 {
		t.Fatalf("SetSideTex failed: %+v", def)
	}
}

func TestCollideTypeName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		v    byte
		want string
	}{
		{CollideWalkThrough, "walkthrough"},
		{CollideSolid, "solid"},
		{CollideLiquidWater, "water"},
		{99, "unknown"},
	}
	for _, tc := range cases {
		if got := CollideTypeName(tc.v); got != tc.want {
			t.Fatalf("CollideTypeName(%d) = %q, want %q", tc.v, got, tc.want)
		}
	}
}

func TestDrawAndSoundTypeNames(t *testing.T) {
	t.Parallel()

	drawCases := []struct {
		value byte
		want  string
	}{
		{DrawOpaque, "opaque"},
		{DrawTransparent, "transparent"},
		{DrawTransparentThick, "transparent_thick"},
		{DrawTranslucent, "translucent"},
		{DrawGas, "gas"},
		{DrawSprite, "sprite"},
		{99, "unknown"},
	}
	for _, tc := range drawCases {
		if got := DrawTypeName(tc.value); got != tc.want {
			t.Fatalf("DrawTypeName(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}

	soundCases := []struct {
		value byte
		want  string
	}{
		{SoundNone, "none"},
		{SoundWood, "wood"},
		{SoundGravel, "gravel"},
		{SoundGrass, "grass"},
		{SoundStone, "stone"},
		{SoundMetal, "metal"},
		{SoundGlass, "glass"},
		{SoundCloth, "cloth"},
		{SoundSand, "sand"},
		{SoundSnow, "snow"},
		{99, "unknown"},
	}
	for _, tc := range soundCases {
		if got := SoundTypeName(tc.value); got != tc.want {
			t.Fatalf("SoundTypeName(%d) = %q, want %q", tc.value, got, tc.want)
		}
	}
}
