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
