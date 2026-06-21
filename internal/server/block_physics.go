package server

import (
	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/world"
)

// RegisterBlockPhysics creates or replaces the physics engine for a level.
// Re-registering preserves the previous physics mode and refreshes dimensions.
func (s *Server) RegisterBlockPhysics(manager *world.Manager) *blocks.PhysicsEngine {
	if manager == nil {
		return nil
	}

	level := manager.Current()
	width, height, length := level.Width, level.Height, level.Length
	if width < 1 || height < 1 || length < 1 {
		return nil
	}

	engine := blocks.NewPhysics(
		width,
		height,
		length,
		func(index int) byte {
			x, y, z := blockPosition(index, width, length)
			block, _ := manager.BlockAt(x, y, z)
			return block
		},
		func(index int, block byte) {
			x, y, z := blockPosition(index, width, length)
			manager.SetBlock(x, y, z, block)
		},
		func(x, y, z int, block byte) {
			if s.codec != nil {
				s.codec.BroadcastSetBlockToLevel(manager, x, y, z, block)
			}
		},
	)

	s.blockPhysicsMu.Lock()
	if previous := s.blockPhysics[manager]; previous != nil {
		engine.SetMode(previous.Mode())
	}
	if s.blockPhysics == nil {
		s.blockPhysics = make(map[*world.Manager]*blocks.PhysicsEngine)
	}
	s.blockPhysics[manager] = engine
	s.blockPhysicsMu.Unlock()

	return engine
}

// UnregisterBlockPhysics removes a level's physics engine.
func (s *Server) UnregisterBlockPhysics(manager *world.Manager) {
	if manager == nil {
		return
	}

	s.blockPhysicsMu.Lock()
	delete(s.blockPhysics, manager)
	s.blockPhysicsMu.Unlock()
}

// QueueBlockPhysics schedules a block update on its owning level.
func (s *Server) QueueBlockPhysics(manager *world.Manager, x, y, z int) {
	engine := s.BlockPhysicsFor(manager)
	if engine != nil {
		engine.Queue(x, y, z)
	}
}

// BlockPhysics returns the main level's physics engine.
func (s *Server) BlockPhysics() *blocks.PhysicsEngine {
	return s.BlockPhysicsFor(s.worlds)
}

// BlockPhysicsFor returns the physics engine associated with a level.
func (s *Server) BlockPhysicsFor(manager *world.Manager) *blocks.PhysicsEngine {
	if manager == nil {
		return nil
	}

	s.blockPhysicsMu.RLock()
	engine := s.blockPhysics[manager]
	s.blockPhysicsMu.RUnlock()
	return engine
}

func (s *Server) tickBlockPhysics() {
	s.blockPhysicsMu.RLock()
	engines := make([]*blocks.PhysicsEngine, 0, len(s.blockPhysics))
	for _, engine := range s.blockPhysics {
		engines = append(engines, engine)
	}
	s.blockPhysicsMu.RUnlock()

	for _, engine := range engines {
		engine.Tick()
	}
}

func blockPosition(index, width, length int) (x, y, z int) {
	y = index / (width * length)
	remainder := index - y*width*length
	z = remainder / width
	x = remainder - z*width
	return x, y, z
}
