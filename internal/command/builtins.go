package command

import (
	"sort"
	"strconv"
	"strings"
)

func helpCommand(registry *Registry) Handler {
	return func(ctx Context, _ []string) (string, bool) {
		playerRank := 0
		if ctx.RankLevel != nil {
			playerRank = ctx.RankLevel()
		}
		registry.mu.RLock()
		names := make([]string, 0, len(registry.handlers))
		for name, entry := range registry.handlers {
			if playerRank >= entry.minRank {
				names = append(names, name)
			}
		}
		registry.mu.RUnlock()

		sort.Strings(names)
		return ctx.tr("command.help.list", "/"+strings.Join(names, ", /")), true
	}
}

func whereCommand(ctx Context, _ []string) (string, bool) {
	return ctx.tr("command.where.position", ctx.Username, ctx.Position.X, ctx.Position.Y, ctx.Position.Z), true
}

func setBlockCommand(ctx Context, args []string) (string, bool) {
	if len(args) != 4 {
		return ctx.tr("command.setblock.usage"), true
	}
	if ctx.World == nil {
		return ctx.tr("command.teleport.unavailable"), true
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
	blockValue, err := strconv.Atoi(args[3])
	if err != nil || blockValue < 0 || blockValue > 255 {
		return ctx.tr("command.shared.invalid_block", err), true
	}

	if !ctx.World.SetBlock(x, y, z, byte(blockValue)) {
		return ctx.tr("command.setblock.oob"), true
	}
	return ctx.tr("command.setblock.done", x, y, z, blockValue), true
}
