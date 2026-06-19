package classic

import "github.com/solar-mc/solar/internal/protocol/wire"

// Classic protocol opcodes — re-exported from the wire package for
// local readability.
const (
	opcodeHandshake            = wire.OpcodeHandshake
	opcodePing                 = wire.OpcodePing
	opcodeLevelInitialize      = wire.OpcodeLevelInitialize
	opcodeLevelData            = wire.OpcodeLevelData
	opcodeLevelFinalize        = wire.OpcodeLevelFinalize
	opcodeSetBlockClient       = wire.OpcodeSetBlockClient
	opcodeSetBlock             = wire.OpcodeSetBlock
	opcodeAddEntity            = wire.OpcodeAddEntity
	opcodeEntityTeleport       = wire.OpcodeEntityTeleport
	opcodeRelPosAndOrientation = wire.OpcodeRelPosAndOrient
	opcodeRelPos               = wire.OpcodeRelPos
	opcodeOrientation          = wire.OpcodeOrientation
	opcodeRemoveEntity         = wire.OpcodeRemoveEntity
	opcodeMessage              = wire.OpcodeMessage
	opcodeKick                 = wire.OpcodeKick
	opcodeTwoWayPing           = wire.OpcodeTwoWayPing
)

// selfID is the Classic protocol identifier for the player itself.
const selfID = wire.SelfID

// Coordinate scaling constants for the Classic protocol.
const (
	coordScale  = wire.CoordScale
	eyeHeight   = wire.EyeHeight
	maxChunkLen = wire.MaxChunkLen
)
