//go:build lua

package lua

import (
	"fmt"

	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// server table methods exposed to Lua as server.* and some as globals.

func (a *api) serverBroadcast(L *glua.LState) int {
	a.srv.BroadcastMessage(L.CheckString(1))
	return 0
}

func (a *api) serverFindPlayer(L *glua.LState) int {
	p := a.srv.FindPlayer(L.CheckString(1))
	if p == nil {
		L.Push(glua.LNil)
	} else {
		L.Push(a.playerValue(p))
	}
	return 1
}

func (a *api) serverOnlineCount(L *glua.LState) int {
	L.Push(glua.LNumber(a.srv.OnlineCount()))
	return 1
}

func (a *api) serverName(L *glua.LState) int {
	L.Push(glua.LString(a.srv.ServerName()))
	return 1
}

func (a *api) serverMotd(L *glua.LState) int {
	L.Push(glua.LString(a.srv.MOTD()))
	return 1
}

func (a *api) serverStop(L *glua.LState) int {
	a.srv.Stop()
	return 0
}

func (a *api) utilColorize(L *glua.LState) int {
	L.Push(glua.LString(plugin.Colorize(plugin.ColorWhite, L.CheckString(1))))
	return 1
}

func (a *api) utilStripColor(L *glua.LState) int {
	L.Push(glua.LString(plugin.StripColor(L.CheckString(1))))
	return 1
}

// serverRegisterCommand: register_command(name, help, fn)
// fn(name, args_table) → string reply
func (a *api) serverRegisterCommand(L *glua.LState) int {
	name := L.CheckString(1)
	help := L.CheckString(2)
	fn := L.CheckFunction(3)
	ok := a.srv.RegisterCommand(name, help, func(p plugin.Player, args []string) string {
		defer func() {
			if r := recover(); r != nil {
				a.L.RaiseError("lua command %s panicked: %v", name, r)
			}
		}()
		argsTbl := a.L.NewTable()
		for i, arg := range args {
			a.L.SetField(argsTbl, fmt.Sprintf("%d", i+1), glua.LString(arg))
		}
		rets := a.callFn(fn, 1, glua.LString(p.Name()), argsTbl)
		if len(rets) == 0 {
			return ""
		}
		if s, ok := rets[0].(glua.LString); ok {
			return string(s)
		}
		return ""
	})
	L.Push(glua.LBool(ok))
	return 1
}
