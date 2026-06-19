package classic

import (
	"encoding/binary"
	"fmt"
	"io"
)

func (s *session) handlePlayerClick() error {
	payload := make([]byte, 14)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read player click payload: %w", err)
	}
	if !s.supportsExt(cpeExtPlayerClick) {
		return nil
	}
	_ = payload[0]                              // button
	_ = payload[1]                              // action
	_ = binary.BigEndian.Uint16(payload[2:4])   // yaw
	_ = binary.BigEndian.Uint16(payload[4:6])   // pitch
	_ = payload[6]                              // entityID
	_ = binary.BigEndian.Uint16(payload[7:9])   // x
	_ = binary.BigEndian.Uint16(payload[9:11])  // y
	_ = binary.BigEndian.Uint16(payload[11:13]) // z
	_ = payload[13]                             // face
	return nil
}

func (s *session) handlePluginMessage() error {
	payload := make([]byte, 65)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read plugin message payload: %w", err)
	}
	if !s.supportsExt(cpeExtPluginMessages) {
		return nil
	}
	_ = payload[0]  // channel
	_ = payload[1:] // data
	return nil
}

func (s *session) handleNotifyAction() error {
	payload := make([]byte, 4)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read notify action payload: %w", err)
	}
	if !s.supportsExt(cpeExtNotifyAction) {
		return nil
	}
	_ = payload[0] // entityID
	_ = payload[1] // action type
	_ = int16(binary.BigEndian.Uint16(payload[2:4]))
	return nil
}

func (s *session) handleNotifyPositionAction() error {
	payload := make([]byte, 8)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read notify position action payload: %w", err)
	}
	if !s.supportsExt(cpeExtNotifyAction) {
		return nil
	}
	_ = payload[0]                            // entityID
	_ = payload[1]                            // action type
	_ = binary.BigEndian.Uint16(payload[2:4]) // x
	_ = binary.BigEndian.Uint16(payload[4:6]) // y
	_ = binary.BigEndian.Uint16(payload[6:8]) // z
	return nil
}
