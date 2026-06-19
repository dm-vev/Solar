package entity

// Position stores the entity position in wire/fixed-point units (1/32 block).
// This preserves sub-block precision from the Classic protocol.
type Position struct {
	X int
	Y int
	Z int
}

// Velocity stores the per-tick movement in wire units (1/32 block).
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
