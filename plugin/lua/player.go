//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// checkPlayer extracts plugin.Player from userdata at idx.
func checkPlayer(L *glua.LState, idx int) plugin.Player {
	v := udValue(L, idx)
	if p, ok := v.(plugin.Player); ok {
		return p
	}
	L.ArgError(idx, "expected player")
	return nil
}

// wrapPlayer wraps a plugin.Player in userdata with the player metatable.
func wrapPlayer(L *glua.LState, p plugin.Player) glua.LValue {
	if p == nil {
		return glua.LNil
	}
	return wrapUD(L, typePlayer, p)
}

var playerMethods = map[string]glua.LGFunction{
	// name
	"name": func(L *glua.LState) int {
		L.Push(glua.LString(checkPlayer(L, 1).Name()))
		return 1
	},

	// message(msg)
	"message": func(L *glua.LState) int {
		checkPlayer(L, 1).Message(L.CheckString(2))
		return 0
	},

	// teleport(x, y, z, yaw, pitch) -> bool
	"teleport": func(L *glua.LState) int {
		p := checkPlayer(L, 1)
		ok := p.Teleport(L.CheckInt(2), L.CheckInt(3), L.CheckInt(4),
			byte(L.CheckInt(5)), byte(L.CheckInt(6)))
		L.Push(glua.LBool(ok))
		return 1
	},

	// kick(reason)
	"kick": func(L *glua.LState) int {
		checkPlayer(L, 1).Kick(L.CheckString(2))
		return 0
	},

	// is_operator() -> bool
	"is_operator": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).IsOperator()))
		return 1
	},

	// position() -> x, y, z
	"position": func(L *glua.LState) int {
		x, y, z := checkPlayer(L, 1).Position()
		L.Push(glua.LNumber(x))
		L.Push(glua.LNumber(y))
		L.Push(glua.LNumber(z))
		return 3
	},

	// set_block(x, y, z, block) -> bool
	"set_block": func(L *glua.LState) int {
		ok := checkPlayer(L, 1).SetBlock(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4), byte(L.CheckInt(5)))
		L.Push(glua.LBool(ok))
		return 1
	},

	// change_block(x, y, z, block) -> bool
	"change_block": func(L *glua.LState) int {
		ok := checkPlayer(L, 1).ChangeBlock(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4), byte(L.CheckInt(5)))
		L.Push(glua.LBool(ok))
		return 1
	},

	// revert_block(x, y, z)
	"revert_block": func(L *glua.LState) int {
		checkPlayer(L, 1).RevertBlock(L.CheckInt(2), L.CheckInt(3), L.CheckInt(4))
		return 0
	},

	// send_block_change(x, y, z, block)
	"send_block_change": func(L *glua.LState) int {
		checkPlayer(L, 1).SendBlockChange(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4), byte(L.CheckInt(5)))
		return 0
	},

	// send_raw_packet(data)
	"send_raw_packet": func(L *glua.LState) int {
		checkPlayer(L, 1).SendRawPacket([]byte(L.CheckString(2)))
		return 0
	},

	// supports_cpe(ext_name) -> bool
	"supports_cpe": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).SupportsCPE(L.CheckString(2))))
		return 1
	},

	// send_cpe_message(type, msg)
	"send_cpe_message": func(L *glua.LState) int {
		checkPlayer(L, 1).SendCpeMessage(byte(L.CheckInt(2)), L.CheckString(3))
		return 0
	},

	// color() -> string
	"color": func(L *glua.LState) int {
		L.Push(glua.LString(checkPlayer(L, 1).Color()))
		return 1
	},

	// set_color(color_str)
	"set_color": func(L *glua.LState) int {
		checkPlayer(L, 1).SetColor(L.CheckString(2))
		return 0
	},

	// model() -> string
	"model": func(L *glua.LState) int {
		L.Push(glua.LString(checkPlayer(L, 1).Model()))
		return 1
	},

	// set_model(model)
	"set_model": func(L *glua.LState) int {
		checkPlayer(L, 1).SetModel(L.CheckString(2))
		return 0
	},

	// set_skin(url)
	"set_skin": func(L *glua.LState) int {
		checkPlayer(L, 1).SetSkin(L.CheckString(2))
		return 0
	},

	// is_hidden() -> bool
	"is_hidden": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).IsHidden()))
		return 1
	},

	// set_hidden(bool)
	"set_hidden": func(L *glua.LState) int {
		checkPlayer(L, 1).SetHidden(L.CheckBool(2))
		return 0
	},

	// is_muted() -> bool
	"is_muted": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).IsMuted()))
		return 1
	},

	// set_muted(bool)
	"set_muted": func(L *glua.LState) int {
		checkPlayer(L, 1).SetMuted(L.CheckBool(2))
		return 0
	},

	// is_frozen() -> bool
	"is_frozen": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).IsFrozen()))
		return 1
	},

	// set_frozen(bool)
	"set_frozen": func(L *glua.LState) int {
		checkPlayer(L, 1).SetFrozen(L.CheckBool(2))
		return 0
	},

	// is_afk() -> bool
	"is_afk": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).IsAfk()))
		return 1
	},

	// set_afk(bool)
	"set_afk": func(L *glua.LState) int {
		checkPlayer(L, 1).SetAfk(L.CheckBool(2))
		return 0
	},

	// kill(cause) -> bool
	"kill": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).Kill(byte(L.CheckInt(2)))))
		return 1
	},

	// respawn()
	"respawn": func(L *glua.LState) int {
		checkPlayer(L, 1).Respawn()
		return 0
	},

	// allow_build() -> bool
	"allow_build": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).AllowBuild()))
		return 1
	},

	// set_allow_build(bool)
	"set_allow_build": func(L *glua.LState) int {
		checkPlayer(L, 1).SetAllowBuild(L.CheckBool(2))
		return 0
	},

	// entity_id() -> number
	"entity_id": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkPlayer(L, 1).EntityID()))
		return 1
	},

	// ip() -> string
	"ip": func(L *glua.LState) int {
		L.Push(glua.LString(checkPlayer(L, 1).IP()))
		return 1
	},

	// yaw() -> number
	"yaw": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkPlayer(L, 1).Yaw()))
		return 1
	},

	// pitch() -> number
	"pitch": func(L *glua.LState) int {
		L.Push(glua.LNumber(checkPlayer(L, 1).Pitch()))
		return 1
	},

	// make_selection(id, label, minx, miny, minz, maxx, maxy, maxz, r, g, b) -> bool
	"make_selection": func(L *glua.LState) int {
		ok := checkPlayer(L, 1).MakeSelection(
			byte(L.CheckInt(2)), L.CheckString(3),
			L.CheckInt(4), L.CheckInt(5), L.CheckInt(6),
			L.CheckInt(7), L.CheckInt(8), L.CheckInt(9),
			byte(L.CheckInt(10)), byte(L.CheckInt(11)), byte(L.CheckInt(12)))
		L.Push(glua.LBool(ok))
		return 1
	},

	// clear_selection(id) -> bool
	"clear_selection": func(L *glua.LState) int {
		L.Push(glua.LBool(checkPlayer(L, 1).ClearSelection(byte(L.CheckInt(2)))))
		return 1
	},

	// cpe() -> cpe table
	"cpe": func(L *glua.LState) int {
		c := checkPlayer(L, 1).CPE()
		if c == nil {
			L.Push(glua.LNil)
			return 1
		}
		L.Push(wrapUD(L, typeCPE, c))
		return 1
	},
}
