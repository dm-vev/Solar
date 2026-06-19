package classic

import "encoding/binary"

func encodeExtInfo(appName string, count int) []byte {
	packet := make([]byte, 67)
	packet[0] = opcodeExtInfo
	writeFixedString(packet[1:65], appName)
	binary.BigEndian.PutUint16(packet[65:67], uint16(count))
	return packet
}

func encodeExtEntry(name string, version uint32) []byte {
	packet := make([]byte, 69)
	packet[0] = opcodeExtEntry
	writeFixedString(packet[1:65], name)
	binary.BigEndian.PutUint32(packet[65:69], version)
	return packet
}

func encodeExtAddPlayerName(id byte, name string) []byte {
	packet := make([]byte, 196)
	packet[0] = opcodeExtAddPlayerName
	packet[1] = id
	writeFixedString(packet[2:66], name)
	writeFixedString(packet[66:130], name)
	writeFixedString(packet[130:194], "Players")
	packet[194] = 0
	return packet
}

func encodeExtRemovePlayerName(id byte) []byte {
	packet := make([]byte, 3)
	packet[0] = opcodeExtRemovePlayerName
	packet[1] = id
	return packet
}

func encodeTwoWayPing(serverToClient bool, id uint16) []byte {
	packet := make([]byte, 4)
	packet[0] = opcodeTwoWayPing
	if serverToClient {
		packet[1] = 1
	}
	binary.BigEndian.PutUint16(packet[2:4], id)
	return packet
}
