package generator

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
func NewClassicTree() *ClassicTree {
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
				if abs(dx) == 2 && abs(dz) == 2 {
					if t.javaRng != nil {
						if t.javaRng.NextFloat() >= 0.5 {
							setBlock(x+dx, yy, z+dz, Leaves)
						}
					} else if t.rng != nil && t.rng.Float32() >= 0.5 {
						setBlock(x+dx, yy, z+dz, Leaves)
					}
				} else {
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
				} else if yy == topStartY {
					if t.javaRng != nil {
						if t.javaRng.NextFloat() >= 0.5 {
							setBlock(x+dx, yy, z+dz, Leaves)
						}
					} else if t.rng != nil && t.rng.Float32() >= 0.5 {
						setBlock(x+dx, yy, z+dz, Leaves)
					}
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

// treeRegistry maps tree type names to constructors.
var treeRegistry = map[string]func() Tree{
	"Classic": func() Tree { return NewClassicTree() },
}

// FindTree returns a tree constructor by name.
func FindTree(name string) (func() Tree, bool) {
	t, ok := treeRegistry[name]
	return t, ok
}

// TreeNames returns all registered tree names.
func TreeNames() []string {
	names := make([]string, 0, len(treeRegistry))
	for name := range treeRegistry {
		names = append(names, name)
	}
	return names
}

// RegisterTree registers a tree constructor under a name.
func RegisterTree(name string, ctor func() Tree) {
	treeRegistry[name] = ctor
}
