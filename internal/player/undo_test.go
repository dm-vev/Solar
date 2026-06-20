package player

import "testing"

func TestUndoPushAndUndo(t *testing.T) {
	u := NewUndoStack(10)
	u.Push([]BlockChange{{0, 0, 0, 1, 2}, {1, 0, 0, 3, 4}})
	u.Push([]BlockChange{{2, 0, 0, 5, 6}})

	if !u.CanUndo() {
		t.Fatal("should be able to undo")
	}
	changes := u.Undo()
	if len(changes) != 1 || changes[0].X != 2 {
		t.Fatalf("undo returned %v, want last batch", changes)
	}
	changes = u.Undo()
	if len(changes) != 2 {
		t.Fatalf("undo returned %d changes, want 2", len(changes))
	}
	if u.CanUndo() {
		t.Fatal("should not be able to undo after all undone")
	}
}

func TestRedo(t *testing.T) {
	u := NewUndoStack(10)
	u.Push([]BlockChange{{0, 0, 0, 1, 2}})

	u.Undo()
	if !u.CanRedo() {
		t.Fatal("should be able to redo")
	}
	changes := u.Redo()
	if len(changes) != 1 || changes[0].New != 2 {
		t.Fatalf("redo returned %v", changes)
	}
	if u.CanRedo() {
		t.Fatal("should not be able to redo after all redone")
	}
}

func TestPushClearsRedo(t *testing.T) {
	u := NewUndoStack(10)
	u.Push([]BlockChange{{0, 0, 0, 1, 2}})
	u.Undo()
	u.Push([]BlockChange{{1, 0, 0, 3, 4}}) // new action clears redo
	if u.CanRedo() {
		t.Fatal("redo should be cleared after new push")
	}
}

func TestMaxLimit(t *testing.T) {
	u := NewUndoStack(2)
	u.Push([]BlockChange{{0, 0, 0, 1, 2}})
	u.Push([]BlockChange{{1, 0, 0, 3, 4}})
	u.Push([]BlockChange{{2, 0, 0, 5, 6}})
	// Should only keep last 2
	u.Undo()
	u.Undo()
	if u.CanUndo() {
		t.Fatal("should only keep max batches")
	}
}
