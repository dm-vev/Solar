package entity

import (
	"sync"
	"sync/atomic"
)

// MaxClassicEntityID is the highest non-self entity ID that fits in Classic packets.
const MaxClassicEntityID uint32 = 254

// Manager owns the current set of active entities.
type Manager struct {
	allocMu sync.Mutex
	nextID  uint32
	tick    atomic.Uint64
	shards  [entityShardCount]entityShard
}

// NewManager creates an empty entity manager.
func NewManager() *Manager {
	m := &Manager{}
	for i := range m.shards {
		m.shards[i].entities = make(map[uint32]*Entity)
	}
	return m
}

// Add inserts a new entity and returns its ID.
func (m *Manager) Add(name string, pos Position) (uint32, bool) {
	if name == "" {
		return 0, false
	}

	m.allocMu.Lock()
	defer m.allocMu.Unlock()

	for attempts := uint32(0); attempts < MaxClassicEntityID; attempts++ {
		m.nextID = (m.nextID % MaxClassicEntityID) + 1
		id := m.nextID
		shard := &m.shards[m.shardIndex(id)]
		shard.mu.Lock()
		if _, exists := shard.entities[id]; !exists {
			shard.entities[id] = &Entity{Name: name, Pos: pos}
			shard.mu.Unlock()
			return id, true
		}
		shard.mu.Unlock()
	}

	return 0, false
}

// Remove deletes an entity by ID.
func (m *Manager) Remove(id uint32) {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.Lock()
	delete(shard.entities, id)
	shard.mu.Unlock()
}

// SetVelocity updates the velocity for an entity.
func (m *Manager) SetVelocity(id uint32, vel Velocity) {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.Lock()
	if e, ok := shard.entities[id]; ok {
		e.Vel = vel
	}
	shard.mu.Unlock()
}

// SetLocation updates the position and rotation for an entity.
func (m *Manager) SetLocation(id uint32, pos Position, yaw, pitch byte) bool {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	e, ok := shard.entities[id]
	if !ok {
		return false
	}
	e.Pos = pos
	e.Yaw = yaw
	e.Pitch = pitch
	return true
}

// ApplyDelta atomically applies a position delta and optional rotation
// update under a single shard lock, returning the resulting state.
func (m *Manager) ApplyDelta(id uint32, dx, dy, dz int, yaw, pitch byte) (Position, byte, byte, bool) {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	e, ok := shard.entities[id]
	if !ok {
		return Position{}, 0, 0, false
	}
	e.Pos.X += dx
	e.Pos.Y += dy
	e.Pos.Z += dz
	if yaw != 0 || pitch != 0 {
		e.Yaw = yaw
		e.Pitch = pitch
	}
	return e.Pos, e.Yaw, e.Pitch, true
}

// Get returns an entity snapshot by ID.
func (m *Manager) Get(id uint32) (Entity, bool) {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	e, ok := shard.entities[id]
	if !ok {
		return Entity{}, false
	}
	return *e, true
}

// Count returns the current number of active entities.
func (m *Manager) Count() int {
	total := 0
	for i := range m.shards {
		shard := &m.shards[i]
		shard.mu.RLock()
		total += len(shard.entities)
		shard.mu.RUnlock()
	}
	return total
}

// Tick advances the entity simulation by one step.
func (m *Manager) Tick() {
	for i := range m.shards {
		shard := &m.shards[i]
		shard.mu.Lock()
		for _, e := range shard.entities {
			e.Pos.X += e.Vel.X
			e.Pos.Y += e.Vel.Y
			e.Pos.Z += e.Vel.Z
		}
		shard.mu.Unlock()
	}
	m.tick.Add(1)
}

// TickCount returns how many ticks the entity simulation has processed.
func (m *Manager) TickCount() uint64 {
	return m.tick.Load()
}
