// Package lua provides Lua scripting support for Solar plugins.
//
// Build the server with -tags=lua to include this package. Without the
// tag, gopher-lua is excluded from the binary and LoadLuaScripts is a
// no-op that logs a warning.
//
// Each .lua file in the configured directory becomes a plugin with its
// own Lua state. Scripts register callbacks via global functions:
//
//	on_connect(fn)              — fn() on player connect
//	on_disconnect(fn)           — fn() on player disconnect
//	on_chat(fn)                 — fn(name, msg) → false to cancel, string to modify
//	on_block_change(fn)         — fn(name, x, y, z, block, placing) → false to cancel
//	on_tick(fn)                 — fn(tick) every server tick
//	on_command(fn)              — fn(name, cmd, args) → false to cancel
//	register_command(name, help, fn) — fn(name, args_table) → string reply
//	broadcast(msg)              — send to all players
//	colorize(msg)               — apply & color codes
//	strip_color(msg)            — remove & color codes
//	server                      — table: broadcast, find_player, online_count,
//	                              server_name, motd, register_command, stop
//
// See plugins/example.lua for a complete example.
package lua
