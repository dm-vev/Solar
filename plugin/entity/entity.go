// Package entity defines the EntityManager interface that plugins use
// to create and manage non-player entities (NPCs, bots, decorative entities).
package entity

// EntityInfo describes a server-side entity that plugins can spawn.
//
//nolint:revive // intentional: re-exported as plugin.X
type EntityInfo struct {
	Name    string
	X, Y, Z int // position in wire units (1/32 block)
	Yaw     byte
	Pitch   byte
	Model   string // e.g. "humanoid", "chicken"
	Skin    string // skin name/URL
}

// EntityManager is the interface plugins use to create and manage
// non-player entities (NPCs, bots, decorative entities).
//
//nolint:revive // intentional: re-exported as plugin.X
type EntityManager interface {
	// Spawn creates a new entity and broadcasts it to all online players.
	// Returns the entity ID (1-254) or 0 if the entity limit is reached.
	Spawn(info EntityInfo) byte

	// Despawn removes an entity by ID and broadcasts removal to all players.
	Despawn(entityID byte) bool

	// Teleport moves an entity to a new position and broadcasts the update.
	Teleport(entityID byte, x, y, z int, yaw, pitch byte) bool

	// Get returns info about an entity by ID.
	Get(entityID byte) (EntityInfo, bool)

	// Count returns the number of active entities (including players).
	Count() int
}
