//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

func checkWorld(L *glua.LState, idx int) plugin.World {
	if v, ok := udValue(L, idx).(plugin.World); ok {
		return v
	}
	L.ArgError(idx, "expected world")
	return nil
}

func wrapWorld(L *glua.LState, w plugin.World) glua.LValue {
	if w == nil {
		return glua.LNil
	}
	return wrapUD(L, typeWorld, w)
}

var worldMethods = map[string]glua.LGFunction{
	// get_block(x, y, z) -> block (number) or nil
	"get_block": func(L *glua.LState) int {
		b, ok := checkWorld(L, 1).GetBlock(L.CheckInt(2), L.CheckInt(3), L.CheckInt(4))
		if !ok {
			L.Push(glua.LNil)
			return 1
		}
		L.Push(glua.LNumber(b))
		return 1
	},

	// set_block(x, y, z, block) -> bool
	"set_block": func(L *glua.LState) int {
		L.Push(glua.LBool(checkWorld(L, 1).SetBlock(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4), byte(L.CheckInt(5)))))
		return 1
	},

	// spawn() -> x, y, z, yaw, pitch
	"spawn": func(L *glua.LState) int {
		x, y, z, yaw, pitch := checkWorld(L, 1).Spawn()
		L.Push(glua.LNumber(x))
		L.Push(glua.LNumber(y))
		L.Push(glua.LNumber(z))
		L.Push(glua.LNumber(yaw))
		L.Push(glua.LNumber(pitch))
		return 5
	},

	// set_spawn(x, y, z, yaw, pitch)
	"set_spawn": func(L *glua.LState) int {
		checkWorld(L, 1).SetSpawn(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4),
			byte(L.CheckInt(5)), byte(L.CheckInt(6)))
		return 0
	},

	// dimensions() -> width, height, length
	"dimensions": func(L *glua.LState) int {
		w, h, l := checkWorld(L, 1).Dimensions()
		L.Push(glua.LNumber(w))
		L.Push(glua.LNumber(h))
		L.Push(glua.LNumber(l))
		return 3
	},

	// save() -> bool
	"save": func(L *glua.LState) int {
		if err := checkWorld(L, 1).Save(); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
}
