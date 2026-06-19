//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

func checkCPE(L *glua.LState, idx int) plugin.CPE {
	if v, ok := udValue(L, idx).(plugin.CPE); ok {
		return v
	}
	L.ArgError(idx, "expected cpe")
	return nil
}

var cpeMethods = map[string]glua.LGFunction{
	// set_env_color(slot, r, g, b)
	"set_env_color": func(L *glua.LState) int {
		checkCPE(L, 1).SetEnvColor(byte(L.CheckInt(2)),
			byte(L.CheckInt(3)), byte(L.CheckInt(4)), byte(L.CheckInt(5)))
		return 0
	},

	// set_weather(weather)
	"set_weather": func(L *glua.LState) int {
		checkCPE(L, 1).SetWeather(byte(L.CheckInt(2)))
		return 0
	},

	// set_hack_control(flying, noclip, speed, respawn_height, third_person)
	"set_hack_control": func(L *glua.LState) int {
		checkCPE(L, 1).SetHackControl(
			L.CheckBool(2), L.CheckBool(3), L.CheckBool(4),
			L.CheckBool(5), L.CheckBool(6))
		return 0
	},

	// set_click_distance(distance)
	"set_click_distance": func(L *glua.LState) int {
		checkCPE(L, 1).SetClickDistance(float64(L.CheckNumber(2)))
		return 0
	},

	// set_text_hotkey(key, action, flags)
	"set_text_hotkey": func(L *glua.LState) int {
		checkCPE(L, 1).SetTextHotkey(byte(L.CheckInt(2)), L.CheckString(3), byte(L.CheckInt(4)))
		return 0
	},

	// hold_this(block, can_change)
	"hold_this": func(L *glua.LState) int {
		checkCPE(L, 1).HoldThis(byte(L.CheckInt(2)), L.CheckBool(3))
		return 0
	},

	// set_block_permission(block, allow_place, allow_delete)
	"set_block_permission": func(L *glua.LState) int {
		checkCPE(L, 1).SetBlockPermission(byte(L.CheckInt(2)),
			L.CheckBool(3), L.CheckBool(4))
		return 0
	},

	// change_model(entity_id, model)
	"change_model": func(L *glua.LState) int {
		checkCPE(L, 1).ChangeModel(byte(L.CheckInt(2)), L.CheckString(3))
		return 0
	},

	// set_map_appearance(url, side_level)
	"set_map_appearance": func(L *glua.LState) int {
		checkCPE(L, 1).SetMapAppearance(L.CheckString(2), byte(L.CheckInt(3)))
		return 0
	},

	// set_inventory_order(block, order)
	"set_inventory_order": func(L *glua.LState) int {
		checkCPE(L, 1).SetInventoryOrder(byte(L.CheckInt(2)), byte(L.CheckInt(3)))
		return 0
	},

	// set_hotbar(slot, block)
	"set_hotbar": func(L *glua.LState) int {
		checkCPE(L, 1).SetHotbar(byte(L.CheckInt(2)), byte(L.CheckInt(3)))
		return 0
	},

	// set_spawnpoint(x, y, z, yaw, pitch)
	"set_spawnpoint": func(L *glua.LState) int {
		checkCPE(L, 1).SetSpawnpoint(
			L.CheckInt(2), L.CheckInt(3), L.CheckInt(4),
			byte(L.CheckInt(5)), byte(L.CheckInt(6)))
		return 0
	},

	// send_effect(effect_id, x, y, z)
	"send_effect": func(L *glua.LState) int {
		checkCPE(L, 1).SendEffect(
			byte(L.CheckInt(2)), byte(L.CheckInt(3)),
			byte(L.CheckInt(4)), byte(L.CheckInt(5)))
		return 0
	},

	// send_selection(id, label, minx, miny, minz, maxx, maxy, maxz, r, g, b)
	"send_selection": func(L *glua.LState) int {
		checkCPE(L, 1).SendSelection(
			byte(L.CheckInt(2)), L.CheckString(3),
			L.CheckInt(4), L.CheckInt(5), L.CheckInt(6),
			L.CheckInt(7), L.CheckInt(8), L.CheckInt(9),
			byte(L.CheckInt(10)), byte(L.CheckInt(11)), byte(L.CheckInt(12)))
		return 0
	},

	// remove_selection(id)
	"remove_selection": func(L *glua.LState) int {
		checkCPE(L, 1).RemoveSelection(byte(L.CheckInt(2)))
		return 0
	},

	// set_map_property(property, value)
	"set_map_property": func(L *glua.LState) int {
		checkCPE(L, 1).SetMapProperty(byte(L.CheckInt(2)), L.CheckInt(3))
		return 0
	},

	// set_entity_property(entity_id, property, value)
	"set_entity_property": func(L *glua.LState) int {
		checkCPE(L, 1).SetEntityProperty(
			byte(L.CheckInt(2)), byte(L.CheckInt(3)), L.CheckInt(4))
		return 0
	},

	// send_plugin_message(channel, data)
	"send_plugin_message": func(L *glua.LState) int {
		checkCPE(L, 1).SendPluginMessage(byte(L.CheckInt(2)), []byte(L.CheckString(3)))
		return 0
	},

	// set_lighting_mode(mode, locked)
	"set_lighting_mode": func(L *glua.LState) int {
		checkCPE(L, 1).SetLightingMode(byte(L.CheckInt(2)), L.CheckBool(3))
		return 0
	},
}
