// broadcast.go implements entity visibility and packet fan-out.
//
// The room (from internal/session.Room) tracks all online sessions.
// Broadcast methods on Codec iterate the room and send packets to
// qualifying peers. Per-level filtering ensures players only see
// entities and block changes on their current level.
//
// Session-level methods (joinRoom, leaveRoom, broadcastToPeers, canSee)
// handle entity spawn/despawn and per-level visibility:
//   - joinRoom: spawns the player's entity for same-level peers and vice versa
//   - leaveRoom: despawns the player's entity from same-level peers
//   - broadcastToPeers: sends a packet to all same-level peers
//   - canSee: consults OnGettingCanSee event for visibility overrides
//
// Codec.BroadcastEntityUpdates runs the per-tick entity position broadcast
// with delta encoding (RelPos/RelPosAndOrient/Orientation/Teleport).

package classic

import (
	"fmt"
	"time"

	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

func (c *Codec) KickAll(reason string) {
	pkt := encodeKick(reason)
	c.room.ForEachPeerExcept(0, func(peer *session) {
		_ = peer.writePacket(pkt)
		peer.fail()
	})
}

// BroadcastMessage sends a chat message to all online players.
func (c *Codec) BroadcastMessage(msg string) {
	if plugin.OnChatSys.HasHandlers() {
		m := msg
		plugin.OnChatSys.Fire(plugin.ChatSysData{Message: &m})
		msg = m
	}
	if plugin.OnChat.HasHandlers() {
		m := msg
		plugin.OnChat.Fire(plugin.ChatData{Source: nil, Message: &m})
		msg = m
	}
	for _, p := range c.OnlinePlayers() {
		p.Message(msg)
	}
}

// OnlinePlayers returns all currently connected sessions as plugin.Player.
func (c *Codec) OnlinePlayers() []plugin.Player {
	peers := c.room.Snapshot()
	out := make([]plugin.Player, len(peers))
	for i, p := range peers {
		out[i] = p
	}
	return out
}

// GetPlayerAFKState returns the player's last action time, AFK-since time,
// and AFK status.
func (c *Codec) GetPlayerAFKState(name string) (time.Time, time.Time, bool) {
	p, ok := c.room.FindByName(name)
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	return p.LastAction(), p.AfkSince(), p.IsAfk()
}

// GetPlayerRank returns the player's rank permission level.
func (c *Codec) GetPlayerRank(name string) int {
	p, ok := c.room.FindByName(name)
	if !ok {
		return 0
	}
	return p.PlayerRank()
}

// FindPlayer returns the online session with the given name (case-insensitive).
func (c *Codec) FindPlayer(name string) plugin.Player {
	p, ok := c.room.FindByName(name)
	if !ok {
		return nil
	}
	return p
}

// BroadcastPacket sends a raw packet to all online players.
func (c *Codec) BroadcastPacket(packet []byte) {
	packet = append([]byte(nil), packet...)
	c.room.ForEachPeerExcept(0, func(peer *session) {
		_ = peer.writePacketNoCopy(packet)
	})
}

// BroadcastPacketToLevel sends a raw packet only to players on the given level.
func (c *Codec) BroadcastPacketToLevel(mgr *world.Manager, packet []byte) {
	packet = append([]byte(nil), packet...)
	c.room.ForEachPeerExcept(0, func(peer *session) {
		if peer.CurrentWorldManager() == mgr {
			_ = peer.writePacketNoCopy(packet)
		}
	})
}

// BroadcastAddEntity broadcasts an add-entity packet to all players.
func (c *Codec) BroadcastAddEntity(id byte, name string, x, y, z int, yaw, pitch byte) {
	c.BroadcastPacket(encodeAddEntity(id, name, entity.Position{X: x, Y: y, Z: z}, yaw, pitch))
}

// BroadcastRemoveEntity broadcasts a remove-entity packet to all players.
func (c *Codec) BroadcastRemoveEntity(id byte) {
	c.BroadcastPacket(encodeRemoveEntity(id))
}

// BroadcastEntityTeleport broadcasts an entity-teleport packet to all players.
func (c *Codec) BroadcastEntityTeleport(id byte, x, y, z int, yaw, pitch byte) {
	c.BroadcastPacket(encodeEntityTeleport(id, entity.Position{X: x, Y: y, Z: z}, yaw, pitch))
}

// BroadcastChangeModel broadcasts a change-model packet to all players.
func (c *Codec) BroadcastChangeModel(entityID byte, model string) {
	c.BroadcastPacket(encodeChangeModel(entityID, model))
}

// EncodeAddEntity returns an add-entity packet. Exported for callers
// that need to batch or route packets themselves (e.g. per-level).
func (c *Codec) EncodeAddEntity(id byte, name string, x, y, z int, yaw, pitch byte) []byte {
	return encodeAddEntity(id, name, entity.Position{X: x, Y: y, Z: z}, yaw, pitch)
}

// EncodeRemoveEntity returns a remove-entity packet.
func (c *Codec) EncodeRemoveEntity(id byte) []byte { return encodeRemoveEntity(id) }

// EncodeEntityTeleport returns an entity-teleport packet.
func (c *Codec) EncodeEntityTeleport(id byte, x, y, z int, yaw, pitch byte) []byte {
	return encodeEntityTeleport(id, entity.Position{X: x, Y: y, Z: z}, yaw, pitch)
}

// EncodeChangeModel returns a change-model packet.
func (c *Codec) EncodeChangeModel(entityID byte, model string) []byte {
	return encodeChangeModel(entityID, model)
}

// ChangeMap sends a different level to the player and switches their active
// world Manager. The player must be online.
func (c *Codec) ChangeMap(p plugin.Player, mgr *world.Manager) error {
	s, ok := p.(*session)
	if !ok {
		return fmt.Errorf("player is not a classic session")
	}
	return s.changeMap(mgr)
}

// PlayersOnLevel returns all online sessions whose active world Manager
// matches mgr (by pointer identity).
func (c *Codec) PlayersOnLevel(mgr *world.Manager) []plugin.Player {
	peers := c.room.Snapshot()
	var out []plugin.Player
	for _, s := range peers {
		if s.CurrentWorldManager() == mgr {
			out = append(out, s)
		}
	}
	return out
}

// PlayerWorldManager returns the active world Manager for the given player,
// or nil if the player is not found.
func (c *Codec) PlayerWorldManager(p plugin.Player) *world.Manager {
	s, ok := c.room.FindByName(p.Name())
	if !ok {
		return nil
	}
	return s.CurrentWorldManager()
}

// MainWorldManager returns the codec's default world Manager.
func (c *Codec) MainWorldManager() *world.Manager {
	return c.worlds
}

// BroadcastSetBlockToLevel sends a set-block packet to all players on the
// level whose Manager matches mgr (by pointer identity).
func (c *Codec) BroadcastSetBlockToLevel(mgr *world.Manager, x, y, z int, block byte) {
	packet := encodeSetBlock(x, y, z, block)
	c.room.ForEachPeerExcept(0, func(peer *session) {
		peer.stateMu.RLock()
		w := peer.worlds
		peer.stateMu.RUnlock()
		if w == mgr {
			_ = peer.writePacket(packet)
		}
	})
}

// BroadcastEntityUpdates runs the per-tick entity position broadcast.
// It snapshots all sessions, finds entities whose position/rotation changed
// since the last tick, and sends each recipient a single concatenated buffer.
// Uses delta encoding: small movements go as RelPos/RelPosAndOrient packets
// (which the client interpolates smoothly); large jumps use EntityTeleport.
// Modeled on MCGalaxy's BroadcastEntityPositions.
func (c *Codec) BroadcastEntityUpdates() {
	peers := c.room.Snapshot()

	if c.sendTimeoutMode == "adaptive" {
		n := len(peers)
		t := adaptiveBase + time.Duration(n)*adaptivePerPlayer
		if t > adaptiveMax {
			t = adaptiveMax
		}
		c.sendTimeoutVal.Store(int64(t))
	}

	if len(peers) < 2 {
		return
	}

	type change struct {
		sess  *session
		pkt   []byte
		pos   entity.Position
		yaw   byte
		pitch byte
	}

	changes := make([]change, 0, len(peers))
	for _, s := range peers {
		eid := s.currentEntityID()
		if eid == 0 {
			continue
		}
		snap, ok := s.entitySnapshot()
		if !ok {
			continue
		}

		lastPos, lastYaw, lastPitch := s.lastBroadcast()
		posChanged := snap.Pos != lastPos
		oriChanged := snap.Yaw != lastYaw || snap.Pitch != lastPitch
		if !posChanged && !oriChanged {
			continue
		}

		var pkt []byte
		id := byte(eid)
		dx, dy, dz := snap.Pos.X-lastPos.X, snap.Pos.Y-lastPos.Y, snap.Pos.Z-lastPos.Z

		switch {
		case posChanged && fitsRelDelta(dx) && fitsRelDelta(dy) && fitsRelDelta(dz) && oriChanged:
			pkt = encodeRelPosAndOrient(id, dx, dy, dz, snap.Yaw, snap.Pitch)
		case posChanged && fitsRelDelta(dx) && fitsRelDelta(dy) && fitsRelDelta(dz):
			pkt = encodeRelPos(id, dx, dy, dz)
		case !posChanged && oriChanged:
			pkt = encodeOrientation(id, snap.Yaw, snap.Pitch)
		default:
			pkt = encodeEntityTeleport(id, snap.Pos, snap.Yaw, snap.Pitch)
		}

		changes = append(changes, change{
			sess:  s,
			pkt:   pkt,
			pos:   snap.Pos,
			yaw:   snap.Yaw,
			pitch: snap.Pitch,
		})
	}

	if len(changes) == 0 {
		return
	}

	for _, dst := range peers {
		dstWorld := dst.CurrentWorldManager()
		var buf []byte
		for _, ch := range changes {
			if ch.sess == dst {
				continue
			}
			// Only send entity updates to peers on the same level.
			if ch.sess.CurrentWorldManager() != dstWorld {
				continue
			}
			buf = append(buf, ch.pkt...)
		}
		if len(buf) > 0 {
			_ = dst.writePacketNoCopy(buf)
		}
	}

	for _, ch := range changes {
		ch.sess.setLastBroadcast(ch.pos, ch.yaw, ch.pitch)
	}
}

func (s *session) joinRoom() {
	if s.room == nil {
		return
	}

	_, entityID, _ := s.sessionIdentity()
	_, joined := s.sessionFlags()
	if joined || entityID == 0 {
		return
	}

	selfState, ok := s.entitySnapshot()
	if !ok {
		return
	}

	peers := s.room.Join(s)

	username := s.currentUsername()
	selfPacket := encodeAddEntity(byte(entityID), username, selfState.Pos, selfState.Yaw, selfState.Pitch)
	myWorld := s.CurrentWorldManager()
	for _, peer := range peers {
		// Only spawn entities for peers on the same level.
		if peer.CurrentWorldManager() != myWorld {
			continue
		}
		peerState, ok := peer.entitySnapshot()
		if ok {
			peerUsername, peerEntityID, _ := peer.sessionIdentity()
			if s.canSee(peer) {
				if err := s.writePacket(encodeAddEntity(byte(peerEntityID), peerUsername, peerState.Pos, peerState.Yaw, peerState.Pitch)); err != nil {
					s.logger.Debug("send peer join packet", "username", s.currentUsername(), "peer", peerUsername, "error", err)
				}
				if s.supportsExt(cpeExtPlayerListName) {
					if err := s.writePacket(encodeExtAddPlayerName(byte(peerEntityID), peerUsername)); err != nil {
						s.logger.Debug("send peer list packet", "username", s.currentUsername(), "peer", peerUsername, "error", err)
					}
				}
			}
		}
		if peer.canSee(s) {
			if err := peer.writePacket(selfPacket); err != nil {
				s.logger.Debug("broadcast join packet", "username", username, "peer", peer.currentUsername(), "error", err)
			}
			if peer.currentSupportsExtPlayerList() {
				if err := peer.writePacket(encodeExtAddPlayerName(byte(entityID), username)); err != nil {
					s.logger.Debug("broadcast player list packet", "username", username, "peer", peer.currentUsername(), "error", err)
				}
			}
		}
	}

	s.markJoined(true)
}

// canSee reports whether s can see peer, consulting OnGettingCanSee handlers.
// ponytail: fires the event per check; join-only path is not hot.
// If no handlers are registered, returns true (default visible).
func (s *session) canSee(peer *session) bool {
	if !plugin.OnGettingCanSee.HasHandlers() {
		return true
	}
	canSee := true
	plugin.OnGettingCanSee.Fire(plugin.GettingCanSeeData{
		Player: s,
		Target: peer,
		CanSee: &canSee,
	})
	return canSee
}

func (s *session) leaveRoom() {
	if s.room == nil {
		return
	}

	_, entityID, _ := s.sessionIdentity()
	_, joined := s.sessionFlags()
	if !joined || entityID == 0 {
		return
	}

	peers := s.room.Leave(entityID)
	myWorld := s.CurrentWorldManager()
	packet := encodeRemoveEntity(byte(entityID))
	for _, peer := range peers {
		if peer.CurrentWorldManager() != myWorld {
			continue
		}
		if err := peer.writePacket(packet); err != nil {
			s.logger.Debug("broadcast leave packet", "entity_id", entityID, "peer", peer.currentUsername(), "error", err)
		}
		if peer.currentSupportsExtPlayerList() {
			if err := peer.writePacket(encodeExtRemovePlayerName(byte(entityID))); err != nil {
				s.logger.Debug("broadcast player list removal", "entity_id", entityID, "peer", peer.currentUsername(), "error", err)
			}
		}
	}
	s.markJoined(false)
}

func (s *session) broadcastToPeers(packet []byte) {
	if s.room == nil {
		return
	}

	_, entityID, _ := s.sessionIdentity()
	if entityID == 0 {
		return
	}

	myWorld := s.CurrentWorldManager()

	s.room.ForEachPeerExcept(entityID, func(peer *session) {
		// ponytail: filter by world manager pointer identity — peers on
		// other levels don't see entity updates from this level.
		if peer.CurrentWorldManager() != myWorld {
			return
		}
		if err := peer.writePacketNoCopy(packet); err != nil {
			s.logger.Debug("broadcast packet", "entity_id", entityID, "peer", peer.currentUsername(), "error", err)
		}
	})
}

func (s *session) entitySnapshot() (entity.Entity, bool) {
	if s.entities == nil {
		return entity.Entity{}, false
	}
	entityID := s.currentEntityID()
	if entityID == 0 {
		return entity.Entity{}, false
	}
	return s.entities.Get(entityID)
}
