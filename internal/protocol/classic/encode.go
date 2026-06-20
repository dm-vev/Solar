package classic

import (
	"encoding/binary"

	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/protocol/wire"
)

func encodeKick(message string) []byte {
	packet := make([]byte, 65)
	packet[0] = opcodeKick
	writeFixedString(packet[1:65], message)
	return packet
}

func encodeAddEntity(id byte, name string, pos entity.Position, yaw, pitch byte) []byte {
	packet := make([]byte, 74)
	packet[0] = opcodeAddEntity
	packet[1] = id
	writeFixedString(packet[2:66], name)
	binary.BigEndian.PutUint16(packet[66:68], uint16(pos.X))
	binary.BigEndian.PutUint16(packet[68:70], uint16(pos.Y+eyeHeight))
	binary.BigEndian.PutUint16(packet[70:72], uint16(pos.Z))
	packet[72] = yaw
	packet[73] = pitch
	return packet
}

func encodeRemoveEntity(id byte) []byte {
	return []byte{opcodeRemoveEntity, id}
}

func encodeEntityTeleport(id byte, pos entity.Position, yaw, pitch byte) []byte {
	packet := make([]byte, 10)
	packet[0] = opcodeEntityTeleport
	packet[1] = id
	binary.BigEndian.PutUint16(packet[2:4], uint16(pos.X))
	binary.BigEndian.PutUint16(packet[4:6], uint16(pos.Y+eyeHeight))
	binary.BigEndian.PutUint16(packet[6:8], uint16(pos.Z))
	packet[8] = yaw
	packet[9] = pitch
	return packet
}

func encodeRelPosAndOrient(id byte, dx, dy, dz int, yaw, pitch byte) []byte {
	packet := make([]byte, 7)
	packet[0] = opcodeRelPosAndOrientation
	packet[1] = id
	packet[2] = byte(int8(dx))
	packet[3] = byte(int8(dy))
	packet[4] = byte(int8(dz))
	packet[5] = yaw
	packet[6] = pitch
	return packet
}

func encodeRelPos(id byte, dx, dy, dz int) []byte {
	packet := make([]byte, 5)
	packet[0] = opcodeRelPos
	packet[1] = id
	packet[2] = byte(int8(dx))
	packet[3] = byte(int8(dy))
	packet[4] = byte(int8(dz))
	return packet
}

func encodeOrientation(id byte, yaw, pitch byte) []byte {
	return []byte{opcodeOrientation, id, yaw, pitch}
}

// fitsRelDelta reports whether a wire-unit delta fits in a signed byte.
func fitsRelDelta(d int) bool {
	return d >= -128 && d <= 127
}

func encodeMessage(messageType byte, message string) []byte {
	packet := make([]byte, 66)
	packet[0] = opcodeMessage
	packet[1] = messageType
	writeFixedString(packet[2:66], message)
	return packet
}

func encodeSetBlock(x, y, z int, blockID byte) []byte {
	packet := make([]byte, 8)
	packet[0] = opcodeSetBlock
	binary.BigEndian.PutUint16(packet[1:3], uint16(x))
	binary.BigEndian.PutUint16(packet[3:5], uint16(y))
	binary.BigEndian.PutUint16(packet[5:7], uint16(z))
	packet[7] = blockID
	return packet
}

func writeFixedString(dst []byte, value string) {
	wire.WriteFixedString(dst, value)
}

func readFixedString(src []byte) string {
	return wire.ReadFixedString(src)
}
