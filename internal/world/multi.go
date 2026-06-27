package world

import (
	"strings"
	"sync"

	"github.com/solar-mc/solar/plugin"
)

// MultiManager manages multiple loaded levels keyed by name (case-insensitive).
// ponytail: pointer-identity comparison used for per-level player tracking;
// switch to a level-ID if levels are ever recycled.
type MultiManager struct {
	mu     sync.RWMutex
	levels map[string]*managedLevel
	main   string
}

type managedLevel struct {
	manager *Manager
	name    string
	path    string
}

func NewMultiManager() *MultiManager {
	return &MultiManager{levels: make(map[string]*managedLevel)}
}

// SetMain registers the main level and records its name.
// Fires OnMainLevelChanging with the new name (modifiable) before applying.
func (m *MultiManager) SetMain(name string, mgr *Manager, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if plugin.OnMainLevelChanging.HasHandlers() {
		n := name
		plugin.OnMainLevelChanging.Fire(plugin.MainLevelChangingData{Map: &n})
		name = n
	}
	m.main = name
	m.levels[strings.ToLower(name)] = &managedLevel{manager: mgr, name: name, path: path}
}

// MainName returns the name of the main level.
func (m *MultiManager) MainName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.main
}

// MainManager returns the main level's Manager, or nil if unset.
func (m *MultiManager) MainManager() *Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ml, ok := m.levels[strings.ToLower(m.main)]; ok {
		return ml.manager
	}
	return nil
}

// Get returns a level's Manager by name (case-insensitive), or nil.
func (m *MultiManager) Get(name string) *Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ml, ok := m.levels[strings.ToLower(name)]; ok {
		return ml.manager
	}
	return nil
}

// Add registers a level with the given name, using the provided Manager.
func (m *MultiManager) Add(name string, mgr *Manager, path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.levels[strings.ToLower(name)] = &managedLevel{manager: mgr, name: name, path: path}
}

// Rename updates a loaded level name and path.
func (m *MultiManager) Rename(oldName, newName, path string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	oldKey := strings.ToLower(oldName)
	ml, ok := m.levels[oldKey]
	if !ok {
		return false
	}
	delete(m.levels, oldKey)
	ml.name = newName
	ml.path = path
	m.levels[strings.ToLower(newName)] = ml
	if strings.EqualFold(m.main, oldName) {
		m.main = newName
	}
	return true
}

// Remove removes a level by name. Returns false if not found.
func (m *MultiManager) Remove(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := strings.ToLower(name)
	if _, ok := m.levels[key]; !ok {
		return false
	}
	delete(m.levels, key)
	return true
}

// Names returns all loaded level names.
func (m *MultiManager) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]string, 0, len(m.levels))
	for _, ml := range m.levels {
		out = append(out, ml.name)
	}
	return out
}

// Has reports whether a level is loaded.
func (m *MultiManager) Has(name string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.levels[strings.ToLower(name)]
	return ok
}

// Path returns the disk path for a level, or "" if not found.
func (m *MultiManager) Path(name string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ml, ok := m.levels[strings.ToLower(name)]; ok {
		return ml.path
	}
	return ""
}
