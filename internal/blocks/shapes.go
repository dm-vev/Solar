// Package drawing implements geometry algorithms for build commands.
//
// All functions are pure: they take coordinates and a callback, and
// call the callback for each block that should be placed. The caller
// is responsible for writing blocks to the world and broadcasting.
//
// Algorithms are ported from MCGalaxy's DrawOps:
//   - Cuboid: bounding box fill
//   - Line: 3D Bresenham
//   - Sphere: integer squared distance (AdvSphere)
//   - Fill: 6-connected flood fill
package blocks

// Vec3 is a 3D integer coordinate.
type Vec3 struct{ X, Y, Z int }

// PlaceFunc is called for each block a geometry operation emits.
type PlaceFunc func(x, y, z int)

// Cuboid fills the bounding box from min to max (inclusive).
// Loop order is Y, Z, X (X innermost) for cache-friendly writes.
func Cuboid(min, max Vec3, place PlaceFunc) {
	for y := min.Y; y <= max.Y; y++ {
		for z := min.Z; z <= max.Z; z++ {
			for x := min.X; x <= max.X; x++ {
				place(x, y, z)
			}
		}
	}
}

// CuboidHollow places only the 6 faces of the bounding box.
func CuboidHollow(min, max Vec3, place PlaceFunc) {
	for y := min.Y; y <= max.Y; y++ {
		for z := min.Z; z <= max.Z; z++ {
			for x := min.X; x <= max.X; x++ {
				if x == min.X || x == max.X ||
					y == min.Y || y == max.Y ||
					z == min.Z || z == max.Z {
					place(x, y, z)
				}
			}
		}
	}
}

// CuboidWalls places only the 4 vertical faces (no top/bottom).
func CuboidWalls(min, max Vec3, place PlaceFunc) {
	for y := min.Y; y <= max.Y; y++ {
		for z := min.Z; z <= max.Z; z++ {
			for x := min.X; x <= max.X; x++ {
				if (x == min.X || x == max.X) && (z == min.Z || z == max.Z) {
					place(x, y, z)
				}
			}
		}
	}
}

// Line draws a 3D Bresenham line from p1 to p2 (inclusive).
func Line(p1, p2 Vec3, place PlaceFunc) {
	dx, dy, dz := p2.X-p1.X, p2.Y-p1.Y, p2.Z-p1.Z
	ax, ay, az := abs(dx), abs(dy), abs(dz)

	sx, sy, sz := sign(dx), sign(dy), sign(dz)
	px, py, pz := p1.X, p1.Y, p1.Z

	// Driving axis is the longest one.
	type axis struct {
		len2, dir int
		coord     *int
	}
	mk := func(delta, s int, coord *int) axis {
		return axis{abs(delta) * 2, s, coord}
	}

	lx := mk(dx, sx, &px)
	ly := mk(dy, sy, &py)
	lz := mk(dz, sz, &pz)

	doLine := func(a1, a2, a3 axis, length int) {
		err1 := a1.len2 - length
		err2 := a2.len2 - length
		for i := 0; i < length; i++ {
			place(px, py, pz)
			if err1 > 0 {
				*a1.coord += a1.dir
				err1 -= a3.len2
			}
			if err2 > 0 {
				*a2.coord += a2.dir
				err2 -= a3.len2
			}
			err1 += a1.len2
			err2 += a2.len2
			*a3.coord += a3.dir
		}
	}

	switch {
	case ax >= ay && ax >= az:
		doLine(ly, lz, lx, ax)
	case ay >= ax && ay >= az:
		doLine(lx, lz, ly, ay)
	default:
		doLine(lx, ly, lz, az)
	}
	place(px, py, pz) // final endpoint
}

// Sphere fills a sphere centered at center with radius R.
// Uses integer squared distance with threshold (R+1)² for smoothness.
func Sphere(center Vec3, R int, place PlaceFunc) {
	upper := (R + 1) * (R + 1)
	for y := center.Y - R; y <= center.Y+R; y++ {
		for z := center.Z - R; z <= center.Z+R; z++ {
			for x := center.X - R; x <= center.X+R; x++ {
				dx, dy, dz := center.X-x, center.Y-y, center.Z-z
				if dx*dx+dy*dy+dz*dz < upper {
					place(x, y, z)
				}
			}
		}
	}
}

// SphereHollow places only the shell of a sphere.
func SphereHollow(center Vec3, R int, place PlaceFunc) {
	upper := (R + 1) * (R + 1)
	inner := (R - 1) * (R - 1)
	for y := center.Y - R; y <= center.Y+R; y++ {
		for z := center.Z - R; z <= center.Z+R; z++ {
			for x := center.X - R; x <= center.X+R; x++ {
				dx, dy, dz := center.X-x, center.Y-y, center.Z-z
				dist := dx*dx + dy*dy + dz*dz
				if dist >= inner && dist < upper {
					place(x, y, z)
				}
			}
		}
	}
}

// FillMode controls which directions a flood fill spreads.
type FillMode int

const (
	FillNormal FillMode = iota // 6-connected (all directions)
	FillUp                     // no down
	FillDown                   // no up
	FillLayer                  // 2D in XZ plane (no up/down)
)

// Fill performs a 6-connected flood fill starting at start.
// It replaces all connected blocks of the same type as the seed block.
// blocks is the flat level array, indexed as x + width*(z + length*y).
// The callback is called for each block to fill.
func Fill(blocks []byte, width, height, length int, start int, mode FillMode, place PlaceFunc) {
	target := blocks[start]
	visited := make([]bool, width*height*length)
	oneY := width * length

	stack := []int{start}
	for len(stack) > 0 {
		idx := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if visited[idx] {
			continue
		}
		visited[idx] = true

		x := idx % width
		rem := idx / width
		z := rem % length
		y := rem / length

		place(x, y, z)

		// X neighbours
		if x+1 < width && blocks[idx+1] == target && !visited[idx+1] {
			stack = append(stack, idx+1)
		}
		if x-1 >= 0 && blocks[idx-1] == target && !visited[idx-1] {
			stack = append(stack, idx-1)
		}
		// Z neighbours
		if z+1 < length && blocks[idx+width] == target && !visited[idx+width] {
			stack = append(stack, idx+width)
		}
		if z-1 >= 0 && blocks[idx-width] == target && !visited[idx-width] {
			stack = append(stack, idx-width)
		}
		// Y+ (up) — skipped in Down and Layer modes
		if mode != FillDown && mode != FillLayer {
			if y+1 < height && blocks[idx+oneY] == target && !visited[idx+oneY] {
				stack = append(stack, idx+oneY)
			}
		}
		// Y- (down) — skipped in Up and Layer modes
		if mode != FillUp && mode != FillLayer {
			if y-1 >= 0 && blocks[idx-oneY] == target && !visited[idx-oneY] {
				stack = append(stack, idx-oneY)
			}
		}
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func sign(v int) int {
	if v > 0 {
		return 1
	}
	if v < 0 {
		return -1
	}
	return 0
}
