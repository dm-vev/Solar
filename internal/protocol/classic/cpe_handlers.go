package classic

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/solar-mc/solar/plugin"
)

func (s *session) handlePlayerClick() error {
	payload := make([]byte, 14)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read player click payload: %w", err)
	}
	if !s.supportsExt(cpeExtPlayerClick) {
		return nil
	}

	if plugin.OnPlayerClick.HasHandlers() {
		plugin.OnPlayerClick.Fire(plugin.PlayerClickData{
			Player:   s,
			Button:   payload[0],
			Action:   payload[1],
			EntityID: payload[6],
			X:        int(binary.BigEndian.Uint16(payload[7:9])),
			Y:        int(binary.BigEndian.Uint16(payload[9:11])),
			Z:        int(binary.BigEndian.Uint16(payload[11:13])),
			Face:     payload[13],
		})
	}
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

	if plugin.OnPluginMessage.HasHandlers() {
		plugin.OnPluginMessage.Fire(plugin.CpePluginMessageData{
			Player:  s,
			Channel: payload[0],
			Data:    append([]byte(nil), payload[1:]...),
		})
	}
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

	if plugin.OnNotifyAction.HasHandlers() {
		plugin.OnNotifyAction.Fire(plugin.NotifyActionData{
			Player: s,
			Action: payload[1],
			Value:  int(int16(binary.BigEndian.Uint16(payload[2:4]))),
		})
	}
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

	if plugin.OnNotifyPositionAction.HasHandlers() {
		plugin.OnNotifyPositionAction.Fire(plugin.NotifyPositionActionData{
			Player: s,
			Action: payload[1],
			X:      int(binary.BigEndian.Uint16(payload[2:4])),
			Y:      int(binary.BigEndian.Uint16(payload[4:6])),
			Z:      int(binary.BigEndian.Uint16(payload[6:8])),
		})
	}
	return nil
}
