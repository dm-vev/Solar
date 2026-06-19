package simple

import (
	"math/rand"

	"github.com/solar-mc/solar/internal/generator/core"
)

// Module groups lightweight built-in generators.
var Module = core.Module{Name: "simple", Generators: Generators}

// Generators returns the list of simple generators.
func Generators() []core.Generator {
	return []core.Generator{
		{Name: "Flat", Type: core.GenTypeSimple, Func: genFlat},
		{Name: "Pixel", Type: core.GenTypeSimple, Func: genPixel},
		{Name: "Empty", Type: core.GenTypeSimple, Func: genEmpty},
		{Name: "Space", Type: core.GenTypeSimple, Func: genSpace},
		{Name: "Rainbow", Type: core.GenTypeSimple, Func: genRainbow},
	}
}

func genFlat(a *core.Args, lvl *core.Level) error {
	biome := core.BiomeOrDefault(a)
	grassHeight := lvl.Height / 2
	if a.Seed >= 0 && a.Seed <= lvl.Height {
		grassHeight = a.Seed
	}
	grassY := grassHeight - 1
	if grassY < 0 {
		grassY = 0
	}

	for y := 0; y <= grassY-1; y++ {
		core.FillLayer(lvl, y, biome.Ground)
	}
	if grassY >= 0 && grassY < lvl.Height {
		core.FillLayer(lvl, grassY, biome.Surface)
	}
	lvl.Spawn.Y = grassHeight + 2
	return nil
}

func genEmpty(a *core.Args, lvl *core.Level) error {
	core.FillLayer(lvl, 0, core.Bedrock)
	lvl.Spawn.Y = 2
	return nil
}

func genPixel(a *core.Args, lvl *core.Level) error {
	maxX := lvl.Width - 1
	maxY := lvl.Height - 1
	maxZ := lvl.Length - 1

	core.FillCuboid(lvl, 0, 1, 0, maxX, maxY, 0, core.WhiteWool)
	core.FillCuboid(lvl, 0, 1, maxZ, maxX, maxY, maxZ, core.WhiteWool)
	core.FillCuboid(lvl, 0, 1, 0, 0, maxY, maxZ, core.WhiteWool)
	core.FillCuboid(lvl, maxX, 1, 0, maxX, maxY, maxZ, core.WhiteWool)
	core.FillCuboid(lvl, 0, 0, 0, maxX, 0, maxZ, core.Bedrock)
	lvl.Spawn.Y = 2
	return nil
}

func genSpace(a *core.Args, lvl *core.Level) error {
	rng := rand.New(rand.NewSource(int64(a.Seed)))
	biome := core.Space

	maxX := lvl.Width - 1
	maxY := lvl.Height - 1
	maxZ := lvl.Length - 1

	nextBlock := func() byte {
		if rng.Intn(100) == 0 {
			return biome.Ground
		}
		return biome.Surface
	}

	core.FillCuboidFn(lvl, 0, 2, 0, maxX, maxY, 0, nextBlock)
	core.FillCuboidFn(lvl, 0, 2, maxZ, maxX, maxY, maxZ, nextBlock)
	core.FillCuboidFn(lvl, 0, 2, 0, 0, maxY, maxZ, nextBlock)
	core.FillCuboidFn(lvl, maxX, 2, 0, maxX, maxY, maxZ, nextBlock)

	core.FillCuboid(lvl, 0, 0, 0, maxX, 0, maxZ, core.Bedrock)
	core.FillCuboidFn(lvl, 0, 1, 0, maxX, 1, maxZ, nextBlock)
	core.FillCuboid(lvl, 0, maxY, 0, maxX, maxY, maxZ, biome.Surface)

	lvl.Spawn.Y = 3
	return nil
}

func genRainbow(a *core.Args, lvl *core.Level) error {
	rng := rand.New(rand.NewSource(int64(a.Seed)))
	nextBlock := func() byte { return byte(rng.Intn(15) + core.RedWool) }

	maxX := lvl.Width - 1
	maxY := lvl.Height - 1
	maxZ := lvl.Length - 1

	core.FillCuboidFn(lvl, 0, 1, 0, maxX, maxY, 0, nextBlock)
	core.FillCuboidFn(lvl, 0, 1, maxZ, maxX, maxY, maxZ, nextBlock)
	core.FillCuboidFn(lvl, 0, 1, 0, 0, maxY, maxZ, nextBlock)
	core.FillCuboidFn(lvl, maxX, 1, 0, maxX, maxY, maxZ, nextBlock)

	core.FillCuboidFn(lvl, 0, 0, 0, maxX, 0, maxZ, nextBlock)
	core.FillCuboidFn(lvl, 0, maxY, 0, maxX, maxY, maxZ, nextBlock)

	lvl.Spawn.Y = 2
	return nil
}
