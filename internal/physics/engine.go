// Package physics implements per-level block physics for Solar.
// It is a Go reimplementation of MCGalaxy's physics engine:
//   - Check list (cells to process) + update list (block changes to apply)
//   - Block ID → handler dispatch table (flat array, O(1))
//   - RandomFlow water/lava with 5-direction bit tracking
//   - Sand/gravel falling, grass grow/die, leaf decay, fire spread
//
// The engine runs in the server tick loop (20 TPS by default).
// Block changes from physics are broadcast to all players on the level.
package physics

import (
	"math/rand"
	"sync"
)

// Block IDs (matching Minecraft Classic / MCGalaxy).
const (
	Air          byte = 0
	Stone        byte = 1
	Grass        byte = 2
	Dirt         byte = 3
	Water        byte = 8
	StillWater   byte = 9
	Lava         byte = 10
	StillLava    byte = 11
	Sand         byte = 12
	Gravel       byte = 13
	Log          byte = 17
	Leaves       byte = 18
	Sponge       byte = 19
	Glass        byte = 20
	Fire         byte = 54
	FastLava     byte = 112
	FloatWood    byte = 110
	LavaSponge   byte = 109
	TNTSmall     byte = 182
	TNTBig       byte = 183
	TNTExplosion byte = 184
	TNTNuke      byte = 186
	Invalid      byte = 255
)

// Physics modes.
const (
	ModeOff      = 0
	ModeNormal   = 1
	ModeAdvanced = 2
)

const removeFlag = 255

// Engine runs block physics for a single level.
// It accesses blocks through getBlock/setBlock callbacks so it shares
// the same block array as the world.Manager (no copy divergence).
type Engine struct {
	mu        sync.Mutex
	width     int
	height    int
	length    int
	mode      int
	checks    []checkEntry
	updates   []updateEntry
	rng       *rand.Rand
	getBlk    func(idx int) byte
	setBlk    func(idx int, block byte)
	broadcast func(x, y, z int, block byte)
}

type checkEntry struct {
	index int
	data  byte
}

type updateEntry struct {
	index int
	block byte
}

// New creates a physics engine for a level with the given dimensions.
// getBlock returns the block at a flat index; setBlock stages a block
// change (applied at end of tick). broadcast is called for each block
// change so clients see the update.
func New(width, height, length int, getBlk func(int) byte, setBlk func(int, byte), broadcast func(x, y, z int, block byte)) *Engine {
	return &Engine{
		width:     width,
		height:    height,
		length:    length,
		mode:      ModeNormal,
		rng:       rand.New(rand.NewSource(rand.Int63())),
		getBlk:    getBlk,
		setBlk:    setBlk,
		broadcast: broadcast,
	}
}

func (e *Engine) Mode() int {
	e.mu.Lock()
	m := e.mode
	e.mu.Unlock()
	return m
}

func (e *Engine) SetMode(mode int) {
	e.mu.Lock()
	e.mode = mode
	e.mu.Unlock()
}

// Queue adds a block at the given coordinates for processing next tick.
func (e *Engine) Queue(x, y, z int) {
	idx := e.posToInt(x, y, z)
	if idx < 0 {
		return
	}
	e.mu.Lock()
	e.checks = append(e.checks, checkEntry{index: idx})
	e.mu.Unlock()
}

// Tick processes one physics tick.
func (e *Engine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeOff || len(e.checks) == 0 {
		return
	}

	adv := e.mode >= ModeAdvanced
	checks := e.checks
	e.checks = e.checks[:0]

	for i := range checks {
		c := &checks[i]
		if c.index < 0 {
			continue
		}
		block := e.getBlock(c.index)
		e.processBlock(c, block, adv)
		if c.data != removeFlag {
			e.checks = append(e.checks, *c)
		}
	}

	// Apply updates.
	for _, u := range e.updates {
		e.applyUpdate(u)
	}
	e.updates = e.updates[:0]
}

func (e *Engine) processBlock(c *checkEntry, block byte, adv bool) {
	x, y, z := e.intToPos(c.index)

	switch block {
	case Air:
		// Air does nothing in basic physics.

	case Water, StillWater:
		e.doLiquid(c, x, y, z, block, false, adv)

	case Lava, StillLava:
		e.doLiquid(c, x, y, z, block, true, adv)

	case FastLava:
		e.doLiquid(c, x, y, z, block, true, adv)

	case Sand, Gravel:
		e.doFalling(c, x, y, z, block, adv)

	case Dirt:
		e.doGrassGrow(c, x, y, z)

	case Grass:
		e.doGrassDie(c, x, y, z)

	case Leaves:
		e.doLeafDecay(c, x, y, z)

	case Fire:
		e.doFire(c, x, y, z, adv)

	case TNTSmall, TNTBig, TNTNuke:
		e.doTNT(c, x, y, z, block, adv)

	case Sponge:
		e.doSponge(c, x, y, z, false)

	case LavaSponge:
		e.doSponge(c, x, y, z, true)

	default:
		c.data = removeFlag
	}
}

// ─── coordinate helpers ───

func (e *Engine) posToInt(x, y, z int) int {
	if x < 0 || x >= e.width || y < 0 || y >= e.height || z < 0 || z >= e.length {
		return -1
	}
	return x + e.width*(z+e.length*y)
}

func (e *Engine) intToPos(idx int) (x, y, z int) {
	y = idx / (e.width * e.length)
	rem := idx - y*e.width*e.length
	z = rem / e.width
	x = rem - z*e.width
	return
}

func (e *Engine) getBlock(idx int) byte {
	if idx < 0 {
		return Invalid
	}
	return e.getBlk(idx)
}

func (e *Engine) setBlock(idx int, block byte) {
	if idx >= 0 {
		e.updates = append(e.updates, updateEntry{index: idx, block: block})
	}
}

func (e *Engine) queueCheck(idx int) {
	if idx >= 0 {
		e.checks = append(e.checks, checkEntry{index: idx})
	}
}

func (e *Engine) applyUpdate(u updateEntry) {
	if u.index < 0 {
		return
	}
	if e.getBlk(u.index) != u.block {
		e.setBlk(u.index, u.block)
		x, y, z := e.intToPos(u.index)
		if e.broadcast != nil {
			e.broadcast(x, y, z, u.block)
		}
		e.activateNeighbours(u.index)
	}
}

func (e *Engine) activateNeighbours(idx int) {
	w := e.width
	wl := e.width * e.length
	e.queueCheck(idx + 1)
	e.queueCheck(idx - 1)
	e.queueCheck(idx + w)
	e.queueCheck(idx - w)
	e.queueCheck(idx + wl)
	e.queueCheck(idx - wl)
}

// ─── liquid physics (water + lava) ───

func (e *Engine) doLiquid(c *checkEntry, x, y, z int, block byte, isLava, adv bool) {
	// Lava delay: upper 3 bits of data must reach 4<<5 before flowing.
	if isLava && block != FastLava {
		if c.data < 4<<5 {
			c.data += 1 << 5
			return
		}
	}

	// Random flow: 5 directions, 25% chance each per tick.
	const (
		flowedXMax = 1 << 0
		flowedXMin = 1 << 1
		flowedZMax = 1 << 2
		flowedZMin = 1 << 3
		flowedYMin = 1 << 4
		flowedAll  = 0x1F
	)

	flow := c.data & flowedAll
	done := true

	// Down (-Y) — always try first.
	if flow&flowedYMin == 0 {
		if e.rng.Intn(4) == 0 || !e.liquidBlocked(e.posToInt(x, y-1, z), isLava, adv) {
			e.trySpread(e.posToInt(x, y-1, z), block, isLava, adv)
			flow |= flowedYMin
		}
	}
	if flow&flowedXMax == 0 {
		if e.rng.Intn(4) == 0 || !e.liquidBlocked(e.posToInt(x+1, y, z), isLava, adv) {
			e.trySpread(e.posToInt(x+1, y, z), block, isLava, adv)
			flow |= flowedXMax
		}
	}
	if flow&flowedXMin == 0 {
		if e.rng.Intn(4) == 0 || !e.liquidBlocked(e.posToInt(x-1, y, z), isLava, adv) {
			e.trySpread(e.posToInt(x-1, y, z), block, isLava, adv)
			flow |= flowedXMin
		}
	}
	if flow&flowedZMax == 0 {
		if e.rng.Intn(4) == 0 || !e.liquidBlocked(e.posToInt(x, y, z+1), isLava, adv) {
			e.trySpread(e.posToInt(x, y, z+1), block, isLava, adv)
			flow |= flowedZMax
		}
	}
	if flow&flowedZMin == 0 {
		if e.rng.Intn(4) == 0 || !e.liquidBlocked(e.posToInt(x, y, z-1), isLava, adv) {
			e.trySpread(e.posToInt(x, y, z-1), block, isLava, adv)
			flow |= flowedZMin
		}
	}

	if flow != flowedAll {
		done = false
	}

	if done {
		c.data = removeFlag
	} else {
		c.data = flow
	}
}

func (e *Engine) liquidBlocked(idx int, isLava, adv bool) bool {
	b := e.getBlock(idx)
	switch b {
	case Air, Invalid:
		return false
	case Water, StillWater:
		return !isLava
	case Lava, StillLava, FastLava:
		return isLava
	case Sand, Gravel, FloatWood:
		return false
	}
	return true
}

func (e *Engine) trySpread(idx int, block byte, isLava, adv bool) {
	if idx < 0 {
		return
	}
	target := e.getBlock(idx)
	switch target {
	case Air:
		if !e.checkSponge(idx, isLava) {
			e.setBlock(idx, block)
		}
	case Water, StillWater:
		if isLava {
			e.setBlock(idx, Stone)
		}
	case Lava, StillLava, FastLava:
		if !isLava {
			e.setBlock(idx, Stone)
		}
	case Sand:
		if isLava && adv {
			e.setBlock(idx, Glass)
		} else {
			e.queueCheck(idx)
		}
	case Gravel:
		e.queueCheck(idx)
	default:
		if adv {
			if isLava && e.lavaKills(target) {
				e.setBlock(idx, Air)
			} else if !isLava && e.waterKills(target) {
				e.setBlock(idx, Air)
			}
		}
	}
}

func (e *Engine) checkSponge(idx int, isLava bool) bool {
	x, y, z := e.intToPos(idx)
	target := Sponge
	if isLava {
		target = LavaSponge
	}
	for dy := -2; dy <= 2; dy++ {
		for dz := -2; dz <= 2; dz++ {
			for dx := -2; dx <= 2; dx++ {
				if e.getBlock(e.posToInt(x+dx, y+dy, z+dz)) == target {
					return true
				}
			}
		}
	}
	return false
}

// ─── sand / gravel falling ───

func (e *Engine) doFalling(c *checkEntry, x, y, z int, block byte, adv bool) {
	idx := e.posToInt(x, y, z)
	movedDown := false

	for yCur := y - 1; yCur >= 0; yCur-- {
		below := e.getBlock(e.posToInt(x, yCur, z))
		if below == Invalid {
			break
		}
		switch below {
		case Air, Water, StillWater, Lava, StillLava:
			movedDown = true
			continue
		case Sapling:
			if adv {
				movedDown = true
				continue
			}
		}
		break
	}

	if movedDown {
		e.setBlock(idx, Air)
		if adv {
			// Advanced: land exactly at hit position.
			landIdx := e.posToInt(x, y-1, z)
			for yCur := y - 1; yCur >= 0; yCur-- {
				below := e.getBlock(e.posToInt(x, yCur, z))
				if below == Air || below == Water || below == StillWater || below == Lava || below == StillLava {
					landIdx = e.posToInt(x, yCur, z)
					continue
				}
				break
			}
			e.setBlock(landIdx, block)
		} else {
			// Normal: land one above the first solid block.
			landY := y - 1
			for yCur := y - 1; yCur >= 0; yCur-- {
				below := e.getBlock(e.posToInt(x, yCur, z))
				if below == Air || below == Water || below == StillWater || below == Lava || below == StillLava {
					landY = yCur
					continue
				}
				break
			}
			e.setBlock(e.posToInt(x, landY, z), block)
		}
		e.activateNeighbours(idx)
	}
	c.data = removeFlag
}

// ─── grass grow / die ───

func (e *Engine) doGrassGrow(c *checkEntry, x, y, z int) {
	if c.data > 20 {
		above := e.getBlock(e.posToInt(x, y+1, z))
		if e.lightPasses(above) {
			e.setBlock(c.index, Grass)
		}
		c.data = removeFlag
	} else {
		c.data++
	}
}

func (e *Engine) doGrassDie(c *checkEntry, x, y, z int) {
	if c.data > 20 {
		above := e.getBlock(e.posToInt(x, y+1, z))
		if !e.lightPasses(above) {
			e.setBlock(c.index, Dirt)
		}
		c.data = removeFlag
	} else {
		c.data++
	}
}

func (e *Engine) lightPasses(b byte) bool {
	switch b {
	case Air, Glass, Leaves, Invalid:
		return true
	}
	return false
}

// ─── leaf decay ───

func (e *Engine) doLeafDecay(c *checkEntry, x, y, z int) {
	if c.data < 5 {
		if e.rng.Intn(10) == 0 {
			c.data++
		}
		return
	}
	// Search for log within range 4.
	if !e.hasNearbyLog(x, y, z) {
		e.setBlock(c.index, Air)
	}
	c.data = removeFlag
}

func (e *Engine) hasNearbyLog(x, y, z int) bool {
	for dy := -4; dy <= 4; dy++ {
		for dz := -4; dz <= 4; dz++ {
			for dx := -4; dx <= 4; dx++ {
				if e.getBlock(e.posToInt(x+dx, y+dy, z+dz)) == Log {
					return true
				}
			}
		}
	}
	return false
}

// ─── fire spread ───

func (e *Engine) doFire(c *checkEntry, x, y, z int, adv bool) {
	if c.data < 2 {
		c.data++
		return
	}

	// Random spread to adjacent air.
	if e.rng.Intn(20) == 0 && c.data%2 == 0 {
		dir := e.rng.Intn(6)
		dx, dy, dz := 0, 0, 0
		switch dir {
		case 0:
			dx = -1
		case 1:
			dx = 1
		case 2:
			dy = -1
		case 3:
			dy = 1
		case 4:
			dz = -1
		case 5:
			dz = 1
		}
		nidx := e.posToInt(x+dx, y+dy, z+dz)
		if e.getBlock(nidx) == Air {
			e.setBlock(nidx, Fire)
		}
	}

	if adv {
		// Spread to adjacent flammable blocks.
		if c.data >= 4 {
			for _, off := range [6][3]int{{-1, 0, 0}, {1, 0, 0}, {0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}} {
				nidx := e.posToInt(x+off[0], y+off[1], z+off[2])
				nb := e.getBlock(nidx)
				if e.lavaKills(nb) && nb != Air {
					e.setBlock(nidx, Fire)
				}
			}
		}
	}

	// Burnout.
	c.data++
	if c.data > 5 {
		r := e.rng.Intn(10)
		if r < 2 {
			e.setBlock(c.index, Air)
		} else if r < 4 {
			c.data = 3 // keep burning
		} else {
			e.setBlock(c.index, Air)
		}
	}
}

// ─── sponge ───

func (e *Engine) doSponge(c *checkEntry, x, y, z int, isLava bool) {
	// Absorb nearby liquid.
	liquid := []byte{Water, StillWater}
	if isLava {
		liquid = []byte{Lava, StillLava}
	}
	for dy := -2; dy <= 2; dy++ {
		for dz := -2; dz <= 2; dz++ {
			for dx := -2; dx <= 2; dx++ {
				idx := e.posToInt(x+dx, y+dy, z+dz)
				b := e.getBlock(idx)
				for _, l := range liquid {
					if b == l {
						e.setBlock(idx, Air)
					}
				}
			}
		}
	}
	c.data = removeFlag
}

// ─── block property helpers ───

func (e *Engine) waterKills(b byte) bool {
	switch b {
	case Sapling:
		return true
	}
	return false
}

func (e *Engine) lavaKills(b byte) bool {
	switch b {
	case Wood, Log, Leaves, Sponge, Sapling:
		return true
	}
	return false
}

// Sapling is block ID 6, Wood is 5 — need these for waterKills/lavaKills.
const (
	Sapling byte = 6
	Wood    byte = 5
)

// ─── TNT explosion (ported from MCGalaxy TntPhysics) ───

func (e *Engine) doTNT(c *checkEntry, x, y, z int, block byte, adv bool) {
	// MCGalaxy: physics < 3 → just remove TNT (no explosion).
	// physics >= 3 → fuse delay (5 ticks) before explosion.
	// physics >= 2 (advanced) → explode immediately via MakeExplosion.
	// Solar: adv = mode >= 2. We explode immediately in advanced mode.
	if !adv {
		e.setBlock(e.posToInt(x, y, z), Air)
		c.data = removeFlag
		return
	}

	power := 0 // SmallTNT
	switch block {
	case TNTBig:
		power = 1
	case TNTNuke:
		power = 4
	}
	e.makeExplosion(x, y, z, power)
	c.data = removeFlag
}

// makeExplosion mirrors MCGalaxy's MakeExplosion + Explode:
// 3 layered passes with increasing radius and decreasing probability.
func (e *Engine) makeExplosion(x, y, z, size int) {
	// Set center to TNT_Explosion (visual block 184).
	centerIdx := e.posToInt(x, y, z)
	if centerIdx >= 0 {
		e.setBlock(centerIdx, TNTExplosion)
	}
	// 3 layers: always destroy, 70% destroy, 30% destroy.
	e.explodeLayer(x, y, z, size+1, -1)
	e.explodeLayer(x, y, z, size+2, 7)
	e.explodeLayer(x, y, z, size+3, 3)
}

// explodeLayer destroys blocks in a cube of the given radius.
// prob < 0 means always destroy. Otherwise prob/10 chance per block.
func (e *Engine) explodeLayer(cx, cy, cz, size, prob int) {
	for x := cx - size; x <= cx+size; x++ {
		for y := cy - size; y <= cy+size; y++ {
			for z := cz - size; z <= cz+size; z++ {
				idx := e.posToInt(x, y, z)
				if idx < 0 {
					continue
				}
				b := e.getBlock(idx)
				if b == Invalid || b == Air {
					continue
				}
				doDestroy := prob < 0 || e.rng.Intn(10) < prob
				if !doDestroy {
					continue
				}
				// Chain reaction: TNT blocks get converted to TNT_Small.
				if b == TNTSmall || b == TNTBig || b == TNTNuke {
					e.queueCheck(idx)
					continue
				}
				// 3 outcomes (matching MCGalaxy):
				// 40% → TNT_Explosion (visual)
				// 40% → Air (destroyed)
				// 20% → Air (was Drop+Dissipate, simplified to Air)
				mode := e.rng.Intn(10) + 1
				switch {
				case mode <= 4:
					e.setBlock(idx, TNTExplosion)
				case mode <= 8:
					e.setBlock(idx, Air)
				default:
					e.setBlock(idx, Air)
				}
			}
		}
	}
}

// IsTNTBlock checks if a block is TNT (exported for physics dispatch).
func IsTNTBlock(b byte) bool {
	switch b {
	case TNTSmall, TNTBig, TNTNuke:
		return true
	}
	return false
}
