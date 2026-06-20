package command

// muteCommand — /mute <name>
func muteCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.moderation.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.mute.usage"), true
	}
	if !ctx.Moderation.MutePlayer(args[0]) {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	return ctx.tr("command.mute.done", args[0]), true
}

// unmuteCommand — /unmute <name>
func unmuteCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.moderation.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.unmute.usage"), true
	}
	if !ctx.Moderation.UnmutePlayer(args[0]) {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	return ctx.tr("command.unmute.done", args[0]), true
}

// freezeCommand — /freeze <name>
func freezeCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.moderation.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.freeze.usage"), true
	}
	if !ctx.Moderation.FreezePlayer(args[0]) {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	return ctx.tr("command.freeze.done", args[0]), true
}

// unfreezeCommand — /unfreeze <name>
func unfreezeCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.moderation.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.unfreeze.usage"), true
	}
	if !ctx.Moderation.UnfreezePlayer(args[0]) {
		return ctx.tr("command.moderation.player_not_found", args[0]), true
	}
	return ctx.tr("command.unfreeze.done", args[0]), true
}

// afkCommand — /afk
// Toggles the player's own AFK status.
func afkCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.moderation.unavailable"), true
	}
	afk, ok := ctx.Moderation.ToggleAFK(ctx.Username)
	if !ok {
		return ctx.tr("command.moderation.player_not_found", ctx.Username), true
	}
	if afk {
		return ctx.tr("command.afk.on"), true
	}
	return ctx.tr("command.afk.off"), true
}

// hideCommand — /hide
// Toggles the player's own hidden status.
func hideCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.moderation.unavailable"), true
	}
	hidden, ok := ctx.Moderation.ToggleHide(ctx.Username)
	if !ok {
		return ctx.tr("command.moderation.player_not_found", ctx.Username), true
	}
	if hidden {
		return ctx.tr("command.hide.on"), true
	}
	return ctx.tr("command.hide.off"), true
}
