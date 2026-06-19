package classic_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"testing"

	"github.com/solar-mc/solar/internal/generator/classic"
	"github.com/solar-mc/solar/internal/generator/core"
)

func TestClassicGenerator(t *testing.T) {
	gen := mustFind(t, "Classic")
	lvl, err := core.Generate(gen, "classic", 32, 32, 32, core.Args{Seed: 12345, Biome: core.Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	nonAir := 0
	for _, b := range lvl.Blocks {
		if b != core.Air {
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

	gen := mustFind(t, "Classic")

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

			lvl, err := core.Generate(gen, "classic", tc.width, tc.height, tc.length, core.Args{Seed: tc.seed, Biome: core.Forest})
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

func TestClassicTerrainShape(t *testing.T) {
	gen := mustFind(t, "Classic")
	lvl, err := core.Generate(gen, "classic", 128, 64, 128, core.Args{Seed: 12345, Biome: core.Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	for x := 0; x < lvl.Width; x++ {
		for z := 0; z < lvl.Length; z++ {
			if lvl.Blocks[core.PackIndex(*lvl, x, 0, z)] != core.Lava {
				t.Fatalf("bottom layer at %d,%d is not lava", x, z)
			}
		}
	}

	var grass, stone, air, water, lava int
	for y := 0; y < lvl.Height; y++ {
		for z := 0; z < lvl.Length; z++ {
			for x := 0; x < lvl.Width; x++ {
				switch lvl.Blocks[core.PackIndex(*lvl, x, y, z)] {
				case core.Grass:
					grass++
				case core.Stone:
					stone++
				case core.Air:
					air++
				case core.StillWater:
					water++
				case core.Lava, core.StillLava:
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
	if stone < grass*4 {
		t.Fatalf("too little stone: grass=%d stone=%d", grass, stone)
	}
}

func TestTreeGenerator(t *testing.T) {
	tree := core.NewClassicTree()
	rng := rand.New(rand.NewSource(0))
	tree.SetData(rng, 6)
	blocks := make(map[[3]int]byte)
	tree.Generate(0, 0, 0, func(x, y, z int, block byte) {
		blocks[[3]int{x, y, z}] = block
	})
	if len(blocks) == 0 {
		t.Fatal("tree generated no blocks")
	}
	if blocks[[3]int{0, 0, 0}] != core.Log {
		t.Fatal("tree trunk missing at origin")
	}
}

func TestModuleRegistersGenerators(t *testing.T) {
	registry := core.NewRegistry()
	registry.RegisterModule(classic.Module)
	if _, ok := registry.Find("Classic"); !ok {
		t.Fatal("Classic not registered")
	}
}

func mustFind(t *testing.T, name string) core.Generator {
	registry := core.NewRegistry()
	registry.RegisterModule(classic.Module)
	gen, ok := registry.Find(name)
	if !ok {
		t.Fatalf("%s generator not found", name)
	}
	return gen
}
