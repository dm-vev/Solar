//go:build lua

package lua

import (
	"time"

	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/playerdb"
)

func unixToTime(unix int64) time.Time {
	return time.Unix(unix, 0)
}

const typePlayerDB = "solar_playerdb"

func checkPlayerDB(L *glua.LState, idx int) playerdb.PlayerDB {
	if v, ok := udValue(L, idx).(playerdb.PlayerDB); ok {
		return v
	}
	L.ArgError(idx, "expected player_db")
	return nil
}

var playerdbMethods = map[string]glua.LGFunction{
	// get(name) -> entry_table or nil
	"get": func(L *glua.LState) int {
		db := checkPlayerDB(L, 1)
		e := db.Get(L.CheckString(2))
		if e == nil {
			L.Push(glua.LNil)
			return 1
		}
		L.Push(entryToTable(L, e))
		return 1
	},

	// save(entry_table)
	"save": func(L *glua.LState) int {
		db := checkPlayerDB(L, 1)
		tbl := L.CheckTable(2)
		db.Save(entryFromTable(L, tbl))
		return 0
	},

	// delete(name) -> bool
	"delete": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayerDB(L, 1).Delete(L.CheckString(2))))
		return 1
	},

	// list() -> table of entries
	"list": func(L *glua.LState) int {
		db := checkPlayerDB(L, 1)
		entries := db.List()
		tbl := L.NewTable()
		for i, e := range entries {
			tbl.RawSetInt(i+1, entryToTable(L, e))
		}
		L.Push(tbl)
		return 1
	},

	// search(prefix) -> table of entries
	"search": func(L *glua.LState) int {
		db := checkPlayerDB(L, 1)
		entries := db.Search(L.CheckString(2))
		tbl := L.NewTable()
		for i, e := range entries {
			tbl.RawSetInt(i+1, entryToTable(L, e))
		}
		L.Push(tbl)
		return 1
	},

	// count() -> number
	"count": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkPlayerDB(L, 1).Count()))
		return 1
	},

	// flush() -> bool
	"flush": func(L *glua.LState) int {
		if err := checkPlayerDB(L, 1).Flush(); err != nil {
			L.Push(glua.LFalse)
			return 1
		}
		L.Push(glua.LTrue)
		return 1
	},
}

func entryToTable(L *glua.LState, e *plugin.PlayerEntry) *glua.LTable {
	tbl := L.NewTable()
	L.SetField(tbl, "name", glua.LString(e.Name))
	L.SetField(tbl, "first_login", glua.LNumber(e.FirstLogin.Unix()))
	L.SetField(tbl, "last_login", glua.LNumber(e.LastLogin.Unix()))
	L.SetField(tbl, "total_time", glua.LNumber(int64(e.TotalTime)))
	L.SetField(tbl, "login_count", glua.LNumber(e.LoginCount))
	L.SetField(tbl, "ip", glua.LString(e.IP))
	L.SetField(tbl, "last_ip", glua.LString(e.LastIP))
	L.SetField(tbl, "deaths", glua.LNumber(e.Deaths))
	L.SetField(tbl, "kicks", glua.LNumber(e.Kicks))
	L.SetField(tbl, "messages_sent", glua.LNumber(e.MessagesSent))
	L.SetField(tbl, "blocks_placed", glua.LNumber(e.BlocksPlaced))
	L.SetField(tbl, "blocks_deleted", glua.LNumber(e.BlocksDeleted))
	L.SetField(tbl, "title", glua.LString(e.Title))
	L.SetField(tbl, "title_color", glua.LString(e.TitleColor))
	L.SetField(tbl, "money", glua.LNumber(e.Money))
	if len(e.Data) > 0 {
		dataTbl := L.NewTable()
		for k, v := range e.Data {
			L.SetField(dataTbl, k, glua.LString(v))
		}
		L.SetField(tbl, "data", dataTbl)
	}
	return tbl
}

func entryFromTable(L *glua.LState, tbl *glua.LTable) *plugin.PlayerEntry {
	e := &plugin.PlayerEntry{}
	e.Name = L.GetField(tbl, "name").String()
	if v := L.GetField(tbl, "first_login"); v != glua.LNil {
		e.FirstLogin = unixToTime(int64(v.(glua.LNumber)))
	}
	if v := L.GetField(tbl, "last_login"); v != glua.LNil {
		e.LastLogin = unixToTime(int64(v.(glua.LNumber)))
	}
	if v := L.GetField(tbl, "total_time"); v != glua.LNil {
		e.TotalTime = time.Duration(int64(v.(glua.LNumber)))
	}
	if v := L.GetField(tbl, "login_count"); v != glua.LNil {
		e.LoginCount = int(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "ip"); v != glua.LNil {
		e.IP = v.String()
	}
	if v := L.GetField(tbl, "last_ip"); v != glua.LNil {
		e.LastIP = v.String()
	}
	if v := L.GetField(tbl, "deaths"); v != glua.LNil {
		e.Deaths = int(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "kicks"); v != glua.LNil {
		e.Kicks = int(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "messages_sent"); v != glua.LNil {
		e.MessagesSent = int(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "blocks_placed"); v != glua.LNil {
		e.BlocksPlaced = int64(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "blocks_deleted"); v != glua.LNil {
		e.BlocksDeleted = int64(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "title"); v != glua.LNil {
		e.Title = v.String()
	}
	if v := L.GetField(tbl, "title_color"); v != glua.LNil {
		e.TitleColor = v.String()
	}
	if v := L.GetField(tbl, "money"); v != glua.LNil {
		e.Money = int(v.(glua.LNumber))
	}
	if v := L.GetField(tbl, "data"); v != glua.LNil {
		if dt, ok := v.(*glua.LTable); ok {
			e.Data = make(map[string]string)
			dt.ForEach(func(k, val glua.LValue) {
				e.Data[k.String()] = val.String()
			})
		}
	}
	return e
}
