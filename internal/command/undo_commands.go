// undo_commands.go implements /undo and /redo for drawing operations.
//
// /undo  — reverts the last drawing batch (cuboid, line, sphere, fill, paste)
// /redo  — re-applies the last undone batch

package command

// undoCommand — /undo
// Reverts the last drawing batch by restoring old blocks.
func undoCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	changes := ctx.Draw.Undo()
	if changes == nil {
		return ctx.tr("command.undo.none"), true
	}
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		ctx.Draw.PlaceBlock(c.X, c.Y, c.Z, c.Old)
	}
	return ctx.tr("command.undo.done", len(changes)), true
}

// redoCommand — /redo
// Re-applies the last undone batch.
func redoCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	changes := ctx.Draw.Redo()
	if changes == nil {
		return ctx.tr("command.redo.none"), true
	}
	for _, c := range changes {
		ctx.Draw.PlaceBlock(c.X, c.Y, c.Z, c.New)
	}
	return ctx.tr("command.redo.done", len(changes)), true
}
