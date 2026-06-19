// Package physics defines the Physics interface that plugins use to
// control block physics simulation.
package physics

// PhysicsMode controls block physics simulation for a level.
//
//nolint:revive // intentional: re-exported as plugin.X
type PhysicsMode int

const (
	PhysicsOff      PhysicsMode = 0
	PhysicsBasic    PhysicsMode = 1
	PhysicsAdvanced PhysicsMode = 2
	PhysicsCustom   PhysicsMode = 3
)

// PhysicsBlock represents a block that physics is acting on.
//
//nolint:revive // intentional: re-exported as plugin.X
type PhysicsBlock struct {
	X, Y, Z int
	Block   byte
	Level   string
}

// PhysicsHandler is called for each physics tick on a block that
// needs processing. Return true to keep the block scheduled for
// the next tick, false to remove it.
//
//nolint:revive // intentional: re-exported as plugin.X
type PhysicsHandler func(block PhysicsBlock) bool

// Physics is the interface plugins use to control block physics.
type Physics interface {
	// Mode returns the current physics mode for the main level.
	Mode() PhysicsMode
	// SetMode changes the physics mode.
	SetMode(mode PhysicsMode)
	// Schedule adds a block at the given coordinates for physics
	// processing on the next tick.
	Schedule(x, y, z int)
	// RegisterHandler registers a custom physics handler that is
	// called for every scheduled block each tick.
	RegisterHandler(handler PhysicsHandler)
}
