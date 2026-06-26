package ranks

import (
	"testing"

	"github.com/solar-mc/solar/plugin/playerdb"
)

func TestDefaultRanks(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) != 7 {
		t.Fatalf("expected 7 default ranks, got %d", len(all))
	}
}

func TestGetByName(t *testing.T) {
	r := NewRegistry()
	guest := r.Get("guest")
	if guest == nil || guest.Permission != PermGuest {
		t.Fatal("guest rank not found")
	}
	op := r.Get("Operator") // case-insensitive
	if op == nil || op.Permission != PermOperator {
		t.Fatal("Operator rank not found (case-insensitive)")
	}
}

func TestGetByPerm(t *testing.T) {
	r := NewRegistry()
	builder := r.GetByPerm(PermBuilder)
	if builder == nil || builder.Name != "builder" {
		t.Fatal("builder rank not found by perm")
	}
}

func TestHasRank(t *testing.T) {
	if !HasRank(PermOperator, PermGuest) {
		t.Fatal("operator should have guest rank")
	}
	if HasRank(PermGuest, PermOperator) {
		t.Fatal("guest should not have operator rank")
	}
}

func TestIsOperator(t *testing.T) {
	if !IsOperator(PermOperator) {
		t.Fatal("operator should be operator")
	}
	if IsOperator(PermGuest) {
		t.Fatal("guest should not be operator")
	}
}

func TestDefaultRank(t *testing.T) {
	r := NewRegistry()
	dr := r.DefaultRank()
	if dr == nil || dr.Name != "guest" {
		t.Fatal("default rank should be guest")
	}
}

func TestPlayerRankPersistence(t *testing.T) {
	db := &rankTestDB{entries: make(map[string]*playerdb.PlayerEntry)}
	r := NewRegistry()
	r.SetPlayerDB(db)

	if got := r.GetPlayerRank("alice"); got != PermGuest {
		t.Fatalf("initial rank = %d, want guest", got)
	}
	if !r.SetPlayerRank("alice", PermOperator) {
		t.Fatal("SetPlayerRank returned false")
	}
	if got := r.GetPlayerRank("alice"); got != PermOperator {
		t.Fatalf("rank = %d, want operator", got)
	}

	db.entries["bad"] = &playerdb.PlayerEntry{Name: "bad", Data: map[string]string{"rank": "not-int"}}
	if got := r.GetPlayerRank("bad"); got != PermGuest {
		t.Fatalf("bad rank = %d, want guest", got)
	}
}

type rankTestDB struct {
	entries map[string]*playerdb.PlayerEntry
}

func (d *rankTestDB) Get(name string) *playerdb.PlayerEntry {
	return d.entries[name]
}

func (d *rankTestDB) Save(entry *playerdb.PlayerEntry) {
	d.entries[entry.Name] = entry
}

func (d *rankTestDB) Delete(name string) bool {
	if _, ok := d.entries[name]; !ok {
		return false
	}
	delete(d.entries, name)
	return true
}

func (d *rankTestDB) List() []*playerdb.PlayerEntry { return nil }
func (d *rankTestDB) Search(string) []*playerdb.PlayerEntry {
	return nil
}
func (d *rankTestDB) Count() int   { return len(d.entries) }
func (d *rankTestDB) Flush() error { return nil }
