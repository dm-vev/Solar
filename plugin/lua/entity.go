//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

func checkEntityManager(L *glua.LState, idx int) plugin.EntityManager {
	if v, ok := udValue(L, idx).(plugin.EntityManager); ok {
		return v
	}
	L.ArgError(idx, "expected entity_manager")
	return nil
}

var entityManagerMethods = map[string]glua.LGFunction{
	// spawn(info_table) -> entity_id (number) or nil
	// info_table: { name=..., x=..., y=..., z=..., yaw=..., pitch=..., model=... }
	"spawn": func(L *glua.LState) int {
		em := checkEntityManager(L, 1)
		tbl := L.CheckTable(2)
		info := plugin.EntityInfo{
			Name:  L.GetField(tbl, "name").String(),
			X:     int(L.GetField(tbl, "x").(glua.LNumber)),
			Y:     int(L.GetField(tbl, "y").(glua.LNumber)),
			Z:     int(L.GetField(tbl, "z").(glua.LNumber)),
			Model: L.GetField(tbl, "model").String(),
		}
		if v := L.GetField(tbl, "yaw"); v != glua.LNil {
			info.Yaw = byte(v.(glua.LNumber))
		}
		if v := L.GetField(tbl, "pitch"); v != glua.LNil {
			info.Pitch = byte(v.(glua.LNumber))
		}
		id := em.Spawn(info)
		if id == 0 {
			L.Push(glua.LNil)
			return 1
		}
		L.Push(glua.LNumber(id))
		return 1
	},

	// despawn(entity_id) -> bool
	"despawn": func(L *glua.LState) int {
		L.Push(glua.LBool(checkEntityManager(L, 1).Despawn(byte(L.CheckInt(2)))))
		return 1
	},

	// teleport(entity_id, x, y, z, yaw, pitch) -> bool
	"teleport": func(L *glua.LState) int {
		L.Push(glua.LBool(checkEntityManager(L, 1).Teleport(
			byte(L.CheckInt(2)), L.CheckInt(3), L.CheckInt(4), L.CheckInt(5),
			byte(L.CheckInt(6)), byte(L.CheckInt(7)))))
		return 1
	},

	// get(entity_id) -> info_table or nil
	"get": func(L *glua.LState) int {
		info, ok := checkEntityManager(L, 1).Get(byte(L.CheckInt(2)))
		if !ok {
			L.Push(glua.LNil)
			return 1
		}
		tbl := L.NewTable()
		L.SetField(tbl, "name", glua.LString(info.Name))
		L.SetField(tbl, "x", glua.LNumber(info.X))
		L.SetField(tbl, "y", glua.LNumber(info.Y))
		L.SetField(tbl, "z", glua.LNumber(info.Z))
		L.SetField(tbl, "yaw", glua.LNumber(info.Yaw))
		L.SetField(tbl, "pitch", glua.LNumber(info.Pitch))
		L.SetField(tbl, "model", glua.LString(info.Model))
		L.Push(tbl)
		return 1
	},

	// count() -> number
	"count": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkEntityManager(L, 1).Count()))
		return 1
	},
}
