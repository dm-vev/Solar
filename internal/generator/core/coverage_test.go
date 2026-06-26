package core

import (
	"math"
	"math/rand"
	"slices"
	"testing"
)

func TestLevelBlockAndMathHelpers(t *testing.T) {
	lvl := NewLevel("helpers", 3, 3, 3)

	if got := PackIndex(*lvl, 1, 1, 1); got != 13 {
		t.Fatalf("PackIndex = %d, want 13", got)
	}
	if !SetBlock(lvl, 1, 1, 1, Stone) {
		t.Fatal("SetBlock returned false for in-bounds coordinates")
	}
	if SetBlock(lvl, -1, 0, 0, Grass) {
		t.Fatal("SetBlock returned true for out-of-bounds coordinates")
	}
	if got := GetBlock(lvl, 1, 1, 1); got != Stone {
		t.Fatalf("GetBlock = %d, want %d", got, Stone)
	}
	if got := GetBlock(lvl, 99, 0, 0); got != Air {
		t.Fatalf("out-of-bounds GetBlock = %d, want air", got)
	}

	FillCuboid(lvl, 2, 2, 2, 1, 1, 1, Dirt)
	if got := GetBlock(lvl, 1, 1, 1); got != Dirt {
		t.Fatalf("FillCuboid did not fill reversed bounds, got %d", got)
	}

	next := byte(WoodPlank)
	FillCuboidFn(lvl, -5, 0, -5, 0, 0, 0, func() byte {
		next++
		return next
	})
	if got := GetBlock(lvl, 0, 0, 0); got != WoodPlank+1 {
		t.Fatalf("FillCuboidFn = %d, want %d", got, WoodPlank+1)
	}

	FillLayer(lvl, 2, Glass)
	if got := GetBlock(lvl, 0, 2, 0); got != Glass {
		t.Fatalf("FillLayer = %d, want glass", got)
	}
	FillLayer(lvl, 99, GoldBlock)
	if got := GetBlock(lvl, 0, 2, 0); got != Glass {
		t.Fatalf("out-of-bounds FillLayer changed block to %d", got)
	}

	if got := Clamp(15, 0, 10); got != 10 {
		t.Fatalf("Clamp high = %d, want 10", got)
	}
	if got := Floor(-1.2); got != -2 {
		t.Fatalf("Floor = %d, want -2", got)
	}
	if got := Floor32(float32(-1.2)); got != -2 {
		t.Fatalf("Floor32 = %d, want -2", got)
	}
	if got := Round(-1.6); got != -2 {
		t.Fatalf("Round = %d, want -2", got)
	}
	if Max(1, 2) != 2 || Min(1, 2) != 1 || MaxF(1, 2) != 2 || MinF(1, 2) != 1 {
		t.Fatal("min/max helpers returned unexpected values")
	}
	if Sqr(3) != 9 {
		t.Fatal("Sqr(3) != 9")
	}
	if got := Dist3(0, 0, 0, 1, 2, 2); got != 3 {
		t.Fatalf("Dist3 = %f, want 3", got)
	}
}

func TestBiomeDefaultsAndDimensionLimit(t *testing.T) {
	t.Cleanup(func() { SetMaxBlocks(64 * 1024 * 1024) })

	if got := Forest.TreeDefault("fallback"); got != "Classic" {
		t.Fatalf("Forest tree default = %q", got)
	}
	if got := Arctic.TreeDefault("fallback"); got != "fallback" {
		t.Fatalf("Arctic tree default = %q", got)
	}
	if got := BiomeOrDefault(&Args{}); got.Name != Forest.Name {
		t.Fatalf("BiomeOrDefault empty = %q", got.Name)
	}
	if got := BiomeOrDefault(&Args{Biome: Desert}); got.Name != Desert.Name {
		t.Fatalf("BiomeOrDefault desert = %q", got.Name)
	}

	SetMaxBlocks(8)
	if err := ValidateDimensions(2, 2, 2); err != nil {
		t.Fatalf("ValidateDimensions at limit: %v", err)
	}
	if err := ValidateDimensions(3, 3, 3); err == nil {
		t.Fatal("ValidateDimensions accepted oversized generator level")
	}
	SetMaxBlocks(-1)
	if err := ValidateDimensions(3, 3, 3); err == nil {
		t.Fatal("negative SetMaxBlocks changed the configured limit")
	}
}

func TestNoiseHelpers(t *testing.T) {
	perm := makePermutation(123)
	if len(perm) != 512 {
		t.Fatalf("permutation length = %d, want 512", len(perm))
	}
	if perm[0] != perm[256] {
		t.Fatal("permutation was not duplicated into second half")
	}
	if fade(0) != 0 || fade(1) != 1 {
		t.Fatal("fade endpoints are wrong")
	}
	if lerp(0.25, 10, 14) != 11 {
		t.Fatal("lerp returned unexpected value")
	}
	if grad(0, 2, 3) != 3 || grad(3, 2, 3) != -2 {
		t.Fatal("grad returned unexpected value")
	}
	if got := perlinNoise(perm, 1.25, 2.5); got < -2 || got > 2 {
		t.Fatalf("perlinNoise out of loose range: %f", got)
	}

	values := []float64{2, 4, 6}
	NormalizeFloat(values)
	if values[0] != 0 || values[1] != 0.5 || values[2] != 1 {
		t.Fatalf("NormalizeFloat = %v", values)
	}
	constValues := []float64{3, 3}
	NormalizeFloat(constValues)
	if constValues[0] != 3 || constValues[1] != 3 {
		t.Fatalf("NormalizeFloat changed flat input: %v", constValues)
	}
	Marble(values)
	if values[0] != 1 || values[1] != 0 || values[2] != 1 {
		t.Fatalf("Marble = %v", values)
	}
	Invert(values)
	if values[0] != 0 || values[1] != 1 || values[2] != 0 {
		t.Fatalf("Invert = %v", values)
	}

	bias := make([]float64, 4)
	ApplyBias(bias, 2, 2, 1, 2, 3, 4, 1)
	if bias[0] != 1 || bias[1] != 2 || bias[2] != 3 || bias[3] != 4 {
		t.Fatalf("ApplyBias corners = %v", bias)
	}

	dst := make([]float64, 4)
	PerlinNoise(12, dst, 2, 2, 1, 1, 0.5)
	if dst[0] != NewNoise(12, 1, 1, 0.5).At(0, 0) {
		t.Fatalf("PerlinNoise first value = %f", dst[0])
	}
	if NewNoise(12, 0, 0, 0).At(1, 1) != NewNoise(12, 0, 0, 0).At(1, 1) {
		t.Fatal("Noise.At is not deterministic")
	}

	slope := CalculateSlope([]float64{0, 3, 4, 8}, 2, 2)
	if slope[0] != 0 || slope[1] != 3 || slope[2] != 4 || slope[3] != math.Sqrt(5*5+4*4) {
		t.Fatalf("CalculateSlope = %v", slope)
	}
	blurred := GaussianBlur5X5([]float64{
		0, 0, 0,
		0, 256, 0,
		0, 0, 0,
	}, 3, 3)
	if blurred[4] <= blurred[0] {
		t.Fatalf("GaussianBlur5X5 center = %f, corner = %f", blurred[4], blurred[0])
	}
	if got := FindThresholdSorted([]float64{}, 0.5); got != 0 {
		t.Fatalf("FindThresholdSorted empty = %f", got)
	}
	sorted := SortCopy([]float64{3, 1, 2})
	if !slices.Equal(sorted, []float64{1, 2, 3}) {
		t.Fatalf("SortCopy = %v", sorted)
	}
	threshold := FindThresholdSorted(sorted, 0.5)
	if threshold != 2 {
		t.Fatalf("FindThresholdSorted = %f, want 2", threshold)
	}
	NormalizeToRange(sorted, 10, 20)
	if sorted[0] != 10 || sorted[1] != 15 || sorted[2] != 20 {
		t.Fatalf("NormalizeToRange = %v", sorted)
	}
	flat := []float64{7, 7}
	NormalizeToRange(flat, 0, 1)
	if flat[0] != 7 || flat[1] != 7 {
		t.Fatalf("NormalizeToRange changed flat input: %v", flat)
	}
}

func TestClassicTreeAndJavaRandom(t *testing.T) {
	tree := NewClassicTree()
	rng := rand.New(rand.NewSource(1))
	height := tree.DefaultSize(rng)
	if height < 5 || height > 7 {
		t.Fatalf("DefaultSize = %d, want 5..7", height)
	}
	tree.SetData(rng, 6)
	if tree.Height() != 6 {
		t.Fatalf("Height = %d, want 6", tree.Height())
	}

	placed := map[[4]int]bool{}
	tree.Generate(5, 5, 5, func(x, y, z int, block byte) {
		placed[[4]int{x, y, z, int(block)}] = true
	})
	if !placed[[4]int{5, 5, 5, Log}] {
		t.Fatal("tree did not place trunk at base")
	}
	if !placed[[4]int{5, 10, 5, Leaves}] {
		t.Fatal("tree did not place top leaves")
	}

	classic, ok := tree.(*ClassicTree)
	if !ok {
		t.Fatal("NewClassicTree did not return *ClassicTree")
	}
	javaRNG := NewJavaRandom(123)
	classic.SetJavaRandom(javaRNG)
	if got := classic.DefaultSize(rng); got < 5 || got > 7 {
		t.Fatalf("java DefaultSize = %d, want 5..7", got)
	}
	if !classic.chanceCornerLeaf() && !classic.chanceCornerLeaf() {
		t.Log("corner leaf chance produced two false values; acceptable deterministic path")
	}

	if abs(-3) != 3 || abs(3) != 3 {
		t.Fatal("abs returned unexpected value")
	}
	if ctor, ok := FindTree("Classic"); !ok || ctor == nil {
		t.Fatal("FindTree did not find Classic")
	}
	RegisterTree("UnitTree", NewClassicTree)
	if names := TreeNames(); !slices.Contains(names, "UnitTree") {
		t.Fatalf("TreeNames missing UnitTree: %v", names)
	}

	jr := NewJavaRandom(1)
	if got := jr.NextInt(0); got != 0 {
		t.Fatalf("NextInt(0) = %d, want 0", got)
	}
	if got := jr.NextInt(4); got < 0 || got >= 4 {
		t.Fatalf("NextInt power-of-two = %d", got)
	}
	if got := jr.NextInt(5); got < 0 || got >= 5 {
		t.Fatalf("NextInt = %d", got)
	}
	if got := jr.NextIntRange(10, 20); got < 10 || got >= 20 {
		t.Fatalf("NextIntRange = %d", got)
	}
	if got := jr.Next(20, 30); got < 20 || got >= 30 {
		t.Fatalf("Next = %d", got)
	}
	if got := jr.NextFloat(); got < 0 || got >= 1 {
		t.Fatalf("NextFloat = %f", got)
	}
	if got := jr.NextDouble(); got < 0 || got >= 1 {
		t.Fatalf("NextDouble = %f", got)
	}
	jr.SetSeed(1)
	if jr.NextInt(1000) != NewJavaRandom(1).NextInt(1000) {
		t.Fatal("SetSeed did not reset JavaRandom sequence")
	}
}

func TestRegistryNamesAndGenerate(t *testing.T) {
	registry := NewRegistry()
	registry.Register(Generator{})
	registry.Register(Generator{Name: "Unit", Type: GenTypeSimple, Func: setUnitBlock})

	names := registry.Names()
	if !slices.Contains(names[GenTypeSimple], "Unit") {
		t.Fatalf("Names missing Unit: %v", names)
	}
	gen, ok := registry.Find("unit")
	if !ok {
		t.Fatal("Find did not match case-insensitively")
	}
	lvl, err := Generate(gen, "generated", 2, 2, 2, Args{})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got := GetBlock(lvl, 0, 0, 0); got != GoldBlock {
		t.Fatalf("generated block = %d, want gold", got)
	}
}

func setUnitBlock(args *Args, lvl *Level) error {
	SetBlock(lvl, 0, 0, 0, GoldBlock)
	return nil
}
