//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// registerEventGlobals sets up all on_* event registration functions.
func (a *api) registerEventGlobals() {
	globals := map[string]glua.LGFunction{
		// ─── player events ───
		"on_connect":                a.onConnect,
		"on_disconnect":             a.onDisconnect,
		"on_start_connecting":       a.onStartConnecting,
		"on_finish_connecting":      a.onFinishConnecting,
		"on_chat":                   a.onChat,
		"on_message_received":       a.onMessageReceived,
		"on_move":                   a.onMove,
		"on_command":                a.onCommand,
		"on_help":                   a.onHelp,
		"on_player_action":          a.onPlayerAction,
		"on_dying":                  a.onDying,
		"on_died":                   a.onDied,
		"on_block_change":           a.onBlockChange,
		"on_block_changed":          a.onBlockChanged,
		"on_click":                  a.onClick,
		"on_player_spawn":           a.onPlayerSpawn,
		"on_sent_map":               a.onSentMap,
		"on_joining_level":          a.onJoiningLevel,
		"on_joined_level":           a.onJoinedLevel,
		"on_setting_color":          a.onSettingColor,
		"on_getting_motd":           a.onGettingMotd,
		"on_sending_motd":           a.onSendingMotd,
		"on_getting_can_see":        a.onGettingCanSee,
		"on_notify_action":          a.onNotifyAction,
		"on_notify_position_action": a.onNotifyPositionAction,

		// ─── server events ───
		"on_connection_received": a.onConnectionReceived,
		"on_tick":                a.onTick,
		"on_shutdown":            a.onShutdown,
		"on_plugins_loaded":      a.onPluginsLoaded,
		"on_config_updated":      a.onConfigUpdated,
		"on_chat_sys":            a.onChatSys,
		"on_chat_from":           a.onChatFrom,
		"on_chat_all":            a.onChatAll,
		"on_plugin_message":      a.onPluginMessage,

		// ─── level events ───
		"on_level_save":            a.onLevelSave,
		"on_level_load":            a.onLevelLoad,
		"on_level_loaded":          a.onLevelLoaded,
		"on_level_added":           a.onLevelAdded,
		"on_level_removed":         a.onLevelRemoved,
		"on_level_unload":          a.onLevelUnload,
		"on_level_deleted":         a.onLevelDeleted,
		"on_level_copied":          a.onLevelCopied,
		"on_level_renamed":         a.onLevelRenamed,
		"on_main_level_changing":   a.onMainLevelChanging,
		"on_physics_update":        a.onPhysicsUpdate,
		"on_physics_state_changed": a.onPhysicsStateChanged,

		// ─── entity events ───
		"on_entity_spawned":   a.onEntitySpawned,
		"on_entity_despawned": a.onEntityDespawned,
		"on_sending_model":    a.onSendingModel,
	}
	for name, fn := range globals {
		a.L.SetGlobal(name, a.L.NewFunction(fn))
	}
}

// ─── player events ───

func (a *api) onConnect(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerConnect.Register(func(_ *plugin.Context, d plugin.PlayerConnectData) {
		a.callFn(fn, 0, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onDisconnect(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerDisconnect.Register(func(_ *plugin.Context, d plugin.PlayerDisconnectData) {
		a.callFn(fn, 0, wrapPlayer(L, d.Player), glua.LString(d.Reason))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onStartConnecting(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerStartConnecting.Register(func(_ *plugin.Context, d plugin.PlayerStartConnectingData) {
		a.callFn(fn, 0, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onFinishConnecting(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerFinishConnecting.Register(func(_ *plugin.Context, d plugin.PlayerFinishConnectingData) {
		a.callFn(fn, 0, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onChat(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerChat.Register(func(ctx *plugin.Context, d plugin.PlayerChatData) {
		if a.cbString(fn, d.Message, wrapPlayer(L, d.Player)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onMessageReceived(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnMessageReceived.Register(func(ctx *plugin.Context, d plugin.MessageReceivedData) {
		if a.cbString(fn, d.Message, wrapPlayer(L, d.Player)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onMove(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerMove.Register(func(ctx *plugin.Context, d plugin.PlayerMoveData) {
		if a.cbCancel(fn,
			wrapPlayer(L, d.Player),
			glua.LNumber(d.X), glua.LNumber(d.Y), glua.LNumber(d.Z),
			glua.LNumber(d.Yaw), glua.LNumber(d.Pitch)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onCommand(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerCommand.Register(func(ctx *plugin.Context, d plugin.PlayerCommandData) {
		if a.cbCancel(fn,
			wrapPlayer(L, d.Player),
			glua.LString(d.Command), glua.LString(d.Args)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onHelp(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerHelp.Register(func(ctx *plugin.Context, d plugin.PlayerHelpData) {
		if a.cbCancel(fn, wrapPlayer(L, d.Player), glua.LString(d.Target)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onPlayerAction(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerAction.Register(func(_ *plugin.Context, d plugin.PlayerActionData) {
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player),
			glua.LString(d.Action), glua.LString(d.Message), glua.LBool(d.Stealth))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onDying(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerDying.Register(func(ctx *plugin.Context, d plugin.PlayerDyingData) {
		if a.cbCancel(fn, wrapPlayer(L, d.Player), glua.LNumber(d.Cause)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onDied(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerDied.Register(func(_ *plugin.Context, d plugin.PlayerDiedData) {
		rets := a.callFn(fn, 1, wrapPlayer(L, d.Player), glua.LNumber(d.Cause), glua.LNumber(*d.Cooldown))
		if len(rets) > 0 {
			if n, ok := rets[0].(glua.LNumber); ok {
				*d.Cooldown = int(n)
			}
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onBlockChange(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnBlockChange.Register(func(ctx *plugin.Context, d plugin.BlockChangeData) {
		placing := glua.LFalse
		if d.Placing {
			placing = glua.LTrue
		}
		if a.cbCancel(fn,
			wrapPlayer(L, d.Player),
			glua.LNumber(d.X), glua.LNumber(d.Y), glua.LNumber(d.Z),
			glua.LNumber(d.Block), placing) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onBlockChanged(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnBlockChanged.Register(func(_ *plugin.Context, d plugin.BlockChangedData) {
		placing := glua.LFalse
		if d.Placing {
			placing = glua.LTrue
		}
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player),
			glua.LNumber(d.X), glua.LNumber(d.Y), glua.LNumber(d.Z),
			glua.LNumber(d.Block), placing)
	}, plugin.PriorityLow)
	return 0
}

func (a *api) onClick(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerClick.Register(func(_ *plugin.Context, d plugin.PlayerClickData) {
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player),
			glua.LNumber(d.Button), glua.LNumber(d.Action), glua.LNumber(d.EntityID),
			glua.LNumber(d.X), glua.LNumber(d.Y), glua.LNumber(d.Z), glua.LNumber(d.Face))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onPlayerSpawn(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPlayerSpawn.Register(func(_ *plugin.Context, d plugin.PlayerSpawnData) {
		rets := a.callFn(fn, 5,
			wrapPlayer(L, d.Player),
			glua.LNumber(*d.X), glua.LNumber(*d.Y), glua.LNumber(*d.Z),
			glua.LNumber(*d.Yaw), glua.LNumber(*d.Pitch))
		if len(rets) >= 5 {
			if n, ok := rets[0].(glua.LNumber); ok {
				*d.X = int(n)
			}
			if n, ok := rets[1].(glua.LNumber); ok {
				*d.Y = int(n)
			}
			if n, ok := rets[2].(glua.LNumber); ok {
				*d.Z = int(n)
			}
			if n, ok := rets[3].(glua.LNumber); ok {
				*d.Yaw = byte(n)
			}
			if n, ok := rets[4].(glua.LNumber); ok {
				*d.Pitch = byte(n)
			}
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onSentMap(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnSentMap.Register(func(_ *plugin.Context, d plugin.SentMapData) {
		a.callFn(fn, 0, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onJoiningLevel(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnJoiningLevel.Register(func(ctx *plugin.Context, d plugin.JoiningLevelData) {
		if a.cbCancel(fn, wrapPlayer(L, d.Player), glua.LString(d.LevelName)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onJoinedLevel(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnJoinedLevel.Register(func(_ *plugin.Context, d plugin.JoinedLevelData) {
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player),
			glua.LString(d.LevelName), glua.LString(d.PrevLevel))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onSettingColor(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnSettingColor.Register(func(_ *plugin.Context, d plugin.SettingColorData) {
		a.cbString(fn, d.Color, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onGettingMotd(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnGettingMotd.Register(func(_ *plugin.Context, d plugin.GettingMotdData) {
		a.cbString(fn, d.Motd, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onSendingMotd(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnSendingMotd.Register(func(_ *plugin.Context, d plugin.SendingMotdData) {
		a.cbString(fn, d.Motd, wrapPlayer(L, d.Player))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onGettingCanSee(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnGettingCanSee.Register(func(_ *plugin.Context, d plugin.GettingCanSeeData) {
		rets := a.callFn(fn, 1,
			wrapPlayer(L, d.Player), wrapPlayer(L, d.Target), glua.LBool(*d.CanSee))
		if len(rets) > 0 {
			if b, ok := rets[0].(glua.LBool); ok {
				*d.CanSee = bool(b)
			}
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onNotifyAction(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnNotifyAction.Register(func(_ *plugin.Context, d plugin.NotifyActionData) {
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player), glua.LNumber(d.Action), glua.LNumber(d.Value))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onNotifyPositionAction(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnNotifyPositionAction.Register(func(_ *plugin.Context, d plugin.NotifyPositionActionData) {
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player), glua.LNumber(d.Action),
			glua.LNumber(d.X), glua.LNumber(d.Y), glua.LNumber(d.Z))
	}, plugin.PriorityNormal)
	return 0
}

// ─── server events ───

func (a *api) onConnectionReceived(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnConnectionReceived.Register(func(ctx *plugin.Context, d plugin.ConnectionReceivedData) {
		if a.cbCancel(fn, glua.LString(d.RemoteAddr)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onTick(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnTick.Register(func(_ *plugin.Context, d plugin.TickData) {
		a.callFn(fn, 0, glua.LNumber(d.Tick))
	}, plugin.PriorityLow)
	return 0
}

func (a *api) onShutdown(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnShutdown.Register(func(_ *plugin.Context, d plugin.ShutdownData) {
		a.callFn(fn, 0, glua.LString(d.Reason))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onPluginsLoaded(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPluginsLoaded.Register(func(_ *plugin.Context, _ plugin.PluginsLoadedData) {
		a.callFn(fn, 0)
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onConfigUpdated(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnConfigUpdated.Register(func(_ *plugin.Context, _ plugin.ConfigUpdatedData) {
		a.callFn(fn, 0)
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onChatSys(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnChatSys.Register(func(_ *plugin.Context, d plugin.ChatSysData) {
		a.cbString(fn, d.Message)
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onChatFrom(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnChatFrom.Register(func(_ *plugin.Context, d plugin.ChatFromData) {
		a.cbString(fn, d.Message, wrapPlayer(L, d.Source))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onChatAll(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnChat.Register(func(_ *plugin.Context, d plugin.ChatData) {
		var p glua.LValue = glua.LNil
		if d.Source != nil {
			p = wrapPlayer(L, d.Source)
		}
		a.cbString(fn, d.Message, p)
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onPluginMessage(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPluginMessage.Register(func(_ *plugin.Context, d plugin.CpePluginMessageData) {
		a.callFn(fn, 0,
			wrapPlayer(L, d.Player), glua.LNumber(d.Channel), glua.LString(string(d.Data)))
	}, plugin.PriorityNormal)
	return 0
}

// ─── level events ───

func (a *api) onLevelSave(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelSave.Register(func(ctx *plugin.Context, _ plugin.LevelSaveData) {
		if a.cbCancel(fn) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelLoad(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelLoad.Register(func(ctx *plugin.Context, d plugin.LevelLoadData) {
		if a.cbCancel(fn, glua.LString(d.Name)) {
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelLoaded(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelLoaded.Register(func(_ *plugin.Context, d plugin.LevelLoadedData) {
		a.callFn(fn, 0, glua.LString(d.Name))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelAdded(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelAdded.Register(func(_ *plugin.Context, d plugin.LevelAddedData) {
		a.callFn(fn, 0, glua.LString(d.Name))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelRemoved(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelRemoved.Register(func(_ *plugin.Context, d plugin.LevelRemovedData) {
		a.callFn(fn, 0, glua.LString(d.Name))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelUnload(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelUnload.Register(func(_ *plugin.Context, d plugin.LevelUnloadData) {
		a.callFn(fn, 0, glua.LString(d.Name))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelDeleted(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelDeleted.Register(func(_ *plugin.Context, d plugin.LevelDeletedData) {
		a.callFn(fn, 0, glua.LString(d.Name))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelCopied(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelCopied.Register(func(_ *plugin.Context, d plugin.LevelCopiedData) {
		a.callFn(fn, 0, glua.LString(d.Source), glua.LString(d.Dest))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onLevelRenamed(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnLevelRenamed.Register(func(_ *plugin.Context, d plugin.LevelRenamedData) {
		a.callFn(fn, 0, glua.LString(d.Source), glua.LString(d.Dest))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onMainLevelChanging(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnMainLevelChanging.Register(func(_ *plugin.Context, d plugin.MainLevelChangingData) {
		a.cbString(fn, d.Map)
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onPhysicsUpdate(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPhysicsUpdate.Register(func(_ *plugin.Context, d plugin.PhysicsUpdateData) {
		a.callFn(fn, 0,
			glua.LNumber(d.X), glua.LNumber(d.Y), glua.LNumber(d.Z),
			glua.LNumber(d.Block), glua.LString(d.Level))
	}, plugin.PriorityLow)
	return 0
}

func (a *api) onPhysicsStateChanged(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnPhysicsStateChanged.Register(func(_ *plugin.Context, d plugin.PhysicsStateChangedData) {
		a.callFn(fn, 0, glua.LString(d.Level), glua.LNumber(d.Mode))
	}, plugin.PriorityNormal)
	return 0
}

// ─── entity events ───

func (a *api) onEntitySpawned(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnEntitySpawned.Register(func(_ *plugin.Context, d plugin.EntitySpawnedData) {
		a.callFn(fn, 0, glua.LString(*d.Name), glua.LString(*d.Model))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onEntityDespawned(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnEntityDespawned.Register(func(_ *plugin.Context, d plugin.EntityDespawnedData) {
		a.callFn(fn, 0, glua.LNumber(d.EntityID))
	}, plugin.PriorityNormal)
	return 0
}

func (a *api) onSendingModel(L *glua.LState) int {
	fn := L.CheckFunction(1)
	plugin.OnSendingModel.Register(func(_ *plugin.Context, d plugin.SendingModelData) {
		var p glua.LValue = glua.LNil
		if d.Player != nil {
			p = wrapPlayer(L, d.Player)
		}
		a.cbString(fn, d.Model, p)
	}, plugin.PriorityNormal)
	return 0
}
