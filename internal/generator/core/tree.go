package core

import (
	"math/rand"
)

// Tree is a procedural tree generator.
type Tree interface {
	DefaultSize(rng *rand.Rand) int
	SetData(rng *rand.Rand, height int)
	Height() int
	Generate(x, y, z int, setBlock func(x, y, z int, block byte))
}

// ClassicTree reproduces the original Minecraft Classic tree shape.
// It matches MCGalaxy's Generator/Foliage/ClassicTree.cs.
type ClassicTree struct {
	height  int
	size    int
	rng     *rand.Rand
	javaRng *JavaRandom
}

// NewClassicTree creates a classic tree generator.
func NewClassicTree() Tree {
	return &ClassicTree{rng: rand.New(rand.NewSource(0))}
}

// SetJavaRandom configures the tree to use the given JavaRandom for leaf randomness.
// When set, this matches MCGalaxy's behaviour of sharing the generator's JavaRandom.
func (t *ClassicTree) SetJavaRandom(r *JavaRandom) {
	t.javaRng = r
}

// DefaultSize returns a random height (5 to 7).
func (t *ClassicTree) DefaultSize(rng *rand.Rand) int {
	if t.javaRng != nil {
		return 5 + t.javaRng.NextInt(3)
	}
	return 5 + rng.Intn(3)
}

// SetData configures the tree.
func (t *ClassicTree) SetData(rng *rand.Rand, height int) {
	t.height = height
	t.size = 2
	t.rng = rng
}

// Height returns the configured tree height.
func (t *ClassicTree) Height() int { return t.height }

// Generate builds a classic tree at (x, y, z).
func (t *ClassicTree) Generate(x, y, z int, setBlock func(x, y, z int, block byte)) {
	h := t.height
	baseHeight := h - 4
	topStartY := y + baseHeight + 2

	// trunk
	for dy := 0; dy < h-1; dy++ {
		setBlock(x, y+dy, z, Log)
	}

	// leaves bottom layer
	for yy := y + baseHeight; yy < topStartY; yy++ {
		for dz := -2; dz <= 2; dz++ {
			for dx := -2; dx <= 2; dx++ {
				corner := abs(dx) == 2 && abs(dz) == 2
				if !corner || t.chanceCornerLeaf() {
					setBlock(x+dx, yy, z+dz, Leaves)
				}
			}
		}
	}

	// leaves top layer
	for yy := topStartY; yy < y+h; yy++ {
		for dz := -1; dz <= 1; dz++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 || dz == 0 {
					setBlock(x+dx, yy, z+dz, Leaves)
					continue
				}
				if yy == topStartY && t.chanceCornerLeaf() {
					setBlock(x+dx, yy, z+dz, Leaves)
				}
			}
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// chanceCornerLeaf returns true if a diagonal leaf block should be placed.
// The original generator uses a 50% chance for corner leaves.
func (t *ClassicTree) chanceCornerLeaf() bool {
	if t.javaRng != nil {
		return t.javaRng.NextFloat() >= 0.5
	}
	return t.rng != nil && t.rng.Float32() >= 0.5
}

// defaultTrees maps tree type names to constructors. It is populated at
// package load time and may be extended via RegisterTree.
var defaultTrees = map[string]func() Tree{
	"Classic": NewClassicTree,
}

// FindTree returns a tree constructor by name.
func FindTree(name string) (func() Tree, bool) {
	t, ok := defaultTrees[name]
	return t, ok
}

// TreeNames returns all registered tree names.
func TreeNames() []string {
	names := make([]string, 0, len(defaultTrees))
	for name := range defaultTrees {
		names = append(names, name)
	}
	return names
}

// RegisterTree registers a tree constructor under a name in the default
// tree registry.
func RegisterTree(name string, ctor func() Tree) {
	defaultTrees[name] = ctor
}

// JavaRandom implements java.util.Random linear congruential generator. It is
// used to reproduce MCGalaxy's Classic generator output exactly.
type JavaRandom struct {
	seed int64
}

const (
	javaMultiplier = 0x5DEECE66D
	javaAddend     = 0xB
	javaMask       = (1 << 48) - 1
)

// NewJavaRandom creates a JavaRandom seeded with the given value.
func NewJavaRandom(seed int) *JavaRandom {
	r := &JavaRandom{}
	r.SetSeed(seed)
	return r
}

// SetSeed reseeds the generator.
func (r *JavaRandom) SetSeed(seed int) {
	r.seed = (int64(seed) ^ javaMultiplier) & javaMask
}

// NextInt returns a non-negative int in [0, n). If n <= 0 it returns 0
// instead of panicking; callers must ensure n is positive.
func (r *JavaRandom) NextInt(n int) int {
	if n <= 0 {
		return 0
	}
	if (n & -n) == n { // power of two
		r.seed = (r.seed*javaMultiplier + javaAddend) & javaMask
		raw := int32(uint64(r.seed) >> (48 - 31))
		return int((int64(n) * int64(raw)) >> 31)
	}
	var bits, val int32
	for {
		r.seed = (r.seed*javaMultiplier + javaAddend) & javaMask
		bits = int32(uint64(r.seed) >> (48 - 31))
		val = bits % int32(n)
		if bits-val+int32(n-1) >= 0 {
			break
		}
	}
	return int(val)
}

// NextIntRange returns an int in [min, max).
func (r *JavaRandom) NextIntRange(min, max int) int {
	return min + r.NextInt(max-min)
}

// Next returns an int in [min, max). Alias matching MCGalaxy JavaRandom.Next.
func (r *JavaRandom) Next(min, max int) int {
	return min + r.NextInt(max-min)
}

// NextFloat returns a float in [0, 1).
func (r *JavaRandom) NextFloat() float32 {
	r.seed = (r.seed*javaMultiplier + javaAddend) & javaMask
	raw := int32(uint64(r.seed) >> (48 - 24))
	return float32(raw) / float32(1<<24)
}

// NextDouble returns a float64 in [0, 1).
func (r *JavaRandom) NextDouble() float64 {
	return float64(r.NextFloat())
}
