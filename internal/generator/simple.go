package generator

import "math/rand"

// SimpleModule groups lightweight built-in generators.
var SimpleModule = Module{Name: "simple", Generators: SimpleGenerators}

// SimpleGenerators returns the list of simple generators.
func SimpleGenerators() []Generator {
	return []Generator{
		{Name: "Flat", Type: GenTypeSimple, Func: genFlat},
		{Name: "Pixel", Type: GenTypeSimple, Func: genPixel},
		{Name: "Empty", Type: GenTypeSimple, Func: genEmpty},
		{Name: "Space", Type: GenTypeSimple, Func: genSpace},
		{Name: "Rainbow", Type: GenTypeSimple, Func: genRainbow},
	}
}

func genFlat(a *Args, lvl *Level) error {
	biome := biomeOrDefault(a)
	grassHeight := lvl.Height / 2
	if a.Seed >= 0 && a.Seed <= lvl.Height {
		grassHeight = a.Seed
	}
	grassY := grassHeight - 1
	if grassY < 0 {
		grassY = 0
	}

	for y := 0; y <= grassY-1; y++ {
		FillLayer(lvl, y, biome.Ground)
	}
	if grassY >= 0 && grassY < lvl.Height {
		FillLayer(lvl, grassY, biome.Surface)
	}
	lvl.Spawn.Y = grassHeight + 2
	return nil
}

func genEmpty(a *Args, lvl *Level) error {
	FillLayer(lvl, 0, Bedrock)
	lvl.Spawn.Y = 2
	return nil
}

func genPixel(a *Args, lvl *Level) error {
	maxX := lvl.Width - 1
	maxY := lvl.Height - 1
	maxZ := lvl.Length - 1

	FillCuboid(lvl, 0, 1, 0, maxX, maxY, 0, WhiteWool)
	FillCuboid(lvl, 0, 1, maxZ, maxX, maxY, maxZ, WhiteWool)
	FillCuboid(lvl, 0, 1, 0, 0, maxY, maxZ, WhiteWool)
	FillCuboid(lvl, maxX, 1, 0, maxX, maxY, maxZ, WhiteWool)
	FillCuboid(lvl, 0, 0, 0, maxX, 0, maxZ, Bedrock)
	lvl.Spawn.Y = 2
	return nil
}

func genSpace(a *Args, lvl *Level) error {
	rng := rand.New(rand.NewSource(int64(a.Seed)))
	biome := Space

	maxX := lvl.Width - 1
	maxY := lvl.Height - 1
	maxZ := lvl.Length - 1

	nextBlock := func() byte {
		if rng.Intn(100) == 0 {
			return biome.Ground
		}
		return biome.Surface
	}

	FillCuboidFn(lvl, 0, 2, 0, maxX, maxY, 0, nextBlock)
	FillCuboidFn(lvl, 0, 2, maxZ, maxX, maxY, maxZ, nextBlock)
	FillCuboidFn(lvl, 0, 2, 0, 0, maxY, maxZ, nextBlock)
	FillCuboidFn(lvl, maxX, 2, 0, maxX, maxY, maxZ, nextBlock)

	FillCuboid(lvl, 0, 0, 0, maxX, 0, maxZ, Bedrock)
	FillCuboidFn(lvl, 0, 1, 0, maxX, 1, maxZ, nextBlock)
	FillCuboid(lvl, 0, maxY, 0, maxX, maxY, maxZ, biome.Surface)

	lvl.Spawn.Y = 3
	return nil
}

func genRainbow(a *Args, lvl *Level) error {
	rng := rand.New(rand.NewSource(int64(a.Seed)))
	nextBlock := func() byte { return byte(rng.Intn(15) + RedWool) }

	maxX := lvl.Width - 1
	maxY := lvl.Height - 1
	maxZ := lvl.Length - 1

	FillCuboidFn(lvl, 0, 1, 0, maxX, maxY, 0, nextBlock)
	FillCuboidFn(lvl, 0, 1, maxZ, maxX, maxY, maxZ, nextBlock)
	FillCuboidFn(lvl, 0, 1, 0, 0, maxY, maxZ, nextBlock)
	FillCuboidFn(lvl, maxX, 1, 0, maxX, maxY, maxZ, nextBlock)

	FillCuboidFn(lvl, 0, 0, 0, maxX, 0, maxZ, nextBlock)
	FillCuboidFn(lvl, 0, maxY, 0, maxX, maxY, maxZ, nextBlock)

	lvl.Spawn.Y = 2
	return nil
}

// FillLayer fills an entire horizontal layer with a single block.
func FillLayer(lvl *Level, y int, block byte) {
	if y < 0 || y >= lvl.Height {
		return
	}
	for z := 0; z < lvl.Length; z++ {
		for x := 0; x < lvl.Width; x++ {
			lvl.Blocks[PackIndex(*lvl, x, y, z)] = block
		}
	}
}

func biomeOrDefault(a *Args) Biome {
	if a.Biome.Name == "" {
		return Forest
	}
	return a.Biome
}
