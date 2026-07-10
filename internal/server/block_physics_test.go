package server

import (
	"testing"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/world"
)

func TestBlockPhysicsUsesSeparateEnginesPerLevel(t *testing.T) {
	t.Parallel()

	mainManager := managerWithLevel("main", 4, 4, 4)
	otherManager := managerWithLevel("other", 8, 8, 8)
	srv := &Server{
		worlds:       mainManager,
		blockPhysics: make(map[*world.Manager]*blocks.PhysicsEngine),
	}

	mainEngine := srv.RegisterBlockPhysics(mainManager)
	otherEngine := srv.RegisterBlockPhysics(otherManager)
	if mainEngine == nil || otherEngine == nil {
		t.Fatal("expected physics engines for both levels")
	}
	if mainEngine == otherEngine {
		t.Fatal("levels unexpectedly share a physics engine")
	}
	if srv.BlockPhysics() != mainEngine {
		t.Fatal("BlockPhysics did not return the main level engine")
	}
	if srv.BlockPhysicsFor(otherManager) != otherEngine {
		t.Fatal("BlockPhysicsFor did not return the requested level engine")
	}
}

func TestRegisterBlockPhysicsRefreshesDimensionsAndPreservesMode(t *testing.T) {
	t.Parallel()

	manager := managerWithLevel("main", 4, 4, 4)
	srv := &Server{
		worlds:       manager,
		blockPhysics: make(map[*world.Manager]*blocks.PhysicsEngine),
	}

	original := srv.RegisterBlockPhysics(manager)
	original.SetMode(blocks.ModeAdvanced)

	manager.SetCurrent(world.Level{
		Name:   "main",
		Width:  8,
		Height: 4,
		Length: 8,
		Blocks: make([]byte, 8*4*8),
	})
	refreshed := srv.RegisterBlockPhysics(manager)
	if refreshed == original {
		t.Fatal("expected a replacement engine after level resize")
	}
	if refreshed.Mode() != blocks.ModeAdvanced {
		t.Fatalf("refreshed mode = %d, want %d", refreshed.Mode(), blocks.ModeAdvanced)
	}

	refreshed.Queue(7, 3, 7)
	refreshed.Tick()
}

func TestUnregisterBlockPhysicsRemovesLevelEngine(t *testing.T) {
	t.Parallel()

	manager := managerWithLevel("main", 4, 4, 4)
	srv := &Server{
		worlds:       manager,
		blockPhysics: make(map[*world.Manager]*blocks.PhysicsEngine),
	}

	srv.RegisterBlockPhysics(manager)
	srv.UnregisterBlockPhysics(manager)
	if srv.BlockPhysicsFor(manager) != nil {
		t.Fatal("physics engine remained registered after unload")
	}
}

func TestBlockPhysicsModeDrivesTNTFuse(t *testing.T) {
	t.Parallel()
	manager := managerWithLevel("main", 7, 7, 7)
	srv := &Server{worlds: manager, blockPhysics: make(map[*world.Manager]*blocks.PhysicsEngine)}
	srv.RegisterBlockPhysics(manager)
	srv.SetBlockPhysicsMode(manager, blocks.ModeHardcore)
	if got := srv.BlockPhysicsMode(manager); got != blocks.ModeHardcore {
		t.Fatalf("mode = %d, want hardcore", got)
	}
	manager.SetBlock(3, 3, 3, blocks.TNTSmall)
	srv.QueueBlockPhysics(manager, 3, 3, 3)
	for range 5 {
		srv.tickBlockPhysics()
	}
	if block, _ := manager.BlockAt(3, 3, 3); block != blocks.TNTSmall {
		t.Fatalf("TNT exploded before fuse completed: %d", block)
	}
	srv.tickBlockPhysics()
	if block, _ := manager.BlockAt(3, 3, 3); block == blocks.TNTSmall {
		t.Fatal("TNT did not explode after fuse")
	}
}

func managerWithLevel(name string, width, height, length int) *world.Manager {
	manager := world.NewManager()
	manager.SetCurrent(world.Level{
		Name:   name,
		Width:  width,
		Height: height,
		Length: length,
		Blocks: make([]byte, width*height*length),
	})
	return manager
}
