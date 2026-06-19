//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// on_connect(fn) — fn() called on player connect.
func (a *api) onConnect(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerConnect.Register(func(_ *plugin.Context, _ plugin.PlayerConnectData) {
		a.callFn(fn, 0)
	}, plugin.PriorityNormal)
	return 0
}

// on_disconnect(fn) — fn() called on player disconnect.
func (a *api) onDisconnect(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerDisconnect.Register(func(_ *plugin.Context, _ plugin.PlayerDisconnectData) {
		a.callFn(fn, 0)
	}, plugin.PriorityNormal)
	return 0
}

// on_chat(fn) — fn(name, msg) → false to cancel, string to modify.
func (a *api) onChat(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerChat.Register(func(ctx *plugin.Context, data plugin.PlayerChatData) {
		rets := a.callFn(fn, 1,
			glua.LString(data.Player.Name()), glua.LString(*data.Message))
		if len(rets) == 0 {
			return
		}
		switch v := rets[0].(type) {
		case glua.LBool:
			if !bool(v) {
				ctx.Cancel()
			}
		case glua.LString:
			*data.Message = string(v)
		}
	}, plugin.PriorityNormal)
	return 0
}

// on_block_change(fn) — fn(name, x, y, z, block, placing) → false to cancel.
func (a *api) onBlockChange(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnBlockChange.Register(func(ctx *plugin.Context, data plugin.BlockChangeData) {
		placing := glua.LFalse
		if data.Placing {
			placing = glua.LTrue
		}
		rets := a.callFn(fn, 1,
			glua.LString(data.Player.Name()),
			glua.LNumber(data.X), glua.LNumber(data.Y), glua.LNumber(data.Z),
			glua.LNumber(data.Block), placing)
		if len(rets) > 0 {
			if b, ok := rets[0].(glua.LBool); ok && !bool(b) {
				ctx.Cancel()
			}
		}
	}, plugin.PriorityNormal)
	return 0
}

// on_tick(fn) — fn(tick) called every server tick.
func (a *api) onTick(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnTick.Register(func(_ *plugin.Context, data plugin.TickData) {
		a.callFn(fn, 0, glua.LNumber(data.Tick))
	}, plugin.PriorityLow)
	return 0
}

// on_command(fn) — fn(name, cmd, args) → false to cancel.
func (a *api) onCommand(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerCommand.Register(func(ctx *plugin.Context, data plugin.PlayerCommandData) {
		rets := a.callFn(fn, 1,
			glua.LString(data.Player.Name()),
			glua.LString(data.Command), glua.LString(data.Args))
		if len(rets) > 0 {
			if b, ok := rets[0].(glua.LBool); ok && !bool(b) {
				ctx.Cancel()
			}
		}
	}, plugin.PriorityNormal)
	return 0
}
