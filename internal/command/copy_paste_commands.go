// copy_paste_commands.go implements the clipboard commands.
//
// /copy [air]  — captures a 2-mark region into the player's clipboard.
//                 The "air" flag includes air blocks in the clipboard.
// /paste [air] — replays the clipboard at a 1-mark location.
//                 The "air" flag pastes air blocks too.

package command

// copyCommand — /copy [air]
// Captures a cuboid region into the player's clipboard.
func copyCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	pasteAir := len(args) > 0 && args[0] == "air"
	_ = pasteAir // /copy captures everything; air flag affects /paste

	ctx.Draw.StartSelection(2, func(marks [][3]int) {
		min, max := marks[0], marks[1]
		if min[0] > max[0] {
			min[0], max[0] = max[0], min[0]
		}
		if min[1] > max[1] {
			min[1], max[1] = max[1], min[1]
		}
		if min[2] > max[2] {
			min[2], max[2] = max[2], min[2]
		}
		ctx.Draw.CopyRegion(min, max)
	})
	return ctx.tr("command.draw.select2"), true
}

// pasteCommand — /paste [air]
// Replays the clipboard at the clicked location.
func pasteCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if !ctx.Draw.HasClipboard() {
		return ctx.tr("command.paste.empty"), true
	}
	pasteAir := len(args) > 0 && args[0] == "air"

	ctx.Draw.StartSelection(1, func(marks [][3]int) {
		count := ctx.Draw.PasteAt(marks[0], pasteAir)
		_ = count
	})
	return ctx.tr("command.draw.select1"), true
}
