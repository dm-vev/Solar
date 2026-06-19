// Package core provides shared types and utilities used by generator families.
package core

import (
	"errors"
	"fmt"
	"math"
)

const (
	// Classic block IDs from MCGalaxy
	Air           = 0
	Stone         = 1
	Grass         = 2
	Dirt          = 3
	Cobblestone   = 4
	WoodPlank     = 5
	Sapling       = 6
	Bedrock       = 7
	Water         = 8
	StillWater    = 9
	Lava          = 10
	StillLava     = 11
	Sand          = 12
	Gravel        = 13
	GoldOre       = 14
	IronOre       = 15
	CoalOre       = 16
	Log           = 17
	Leaves        = 18
	Sponge        = 19
	Glass         = 20
	RedWool       = 21
	OrangeWool    = 22
	YellowWool    = 23
	LimeWool      = 24
	GreenWool     = 25
	TealWool      = 26
	AquaWool      = 27
	BlueWool      = 28
	PurpleWool    = 29
	IndigoWool    = 30
	VioletWool    = 31
	MagentaWool   = 32
	PinkWool      = 33
	BlackWool     = 34
	WhiteWool     = 35
	Dandelion     = 37
	Rose          = 38
	BrownMushroom = 39
	RedMushroom   = 40
	GoldBlock     = 41
	IronBlock     = 42
	DoubleSlab    = 43
	Slab          = 44
	Brick         = 45
	TNT           = 46
	Bookshelf     = 47
	MossyStone    = 48
	Obsidian      = 49
)

// Level is a mutable world used by generators. It mirrors world.Level but
// allows direct block access without locking.
type Level struct {
	Name   string
	Width  int
	Height int
	Length int
	Blocks []byte
	Spawn  Spawn
}

// Spawn describes the default spawn point.
type Spawn struct {
	X     int
	Y     int
	Z     int
	Yaw   byte
	Pitch byte
}

// Volume returns the number of blocks in the level.
func (l Level) Volume() int {
	return l.Width * l.Height * l.Length
}

// PackIndex returns the flat index for (x, y, z).
func PackIndex(l Level, x, y, z int) int {
	return ((y*l.Length)+z)*l.Width + x
}

// SetBlock sets a block at the given coordinates.
func SetBlock(l *Level, x, y, z int, block byte) bool {
	if x < 0 || y < 0 || z < 0 || x >= l.Width || y >= l.Height || z >= l.Length {
		return false
	}
	l.Blocks[PackIndex(*l, x, y, z)] = block
	return true
}

// GetBlock returns a block at the given coordinates.
func GetBlock(l *Level, x, y, z int) byte {
	if x < 0 || y < 0 || z < 0 || x >= l.Width || y >= l.Height || z >= l.Length {
		return Air
	}
	return l.Blocks[PackIndex(*l, x, y, z)]
}

// NewLevel creates a level filled with air.
func NewLevel(name string, width, height, length int) *Level {
	return &Level{
		Name:   name,
		Width:  width,
		Height: height,
		Length: length,
		Blocks: make([]byte, width*height*length),
		Spawn:  Spawn{X: width / 2, Y: height * 3 / 4, Z: length / 2},
	}
}

// genMaxBlocks is the hard cap on generated world volume. Set once at
// startup via SetMaxBlocks; defaults to 64M blocks.
var genMaxBlocks int64 = 64 * 1024 * 1024

// SetMaxBlocks updates the generator volume limit. Must be called
// before any world is generated.
func SetMaxBlocks(n int) {
	if n > 0 {
		genMaxBlocks = int64(n)
	}
}

// ValidateDimensions checks if level dimensions are reasonable.
func ValidateDimensions(width, height, length int) error {
	if width < 1 || height < 1 || length < 1 {
		return errors.New("dimensions must be positive")
	}
	volume := int64(width) * int64(height) * int64(length)
	if volume > genMaxBlocks {
		return fmt.Errorf("volume %d exceeds limit %d", volume, genMaxBlocks)
	}
	return nil
}

// FillCuboid fills a rectangular region with a block.
func FillCuboid(l *Level, x1, y1, z1, x2, y2, z2 int, block byte) {
	minX, maxX := minMax(x1, x2, l.Width-1)
	minY, maxY := minMax(y1, y2, l.Height-1)
	minZ, maxZ := minMax(z1, z2, l.Length-1)
	for y := minY; y <= maxY; y++ {
		for z := minZ; z <= maxZ; z++ {
			for x := minX; x <= maxX; x++ {
				l.Blocks[PackIndex(*l, x, y, z)] = block
			}
		}
	}
}

// FillCuboidFn fills a region with blocks returned by a callback.
func FillCuboidFn(l *Level, x1, y1, z1, x2, y2, z2 int, nextBlock func() byte) {
	minX, maxX := minMax(x1, x2, l.Width-1)
	minY, maxY := minMax(y1, y2, l.Height-1)
	minZ, maxZ := minMax(z1, z2, l.Length-1)
	for y := minY; y <= maxY; y++ {
		for z := minZ; z <= maxZ; z++ {
			for x := minX; x <= maxX; x++ {
				l.Blocks[PackIndex(*l, x, y, z)] = nextBlock()
			}
		}
	}
}

func minMax(a, b, limit int) (int, int) {
	if a > b {
		a, b = b, a
	}
	if a < 0 {
		a = 0
	}
	if b > limit {
		b = limit
	}
	return a, b
}

// Clamp restricts a value to a range.
func Clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Floor mirrors C# Math.Floor for integers.
func Floor(value float64) int {
	i := int(value)
	if value < 0 && float64(i) != value {
		i--
	}
	return i
}

// Floor32 mirrors MCGalaxy's ClassicGenerator.Floor(float).
func Floor32(value float32) int {
	i := int(value)
	if value < float32(i) {
		i--
	}
	return i
}

// Round mirrors C# Math.Round (half to even, but simplified).
func Round(value float64) int {
	if value < 0 {
		return -Floor(-value + 0.5)
	}
	return Floor(value + 0.5)
}

// Max returns the larger of two ints.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Min returns the smaller of two ints.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MaxF returns the larger of two floats.
func MaxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// MinF returns the smaller of two floats.
func MinF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Sqr returns x squared.
func Sqr(x float64) float64 { return x * x }

// Dist3 returns the Euclidean distance between two points.
func Dist3(x1, y1, z1, x2, y2, z2 float64) float64 {
	return math.Sqrt(Sqr(x1-x2) + Sqr(y1-y2) + Sqr(z1-z2))
}

// FillLayer fills an entire horizontal layer with a single block.
func FillLayer(l *Level, y int, block byte) {
	if y < 0 || y >= l.Height {
		return
	}
	for z := 0; z < l.Length; z++ {
		for x := 0; x < l.Width; x++ {
			l.Blocks[PackIndex(*l, x, y, z)] = block
		}
	}
}

func BiomeOrDefault(a *Args) Biome {
	if a.Biome.Name == "" {
		return Forest
	}
	return a.Biome
}
