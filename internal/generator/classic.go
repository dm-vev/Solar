package generator

import (
	"math"
	"math/rand"
)

// ClassicGenerators returns the MCGalaxy Classic generator.
func ClassicGenerators() []Generator {
	return []Generator{{
		Name: "Classic",
		Type: GenTypeSimple,
		Desc: "Original Minecraft Classic terrain with caves and trees",
		Func: genClassic,
	}}
}

type classicGen struct {
	lvl        *Level
	width      int
	length     int
	height     int
	oneY       int
	waterLevel int
	minHeight  int
	biome      Biome
	rnd        *JavaRandom
	seed       int
	heightmap  []int16
}

func genClassic(args *Args, lvl *Level) error {
	g := &classicGen{
		lvl:        lvl,
		width:      lvl.Width,
		length:     lvl.Length,
		height:     lvl.Height,
		oneY:       lvl.Width * lvl.Length,
		waterLevel: lvl.Height / 2,
		minHeight:  lvl.Height,
		biome:      biomeOrDefault(args),
		rnd:        NewJavaRandom(args.Seed),
		seed:       args.Seed,
	}
	g.createHeightmap()
	g.createStrata()
	g.carveCaves()
	g.carveOreVeins(0.9, CoalOre)
	g.carveOreVeins(0.7, IronOre)
	g.carveOreVeins(0.5, GoldOre)
	g.floodFillWaterBorders()
	g.floodFillWater()
	g.floodFillLava()
	g.createSurfaceLayer()
	g.plantFlowers()
	g.plantMushrooms()
	g.plantTrees()
	g.placeSpawn()
	return nil
}

func (g *classicGen) createHeightmap() {
	n1 := newMCCombinedNoise(newMCOctaveNoise(8, g.rnd), newMCOctaveNoise(8, g.rnd))
	n2 := newMCCombinedNoise(newMCOctaveNoise(8, g.rnd), newMCOctaveNoise(8, g.rnd))
	n3 := newMCOctaveNoise(6, g.rnd)

	hMap := make([]int16, g.width*g.length)
	idx := 0
	for z := 0; z < g.length; z++ {
		for x := 0; x < g.width; x++ {
			nx := float64(float32(x) * 1.3)
			nz := float64(float32(z) * 1.3)
			hLow := n1.compute(nx, nz)/12 - 2
			height := hLow
			if n3.compute(float64(x), float64(z)) <= 0 {
				hHigh := n2.compute(nx, nz)/10 + 3
				height = math.Max(hLow, hHigh)
			}
			if height < 0 {
				height *= float64(float32(0.8))
			}
			adjHeight := int(height + float64(g.waterLevel))
			if adjHeight < g.minHeight {
				g.minHeight = adjHeight
			}
			hMap[idx] = int16(adjHeight)
			idx++
		}
	}
	g.heightmap = hMap
}

func (g *classicGen) createStrata() {
	n := newMCOctaveNoise(8, g.rnd)
	minStoneY := g.createStrataFast()
	ground := g.biome.Ground
	cliff := g.biome.Cliff

	hMapIndex := 0
	for z := 0; z < g.length; z++ {
		for x := 0; x < g.width; x++ {
			dirtThickness := int(n.compute(float64(x), float64(z))/24 - 4)
			dirtHeight := int(g.heightmap[hMapIndex])
			stoneHeight := dirtHeight + dirtThickness

			stoneHeight = Min(stoneHeight, g.height-1)
			dirtHeight = Min(dirtHeight, g.height-1)

			mapIndex := minStoneY*g.oneY + z*g.width + x
			for y := minStoneY; y <= stoneHeight; y++ {
				g.lvl.Blocks[mapIndex] = cliff
				mapIndex += g.oneY
			}

			stoneHeight = Max(stoneHeight, 0)
			mapIndex = (stoneHeight+1)*g.oneY + z*g.width + x
			for y := stoneHeight + 1; y <= dirtHeight; y++ {
				g.lvl.Blocks[mapIndex] = ground
				mapIndex += g.oneY
			}
			hMapIndex++
		}
	}
}

func (g *classicGen) createStrataFast() int {
	mapIndex := 0
	count := g.length * g.width
	for i := 0; i < count; i++ {
		g.lvl.Blocks[mapIndex] = Lava
		mapIndex++
	}

	stoneHeight := g.minHeight - 14
	if stoneHeight <= 0 {
		return 1
	}
	cliff := g.biome.Cliff
	count = stoneHeight * g.length * g.width
	for i := 0; i < count; i++ {
		g.lvl.Blocks[mapIndex] = cliff
		mapIndex++
	}
	return stoneHeight
}

func (g *classicGen) carveCaves() {
	cavesCount := len(g.lvl.Blocks) / 8192
	for i := 0; i < cavesCount; i++ {
		caveX := float64(g.rnd.NextInt(g.width))
		caveY := float64(g.rnd.NextInt(g.height))
		caveZ := float64(g.rnd.NextInt(g.length))
		caveLen := int(float64(g.rnd.NextFloat()) * float64(g.rnd.NextFloat()) * 200)
		theta := float64(g.rnd.NextFloat()) * 2 * math.Pi
		deltaTheta := 0.0
		phi := float64(g.rnd.NextFloat()) * 2 * math.Pi
		deltaPhi := 0.0
		caveRadius := float64(g.rnd.NextFloat()) * float64(g.rnd.NextFloat())

		for j := 0; j < caveLen; j++ {
			caveX += math.Sin(theta) * math.Cos(phi)
			caveZ += math.Cos(theta) * math.Cos(phi)
			caveY += math.Sin(phi)

			theta += deltaTheta * 0.2
			deltaTheta = deltaTheta*0.9 + float64(g.rnd.NextFloat()) - float64(g.rnd.NextFloat())
			phi = phi/2 + deltaPhi/4
			deltaPhi = deltaPhi*0.75 + float64(g.rnd.NextFloat()) - float64(g.rnd.NextFloat())
			if g.rnd.NextFloat() < 0.25 {
				continue
			}

			cenX := int(caveX + float64(g.rnd.NextInt(4)-2)*0.2)
			cenY := int(caveY + float64(g.rnd.NextInt(4)-2)*0.2)
			cenZ := int(caveZ + float64(g.rnd.NextInt(4)-2)*0.2)

			radius := (float64(g.height) - float64(cenY)) / float64(g.height)
			radius = 1.2 + (radius*3.5+1)*caveRadius
			radius *= math.Sin(float64(j) * math.Pi / float64(caveLen))
			g.fillOblateSpheroid(cenX, cenY, cenZ, float32(radius), Air)
		}
	}
}

func (g *classicGen) carveOreVeins(abundance float64, block byte) {
	numVeins := int(float64(len(g.lvl.Blocks)) * abundance / 16384)
	for i := 0; i < numVeins; i++ {
		veinX := float64(g.rnd.NextInt(g.width))
		veinY := float64(g.rnd.NextInt(g.height))
		veinZ := float64(g.rnd.NextInt(g.length))
		veinLen := int(float64(g.rnd.NextFloat()) * float64(g.rnd.NextFloat()) * 75.0 * abundance)
		theta := float64(g.rnd.NextFloat()) * 2 * math.Pi
		deltaTheta := 0.0
		phi := float64(g.rnd.NextFloat()) * 2 * math.Pi
		deltaPhi := 0.0

		for j := 0; j < veinLen; j++ {
			veinX += math.Sin(theta) * math.Cos(phi)
			veinZ += math.Cos(theta) * math.Cos(phi)
			veinY += math.Sin(phi)

			theta = deltaTheta * 0.2
			deltaTheta = deltaTheta*0.9 + float64(g.rnd.NextFloat()) - float64(g.rnd.NextFloat())
			phi = phi/2 + deltaPhi/4
			deltaPhi = deltaPhi*0.9 + float64(g.rnd.NextFloat()) - float64(g.rnd.NextFloat())

			radius := abundance*math.Sin(float64(j)*math.Pi/float64(veinLen)) + 1
			g.fillOblateSpheroid(int(veinX), int(veinY), int(veinZ), float32(radius), block)
		}
	}
}

func (g *classicGen) fillOblateSpheroid(x, y, z int, radius float32, block byte) {
	xBeg := Floor32(max32(float32(x)-radius, 0))
	xEnd := Floor32(min32(float32(x)+radius, float32(g.width-1)))
	yBeg := Floor32(max32(float32(y)-radius, 0))
	yEnd := Floor32(min32(float32(y)+radius, float32(g.height-1)))
	zBeg := Floor32(max32(float32(z)-radius, 0))
	zEnd := Floor32(min32(float32(z)+radius, float32(g.length-1)))
	radiusSq := radius * radius

	for yy := yBeg; yy <= yEnd; yy++ {
		for zz := zBeg; zz <= zEnd; zz++ {
			for xx := xBeg; xx <= xEnd; xx++ {
				dx := xx - x
				dy := yy - y
				dz := zz - z
				if float32(dx*dx+2*dy*dy+dz*dz) < radiusSq {
					idx := (yy*g.length+zz)*g.width + xx
					if g.lvl.Blocks[idx] == Stone {
						g.lvl.Blocks[idx] = block
					}
				}
			}
		}
	}
}

func (g *classicGen) floodFillWaterBorders() {
	waterY := g.waterLevel - 1
	if waterY < 0 || waterY >= g.height {
		return
	}
	water := g.biome.Water
	if water == Air {
		return
	}

	idx1 := (waterY*g.length+0)*g.width + 0
	idx2 := (waterY*g.length+(g.length-1))*g.width + 0
	for x := 0; x < g.width; x++ {
		g.floodFill(idx1, water)
		g.floodFill(idx2, water)
		idx1++
		idx2++
	}

	idx1 = (waterY*g.length+0)*g.width + 0
	idx2 = (waterY*g.length+0)*g.width + (g.width - 1)
	for z := 0; z < g.length; z++ {
		g.floodFill(idx1, water)
		g.floodFill(idx2, water)
		idx1 += g.width
		idx2 += g.width
	}
}

func (g *classicGen) floodFillWater() {
	numSources := g.width * g.length / 800
	water := g.biome.Water
	if water == Air {
		return
	}
	for i := 0; i < numSources; i++ {
		x := g.rnd.NextInt(g.width)
		z := g.rnd.NextInt(g.length)
		y := g.waterLevel - g.rnd.NextIntRange(1, 3)
		if y < 0 || y >= g.height {
			continue
		}
		g.floodFill((y*g.length+z)*g.width+x, water)
	}
}

func (g *classicGen) floodFillLava() {
	numSources := g.width * g.length / 20000
	for i := 0; i < numSources; i++ {
		x := g.rnd.NextInt(g.width)
		z := g.rnd.NextInt(g.length)
		y := int(float32(g.waterLevel-3) * g.rnd.NextFloat() * g.rnd.NextFloat())
		if y < 0 || y >= g.height {
			continue
		}
		g.floodFill((y*g.length+z)*g.width+x, StillLava)
	}
}

func (g *classicGen) floodFill(startIndex int, block byte) {
	if startIndex < 0 || startIndex >= len(g.lvl.Blocks) {
		return
	}
	stack := make([]int, 0, 1024)
	stack = append(stack, startIndex)
	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if g.lvl.Blocks[idx] != Air {
			continue
		}
		g.lvl.Blocks[idx] = block

		x := idx % g.width
		y := idx / g.oneY
		z := (idx / g.width) % g.length

		if x > 0 {
			stack = append(stack, idx-1)
		}
		if x < g.width-1 {
			stack = append(stack, idx+1)
		}
		if z > 0 {
			stack = append(stack, idx-g.width)
		}
		if z < g.length-1 {
			stack = append(stack, idx+g.width)
		}
		if y > 0 {
			stack = append(stack, idx-g.oneY)
		}
	}
}

func (g *classicGen) createSurfaceLayer() {
	n1 := newMCOctaveNoise(8, g.rnd)
	n2 := newMCOctaveNoise(8, g.rnd)
	surface := g.biome.Surface
	sandy := g.biome.BeachSandy
	rocky := g.biome.BeachRocky
	water := g.biome.Water

	hMapIndex := 0
	for z := 0; z < g.length; z++ {
		for x := 0; x < g.width; x++ {
			y := int(g.heightmap[hMapIndex])
			hMapIndex++
			if y < 0 || y >= g.height {
				continue
			}
			idx := (y*g.length+z)*g.width + x
			var blockAbove byte
			if y < g.height-1 {
				blockAbove = g.lvl.Blocks[idx+g.oneY]
			} else {
				blockAbove = Air
			}
			if blockAbove == water && n2.compute(float64(x), float64(z)) > 12 {
				g.lvl.Blocks[idx] = rocky
			} else if blockAbove == Air {
				if y <= g.waterLevel && n1.compute(float64(x), float64(z)) > 8 {
					g.lvl.Blocks[idx] = sandy
				} else {
					g.lvl.Blocks[idx] = surface
				}
			}
		}
	}
}

func (g *classicGen) plantFlowers() {
	numPatches := g.width * g.length / 3000
	surface := g.biome.Surface
	for i := 0; i < numPatches; i++ {
		flowerType := byte(Dandelion + g.rnd.NextInt(2))
		patchX := g.rnd.NextInt(g.width)
		patchZ := g.rnd.NextInt(g.length)
		for j := 0; j < 10; j++ {
			flowerX := patchX
			flowerZ := patchZ
			for k := 0; k < 5; k++ {
				flowerX += g.randomCenteredOffset()
				flowerZ += g.randomCenteredOffset()
				if flowerX < 0 || flowerZ < 0 || flowerX >= g.width || flowerZ >= g.length {
					continue
				}
				flowerY := int(g.heightmap[flowerZ*g.width+flowerX]) + 1
				if flowerY <= 0 || flowerY >= g.height {
					continue
				}
				idx := (flowerY*g.length+flowerZ)*g.width + flowerX
				if g.lvl.Blocks[idx] == Air && g.lvl.Blocks[idx-g.oneY] == surface {
					g.lvl.Blocks[idx] = flowerType
				}
			}
		}
	}
}

func (g *classicGen) plantMushrooms() {
	numPatches := len(g.lvl.Blocks) / 2000
	cliff := g.biome.Cliff
	for i := 0; i < numPatches; i++ {
		mushType := byte(BrownMushroom + g.rnd.NextInt(2))
		patchX := g.rnd.NextInt(g.width)
		patchY := g.rnd.NextInt(g.height)
		patchZ := g.rnd.NextInt(g.length)
		for j := 0; j < 20; j++ {
			mushX, mushY, mushZ := patchX, patchY, patchZ
			for k := 0; k < 5; k++ {
				mushX += g.randomCenteredOffset()
				mushZ += g.randomCenteredOffset()
				if mushX < 0 || mushZ < 0 || mushX >= g.width || mushZ >= g.length {
					continue
				}
				solidHeight := int(g.heightmap[mushZ*g.width+mushX])
				if mushY >= solidHeight-1 {
					continue
				}
				idx := (mushY*g.length+mushZ)*g.width + mushX
				if g.lvl.Blocks[idx] == Air && g.lvl.Blocks[idx-g.oneY] == cliff {
					g.lvl.Blocks[idx] = mushType
				}
			}
		}
	}
}

func (g *classicGen) plantTrees() {
	numPatches := g.width * g.length / 4000
	surface := g.biome.Surface
	treeType := g.biome.TreeDefault("Classic")
	ctor, ok := FindTree(treeType)
	if !ok {
		return
	}
	tree := ctor()
	if ct, ok := tree.(*ClassicTree); ok {
		ct.SetJavaRandom(g.rnd)
	}
	goRng := rand.New(rand.NewSource(int64(g.seed)))

	for i := 0; i < numPatches; i++ {
		g.plantTreePatch(surface, tree, goRng)
	}
}

// plantTreePatch attempts to grow a cluster of trees around one random point.
func (g *classicGen) plantTreePatch(surface byte, tree Tree, goRng *rand.Rand) {
	patchX := g.rnd.NextInt(g.width)
	patchZ := g.rnd.NextInt(g.length)
	for j := 0; j < 20; j++ {
		treeX := patchX
		treeZ := patchZ
		for k := 0; k < 20; k++ {
			treeX += g.randomCenteredOffset()
			treeZ += g.randomCenteredOffset()
			if !g.inBounds(treeX, treeZ) || g.rnd.NextFloat() >= 0.25 {
				continue
			}
			g.tryGrowTree(treeX, treeZ, surface, tree, goRng)
		}
	}
}

// tryGrowTree grows a single tree at a location if the surface and space allow.
func (g *classicGen) tryGrowTree(treeX, treeZ int, surface byte, tree Tree, goRng *rand.Rand) {
	treeY := int(g.heightmap[treeZ*g.width+treeX]) + 1
	if treeY >= g.height {
		return
	}
	treeHeight := tree.DefaultSize(goRng)
	idx := (treeY*g.length+treeZ)*g.width + treeX
	blockUnder := byte(Air)
	if treeY > 0 {
		blockUnder = g.lvl.Blocks[idx-g.oneY]
	}
	if blockUnder != surface {
		return
	}
	if !g.canGrowTree(treeX, treeY, treeZ, treeHeight) {
		return
	}
	tree.SetData(goRng, treeHeight)
	tree.Generate(treeX, treeY, treeZ, func(xT, yT, zT int, bT byte) {
		idx := (yT*g.length+zT)*g.width + xT
		if bT == Leaves && g.lvl.Blocks[idx] == Log {
			return
		}
		g.lvl.Blocks[idx] = bT
	})
}

func (g *classicGen) randomCenteredOffset() int {
	positive := g.rnd.NextInt(6)
	negative := g.rnd.NextInt(6)
	return positive - negative
}

// inBounds reports whether x and z lie within the horizontal level bounds.
func (g *classicGen) inBounds(x, z int) bool {
	return x >= 0 && z >= 0 && x < g.width && z < g.length
}

func (g *classicGen) canGrowTree(x, y, z, height int) bool {
	if y < 0 || y+height-1 >= g.height {
		return false
	}
	if x-2 < 0 || x+2 >= g.width || z-2 < 0 || z+2 >= g.length {
		return false
	}
	baseHeight := height - 4
	for yy := y; yy < y+baseHeight; yy++ {
		for zz := z - 1; zz <= z+1; zz++ {
			for xx := x - 1; xx <= x+1; xx++ {
				idx := (yy*g.length+zz)*g.width + xx
				if g.lvl.Blocks[idx] != Air {
					return false
				}
			}
		}
	}
	for yy := y + baseHeight; yy < y+height; yy++ {
		for zz := z - 2; zz <= z+2; zz++ {
			for xx := x - 2; xx <= x+2; xx++ {
				idx := (yy*g.length+zz)*g.width + xx
				if g.lvl.Blocks[idx] != Air {
					return false
				}
			}
		}
	}
	return true
}

func (g *classicGen) placeSpawn() {
	cx := g.width / 2
	cz := g.length / 2
	idx := cz*g.width + cx
	if idx < 0 || idx >= len(g.heightmap) {
		return
	}
	g.lvl.Spawn.X = cx
	g.lvl.Spawn.Z = cz
	g.lvl.Spawn.Y = int(g.heightmap[idx]) + 2
	if g.lvl.Spawn.Y < 1 || g.lvl.Spawn.Y >= g.height {
		g.lvl.Spawn.Y = g.height / 2
	}
	g.lvl.Spawn.Yaw = 0
	g.lvl.Spawn.Pitch = 0
}
