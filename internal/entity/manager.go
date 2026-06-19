package entity

import "sync/atomic"

// Manager owns the current set of active entities.
type Manager struct {
	nextID atomic.Uint32
	tick   atomic.Uint64
	shards [entityShardCount]entityShard
}

// NewManager creates an empty entity manager.
func NewManager() *Manager {
	m := &Manager{}
	for i := range m.shards {
		m.shards[i].entities = make(map[uint32]Entity)
	}
	return m
}

// Add inserts a new entity and returns its ID.
func (m *Manager) Add(name string, pos Position) (uint32, bool) {
	if name == "" {
		return 0, false
	}

	id := m.nextID.Add(1)
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.Lock()
	shard.entities[id] = Entity{Name: name, Pos: pos}
	shard.mu.Unlock()
	return id, true
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
	if entity, ok := shard.entities[id]; ok {
		entity.Vel = vel
		shard.entities[id] = entity
	}
	shard.mu.Unlock()
}

// SetLocation updates the position and rotation for an entity.
func (m *Manager) SetLocation(id uint32, pos Position, yaw, pitch byte) bool {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.Lock()
	defer shard.mu.Unlock()

	entity, ok := shard.entities[id]
	if !ok {
		return false
	}
	entity.Pos = pos
	entity.Yaw = yaw
	entity.Pitch = pitch
	shard.entities[id] = entity
	return true
}

// Get returns an entity snapshot by ID.
func (m *Manager) Get(id uint32) (Entity, bool) {
	shard := &m.shards[m.shardIndex(id)]
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	entity, ok := shard.entities[id]
	return entity, ok
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
		for id, entity := range shard.entities {
			entity.Pos.X += entity.Vel.X
			entity.Pos.Y += entity.Vel.Y
			entity.Pos.Z += entity.Vel.Z
			shard.entities[id] = entity
		}
		shard.mu.Unlock()
	}
	m.tick.Add(1)
}

// TickCount returns how many ticks the entity simulation has processed.
func (m *Manager) TickCount() uint64 {
	return m.tick.Load()
}
