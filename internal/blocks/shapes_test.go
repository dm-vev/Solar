package blocks

import (
	"testing"
)

func TestCuboid(t *testing.T) {
	var count int
	Cuboid(Vec3{0, 0, 0}, Vec3{2, 2, 2}, func(x, y, z int) { count++ })
	if count != 27 { // 3×3×3
		t.Fatalf("Cuboid count = %d, want 27", count)
	}
}

func TestCuboidHollow(t *testing.T) {
	var count int
	CuboidHollow(Vec3{0, 0, 0}, Vec3{2, 2, 2}, func(x, y, z int) { count++ })
	// 3×3×3 hollow = 27 - 1 (inner) = 26
	if count != 26 {
		t.Fatalf("CuboidHollow count = %d, want 26", count)
	}
}

func TestCuboidWalls(t *testing.T) {
	placed := map[Vec3]bool{}
	CuboidWalls(Vec3{0, 0, 0}, Vec3{2, 2, 2}, func(x, y, z int) {
		placed[Vec3{x, y, z}] = true
	})
	if len(placed) != 24 {
		t.Fatalf("CuboidWalls count = %d, want 24", len(placed))
	}
	if !placed[Vec3{1, 1, 0}] || !placed[Vec3{0, 1, 1}] {
		t.Fatalf("CuboidWalls missed vertical face blocks: %v", placed)
	}
	if placed[Vec3{1, 1, 1}] || placed[Vec3{1, 0, 1}] || placed[Vec3{1, 2, 1}] {
		t.Fatalf("CuboidWalls placed interior or floor/ceiling blocks: %v", placed)
	}
}

func TestLine(t *testing.T) {
	var points []Vec3
	Line(Vec3{0, 0, 0}, Vec3{5, 0, 0}, func(x, y, z int) {
		points = append(points, Vec3{x, y, z})
	})
	if len(points) != 6 { // 0..5 inclusive
		t.Fatalf("Line count = %d, want 6", len(points))
	}
	for i, p := range points {
		if p.X != i || p.Y != 0 || p.Z != 0 {
			t.Fatalf("Line point %d = %v, want (%d,0,0)", i, p, i)
		}
	}
}

func TestLineDiagonal(t *testing.T) {
	var count int
	Line(Vec3{0, 0, 0}, Vec3{3, 3, 3}, func(x, y, z int) { count++ })
	if count != 4 { // 3+1 points for a diagonal
		t.Fatalf("Line diagonal count = %d, want 4", count)
	}
}

func TestSphere(t *testing.T) {
	var count int
	Sphere(Vec3{10, 10, 10}, 2, func(x, y, z int) { count++ })
	if count == 0 {
		t.Fatal("Sphere produced 0 blocks")
	}
	// R=2, upper=(3)²=9. Center block: dist=0 < 9 → included.
	// Radius 2 sphere should have roughly 33 blocks with (R+1)² threshold.
	if count < 50 || count > 120 {
		t.Fatalf("Sphere R=2 count = %d, expected 50-120", count)
	}
}

func TestSphereHollow(t *testing.T) {
	var count int
	SphereHollow(Vec3{10, 10, 10}, 3, func(x, y, z int) { count++ })
	if count == 0 {
		t.Fatal("SphereHollow produced 0 blocks")
	}
}

func TestFill(t *testing.T) {
	// 5×1×5 flat plane filled with stone, one air in center
	blocks := make([]byte, 25)
	for i := range blocks {
		blocks[i] = 1 // stone
	}
	blocks[12] = 0 // air in center

	var filled []int
	Fill(blocks, 5, 1, 5, 0, FillNormal, func(x, y, z int) {
		filled = append(filled, x+y*0+z*5)
	})
	// Should fill all stone (24 blocks, skipping the air at 12)
	if len(filled) != 24 {
		t.Fatalf("Fill count = %d, want 24", len(filled))
	}
}

func TestFillLayer(t *testing.T) {
	// 3×3×3 cube, fill from center with layer mode (no Y spread)
	blocks := make([]byte, 27)
	for i := range blocks {
		blocks[i] = 1
	}
	var count int
	Fill(blocks, 3, 3, 3, 13, FillLayer, func(x, y, z int) { count++ })
	// Layer mode: only fills the Y=1 plane (3×3=9 blocks)
	if count != 9 {
		t.Fatalf("FillLayer count = %d, want 9", count)
	}
}
