//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// playerValue wraps a plugin.Player in a Lua table with methods.
// Each method is a closure that captures the player interface.
func (a *api) playerValue(p plugin.Player) glua.LValue {
	tbl := a.L.NewTable()
	a.L.SetField(tbl, "name", glua.LString(p.Name()))
	a.L.SetField(tbl, "message", a.L.NewFunction(func(L *glua.LState) int {
		p.Message(L.CheckString(1))
		return 0
	}))
	a.L.SetField(tbl, "kick", a.L.NewFunction(func(L *glua.LState) int {
		p.Kick(L.CheckString(1))
		return 0
	}))
	a.L.SetField(tbl, "is_op", a.L.NewFunction(func(L *glua.LState) int {
		L.Push(glua.LBool(p.IsOperator()))
		return 1
	}))
	a.L.SetField(tbl, "position", a.L.NewFunction(func(L *glua.LState) int {
		x, y, z := p.Position()
		L.Push(glua.LNumber(x))
		L.Push(glua.LNumber(y))
		L.Push(glua.LNumber(z))
		return 3
	}))
	return tbl
}
