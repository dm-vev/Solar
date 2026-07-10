package command

import (
	"sort"
	"strings"
)

// gotoCommand — /goto <level>
// Teleports the player to the named level.
func gotoCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.goto.usage"), true
	}
	if !ctx.Levels.Goto(args[0]) {
		return ctx.tr("command.goto.not_found", args[0]), true
	}
	return ctx.tr("command.goto.done", args[0]), true
}

// mainCommand — /main
// Teleports the player to the main level.
func mainCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	mainName := ctx.Levels.MainLevel()
	if mainName == "" {
		return ctx.tr("command.level.unavailable"), true
	}
	if !ctx.Levels.Goto(mainName) {
		return ctx.tr("command.goto.not_found", mainName), true
	}
	return ctx.tr("command.goto.done", mainName), true
}

// loadCommand — /load <name>
// Loads a level from disk.
func loadCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.load.usage"), true
	}
	if !ctx.Levels.LoadLevel(args[0]) {
		return ctx.tr("command.load.failed", args[0]), true
	}
	return ctx.tr("command.load.done", args[0]), true
}

// unloadCommand — /unload <name>
// Unloads a non-main level with no online players.
func unloadCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.unload.usage"), true
	}
	if strings.EqualFold(args[0], ctx.Levels.MainLevel()) {
		return ctx.tr("command.unload.main"), true
	}
	if !ctx.Levels.UnloadLevel(args[0]) {
		return ctx.tr("command.unload.failed", args[0]), true
	}
	return ctx.tr("command.unload.done", args[0]), true
}

// reloadCommand — /reload
// Reloads the player's current level from disk.
func reloadCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	if !ctx.Levels.ReloadLevel() {
		return ctx.tr("command.reload.failed"), true
	}
	return ctx.tr("command.reload.done"), true
}

// levelsCommand — /levels
// Lists loaded levels and levels on disk.
func levelsCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	loaded := ctx.Levels.ListLevels()
	files := ctx.Levels.ListLevelFiles()

	sort.Strings(loaded)
	sort.Strings(files)

	return ctx.tr("command.levels.list",
		strings.Join(loaded, ", "), strings.Join(files, ", ")), true
}

// physicsCommand — /physics [0|1|2|3|4|5]
// Sets or shows the physics mode for the current level.
func physicsCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	if len(args) == 0 {
		mode := ctx.Levels.PhysicsMode()
		return ctx.tr("command.blocks.current", mode), true
	}
	var mode int
	switch args[0] {
	case "0", "off":
		mode = 0
	case "1", "basic":
		mode = 1
	case "2", "advanced":
		mode = 2
	case "3", "hardcore", "custom":
		mode = 3
	case "4", "instant":
		mode = 4
	case "5", "doors":
		mode = 5
	default:
		return ctx.tr("command.blocks.usage"), true
	}
	ctx.Levels.SetPhysicsMode(mode)
	return ctx.tr("command.blocks.set", mode), true
}
