package blockdef

import (
	"sort"
	"sync"
)

// Registry holds block definitions and supports global + per-level scopes.
// In Solar's single-world architecture both scopes share the same store;
// when multi-world is added, per-level registries can split off.
type Registry struct {
	mu   sync.RWMutex
	defs map[byte]*BlockDefinition
	dir  string
}

// NewRegistry creates an empty registry. dir is the filesystem path
// where JSON definition files are stored.
func NewRegistry(dir string) *Registry {
	return &Registry{
		defs: make(map[byte]*BlockDefinition),
		dir:  dir,
	}
}

// Dir returns the storage directory.
func (r *Registry) Dir() string {
	return r.dir
}

// Add inserts or replaces a block definition.
func (r *Registry) Add(def BlockDefinition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	d := def
	r.defs[d.ID] = &d
}

// Remove deletes a block definition. Returns false if not found.
func (r *Registry) Remove(id byte) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.defs[id]; !ok {
		return false
	}
	delete(r.defs, id)
	return true
}

// Get returns a copy of the block definition for id.
func (r *Registry) Get(id byte) (BlockDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.defs[id]
	if !ok {
		return BlockDefinition{}, false
	}
	return *d, true
}

// All returns copies of all registered definitions sorted by ID.
func (r *Registry) All() []BlockDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]BlockDefinition, 0, len(r.defs))
	for _, d := range r.defs {
		result = append(result, *d)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// FreeID returns the lowest unused custom block ID (>= FirstCustomBlock).
// Returns 0 if all IDs are taken.
func (r *Registry) FreeID() byte {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for id := byte(FirstCustomBlock); id <= MaxBlockID; id++ {
		if _, ok := r.defs[id]; !ok {
			return id
		}
	}
	return 0
}

// Has reports whether a definition exists for id.
func (r *Registry) Has(id byte) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.defs[id]
	return ok
}

// Count returns the number of registered definitions.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.defs)
}
