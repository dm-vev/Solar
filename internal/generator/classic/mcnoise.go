package classic

import "github.com/solar-mc/solar/internal/generator/core"

// mcNoise.go contains the exact MCGalaxy Classic noise implementations.
// Based on MCGalaxy/Generator/Classic/Noise.cs and
// http://mrl.nyu.edu/~perlin/noise/

// mcImprovedNoise implements MCGalaxy's ImprovedNoise for the Classic generator.
type mcImprovedNoise struct {
	p [512]byte
}

// newMCImprovedNoise creates a noise instance using MCGalaxy's permutation.
func newMCImprovedNoise(rnd *core.JavaRandom) *mcImprovedNoise {
	n := &mcImprovedNoise{}
	for i := 0; i < 256; i++ {
		n.p[i] = byte(i)
	}
	for i := 0; i < 256; i++ {
		j := rnd.Next(i, 256)
		n.p[i], n.p[j] = n.p[j], n.p[i]
	}
	for i := 0; i < 256; i++ {
		n.p[i+256] = n.p[i]
	}
	return n
}

// compute evaluates the noise at (x, z). In MCGalaxy y is always 0.
func (n *mcImprovedNoise) compute(x, y float64) float64 {
	xFloor := mcNoiseFloor(x)
	yFloor := mcNoiseFloor(y)
	X := xFloor & 0xFF
	Y := yFloor & 0xFF
	x -= float64(xFloor)
	y -= float64(yFloor)

	u := x * x * x * (x*(x*6-15) + 10) // Fade(x)
	v := y * y * y * (y*(y*6-15) + 10) // Fade(y)
	A := int(n.p[X]) + Y
	B := int(n.p[X+1]) + Y

	const xFlags int = 0x46552222
	const yFlags int = 0x2222550A

	hash := (int(n.p[n.p[A]]) & 0xF) << 1
	g22 := float64(((xFlags>>hash)&3)-1)*x + float64(((yFlags>>hash)&3)-1)*y

	hash = (int(n.p[n.p[B]]) & 0xF) << 1
	g12 := float64(((xFlags>>hash)&3)-1)*(x-1) + float64(((yFlags>>hash)&3)-1)*y
	c1 := g22 + u*(g12-g22)

	hash = (int(n.p[n.p[A+1]]) & 0xF) << 1
	g21 := float64(((xFlags>>hash)&3)-1)*x + float64(((yFlags>>hash)&3)-1)*(y-1)

	hash = (int(n.p[n.p[B+1]]) & 0xF) << 1
	g11 := float64(((xFlags>>hash)&3)-1)*(x-1) + float64(((yFlags>>hash)&3)-1)*(y-1)
	c2 := g21 + u*(g11-g21)

	return c1 + v*(c2-c1)
}

func mcNoiseFloor(value float64) int {
	if value >= 0 {
		return int(value)
	}
	return int(value) - 1
}

// mcOctaveNoise implements MCGalaxy's OctaveNoise.
type mcOctaveNoise struct {
	baseNoise []*mcImprovedNoise
}

// newMCOctaveNoise creates an octave noise with the given number of octaves.
func newMCOctaveNoise(octaves int, rnd *core.JavaRandom) *mcOctaveNoise {
	n := &mcOctaveNoise{baseNoise: make([]*mcImprovedNoise, octaves)}
	for i := 0; i < octaves; i++ {
		n.baseNoise[i] = newMCImprovedNoise(rnd)
	}
	return n
}

// compute sums multiple octaves with amplitude doubling and frequency halving.
func (n *mcOctaveNoise) compute(x, y float64) float64 {
	amplitude := 1.0
	frequency := 1.0
	sum := 0.0
	for i := 0; i < len(n.baseNoise); i++ {
		sum += n.baseNoise[i].compute(x*frequency, y*frequency) * amplitude
		amplitude *= 2.0
		frequency *= 0.5
	}
	return sum
}

// mcCombinedNoise implements MCGalaxy's CombinedNoise.
type mcCombinedNoise struct {
	noise1, noise2 *mcOctaveNoise
}

// newMCCombinedNoise creates a combined noise.
func newMCCombinedNoise(noise1, noise2 *mcOctaveNoise) *mcCombinedNoise {
	return &mcCombinedNoise{noise1: noise1, noise2: noise2}
}

// compute returns noise1(x + noise2(x, y), y).
func (c *mcCombinedNoise) compute(x, y float64) float64 {
	offset := c.noise2.compute(x, y)
	return c.noise1.compute(x+offset, y)
}
