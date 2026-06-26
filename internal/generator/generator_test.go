package generator

import (
	"slices"
	"testing"
)

func TestDefaultRegistryWrappers(t *testing.T) {
	modules := BuiltinModules()
	if len(modules) != 4 {
		t.Fatalf("BuiltinModules = %d, want 4", len(modules))
	}

	RegisterDefaults()
	registry := DefaultRegistry()
	if registry == nil {
		t.Fatal("DefaultRegistry returned nil")
	}

	gen, ok := Find("flat")
	if !ok {
		t.Fatal("Find did not find Flat after RegisterDefaults")
	}
	lvl, err := Generate(gen, "flat", 4, 4, 4, Args{Seed: 2, Biome: Forest})
	if err != nil {
		t.Fatalf("Generate flat: %v", err)
	}
	if got := GetBlock(lvl, 0, 1, 0); got != Grass {
		t.Fatalf("flat surface block = %d, want grass", got)
	}

	names := Names()
	if !slices.Contains(names[GenType("Simple")], "Flat") {
		t.Fatalf("Names missing Flat: %v", names)
	}
	if len(AllGenerators()) == 0 {
		t.Fatal("AllGenerators returned no generators")
	}
}
