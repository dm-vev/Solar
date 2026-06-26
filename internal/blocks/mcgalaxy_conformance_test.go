package blocks

import (
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

// TestMCGalaxy_TNTFuseMode verifies that TNT has a fuse delay before explosion.
// In MCGalaxy, physics level 3 triggers a 5-tick fuse before exploding.
// Solar doesn't have this — it explodes immediately in advanced mode.
func TestMCGalaxy_TNTFuseMode(t *testing.T) {
	// This test documents the missing fuse mode.
	// Solar's ModeAdvanced (2) maps to immediate explosion,
	// while MCGalaxy needs physics >= 4 for that.
	// MCGalaxy physics 3 = fuse (5 ticks delay with visual).
	// Skip this test until Solar adds a physics level 3 equivalent.
	t.Skip("TNT fuse mode (physics==3) not implemented in Solar — MCGalaxy has 5-tick fuse delay before explosion")
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

// TestMCGalaxy_TNTExplosionHasDropOutcome verifies that some blocks survive
// the explosion as drops (not immediately destroyed). In MCGalaxy, 20% of
// destroyed blocks become Drop+Dissipate (they briefly persist).
// Solar replaces this with Air (immediate destruction).
func TestMCGalaxy_TNTExplosionHasDropOutcome(t *testing.T) {
	// In MCGalaxy, the Drop+Dissipate outcome means the block is added as
	// a check with Drop/Dissipate physics args, not immediately set to Air.
	// Solar always sets to Air or TNTExplosion.
	// 
	// We can verify this by checking that the ratio of TNTExplosion to Air
	// is roughly 40/60 (MCGalaxy) vs 40/60 (Solar also 40/60 for ≤8).
	// Actually the difference is subtle — Drop+Dissipate eventually becomes
	// Air anyway. The observable difference is timing, not final state.
	//
	// This is hard to test without mocking the physics args.
	// Marking as a known discrepancy.
	t.Skip("TNT Drop+Dissipate outcome not distinguishable from Air in final state — timing difference only")
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
