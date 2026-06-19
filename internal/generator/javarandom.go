package generator

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
