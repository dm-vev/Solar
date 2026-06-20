// undo.go implements per-player undo/redo history for block changes.
//
// Each drawing operation (cuboid, line, sphere, fill, paste) pushes a
// batch of block changes onto the player's undo stack. /undo pops the
// last batch and restores the old blocks. /redo re-applies the batch.
//
// The stack is bounded (default 200 batches) to limit memory usage.

package player

// BlockChange records a single block change for undo.
type BlockChange struct {
	X, Y, Z int
	Old     byte
	New     byte
}

// UndoStack holds per-player undo/redo history.
type UndoStack struct {
	undo [][]BlockChange
	redo [][]BlockChange
	max  int
}

// NewUndoStack creates a new undo stack with the given capacity.
func NewUndoStack(max int) *UndoStack {
	if max <= 0 {
		max = 200
	}
	return &UndoStack{max: max}
}

// Push records a batch of block changes for undo.
func (u *UndoStack) Push(changes []BlockChange) {
	u.undo = append(u.undo, changes)
	if len(u.undo) > u.max {
		u.undo = u.undo[1:]
	}
	u.redo = u.redo[:0] // clear redo on new action
}

// Undo pops the last batch and returns the changes to revert.
// Returns nil if there's nothing to undo.
func (u *UndoStack) Undo() []BlockChange {
	if len(u.undo) == 0 {
		return nil
	}
	last := u.undo[len(u.undo)-1]
	u.undo = u.undo[:len(u.undo)-1]
	u.redo = append(u.redo, last)
	return last
}

// Redo re-applies the last undone batch and returns the changes.
// Returns nil if there's nothing to redo.
func (u *UndoStack) Redo() []BlockChange {
	if len(u.redo) == 0 {
		return nil
	}
	last := u.redo[len(u.redo)-1]
	u.redo = u.redo[:len(u.redo)-1]
	u.undo = append(u.undo, last)
	return last
}

// CanUndo reports whether there are changes to undo.
func (u *UndoStack) CanUndo() bool { return len(u.undo) > 0 }

// CanRedo reports whether there are changes to redo.
func (u *UndoStack) CanRedo() bool { return len(u.redo) > 0 }

// Clear empties both undo and redo stacks.
func (u *UndoStack) Clear() {
	u.undo = u.undo[:0]
	u.redo = u.redo[:0]
}
