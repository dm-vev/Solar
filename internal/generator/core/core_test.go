package core_test

import (
	"testing"

	"github.com/solar-mc/solar/internal/generator/core"
)

func TestNewLevel(t *testing.T) {
	lvl := core.NewLevel("test", 4, 3, 5)
	if lvl.Volume() != 60 {
		t.Fatalf("volume = %d, want 60", lvl.Volume())
	}
}

func TestValidateDimensions(t *testing.T) {
	if err := core.ValidateDimensions(1, 1, 1); err != nil {
		t.Fatalf("valid dimensions failed: %v", err)
	}
	if err := core.ValidateDimensions(0, 1, 1); err == nil {
		t.Fatal("expected error for zero width")
	}
	if err := core.ValidateDimensions(10000, 10000, 10000); err == nil {
		t.Fatal("expected error for oversized dimensions")
	}
}

func TestParseArgs(t *testing.T) {
	args, err := core.ParseArgs("123 Forest")
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
	args, err := core.ParseArgs("Desert")
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
	_, err := core.ParseArgs("Void")
	if err == nil {
		t.Fatal("expected error for unknown biome")
	}
}

func TestRegistryRegisterModule(t *testing.T) {
	t.Parallel()

	registry := core.NewRegistry()
	registry.RegisterModule(core.Module{
		Name: "test",
		Generators: func() []core.Generator {
			return []core.Generator{
				{Name: "One", Type: core.GenTypeSimple, Func: emptyGen},
				{Name: "Two", Type: core.GenTypeAdvanced, Func: emptyGen},
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

func emptyGen(a *core.Args, lvl *core.Level) error { return nil }
