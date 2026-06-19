package classic

import "github.com/solar-mc/solar/plugin"

var _ plugin.Player = (*session)(nil)

// ─── remaining plugin.Player methods on *session ───
//
// Backing state lives on the session struct (color, model, hidden, muted,
// frozen, afk, allowBuild) under stateMu. Defaults are set in ServeConn.

func (s *session) SendCpeMessage(messageType byte, msg string) {
	_ = s.writePacket(encodeMessage(messageType, msg))
}

func (s *session) SetColor(color string) {
	s.stateMu.Lock()
	s.color = color
	s.stateMu.Unlock()
}

func (s *session) Color() string {
	s.stateMu.RLock()
	c := s.color
	s.stateMu.RUnlock()
	return c
}

func (s *session) SetModel(model string) {
	s.stateMu.Lock()
	s.model = model
	s.stateMu.Unlock()
	entityID := s.currentEntityID()
	if entityID != 0 {
		_ = s.writePacket(encodeChangeModel(byte(entityID), model))
		s.broadcastToPeers(encodeChangeModel(byte(entityID), model))
	}
}

func (s *session) Model() string {
	s.stateMu.RLock()
	m := s.model
	s.stateMu.RUnlock()
	return m
}

func (s *session) SetSkin(skinURL string) {
	// ponytail: no dedicated skin packet in classic; reuse ChangeModel so the
	// client reloads the named skin. Add a real SkinURL packet if a CPE ext needs it.
	entityID := s.currentEntityID()
	if entityID != 0 {
		_ = s.writePacket(encodeChangeModel(byte(entityID), skinURL))
	}
}

func (s *session) IsHidden() bool {
	s.stateMu.RLock()
	h := s.hidden
	s.stateMu.RUnlock()
	return h
}

func (s *session) SetHidden(hidden bool) {
	s.stateMu.Lock()
	wasHidden := s.hidden
	s.hidden = hidden
	s.stateMu.Unlock()
	if wasHidden == hidden {
		return
	}
	entityID := s.currentEntityID()
	if entityID == 0 {
		return
	}
	if hidden {
		s.broadcastToPeers(encodeRemoveEntity(byte(entityID)))
	} else {
		if snap, ok := s.entitySnapshot(); ok {
			s.broadcastToPeers(encodeAddEntity(byte(entityID), s.currentUsername(), snap.Pos, snap.Yaw, snap.Pitch))
		}
	}
}

func (s *session) IsMuted() bool {
	s.stateMu.RLock()
	m := s.muted
	s.stateMu.RUnlock()
	return m
}

func (s *session) SetMuted(muted bool) {
	s.stateMu.Lock()
	s.muted = muted
	s.stateMu.Unlock()
}

func (s *session) IsFrozen() bool {
	s.stateMu.RLock()
	f := s.frozen
	s.stateMu.RUnlock()
	return f
}

func (s *session) SetFrozen(frozen bool) {
	s.stateMu.Lock()
	s.frozen = frozen
	s.stateMu.Unlock()
}

func (s *session) IsAfk() bool {
	s.stateMu.RLock()
	a := s.afk
	s.stateMu.RUnlock()
	return a
}

func (s *session) IP() string {
	return s.conn.RemoteAddr().String()
}

func (s *session) EntityID() uint32 {
	return s.currentEntityID()
}

func (s *session) Yaw() byte {
	if snap, ok := s.entitySnapshot(); ok {
		return snap.Yaw
	}
	return 0
}

func (s *session) Pitch() byte {
	if snap, ok := s.entitySnapshot(); ok {
		return snap.Pitch
	}
	return 0
}

func (s *session) SendBlockChange(x, y, z int, block byte) {
	_ = s.writePacket(encodeSetBlock(x, y, z, block))
}

func (s *session) SendRawPacket(data []byte) {
	_ = s.writePacket(data)
}

func (s *session) Respawn() {
	if s.worlds == nil {
		return
	}
	spawn := s.worlds.Spawn()
	s.teleportSelf(spawn.X, spawn.Y, spawn.Z, spawn.Yaw, spawn.Pitch)
}

func (s *session) AllowBuild() bool {
	s.stateMu.RLock()
	a := s.allowBuild
	s.stateMu.RUnlock()
	return a
}

func (s *session) SetAllowBuild(allowed bool) {
	s.stateMu.Lock()
	s.allowBuild = allowed
	s.stateMu.Unlock()
}

func (s *session) ChangeBlock(x, y, z int, block byte) bool {
	return s.applyBlockChange(x, y, z, block, true) == nil
}

func (s *session) RevertBlock(x, y, z int) {
	if s.worlds == nil {
		return
	}
	if block, ok := s.worlds.BlockAt(x, y, z); ok {
		s.SendBlockChange(x, y, z, block)
	}
}

func (s *session) MakeSelection(id byte, label string, minX, minY, minZ, maxX, maxY, maxZ int, r, g, b byte) bool {
	if !s.supportsExt(cpeExtSelectionCuboid) {
		return false
	}
	pkt := encodeMakeSelection(id, label, uint16(minX), uint16(minY), uint16(minZ), uint16(maxX), uint16(maxY), uint16(maxZ), int16(r), int16(g), int16(b), 255)
	return s.writePacket(pkt) == nil
}

func (s *session) ClearSelection(id byte) bool {
	if !s.supportsExt(cpeExtSelectionCuboid) {
		return false
	}
	return s.writePacket(encodeRemoveSelection(id)) == nil
}
