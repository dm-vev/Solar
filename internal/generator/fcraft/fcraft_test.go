package fcraft_test

import (
	"testing"

	"github.com/solar-mc/solar/internal/generator/core"
	"github.com/solar-mc/solar/internal/generator/fcraft"
)

func TestFCraftGenerator(t *testing.T) {
	gen := mustFind(t, "Hills")
	lvl, err := core.Generate(gen, "hills", 32, 32, 32, core.Args{Seed: 42, Biome: core.Forest})
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
		t.Fatal("fcraft generator produced only air")
	}
}

func TestModuleRegistersGenerators(t *testing.T) {
	registry := core.NewRegistry()
	registry.RegisterModule(fcraft.Module)
	for _, name := range fcraft.Templates {
		if _, ok := registry.Find(name); !ok {
			t.Fatalf("%s not registered", name)
		}
	}
}

func mustFind(t *testing.T, name string) core.Generator {
	registry := core.NewRegistry()
	registry.RegisterModule(fcraft.Module)
	gen, ok := registry.Find(name)
	if !ok {
		t.Fatalf("%s generator not found", name)
	}
	return gen
}
