//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// api holds the Lua state and server handle, and provides methods to
// register the Lua globals and call Lua functions from Go.
type api struct {
	L   *glua.LState
	srv plugin.Server
}

func newAPI(L *glua.LState, srv plugin.Server) *api {
	return &api{L: L, srv: srv}
}

// register sets up all Lua globals: the server table, event hooks,
// utility functions, and command registration.
func (a *api) register() {
	srvTbl := a.L.NewTable()
	a.L.SetField(srvTbl, "broadcast", a.L.NewFunction(a.serverBroadcast))
	a.L.SetField(srvTbl, "find_player", a.L.NewFunction(a.serverFindPlayer))
	a.L.SetField(srvTbl, "online_count", a.L.NewFunction(a.serverOnlineCount))
	a.L.SetField(srvTbl, "server_name", a.L.NewFunction(a.serverName))
	a.L.SetField(srvTbl, "motd", a.L.NewFunction(a.serverMotd))
	a.L.SetField(srvTbl, "register_command", a.L.NewFunction(a.serverRegisterCommand))
	a.L.SetField(srvTbl, "stop", a.L.NewFunction(a.serverStop))
	a.L.SetGlobal("server", srvTbl)

	a.L.SetGlobal("broadcast", a.L.NewFunction(a.serverBroadcast))
	a.L.SetGlobal("colorize", a.L.NewFunction(a.utilColorize))
	a.L.SetGlobal("strip_color", a.L.NewFunction(a.utilStripColor))

	a.L.SetGlobal("on_connect", a.L.NewFunction(a.onConnect))
	a.L.SetGlobal("on_disconnect", a.L.NewFunction(a.onDisconnect))
	a.L.SetGlobal("on_chat", a.L.NewFunction(a.onChat))
	a.L.SetGlobal("on_block_change", a.L.NewFunction(a.onBlockChange))
	a.L.SetGlobal("on_tick", a.L.NewFunction(a.onTick))
	a.L.SetGlobal("on_command", a.L.NewFunction(a.onCommand))
}

// callFn calls a Lua function with args, recovering from panics.
// Returns nil on error or panic.
func (a *api) callFn(fn *glua.LFunction, nret int, args ...glua.LValue) []glua.LValue {
	defer func() { _ = recover() }()
	L := a.L
	if err := L.CallByParam(glua.P{Fn: fn, NRet: nret, Protect: true}, args...); err != nil {
		return nil
	}
	rets := make([]glua.LValue, nret)
	for i := 0; i < nret; i++ {
		rets[i] = L.Get(-nret + i)
	}
	if nret > 0 {
		L.Pop(nret)
	}
	return rets
}
