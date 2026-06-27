// handlers.go processes incoming client packets.
//
// Each handler reads the packet payload from the session's buffered
// reader, validates it, applies the change to the world/session, and
// fires the appropriate plugin events.
//
// Handlers:
//   - handleSetBlock: player places or removes a block
//   - handleEntityTeleport: player moves (full position update)
//   - handleRelativePosition*: player moves (delta-encoded position)
//   - handleMessage: player sends chat or a command
//   - applyBlockChange: shared block change logic (fires OnBlockChange,
//     writes to world, records in BlockDB, tracks stats, broadcasts)
//
// All handlers are called from the session read loop (session.run).
// They must not block on network I/O — use writePacket for responses.

package classic

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
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

	// If a drawing selection is active, intercept the click as a mark.
	if s.markState != nil {
		s.touchLastAction()
		s.RevertBlock(x, y, z) // don't actually place the block
		s.markState.marks[s.markState.index] = markPos{x, y, z}
		s.markState.index++
		s.Message(s.Tr("command.draw.mark") + " #" + fmt.Sprintf("%d", s.markState.index))
		if s.markState.index >= len(s.markState.marks) {
			cb := s.markState.callback
			marks := s.markState.marks
			s.markState = nil // clear selection
			cb(marks)
		}
		return nil
	}

	if !s.AllowBuild() {
		if s.worlds != nil {
			if block, ok := s.worlds.BlockAt(x, y, z); ok {
				s.SendBlockChange(x, y, z, block)
			}
		}
		return nil
	}

	// Per-block permission check.
	if place && !s.CanPlaceBlock(blockID) {
		if s.worlds != nil {
			if block, ok := s.worlds.BlockAt(x, y, z); ok {
				s.SendBlockChange(x, y, z, block)
			}
		}
		s.Message(s.Tr("command.draw.cannot_place", blockID))
		return nil
	}
	if !place {
		// Check the block being deleted
		if s.worlds != nil {
			if old, ok := s.worlds.BlockAt(x, y, z); ok {
				if !s.CanDeleteBlock(old) {
					s.SendBlockChange(x, y, z, old)
					s.Message(s.Tr("command.draw.cannot_delete", old))
					return nil
				}
			}
		}
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

	// Update last activity for AFK detection.
	s.touchLastAction()

	// Check for special blocks at the player's feet.
	s.checkSpecialBlocks(position.X/32, (position.Y/32)-1, position.Z/32)

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

	// Update last activity for AFK detection.
	s.touchLastAction()

	// Check special blocks for the moving player.
	if targetID == s.currentEntityID() {
		x, y, z := s.Position()
		s.checkSpecialBlocks(x/32, (y/32)-1, z/32)
	}
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

	// Update last activity for AFK detection.
	s.touchLastAction()

	// Anti-spam chat check.
	if s.spamChecker != nil {
		if s.spamChecker.IsMuted(s.currentUsername()) {
			s.Message(s.Tr("player.muted"))
			return nil
		}
		r := s.spamChecker.CheckChat(s.currentUsername())
		if r.Exceeded {
			s.handleSpamResult(r)
			if r.Action == player.SpamActionKick {
				return nil
			}
			s.Message(s.Tr("player.chat_exceeded", r.Count, r.Max))
			return nil
		}
	}

	// Track message count in PlayerDB.
	if s.playerDB != nil {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			e.MessagesSent++
			s.playerDB.Save(e)
		}
	}

	if s.IsMuted() {
		s.Message(s.Tr("general.muted"))
		return nil
	}

	if strings.HasPrefix(text, "//") {
		text = text[1:]
	} else if strings.HasPrefix(text, "/") {
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

	formatted := fmt.Sprintf("%s<%s> &f%s", s.Color(), s.currentUsername(), text)
	if plugin.OnChatFrom.HasHandlers() {
		plugin.OnChatFrom.Fire(plugin.ChatFromData{Source: s, Message: &formatted})
	}
	if plugin.OnChat.HasHandlers() {
		plugin.OnChat.Fire(plugin.ChatData{Source: s, Message: &formatted})
	}
	s.Message(formatted)

	// Broadcast to peers, respecting ignore lists and level filtering.
	s.room.ForEachPeerExcept(s.currentEntityID(), func(peer *session) {
		if peer.CurrentWorldManager() != s.CurrentWorldManager() {
			return
		}
		if peer.isIgnoring(s.currentUsername()) {
			return
		}
		peer.Message(formatted)
	})
	return nil
}

// handleSpamResult applies the configured action when a rate limit is exceeded.
func (s *session) handleSpamResult(r player.SpamResult) {
	switch r.Action {
	case player.SpamActionKick:
		s.disconnect(s.Tr("player.kick"))
	case player.SpamActionMute:
		s.Message(s.Tr("player.muted"))
	case player.SpamActionWarn:
		s.Message(s.Tr("player.warn", r.Count, r.Max))
	}
}

func (s *session) handleCommand(line string) error {
	if s.commands == nil {
		return nil
	}

	// Anti-spam command check.
	if s.spamChecker != nil {
		r := s.spamChecker.CheckCommand(s.currentUsername())
		if r.Exceeded {
			s.handleSpamResult(r)
			if r.Action == player.SpamActionKick {
				return nil
			}
			s.Message(s.Tr("player.cmd_exceeded", r.Count, r.Max))
			return nil
		}
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
	s.Message(reply)
	return nil
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

func specialBlockType(b byte) blocks.SpecialType {
	if blocks.IsMessageBlock(b) {
		return blocks.SpecialMessage
	}
	if blocks.IsPortal(b) {
		return blocks.SpecialPortal
	}
	if blocks.IsDoor(b) {
		return blocks.SpecialDoor
	}
	return blocks.SpecialNone
}

// checkSpecialBlocks fires message blocks and portals at the player's position.
// Only fires once per block — tracks the last checked position to avoid spam.
func (s *session) checkSpecialBlocks(x, y, z int) {
	if s.specialBlocks == nil || s.worlds == nil {
		return
	}
	// Skip if same block as last check.
	if x == s.lastSpecialBlock[0] && y == s.lastSpecialBlock[1] && z == s.lastSpecialBlock[2] {
		return
	}
	s.lastSpecialBlock = [3]int{x, y, z}

	// Check the block at feet level and one below.
	for dy := 0; dy >= -1; dy-- {
		entry := s.specialBlocks.Get(x, y+dy, z)
		if entry == nil {
			continue
		}
		switch entry.Type {
		case blocks.SpecialMessage:
			if entry.Message != "" {
				msg := strings.ReplaceAll(entry.Message, "@p", s.currentUsername())
				// MCGalaxy: if message starts with /, execute as command.
				// Supports piped commands: "text |/cmd1 |/cmd2"
				if strings.HasPrefix(msg, "/") {
					s.handleCommand(msg)
				} else if idx := strings.Index(msg, " |/"); idx >= 0 {
					text := msg[:idx]
					if text != "" {
						s.Message(text)
					}
					rest := msg[idx+3:] // skip " |/"
					for _, part := range strings.Split(rest, " |/") {
						cmd := strings.TrimSpace(part)
						if cmd != "" {
							s.handleCommand("/" + cmd)
						}
					}
				} else {
					s.Message(msg)
				}
			}
		case blocks.SpecialPortal:
			if entry.PortalLevel != "" {
				// Cross-level portal — switch level, then teleport to destination.
				if s.gotoLevel != nil {
					if s.gotoLevel(s, entry.PortalLevel) {
						s.teleportSelf(entry.PortalDst[0], entry.PortalDst[1], entry.PortalDst[2], s.Yaw(), s.Pitch())
					}
				}
			} else {
				// Same-level portal — teleport.
				s.teleportSelf(entry.PortalDst[0], entry.PortalDst[1], entry.PortalDst[2], s.Yaw(), s.Pitch())
			}
		case blocks.SpecialDoor:
			// Door toggle: air ↔ solid block.
			if s.worlds != nil {
				b, ok := s.worlds.BlockAt(x, y+dy, z)
				if ok {
					if b == 0 {
						s.applyBlockChange(x, y+dy, z, entry.DoorBlock, true)
					} else {
						s.applyBlockChange(x, y+dy, z, 0, true)
					}
				}
			}
		}
	}
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

	// Record old block before overwriting.
	var oldBlock byte
	if b, ok := s.worlds.BlockAt(x, y, z); ok {
		oldBlock = b
	}

	// Anti-spam block check (before SetBlock so we can reject).
	if s.spamChecker != nil {
		r := s.spamChecker.CheckBlock(s.currentUsername())
		if r.Exceeded {
			s.handleSpamResult(r)
			// Revert the block on the client so they see it didn't work.
			if s.worlds != nil {
				if block, ok := s.worlds.BlockAt(x, y, z); ok {
					s.SendBlockChange(x, y, z, block)
				}
			}
			return nil
		}
	}

	if !s.worlds.SetBlock(x, y, z, blockID) {
		return fmt.Errorf("block position out of bounds: %d %d %d", x, y, z)
	}

	// Register/remove special blocks.
	if s.specialBlocks != nil {
		if placing {
			if blocks.IsSpecialBlock(blockID) && !blocks.IsTNT(blockID) {
				s.specialBlocks.Set(x, y, z, &blocks.SpecialEntry{
					Type: specialBlockType(blockID),
				})
			}
			// Remove special block entry if overwriting one with a non-special block.
			// But preserve door entries — the door toggle places solid blocks (Log, etc.)
			// which are not special, and we must not delete the door's registry entry.
			if !blocks.IsSpecialBlock(blockID) {
				if existing := s.specialBlocks.Get(x, y, z); existing != nil {
					if existing.Type != blocks.SpecialDoor {
						s.specialBlocks.Remove(x, y, z)
					}
				}
			}
		} else {
			// Block deleted (air) — remove any special block entry at this position.
			// Door toggle uses applyBlockChange with blockID=0, but door entries
			// must survive the toggle. We detect the door toggle by checking if
			// the existing entry is a door — if so, keep it.
			if existing := s.specialBlocks.Get(x, y, z); existing != nil {
				if existing.Type != blocks.SpecialDoor {
					s.specialBlocks.Remove(x, y, z)
				}
			}
		}
	}

	// Queue block for physics processing.
	if s.queuePhysics != nil {
		s.queuePhysics(s.worlds, x, y, z)
	}

	// Record in BlockDB.
	if s.blockDB != nil {
		s.blockDB.Add(plugin.BlockEntry{
			PlayerID: s.playerDBID,
			Time:     time.Now(),
			X:        x, Y: y, Z: z,
			OldBlock: oldBlock,
			NewBlock: blockID,
			Flags:    plugin.BlockManualPlace,
		})
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
