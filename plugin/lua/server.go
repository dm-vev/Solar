//go:build lua

package lua

import (
	"fmt"

	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// registerServerTable creates the `server` global table with all 28 Server methods.
func (a *api) registerServerTable() {
	srvTbl := a.L.NewTable()

	methods := map[string]glua.LGFunction{
		// broadcast(msg)
		"broadcast": a.serverBroadcast,
		// broadcast_to(scope, source_player, msg)
		"broadcast_to": a.serverBroadcastTo,
		// online_players() -> table of players
		"online_players": a.serverOnlinePlayers,
		// online_count() -> number
		"online_count": a.serverOnlineCount,
		// find_player(name) -> player or nil
		"find_player": a.serverFindPlayer,
		// world() -> world
		"world": a.serverWorld,
		// max_players() -> number
		"max_players": a.serverMaxPlayers,
		// server_name() -> string
		"server_name": a.serverServerName,
		// motd() -> string
		"motd": a.serverMotd,
		// register_command(name, help, fn) -> bool
		"register_command": a.serverRegisterCommand,
		// unregister_command(name) -> bool
		"unregister_command": a.serverUnregisterCommand,
		// ban_player(name, reason) -> bool
		"ban_player": a.serverBanPlayer,
		// unban_player(name) -> bool
		"unban_player": a.serverUnbanPlayer,
		// is_whitelist_enabled() -> bool
		"is_whitelist_enabled": a.serverIsWhitelistEnabled,
		// set_whitelist_enabled(bool)
		"set_whitelist_enabled": a.serverSetWhitelistEnabled,
		// whitelist_add(name) -> bool
		"whitelist_add": a.serverWhitelistAdd,
		// whitelist_remove(name) -> bool
		"whitelist_remove": a.serverWhitelistRemove,
		// is_operator(name) -> bool
		"is_operator": a.serverIsOperator,
		// add_operators(names...) -> bool
		"add_operators": a.serverAddOperators,
		// operator_names() -> table
		"operator_names": a.serverOperatorNames,
		// save_state() -> bool
		"save_state": a.serverSaveState,
		// stop()
		"stop": a.serverStop,
		// levels() -> level_manager
		"levels": a.serverLevels,
		// change_map(player, level_name) -> bool
		"change_map": a.serverChangeMap,
		// physics() -> physics
		"physics": a.serverPhysics,
		// entities() -> entity_manager
		"entities": a.serverEntities,
		// config() -> config
		"config": a.serverConfig,
		// scheduler() -> scheduler
		"scheduler": a.serverScheduler,
	}

	a.L.SetField(srvTbl, "__index", a.L.SetFuncs(a.L.NewTable(), methods))
	mt := a.L.NewTable()
	a.L.SetField(mt, "__index", a.L.SetFuncs(a.L.NewTable(), methods))
	a.L.SetMetatable(srvTbl, mt)
	a.L.SetGlobal("server", srvTbl)

	// Global shortcuts
	a.L.SetGlobal("broadcast", a.L.NewFunction(a.serverBroadcast))
	a.L.SetGlobal("colorize", a.L.NewFunction(a.utilColorize))
	a.L.SetGlobal("strip_color", a.L.NewFunction(a.utilStripColor))
}

// ─── server methods ───

func (a *api) serverBroadcast(L *glua.LState) int {
	a.srv.BroadcastMessage(L.CheckString(1))
	return 0
}

func (a *api) serverBroadcastTo(L *glua.LState) int {
	scope := L.CheckString(1)
	source := checkPlayer(L, 2)
	msg := L.CheckString(3)
	a.srv.BroadcastMessageTo(scope, source, msg)
	return 0
}

func (a *api) serverOnlinePlayers(L *glua.LState) int {
	tbl := L.NewTable()
	for i, p := range a.srv.OnlinePlayers() {
		tbl.RawSetInt(i+1, wrapPlayer(L, p))
	}
	L.Push(tbl)
	return 1
}

func (a *api) serverOnlineCount(L *glua.LState) int {
	L.Push(glua.LNumber(a.srv.OnlineCount()))
	return 1
}

func (a *api) serverFindPlayer(L *glua.LState) int {
	L.Push(wrapPlayer(L, a.srv.FindPlayer(L.CheckString(1))))
	return 1
}

func (a *api) serverWorld(L *glua.LState) int {
	L.Push(wrapWorld(L, a.srv.World()))
	return 1
}

func (a *api) serverMaxPlayers(L *glua.LState) int {
	L.Push(glua.LNumber(a.srv.MaxPlayers()))
	return 1
}

func (a *api) serverServerName(L *glua.LState) int {
	L.Push(glua.LString(a.srv.ServerName()))
	return 1
}

func (a *api) serverMotd(L *glua.LState) int {
	L.Push(glua.LString(a.srv.MOTD()))
	return 1
}

func (a *api) serverRegisterCommand(L *glua.LState) int {
	name := L.CheckString(1)
	help := L.CheckString(2)
	fn := L.CheckFunction(3)
	ok := a.srv.RegisterCommand(name, help, func(p plugin.Player, args []string) string {
		if a.closed {
			return ""
		}
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

func (a *api) serverUnregisterCommand(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.UnregisterCommand(L.CheckString(1))))
	return 1
}

func (a *api) serverBanPlayer(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.BanPlayer(L.CheckString(1), L.CheckString(2))))
	return 1
}

func (a *api) serverUnbanPlayer(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.UnbanPlayer(L.CheckString(1))))
	return 1
}

func (a *api) serverIsWhitelistEnabled(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.IsWhitelistEnabled()))
	return 1
}

func (a *api) serverSetWhitelistEnabled(L *glua.LState) int {
	a.srv.SetWhitelistEnabled(L.CheckBool(1))
	return 0
}

func (a *api) serverWhitelistAdd(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.WhitelistAdd(L.CheckString(1))))
	return 1
}

func (a *api) serverWhitelistRemove(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.WhitelistRemove(L.CheckString(1))))
	return 1
}

func (a *api) serverIsOperator(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.IsOperator(L.CheckString(1))))
	return 1
}

func (a *api) serverAddOperators(L *glua.LState) int {
	var names []string
	for i := 1; i <= L.GetTop(); i++ {
		names = append(names, L.CheckString(i))
	}
	L.Push(glua.LBool(a.srv.AddOperators(names...)))
	return 1
}

func (a *api) serverOperatorNames(L *glua.LState) int {
	tbl := L.NewTable()
	for i, name := range a.srv.OperatorNames() {
		tbl.RawSetInt(i+1, glua.LString(name))
	}
	L.Push(tbl)
	return 1
}

func (a *api) serverSaveState(L *glua.LState) int {
	L.Push(glua.LBool(a.srv.SaveState()))
	return 1
}

func (a *api) serverStop(L *glua.LState) int {
	a.srv.Stop()
	return 0
}

func (a *api) serverLevels(L *glua.LState) int {
	L.Push(wrapLevelManager(L, a.srv.Levels()))
	return 1
}

func (a *api) serverChangeMap(L *glua.LState) int {
	p := checkPlayer(L, 1)
	L.Push(glua.LBool(a.srv.ChangeMap(p, L.CheckString(2))))
	return 1
}

func (a *api) serverPhysics(L *glua.LState) int {
	L.Push(wrapUD(L, typePhysics, a.srv.Physics()))
	return 1
}

func (a *api) serverEntities(L *glua.LState) int {
	L.Push(wrapUD(L, typeEntityManager, a.srv.Entities()))
	return 1
}

func (a *api) serverConfig(L *glua.LState) int {
	L.Push(wrapUD(L, typeConfig, a.srv.Config()))
	return 1
}

func (a *api) serverScheduler(L *glua.LState) int {
	L.Push(wrapUD(L, typeScheduler, a.srv.Scheduler()))
	return 1
}

func (a *api) utilColorize(L *glua.LState) int {
	L.Push(glua.LString(plugin.Colorize(plugin.ColorWhite, L.CheckString(1))))
	return 1
}

func (a *api) utilStripColor(L *glua.LState) int {
	L.Push(glua.LString(plugin.StripColor(L.CheckString(1))))
	return 1
}
