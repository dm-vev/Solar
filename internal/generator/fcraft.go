package generator

import (
	"math"
	"math/rand"
)

// fCraftTemplates lists the named fCraft generator templates.
var fCraftTemplates = []string{
	"Archipelago", "Atoll", "Bay", "Dunes", "Hills", "Ice", "Island2",
	"Lake", "Mountains2", "Peninsula", "Random", "River", "Streams",
}

// FCraftGenerators returns all fCraft template generators.
func FCraftGenerators() []Generator {
	gens := make([]Generator, 0, len(fCraftTemplates))
	for _, t := range fCraftTemplates {
		t := t
		gens = append(gens, Generator{
			Name: t,
			Type: GenTypeFCraft,
			Desc: "fCraft " + t + " terrain template",
			Func: func(args *Args, lvl *Level) error {
				return genFCraftTemplate(args, lvl, t)
			},
		})
	}
	return gens
}

type fCraftMapGenArgs struct {
	Seed               int
	Biome              Biome
	FeatureScale       int
	DetailScale        int
	Roughness          float64
	UseBias            bool
	DelayBias          bool
	RaisedCorners      int
	LoweredCorners     int
	Bias               float64
	MidPoint           float64
	MarbledHeightmap   bool
	InvertHeightmap    bool
	MatchWaterCoverage bool
	WaterCoverage      float64
	MaxHeight          int
	MaxDepth           int
	MaxHeightVariation int
	MaxDepthVariation  int
	WaterLevel         int
	AddWater           bool
	AddBeaches         bool
	AddTrees           bool
	AddSnow            bool
	SnowAltitude       int
	SnowTransition     int
	CliffSmoothing     bool
	CliffThreshold     float64
	BeachExtent        int
	BeachHeight        int
	TreeSpacingMin     int
	TreeSpacingMax     int
}

func makeTemplate(name string) fCraftMapGenArgs {
	// Defaults similar to MCGalaxy's fCraft defaults.
	args := fCraftMapGenArgs{
		Seed:               0,
		FeatureScale:       32,
		DetailScale:        4,
		Roughness:          0.5,
		UseBias:            true,
		RaisedCorners:      2,
		LoweredCorners:     2,
		Bias:               0.5,
		MidPoint:           0.0,
		MarbledHeightmap:   false,
		InvertHeightmap:    false,
		MatchWaterCoverage: true,
		WaterCoverage:      0.5,
		MaxHeight:          24,
		MaxDepth:           12,
		MaxHeightVariation: 0,
		MaxDepthVariation:  0,
		AddWater:           true,
		AddBeaches:         true,
		AddTrees:           true,
		AddSnow:            false,
		SnowAltitude:       64,
		SnowTransition:     8,
		CliffSmoothing:     true,
		CliffThreshold:     0.4,
		BeachExtent:        3,
		BeachHeight:        2,
		TreeSpacingMin:     6,
		TreeSpacingMax:     12,
	}

	switch name {
	case "Archipelago":
		args.FeatureScale = 16
		args.DetailScale = 4
		args.Roughness = 0.6
		args.Bias = 0.4
		args.MidPoint = 0.5
	case "Atoll":
		args.FeatureScale = 32
		args.DetailScale = 4
		args.Roughness = 0.5
		args.Bias = 0.6
		args.MidPoint = 0.0
		args.RaisedCorners = 4
	case "Bay":
		args.FeatureScale = 48
		args.DetailScale = 4
		args.Roughness = 0.5
		args.Bias = 0.3
		args.MidPoint = 0.0
		args.LoweredCorners = 2
	case "Dunes":
		args.FeatureScale = 16
		args.DetailScale = 2
		args.Roughness = 0.8
		args.Bias = 0.3
		args.CliffThreshold = 0.6
	case "Hills":
		args.FeatureScale = 32
		args.DetailScale = 4
		args.Roughness = 0.5
		args.MaxHeight = 16
	case "Ice":
		args.FeatureScale = 32
		args.DetailScale = 4
		args.Roughness = 0.4
		args.Bias = 0.4
		args.AddSnow = true
	case "Island2":
		args.FeatureScale = 24
		args.DetailScale = 4
		args.Roughness = 0.5
		args.Bias = 0.5
		args.RaisedCorners = 4
	case "Lake":
		args.FeatureScale = 48
		args.DetailScale = 4
		args.Roughness = 0.4
		args.Bias = -0.2
		args.MidPoint = 0.5
	case "Mountains2":
		args.FeatureScale = 16
		args.DetailScale = 4
		args.Roughness = 0.7
		args.MaxHeight = 48
		args.MaxDepth = 20
		args.CliffThreshold = 0.5
	case "Peninsula":
		args.FeatureScale = 40
		args.DetailScale = 4
		args.Roughness = 0.5
		args.Bias = 0.3
		args.RaisedCorners = 2
	case "River":
		args.FeatureScale = 40
		args.DetailScale = 4
		args.Roughness = 0.5
		args.Bias = -0.1
		args.MidPoint = 0.2
	case "Streams":
		args.FeatureScale = 24
		args.DetailScale = 4
		args.Roughness = 0.6
		args.Bias = -0.1
		args.MidPoint = 0.1
	case "Random":
		args.FeatureScale = 20 + rand.Intn(40)
		args.DetailScale = 2 + rand.Intn(6)
		args.Roughness = 0.3 + rand.Float64()*0.5
	}
	return args
}

func genFCraftTemplate(args *Args, lvl *Level, template string) error {
	genArgs := makeTemplate(template)
	genArgs.Biome = biomeOrDefault(args)
	genArgs.Seed = args.Seed

	ratio := float64(lvl.Height) / 96.0
	genArgs.MaxHeight = Round(float64(genArgs.MaxHeight) * ratio)
	genArgs.MaxDepth = Round(float64(genArgs.MaxDepth) * ratio)
	genArgs.SnowAltitude = Round(float64(genArgs.SnowAltitude) * ratio)
	genArgs.WaterLevel = (lvl.Height - 1) / 2

	newFCraftMapGen(genArgs).Generate(lvl)
	lvl.Spawn.Y = genArgs.WaterLevel + 2
	return nil
}

type fCraftMapGen struct {
	args       fCraftMapGenArgs
	randSource *rand.Rand
	noise      *Noise
	heightmap  []float64
	slopemap   []float64
	surfaceMap []uint16
}

func newFCraftMapGen(args fCraftMapGenArgs) *fCraftMapGen {
	return &fCraftMapGen{
		args:       args,
		randSource: rand.New(rand.NewSource(int64(args.Seed))),
		noise:      NewNoise(args.Seed, args.FeatureScale, args.DetailScale, args.Roughness),
	}
}

func (g *fCraftMapGen) Generate(lvl *Level) {
	g.generateHeightmap(lvl.Width, lvl.Length)
	g.generateMap(lvl)
}

func (g *fCraftMapGen) generateHeightmap(width, length int) {
	heightmap := make([]float64, width*length)
	PerlinNoise(g.args.Seed, heightmap, width, length, g.args.FeatureScale, g.args.DetailScale, g.args.Roughness)

	if g.args.UseBias && !g.args.DelayBias {
		NormalizeFloat(heightmap)
		g.applyBias(heightmap, width, length)
	}
	NormalizeFloat(heightmap)
	if g.args.MarbledHeightmap {
		Marble(heightmap)
	}
	if g.args.InvertHeightmap {
		Invert(heightmap)
	}
	if g.args.UseBias && g.args.DelayBias {
		NormalizeFloat(heightmap)
		g.applyBias(heightmap, width, length)
	}
	NormalizeFloat(heightmap)
	g.heightmap = heightmap
}

func (g *fCraftMapGen) applyBias(heightmap []float64, width, length int) {
	corners := make([]float64, 4)
	c := 0
	for i := 0; i < g.args.RaisedCorners; i++ {
		corners[c] = g.args.Bias
		c++
	}
	for i := 0; i < g.args.LoweredCorners; i++ {
		corners[c] = -g.args.Bias
		c++
	}
	midpoint := g.args.MidPoint * g.args.Bias
	// Shuffle corners deterministically using the generator seed.
	keys := make([]int, len(corners))
	for i := range keys {
		keys[i] = g.randSource.Int()
	}
	// Sort corners by keys.
	for i := 1; i < len(corners); i++ {
		for j := i; j > 0 && keys[j-1] > keys[j]; j-- {
			keys[j-1], keys[j] = keys[j], keys[j-1]
			corners[j-1], corners[j] = corners[j], corners[j-1]
		}
	}
	ApplyBias(heightmap, width, length, corners[0], corners[1], corners[2], corners[3], midpoint)
}

func (g *fCraftMapGen) generateMap(lvl *Level) {
	width := lvl.Width
	length := lvl.Length
	args := g.args

	desiredWaterLevel := 0.5
	if args.MatchWaterCoverage {
		sorted := SortCopy(g.heightmap)
		desiredWaterLevel = FindThresholdSorted(sorted, args.WaterCoverage)
	}

	aboveWaterMultiplier := 0.0
	if desiredWaterLevel != 1 {
		aboveWaterMultiplier = float64(args.MaxHeight) / (1 - desiredWaterLevel)
	}

	var blurred []float64
	if args.CliffSmoothing {
		blurred = GaussianBlur5X5(g.heightmap, width, length)
		g.slopemap = CalculateSlope(blurred, width, length)
	} else {
		g.slopemap = CalculateSlope(g.heightmap, width, length)
	}
	_ = blurred

	var altmap []float64
	if args.MaxHeightVariation != 0 || args.MaxDepthVariation != 0 {
		altmap = make([]float64, width*length)
		blendDetail := int(math.Log(math.Max(float64(width), float64(length))) / math.Log(2))
		if blendDetail < 1 {
			blendDetail = 1
		}
		PerlinNoise(g.randSource.Int(), altmap, width, length, blendDetail, 3, 0.5)
		NormalizeToRange(altmap, -1, 1)
	}
	g.surfaceMap = make([]uint16, width*length)

	for z := 0; z < length; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			level, surface := g.fillColumn(lvl, x, z, i, desiredWaterLevel, aboveWaterMultiplier, altmap)
			g.surfaceMap[i] = uint16(surface)
			_ = level
		}
	}

	if args.AddBeaches {
		g.addBeaches(lvl)
	}
	if args.AddTrees && g.args.Biome.TreeDefault("fCraft") != "" {
		g.generateTrees(lvl)
	}
}

func (g *fCraftMapGen) fillColumn(lvl *Level, x, z, i int, desiredWaterLevel, aboveWaterMultiplier float64, altmap []float64) (int, int) {
	if g.heightmap[i] < desiredWaterLevel {
		return g.fillColumnUnderwater(lvl, x, z, i, desiredWaterLevel, altmap)
	}
	return g.fillColumnAboveWater(lvl, x, z, i, desiredWaterLevel, aboveWaterMultiplier, altmap)
}

func (g *fCraftMapGen) fillColumnUnderwater(lvl *Level, x, z, i int, desiredWaterLevel float64, altmap []float64) (int, int) {
	width := lvl.Width
	length := lvl.Length
	height := lvl.Height
	args := g.args

	depth := float64(args.MaxDepth)
	if altmap != nil {
		depth += altmap[i] * float64(args.MaxDepthVariation)
	}
	slope := g.slopemap[i] * depth
	level := args.WaterLevel - Round((1-g.heightmap[i]/desiredWaterLevel)*depth)
	surface := args.WaterLevel

	if args.AddWater {
		g.fillUnderwaterWithWater(lvl, x, z, width, length, height, level, args)
	} else {
		g.fillColumnGround(lvl, x, z, width, length, height, level, slope, args, args.Biome.Surface)
	}
	return surface, surface
}

func (g *fCraftMapGen) fillUnderwaterWithWater(lvl *Level, x, z, width, length, height, level int, args fCraftMapGenArgs) {
	idx := (args.WaterLevel*length+z)*width + x
	if args.WaterLevel >= 0 && args.WaterLevel < height {
		lvl.Blocks[idx] = args.Biome.Water
	}
	for yy := args.WaterLevel; yy > level && yy >= 0; yy-- {
		if yy < height {
			lvl.Blocks[idx] = args.Biome.Water
		}
		idx -= width * length
	}
	idx = ((level+1)*length+z)*width + x
	for yy := level; yy >= 0; yy-- {
		idx -= length * width
		if yy >= height {
			continue
		}
		if level-yy < 3 {
			lvl.Blocks[idx] = args.Biome.BeachSandy
		} else {
			lvl.Blocks[idx] = args.Biome.Bedrock
		}
	}
}

func (g *fCraftMapGen) fillColumnAboveWater(lvl *Level, x, z, i int, desiredWaterLevel, aboveWaterMultiplier float64, altmap []float64) (int, int) {
	width := lvl.Width
	length := lvl.Length
	height := lvl.Height
	args := g.args

	maxH := float64(args.MaxHeight)
	if altmap != nil {
		maxH += altmap[i] * float64(args.MaxHeightVariation)
	}
	slope := g.slopemap[i] * maxH
	level := args.WaterLevel
	if maxH != 0 {
		level = args.WaterLevel + Round((g.heightmap[i]-desiredWaterLevel)*aboveWaterMultiplier/float64(args.MaxHeight)*maxH)
	}
	surface := level

	snowStart := args.SnowAltitude - args.SnowTransition
	snow := args.AddSnow && (level > args.SnowAltitude || (level > snowStart && g.randSource.Float64() < float64(level-snowStart)/float64(args.SnowAltitude-snowStart)))

	topBlock := args.Biome.Surface
	if snow {
		topBlock = WhiteWool
	}
	if slope >= args.CliffThreshold {
		topBlock = args.Biome.Cliff
	}

	g.fillColumnGround(lvl, x, z, width, length, height, level, slope, args, topBlock)

	if snow && slope < args.CliffThreshold {
		idx := (level*length+z)*width + x
		if level >= 0 && level < height {
			lvl.Blocks[idx] = WhiteWool
		}
	}

	return surface, surface
}

// fillColumnGround fills a column from level down to 0 with surface, ground,
// and bedrock layers based on slope and cliff threshold.
func (g *fCraftMapGen) fillColumnGround(lvl *Level, x, z, width, length, height, level int, slope float64, args fCraftMapGenArgs, topBlock byte) {
	idx := (level*length+z)*width + x
	if level >= 0 && level < height {
		lvl.Blocks[idx] = topBlock
	}
	for yy := level - 1; yy >= 0; yy-- {
		idx -= length * width
		if yy >= height {
			continue
		}
		if level-yy < args.groundThickness() {
			if slope < args.CliffThreshold {
				lvl.Blocks[idx] = args.Biome.Ground
			} else {
				lvl.Blocks[idx] = args.Biome.Cliff
			}
		} else {
			lvl.Blocks[idx] = args.Biome.Bedrock
		}
	}
}

func randRange(r *rand.Rand, min, max int) int {
	if max <= min {
		return min
	}
	return min + r.Intn(max-min+1)
}

func (a fCraftMapGenArgs) groundThickness() int {
	if a.Biome.Name == "Arctic" {
		return 1
	}
	return 5
}

func (g *fCraftMapGen) addBeaches(lvl *Level) {
	args := g.args
	width := lvl.Width
	length := lvl.Length
	beachExtentSqr := (args.BeachExtent + 1) * (args.BeachExtent + 1)

	for x := 0; x < width; x++ {
		for z := 0; z < length; z++ {
			for y := args.WaterLevel; y <= args.WaterLevel+args.BeachHeight; y++ {
				idx := (y*length+z)*width + x
				if lvl.Blocks[idx] != args.Biome.Surface {
					continue
				}
				found := false
				for dx := -args.BeachExtent; dx <= args.BeachExtent && !found; dx++ {
					for dz := -args.BeachExtent; dz <= args.BeachExtent && !found; dz++ {
						for dy := -args.BeachHeight; dy <= 0; dy++ {
							if dx*dx+dy*dy+dz*dz > beachExtentSqr {
								continue
							}
							xx := x + dx
							yy := y + dy
							zz := z + dz
							if xx < 0 || xx >= width || yy < 0 || yy >= lvl.Height || zz < 0 || zz >= length {
								continue
							}
							bidx := (yy*length+zz)*width + xx
							if lvl.Blocks[bidx] == args.Biome.Water {
								found = true
								break
							}
						}
					}
				}
				if found {
					lvl.Blocks[idx] = args.Biome.BeachSandy
					if y > 0 {
						underIdx := ((y-1)*length+z)*width + x
						if lvl.Blocks[underIdx] == args.Biome.Ground {
							lvl.Blocks[underIdx] = args.Biome.BeachSandy
						}
					}
				}
			}
		}
	}
}

func (g *fCraftMapGen) generateTrees(lvl *Level) {
	args := g.args
	width := lvl.Width
	length := lvl.Length
	rn := rand.New(rand.NewSource(int64(args.Seed) + 1))
	treeType := args.Biome.TreeDefault("fCraft")
	ctor, ok := FindTree(treeType)
	if !ok {
		return
	}
	tree := ctor()

	for x := 0; x < width; x += rn.Intn(args.TreeSpacingMax-args.TreeSpacingMin+1) + args.TreeSpacingMin {
		for z := 0; z < length; z += rn.Intn(args.TreeSpacingMax-args.TreeSpacingMin+1) + args.TreeSpacingMin {
			nx := x + randRange(rn, -args.TreeSpacingMin/2, args.TreeSpacingMax/2)
			nz := z + randRange(rn, -args.TreeSpacingMin/2, args.TreeSpacingMax/2)
			if nx < 0 || nx >= width || nz < 0 || nz >= length {
				continue
			}
			ny := int(g.surfaceMap[nx*length+nz])
			idx := (ny*length+nz)*width + nx
			if lvl.Blocks[idx] == args.Biome.Surface && g.slopemap[nx*length+nz] < 0.5 {
				tree.SetData(rn, tree.DefaultSize(rn))
				nh := tree.Height()
				if ny+1+nh+nh/2 > lvl.Height {
					continue
				}
				tree.Generate(nx, ny+1, nz, func(xT, yT, zT int, bT byte) {
					if xT < 0 || xT >= width || yT < 0 || yT >= lvl.Height || zT < 0 || zT >= length {
						return
					}
					bidx := (yT*length+zT)*width + xT
					if lvl.Blocks[bidx] == Air {
						lvl.Blocks[bidx] = bT
					}
				})
			}
		}
	}
}
