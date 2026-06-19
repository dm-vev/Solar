//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

func checkPhysics(L *glua.LState, idx int) plugin.Physics {
	if v, ok := udValue(L, idx).(plugin.Physics); ok {
		return v
	}
	L.ArgError(idx, "expected physics")
	return nil
}

var physicsMethods = map[string]glua.LGFunction{
	// mode() -> number
	"mode": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkPhysics(L, 1).Mode()))
		return 1
	},
	// set_mode(mode)
	"set_mode": func(L *glua.LState) int {
		checkPhysics(L, 1).SetMode(plugin.PhysicsMode(L.CheckInt(2)))
		return 0
	},
	// schedule(x, y, z)
	"schedule": func(L *glua.LState) int {
		checkPhysics(L, 1).Schedule(L.CheckInt(2), L.CheckInt(3), L.CheckInt(4))
		return 0
	},
	// register_handler(fn) — fn(x, y, z, block, level) -> bool (false = remove)
	"register_handler": func(L *glua.LState) int {
		ph := checkPhysics(L, 1)
		fn := L.CheckFunction(2)
		ph.RegisterHandler(func(pb plugin.PhysicsBlock) bool {
			// ponytail: can't access api from static method map; use closure capture
			// The LState is accessible via L (the state that called this)
			// but we need the *api for callFn. Store it on the LState registry.
			// Simpler: just call directly with panic recovery.
			defer func() { _ = recover() }()
			if err := L.CallByParam(glua.P{
				Fn:      fn,
				NRet:    1,
				Protect: true,
			}, glua.LNumber(pb.X), glua.LNumber(pb.Y), glua.LNumber(pb.Z),
				glua.LNumber(pb.Block), glua.LString(pb.Level)); err != nil {
				return true
			}
			ret := L.Get(-1)
			L.Pop(1)
			if b, ok := ret.(glua.LBool); ok {
				return bool(b)
			}
			return true
		})
		return 0
	},
}
