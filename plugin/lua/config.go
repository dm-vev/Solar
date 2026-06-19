//go:build lua

package lua

import (
	"time"

	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

func checkConfig(L *glua.LState, idx int) plugin.Config {
	if v, ok := udValue(L, idx).(plugin.Config); ok {
		return v
	}
	L.ArgError(idx, "expected config")
	return nil
}

// cfgGetStr/cfgGetNum/cfgGetBool are helpers for config getters.
func cfgGetStr(fn func(plugin.Config) string) glua.LGFunction {
	return func(L *glua.LState) int {
		L.Push(glua.LString(fn(checkConfig(L, 1))))
		return 1
	}
}

func cfgGetNum(fn func(plugin.Config) int) glua.LGFunction {
	return func(L *glua.LState) int {
		L.Push(glua.LNumber(fn(checkConfig(L, 1))))
		return 1
	}
}

func cfgGetDur(fn func(plugin.Config) time.Duration) glua.LGFunction {
	return func(L *glua.LState) int {
		L.Push(glua.LNumber(fn(checkConfig(L, 1)) / time.Millisecond))
		return 1
	}
}

func cfgGetBool(fn func(plugin.Config) bool) glua.LGFunction {
	return func(L *glua.LState) int {
		L.Push(glua.LBool(fn(checkConfig(L, 1))))
		return 1
	}
}

func cfgGetStrs(fn func(plugin.Config) []string) glua.LGFunction {
	return func(L *glua.LState) int {
		tbl := L.NewTable()
		for i, s := range fn(checkConfig(L, 1)) {
			tbl.RawSetInt(i+1, glua.LString(s))
		}
		L.Push(tbl)
		return 1
	}
}

var configMethods = map[string]glua.LGFunction{
	"name":              cfgGetStr(plugin.Config.Name),
	"motd":              cfgGetStr(plugin.Config.MOTD),
	"max_players":       cfgGetNum(plugin.Config.MaxPlayers),
	"connect_rate":      cfgGetNum(plugin.Config.ConnectRate),
	"read_timeout":      cfgGetDur(plugin.Config.ReadTimeout),
	"write_timeout":     cfgGetDur(plugin.Config.WriteTimeout),
	"tcp_nodelay":       cfgGetBool(plugin.Config.TCPNoDelay),
	"tick_interval":     cfgGetDur(plugin.Config.TickInterval),
	"default_width":     cfgGetNum(plugin.Config.DefaultWidth),
	"default_height":    cfgGetNum(plugin.Config.DefaultHeight),
	"default_length":    cfgGetNum(plugin.Config.DefaultLength),
	"max_blocks":        cfgGetNum(plugin.Config.MaxBlocks),
	"autosave_interval": cfgGetDur(plugin.Config.AutosaveInterval),
	"operators":         cfgGetStrs(plugin.Config.Operators),
	"whitelist_enabled": cfgGetBool(plugin.Config.WhitelistEnabled),

	// setters
	"set_name": func(L *glua.LState) int {
		checkConfig(L, 1).SetName(L.CheckString(2))
		return 0
	},
	"set_motd": func(L *glua.LState) int {
		checkConfig(L, 1).SetMOTD(L.CheckString(2))
		return 0
	},
	"set_max_players": func(L *glua.LState) int {
		checkConfig(L, 1).SetMaxPlayers(L.CheckInt(2))
		return 0
	},
	"set_connect_rate": func(L *glua.LState) int {
		checkConfig(L, 1).SetConnectRate(L.CheckInt(2))
		return 0
	},
	"set_tcp_nodelay": func(L *glua.LState) int {
		checkConfig(L, 1).SetTCPNoDelay(L.CheckBool(2))
		return 0
	},
	"set_tick_interval": func(L *glua.LState) int {
		checkConfig(L, 1).SetTickInterval(durationFromMS(L, 2))
		return 0
	},
	"set_autosave_interval": func(L *glua.LState) int {
		checkConfig(L, 1).SetAutosaveInterval(durationFromMS(L, 2))
		return 0
	},
	"set_whitelist_enabled": func(L *glua.LState) int {
		checkConfig(L, 1).SetWhitelistEnabled(L.CheckBool(2))
		return 0
	},
	"add_operator": func(L *glua.LState) int {
		L.Push(glua.LBool(checkConfig(L, 1).AddOperator(L.CheckString(2))))
		return 1
	},
	"remove_operator": func(L *glua.LState) int {
		L.Push(glua.LBool(checkConfig(L, 1).RemoveOperator(L.CheckString(2))))
		return 1
	},
}
