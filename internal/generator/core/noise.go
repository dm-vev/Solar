package core

import (
	"math"
	"math/rand"
)

// Noise implements a deterministic 2D Perlin-like noise generator.
func makePermutation(seed int) []int {
	rng := rand.New(rand.NewSource(int64(seed)))
	p := make([]int, 512)
	for i := 0; i < 256; i++ {
		p[i] = i
	}
	for i := 255; i > 0; i-- {
		j := rng.Intn(i + 1)
		p[i], p[j] = p[j], p[i]
	}
	for i := 0; i < 256; i++ {
		p[i+256] = p[i]
	}
	return p
}

func fade(t float64) float64       { return t * t * t * (t*(t*6-15) + 10) }
func lerp(t, a, b float64) float64 { return a + t*(b-a) }
func grad(hash int, x, y float64) float64 {
	h := hash & 3
	u := float64(h & 1)
	if h&2 != 0 {
		u = -u
	}
	v := float64((h + 1) & 1)
	if h&2 != 0 {
		v = -v
	}
	return u*x + v*y
}

func perlinNoise(p []int, x, y float64) float64 {
	X := int(math.Floor(x)) & 255
	Y := int(math.Floor(y)) & 255
	x -= math.Floor(x)
	y -= math.Floor(y)
	u := fade(x)
	v := fade(y)

	A := p[X] + Y
	AA := p[A]
	AB := p[A+1]
	B := p[X+1] + Y
	BA := p[B]
	BB := p[B+1]

	return lerp(v,
		lerp(u, grad(p[AA], x, y), grad(p[BA], x-1, y)),
		lerp(u, grad(p[AB], x, y-1), grad(p[BB], x-1, y-1)))
}

// NormalizeFloat rescales a slice of floats to [0, 1].
func NormalizeFloat(values []float64) {
	min, max := values[0], values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	delta := max - min
	if delta == 0 {
		return
	}
	for i := range values {
		values[i] = (values[i] - min) / delta
	}
}

// Marble applies a marble-like transformation to a heightmap.
func Marble(values []float64) {
	for i := range values {
		values[i] = math.Abs(values[i]*2 - 1)
	}
}

// Invert inverts a heightmap.
func Invert(values []float64) {
	for i := range values {
		values[i] = 1 - values[i]
	}
}

// ApplyBias adds corner and midpoint bias to a heightmap.
func ApplyBias(values []float64, width, length int, c1, c2, c3, c4, midpoint float64) {
	for z := 0; z < length; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			fx := float64(x) / float64(width-1)
			fz := float64(z) / float64(length-1)
			// bilinear corner interpolation
			corner := c1*(1-fx)*(1-fz) + c2*fx*(1-fz) + c3*(1-fx)*fz + c4*fx*fz
			mid := midpoint * (1 - math.Abs(fx-0.5)*2) * (1 - math.Abs(fz-0.5)*2)
			values[i] += corner + mid
		}
	}
}

// PerlinNoise fills a 2D float slice with Perlin noise.
func PerlinNoise(seed int, dst []float64, width, length int, featureScale, detailScale int, roughness float64) {
	noise := NewNoise(seed, featureScale, detailScale, roughness)
	for z := 0; z < length; z++ {
		for x := 0; x < width; x++ {
			dst[z*width+x] = noise.At(x, z)
		}
	}
}

// Noise is a configurable 2D noise generator.
type Noise struct {
	seed         int
	featureScale int
	detailScale  int
	roughness    float64
	permutations []int
}

// NewNoise creates a noise generator.
func NewNoise(seed int, featureScale, detailScale int, roughness float64) *Noise {
	return &Noise{
		seed:         seed,
		featureScale: featureScale,
		detailScale:  detailScale,
		roughness:    roughness,
		permutations: makePermutation(seed),
	}
}

// At returns noise at (x, z).
func (n *Noise) At(x, z int) float64 {
	scaleX := 1.0 / math.Max(1, float64(n.featureScale))
	scaleZ := 1.0 / math.Max(1, float64(n.featureScale))
	return perlinNoise(n.permutations, float64(x)*scaleX, float64(z)*scaleZ)
}

// CalculateSlope computes a slope map from a heightmap.
func CalculateSlope(heightmap []float64, width, length int) []float64 {
	slope := make([]float64, width*length)
	for z := 0; z < length; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			var dx, dz float64
			if x > 0 {
				dx = math.Abs(heightmap[i] - heightmap[i-1])
			}
			if z > 0 {
				dz = math.Abs(heightmap[i] - heightmap[i-width])
			}
			slope[i] = math.Sqrt(dx*dx + dz*dz)
		}
	}
	return slope
}

// GaussianBlur5X5 applies a 5x5 Gaussian blur to a heightmap.
func GaussianBlur5X5(src []float64, width, length int) []float64 {
	kernel := []float64{
		1, 4, 6, 4, 1,
		4, 16, 24, 16, 4,
		6, 24, 36, 24, 6,
		4, 16, 24, 16, 4,
		1, 4, 6, 4, 1,
	}
	const sum = 256.0

	dst := make([]float64, width*length)
	for z := 0; z < length; z++ {
		for x := 0; x < width; x++ {
			i := z*width + x
			var total float64
			for ky := 0; ky < 5; ky++ {
				for kx := 0; kx < 5; kx++ {
					sx := Clamp(x+kx-2, 0, width-1)
					sz := Clamp(z+ky-2, 0, length-1)
					total += src[sz*width+sx] * kernel[ky*5+kx]
				}
			}
			dst[i] = total / sum
		}
	}
	return dst
}

// FindThresholdSorted returns the threshold from a sorted copy.
func FindThresholdSorted(values []float64, fraction float64) float64 {
	if len(values) == 0 {
		return 0
	}
	idx := int(float64(len(values)-1) * fraction)
	if idx < 0 {
		idx = 0
	}
	return values[idx]
}

// SortCopy returns a sorted copy of the values.
func SortCopy(values []float64) []float64 {
	copyVals := append([]float64(nil), values...)
	// Simple insertion sort for deterministic behavior.
	for i := 1; i < len(copyVals); i++ {
		key := copyVals[i]
		j := i - 1
		for j >= 0 && copyVals[j] > key {
			copyVals[j+1] = copyVals[j]
			j--
		}
		copyVals[j+1] = key
	}
	return copyVals
}

// NormalizeToRange rescales values to [min, max].
func NormalizeToRange(values []float64, min, max float64) {
	lo, hi := values[0], values[0]
	for _, v := range values {
		if v < lo {
			lo = v
		}
		if v > hi {
			hi = v
		}
	}
	delta := hi - lo
	if delta == 0 {
		return
	}
	for i := range values {
		values[i] = min + (values[i]-lo)/delta*(max-min)
	}
}
