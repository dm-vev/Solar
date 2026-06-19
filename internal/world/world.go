package world

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/solar-mc/solar/internal/generator"
)

// FromGeneratorLevel converts a generator.Level to a world.Level.
// This centralises the mapping so that server and protocol code do not
// duplicate the field-by-field copy.
func FromGeneratorLevel(lvl *generator.Level) Level {
	return Level{
		Name:   lvl.Name,
		Width:  lvl.Width,
		Height: lvl.Height,
		Length: lvl.Length,
		Blocks: lvl.Blocks,
		Spawn: Spawn{
			X:     lvl.Spawn.X,
			Y:     lvl.Spawn.Y,
			Z:     lvl.Spawn.Z,
			Yaw:   lvl.Spawn.Yaw,
			Pitch: lvl.Spawn.Pitch,
		},
	}
}

const (
	fileMagic   = "SWLD"
	fileVersion = 1
	maxBlocks   = 64 * 1024 * 1024
)

// Spawn describes the spawn point for a world.
type Spawn struct {
	X     int
	Y     int
	Z     int
	Yaw   byte
	Pitch byte
}

// Level is the minimal world state needed by the bootstrap protocol.
type Level struct {
	Name   string
	Width  int
	Height int
	Length int
	Blocks []byte
	Spawn  Spawn
}

// Volume returns the number of blocks in the level.
func (l Level) Volume() int {
	return l.Width * l.Height * l.Length
}

// Manager owns the current world snapshot.
type Manager struct {
	mu    sync.RWMutex
	level Level
	ticks atomic.Uint64
}

// NewManager creates the world manager with a single bootstrap world.
func NewManager() *Manager {
	return &Manager{
		level: bootstrapLevel(),
	}
}

// Current returns a copy of the active world.
func (m *Manager) Current() Level {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneLevel(m.level)
}

// SetCurrent replaces the active world snapshot.
func (m *Manager) SetCurrent(level Level) {
	level = normalizeLevel(level)
	m.mu.Lock()
	m.level = level
	m.mu.Unlock()
}

// Load replaces the active world snapshot with data from disk.
func (m *Manager) Load(path string) error {
	level, err := LoadLevel(path)
	if err != nil {
		return err
	}
	m.SetCurrent(level)
	return nil
}

// Save persists the active world snapshot to disk.
func (m *Manager) Save(path string) error {
	return m.Current().Save(path)
}

// SetBlock updates a single block in the active world snapshot.
func (m *Manager) SetBlock(x, y, z int, block byte) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !contains(m.level, x, y, z) {
		return false
	}

	m.level.Blocks[packIndex(m.level, x, y, z)] = block
	return true
}

// SetSpawn updates the spawn point of the active world snapshot.
func (m *Manager) SetSpawn(spawn Spawn) {
	m.mu.Lock()
	m.level.Spawn = spawn
	m.mu.Unlock()
}

// BlockAt returns the block at the given coordinates.
func (m *Manager) BlockAt(x, y, z int) (byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if !contains(m.level, x, y, z) {
		return 0, false
	}

	return m.level.Blocks[packIndex(m.level, x, y, z)], true
}

// Tick advances the world clock by one step.
func (m *Manager) Tick() {
	m.ticks.Add(1)
}

// TickCount returns how many ticks the world has processed.
func (m *Manager) TickCount() uint64 {
	return m.ticks.Load()
}

// LoadAsync loads world data on a background goroutine.
func (m *Manager) LoadAsync(path string) <-chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- m.Load(path)
	}()
	return ch
}

// SaveAsync saves world data on a background goroutine.
func (m *Manager) SaveAsync(path string) <-chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- m.Save(path)
	}()
	return ch
}

func cloneLevel(level Level) Level {
	clone := level
	if level.Blocks != nil {
		clone.Blocks = append([]byte(nil), level.Blocks...)
	}
	return clone
}

func normalizeLevel(level Level) Level {
	level = cloneLevel(level)
	if level.Width < 1 {
		level.Width = 1
	}
	if level.Height < 1 {
		level.Height = 1
	}
	if level.Length < 1 {
		level.Length = 1
	}
	volume := level.Volume()
	if volume > maxBlocks {
		volume = maxBlocks
	}
	if len(level.Blocks) != volume {
		blocks := make([]byte, volume)
		copy(blocks, level.Blocks)
		level.Blocks = blocks
	}
	return level
}

func validateLevelBounds(level Level) error {
	if level.Width < 1 || level.Height < 1 || level.Length < 1 {
		return fmt.Errorf("world dimensions must be positive")
	}
	volume := int64(level.Width) * int64(level.Height) * int64(level.Length)
	if volume < 1 || volume > maxBlocks {
		return fmt.Errorf("world volume %d exceeds limit %d", volume, maxBlocks)
	}
	if len(level.Blocks) > maxBlocks {
		return fmt.Errorf("block payload length %d exceeds limit %d", len(level.Blocks), maxBlocks)
	}
	return nil
}

func contains(level Level, x, y, z int) bool {
	return x >= 0 && y >= 0 && z >= 0 && x < level.Width && y < level.Height && z < level.Length
}

func packIndex(level Level, x, y, z int) int {
	return ((y*level.Length)+z)*level.Width + x
}
