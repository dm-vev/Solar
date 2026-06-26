package entity

import "testing"

func TestManagerApplyDelta(t *testing.T) {
	manager := NewManager()
	id, ok := manager.Add("alice", Position{X: 1, Y: 2, Z: 3})
	if !ok {
		t.Fatal("Add failed")
	}
	pos, yaw, pitch, ok := manager.ApplyDelta(id, 2, -1, 3, 4, 5)
	if !ok {
		t.Fatal("ApplyDelta returned false")
	}
	if pos != (Position{X: 3, Y: 1, Z: 6}) || yaw != 4 || pitch != 5 {
		t.Fatalf("ApplyDelta = %+v yaw=%d pitch=%d", pos, yaw, pitch)
	}
	if _, _, _, ok := manager.ApplyDelta(999, 1, 1, 1, 0, 0); ok {
		t.Fatal("ApplyDelta succeeded for missing entity")
	}
}
