package entity

import "sync"

// Sixteen shards keep entity updates granular without adding lock overhead.
const entityShardCount = 16

type entityShard struct {
	mu       sync.RWMutex
	entities map[uint32]Entity
}

func (m *Manager) shardIndex(id uint32) int {
	return int(id % entityShardCount)
}
