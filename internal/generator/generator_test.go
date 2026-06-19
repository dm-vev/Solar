package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"math/rand"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	RegisterDefaults()
	os.Exit(m.Run())
}

func TestNewLevel(t *testing.T) {
	lvl := NewLevel("test", 4, 3, 5)
	if lvl.Volume() != 60 {
		t.Fatalf("volume = %d, want 60", lvl.Volume())
	}
}

func TestValidateDimensions(t *testing.T) {
	if err := ValidateDimensions(1, 1, 1); err != nil {
		t.Fatalf("valid dimensions failed: %v", err)
	}
	if err := ValidateDimensions(0, 1, 1); err == nil {
		t.Fatal("expected error for zero width")
	}
	if err := ValidateDimensions(10000, 10000, 10000); err == nil {
		t.Fatal("expected error for oversized dimensions")
	}
}

func TestSimpleFlatGenerator(t *testing.T) {
	gen, ok := Find("Flat")
	if !ok {
		t.Fatal("Flat generator not found")
	}
	lvl, err := Generate(gen, "flat", 16, 16, 16, Args{Seed: 8, Biome: Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if lvl.Spawn.Y != 10 {
		t.Fatalf("spawn y = %d, want 10", lvl.Spawn.Y)
	}
}

func TestSimpleEmptyGenerator(t *testing.T) {
	gen, ok := Find("Empty")
	if !ok {
		t.Fatal("Empty generator not found")
	}
	lvl, err := Generate(gen, "empty", 8, 8, 8, Args{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for x := 0; x < 8; x++ {
		for z := 0; z < 8; z++ {
			if lvl.Blocks[PackIndex(*lvl, x, 0, z)] != Bedrock {
				t.Fatalf("bottom layer is not bedrock at %d,%d", x, z)
			}
		}
	}
}

func TestClassicGenerator(t *testing.T) {
	gen, ok := Find("Classic")
	if !ok {
		t.Fatal("Classic generator not found")
	}
	lvl, err := Generate(gen, "classic", 32, 32, 32, Args{Seed: 12345, Biome: Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Verify the world has some non-air blocks and a sane spawn.
	nonAir := 0
	for _, b := range lvl.Blocks {
		if b != Air {
			nonAir++
		}
	}
	if nonAir == 0 {
		t.Fatal("classic generator produced only air")
	}
	if lvl.Spawn.Y < 1 || lvl.Spawn.Y >= lvl.Height {
		t.Fatalf("spawn y = %d, want within level", lvl.Spawn.Y)
	}
}

func TestClassicGeneratorMatchesMCGalaxy(t *testing.T) {
	t.Parallel()

	gen, ok := Find("Classic")
	if !ok {
		t.Fatal("Classic generator not found")
	}

	cases := []struct {
		name          string
		seed          int
		width         int
		height        int
		length        int
		wantBlocksSHA string
	}{
		{name: "classic-default-seed", seed: 12345, width: 128, height: 64, length: 128, wantBlocksSHA: "270b5bd06a0063985c021997feede35cecc3eb10a04b410f8e7cbb51f23baafa"},
		{name: "classic-zero-seed", seed: 0, width: 128, height: 64, length: 128, wantBlocksSHA: "6aa26fa9afbbf338f0affc6960eea606a0de0cc09f8a94f40664713cda9c146b"},
		{name: "classic-64-cube", seed: 12345, width: 64, height: 64, length: 64, wantBlocksSHA: "30d1919fc29012fafd2a66455de55676b067cf5dfd52faaab51480270911fbec"},
		{name: "classic-small", seed: 42, width: 32, height: 32, length: 32, wantBlocksSHA: "b2e0fb92a45f76c6c1cd634cf9ff5e54ad07c4c7e5b5d3533e4c863b5b9b7296"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			lvl, err := Generate(gen, "classic", tc.width, tc.height, tc.length, Args{Seed: tc.seed, Biome: Forest})
			if err != nil {
				t.Fatalf("generate: %v", err)
			}
			sum := sha256.Sum256(lvl.Blocks)
			got := hex.EncodeToString(sum[:])
			if got != tc.wantBlocksSHA {
				t.Fatalf("blocks sha256 = %s, want %s", got, tc.wantBlocksSHA)
			}
		})
	}
}

func TestFCraftGenerator(t *testing.T) {
	gen, ok := Find("Hills")
	if !ok {
		t.Fatal("Hills generator not found")
	}
	lvl, err := Generate(gen, "hills", 32, 32, 32, Args{Seed: 42, Biome: Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	nonAir := 0
	for _, b := range lvl.Blocks {
		if b != Air {
			nonAir++
		}
	}
	if nonAir == 0 {
		t.Fatal("fcraft generator produced only air")
	}
}

func TestHeightmapGenerator(t *testing.T) {
	// Create a simple 4x4 grayscale image.
	img := image.NewGray(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetGray(x, y, color.Gray{Y: byte((y*4 + x) * 4)})
		}
	}

	lvl := NewLevel("h", 4, 16, 4)
	if err := applyHeightmap(lvl, img, Forest); err != nil {
		t.Fatalf("applyHeightmap: %v", err)
	}
	if lvl.Spawn.Y < 1 || lvl.Spawn.Y >= lvl.Height {
		t.Fatalf("spawn y = %d, want within level", lvl.Spawn.Y)
	}
}

func TestParseArgs(t *testing.T) {
	args, err := ParseArgs("123 Forest")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if args.Seed != 123 {
		t.Fatalf("seed = %d, want 123", args.Seed)
	}
	if args.Biome.Name != "Forest" {
		t.Fatalf("biome = %s, want Forest", args.Biome.Name)
	}
	if args.RandomSeed {
		t.Fatal("expected explicit seed, got random seed")
	}
}

func TestParseArgsRandomSeed(t *testing.T) {
	args, err := ParseArgs("Desert")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !args.RandomSeed {
		t.Fatal("expected random seed when no seed specified")
	}
	if args.Biome.Name != "Desert" {
		t.Fatalf("biome = %s, want Desert", args.Biome.Name)
	}
}

func TestParseArgsUnknownBiome(t *testing.T) {
	_, err := ParseArgs("Void")
	if err == nil {
		t.Fatal("expected error for unknown biome")
	}
}

func TestRegistryContainsGenerators(t *testing.T) {
	for _, name := range []string{"Flat", "Empty", "Classic", "Hills", "Heightmap"} {
		if _, ok := Find(name); !ok {
			t.Fatalf("%s generator not registered", name)
		}
	}
}

func TestRegistryRegisterModule(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.RegisterModule(Module{
		Name: "test",
		Generators: func() []Generator {
			return []Generator{
				{Name: "One", Type: GenTypeSimple, Func: genEmpty},
				{Name: "Two", Type: GenTypeAdvanced, Func: genEmpty},
			}
		},
	})

	if _, ok := registry.Find("one"); !ok {
		t.Fatal("module generator One was not registered")
	}
	if _, ok := registry.Find("TWO"); !ok {
		t.Fatal("module generator Two was not registered")
	}
	if got := len(registry.All()); got != 2 {
		t.Fatalf("registered generators = %d, want 2", got)
	}
}

func TestBuiltinModulesRegisterExpectedGenerators(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	for _, module := range BuiltinModules() {
		registry.RegisterModule(module)
	}

	for _, name := range []string{"Flat", "Empty", "Classic", "Hills", "Heightmap"} {
		if _, ok := registry.Find(name); !ok {
			t.Fatalf("%s generator not registered by built-in modules", name)
		}
	}
}

func TestTreeGenerator(t *testing.T) {
	tree := NewClassicTree()
	rng := newRandom(0)
	tree.SetData(rng, 6)
	blocks := make(map[[3]int]byte)
	tree.Generate(0, 0, 0, func(x, y, z int, block byte) {
		blocks[[3]int{x, y, z}] = block
	})
	if len(blocks) == 0 {
		t.Fatal("tree generated no blocks")
	}
	if blocks[[3]int{0, 0, 0}] != Log {
		t.Fatal("tree trunk missing at origin")
	}
}

func newRandom(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }

func TestClassicTerrainShape(t *testing.T) {
	gen, ok := Find("Classic")
	if !ok {
		t.Fatal("Classic generator not found")
	}
	lvl, err := Generate(gen, "classic", 128, 64, 128, Args{Seed: 12345, Biome: Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Bottom layer should be lava.
	for x := 0; x < lvl.Width; x++ {
		for z := 0; z < lvl.Length; z++ {
			if lvl.Blocks[PackIndex(*lvl, x, 0, z)] != Lava {
				t.Fatalf("bottom layer at %d,%d is not lava: %d", x, z, lvl.Blocks[PackIndex(*lvl, x, 0, z)])
			}
		}
	}

	// There should be grass exposed to air, stone below, and some air pockets.
	var grass, stone, air, water, lava int
	for y := 0; y < lvl.Height; y++ {
		for z := 0; z < lvl.Length; z++ {
			for x := 0; x < lvl.Width; x++ {
				switch lvl.Blocks[PackIndex(*lvl, x, y, z)] {
				case Grass:
					grass++
				case Stone:
					stone++
				case Air:
					air++
				case StillWater:
					water++
				case Lava, StillLava:
					lava++
				}
			}
		}
	}
	if grass == 0 {
		t.Fatal("no grass generated")
	}
	if stone == 0 {
		t.Fatal("no stone generated")
	}
	if air == 0 {
		t.Fatal("no air generated")
	}
	if lava == 0 {
		t.Fatal("no lava generated")
	}
	// Stone should be the bulk of the volume.
	if stone < grass*4 {
		t.Fatalf("too little stone: grass=%d stone=%d", grass, stone)
	}

	// Print a cross-section so a human can inspect the terrain.
	midX := lvl.Width / 2
	var sb strings.Builder
	for y := lvl.Height - 1; y >= 0; y-- {
		for z := 0; z < lvl.Length; z++ {
			b := lvl.Blocks[PackIndex(*lvl, midX, y, z)]
			switch b {
			case Air:
				sb.WriteByte(' ')
			case Grass:
				sb.WriteByte('g')
			case Dirt:
				sb.WriteByte('d')
			case Stone:
				sb.WriteByte('s')
			case Water, StillWater:
				sb.WriteByte('~')
			case Lava, StillLava:
				sb.WriteByte('L')
			case Log:
				sb.WriteByte('T')
			case Leaves:
				sb.WriteByte('l')
			case GoldOre, IronOre, CoalOre:
				sb.WriteByte('o')
			case Dandelion, Rose:
				sb.WriteByte('f')
			case BrownMushroom, RedMushroom:
				sb.WriteByte('m')
			case Sand:
				sb.WriteByte('S')
			case Gravel:
				sb.WriteByte('r')
			default:
				sb.WriteByte('?')
			}
		}
		sb.WriteByte('\n')
	}
	t.Logf("Classic cross-section (x=%d):\n%s", midX, sb.String())
}
