package command

import (
	"sort"
	"strconv"
	"strings"
)

func teleportCommand(ctx Context, args []string) (string, bool) {
	if ctx.World == nil {
		return ctx.tr("command.teleport.unavailable"), true
	}
	if len(args) != 3 && len(args) != 5 {
		return ctx.tr("command.teleport.usage"), true
	}

	x, err := strconv.Atoi(args[0])
	if err != nil {
		return ctx.tr("command.shared.invalid_x", err), true
	}
	y, err := strconv.Atoi(args[1])
	if err != nil {
		return ctx.tr("command.shared.invalid_y", err), true
	}
	z, err := strconv.Atoi(args[2])
	if err != nil {
		return ctx.tr("command.shared.invalid_z", err), true
	}
	yaw := ctx.Yaw
	pitch := ctx.Pitch
	if len(args) == 5 {
		yawValue, err := strconv.Atoi(args[3])
		if err != nil {
			return ctx.tr("command.shared.invalid_yaw", err), true
		}
		pitchValue, err := strconv.Atoi(args[4])
		if err != nil {
			return ctx.tr("command.shared.invalid_pitch", err), true
		}
		yaw = byte(yawValue)
		pitch = byte(pitchValue)
	}

	if !ctx.World.MovePlayer(x, y, z, yaw, pitch) {
		return ctx.tr("command.teleport.failed"), true
	}
	return ctx.tr("command.teleport.done", x, y, z), true
}

func setSpawnCommand(ctx Context, args []string) (string, bool) {
	if ctx.World == nil {
		return ctx.tr("command.setspawn.unavailable"), true
	}
	if len(args) != 0 && len(args) != 3 && len(args) != 5 {
		return ctx.tr("command.setspawn.usage"), true
	}

	x, y, z := ctx.Position.X, ctx.Position.Y, ctx.Position.Z
	yaw := ctx.Yaw
	pitch := ctx.Pitch
	if len(args) >= 3 {
		var err error
		x, err = strconv.Atoi(args[0])
		if err != nil {
			return ctx.tr("command.shared.invalid_x", err), true
		}
		y, err = strconv.Atoi(args[1])
		if err != nil {
			return ctx.tr("command.shared.invalid_y", err), true
		}
		z, err = strconv.Atoi(args[2])
		if err != nil {
			return ctx.tr("command.shared.invalid_z", err), true
		}
	}
	if len(args) == 5 {
		yawValue, err := strconv.Atoi(args[3])
		if err != nil {
			return ctx.tr("command.shared.invalid_yaw", err), true
		}
		pitchValue, err := strconv.Atoi(args[4])
		if err != nil {
			return ctx.tr("command.shared.invalid_pitch", err), true
		}
		yaw = byte(yawValue)
		pitch = byte(pitchValue)
	}

	if !ctx.World.SetSpawn(x, y, z, yaw, pitch) {
		return ctx.tr("command.setspawn.failed"), true
	}
	return ctx.tr("command.setspawn.done", x, y, z), true
}

func saveCommand(ctx Context, _ []string) (string, bool) {
	if ctx.Persistence == nil {
		return ctx.tr("command.save.unavailable"), true
	}
	if !ctx.Persistence.SaveState() {
		return ctx.tr("command.save.failed"), true
	}
	return ctx.tr("command.save.queued"), true
}

func kickCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.kick.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.kick.usage"), true
	}

	name := args[0]
	if strings.TrimSpace(name) == "" {
		return ctx.tr("command.shared.invalid_name"), true
	}
	reason := strings.TrimSpace(strings.Join(args[1:], " "))
	if !ctx.Moderation.KickPlayer(name, reason) {
		return ctx.tr("command.shared.player_not_found", name), true
	}
	if reason == "" {
		return ctx.tr("command.kick.done", name), true
	}
	return ctx.tr("command.kick.done_reason", name, reason), true
}

func banCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.ban.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.ban.usage"), true
	}

	name := args[0]
	if strings.TrimSpace(name) == "" {
		return ctx.tr("command.shared.invalid_name"), true
	}
	reason := strings.TrimSpace(strings.Join(args[1:], " "))
	if !ctx.Moderation.BanPlayer(name, reason) {
		return ctx.tr("command.ban.failed"), true
	}
	if reason == "" {
		return ctx.tr("command.ban.done", name), true
	}
	return ctx.tr("command.ban.done_reason", name, reason), true
}

func unbanCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.unban.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.unban.usage"), true
	}
	if strings.TrimSpace(args[0]) == "" {
		return ctx.tr("command.shared.invalid_name"), true
	}
	if !ctx.Moderation.UnbanPlayer(args[0]) {
		return ctx.tr("command.unban.not_banned", args[0]), true
	}
	return ctx.tr("command.unban.done", args[0]), true
}

func newLevelCommand(ctx Context, args []string) (string, bool) {
	if ctx.World == nil {
		return ctx.tr("command.newlvl.unavailable"), true
	}
	if len(args) < 5 {
		return ctx.tr("command.newlvl.usage"), true
	}

	name := args[0]
	theme := args[1]
	width, err := strconv.Atoi(args[2])
	if err != nil {
		return ctx.tr("command.shared.invalid_width", err), true
	}
	if width <= 0 {
		return ctx.tr("command.shared.invalid_width", "must be positive"), true
	}
	height, err := strconv.Atoi(args[3])
	if err != nil {
		return ctx.tr("command.shared.invalid_height", err), true
	}
	if height <= 0 {
		return ctx.tr("command.shared.invalid_height", "must be positive"), true
	}
	length, err := strconv.Atoi(args[4])
	if err != nil {
		return ctx.tr("command.shared.invalid_length", err), true
	}
	if length <= 0 {
		return ctx.tr("command.shared.invalid_length", "must be positive"), true
	}
	seed := ""
	if len(args) > 5 {
		seed = strings.Join(args[5:], " ")
	}

	if !ctx.World.GenerateWorld(name, theme, width, height, length, seed) {
		return ctx.tr("command.newlvl.failed"), true
	}
	return ctx.tr("command.newlvl.done", name, theme, width, height, length), true
}

func whitelistCommand(ctx Context, args []string) (string, bool) {
	if len(args) == 0 {
		return whitelistStatus(ctx), true
	}

	switch strings.ToLower(args[0]) {
	case "on":
		return whitelistOn(ctx)
	case "off":
		return whitelistOff(ctx)
	case "add":
		return whitelistAddCmd(ctx, args)
	case "remove":
		return whitelistRemoveCmd(ctx, args)
	case "list":
		return whitelistList(ctx)
	default:
		return ctx.tr("command.whitelist.usage"), true
	}
}

func whitelistOn(ctx Context) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.whitelist.unavailable"), true
	}
	if !ctx.Moderation.SetWhitelistEnabled(true) {
		return ctx.tr("command.whitelist.already_on"), true
	}
	return ctx.tr("command.whitelist.enabled"), true
}

func whitelistOff(ctx Context) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.whitelist.unavailable"), true
	}
	if !ctx.Moderation.SetWhitelistEnabled(false) {
		return ctx.tr("command.whitelist.already_off"), true
	}
	return ctx.tr("command.whitelist.disabled"), true
}

func whitelistAddCmd(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.whitelist.unavailable"), true
	}
	if len(args) != 2 {
		return ctx.tr("command.whitelist.add.usage"), true
	}
	if strings.TrimSpace(args[1]) == "" {
		return ctx.tr("command.shared.invalid_name"), true
	}
	if !ctx.Moderation.WhitelistAdd(args[1]) {
		return ctx.tr("command.whitelist.add.already", args[1]), true
	}
	return ctx.tr("command.whitelist.add.done", args[1]), true
}

func whitelistRemoveCmd(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return ctx.tr("command.whitelist.unavailable"), true
	}
	if len(args) != 2 {
		return ctx.tr("command.whitelist.remove.usage"), true
	}
	if strings.TrimSpace(args[1]) == "" {
		return ctx.tr("command.shared.invalid_name"), true
	}
	if !ctx.Moderation.WhitelistRemove(args[1]) {
		return ctx.tr("command.whitelist.remove.not", args[1]), true
	}
	return ctx.tr("command.whitelist.remove.done", args[1]), true
}

func whitelistList(ctx Context) (string, bool) {
	if ctx.Players == nil {
		return ctx.tr("command.whitelist.unavailable"), true
	}
	names := ctx.Players.ListWhitelisted()
	if len(names) == 0 {
		return ctx.tr("command.whitelist.list.none"), true
	}
	sort.Strings(names)
	return ctx.tr("command.whitelist.list.items", strings.Join(names, ", ")), true
}

func playersCommand(ctx Context, _ []string) (string, bool) {
	if ctx.Players == nil {
		return ctx.tr("command.players.unavailable"), true
	}
	names := ctx.Players.ListPlayers()
	if len(names) == 0 {
		return ctx.tr("command.players.none"), true
	}
	sort.Strings(names)
	return ctx.tr("command.players.list", strings.Join(names, ", ")), true
}

func whitelistStatus(ctx Context) string {
	if ctx.Moderation == nil {
		return ctx.tr("command.whitelist.unavailable")
	}
	if ctx.Moderation.WhitelistEnabled() {
		return ctx.tr("command.whitelist.status.on")
	}
	return ctx.tr("command.whitelist.status.off")
}
