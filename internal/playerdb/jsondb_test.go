package playerdb

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solar-mc/solar/plugin/playerdb"
)

func TestJSONDB_SaveGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Save(&playerdb.PlayerEntry{
		Name:       "Alice",
		FirstLogin: time.Now(),
		LoginCount: 1,
		IP:         "1.2.3.4",
	})

	got := db.Get("alice")
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Name != "Alice" {
		t.Fatalf("name = %q, want Alice", got.Name)
	}
	if got.LoginCount != 1 {
		t.Fatalf("login_count = %d, want 1", got.LoginCount)
	}
}

func TestJSONDB_Persist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Save(&playerdb.PlayerEntry{Name: "Bob", LoginCount: 5})
	if err := db.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}

	db2, err := New(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got := db2.Get("bob")
	if got == nil || got.LoginCount != 5 {
		t.Fatalf("after reload: %+v", got)
	}
}

func TestJSONDB_Delete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Save(&playerdb.PlayerEntry{Name: "Charlie"})
	if !db.Delete("charlie") {
		t.Fatal("Delete returned false")
	}
	if db.Get("charlie") != nil {
		t.Fatal("Get returned entry after delete")
	}
	if db.Delete("charlie") {
		t.Fatal("Delete returned true for missing entry")
	}
}

func TestJSONDB_Search(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, name := range []string{"Alice", "Albert", "Bob", "Charlie"} {
		db.Save(&playerdb.PlayerEntry{Name: name})
	}

	results := db.Search("Al")
	if len(results) != 2 {
		t.Fatalf("Search(Al) = %d results, want 2", len(results))
	}
}

func TestJSONDB_ListSortedCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Save(&playerdb.PlayerEntry{Name: "Charlie", LoginCount: 1})
	db.Save(&playerdb.PlayerEntry{Name: "alice", LoginCount: 2})
	db.Save(&playerdb.PlayerEntry{Name: "Bob", LoginCount: 3})

	results := db.List()
	if len(results) != 3 {
		t.Fatalf("List = %d results, want 3", len(results))
	}
	if results[0].Name != "alice" || results[1].Name != "Bob" || results[2].Name != "Charlie" {
		t.Fatalf("List order = %q, %q, %q", results[0].Name, results[1].Name, results[2].Name)
	}
	results[0].LoginCount = 99
	if got := db.Get("alice").LoginCount; got != 2 {
		t.Fatalf("List returned mutable entry, stored LoginCount = %d", got)
	}
}

func TestJSONDB_Count(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Save(&playerdb.PlayerEntry{Name: "A"})
	db.Save(&playerdb.PlayerEntry{Name: "B"})
	db.Save(&playerdb.PlayerEntry{Name: "C"})

	if db.Count() != 3 {
		t.Fatalf("Count = %d, want 3", db.Count())
	}
}

func TestJSONDB_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	_ = os.WriteFile(path, []byte{}, 0o644)
	db, err := New(path)
	if err != nil {
		t.Fatalf("New with empty file: %v", err)
	}
	if db.Count() != 0 {
		t.Fatalf("Count = %d, want 0", db.Count())
	}
}

func TestEnsureEntry_New(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	e := EnsureEntry(db, "NewPlayer", "5.6.7.8")
	if e.LoginCount != 1 {
		t.Fatalf("LoginCount = %d, want 1", e.LoginCount)
	}
	if e.IP != "5.6.7.8" {
		t.Fatalf("IP = %q, want 5.6.7.8", e.IP)
	}
	if e.FirstLogin.IsZero() {
		t.Fatal("FirstLogin is zero")
	}
}

func TestEnsureEntry_Existing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "playerdb.json")
	db, err := New(path)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	db.Save(&playerdb.PlayerEntry{
		Name:       "OldPlayer",
		LoginCount: 3,
		IP:         "1.1.1.1",
	})

	e := EnsureEntry(db, "oldplayer", "2.2.2.2")
	if e.LoginCount != 4 {
		t.Fatalf("LoginCount = %d, want 4", e.LoginCount)
	}
	if e.IP != "2.2.2.2" {
		t.Fatalf("IP = %q, want 2.2.2.2", e.IP)
	}
	if e.LastIP != "1.1.1.1" {
		t.Fatalf("LastIP = %q, want 1.1.1.1", e.LastIP)
	}
}
