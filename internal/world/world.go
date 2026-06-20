package world

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"sync"
	"sync/atomic"

	"github.com/solar-mc/solar/internal/generator"
)

var (
	gzipWriterPool = sync.Pool{
		New: func() any {
			if w, err := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed); err == nil {
				return w
			}
			return gzip.NewWriter(io.Discard)
		},
	}
	flateWriterPool = sync.Pool{
		New: func() any {
			if w, err := flate.NewWriter(io.Discard, flate.BestSpeed); err == nil {
				return w
			}
			w, _ := flate.NewWriter(io.Discard, flate.DefaultCompression)
			return w
		},
	}
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
		Env: DefaultEnv(),
	}
}

const (
	fileMagic   = "SWLD"
	fileVersion = 1
)

// maxBlocks is the hard cap on world volume. Set once at startup via
// SetMaxBlocks; defaults to 64M blocks.
var maxBlocks int64 = 64 * 1024 * 1024

// SetMaxBlocks updates the global world volume limit. Must be called
// before any world is loaded or generated.
func SetMaxBlocks(n int) {
	if n > 0 {
		maxBlocks = int64(n)
	}
}

// MaxBlocks returns the current world volume limit.
func MaxBlocks() int64 {
	return maxBlocks
}

// Spawn describes the spawn point for a world.
type Spawn struct {
	X     int
	Y     int
	Z     int
	Yaw   byte
	Pitch byte
}

// EnvColor is an RGB color for a CPE environment slot.
type EnvColor struct {
	R, G, B byte
	Set     bool // false = use client default
}

// Env holds per-level environment properties that drive CPE packets.
type Env struct {
	Weather        byte  // 0=sunny, 1=raining, 2=snowing
	EdgeLevel      int16 // map sides bottom level
	SidesLevel     int16 // map sides top level
	CloudsLevel    int16 // clouds height
	MaxFog         int16 // max fog distance (0 = default)
	ExpFog         bool
	CloudsSpeed    int32 // clouds speed * 256
	WeatherSpeed   int32 // weather speed * 256
	WeatherFade    int32 // weather fade * 256
	SkyboxHorSpeed int32
	SkyboxVerSpeed int32
	Colors         [5]EnvColor // 0=sky, 1=cloud, 2=fog, 3=ambient, 4=diffuse
	LightingMode   byte        // 0=classic, 1=fancy
	LightingLock   bool
	MOTD           string
}

// DefaultEnv returns environment defaults (classic Minecraft look).
func DefaultEnv() Env {
	return Env{
		EdgeLevel:   -1, // signals "not set"
		SidesLevel:  -1,
		CloudsLevel: -1,
		MaxFog:      -1,
	}
}

// Level is the minimal world state needed by the bootstrap protocol.
type Level struct {
	Name   string
	Width  int
	Height int
	Length int
	Blocks []byte
	Spawn  Spawn
	Env    Env
}

// Volume returns the number of blocks in the level.
func (l Level) Volume() int {
	return l.Width * l.Height * l.Length
}

// LevelStream holds a snapshot of the world plus a pre-compressed payload
// for sending to clients. The payload is gzip or raw flate data depending on
// the client's FastMap support.
type LevelStream struct {
	Level   Level
	Payload []byte
	FastMap bool
}

// Manager owns the current world snapshot.
type Manager struct {
	mu          sync.RWMutex
	level       Level
	ticks       atomic.Uint64
	generation  uint64 // bumped on SetCurrent/SetSpawn (full-world changes)
	blocksDirty bool   // set on SetBlock, cleared on next LevelStream re-compress
	cache       levelCache
}

type levelCache struct {
	generation uint64
	gzipData   []byte
	flateData  []byte
}

// NewManager creates the world manager with a single bootstrap world.
func NewManager() *Manager {
	return &Manager{
		level: bootstrapLevel(),
	}
}

// Current returns a copy of the active world. The Blocks slice is cloned
// so callers can mutate it without affecting the shared state.
func (m *Manager) Current() Level {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneLevel(m.level)
}

// Spawn returns the spawn point of the active world snapshot. This is a
// safe accessor for callers that only need the spawn (e.g. computing a
// fallback position) without cloning the entire block array.
func (m *Manager) Spawn() Spawn {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.level.Spawn
}

// SetCurrent replaces the active world snapshot.
func (m *Manager) SetCurrent(level Level) {
	level = normalizeLevel(level)
	m.mu.Lock()
	m.level = level
	m.generation++
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
	m.blocksDirty = true
	return true
}

// SetSpawn updates the spawn point of the active world snapshot.
func (m *Manager) SetSpawn(spawn Spawn) {
	m.mu.Lock()
	m.level.Spawn = spawn
	m.generation++
	m.mu.Unlock()
}

// LevelStream returns a consistent snapshot of the level with a pre-compressed
// payload suitable for streaming to clients. The compressed payload is cached and
// reused until the world is modified.
func (m *Manager) LevelStream(fastMap bool) (LevelStream, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cache.generation != m.generation || m.blocksDirty {
		m.cache.generation = m.generation
		m.blocksDirty = false
		m.cache.gzipData = nil
		m.cache.flateData = nil
	}

	payload, err := m.cachedPayload(fastMap)
	if err != nil {
		return LevelStream{}, err
	}

	return LevelStream{
		Level:   m.level,
		Payload: append([]byte(nil), payload...),
		FastMap: fastMap,
	}, nil
}

// cachedPayload returns the cached payload for the requested format, compressing
// it on demand if necessary.
func (m *Manager) cachedPayload(fastMap bool) ([]byte, error) {
	if fastMap {
		if len(m.cache.flateData) > 0 {
			return m.cache.flateData, nil
		}
		payload, err := compressLevel(m.level, true)
		if err != nil {
			return nil, err
		}
		m.cache.flateData = payload
		return payload, nil
	}

	if len(m.cache.gzipData) > 0 {
		return m.cache.gzipData, nil
	}
	payload, err := compressLevel(m.level, false)
	if err != nil {
		return nil, err
	}
	m.cache.gzipData = payload
	return payload, nil
}

func compressLevel(level Level, fastMap bool) ([]byte, error) {
	var buf bytes.Buffer
	if fastMap {
		w := flateWriterPool.Get().(*flate.Writer)
		w.Reset(&buf)
		defer flateWriterPool.Put(w)
		if err := writeLevelPayload(w, level); err != nil {
			return nil, err
		}
	} else {
		w := gzipWriterPool.Get().(*gzip.Writer)
		w.Reset(&buf)
		defer gzipWriterPool.Put(w)
		if err := writeLevelPayload(w, level); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func writeLevelPayload(writer io.WriteCloser, level Level) error {
	defer writer.Close()

	header := make([]byte, 4)
	header[0] = byte(level.Volume() >> 24)
	header[1] = byte(level.Volume() >> 16)
	header[2] = byte(level.Volume() >> 8)
	header[3] = byte(level.Volume())
	if _, err := writer.Write(header); err != nil {
		return fmt.Errorf("write map header: %w", err)
	}
	if _, err := writer.Write(level.Blocks); err != nil {
		return fmt.Errorf("write map blocks: %w", err)
	}
	return writer.Close()
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
	if int64(volume) > maxBlocks {
		volume = int(maxBlocks)
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
	if int64(len(level.Blocks)) > maxBlocks {
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
