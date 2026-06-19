//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

func checkLevel(L *glua.LState, idx int) plugin.Level {
	if v, ok := udValue(L, idx).(plugin.Level); ok {
		return v
	}
	L.ArgError(idx, "expected level")
	return nil
}

func wrapLevel(L *glua.LState, l plugin.Level) glua.LValue {
	if l == nil {
		return glua.LNil
	}
	return wrapUD(L, typeLevel, l)
}

func checkLevelManager(L *glua.LState, idx int) plugin.LevelManager {
	if v, ok := udValue(L, idx).(plugin.LevelManager); ok {
		return v
	}
	L.ArgError(idx, "expected level_manager")
	return nil
}

func wrapLevelManager(L *glua.LState, lm plugin.LevelManager) glua.LValue {
	if lm == nil {
		return glua.LNil
	}
	return wrapUD(L, typeLevelManager, lm)
}

var levelMethods = map[string]glua.LGFunction{
	"name": func(L *glua.LState) int {
		L.Push(glua.LString(checkLevel(L, 1).Name()))
		return 1
	},
	"get_block": func(L *glua.LState) int {
		b, ok := checkLevel(L, 1).GetBlock(L.CheckInt(2), L.CheckInt(3), L.CheckInt(4))
		if !ok {
			L.Push(glua.LNil)
			return 1
		}
		L.Push(glua.LNumber(b))
		return 1
	},
	"set_block": func(L *glua.LState) int {
		L.Push(glua.LBool(checkLevel(L, 1).SetBlock(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4), byte(L.CheckInt(5)))))
		return 1
	},
	"spawn": func(L *glua.LState) int {
		x, y, z, yaw, pitch := checkLevel(L, 1).Spawn()
		L.Push(glua.LNumber(x))
		L.Push(glua.LNumber(y))
		L.Push(glua.LNumber(z))
		L.Push(glua.LNumber(yaw))
		L.Push(glua.LNumber(pitch))
		return 5
	},
	"set_spawn": func(L *glua.LState) int {
		checkLevel(L, 1).SetSpawn(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4),
			byte(L.CheckInt(5)), byte(L.CheckInt(6)))
		return 0
	},
	"dimensions": func(L *glua.LState) int {
		w, h, l := checkLevel(L, 1).Dimensions()
		L.Push(glua.LNumber(w))
		L.Push(glua.LNumber(h))
		L.Push(glua.LNumber(l))
		return 3
	},
	"save": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Save(); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	"player_count": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkLevel(L, 1).PlayerCount()))
		return 1
	},
	"players": func(L *glua.LState) int {
		tbl := L.NewTable()
		for i, p := range checkLevel(L, 1).Players() {
			tbl.RawSetInt(i+1, wrapPlayer(L, p))
		}
		L.Push(tbl)
		return 1
	},
	"message": func(L *glua.LState) int {
		checkLevel(L, 1).Message(L.CheckString(2))
		return 0
	},
	"rename": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Rename(L.CheckString(2)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	"copy": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Copy(L.CheckString(2)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	"backup": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Backup(L.CheckString(2)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	"delete": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Delete(); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	"resize": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Resize(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	"reload": func(L *glua.LState) int {
		if err := checkLevel(L, 1).Reload(); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
}

var levelManagerMethods = map[string]glua.LGFunction{
	// current() -> level
	"current": func(L *glua.LState) int {
		L.Push(wrapLevel(L, checkLevelManager(L, 1).Current()))
		return 1
	},
	// find(name) -> level or nil
	"find": func(L *glua.LState) int {
		l := checkLevelManager(L, 1).Find(L.CheckString(2))
		L.Push(wrapLevel(L, l))
		return 1
	},
	// create(name, w, h, l, generator, seed) -> level or nil
	"create": func(L *glua.LState) int {
		l, err := checkLevelManager(L, 1).Create(
			L.CheckString(2), L.CheckInt(3), L.CheckInt(4), L.CheckInt(5),
			L.CheckString(6), L.CheckString(7))
		if err != nil {
			L.Push(glua.LNil)
			L.Push(glua.LString(err.Error()))
			return 2
		}
		L.Push(wrapLevel(L, l))
		L.Push(glua.LNil)
		return 2
	},
	// load(name) -> level or nil
	"load": func(L *glua.LState) int {
		l, err := checkLevelManager(L, 1).Load(L.CheckString(2))
		if err != nil {
			L.Push(glua.LNil)
			L.Push(glua.LString(err.Error()))
			return 2
		}
		L.Push(wrapLevel(L, l))
		L.Push(glua.LNil)
		return 2
	},
	// unload(name) -> bool
	"unload": func(L *glua.LState) int {
		L.Push(glua.LBool(checkLevelManager(L, 1).Unload(L.CheckString(2))))
		return 1
	},
	// save_all() -> bool
	"save_all": func(L *glua.LState) int {
		if err := checkLevelManager(L, 1).SaveAll(); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	// list() -> table of names
	"list": func(L *glua.LState) int {
		tbl := L.NewTable()
		for i, name := range checkLevelManager(L, 1).List() {
			tbl.RawSetInt(i+1, glua.LString(name))
		}
		L.Push(tbl)
		return 1
	},
	// list_files() -> table of names
	"list_files": func(L *glua.LState) int {
		tbl := L.NewTable()
		for i, name := range checkLevelManager(L, 1).ListFiles() {
			tbl.RawSetInt(i+1, glua.LString(name))
		}
		L.Push(tbl)
		return 1
	},
	// rename_level(old, new)
	"rename_level": func(L *glua.LState) int {
		if err := checkLevelManager(L, 1).RenameLevel(
			L.CheckString(2), L.CheckString(3)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	// delete_level(name)
	"delete_level": func(L *glua.LState) int {
		if err := checkLevelManager(L, 1).DeleteLevel(L.CheckString(2)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	// copy_level(src, dest)
	"copy_level": func(L *glua.LState) int {
		if err := checkLevelManager(L, 1).CopyLevel(
			L.CheckString(2), L.CheckString(3)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
	// backup_level(name, backup_name)
	"backup_level": func(L *glua.LState) int {
		if err := checkLevelManager(L, 1).BackupLevel(
			L.CheckString(2), L.CheckString(3)); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
}
