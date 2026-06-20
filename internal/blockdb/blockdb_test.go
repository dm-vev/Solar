package blockdb

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/solar-mc/solar/plugin/blockdb"
)

func TestFileDB_AddAndCount(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Add(blockdb.Entry{PlayerID: 1, Time: time.Now(), X: 10, Y: 20, Z: 30, OldBlock: 0, NewBlock: 7, Flags: blockdb.ManualPlace})
	db.Add(blockdb.Entry{PlayerID: 2, Time: time.Now(), X: 11, Y: 21, Z: 31, OldBlock: 7, NewBlock: 0, Flags: blockdb.ManualPlace})

	if db.Count() != 2 {
		t.Fatalf("Count = %d, want 2", db.Count())
	}
}

func TestFileDB_FlushAndReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Add(blockdb.Entry{PlayerID: 1, Time: time.Now(), X: 10, Y: 20, Z: 30, OldBlock: 0, NewBlock: 7})
	db.Add(blockdb.Entry{PlayerID: 2, Time: time.Now(), X: 11, Y: 21, Z: 31, OldBlock: 7, NewBlock: 0})

	if err := db.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if db.Count() != 2 {
		t.Fatalf("Count after flush = %d, want 2", db.Count())
	}

	db2, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if db2.Count() != 2 {
		t.Fatalf("Count after reload = %d, want 2", db2.Count())
	}
}

func TestFileDB_ChangesAt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	now := time.Now()
	db.Add(blockdb.Entry{PlayerID: 1, Time: now, X: 10, Y: 20, Z: 30, OldBlock: 0, NewBlock: 7})
	db.Add(blockdb.Entry{PlayerID: 2, Time: now, X: 11, Y: 21, Z: 31, OldBlock: 7, NewBlock: 0})
	db.Add(blockdb.Entry{PlayerID: 1, Time: now, X: 10, Y: 20, Z: 30, OldBlock: 7, NewBlock: 3})

	if err := db.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	changes := db.ChangesAt(10, 20, 30)
	if len(changes) != 2 {
		t.Fatalf("ChangesAt(10,20,30) = %d entries, want 2", len(changes))
	}
}

func TestFileDB_ChangesBy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	now := time.Now()
	db.Add(blockdb.Entry{PlayerID: 1, Time: now.Add(-2 * time.Hour), X: 1, Y: 1, Z: 1, OldBlock: 0, NewBlock: 7})
	db.Add(blockdb.Entry{PlayerID: 2, Time: now.Add(-1 * time.Hour), X: 2, Y: 2, Z: 2, OldBlock: 0, NewBlock: 7})
	db.Add(blockdb.Entry{PlayerID: 1, Time: now, X: 3, Y: 3, Z: 3, OldBlock: 7, NewBlock: 0})

	if err := db.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	// All changes by player 1
	changes := db.ChangesBy(1, time.Time{}, time.Time{}, 0)
	if len(changes) != 2 {
		t.Fatalf("ChangesBy(1) = %d, want 2", len(changes))
	}

	// Changes by player 1 in the last 90 minutes (should get 1)
	changes = db.ChangesBy(1, now.Add(-90*time.Minute), time.Time{}, 0)
	if len(changes) != 1 {
		t.Fatalf("ChangesBy(1, since=90min) = %d, want 1", len(changes))
	}

	// Limited to 1 result
	changes = db.ChangesBy(1, time.Time{}, time.Time{}, 1)
	if len(changes) != 1 {
		t.Fatalf("ChangesBy(1, max=1) = %d, want 1", len(changes))
	}
}

func TestFileDB_Clear(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Add(blockdb.Entry{PlayerID: 1, Time: time.Now(), X: 1, Y: 1, Z: 1, OldBlock: 0, NewBlock: 7})
	if err := db.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if db.Count() != 1 {
		t.Fatalf("Count = %d, want 1", db.Count())
	}

	if err := db.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	if db.Count() != 0 {
		t.Fatalf("Count after clear = %d, want 0", db.Count())
	}
}

func TestFileDB_Disabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.cbdb")
	db, err := New(path, 128, 64, 128)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.SetEnabled(false)
	db.Add(blockdb.Entry{PlayerID: 1, Time: time.Now(), X: 1, Y: 1, Z: 1, OldBlock: 0, NewBlock: 7})

	if db.Count() != 0 {
		t.Fatalf("Count when disabled = %d, want 0", db.Count())
	}
}
