// Package wire defines the Classic/ClassiCube protocol constants and
// wire-format primitives shared by the server codec and load-test client.
package wire

import "strings"

// Classic protocol opcodes.
const (
	OpcodeHandshake        byte = 0
	OpcodePing             byte = 1
	OpcodeLevelInitialize  byte = 2
	OpcodeLevelData        byte = 3
	OpcodeLevelFinalize    byte = 4
	OpcodeSetBlockClient   byte = 5
	OpcodeSetBlock         byte = 6
	OpcodeAddEntity        byte = 7
	OpcodeEntityTeleport   byte = 8
	OpcodeRelPosAndOrient  byte = 9
	OpcodeRelPos           byte = 10
	OpcodeOrientation      byte = 11
	OpcodeRemoveEntity     byte = 12
	OpcodeMessage          byte = 13
	OpcodeKick             byte = 14
	OpcodeExtInfo          byte = 16
	OpcodeExtEntry         byte = 17
	OpcodeExtAddPlayerName byte = 22
	OpcodeExtRemovePlayer  byte = 24
	OpcodeTwoWayPing       byte = 43
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
