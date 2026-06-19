//go:build lua

package plugin

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

// registerLuaAPI exposes the plugin API to a Lua state.
//
// Lua globals available to scripts:
//
//	server                          — table: broadcast, find_player, online_count,
//	                                  server_name, motd, register_command, stop
//	broadcast(msg)                  — shorthand for server.broadcast
//	colorize(msg)                   — applies & color codes
//	strip_color(msg)                — removes & color codes
//	on_connect(fn)                  — fn() called on player connect
//	on_disconnect(fn)               — fn() called on player disconnect
//	on_chat(fn)                     — fn(name, msg) -> return false to cancel, return string to modify
//	on_block_change(fn)             — fn(name, x, y, z, block, placing) -> return false to cancel
//	on_tick(fn)                     — fn(tick) called every server tick
//	on_command(fn)                  — fn(name, cmd, args) -> return false to cancel
//	register_command(name, help, fn)— fn(name, args_table) -> return string reply
type luaServer struct {
	srv Server
	L   *lua.LState
}

func registerLuaAPI(L *lua.LState, srv Server) {
	ls := &luaServer{srv: srv, L: L}

	srvTbl := L.NewTable()
	L.SetField(srvTbl, "broadcast", L.NewFunction(ls.luaBroadcast))
	L.SetField(srvTbl, "find_player", L.NewFunction(ls.luaFindPlayer))
	L.SetField(srvTbl, "online_count", L.NewFunction(ls.luaOnlineCount))
	L.SetField(srvTbl, "server_name", L.NewFunction(ls.luaServerName))
	L.SetField(srvTbl, "motd", L.NewFunction(ls.luaMotd))
	L.SetField(srvTbl, "register_command", L.NewFunction(ls.luaRegisterCommand))
	L.SetField(srvTbl, "stop", L.NewFunction(ls.luaStop))
	L.SetGlobal("server", srvTbl)

	L.SetGlobal("broadcast", L.NewFunction(ls.luaBroadcast))
	L.SetGlobal("colorize", L.NewFunction(ls.luaColorize))
	L.SetGlobal("strip_color", L.NewFunction(ls.luaStripColor))

	L.SetGlobal("on_connect", L.NewFunction(ls.onConnect))
	L.SetGlobal("on_disconnect", L.NewFunction(ls.onDisconnect))
	L.SetGlobal("on_chat", L.NewFunction(ls.onChat))
	L.SetGlobal("on_block_change", L.NewFunction(ls.onBlockChange))
	L.SetGlobal("on_tick", L.NewFunction(ls.onTick))
	L.SetGlobal("on_command", L.NewFunction(ls.onCommand))
}

// callFn calls a Lua function with args, discarding panic.
func (ls *luaServer) callFn(fn *lua.LFunction, nret int, args ...lua.LValue) []lua.LValue {
	defer func() { _ = recover() }()
	L := ls.L
	if err := L.CallByParam(lua.P{Fn: fn, NRet: nret, Protect: true}, args...); err != nil {
		return nil
	}
	rets := make([]lua.LValue, nret)
	for i := 0; i < nret; i++ {
		rets[i] = L.Get(-nret + i)
	}
	if nret > 0 {
		L.Pop(nret)
	}
	return rets
}

func (ls *luaServer) onConnect(L *lua.LState) int {
	fn := L.CheckFunction(1)
	OnPlayerConnect.Register(func(_ *Context, _ PlayerConnectData) {
		ls.callFn(fn, 0)
	}, PriorityNormal)
	return 0
}

func (ls *luaServer) onDisconnect(L *lua.LState) int {
	fn := L.CheckFunction(1)
	OnPlayerDisconnect.Register(func(_ *Context, _ PlayerDisconnectData) {
		ls.callFn(fn, 0)
	}, PriorityNormal)
	return 0
}

func (ls *luaServer) onChat(L *lua.LState) int {
	fn := L.CheckFunction(1)
	OnPlayerChat.Register(func(ctx *Context, data PlayerChatData) {
		rets := ls.callFn(fn, 1,
			lua.LString(data.Player.Name()), lua.LString(*data.Message))
		if len(rets) == 0 {
			return
		}
		switch v := rets[0].(type) {
		case lua.LBool:
			if !bool(v) {
				ctx.Cancel()
			}
		case lua.LString:
			*data.Message = string(v)
		}
	}, PriorityNormal)
	return 0
}

func (ls *luaServer) onBlockChange(L *lua.LState) int {
	fn := L.CheckFunction(1)
	OnBlockChange.Register(func(ctx *Context, data BlockChangeData) {
		placing := lua.LFalse
		if data.Placing {
			placing = lua.LTrue
		}
		rets := ls.callFn(fn, 1,
			lua.LString(data.Player.Name()),
			lua.LNumber(data.X), lua.LNumber(data.Y), lua.LNumber(data.Z),
			lua.LNumber(data.Block), placing)
		if len(rets) > 0 {
			if b, ok := rets[0].(lua.LBool); ok && !bool(b) {
				ctx.Cancel()
			}
		}
	}, PriorityNormal)
	return 0
}

func (ls *luaServer) onTick(L *lua.LState) int {
	fn := L.CheckFunction(1)
	OnTick.Register(func(_ *Context, data TickData) {
		ls.callFn(fn, 0, lua.LNumber(data.Tick))
	}, PriorityLow)
	return 0
}

func (ls *luaServer) onCommand(L *lua.LState) int {
	fn := L.CheckFunction(1)
	OnPlayerCommand.Register(func(ctx *Context, data PlayerCommandData) {
		rets := ls.callFn(fn, 1,
			lua.LString(data.Player.Name()),
			lua.LString(data.Command), lua.LString(data.Args))
		if len(rets) > 0 {
			if b, ok := rets[0].(lua.LBool); ok && !bool(b) {
				ctx.Cancel()
			}
		}
	}, PriorityNormal)
	return 0
}

// ─── server table methods ───

func (ls *luaServer) luaBroadcast(L *lua.LState) int {
	ls.srv.BroadcastMessage(L.CheckString(1))
	return 0
}

func (ls *luaServer) luaFindPlayer(L *lua.LState) int {
	p := ls.srv.FindPlayer(L.CheckString(1))
	if p == nil {
		L.Push(lua.LNil)
	} else {
		L.Push(ls.playerValue(p))
	}
	return 1
}

func (ls *luaServer) luaOnlineCount(L *lua.LState) int {
	L.Push(lua.LNumber(ls.srv.OnlineCount()))
	return 1
}

func (ls *luaServer) luaServerName(L *lua.LState) int {
	L.Push(lua.LString(ls.srv.ServerName()))
	return 1
}

func (ls *luaServer) luaMotd(L *lua.LState) int {
	L.Push(lua.LString(ls.srv.MOTD()))
	return 1
}

func (ls *luaServer) luaStop(L *lua.LState) int {
	ls.srv.Stop()
	return 0
}

func (ls *luaServer) luaColorize(L *lua.LState) int {
	L.Push(lua.LString(Colorize(ColorWhite, L.CheckString(1))))
	return 1
}

func (ls *luaServer) luaStripColor(L *lua.LState) int {
	L.Push(lua.LString(StripColor(L.CheckString(1))))
	return 1
}

func (ls *luaServer) luaRegisterCommand(L *lua.LState) int {
	name := L.CheckString(1)
	help := L.CheckString(2)
	fn := L.CheckFunction(3)
	ok := ls.srv.RegisterCommand(name, help, func(p Player, args []string) string {
		defer func() {
			if r := recover(); r != nil {
				ls.L.RaiseError("lua command %s panicked: %v", name, r)
			}
		}()
		argsTbl := ls.L.NewTable()
		for i, a := range args {
			ls.L.SetField(argsTbl, fmt.Sprintf("%d", i+1), lua.LString(a))
		}
		rets := ls.callFn(fn, 1, lua.LString(p.Name()), argsTbl)
		if len(rets) == 0 {
			return ""
		}
		if s, ok := rets[0].(lua.LString); ok {
			return string(s)
		}
		return ""
	})
	L.Push(lua.LBool(ok))
	return 1
}

// playerValue wraps a Player in a Lua table with methods.
func (ls *luaServer) playerValue(p Player) lua.LValue {
	tbl := ls.L.NewTable()
	ls.L.SetField(tbl, "name", lua.LString(p.Name()))
	ls.L.SetField(tbl, "message", ls.L.NewFunction(func(L *lua.LState) int {
		p.Message(L.CheckString(1))
		return 0
	}))
	ls.L.SetField(tbl, "kick", ls.L.NewFunction(func(L *lua.LState) int {
		p.Kick(L.CheckString(1))
		return 0
	}))
	ls.L.SetField(tbl, "is_op", ls.L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LBool(p.IsOperator()))
		return 1
	}))
	ls.L.SetField(tbl, "position", ls.L.NewFunction(func(L *lua.LState) int {
		x, y, z := p.Position()
		L.Push(lua.LNumber(x))
		L.Push(lua.LNumber(y))
		L.Push(lua.LNumber(z))
		return 3
	}))
	return tbl
}
