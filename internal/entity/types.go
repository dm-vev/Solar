package entity

// Position stores the coarse world position for an entity.
type Position struct {
	X int
	Y int
	Z int
}

// Velocity stores the per-tick movement for an entity.
type Velocity struct {
	X int
	Y int
	Z int
}

// Entity is the minimal server-side actor state.
type Entity struct {
	Name  string
	Pos   Position
	Yaw   byte
	Pitch byte
	Vel   Velocity
}
