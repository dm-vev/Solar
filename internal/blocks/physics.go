// Package physics implements per-level block physics for Solar.
// It is a Go reimplementation of MCGalaxy's physics engine:
//   - Check list (cells to process) + update list (block changes to apply)
//   - Block ID → handler dispatch table (flat array, O(1))
//   - RandomFlow water/lava with 5-direction bit tracking
//   - Sand/gravel falling, grass grow/die, leaf decay, fire spread
//
// The engine runs in the server tick loop (20 TPS by default).
// Block changes from physics are broadcast to all players on the level.
package blocks

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
	CoalOre      byte = 16
	Obsidian     byte = 49
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
type PhysicsEngine struct {
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
func NewPhysics(width, height, length int, getBlk func(int) byte, setBlk func(int, byte), broadcast func(x, y, z int, block byte)) *PhysicsEngine {
	return &PhysicsEngine{
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

func (e *PhysicsEngine) Mode() int {
	e.mu.Lock()
	m := e.mode
	e.mu.Unlock()
	return m
}

func (e *PhysicsEngine) SetMode(mode int) {
	e.mu.Lock()
	e.mode = mode
	e.mu.Unlock()
}

// Queue adds a block at the given coordinates for processing next tick.
func (e *PhysicsEngine) Queue(x, y, z int) {
	idx := e.posToInt(x, y, z)
	if idx < 0 {
		return
	}
	e.mu.Lock()
	e.checks = append(e.checks, checkEntry{index: idx})
	e.mu.Unlock()
}

// Tick processes one physics tick.
func (e *PhysicsEngine) Tick() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.mode == ModeOff || len(e.checks) == 0 {
		return
	}

	adv := e.mode >= ModeAdvanced
	checks := make([]checkEntry, len(e.checks))
	copy(checks, e.checks)
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

func (e *PhysicsEngine) processBlock(c *checkEntry, block byte, adv bool) {
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

func (e *PhysicsEngine) posToInt(x, y, z int) int {
	if x < 0 || x >= e.width || y < 0 || y >= e.height || z < 0 || z >= e.length {
		return -1
	}
	return x + e.width*(z+e.length*y)
}

func (e *PhysicsEngine) intToPos(idx int) (x, y, z int) {
	y = idx / (e.width * e.length)
	rem := idx - y*e.width*e.length
	z = rem / e.width
	x = rem - z*e.width
	return
}

func (e *PhysicsEngine) getBlock(idx int) byte {
	if idx < 0 {
		return Invalid
	}
	return e.getBlk(idx)
}

func (e *PhysicsEngine) setBlock(idx int, block byte) {
	if idx >= 0 {
		e.updates = append(e.updates, updateEntry{index: idx, block: block})
	}
}

func (e *PhysicsEngine) queueCheck(idx int) {
	if idx >= 0 {
		e.checks = append(e.checks, checkEntry{index: idx})
	}
}

func (e *PhysicsEngine) applyUpdate(u updateEntry) {
	if u.index < 0 {
		return
	}
	if e.getBlk(u.index) != u.block {
		e.setBlk(u.index, u.block)
		x, y, z := e.intToPos(u.index)
		if e.broadcast != nil {
			e.broadcast(x, y, z, u.block)
		}
		// Queue the new block itself for physics processing.
		e.queueCheck(u.index)
		e.activateNeighbours(u.index)
	}
}

func (e *PhysicsEngine) activateNeighbours(idx int) {
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

func (e *PhysicsEngine) doLiquid(c *checkEntry, x, y, z int, block byte, isLava, adv bool) {
	// Lava delay: upper 3 bits of data must reach 4<<5 (128) before flowing.
	// Matches MCGalaxy SimpleLiquidPhysics.DoLava.
	if isLava && block != FastLava {
		if c.data < (4 << 5) {
			c.data += 1 << 5
			return
		}
	}

	// Sponge check: if near sponge, remove this liquid block.
	if e.checkSponge(c.index, isLava) {
		e.setBlock(c.index, Air)
		c.data = removeFlag
		return
	}

	const (
		flowedXMax = 1 << 0
		flowedXMin = 1 << 1
		flowedZMax = 1 << 2
		flowedZMin = 1 << 3
		flowedYMin = 1 << 4
		flowedAll  = 0x1F
	)

	flow := c.data & flowedAll

	// Pass 1: random spread — 25% chance per direction per tick.
	// Matches MCGalaxy DoWaterRandowFlow / DoLavaRandowFlow.
	type dirInfo struct {
		flag byte
		idx  int
	}
	dirs := [5]dirInfo{
		{flowedXMax, e.posToInt(x + 1, y, z)},
		{flowedXMin, e.posToInt(x - 1, y, z)},
		{flowedZMax, e.posToInt(x, y, z + 1)},
		{flowedZMin, e.posToInt(x, y, z - 1)},
		{flowedYMin, e.posToInt(x, y - 1, z)},
	}

	for _, d := range dirs {
		if flow&d.flag == 0 && e.rng.Intn(4) == 0 {
			e.trySpread(d.idx, block, isLava, adv)
			flow |= d.flag
		}
	}

	// Pass 2: mark blocked directions (no spread, just mark as done).
	// Matches MCGalaxy: if not flowed and blocked, mark flowed without spreading.
	for _, d := range dirs {
		if flow&d.flag == 0 && e.liquidBlocked(d.idx, isLava, adv) {
			flow |= d.flag
		}
	}

	if flow == flowedAll {
		// All directions handled — remove from checks.
		// For lava (non-fast): re-randomize upper 3 bits for re-delay.
		// Matches MCGalaxy DoLavaRandowFlow: data &= mask; data |= rand.Next(3)<<5
		if isLava && block != FastLava {
			c.data = byte(e.rng.Intn(3)) << 5
		} else {
			c.data = removeFlag
		}
	} else {
		// Not all directions handled — keep in checks with updated flow data.
		// For lava (non-fast): re-randomize upper 3 bits for re-delay.
		if isLava && block != FastLava {
			c.data = flow | byte(e.rng.Intn(3))<<5
		} else {
			c.data = flow
		}
	}
}

func (e *PhysicsEngine) liquidBlocked(idx int, isLava, adv bool) bool {
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
	default:
		// In advanced mode, blocks that water/lava kills are not blocked.
		// Matches MCGalaxy WaterBlocked/LavaBlocked: if Props.WaterKills/LavaKills
		// and physics > 1, the liquid can spread into the block.
		if adv {
			if isLava && e.lavaKills(b) {
				return false
			}
			if !isLava && e.waterKills(b) {
				return false
			}
		}
		return true
	}
}

func (e *PhysicsEngine) trySpread(idx int, block byte, isLava, adv bool) {
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

func (e *PhysicsEngine) checkSponge(idx int, isLava bool) bool {
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
// Matches MCGalaxy OtherPhysics.DoFalling:
// - Advanced: scans one block down, moves there (one block per tick).
// - Normal: scans all the way down, lands one above first solid.

func (e *PhysicsEngine) doFalling(c *checkEntry, x, y, z int, block byte, adv bool) {
	idx := e.posToInt(x, y, z)
	movedDown := false
	landIdx := -1

	for yCur := y - 1; yCur >= 0; yCur-- {
		belowIdx := e.posToInt(x, yCur, z)
		below := e.getBlock(belowIdx)
		hitBlock := false

		switch below {
		case Air, Water, StillWater, Lava, StillLava:
			movedDown = true
			landIdx = belowIdx
		case Sapling:
			if adv {
				movedDown = true
				landIdx = belowIdx
			} else {
				hitBlock = true
			}
		default:
			hitBlock = true
		}

		if hitBlock {
			if !adv {
				landIdx = e.posToInt(x, yCur+1, z)
			}
			break
		}
		if adv {
			break
		}
	}

	if movedDown && landIdx >= 0 {
		e.setBlock(idx, Air)
		e.setBlock(landIdx, block)
		e.activateNeighbours(idx)
	}
	c.data = removeFlag
}

// ─── grass grow / die ───

func (e *PhysicsEngine) doGrassGrow(c *checkEntry, x, y, z int) {
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

func (e *PhysicsEngine) doGrassDie(c *checkEntry, x, y, z int) {
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

func (e *PhysicsEngine) lightPasses(b byte) bool {
	switch b {
	case Air, Glass, Leaves, Invalid:
		return true
	}
	return false
}

// ─── leaf decay ───
// Matches MCGalaxy LeafPhysics.DoLeafDecay: flood-fill distance through
// leaves. A leaf survives only if a log is reachable via a path of leaves
// within 4 steps.

func (e *PhysicsEngine) doLeafDecay(c *checkEntry, x, y, z int) {
	if c.data < 5 {
		if e.rng.Intn(10) == 0 {
			c.data++
		}
		return
	}
	if !e.leafConnectedToLog(x, y, z) {
		e.setBlock(c.index, Air)
	}
	c.data = removeFlag
}

// leafConnectedToLog implements MCGalaxy's DoLeafDecay flood-fill:
//   - 9×9×9 grid centered on the leaf (range 4)
//   - Log = distance 0, Leaves = unvisited (-2), Other = wall (-1)
//   - Propagate distances from logs through adjacent leaves (BFS)
//   - Leaf survives if its distance >= 0 (connected to a log through leaves)
const leafRange = 4

func (e *PhysicsEngine) leafConnectedToLog(x, y, z int) bool {
	const size = leafRange*2 + 1 // 9
	dists := make([]int, size*size*size)

	idx := 0
	for dy := -leafRange; dy <= leafRange; dy++ {
		for dz := -leafRange; dz <= leafRange; dz++ {
			for dx := -leafRange; dx <= leafRange; dx++ {
				b := e.getBlock(e.posToInt(x+dx, y+dy, z+dz))
				switch b {
				case Log:
					dists[idx] = 0
				case Leaves:
					dists[idx] = -2
				default:
					dists[idx] = -1
				}
				idx++
			}
		}
	}

	const oneX = 1
	const oneZ = size
	const oneY = size * size

	for dist := 1; dist <= leafRange; dist++ {
		idx = 0
		for dy := -leafRange; dy <= leafRange; dy++ {
			for dz := -leafRange; dz <= leafRange; dz++ {
				for dx := -leafRange; dx <= leafRange; dx++ {
					if dists[idx] != dist-1 {
						idx++
						continue
					}
					if dx > -leafRange && dists[idx-oneX] == -2 {
						dists[idx-oneX] = dist
					}
					if dx < leafRange && dists[idx+oneX] == -2 {
						dists[idx+oneX] = dist
					}
					if dy > -leafRange && dists[idx-oneY] == -2 {
						dists[idx-oneY] = dist
					}
					if dy < leafRange && dists[idx+oneY] == -2 {
						dists[idx+oneY] = dist
					}
					if dz > -leafRange && dists[idx-oneZ] == -2 {
						dists[idx-oneZ] = dist
					}
					if dz < leafRange && dists[idx+oneZ] == -2 {
						dists[idx+oneZ] = dist
					}
					idx++
				}
			}
		}
	}

	center := leafRange*oneX + leafRange*oneY + leafRange*oneZ
	return dists[center] >= 0
}

// ─── fire spread ───
// Matches MCGalaxy FirePhysics.Do:
// - Spread chance: 1/19 per tick (rand.Next(1,20)==1)
// - Advanced: diagonal expansion + direct neighbor expansion
// - Burnout: CoalOre (20%), Obsidian (20%), Air (40%), keep burning (10%)

func (e *PhysicsEngine) doFire(c *checkEntry, x, y, z int, adv bool) {
	if c.data < 2 {
		c.data++
		return
	}

	// Random spread to adjacent air (1/19 chance).
	if e.rng.Intn(19) == 0 && c.data%2 == 0 {
		max := e.rng.Intn(18) + 1 // 1..18, matching rand.Next(1, 19)
		switch {
		case max <= 3:
			e.expandToAir(x-1, y, z)
		case max <= 6:
			e.expandToAir(x+1, y, z)
		case max <= 9:
			e.expandToAir(x, y-1, z)
		case max <= 12:
			e.expandToAir(x, y+1, z)
		case max <= 15:
			e.expandToAir(x, y, z-1)
		case max <= 18:
			e.expandToAir(x, y, z+1)
		}
	}

	if adv {
		// Diagonal expansion: check 12 diagonal positions for flammable blocks.
		// If flammable, set fire to the axis-aligned blocks between.
		for dy := -1; dy <= 1; dy++ {
			for _, dz := range [2]int{-1, 1} {
				for _, dx := range [2]int{-1, 1} {
					e.expandDiagonal(x, y, z, dx, dy, dz)
				}
			}
		}

		// Delay before direct neighbor expansion.
		if c.data < 4 {
			c.data++
			return
		}

		// Direct neighbor expansion: TNT → explosion, flammable → fire.
		for _, off := range [6][3]int{{-1, 0, 0}, {1, 0, 0}, {0, -1, 0}, {0, 1, 0}, {0, 0, -1}, {0, 0, 1}} {
			nidx := e.posToInt(x+off[0], y+off[1], z+off[2])
			nb := e.getBlock(nidx)
			if nb == TNTSmall {
				e.makeExplosion(x+off[0], y+off[1], z+off[2], 0)
			} else if nb != Air && e.lavaKills(nb) {
				e.setBlock(nidx, Fire)
			}
		}
	}

	// Burnout.
	c.data++
	if c.data > 5 {
		r := e.rng.Intn(9) + 1 // 1..9, matching rand.Next(1, 10)
		switch {
		case r <= 2:
			e.setBlock(c.index, CoalOre)
		case r <= 4:
			e.setBlock(c.index, Obsidian)
		case r <= 8:
			e.setBlock(c.index, Air)
		default:
			c.data = 3 // keep burning
		}
	}
}

func (e *PhysicsEngine) expandToAir(x, y, z int) {
	idx := e.posToInt(x, y, z)
	if e.getBlock(idx) == Air {
		e.setBlock(idx, Fire)
	}
}

// expandDiagonal checks if the block at the diagonal position is flammable.
// If so, sets fire to the axis-aligned blocks between the fire and the diagonal.
// Matches MCGalaxy FirePhysics.ExpandDiagonal.
func (e *PhysicsEngine) expandDiagonal(x, y, z, dx, dy, dz int) {
	b := e.getBlock(e.posToInt(x+dx, y+dy, z+dz))
	if b == Air || !e.lavaKills(b) {
		return
	}
	if dx != 0 {
		e.setBlock(e.posToInt(x+dx, y, z), Fire)
	}
	if dy != 0 {
		e.setBlock(e.posToInt(x, y+dy, z), Fire)
	}
	if dz != 0 {
		e.setBlock(e.posToInt(x, y, z+dz), Fire)
	}
}

// ─── sponge ───

func (e *PhysicsEngine) doSponge(c *checkEntry, x, y, z int, isLava bool) {
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

func (e *PhysicsEngine) waterKills(b byte) bool {
	switch b {
	case Sapling:
		return true
	}
	return false
}

func (e *PhysicsEngine) lavaKills(b byte) bool {
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

func (e *PhysicsEngine) doTNT(c *checkEntry, x, y, z int, block byte, adv bool) {
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
func (e *PhysicsEngine) makeExplosion(x, y, z, size int) {
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
func (e *PhysicsEngine) explodeLayer(cx, cy, cz, size, prob int) {
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
