package command

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// aboutCommand — /about [x y z] or /about (uses player position)
// Shows the block change history at the given coordinates.
func aboutCommand(ctx Context, args []string) (string, bool) {
	if ctx.BlockDB == nil {
		return ctx.tr("command.about.unavailable"), true
	}

	x, y, z := ctx.Position.X, ctx.Position.Y, ctx.Position.Z
	if len(args) == 3 {
		var err error
		if x, err = strconv.Atoi(args[0]); err != nil {
			return ctx.tr("command.shared.invalid_x", err), true
		}
		if y, err = strconv.Atoi(args[1]); err != nil {
			return ctx.tr("command.shared.invalid_y", err), true
		}
		if z, err = strconv.Atoi(args[2]); err != nil {
			return ctx.tr("command.shared.invalid_z", err), true
		}
	} else if len(args) != 0 {
		return ctx.tr("command.about.usage"), true
	}

	entries := ctx.BlockDB.ChangesAt(x, y, z)
	if len(entries) == 0 {
		return ctx.tr("command.about.none", x, y, z), true
	}

	var sb strings.Builder
	for i, e := range entries {
		if i > 0 {
			sb.WriteString("\n")
		}
		age := time.Since(e.Time).Round(time.Second)
		sb.WriteString(fmt.Sprintf("&7[%s] &e%s &7changed &b%d &7→ &a%d &7(%s ago)",
			e.Time.Format("2006-01-02 15:04:05"),
			e.PlayerName, e.OldBlock, e.NewBlock, age))
	}
	return sb.String(), true
}

// undoCommand — /undo [timespan]
// Undoes the player's own block changes within the given timespan (default 5m).
func undoCommand(ctx Context, args []string) (string, bool) {
	if ctx.BlockDB == nil {
		return ctx.tr("command.undo.unavailable"), true
	}

	duration := 5 * time.Minute
	if len(args) > 0 {
		d, err := time.ParseDuration(args[0])
		if err != nil {
			return ctx.tr("command.undo.invalid_duration", args[0]), true
		}
		duration = d
	}

	since := time.Now().Add(-duration)
	entries := ctx.BlockDB.ChangesBy(ctx.Username, since, 0)
	if len(entries) == 0 {
		return ctx.tr("command.undo.none"), true
	}

	reverted := 0
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if ctx.BlockDB.RevertBlock(e.X, e.Y, e.Z, e.OldBlock) {
			reverted++
		}
	}
	return ctx.tr("command.undo.done", reverted), true
}

// blockDBCommand — /blockdb [enable|disable|clear|stats]
func blockDBCommand(ctx Context, args []string) (string, bool) {
	if ctx.BlockDB == nil {
		return ctx.tr("command.blockdb.unavailable"), true
	}

	if len(args) == 0 {
		return ctx.tr("command.blockdb.usage"), true
	}

	switch strings.ToLower(args[0]) {
	case "enable":
		ctx.BlockDB.SetEnabled(true)
		return ctx.tr("command.blockdb.enabled"), true
	case "disable":
		ctx.BlockDB.SetEnabled(false)
		return ctx.tr("command.blockdb.disabled"), true
	case "clear":
		if err := ctx.BlockDB.Clear(); err != nil {
			return ctx.tr("command.blockdb.clear_failed", err), true
		}
		return ctx.tr("command.blockdb.cleared"), true
	case "stats":
		count := ctx.BlockDB.Count()
		enabled := ctx.BlockDB.Enabled()
		status := "off"
		if enabled {
			status = "on"
		}
		return ctx.tr("command.blockdb.stats", count, status), true
	default:
		return ctx.tr("command.blockdb.usage"), true
	}
}
