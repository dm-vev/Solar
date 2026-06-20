package blockdb

import (
	"strings"
	"sync"
	"sync/atomic"
)

// NameConverter maps player names to compact integer IDs for BlockDB.
// IDs are stable across restarts (persisted in PlayerDB Data["blockdb_id"]).
// ponytail: simple in-memory map; IDs are 1-based, 0 = server/console.
type NameConverter struct {
	mu     sync.Mutex
	nextID atomic.Int32
	ids    map[string]int32
}

func NewNameConverter() *NameConverter {
	nc := &NameConverter{ids: make(map[string]int32)}
	nc.nextID.Store(1)
	return nc
}

// Get returns the integer ID for a player name, assigning one if needed.
func (nc *NameConverter) Get(name string) int32 {
	key := strings.ToLower(name)
	nc.mu.Lock()
	if id, ok := nc.ids[key]; ok {
		nc.mu.Unlock()
		return id
	}
	id := nc.nextID.Add(1) - 1
	nc.ids[key] = id
	nc.mu.Unlock()
	return id
}

// Set assigns a specific ID for a player name (used when restoring from PlayerDB).
func (nc *NameConverter) Set(name string, id int32) {
	key := strings.ToLower(name)
	nc.mu.Lock()
	nc.ids[key] = id
	if id >= nc.nextID.Load() {
		nc.nextID.Store(id + 1)
	}
	nc.mu.Unlock()
}
