package classic

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/protocol/wire"
)

func (s *session) handleSetBlock() error {
	payload := make([]byte, 8)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read set block payload: %w", err)
	}

	x := int(binary.BigEndian.Uint16(payload[0:2]))
	y := int(binary.BigEndian.Uint16(payload[2:4]))
	z := int(binary.BigEndian.Uint16(payload[4:6]))
	place := payload[6] != 0
	blockID := payload[7]
	if !place {
		blockID = 0
	}

	return s.applyBlockChange(x, y, z, blockID, true)
}

func (s *session) handleEntityTeleport() error {
	payload := make([]byte, 9)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read entity teleport payload: %w", err)
	}

	targetID := s.resolveEntityID(payload[0])
	position := decodeClassicPosition(payload[1:7])
	yaw := payload[7]
	pitch := payload[8]

	if s.entities != nil {
		s.entities.SetLocation(targetID, position, yaw, pitch)
	}
	s.broadcastToPeers(encodeEntityTeleport(byte(targetID), position, yaw, pitch))
	return nil
}

func (s *session) handleRelativePosition(includePosition, includeRotation bool) error {
	switch {
	case includePosition && includeRotation:
		return s.handleRelativePositionAndOrientation()
	case includePosition:
		return s.handleRelativePositionOnly()
	case includeRotation:
		return s.handleOrientationOnly()
	default:
		return nil
	}
}

func (s *session) handleRelativePositionAndOrientation() error {
	payload := make([]byte, 6)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read relative position+orientation payload: %w", err)
	}

	targetID := s.resolveEntityID(payload[0])
	return s.applyRelativeUpdate(targetID, payload[1], payload[2], payload[3], payload[4], payload[5])
}

func (s *session) handleRelativePositionOnly() error {
	payload := make([]byte, 4)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read relative position payload: %w", err)
	}

	targetID := s.resolveEntityID(payload[0])
	return s.applyRelativeUpdate(targetID, payload[1], payload[2], payload[3], 0, 0)
}

func (s *session) handleOrientationOnly() error {
	payload := make([]byte, 3)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read orientation payload: %w", err)
	}

	targetID := s.resolveEntityID(payload[0])
	return s.applyRelativeUpdate(targetID, 0, 0, 0, payload[1], payload[2])
}

func (s *session) applyRelativeUpdate(targetID uint32, dx, dy, dz, yaw, pitch byte) error {
	if s.entities == nil {
		return nil
	}

	entitySnapshot, ok := s.entities.Get(targetID)
	if !ok {
		return nil
	}

	position := entitySnapshot.Pos
	if dx != 0 || dy != 0 || dz != 0 {
		position.X += decodeClassicDelta(dx)
		position.Y += decodeClassicDelta(dy)
		position.Z += decodeClassicDelta(dz)
	}

	newYaw := entitySnapshot.Yaw
	newPitch := entitySnapshot.Pitch
	if yaw != 0 || pitch != 0 {
		newYaw = yaw
		newPitch = pitch
	}
	s.entities.SetLocation(targetID, position, newYaw, newPitch)
	s.broadcastToPeers(encodeEntityTeleport(byte(targetID), position, newYaw, newPitch))
	return nil
}

func (s *session) handleMessage() error {
	payload := make([]byte, 65)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read message payload: %w", err)
	}

	text := readFixedString(payload[1:])
	if text == "" {
		return nil
	}

	if strings.HasPrefix(text, "/") {
		return s.handleCommand(text)
	}

	packet := encodeMessage(selfID, fmt.Sprintf("<%s> %s", s.currentUsername(), text))
	if err := s.writePacket(packet); err != nil {
		return err
	}
	s.broadcastToPeers(packet)
	return nil
}

func (s *session) handleCommand(line string) error {
	if s.commands == nil {
		return nil
	}

	ctx := s.buildCommandContextFn()
	reply, handled := s.commands.Execute(ctx, line)
	if !handled || reply == "" {
		return nil
	}
	return s.writePacket(encodeMessage(selfID, reply))
}

// buildCommandContextFn assembles the command execution context. It uses
// the builder function injected via SetCommandContextBuilder, falling back
// to a nil context if no builder is configured.
func (s *session) buildCommandContextFn() command.Context {
	if s.buildCommandContext != nil {
		return s.buildCommandContext(s)
	}
	return command.Context{}
}

func (s *session) applyBlockChange(x, y, z int, blockID byte, echo bool) error {
	if s.worlds == nil {
		return nil
	}
	if !s.worlds.SetBlock(x, y, z, blockID) {
		return fmt.Errorf("block position out of bounds: %d %d %d", x, y, z)
	}

	packet := encodeSetBlock(x, y, z, blockID)
	if echo {
		if err := s.writePacket(packet); err != nil {
			return err
		}
	}
	s.broadcastToPeers(packet)
	return nil
}

func (s *session) resolveEntityID(packetID byte) uint32 {
	if packetID == selfID || packetID == 0 {
		return s.currentEntityID()
	}
	return uint32(packetID)
}

func decodeClassicPosition(payload []byte) entity.Position {
	return entity.Position{
		X: decodeClassicCoord(int(binary.BigEndian.Uint16(payload[0:2]))),
		Y: decodeClassicCoord(int(binary.BigEndian.Uint16(payload[2:4])) - eyeHeight),
		Z: decodeClassicCoord(int(binary.BigEndian.Uint16(payload[4:6]))),
	}
}

func decodeClassicCoord(raw int) int {
	return int(math.Round(float64(raw) / float64(coordScale)))
}

func decodeClassicDelta(raw byte) int {
	return int(math.Round(float64(int8(raw)) / float64(coordScale)))
}

func (s *session) writeKick(message string) error {
	return s.writePacket(encodeKick(message))
}

func (s *session) writePacket(packet []byte) error {
	packetCopy := append([]byte(nil), packet...)
	select {
	case s.outbox <- packetCopy:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	default:
		s.fail()
		return io.ErrShortWrite
	}
}

func (s *session) writeLoop() {
	defer close(s.writerDone)
	defer s.closeConn()

	for {
		select {
		case packet := <-s.outbox:
			if packet == nil {
				return
			}
			if _, err := s.writer.Write(packet); err != nil {
				return
			}
			if err := s.writer.Flush(); err != nil {
				return
			}
		case <-s.stop:
			for {
				select {
				case packet := <-s.outbox:
					if packet == nil {
						return
					}
					if _, err := s.writer.Write(packet); err != nil {
						return
					}
					if err := s.writer.Flush(); err != nil {
						return
					}
				default:
					return
				}
			}
		}
	}
}

func (s *session) fail() {
	s.closeStop()
}

func (s *session) disconnect(message string) {
	if strings.TrimSpace(message) == "" {
		message = "kicked"
	}
	_ = s.writeKick(message)
	s.fail()
}

func (s *session) closeStop() {
	s.stopOnce.Do(func() {
		close(s.stop)
	})
}

func (s *session) closeConn() {
	s.connOnce.Do(func() {
		if s.conn != nil {
			_ = s.conn.Close()
		}
	})
}

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
	binary.BigEndian.PutUint16(packet[66:68], uint16(pos.X*coordScale))
	binary.BigEndian.PutUint16(packet[68:70], uint16(pos.Y*coordScale+eyeHeight))
	binary.BigEndian.PutUint16(packet[70:72], uint16(pos.Z*coordScale))
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
	binary.BigEndian.PutUint16(packet[2:4], uint16(pos.X*coordScale))
	binary.BigEndian.PutUint16(packet[4:6], uint16(pos.Y*coordScale+eyeHeight))
	binary.BigEndian.PutUint16(packet[6:8], uint16(pos.Z*coordScale))
	packet[8] = yaw
	packet[9] = pitch
	return packet
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
