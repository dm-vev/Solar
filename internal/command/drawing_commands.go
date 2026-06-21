// drawing_commands.go implements build commands that use the mark system.
//
// All drawing commands work by starting a 2-click block selection:
//   1. Player runs /cuboid <block>
//   2. Server intercepts the next 2 block clicks as marks
//   3. The geometry algorithm runs, placing blocks via PlaceBlock
//
// Commands:
//   /cuboid <block> [hollow|walls] — fill a box
//   /line <block>                  — draw a 3D Bresenham line
//   /sphere <block> [hollow] <r>   — draw a sphere
//   /fill <block>                  — flood fill connected blocks

package command

import (
	"strconv"

	"github.com/solar-mc/solar/internal/blocks"
)

func parseBlock(s string) (byte, bool) {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 || v > 255 {
		return 0, false
	}
	return byte(v), true
}

// checkDrawLimit verifies the block count doesn't exceed the player's draw limit.
func checkDrawLimit(ctx Context, blockCount int) (string, bool) {
	limit := ctx.Draw.DrawLimit()
	if limit > 0 && blockCount > limit {
		return ctx.tr("command.draw.limit", blockCount, limit), true
	}
	return "", false
}

// cuboidVolume returns the number of blocks in a cuboid from min to max.
func cuboidVolume(min, max blocks.Vec3) int {
	return (max.X - min.X + 1) * (max.Y - min.Y + 1) * (max.Z - min.Z + 1)
}

// cuboidCommand — /cuboid <block> [hollow|walls]
func cuboidCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.cuboid.usage"), true
	}
	block, ok := parseBlock(args[0])
	if !ok {
		return ctx.tr("command.draw.invalid_block", args[0]), true
	}
	hollow := len(args) > 1 && args[1] == "hollow"
	walls := len(args) > 1 && args[1] == "walls"

	// Check per-block permission.
	if !ctx.Draw.CanPlace(block) {
		return ctx.tr("command.draw.cannot_place", block), true
	}

	ctx.Draw.StartSelection(2, func(marks [][3]int) {
		min := blocks.Vec3{marks[0][0], marks[0][1], marks[0][2]}
		max := blocks.Vec3{marks[1][0], marks[1][1], marks[1][2]}
		if min.X > max.X {
			min.X, max.X = max.X, min.X
		}
		if min.Y > max.Y {
			min.Y, max.Y = max.Y, min.Y
		}
		if min.Z > max.Z {
			min.Z, max.Z = max.Z, min.Z
		}
		// Check draw limit before executing.
		vol := cuboidVolume(min, max)
		limit := ctx.Draw.DrawLimit()
		if limit > 0 && vol > limit {
			return // silently fail — too many blocks
		}
		ctx.Draw.BeginBatch()
		place := func(x, y, z int) {
			old, _ := ctx.Draw.GetBlockAt(x, y, z)
			if ctx.Draw.PlaceBlock(x, y, z, block) {
				ctx.Draw.RecordChange(x, y, z, old, block)
			}
		}
		switch {
		case hollow:
			blocks.CuboidHollow(min, max, place)
		case walls:
			blocks.CuboidWalls(min, max, place)
		default:
			blocks.Cuboid(min, max, place)
		}
		ctx.Draw.CommitBatch()
	})
	return ctx.tr("command.draw.select2"), true
}

// lineCommand — /line <block>
func lineCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.line.usage"), true
	}
	block, ok := parseBlock(args[0])
	if !ok {
		return ctx.tr("command.draw.invalid_block", args[0]), true
	}
	if !ctx.Draw.CanPlace(block) {
		return ctx.tr("command.draw.cannot_place", block), true
	}
	ctx.Draw.StartSelection(2, func(marks [][3]int) {
		p1 := blocks.Vec3{marks[0][0], marks[0][1], marks[0][2]}
		p2 := blocks.Vec3{marks[1][0], marks[1][1], marks[1][2]}
		ctx.Draw.BeginBatch()
		blocks.Line(p1, p2, func(x, y, z int) {
			old, _ := ctx.Draw.GetBlockAt(x, y, z)
			if ctx.Draw.PlaceBlock(x, y, z, block) {
				ctx.Draw.RecordChange(x, y, z, old, block)
			}
		})
		ctx.Draw.CommitBatch()
	})
	return ctx.tr("command.draw.select2"), true
}

// sphereCommand — /sphere <block> [hollow] [radius]
func sphereCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.sphere.usage"), true
	}
	block, ok := parseBlock(args[0])
	if !ok {
		return ctx.tr("command.draw.invalid_block", args[0]), true
	}
	if !ctx.Draw.CanPlace(block) {
		return ctx.tr("command.draw.cannot_place", block), true
	}
	hollow := false
	radius := 3
	for i := 1; i < len(args); i++ {
		if args[i] == "hollow" {
			hollow = true
		} else if r, err := strconv.Atoi(args[i]); err == nil && r > 0 {
			radius = r
		}
	}
	ctx.Draw.StartSelection(1, func(marks [][3]int) {
		center := blocks.Vec3{marks[0][0], marks[0][1], marks[0][2]}
		ctx.Draw.BeginBatch()
		if hollow {
			blocks.SphereHollow(center, radius, func(x, y, z int) {
				old, _ := ctx.Draw.GetBlockAt(x, y, z)
				if ctx.Draw.PlaceBlock(x, y, z, block) {
					ctx.Draw.RecordChange(x, y, z, old, block)
				}
			})
		} else {
			blocks.Sphere(center, radius, func(x, y, z int) {
				old, _ := ctx.Draw.GetBlockAt(x, y, z)
				if ctx.Draw.PlaceBlock(x, y, z, block) {
					ctx.Draw.RecordChange(x, y, z, old, block)
				}
			})
		}
		ctx.Draw.CommitBatch()
	})
	return ctx.tr("command.draw.select1"), true
}

// fillCommand — /fill <block>
func fillCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.fill.usage"), true
	}
	block, ok := parseBlock(args[0])
	if !ok {
		return ctx.tr("command.draw.invalid_block", args[0]), true
	}
	if !ctx.Draw.CanPlace(block) {
		return ctx.tr("command.draw.cannot_place", block), true
	}
	ctx.Draw.StartSelection(1, func(marks [][3]int) {
		w, h, l := ctx.Draw.LevelDims()
		x, y, z := marks[0][0], marks[0][1], marks[0][2]
		if x < 0 || x >= w || y < 0 || y >= h || z < 0 || z >= l {
			return
		}
		// Get the current level blocks for flood fill.
		// ponytail: reading blocks one-by-one is slow for large fills;
		// a bulk read API would be better. Acceptable for now.
		levelBlocks := make([]byte, w*h*l)
		for yy := 0; yy < h; yy++ {
			for zz := 0; zz < l; zz++ {
				for xx := 0; xx < w; xx++ {
					b, _ := ctx.Draw.GetBlockAt(xx, yy, zz)
					levelBlocks[xx+w*(zz+l*yy)] = b
				}
			}
		}
		startIdx := x + w*(z+l*y)
		ctx.Draw.BeginBatch()
		blocks.Fill(levelBlocks, w, h, l, startIdx, blocks.FillNormal, func(fx, fy, fz int) {
			old, _ := ctx.Draw.GetBlockAt(fx, fy, fz)
			if ctx.Draw.PlaceBlock(fx, fy, fz, block) {
				ctx.Draw.RecordChange(fx, fy, fz, old, block)
			}
		})
		ctx.Draw.CommitBatch()
	})
	return ctx.tr("command.draw.select1"), true
}
