package blocks

import (
	"math/rand"
	"testing"
)

// This file contains tests that verify Solar's physics engine matches
// the ACTUAL MCGalaxy behavior, based on the MCGalaxy source code at
// https://github.com/ClassiCube/MCGalaxy/tree/master/MCGalaxy/Blocks/Physics
//
// Tests named TestMCGalaxy_* are strict conformance tests. If they fail,
// Solar's behavior diverges from MCGalaxy and must be fixed.

// ─── 1. Water spread speed ───
// MCGalaxy: water only spreads in a direction when rand.Next(4)==0 (25% per tick).
// Even if the target is air (not blocked), the random check is applied FIRST.
// Solar combines: "if random OR !blocked → spread". This means Solar spreads
// to unblocked directions immediately, while MCGalaxy takes several ticks.
//
// MCGalaxy SimpleLiquidPhysics.DoWaterRandowFlow:
//   if ((data & flowed_xMax) == 0 && rand.Next(4) == 0) {
//       PhysWater(...); data |= flowed_xMax;
//   }
//   // ... then later:
//   if ((data & flowed_xMax) == 0 && WaterBlocked(...)) {
//       data |= flowed_xMax;  // mark blocked, but DON'T spread
//   }
//
// Solar doLiquid:
//   if e.rng.Intn(4) == 0 || !e.liquidBlocked(...) → spread
//
// DISCREPANCY: Solar water spreads much faster than MCGalaxy.

// TestMCGalaxy_WaterDoesNotSpreadImmediatelyToAir verifies that water does NOT
// always spread to air on the first tick. MCGalaxy uses 25% random chance.
func TestMCGalaxy_WaterDoesNotSpreadImmediatelyToAir(t *testing.T) {
	spreadCount := 0
	for attempt := 0; attempt < 100; attempt++ {
		e, blocks := makeEngine(5, 1, 5)
		blocks[e.posToInt(2, 0, 2)] = Water
		e.Queue(2, 0, 2)
		e.Tick()
		waterCells := 0
		for _, b := range blocks {
			if b == Water {
				waterCells++
			}
		}
		if waterCells > 1 {
			spreadCount++
		}
	}
	// MCGalaxy: 25% per direction per tick. With 4 horizontal directions,
	// the probability of spreading to at least one is ~68% per tick.
	// But it should NOT spread 100% of the time.
	if spreadCount == 100 {
		t.Fatal("water spread on every single tick — MCGalaxy uses 25% random chance per direction, should not always spread immediately")
	}
}

// ─── 2. Lava re-delay after flowing ───
// MCGalaxy DoLavaRandowFlow: after flowing, lava's upper 3 bits are reset:
//   data &= flowed_maskAll;  // clear upper bits
//   data |= (byte)(rand.Next(3) << 5);  // random 0-2
// This means lava gets a 2-4 tick re-delay after each flow step.
// Solar: preserves bit 7 (0x80), no re-delay. Lava flows every tick after initial delay.
//
// DISCREPANCY: Solar lava flows much faster than MCGalaxy after initial delay.

// TestMCGalaxy_LavaReDelaysAfterFlow verifies that lava does not flow every
// single tick after the initial delay. MCGalaxy re-randomizes the delay.
func TestMCGalaxy_LavaReDelaysAfterFlow(t *testing.T) {
	e, blocks := makeEngine(7, 1, 7)
	blocks[e.posToInt(3, 0, 3)] = Lava
	e.Queue(3, 0, 3)

	// Tick past initial delay (4 ticks).
	for i := 0; i < 5; i++ {
		e.Tick()
	}

	// Now count lava cells after each tick. MCGalaxy lava should NOT
	// spread every single tick — it re-delays randomly.
	spreadEveryTick := true
	prevCount := countBlock(blocks, Lava)
	for i := 0; i < 20; i++ {
		e.Tick()
		curCount := countBlock(blocks, Lava)
		if curCount == prevCount {
			spreadEveryTick = false
		}
		prevCount = curCount
	}
	if spreadEveryTick {
		t.Fatal("lava spread on every tick after initial delay — MCGalaxy re-delays lava randomly (2-4 ticks between flows)")
	}
}

// ─── 3. Sand falling in advanced mode ───
// MCGalaxy DoFalling: in advanced mode (physics > 1), the scan loop breaks
// after the FIRST block below:
//   do {
//       index = IntOffset(index, 0, -1, 0); yCur--;
//       cur = GetBlock(x, yCur, z);
//       if (cur == Invalid) break;
//       switch (cur) { case Air: case Water: case Lava: movedDown=true; break; ... }
//       if (hitBlock || physics > 1) break;  // ← ALWAYS breaks in advanced
//   } while (true);
//
// Then: AddUpdate(index, C.Block) — sand moves down exactly ONE block per tick.
//
// Solar: scans ALL the way down and places sand at the lowest position.
//
// DISCREPANCY: Solar sand teleports to the bottom; MCGalaxy sand falls one block per tick.

// TestMCGalaxy_SandFallsOneBlockPerTickAdvanced verifies that in advanced mode,
// sand falls exactly one block per tick, not all the way to the bottom.
func TestMCGalaxy_SandFallsOneBlockPerTickAdvanced(t *testing.T) {
	e, blocks := makeEngine(1, 10, 1)
	blocks[e.posToInt(0, 9, 0)] = Sand
	e.Queue(0, 9, 0)
	e.Tick()

	// MCGalaxy: sand should be at y=8 (one block down), NOT at y=0 (bottom).
	if blocks[e.posToInt(0, 9, 0)] != Air {
		t.Fatalf("sand should have left y=9: got %d", blocks[e.posToInt(0, 9, 0)])
	}
	if blocks[e.posToInt(0, 8, 0)] != Sand {
		// Check if it went all the way to the bottom (Solar behavior)
		if blocks[e.posToInt(0, 0, 0)] == Sand {
			t.Fatal("sand fell all the way to bottom in one tick — MCGalaxy advanced mode: sand falls one block per tick")
		}
		t.Fatalf("sand not at y=8 (one block down): got %d at y=8", blocks[e.posToInt(0, 8, 0)])
	}
}

// ─── 4. Leaf decay algorithm ───
// MCGalaxy LeafPhysics.DoLeafDecay: uses a flood-fill distance algorithm.
// A leaf survives only if there's a path of leaves connecting it to a log
// within 4 steps. A leaf with a log nearby but no leaf path still decays.
//
// Solar hasNearbyLog: simply checks if ANY log exists within 4-block radius.
// A leaf survives if any log is in the 9×9×9 cube, regardless of leaf paths.
//
// Test: leaf with log 4 blocks away but separated by air should decay in
// MCGalaxy but survive in Solar.
//
// DISCREPANCY: Solar leaves survive when MCGalaxy leaves would decay.

// TestMCGalaxy_LeafDecaysWithoutLeafPath verifies that a leaf decays when
// the nearest log is separated by air (no leaf path), even if within 4 blocks.
func TestMCGalaxy_LeafDecaysWithoutLeafPath(t *testing.T) {
	e, blocks := makeEngine(11, 3, 3)
	// Leaf at center, log 3 blocks away but with air in between (no leaf path).
	blocks[e.posToInt(5, 1, 1)] = Leaves
	blocks[e.posToInt(8, 1, 1)] = Log
	e.Queue(5, 1, 1)
	for i := 0; i < 300; i++ {
		e.Tick()
	}
	// MCGalaxy: leaf should decay (no leaf path to log).
	// Solar: leaf survives (log is within 4-block radius).
	if blocks[e.posToInt(5, 1, 1)] == Leaves {
		t.Fatal("leaf survived despite no leaf path to log — MCGalaxy uses flood-fill distance through leaves, not simple radius check")
	}
}

// ─── 5. Fire burnout outcomes ───
// MCGalaxy FirePhysics.Do burnout:
//   rand.Next(1, 10):
//   ≤2 → CoalOre + Drop/Dissipate
//   ≤4 → Obsidian + Drop/Dissipate
//   ≤8 → Air
//   >8 → keep burning (data = 3)
//
// Solar doFire burnout:
//   rand.Intn(10):
//   <2 → Air
//   <4 → keep burning (data = 3)
//   ≥4 → Air
//
// DISCREPANCY: Solar produces only Air; MCGalaxy produces CoalOre and Obsidian.

// TestMCGalaxy_FireProducesCoalOrObsidian verifies that fire burnout can
// produce CoalOre or Obsidian blocks, not just Air.
func TestMCGalaxy_FireProducesCoalOrObsidian(t *testing.T) {
	const CoalOre byte = 16
	const Obsidian byte = 49

	// Run many iterations to hit all burnout outcomes.
	for attempt := 0; attempt < 100; attempt++ {
		e, blocks := makeEngine(5, 5, 5)
		blocks[e.posToInt(2, 2, 2)] = Fire
		e.Queue(2, 2, 2)
		for i := 0; i < 200; i++ {
			e.Tick()
		}
		for _, b := range blocks {
			if b == CoalOre || b == Obsidian {
				return // Found a CoalOre or Obsidian — pass.
			}
		}
	}
	t.Fatal("fire never produced CoalOre or Obsidian — MCGalaxy burnout has 20% CoalOre, 20% Obsidian, 40% Air, 20% keep-burning")
}

// ─── 6. Fire spread probability ───
// MCGalaxy: rand.Next(1, 20) == 1 → 1/19 chance
// Solar: rng.Intn(20) == 0 → 1/20 chance
// Minor discrepancy, but technically different.

// TestMCGalaxy_FireSpreadChance verifies the fire spread chance is 1/19 per tick.
// We can't test the exact probability, but we can verify it's not 1/20 by
// checking the spread rate over many iterations.
// (This is a soft test — exact probability testing would require mocking rng.)

// ─── 7. Fire advanced spread — diagonal expansion ───
// MCGalaxy FirePhysics: in advanced mode, calls ExpandDiagonal for 12 diagonal
// directions (±x, ±y, ±z combinations) in addition to direct neighbor expansion.
// Solar: only expands to 6 direct neighbors.
//
// DISCREPANCY: Solar fire doesn't spread diagonally in advanced mode.

// TestMCGalaxy_FireSpreadsDiagonally verifies that fire can spread to
// diagonal neighbors in advanced mode.
func TestMCGalaxy_FireSpreadsDiagonally(t *testing.T) {
	for attempt := 0; attempt < 200; attempt++ {
		e, blocks := makeEngine(5, 5, 5)
		// Fire at center, flammable block at diagonal position.
		blocks[e.posToInt(2, 2, 2)] = Fire
		blocks[e.posToInt(3, 3, 3)] = Wood // diagonal: +x, +y, +z
		// Block all direct neighbors with stone to force diagonal spread.
		blocks[e.posToInt(3, 2, 2)] = Stone
		blocks[e.posToInt(1, 2, 2)] = Stone
		blocks[e.posToInt(2, 3, 2)] = Stone
		blocks[e.posToInt(2, 1, 2)] = Stone
		blocks[e.posToInt(2, 2, 3)] = Stone
		blocks[e.posToInt(2, 2, 1)] = Stone
		e.Queue(2, 2, 2)
		for i := 0; i < 200; i++ {
			e.Tick()
		}
		if blocks[e.posToInt(3, 3, 3)] != Wood {
			return // Fire spread diagonally and burned the wood.
		}
	}
	t.Fatal("fire never spread diagonally — MCGalaxy ExpandDiagonal spreads to 12 diagonal positions in advanced mode")
}

// ─── 8. TNT fuse mode (physics == 3) ───
// MCGalaxy TntPhysics.DoSmallTnt:
//   physics < 3 → just remove (Blockchange to Air)
//   physics == 3 → fuse delay (5 ticks, ToggleFuse)
//   physics > 3 (i.e. >= 4) → immediate explosion
//
// Solar doTNT:
//   !adv (mode < 2) → just remove
//   adv (mode >= 2) → immediate explosion
//
// DISCREPANCY: Solar maps "advanced" (mode 2) to immediate explosion.
// MCGalaxy requires physics >= 4 for immediate explosion; physics 2-3
// have different behavior (physics 2 = still just remove, physics 3 = fuse).
// Solar has no fuse mode at all.

func TestMCGalaxy_TNTFuseMode(t *testing.T) {
	e, cells := makeEngine(9, 9, 9)
	e.SetMode(ModeHardcore)
	center := e.posToInt(4, 4, 4)
	above := e.posToInt(4, 5, 4)
	cells[center] = TNTSmall
	e.Queue(4, 4, 4)
	for tick := 1; tick <= 5; tick++ {
		e.Tick()
		if cells[center] != TNTSmall {
			t.Fatalf("tick %d: TNT exploded before fuse completed", tick)
		}
		want := StillLava
		if tick%2 == 0 {
			want = Air
		}
		if cells[above] != want {
			t.Fatalf("tick %d: fuse block = %d, want %d", tick, cells[above], want)
		}
	}
	e.Tick()
	if cells[center] == TNTSmall {
		t.Fatal("TNT did not explode on the sixth processing")
	}

	for _, test := range []struct {
		name string
		mode int
		want byte
	}{
		{"advanced removes", ModeAdvanced, Air},
		{"instant explodes", ModeInstant, TNTExplosion},
	} {
		t.Run(test.name, func(t *testing.T) {
			e, cells := makeEngine(7, 7, 7)
			e.SetMode(test.mode)
			index := e.posToInt(3, 3, 3)
			cells[index] = TNTSmall
			e.Queue(3, 3, 3)
			e.Tick()
			if cells[index] != test.want {
				t.Fatalf("center = %d, want %d", cells[index], test.want)
			}
		})
	}
}

// ─── 9. TNT explosion outcomes ───
// MCGalaxy Explode:
//   rand.Next(1, 10+1):
//   ≤4 → TNT_Explosion (visual)
//   ≤8 → Air
//   >8 → Drop + Dissipate (block NOT destroyed, drops and dissipates)
//
// Solar explodeLayer:
//   rand.Intn(10)+1:
//   ≤4 → TNTExplosion
//   ≤8 → Air
//   >8 → Air (simplified — was Drop+Dissipate in MCGalaxy)
//
// DISCREPANCY: MCGalaxy has 20% Drop+Dissipate (block survives temporarily),
// Solar replaces with Air (always destroyed).

func TestMCGalaxy_TNTExplosionHasDropOutcome(t *testing.T) {
	var debrisSeed int64 = -1
	for seed := int64(0); seed < 1000; seed++ {
		e, cells := makeEngine(1, 1, 1)
		cells[0] = Stone
		e.rng = rand.New(rand.NewSource(seed))
		e.explodeLayer(0, 0, 0, 0, -1)
		if len(e.updates) == 0 && len(e.checks) == 1 && e.checks[0].debris {
			debrisSeed = seed
			break
		}
	}
	if debrisSeed < 0 {
		t.Fatal("no deterministic debris outcome found")
	}

	moveSeed := seedForTNTDebris(t, func(r *rand.Rand) bool {
		return r.Intn(99) >= 8 && r.Intn(99) < 50 && r.Intn(99) < 49
	})
	e, cells := makeEngine(1, 3, 1)
	cells[e.posToInt(0, 2, 0)] = Stone
	e.rng = rand.New(rand.NewSource(moveSeed))
	e.queueDebris(e.posToInt(0, 2, 0), true)
	e.Tick()
	if cells[e.posToInt(0, 2, 0)] != Air || cells[e.posToInt(0, 1, 0)] != Stone {
		t.Fatalf("debris did not fall one block: %v", cells)
	}

	dissipateSeed := seedForTNTDebris(t, func(r *rand.Rand) bool { return r.Intn(99) < 8 })
	e.rng = rand.New(rand.NewSource(dissipateSeed))
	e.Tick()
	if cells[e.posToInt(0, 1, 0)] != Air {
		t.Fatal("debris did not dissipate")
	}
}

func seedForTNTDebris(t *testing.T, matches func(*rand.Rand) bool) int64 {
	t.Helper()
	for seed := int64(0); seed < 10000; seed++ {
		if matches(rand.New(rand.NewSource(seed))) {
			return seed
		}
	}
	t.Fatal("no deterministic TNT seed found")
	return 0
}

func TestMCGalaxy_TNTCheckDedupPreservesFuseState(t *testing.T) {
	e, cells := makeEngine(9, 9, 9)
	e.SetMode(ModeHardcore)
	first := e.posToInt(3, 4, 4)
	second := e.posToInt(4, 4, 4)
	cells[first], cells[second] = TNTSmall, TNTSmall
	e.checks = []checkEntry{{index: first, data: 5}, {index: second, data: 3}}
	e.queued[first], e.queued[second] = 0, 1
	e.Tick()
	for _, check := range e.checks {
		if check.index == second && check.data == 4 {
			return
		}
	}
	t.Fatalf("second fuse state was reset: %+v", e.checks)
}

func TestMCGalaxy_TNTDebrisDoesNotMoveIntoStagedUpdate(t *testing.T) {
	e, cells := makeEngine(1, 3, 1)
	source := e.posToInt(0, 2, 0)
	below := e.posToInt(0, 1, 0)
	cells[source] = Stone
	e.setBlock(below, Glass)
	e.rng = rand.New(rand.NewSource(seedForTNTDebris(t, func(r *rand.Rand) bool {
		return r.Intn(99) >= 8 && r.Intn(99) < 50 && r.Intn(99) < 49
	})))
	e.queueDebris(source, true)
	e.Tick()
	if cells[source] != Stone || cells[below] != Glass {
		t.Fatalf("debris was lost on staged destination: %v", cells)
	}
}

// ─── 10. Water flow: MCGalaxy separates random and blocked checks ───
// MCGalaxy DoWaterRandowFlow:
//   Pass 1: if (rand.Next(4) == 0) → spread + mark flowed
//   Pass 2: if (still not flowed && WaterBlocked) → mark flowed (but NO spread)
//
// Solar doLiquid:
//   if (rand.Next(4) == 0 || !liquidBlocked) → spread + mark flowed
//
// In MCGalaxy, if random fails and target is NOT blocked, the direction
// stays unflowed and is retried next tick. In Solar, if the target is not
// blocked, it spreads immediately regardless of random.
//
// This means Solar water/lava spreads to all unblocked directions much
// faster than MCGalaxy. Already covered by TestMCGalaxy_WaterDoesNotSpreadImmediatelyToAir.

// ─── Summary of discrepancies ───
// The following MCGalaxy behaviors are NOT correctly implemented in Solar:
//
// 1. Water/lava spread: Solar spreads immediately to unblocked directions;
//    MCGalaxy uses 25% random per direction per tick. (MAJOR)
// 2. Lava re-delay: MCGalaxy re-randomizes lava delay (2-4 ticks) after
//    each flow step; Solar has no re-delay. (MAJOR)
// 3. Sand falling (advanced): MCGalaxy falls one block per tick; Solar
//    teleports to the bottom. (MAJOR)
// 4. Leaf decay: MCGalaxy uses flood-fill distance through leaves; Solar
//    uses simple radius check. (MODERATE)
// 5. Fire burnout: MCGalaxy produces CoalOre/Obsidian; Solar only Air. (MODERATE)
// 6. Fire spread chance: MCGalaxy 1/19; Solar 1/20. (MINOR)
// 7. Fire diagonal spread: MCGalaxy expands diagonally; Solar doesn't. (MODERATE)
// 8. TNT fuse mode: MCGalaxy has 5-tick fuse at physics 3; Solar has none. (MODERATE)
// 9. TNT Drop+Dissipate: MCGalaxy has 20% drop outcome; Solar replaced with Air. (MINOR)
// 10. Water/lava flow: combined random+blocked check vs separate passes. (covered by #1)
