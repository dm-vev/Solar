package command

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func teleportCommand(ctx Context, args []string) (string, bool) {
	if ctx.World == nil {
		return "teleport is unavailable", true
	}
	if len(args) != 3 && len(args) != 5 {
		return "usage: /tp x y z [yaw pitch]", true
	}

	x, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Sprintf("invalid x: %v", err), true
	}
	y, err := strconv.Atoi(args[1])
	if err != nil {
		return fmt.Sprintf("invalid y: %v", err), true
	}
	z, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Sprintf("invalid z: %v", err), true
	}
	yaw := ctx.Yaw
	pitch := ctx.Pitch
	if len(args) == 5 {
		yawValue, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Sprintf("invalid yaw: %v", err), true
		}
		pitchValue, err := strconv.Atoi(args[4])
		if err != nil {
			return fmt.Sprintf("invalid pitch: %v", err), true
		}
		yaw = byte(yawValue)
		pitch = byte(pitchValue)
	}

	if !ctx.World.MovePlayer(x, y, z, yaw, pitch) {
		return "teleport failed", true
	}
	return fmt.Sprintf("teleported to %d %d %d", x, y, z), true
}

func setSpawnCommand(ctx Context, args []string) (string, bool) {
	if ctx.World == nil {
		return "spawn update is unavailable", true
	}
	if len(args) != 0 && len(args) != 3 && len(args) != 5 {
		return "usage: /setspawn [x y z [yaw pitch]]", true
	}

	x, y, z := ctx.Position.X, ctx.Position.Y, ctx.Position.Z
	yaw := ctx.Yaw
	pitch := ctx.Pitch
	if len(args) >= 3 {
		var err error
		x, err = strconv.Atoi(args[0])
		if err != nil {
			return fmt.Sprintf("invalid x: %v", err), true
		}
		y, err = strconv.Atoi(args[1])
		if err != nil {
			return fmt.Sprintf("invalid y: %v", err), true
		}
		z, err = strconv.Atoi(args[2])
		if err != nil {
			return fmt.Sprintf("invalid z: %v", err), true
		}
	}
	if len(args) == 5 {
		yawValue, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Sprintf("invalid yaw: %v", err), true
		}
		pitchValue, err := strconv.Atoi(args[4])
		if err != nil {
			return fmt.Sprintf("invalid pitch: %v", err), true
		}
		yaw = byte(yawValue)
		pitch = byte(pitchValue)
	}

	if !ctx.World.SetSpawn(x, y, z, yaw, pitch) {
		return "spawn update failed", true
	}
	return fmt.Sprintf("spawn set to %d %d %d", x, y, z), true
}

func saveCommand(ctx Context, _ []string) (string, bool) {
	if ctx.Persistence == nil {
		return "save is unavailable", true
	}
	if !ctx.Persistence.SaveState() {
		return "save failed", true
	}
	return "save queued", true
}

func kickCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return "kick is unavailable", true
	}
	if len(args) < 1 {
		return "usage: /kick name [reason...]", true
	}

	name := args[0]
	if strings.TrimSpace(name) == "" {
		return "invalid name", true
	}
	reason := strings.TrimSpace(strings.Join(args[1:], " "))
	if !ctx.Moderation.KickPlayer(name, reason) {
		return fmt.Sprintf("player not found: %s", name), true
	}
	if reason == "" {
		return fmt.Sprintf("kicked %s", name), true
	}
	return fmt.Sprintf("kicked %s: %s", name, reason), true
}

func banCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return "ban is unavailable", true
	}
	if len(args) < 1 {
		return "usage: /ban name [reason...]", true
	}

	name := args[0]
	if strings.TrimSpace(name) == "" {
		return "invalid name", true
	}
	reason := strings.TrimSpace(strings.Join(args[1:], " "))
	if !ctx.Moderation.BanPlayer(name, reason) {
		return "ban failed", true
	}
	if reason == "" {
		return fmt.Sprintf("banned %s", name), true
	}
	return fmt.Sprintf("banned %s: %s", name, reason), true
}

func unbanCommand(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return "unban is unavailable", true
	}
	if len(args) != 1 {
		return "usage: /unban name", true
	}
	if strings.TrimSpace(args[0]) == "" {
		return "invalid name", true
	}
	if !ctx.Moderation.UnbanPlayer(args[0]) {
		return fmt.Sprintf("player not banned: %s", args[0]), true
	}
	return fmt.Sprintf("unbanned %s", args[0]), true
}

func newLevelCommand(ctx Context, args []string) (string, bool) {
	if ctx.World == nil {
		return "world generation is unavailable", true
	}
	if len(args) < 5 {
		return "usage: /newlvl name theme width height length [seed]", true
	}

	name := args[0]
	theme := args[1]
	width, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Sprintf("invalid width: %v", err), true
	}
	height, err := strconv.Atoi(args[3])
	if err != nil {
		return fmt.Sprintf("invalid height: %v", err), true
	}
	length, err := strconv.Atoi(args[4])
	if err != nil {
		return fmt.Sprintf("invalid length: %v", err), true
	}
	seed := ""
	if len(args) > 5 {
		seed = strings.Join(args[5:], " ")
	}

	if !ctx.World.GenerateWorld(name, theme, width, height, length, seed) {
		return "level generation failed", true
	}
	return fmt.Sprintf("generated level %s (%s %dx%dx%d)", name, theme, width, height, length), true
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
		return "usage: /whitelist [on|off|add|remove|list]", true
	}
}

func whitelistOn(ctx Context) (string, bool) {
	if ctx.Moderation == nil {
		return "whitelist is unavailable", true
	}
	if !ctx.Moderation.SetWhitelistEnabled(true) {
		return "whitelist already enabled", true
	}
	return "whitelist enabled", true
}

func whitelistOff(ctx Context) (string, bool) {
	if ctx.Moderation == nil {
		return "whitelist is unavailable", true
	}
	if !ctx.Moderation.SetWhitelistEnabled(false) {
		return "whitelist already disabled", true
	}
	return "whitelist disabled", true
}

func whitelistAddCmd(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return "whitelist is unavailable", true
	}
	if len(args) != 2 {
		return "usage: /whitelist add name", true
	}
	if strings.TrimSpace(args[1]) == "" {
		return "invalid name", true
	}
	if !ctx.Moderation.WhitelistAdd(args[1]) {
		return fmt.Sprintf("player already whitelisted: %s", args[1]), true
	}
	return fmt.Sprintf("whitelisted %s", args[1]), true
}

func whitelistRemoveCmd(ctx Context, args []string) (string, bool) {
	if ctx.Moderation == nil {
		return "whitelist is unavailable", true
	}
	if len(args) != 2 {
		return "usage: /whitelist remove name", true
	}
	if strings.TrimSpace(args[1]) == "" {
		return "invalid name", true
	}
	if !ctx.Moderation.WhitelistRemove(args[1]) {
		return fmt.Sprintf("player not whitelisted: %s", args[1]), true
	}
	return fmt.Sprintf("removed %s from whitelist", args[1]), true
}

func whitelistList(ctx Context) (string, bool) {
	if ctx.Players == nil {
		return "whitelist is unavailable", true
	}
	names := ctx.Players.ListWhitelisted()
	if len(names) == 0 {
		return "whitelist: none", true
	}
	sort.Strings(names)
	return "whitelist: " + strings.Join(names, ", "), true
}

func playersCommand(ctx Context, _ []string) (string, bool) {
	if ctx.Players == nil {
		return "player list is unavailable", true
	}
	names := ctx.Players.ListPlayers()
	if len(names) == 0 {
		return "players: none", true
	}
	sort.Strings(names)
	return "players: " + strings.Join(names, ", "), true
}

func whitelistStatus(ctx Context) string {
	if ctx.Moderation == nil {
		return "whitelist status unavailable"
	}
	if ctx.Moderation.WhitelistEnabled() {
		return "whitelist: on"
	}
	return "whitelist: off"
}
