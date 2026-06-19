package command

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func helpCommand(registry *Registry) Handler {
	return func(_ Context, _ []string) (string, bool) {
		registry.mu.RLock()
		names := make([]string, 0, len(registry.handlers))
		for name := range registry.handlers {
			names = append(names, name)
		}
		registry.mu.RUnlock()

		sort.Strings(names)
		return "commands: /" + strings.Join(names, ", /"), true
	}
}

func whereCommand(ctx Context, _ []string) (string, bool) {
	return fmt.Sprintf("%s is at %d %d %d", ctx.Username, ctx.Position.X, ctx.Position.Y, ctx.Position.Z), true
}

func setBlockCommand(ctx Context, args []string) (string, bool) {
	if len(args) != 4 {
		return "usage: /setblock x y z block", true
	}
	if ctx.World == nil {
		return "block updates are unavailable", true
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
	blockValue, err := strconv.Atoi(args[3])
	if err != nil {
		return fmt.Sprintf("invalid block: %v", err), true
	}

	if !ctx.World.SetBlock(x, y, z, byte(blockValue)) {
		return "block position out of bounds", true
	}
	return fmt.Sprintf("set block at %d %d %d to %d", x, y, z, blockValue), true
}
