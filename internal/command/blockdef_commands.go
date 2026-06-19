package command

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/solar-mc/solar/internal/blockdef"
)

func globalBlockCommand(ctx Context, args []string) (string, bool) {
	return blockDefCommand(ctx, args, "/gb")
}

func levelBlockCommand(ctx Context, args []string) (string, bool) {
	return blockDefCommand(ctx, args, "/lb")
}

func blockDefCommand(ctx Context, args []string, cmd string) (string, bool) {
	if ctx.BlockDefs == nil {
		return "block definitions are unavailable", true
	}
	if len(args) == 0 {
		return blockDefUsage(cmd)
	}

	switch strings.ToLower(args[0]) {
	case "add", "create":
		return blockDefAdd(ctx, args[1:], cmd)
	case "edit":
		return blockDefEdit(ctx, args[1:], cmd)
	case "remove", "delete":
		return blockDefRemove(ctx, args[1:], cmd)
	case "info", "about":
		return blockDefInfo(ctx, args[1:])
	case "list", "ids":
		return blockDefList(ctx)
	default:
		return blockDefUsage(cmd)
	}
}

func blockDefUsage(cmd string) (string, bool) {
	return fmt.Sprintf("usage: %s <add|edit|remove|info|list>", cmd), true
}

func blockDefAdd(ctx Context, args []string, cmd string) (string, bool) {
	if len(args) == 0 {
		id := ctx.BlockDefs.FreeBlockID()
		if id == 0 {
			return "no free block IDs available", true
		}
		return fmt.Sprintf("usage: %s add <id> [name] (next free ID: %d)", cmd, id), true
	}

	id, err := parseBlockID(args[0])
	if err != nil {
		return fmt.Sprintf("invalid id: %v", err), true
	}

	if id < blockdef.FirstCustomBlock {
		return fmt.Sprintf("custom block IDs must be >= %d", blockdef.FirstCustomBlock), true
	}

	def := blockdef.Default(id)
	if len(args) > 1 {
		def.Name = args[1]
	}

	if !ctx.BlockDefs.AddBlockDef(def) {
		return "failed to add block definition", true
	}
	return fmt.Sprintf("added block %d (%s)", id, def.Name), true
}

func blockDefEdit(ctx Context, args []string, cmd string) (string, bool) {
	if len(args) < 3 {
		return fmt.Sprintf("usage: %s edit <id> <property> <value>", cmd), true
	}

	id, err := parseBlockID(args[0])
	if err != nil {
		return fmt.Sprintf("invalid id: %v", err), true
	}

	def, ok := ctx.BlockDefs.GetBlockDef(id)
	if !ok {
		return fmt.Sprintf("block %d is not defined", id), true
	}

	prop := strings.ToLower(args[1])
	val := args[2]

	if err := editBlockProp(&def, prop, val); err != nil {
		return fmt.Sprintf("invalid value for %s: %v", prop, err), true
	}

	if !ctx.BlockDefs.AddBlockDef(def) {
		return "failed to update block definition", true
	}
	return fmt.Sprintf("updated block %d %s = %s", id, prop, val), true
}

func blockDefRemove(ctx Context, args []string, cmd string) (string, bool) {
	if len(args) != 1 {
		return fmt.Sprintf("usage: %s remove <id>", cmd), true
	}

	id, err := parseBlockID(args[0])
	if err != nil {
		return fmt.Sprintf("invalid id: %v", err), true
	}

	if !ctx.BlockDefs.RemoveBlockDef(id) {
		return fmt.Sprintf("block %d is not defined", id), true
	}
	return fmt.Sprintf("removed block %d", id), true
}

func blockDefInfo(ctx Context, args []string) (string, bool) {
	if len(args) != 1 {
		return "usage: info <id>", true
	}

	id, err := parseBlockID(args[0])
	if err != nil {
		return fmt.Sprintf("invalid id: %v", err), true
	}

	def, ok := ctx.BlockDefs.GetBlockDef(id)
	if !ok {
		return fmt.Sprintf("block %d is not defined", id), true
	}

	return formatBlockDef(def), true
}

func blockDefList(ctx Context) (string, bool) {
	defs := ctx.BlockDefs.ListBlockDefs()
	if len(defs) == 0 {
		return "no custom blocks defined", true
	}

	parts := make([]string, len(defs))
	for i, d := range defs {
		parts[i] = fmt.Sprintf("%d:%s", d.ID, d.Name)
	}
	return fmt.Sprintf("blocks (%d): %s", len(defs), strings.Join(parts, ", ")), true
}

func formatBlockDef(d blockdef.BlockDefinition) string {
	return fmt.Sprintf("block %d: name=%s collide=%s speed=%.1f tex=%d/%d/%d draw=%s sound=%s shape=%d fallback=%d fog=%d(%d,%d,%d) bbox=%d,%d,%d-%d,%d,%d",
		d.ID, d.Name, blockdef.CollideTypeName(d.CollideType), d.Speed,
		d.TopTex, d.RightTex, d.BottomTex,
		blockdef.DrawTypeName(d.BlockDraw), blockdef.SoundTypeName(d.WalkSound),
		d.Shape, d.FallBack, d.FogDensity, d.FogR, d.FogG, d.FogB,
		d.MinX, d.MinY, d.MinZ, d.MaxX, d.MaxY, d.MaxZ,
	)
}

func parseBlockID(s string) (byte, error) {
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if v < 0 || v > 255 {
		return 0, fmt.Errorf("id must be 0..255")
	}
	return byte(v), nil
}

func editBlockProp(def *blockdef.BlockDefinition, prop, val string) error {
	switch prop {
	case "name":
		def.Name = val
	case "collide", "collidetype":
		v, err := parseByteProp(val, 7)
		if err != nil {
			return err
		}
		def.CollideType = v
	case "speed":
		v, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return err
		}
		def.Speed = v
	case "toptex", "bottomtex", "sidetex", "alltex", "lefttex", "righttex", "fronttex", "backtex":
		return editTexture(def, prop, val)
	case "blockslight":
		def.BlocksLight = parseBoolProp(val)
	case "sound", "walksound":
		v, err := parseByteProp(val, 9)
		if err != nil {
			return err
		}
		def.WalkSound = v
	case "fullbright":
		def.FullBright = parseBoolProp(val)
	case "shape":
		v, err := parseByteProp(val, 16)
		if err != nil {
			return err
		}
		def.Shape = v
	case "blockdraw", "draw":
		v, err := parseByteProp(val, 5)
		if err != nil {
			return err
		}
		def.BlockDraw = v
	case "fallback":
		v, err := parseByteProp(val, 65)
		if err != nil {
			return err
		}
		def.FallBack = v
	case "fogdensity":
		v, err := parseByteProp(val, 255)
		if err != nil {
			return err
		}
		def.FogDensity = v
	case "fogcolor":
		return editFogColor(def, val)
	case "min":
		return editBBox(&def.MinX, &def.MinY, &def.MinZ, val)
	case "max":
		return editBBox(&def.MaxX, &def.MaxY, &def.MaxZ, val)
	default:
		return fmt.Errorf("unknown property %q (valid: name, collide, speed, toptex, sidetex, alltex, bottomtex, lefttex, righttex, fronttex, backtex, blockslight, sound, fullbright, shape, blockdraw, fallback, fogdensity, fogcolor, min, max)", prop)
	}
	return nil
}

func editTexture(def *blockdef.BlockDefinition, prop, val string) error {
	v, err := parseByteProp(val, 255)
	if err != nil {
		return err
	}
	switch prop {
	case "toptex":
		def.TopTex = v
	case "bottomtex":
		def.BottomTex = v
	case "sidetex":
		def.SetSideTex(v)
	case "alltex":
		def.SetAllTex(v)
	case "lefttex":
		def.LeftTex = v
	case "righttex":
		def.RightTex = v
	case "fronttex":
		def.FrontTex = v
	case "backtex":
		def.BackTex = v
	}
	return nil
}

func editFogColor(def *blockdef.BlockDefinition, val string) error {
	parts := strings.FieldsFunc(val, func(r rune) bool { return r == ' ' || r == ',' })
	if len(parts) != 3 {
		return fmt.Errorf("expected r g b")
	}
	r, err := parseByteProp(parts[0], 255)
	if err != nil {
		return err
	}
	g, err := parseByteProp(parts[1], 255)
	if err != nil {
		return err
	}
	b, err := parseByteProp(parts[2], 255)
	if err != nil {
		return err
	}
	def.FogR, def.FogG, def.FogB = r, g, b
	return nil
}

func editBBox(x, y, z *byte, val string) error {
	parts := strings.FieldsFunc(val, func(r rune) bool { return r == ' ' || r == ',' })
	if len(parts) != 3 {
		return fmt.Errorf("expected x y z")
	}
	xv, err := parseByteProp(parts[0], 16)
	if err != nil {
		return err
	}
	yv, err := parseByteProp(parts[1], 16)
	if err != nil {
		return err
	}
	zv, err := parseByteProp(parts[2], 16)
	if err != nil {
		return err
	}
	*x, *y, *z = xv, yv, zv
	return nil
}

func parseByteProp(s string, max int) (byte, error) {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	if v < 0 || v > max {
		return 0, fmt.Errorf("must be 0..%d", max)
	}
	return byte(v), nil
}

func parseBoolProp(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	default:
		return false
	}
}
