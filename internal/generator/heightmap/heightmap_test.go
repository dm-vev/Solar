package heightmap_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/solar-mc/solar/internal/generator/core"
	"github.com/solar-mc/solar/internal/generator/heightmap"
)

func TestHeightmapGenerator(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.SetGray(x, y, color.Gray{Y: byte((y*4 + x) * 4)})
		}
	}

	lvl := core.NewLevel("h", 4, 16, 4)
	if err := heightmap.Apply(lvl, img, core.Forest); err != nil {
		t.Fatalf("applyHeightmap: %v", err)
	}
	if lvl.Spawn.Y < 1 || lvl.Spawn.Y >= lvl.Height {
		t.Fatalf("spawn y = %d, want within level", lvl.Spawn.Y)
	}
}

func TestModuleRegistersGenerators(t *testing.T) {
	registry := core.NewRegistry()
	registry.RegisterModule(heightmap.Module)
	if _, ok := registry.Find("Heightmap"); !ok {
		t.Fatal("Heightmap not registered")
	}
}
