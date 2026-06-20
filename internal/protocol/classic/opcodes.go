// opcodes.go defines the Minecraft Classic protocol packet IDs.
//
// These are the single-byte opcodes that prefix every packet in the
// Classic wire protocol. The server reads the opcode first, then
// dispatches to the appropriate handler based on the value.
//
// Opcodes 0x00–0x0D are defined by the original Minecraft Classic
// protocol. CPE extension packets use opcodes 0x10 and above.

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
	opcodeExtInfo              = wire.OpcodeExtInfo
	opcodeExtEntry             = wire.OpcodeExtEntry
	opcodeExtAddPlayerName     = wire.OpcodeExtAddPlayerName
	opcodeExtRemovePlayerName  = wire.OpcodeExtRemovePlayer
	opcodeTwoWayPing           = wire.OpcodeTwoWayPing
	opcodeSetClickDistance     = wire.OpcodeSetClickDistance
	opcodeCustomBlockSupport   = wire.OpcodeCustomBlockSupport
	opcodeHoldThis             = wire.OpcodeHoldThis
	opcodeSetTextHotkey        = wire.OpcodeSetTextHotkey
	opcodeExtAddEntity         = wire.OpcodeExtAddEntity
	opcodeEnvColors            = wire.OpcodeEnvColors
	opcodeMakeSelection        = wire.OpcodeMakeSelection
	opcodeRemoveSelection      = wire.OpcodeRemoveSelection
	opcodeSetBlockPermission   = wire.OpcodeSetBlockPermission
	opcodeChangeModel          = wire.OpcodeChangeModel
	opcodeEnvSetMapAppearance  = wire.OpcodeEnvSetMapAppearance
	opcodeEnvWeatherType       = wire.OpcodeEnvWeatherType
	opcodeHackControl          = wire.OpcodeHackControl
	opcodeExtAddEntity2        = wire.OpcodeExtAddEntity2
	opcodePlayerClick          = wire.OpcodePlayerClick
	opcodeDefineBlock          = wire.OpcodeDefineBlock
	opcodeUndefineBlock        = wire.OpcodeUndefineBlock
	opcodeDefineBlockExt       = wire.OpcodeDefineBlockExt
	opcodeBulkBlockUpdate      = wire.OpcodeBulkBlockUpdate
	opcodeSetTextColor         = wire.OpcodeSetTextColor
	opcodeSetMapEnvURL         = wire.OpcodeSetMapEnvURL
	opcodeSetMapEnvProperty    = wire.OpcodeSetMapEnvProperty
	opcodeSetEntityProperty    = wire.OpcodeSetEntityProperty
	opcodeSetInventoryOrder    = wire.OpcodeSetInventoryOrder
	opcodeSetHotbar            = wire.OpcodeSetHotbar
	opcodeSetSpawnpoint        = wire.OpcodeSetSpawnpoint
	opcodeVelocityControl      = wire.OpcodeVelocityControl
	opcodeDefineEffect         = wire.OpcodeDefineEffect
	opcodeSpawnEffect          = wire.OpcodeSpawnEffect
	opcodeDefineModel          = wire.OpcodeDefineModel
	opcodeDefineModelPart      = wire.OpcodeDefineModelPart
	opcodeUndefineModel        = wire.OpcodeUndefineModel
	opcodePluginMessage        = wire.OpcodePluginMessage
	opcodeEntityTeleportExt    = wire.OpcodeEntityTeleportExt
	opcodeLightingMode         = wire.OpcodeLightingMode
	opcodeCinematicGui         = wire.OpcodeCinematicGui
	opcodeNotifyAction         = wire.OpcodeNotifyAction
	opcodeNotifyPositionAction = wire.OpcodeNotifyPositionAction
	opcodeToggleBlockList      = wire.OpcodeToggleBlockList
)

// selfID is the Classic protocol identifier for the player itself.
const selfID = wire.SelfID

// Coordinate scaling constants for the Classic protocol.
const (
	coordScale  = wire.CoordScale
	eyeHeight   = wire.EyeHeight
	maxChunkLen = wire.MaxChunkLen
)
