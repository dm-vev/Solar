//go:build lua

package lua

import (
	"time"

	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

const (
	typePlayer        = "solar_player"
	typeCPE           = "solar_cpe"
	typeWorld         = "solar_world"
	typeLevel         = "solar_level"
	typeLevelManager  = "solar_level_manager"
	typeConfig        = "solar_config"
	typePhysics       = "solar_physics"
	typeEntityManager = "solar_entity_manager"
	typeScheduler     = "solar_scheduler"
	typeTask          = "solar_task"
)

// api holds the Lua state, server handle, and closed flag.
// All Lua callbacks check closed before invoking Lua code.
type api struct {
	L      *glua.LState
	srv    plugin.Server
	closed bool
}

func newAPI(L *glua.LState, srv plugin.Server) *api {
	return &api{L: L, srv: srv}
}

// register sets up all Lua globals, type metatables, and constants.
func (a *api) register() {
	a.registerTypes()
	a.registerServerTable()
	a.registerEventGlobals()
	a.registerConstants()
}

func (a *api) registerTypes() {
	a.makeType(typePlayer, playerMethods)
	a.makeType(typeCPE, cpeMethods)
	a.makeType(typeWorld, worldMethods)
	a.makeType(typeLevel, levelMethods)
	a.makeType(typeLevelManager, levelManagerMethods)
	a.makeType(typeConfig, configMethods)
	a.makeType(typePhysics, physicsMethods)
	a.makeType(typeEntityManager, entityManagerMethods)
	a.makeType(typeScheduler, schedulerMethods)
	a.makeType(typeTask, taskMethods)
	a.makeType(typePlayerDB, playerdbMethods)
}

// makeType creates a metatable with __index pointing to the methods map,
// and stores it as a global so wrapUD can retrieve it later.
func (a *api) makeType(name string, methods map[string]glua.LGFunction) {
	mt := a.L.NewTable()
	idx := a.L.SetFuncs(a.L.NewTable(), methods)
	mt.RawSetString("__index", idx)
	a.L.SetGlobal(name, mt)
}

func (a *api) registerConstants() {
	colors := a.L.NewTable()
	colors.RawSetString("black", glua.LString(plugin.ColorBlack))
	colors.RawSetString("navy", glua.LString(plugin.ColorNavy))
	colors.RawSetString("green", glua.LString(plugin.ColorGreen))
	colors.RawSetString("teal", glua.LString(plugin.ColorTeal))
	colors.RawSetString("maroon", glua.LString(plugin.ColorMaroon))
	colors.RawSetString("purple", glua.LString(plugin.ColorPurple))
	colors.RawSetString("gold", glua.LString(plugin.ColorGold))
	colors.RawSetString("silver", glua.LString(plugin.ColorSilver))
	colors.RawSetString("gray", glua.LString(plugin.ColorGray))
	colors.RawSetString("blue", glua.LString(plugin.ColorBlue))
	colors.RawSetString("lime", glua.LString(plugin.ColorLime))
	colors.RawSetString("aqua", glua.LString(plugin.ColorAqua))
	colors.RawSetString("red", glua.LString(plugin.ColorRed))
	colors.RawSetString("pink", glua.LString(plugin.ColorPink))
	colors.RawSetString("yellow", glua.LString(plugin.ColorYellow))
	colors.RawSetString("white", glua.LString(plugin.ColorWhite))
	a.L.SetGlobal("colors", colors)

	physicsModes := a.L.NewTable()
	physicsModes.RawSetString("off", glua.LNumber(plugin.PhysicsOff))
	physicsModes.RawSetString("basic", glua.LNumber(plugin.PhysicsBasic))
	physicsModes.RawSetString("advanced", glua.LNumber(plugin.PhysicsAdvanced))
	physicsModes.RawSetString("custom", glua.LNumber(plugin.PhysicsCustom))
	a.L.SetGlobal("physics_mode", physicsModes)
}

// ─── helpers ───

// wrapUD creates a userdata value wrapping a Go value with a type metatable.
func wrapUD(L *glua.LState, typeName string, value any) *glua.LUserData {
	ud := L.NewUserData()
	ud.Value = value
	L.SetMetatable(ud, L.GetGlobal(typeName))
	return ud
}

// udValue extracts the Go value from a userdata at the given stack index.
func udValue(L *glua.LState, idx int) any {
	ud := L.CheckUserData(idx)
	if ud == nil {
		return nil
	}
	return ud.Value
}

// callFn calls a Lua function with args, recovering from panics.
// Returns nil on error or if the api is closed.
func (a *api) callFn(fn *glua.LFunction, nret int, args ...glua.LValue) []glua.LValue {
	if a.closed {
		return nil
	}
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

// cbCancel calls fn with args, returns true if the callback returned false.
func (a *api) cbCancel(fn *glua.LFunction, args ...glua.LValue) bool {
	rets := a.callFn(fn, 1, args...)
	if len(rets) == 0 {
		return false
	}
	if b, ok := rets[0].(glua.LBool); ok {
		return !bool(b)
	}
	return false
}

// cbString calls fn with args (last arg is a string), returns cancelled + modified.
// If the callback returns false → cancelled. If it returns a string → modified.
func (a *api) cbString(fn *glua.LFunction, msg *string, prefix ...glua.LValue) (cancelled bool) {
	args := append(prefix, glua.LString(*msg))
	rets := a.callFn(fn, 1, args...)
	if len(rets) == 0 {
		return false
	}
	switch v := rets[0].(type) {
	case glua.LBool:
		if !bool(v) {
			return true
		}
	case glua.LString:
		*msg = string(v)
	}
	return false
}

// durationFromMS converts a Lua number (milliseconds) to time.Duration.
func durationFromMS(L *glua.LState, idx int) time.Duration {
	return time.Duration(L.CheckInt(idx)) * time.Millisecond
}
