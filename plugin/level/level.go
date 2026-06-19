// Package level defines the Level and LevelManager interfaces that
// plugins use to manage multiple loaded levels.
package level

import "github.com/solar-mc/solar/plugin/player"

// Level represents a loaded world/level that plugins can interact with.
type Level interface {
	// Name returns the level name.
	Name() string
	// GetBlock returns the block ID at the given coordinates.
	GetBlock(x, y, z int) (byte, bool)
	// SetBlock sets a block and broadcasts to players on this level.
	SetBlock(x, y, z int, block byte) bool
	// Spawn returns the spawn point in block coordinates.
	Spawn() (x, y, z int, yaw, pitch byte)
	// SetSpawn updates the spawn point.
	SetSpawn(x, y, z int, yaw, pitch byte)
	// Dimensions returns the level dimensions.
	Dimensions() (width, height, length int)
	// Save persists the level to disk.
	Save() error
	// PlayerCount returns the number of players on this level.
	PlayerCount() int
	// Players returns all players currently on this level.
	Players() []player.Player
	// Rename changes the level name on disk.
	Rename(newName string) error
	// Copy creates a copy of this level with the given name.
	Copy(destName string) error
	// Backup creates a backup of this level.
	Backup(backupName string) error
	// Delete removes the level from disk. The level must be unloaded first.
	Delete() error
	// Resize changes the level dimensions.
	Resize(width, height, length int) error
	// Reload reloads the level from disk, keeping players connected.
	Reload() error
	// Message broadcasts a chat message to all players on this level.
	Message(msg string)
}

// LevelManager manages multiple loaded levels.
//
//nolint:revive // intentional: re-exported as plugin.X
type LevelManager interface {
	// Current returns the main/default level.
	Current() Level
	// Find returns a loaded level by name (case-insensitive), or nil.
	Find(name string) Level
	// Create generates a new level with the given dimensions.
	// generator is e.g. "Classic", "Flat", "fCraft".
	Create(name string, width, height, length int, generator string, seed string) (Level, error)
	// Load loads a level from disk by name.
	Load(name string) (Level, error)
	// Unload removes a level from memory. Returns false if players are still on it.
	Unload(name string) bool
	// SaveAll saves all loaded levels to disk.
	SaveAll() error
	// List returns names of all loaded levels.
	List() []string
	// ListFiles returns names of all level files on disk.
	ListFiles() []string
	// Rename renames a level on disk.
	RenameLevel(oldName, newName string) error
	// DeleteLevel deletes a level file from disk.
	DeleteLevel(name string) error
	// CopyLevel copies a level file on disk.
	CopyLevel(srcName, destName string) error
	// BackupLevel creates a backup of a level.
	BackupLevel(name, backupName string) error
}
