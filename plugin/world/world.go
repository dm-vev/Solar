// Package world defines the World interface that plugins use to
// interact with the world.
package world

// World is the interface plugins use to interact with the world.
type World interface {
	// GetBlock returns the block ID at the given coordinates.
	// Returns false if out of bounds.
	GetBlock(x, y, z int) (byte, bool)

	// SetBlock sets a block in the world and broadcasts to all players.
	// Returns false if out of bounds.
	SetBlock(x, y, z int, block byte) bool

	// Spawn returns the world spawn point in block coordinates.
	Spawn() (x, y, z int, yaw, pitch byte)

	// SetSpawn updates the world spawn point.
	SetSpawn(x, y, z int, yaw, pitch byte)

	// Dimensions returns the world dimensions in blocks.
	Dimensions() (width, height, length int)

	// Save persists the world to disk.
	Save() error
}
