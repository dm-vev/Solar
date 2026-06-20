// teleport_commands.go implements teleport and chat commands.
//
// Teleport commands:
//   /spawn       — teleport to the world spawn point
//   /back        — return to the last position before a teleport/death
//   /tpa <name>  — teleport to another player (direct, no request flow)
//   /summon <name> — teleport another player to you (operator)
//
// Chat commands:
//   /me <action> — send an IRC-style action message (* Player does X)
//   /whisper <name> <msg> — send a private message to another player
//   /ignore <name> — toggle ignoring a player's chat messages

package command

import "strings"

// spawnCommand — /spawn
// Teleports the player to the world spawn point.
func spawnCommand(ctx Context, args []string) (string, bool) {
	if ctx.Teleport == nil {
		return ctx.tr("command.teleport.unavailable"), true
	}
	x, y, z, yaw, pitch := ctx.Teleport.SpawnPoint()
	if !ctx.World.MovePlayer(x, y, z, yaw, pitch) {
		return ctx.tr("command.teleport.failed"), true
	}
	return ctx.tr("command.spawn.done"), true
}

// backCommand — /back
// Returns the player to their last position before a teleport or death.
func backCommand(ctx Context, args []string) (string, bool) {
	if ctx.Teleport == nil {
		return ctx.tr("command.teleport.unavailable"), true
	}
	if !ctx.Teleport.Back() {
		return ctx.tr("command.back.none"), true
	}
	return ctx.tr("command.back.done"), true
}

// tpaCommand — /tpa <name>
// Teleports the player to the named player (direct teleport, no request).
func tpaCommand(ctx Context, args []string) (string, bool) {
	if ctx.Teleport == nil {
		return ctx.tr("command.teleport.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.tpa.usage"), true
	}
	if !ctx.Teleport.TeleportToPlayer(args[0]) {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	return ctx.tr("command.tpa.done", args[0]), true
}

// summonCommand — /summon <name>
// Teleports the named player to the caller (operator only).
func summonCommand(ctx Context, args []string) (string, bool) {
	if ctx.Teleport == nil {
		return ctx.tr("command.teleport.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.summon.usage"), true
	}
	if !ctx.Teleport.SummonPlayer(args[0]) {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	return ctx.tr("command.summon.done", args[0]), true
}

// meCommand — /me <action>
// Sends an IRC-style action message to all players.
func meCommand(ctx Context, args []string) (string, bool) {
	if ctx.Chat == nil {
		return ctx.tr("command.chat.unavailable"), true
	}
	if len(args) == 0 {
		return ctx.tr("command.me.usage"), true
	}
	ctx.Chat.Me(strings.Join(args, " "))
	return "", false
}

// whisperCommand — /whisper <name> <message>
// Sends a private message to another player.
func whisperCommand(ctx Context, args []string) (string, bool) {
	if ctx.Chat == nil {
		return ctx.tr("command.chat.unavailable"), true
	}
	if len(args) < 2 {
		return ctx.tr("command.whisper.usage"), true
	}
	target := args[0]
	msg := strings.Join(args[1:], " ")
	if !ctx.Chat.Whisper(target, msg) {
		return ctx.tr("command.moderation.player_not_found", target), true
	}
	return "", false
}

// ignoreCommand — /ignore <name>
// Toggles ignoring a player's chat messages.
func ignoreCommand(ctx Context, args []string) (string, bool) {
	if ctx.Chat == nil {
		return ctx.tr("command.chat.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.ignore.usage"), true
	}
	ignored, ok := ctx.Chat.Ignore(args[0])
	if !ok {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	if ignored {
		return ctx.tr("command.ignore.on", args[0]), true
	}
	return ctx.tr("command.ignore.off", args[0]), true
}
