package classic

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	opcodeExtInfo             = 16
	opcodeExtEntry            = 17
	opcodeExtAddPlayerName    = 22
	opcodeExtRemovePlayerName = 24

	cpeExtPlayerListName    = "ExtPlayerList"
	cpeExtPlayerListVersion = 2
	cpeTwoWayPingName       = "TwoWayPing"
	cpeFastMapName          = "FastMap"
)

func (s *session) negotiateCPE() error {
	extensions := []struct {
		name    string
		version uint32
	}{
		{name: cpeExtPlayerListName, version: cpeExtPlayerListVersion},
		{name: cpeTwoWayPingName, version: 1},
		{name: cpeFastMapName, version: 1},
	}

	if err := s.writePacket(encodeExtInfo(s.serverName, len(extensions))); err != nil {
		return fmt.Errorf("write ext info: %w", err)
	}
	for _, extension := range extensions {
		if err := s.writePacket(encodeExtEntry(extension.name, extension.version)); err != nil {
			return fmt.Errorf("write ext entry %s: %w", extension.name, err)
		}
	}

	_, extCount, err := s.readExtInfo()
	if err != nil {
		return err
	}

	var supportsExtPlayerList bool
	var supportsTwoWayPing bool
	var supportsFastMap bool
	for i := 0; i < extCount; i++ {
		name, version, err := s.readExtEntry()
		if err != nil {
			return err
		}
		switch name {
		case cpeExtPlayerListName:
			if version >= cpeExtPlayerListVersion {
				supportsExtPlayerList = true
			}
		case cpeTwoWayPingName:
			if version >= 1 {
				supportsTwoWayPing = true
			}
		case cpeFastMapName:
			if version >= 1 {
				supportsFastMap = true
			}
		}
	}
	s.setSupports(supportsExtPlayerList, supportsTwoWayPing, supportsFastMap)

	return nil
}

func (s *session) handleTwoWayPing() error {
	payload := make([]byte, 3)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read two way ping payload: %w", err)
	}
	if !s.currentSupportsTwoWayPing() {
		return nil
	}
	if payload[0] != 0 {
		return nil
	}
	return s.writePacket(encodeTwoWayPing(false, binary.BigEndian.Uint16(payload[1:3])))
}

func (s *session) readExtInfo() (string, int, error) {
	packet := make([]byte, 67)
	if _, err := io.ReadFull(s.reader, packet); err != nil {
		return "", 0, fmt.Errorf("read ext info payload: %w", err)
	}
	if packet[0] != opcodeExtInfo {
		return "", 0, fmt.Errorf("read ext info payload: unexpected opcode %d", packet[0])
	}

	return readFixedString(packet[1:65]), int(binary.BigEndian.Uint16(packet[65:67])), nil
}

func (s *session) readExtEntry() (string, uint32, error) {
	packet := make([]byte, 69)
	if _, err := io.ReadFull(s.reader, packet); err != nil {
		return "", 0, fmt.Errorf("read ext entry payload: %w", err)
	}
	if packet[0] != opcodeExtEntry {
		return "", 0, fmt.Errorf("read ext entry payload: unexpected opcode %d", packet[0])
	}

	return readFixedString(packet[1:65]), binary.BigEndian.Uint32(packet[65:69]), nil
}
