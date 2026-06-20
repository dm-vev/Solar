// Package specialblocks implements interactive blocks: doors, portals,
// message blocks, and TNT explosions.
//
// Special blocks are stored in a per-level Registry keyed by block
// coordinates. When a player places a special block, it's registered.
// When a player steps on a special block (detected via movement),
// the appropriate action fires.
//
// TNT is handled differently — it explodes immediately on placement
// (when physics is enabled), destroying blocks in a radius.
//
// Block IDs (matching MCGalaxy):
//   Message blocks: 130-134 (MB_White through MB_Lava)
//   Portals: 160-162 (Portal_Air/Water/Lava), 175-176 (Portal_Blue/Orange)
//   Doors: 201 (Door_Log_air), 168-174 (oDoor variants)
//   TNT: 182 (small), 183 (big), 186 (nuke), 184 (explosion visual)

package blocks

import (
	"sync"
)

// Block IDs for special blocks.
const (
	MBWhite      byte = 130
	MBBlack      byte = 131
	MBAir        byte = 132
	MBWater      byte = 133
	MBLava       byte = 134
	PortalAir    byte = 160
	PortalWater  byte = 161
	PortalLava   byte = 162
	PortalBlue   byte = 175
	PortalOrange byte = 176
	DoorLogAir   byte = 201
)

// SpecialType identifies what kind of special block a coordinate holds.
type SpecialType int

const (
	SpecialNone    SpecialType = 0
	SpecialMessage SpecialType = 1
	SpecialPortal  SpecialType = 2
	SpecialDoor    SpecialType = 3
)

// Entry stores metadata for a special block at a coordinate.
type SpecialEntry struct {
	Type        SpecialType
	Message     string // for message blocks
	PortalDst   [3]int // for portals: destination coords
	PortalLevel string // for portals: destination level (empty = same)
	DoorBlock   byte   // for doors: the solid block to restore
}

// key packs coordinates into a single int64 for map key.
func key(x, y, z int) int64 {
	return int64(x) | int64(y)<<20 | int64(z)<<40
}

// Registry stores special blocks for a single level.
type SpecialRegistry struct {
	mu      sync.RWMutex
	entries map[int64]*SpecialEntry
}

// NewRegistry creates an empty special block registry.
func NewSpecialRegistry() *SpecialRegistry {
	return &SpecialRegistry{entries: make(map[int64]*SpecialEntry)}
}

// Set registers a special block at the given coordinates.
func (r *SpecialRegistry) Set(x, y, z int, e *SpecialEntry) {
	r.mu.Lock()
	r.entries[key(x, y, z)] = e
	r.mu.Unlock()
}

// Get returns the special block entry at the given coordinates, or nil.
func (r *SpecialRegistry) Get(x, y, z int) *SpecialEntry {
	r.mu.RLock()
	e := r.entries[key(x, y, z)]
	r.mu.RUnlock()
	return e
}

// Remove deletes a special block entry.
func (r *SpecialRegistry) Remove(x, y, z int) {
	r.mu.Lock()
	delete(r.entries, key(x, y, z))
	r.mu.Unlock()
}

// IsMessageBlock returns true if the block ID is a message block type.
func IsMessageBlock(b byte) bool {
	return b >= MBWhite && b <= MBLava
}

// IsPortal returns true if the block ID is a portal type.
func IsPortal(b byte) bool {
	switch b {
	case PortalAir, PortalWater, PortalLava, PortalBlue, PortalOrange:
		return true
	}
	return false
}

// IsDoor returns true if the block ID is a door type.
func IsDoor(b byte) bool {
	return b == DoorLogAir
}

// IsTNT returns true if the block ID is a TNT type.
func IsTNT(b byte) bool {
	switch b {
	case TNTSmall, TNTBig, TNTNuke:
		return true
	}
	return false
}

// TNTRadius returns the explosion radius for a TNT type.
func TNTRadius(b byte) int {
	switch b {
	case TNTSmall:
		return 3
	case TNTBig:
		return 4
	case TNTNuke:
		return 7
	}
	return 0
}

// IsSpecialBlock returns true if the block ID is any special block type.
func IsSpecialBlock(b byte) bool {
	return IsMessageBlock(b) || IsPortal(b) || IsDoor(b) || IsTNT(b)
}
