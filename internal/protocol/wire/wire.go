// Package wire defines the Classic/ClassiCube protocol constants and
// wire-format primitives shared by the server codec and load-test client.
package wire

import "strings"

// Classic protocol opcodes.
const (
	OpcodeHandshake            byte = 0
	OpcodePing                 byte = 1
	OpcodeLevelInitialize      byte = 2
	OpcodeLevelData            byte = 3
	OpcodeLevelFinalize        byte = 4
	OpcodeSetBlockClient       byte = 5
	OpcodeSetBlock             byte = 6
	OpcodeAddEntity            byte = 7
	OpcodeEntityTeleport       byte = 8
	OpcodeRelPosAndOrient      byte = 9
	OpcodeRelPos               byte = 10
	OpcodeOrientation          byte = 11
	OpcodeRemoveEntity         byte = 12
	OpcodeMessage              byte = 13
	OpcodeKick                 byte = 14
	OpcodeExtInfo              byte = 16
	OpcodeExtEntry             byte = 17
	OpcodeSetClickDistance     byte = 18
	OpcodeCustomBlockSupport   byte = 19
	OpcodeHoldThis             byte = 20
	OpcodeSetTextHotkey        byte = 21
	OpcodeExtAddPlayerName     byte = 22
	OpcodeExtAddEntity         byte = 23
	OpcodeExtRemovePlayer      byte = 24
	OpcodeEnvColors            byte = 25
	OpcodeMakeSelection        byte = 26
	OpcodeRemoveSelection      byte = 27
	OpcodeSetBlockPermission   byte = 28
	OpcodeChangeModel          byte = 29
	OpcodeEnvSetMapAppearance  byte = 30
	OpcodeEnvWeatherType       byte = 31
	OpcodeHackControl          byte = 32
	OpcodeExtAddEntity2        byte = 33
	OpcodePlayerClick          byte = 34
	OpcodeDefineBlock          byte = 35
	OpcodeUndefineBlock        byte = 36
	OpcodeDefineBlockExt       byte = 37
	OpcodeBulkBlockUpdate      byte = 38
	OpcodeSetTextColor         byte = 39
	OpcodeSetMapEnvURL         byte = 40
	OpcodeSetMapEnvProperty    byte = 41
	OpcodeSetEntityProperty    byte = 42
	OpcodeTwoWayPing           byte = 43
	OpcodeSetInventoryOrder    byte = 44
	OpcodeSetHotbar            byte = 45
	OpcodeSetSpawnpoint        byte = 46
	OpcodeVelocityControl      byte = 47
	OpcodeDefineEffect         byte = 48
	OpcodeSpawnEffect          byte = 49
	OpcodeDefineModel          byte = 50
	OpcodeDefineModelPart      byte = 51
	OpcodeUndefineModel        byte = 52
	OpcodePluginMessage        byte = 53
	OpcodeEntityTeleportExt    byte = 54
	OpcodeLightingMode         byte = 55
	OpcodeCinematicGui         byte = 56
	OpcodeNotifyAction         byte = 57
	OpcodeNotifyPositionAction byte = 58
	OpcodeToggleBlockList      byte = 59
)

// SelfID is the Classic protocol identifier for the player itself.
const SelfID byte = 255

// Coordinate scaling constants for the Classic protocol.
const (
	CoordScale  = 32
	EyeHeight   = 51
	MaxChunkLen = 1024
)

// WriteFixedString pads value with spaces to fill dst.
func WriteFixedString(dst []byte, value string) {
	for i := range dst {
		dst[i] = ' '
	}
	copy(dst, value)
}

// ReadFixedString trims trailing spaces and null bytes.
func ReadFixedString(src []byte) string {
	return strings.TrimRight(string(src), " \x00")
}
