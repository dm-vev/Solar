package classic

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/protocol/wire"
	"github.com/solar-mc/solar/plugin"
)

// payloadPool reuses small read buffers used by packet handlers.
// The largest fixed payload is handleMessage at 65 bytes.
var payloadPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 65)
		return &buf
	},
}

func acquirePayload(size int) []byte {
	buf := payloadPool.Get().(*[]byte)
	if cap(*buf) < size {
		newBuf := make([]byte, size)
		payloadPool.Put(buf)
		return newBuf
	}
	*buf = (*buf)[:size]
	return *buf
}

func releasePayload(buf []byte) {
	if cap(buf) < 65 {
		return
	}
	payloadPool.Put(&buf)
}

func (s *session) handleSetBlock() error {
	payload := acquirePayload(8)
	defer releasePayload(payload)
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

	if !s.AllowBuild() {
		if s.worlds != nil {
			if block, ok := s.worlds.BlockAt(x, y, z); ok {
				s.SendBlockChange(x, y, z, block)
			}
		}
		return nil
	}

	return s.applyBlockChange(x, y, z, blockID, true)
}

func (s *session) handleEntityTeleport() error {
	payload := acquirePayload(9)
	defer releasePayload(payload)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read entity teleport payload: %w", err)
	}

	if s.IsFrozen() {
		return nil
	}

	// Client always sends its own position. The ID byte is either
	// selfID (255) or a held block ID (HeldBlock CPE extension).
	targetID := s.currentEntityID()
	position := decodeClassicPosition(payload[1:7])
	yaw := payload[7]
	pitch := payload[8]

	if s.entities != nil {
		s.entities.SetLocation(targetID, position, yaw, pitch)
	}

	if plugin.OnPlayerMove.HasHandlers() {
		plugin.OnPlayerMove.Fire(plugin.PlayerMoveData{
			Player: s,
			X:      position.X,
			Y:      position.Y,
			Z:      position.Z,
			Yaw:    yaw,
			Pitch:  pitch,
		})
	}
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
	payload := acquirePayload(6)
	defer releasePayload(payload)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read relative position+orientation payload: %w", err)
	}

	if s.IsFrozen() {
		return nil
	}

	targetID := s.resolveEntityID(payload[0])
	return s.applyRelativeUpdate(targetID, payload[1], payload[2], payload[3], payload[4], payload[5])
}

func (s *session) handleRelativePositionOnly() error {
	payload := acquirePayload(4)
	defer releasePayload(payload)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read relative position payload: %w", err)
	}

	if s.IsFrozen() {
		return nil
	}

	targetID := s.resolveEntityID(payload[0])
	return s.applyRelativeUpdate(targetID, payload[1], payload[2], payload[3], 0, 0)
}

func (s *session) handleOrientationOnly() error {
	payload := acquirePayload(3)
	defer releasePayload(payload)
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

	s.entities.ApplyDelta(targetID,
		decodeClassicDelta(dx), decodeClassicDelta(dy), decodeClassicDelta(dz),
		yaw, pitch)
	return nil
}

func (s *session) handleMessage() error {
	payload := acquirePayload(65)
	defer releasePayload(payload)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read message payload: %w", err)
	}

	text := readFixedString(payload[1:])
	if text == "" {
		return nil
	}

	// Track message count in PlayerDB.
	if s.playerDB != nil {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			e.MessagesSent++
			s.playerDB.Save(e)
		}
	}

	if s.IsMuted() {
		s.Message("&cYou are muted.")
		return nil
	}

	if strings.HasPrefix(text, "/") {
		return s.handleCommand(text)
	}

	if plugin.OnPlayerChat.HasHandlers() {
		msg := text
		ctx := plugin.OnPlayerChat.Fire(plugin.PlayerChatData{
			Player:  s,
			Message: &msg,
		})
		if ctx.Cancelled() {
			return nil
		}
		text = msg
	}

	packet := encodeMessage(selfID, fmt.Sprintf("<%s> %s", s.currentUsername(), text))
	formatted := readFixedString(packet[2:])
	if plugin.OnChatFrom.HasHandlers() {
		plugin.OnChatFrom.Fire(plugin.ChatFromData{Source: s, Message: &formatted})
	}
	if plugin.OnChat.HasHandlers() {
		plugin.OnChat.Fire(plugin.ChatData{Source: s, Message: &formatted})
	}
	if formatted != readFixedString(packet[2:]) {
		packet = encodeMessage(selfID, formatted)
	}
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

	if plugin.OnPlayerCommand.HasHandlers() {
		cmdLine := strings.TrimPrefix(line, "/")
		parts := strings.SplitN(cmdLine, " ", 2)
		cmdName := parts[0]
		cmdArgs := ""
		if len(parts) > 1 {
			cmdArgs = parts[1]
		}
		ctx := plugin.OnPlayerCommand.Fire(plugin.PlayerCommandData{
			Player:  s,
			Command: cmdName,
			Args:    cmdArgs,
		})
		if ctx.Cancelled() {
			return nil
		}
		if cmdName == "help" && plugin.OnPlayerHelp.HasHandlers() {
			helpCtx := plugin.OnPlayerHelp.Fire(plugin.PlayerHelpData{
				Player: s,
				Target: cmdArgs,
			})
			if helpCtx.Cancelled() {
				return nil
			}
		}
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

	placing := blockID != 0
	if plugin.OnBlockChange.HasHandlers() {
		ctx := plugin.OnBlockChange.Fire(plugin.BlockChangeData{
			Player:  s,
			X:       x,
			Y:       y,
			Z:       z,
			Block:   blockID,
			Placing: placing,
		})
		if ctx.Cancelled() {
			return nil
		}
	}

	if !s.worlds.SetBlock(x, y, z, blockID) {
		return fmt.Errorf("block position out of bounds: %d %d %d", x, y, z)
	}

	// Track block stats in PlayerDB.
	if s.playerDB != nil {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			if placing {
				e.BlocksPlaced++
			} else {
				e.BlocksDeleted++
			}
			s.playerDB.Save(e)
		}
	}

	if plugin.OnBlockChanged.HasHandlers() {
		plugin.OnBlockChanged.Fire(plugin.BlockChangedData{
			Player:  s,
			X:       x,
			Y:       y,
			Z:       z,
			Block:   blockID,
			Placing: placing,
		})
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

// decodeClassicPosition extracts a wire-coordinate position from a 6-byte
// payload. Coordinates are stored as-is in wire units (1/32 block).
func decodeClassicPosition(payload []byte) entity.Position {
	return entity.Position{
		X: int(int16(binary.BigEndian.Uint16(payload[0:2]))),
		Y: int(int16(binary.BigEndian.Uint16(payload[2:4]))) - eyeHeight,
		Z: int(int16(binary.BigEndian.Uint16(payload[4:6]))),
	}
}

// decodeClassicDelta extracts a signed wire-unit delta from a byte.
func decodeClassicDelta(raw byte) int {
	return int(int8(raw))
}

func (s *session) writeKick(message string) error {
	return s.writePacket(encodeKick(message))
}

// writePacket queues a packet for asynchronous writing. The packet slice
// is copied because the caller may reuse the underlying buffer.
// Blocks up to sendTimeout if the outbox is full, then disconnects.
func (s *session) writePacket(packet []byte) error {
	packetCopy := append([]byte(nil), packet...)
	select {
	case s.outbox <- packetCopy:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	default:
	}
	timeout := time.Duration(s.sendTimeoutVal.Load())
	if timeout <= 0 {
		s.fail()
		return io.ErrShortWrite
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case s.outbox <- packetCopy:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	case <-timer.C:
		s.fail()
		return io.ErrShortWrite
	}
}

// writePacketNoCopy queues a broadcast packet without copying. The caller must
// guarantee that the packet slice will not be modified after this call.
// Drops the packet if the outbox is full — broadcast packets are best-effort.
func (s *session) writePacketNoCopy(packet []byte) error {
	select {
	case s.outbox <- packet:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	default:
		return nil
	}
}

func (s *session) writeLoop() {
	defer close(s.writerDone)
	defer s.closeConn()

	for {
		select {
		case packet := <-s.outbox:
			if !s.writePacketBuffer(packet) {
				return
			}
			// Batch any additional packets already queued so that small
			// broadcasts do not trigger a syscall per packet.
			if !s.drainOutbox(s.writeBatchSize) {
				return
			}
			if !s.flushWriter() {
				return
			}
		case <-s.stop:
			if !s.drainOutbox(s.shutdownBatchSize) {
				return
			}
			if !s.flushWriter() {
				return
			}
			return
		}
	}
}

// writePacketBuffer writes a packet to the buffered writer without flushing.
func (s *session) writePacketBuffer(packet []byte) bool {
	if packet == nil {
		return false
	}
	if s.writeDeadline > 0 {
		if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeDeadline)); err != nil {
			return false
		}
	}
	_, err := s.writer.Write(packet)
	return err == nil
}

// drainOutbox pulls up to max packets from the outbox and writes them without
// flushing. Returns false if a write failed and the loop should stop.
func (s *session) drainOutbox(max int) bool {
	for i := 0; i < max; i++ {
		select {
		case packet := <-s.outbox:
			if !s.writePacketBuffer(packet) {
				return false
			}
		default:
			return true
		}
	}
	return true
}

// flushWriter flushes the buffered writer. Returns false on error.
func (s *session) flushWriter() bool {
	return s.writer.Flush() == nil
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
