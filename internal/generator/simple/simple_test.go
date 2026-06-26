package simple_test

import (
	"testing"

	"github.com/solar-mc/solar/internal/generator/core"
	"github.com/solar-mc/solar/internal/generator/simple"
)

func TestSimpleFlatGenerator(t *testing.T) {
	gen := mustFind(t, "Flat")
	lvl, err := core.Generate(gen, "flat", 16, 16, 16, core.Args{Seed: 8, Biome: core.Forest})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if lvl.Spawn.Y != 10 {
		t.Fatalf("spawn y = %d, want 10", lvl.Spawn.Y)
	}
}

func TestSimpleEmptyGenerator(t *testing.T) {
	gen := mustFind(t, "Empty")
	lvl, err := core.Generate(gen, "empty", 8, 8, 8, core.Args{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for x := 0; x < 8; x++ {
		for z := 0; z < 8; z++ {
			if lvl.Blocks[core.PackIndex(*lvl, x, 0, z)] != core.Bedrock {
				t.Fatalf("bottom layer is not bedrock at %d,%d", x, z)
			}
		}
	}
}

func TestSimplePixelGenerator(t *testing.T) {
	gen := mustFind(t, "Pixel")
	lvl, err := core.Generate(gen, "pixel", 4, 4, 4, core.Args{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := lvl.Spawn.Y; got != 2 {
		t.Fatalf("spawn y = %d, want 2", got)
	}
	if got := core.GetBlock(lvl, 0, 0, 0); got != core.Bedrock {
		t.Fatalf("bottom block = %d, want bedrock", got)
	}
	if got := core.GetBlock(lvl, 0, 2, 2); got != core.WhiteWool {
		t.Fatalf("wall block = %d, want white wool", got)
	}
}

func TestSimpleSpaceGenerator(t *testing.T) {
	gen := mustFind(t, "Space")
	lvl, err := core.Generate(gen, "space", 4, 4, 4, core.Args{Seed: 1})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := lvl.Spawn.Y; got != 3 {
		t.Fatalf("spawn y = %d, want 3", got)
	}
	if got := core.GetBlock(lvl, 0, 0, 0); got != core.Bedrock {
		t.Fatalf("bottom block = %d, want bedrock", got)
	}
	if got := core.GetBlock(lvl, 1, 3, 1); got != core.Obsidian {
		t.Fatalf("top block = %d, want obsidian", got)
	}
}

func TestSimpleRainbowGenerator(t *testing.T) {
	gen := mustFind(t, "Rainbow")
	lvl, err := core.Generate(gen, "rainbow", 4, 4, 4, core.Args{Seed: 2})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := lvl.Spawn.Y; got != 2 {
		t.Fatalf("spawn y = %d, want 2", got)
	}
	block := core.GetBlock(lvl, 0, 0, 0)
	if block < core.RedWool || block > core.WhiteWool {
		t.Fatalf("rainbow block = %d, want wool range", block)
	}
}

func TestModuleRegistersGenerators(t *testing.T) {
	registry := core.NewRegistry()
	registry.RegisterModule(simple.Module)
	for _, name := range []string{"Flat", "Empty", "Pixel", "Space", "Rainbow"} {
		if _, ok := registry.Find(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
}

func mustFind(t *testing.T, name string) core.Generator {
	registry := core.NewRegistry()
	registry.RegisterModule(simple.Module)
	gen, ok := registry.Find(name)
	if !ok {
		t.Fatalf("%s generator not found", name)
	}
	return gen
}
