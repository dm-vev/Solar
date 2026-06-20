package physics

import (
	"sync"
	"testing"
)

func makeEngine(w, h, l int) (*Engine, []byte) {
	blocks := make([]byte, w*h*l)
	var mu sync.Mutex
	broadcast := func(x, y, z int, block byte) {
		mu.Lock()
		_ = block
		mu.Unlock()
	}
	e := New(blocks, w, h, l, broadcast)
	e.SetMode(ModeAdvanced)
	return e, blocks
}

func TestSandFalls(t *testing.T) {
	e, blocks := makeEngine(3, 5, 3)
	// Place sand at y=4, air below.
	idx := e.posToInt(1, 4, 1)
	blocks[idx] = Sand
	e.Queue(1, 4, 1)
	e.Tick()
	// Sand should have moved down.
	if blocks[idx] != Air {
		t.Fatalf("sand at (1,4,1) = %d, want Air(0)", blocks[idx])
	}
	// Should land at bottom or one above.
	found := false
	for y := 0; y < 5; y++ {
		if blocks[e.posToInt(1, y, 1)] == Sand {
			found = true
		}
	}
	if !found {
		t.Fatal("sand not found after falling")
	}
}

func TestWaterSpreads(t *testing.T) {
	e, blocks := makeEngine(5, 3, 5)
	// Place water at center.
	blocks[e.posToInt(2, 1, 2)] = Water
	e.Queue(2, 1, 2)

	// Tick several times for random flow.
	for i := 0; i < 50; i++ {
		e.Tick()
	}

	// Water should have spread to at least one neighbour.
	spread := 0
	for y := 0; y < 3; y++ {
		for z := 0; z < 5; z++ {
			for x := 0; x < 5; x++ {
				if blocks[e.posToInt(x, y, z)] == Water {
					spread++
				}
			}
		}
	}
	if spread < 2 {
		t.Fatalf("water spread to %d cells, want >= 2", spread)
	}
}

func TestGrassGrows(t *testing.T) {
	e, blocks := makeEngine(3, 3, 3)
	// Dirt with air above.
	blocks[e.posToInt(1, 1, 1)] = Dirt
	e.Queue(1, 1, 1)
	// Tick 25 times (needs data > 20).
	for i := 0; i < 25; i++ {
		e.Tick()
	}
	if blocks[e.posToInt(1, 1, 1)] != Grass {
		t.Fatalf("dirt should have grown to grass, got %d", blocks[e.posToInt(1, 1, 1)])
	}
}

func TestGrassDies(t *testing.T) {
	e, blocks := makeEngine(3, 3, 3)
	// Grass with stone above (blocks light).
	blocks[e.posToInt(1, 1, 1)] = Grass
	blocks[e.posToInt(1, 2, 1)] = Stone
	e.Queue(1, 1, 1)
	for i := 0; i < 25; i++ {
		e.Tick()
	}
	if blocks[e.posToInt(1, 1, 1)] != Dirt {
		t.Fatalf("grass should have died to dirt, got %d", blocks[e.posToInt(1, 1, 1)])
	}
}

func TestLavaMakesStone(t *testing.T) {
	e, blocks := makeEngine(3, 3, 3)
	// Lava next to water.
	blocks[e.posToInt(0, 1, 1)] = Lava
	blocks[e.posToInt(1, 1, 1)] = Water
	blocks[e.posToInt(2, 1, 1)] = Air
	e.Queue(0, 1, 1)
	e.Queue(1, 1, 1)
	// Lava needs delay (4<<5 = 128 in upper bits). Tick enough.
	for i := 0; i < 200; i++ {
		e.Tick()
	}
	// Water+lava should produce stone somewhere.
	hasStone := false
	for _, b := range blocks {
		if b == Stone {
			hasStone = true
		}
	}
	if !hasStone {
		t.Fatal("no stone produced from lava+water")
	}
}

func TestPhysicsOff(t *testing.T) {
	e, blocks := makeEngine(3, 3, 3)
	e.SetMode(ModeOff)
	blocks[e.posToInt(1, 2, 1)] = Sand
	e.Queue(1, 2, 1)
	e.Tick()
	if blocks[e.posToInt(1, 2, 1)] != Sand {
		t.Fatal("sand moved when physics is off")
	}
}

func TestLeafDecay(t *testing.T) {
	e, blocks := makeEngine(9, 9, 9)
	// Leaves with no log nearby.
	blocks[e.posToInt(4, 4, 4)] = Leaves
	e.Queue(4, 4, 4)
	// Tick enough for decay (data needs 5 + random delay).
	for i := 0; i < 200; i++ {
		e.Tick()
	}
	if blocks[e.posToInt(4, 4, 4)] != Air {
		t.Fatalf("leaves should have decayed, got %d", blocks[e.posToInt(4, 4, 4)])
	}
}

func TestLeafNoDecayWithLog(t *testing.T) {
	e, blocks := makeEngine(9, 9, 9)
	// Leaves with log nearby.
	blocks[e.posToInt(4, 4, 4)] = Leaves
	blocks[e.posToInt(4, 5, 4)] = Log
	e.Queue(4, 4, 4)
	for i := 0; i < 200; i++ {
		e.Tick()
	}
	if blocks[e.posToInt(4, 4, 4)] != Leaves {
		t.Fatal("leaves decayed despite nearby log")
	}
}
