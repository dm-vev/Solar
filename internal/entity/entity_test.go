package entity

import (
	"fmt"
	"sync"
	"testing"
)

func TestManagerTickMovesEntities(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	id, ok := mgr.Add("tester", Position{X: 1, Y: 2, Z: 3})
	if !ok {
		t.Fatal("Add returned false")
	}

	mgr.SetVelocity(id, Velocity{X: 2, Y: -1, Z: 4})
	mgr.Tick()

	got, ok := mgr.Get(id)
	if !ok {
		t.Fatal("Get returned false")
	}
	if got.Pos.X != 3 || got.Pos.Y != 1 || got.Pos.Z != 7 {
		t.Fatalf("position = %+v, want 3,1,7", got.Pos)
	}
	if got.Name != "tester" {
		t.Fatalf("name = %q, want tester", got.Name)
	}
	if mgr.TickCount() != 1 {
		t.Fatalf("TickCount = %d, want 1", mgr.TickCount())
	}
}

func TestManagerRemove(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	id, ok := mgr.Add("tester", Position{})
	if !ok {
		t.Fatal("Add returned false")
	}
	mgr.Remove(id)

	if _, ok := mgr.Get(id); ok {
		t.Fatal("entity still present after Remove")
	}
	if mgr.Count() != 0 {
		t.Fatalf("Count = %d, want 0", mgr.Count())
	}
}

func TestManagerAllocatesClassicWireIDs(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	seen := make(map[uint32]bool, MaxClassicEntityID)
	for i := uint32(0); i < MaxClassicEntityID; i++ {
		id, ok := mgr.Add(fmt.Sprintf("entity-%d", i), Position{})
		if !ok {
			t.Fatalf("Add(%d) returned false", i)
		}
		if id == 0 || id > MaxClassicEntityID {
			t.Fatalf("id = %d, want 1..%d", id, MaxClassicEntityID)
		}
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
	}

	if id, ok := mgr.Add("overflow", Position{}); ok {
		t.Fatalf("Add returned id %d after entity space was full", id)
	}

	const removedID = 17
	mgr.Remove(removedID)
	id, ok := mgr.Add("reused", Position{})
	if !ok {
		t.Fatal("Add did not reuse freed entity ID")
	}
	if id != removedID {
		t.Fatalf("reused id = %d, want %d", id, removedID)
	}
}

func TestManagerConcurrentAddDoesNotDuplicateIDs(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	const total = 64
	ids := make(chan uint32, total)
	var wg sync.WaitGroup
	wg.Add(total)
	for i := 0; i < total; i++ {
		go func(i int) {
			defer wg.Done()
			id, ok := mgr.Add(fmt.Sprintf("entity-%d", i), Position{})
			if !ok {
				t.Errorf("Add(%d) returned false", i)
				return
			}
			ids <- id
		}(i)
	}
	wg.Wait()
	close(ids)

	seen := make(map[uint32]bool, total)
	for id := range ids {
		if id == 0 || id > MaxClassicEntityID {
			t.Fatalf("id = %d, want 1..%d", id, MaxClassicEntityID)
		}
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
	}
	if len(seen) != total {
		t.Fatalf("allocated %d IDs, want %d", len(seen), total)
	}
}

func TestManagerConcurrentShardAccess(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	firstID, ok := mgr.Add("alpha", Position{})
	if !ok {
		t.Fatal("first Add returned false")
	}
	secondID, ok := mgr.Add("beta", Position{})
	if !ok {
		t.Fatal("second Add returned false")
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			mgr.SetLocation(firstID, Position{X: i, Y: i, Z: i}, byte(i), byte(i))
			mgr.Tick()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			mgr.SetLocation(secondID, Position{X: i, Y: i, Z: i}, byte(i), byte(i))
			_, _ = mgr.Get(firstID)
		}
	}()

	wg.Wait()

	if got := mgr.Count(); got != 2 {
		t.Fatalf("Count = %d, want 2", got)
	}
	if got := mgr.TickCount(); got != 1000 {
		t.Fatalf("TickCount = %d, want 1000", got)
	}
}
