// plugin_player.go implements the plugin.Player interface on *Session.
//
// This file contains the player-facing methods that plugins call:
//   - Chat: Message, SendCpeMessage
//   - Movement: Teleport, Position, Yaw, Pitch, Respawn
//   - Moderation: Kick, IsOperator, IsMuted, SetMuted, IsFrozen,
//     SetFrozen, IsAfk, SetAfk, IsHidden, SetHidden
//   - Building: SetBlock, ChangeBlock, RevertBlock, SendBlockChange,
//     AllowBuild, SetAllowBuild, MakeSelection, ClearSelection
//   - Identity: Name, EntityID, IP, Color, SetColor, Model, SetModel,
//     SetSkin, Language, SetLanguage
//   - CPE: SupportsCPE, CPE (returns the CPE interface)
//   - Combat: Kill (fires OnPlayerDying/Died, respawns)
//
// State changes (SetColor, SetModel, SetHidden, etc.) broadcast the
// appropriate packets to peers and fire plugin events.

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
	if plugin.OnSettingColor.HasHandlers() {
		c := color
		plugin.OnSettingColor.Fire(plugin.SettingColorData{Player: s, Color: &c})
		color = c
	}
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
	if plugin.OnSendingModel.HasHandlers() {
		m := model
		plugin.OnSendingModel.Fire(plugin.SendingModelData{Player: s, Model: &m})
		model = m
	}
	s.stateMu.Lock()
	s.model = model
	s.stateMu.Unlock()
	entityID := s.currentEntityID()
	if entityID != 0 {
		pkt := encodeChangeModel(byte(entityID), model)
		_ = s.writePacket(pkt)
		s.broadcastToPeers(pkt)
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
	if plugin.OnPlayerAction.HasHandlers() {
		action := "hide"
		if !hidden {
			action = "un-hide"
		}
		plugin.OnPlayerAction.Fire(plugin.PlayerActionData{
			Player:  s,
			Action:  action,
			Message: action,
		})
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

func (s *session) SetAfk(afk bool) {
	s.stateMu.Lock()
	was := s.afk
	s.afk = afk
	s.stateMu.Unlock()
	if was == afk {
		return
	}
	if plugin.OnPlayerAction.HasHandlers() {
		action := "afk"
		if !afk {
			action = "un-afk"
		}
		plugin.OnPlayerAction.Fire(plugin.PlayerActionData{
			Player:  s,
			Action:  action,
			Message: action,
		})
	}
}

// Kill triggers player death. Fires OnPlayerDying (cancelable); if not
// cancelled, fires OnPlayerDied and respawns. Returns false if cancelled.
func (s *session) Kill(cause byte) bool {
	if plugin.OnPlayerDying.HasHandlers() {
		ctx := plugin.OnPlayerDying.Fire(plugin.PlayerDyingData{Player: s, Cause: cause})
		if ctx.Cancelled() {
			return false
		}
	}
	cooldown := 0
	if plugin.OnPlayerDied.HasHandlers() {
		plugin.OnPlayerDied.Fire(plugin.PlayerDiedData{Player: s, Cause: cause, Cooldown: &cooldown})
	}
	if s.playerDB != nil {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			e.Deaths++
			s.playerDB.Save(e)
		}
	}
	s.Respawn()
	return true
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
	x, y, z, yaw, pitch := spawn.X, spawn.Y, spawn.Z, spawn.Yaw, spawn.Pitch
	if plugin.OnPlayerSpawn.HasHandlers() {
		plugin.OnPlayerSpawn.Fire(plugin.PlayerSpawnData{
			Player: s,
			X:      &x, Y: &y, Z: &z,
			Yaw: &yaw, Pitch: &pitch,
		})
	}
	s.teleportSelf(x, y, z, yaw, pitch)
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

func (s *session) Language() string {
	s.stateMu.RLock()
	l := s.lang
	s.stateMu.RUnlock()
	if l == "" {
		l = "en"
	}
	return l
}

func (s *session) SetLanguage(lang string) {
	s.stateMu.Lock()
	s.lang = lang
	s.stateMu.Unlock()
	// Persist to PlayerDB.
	if s.playerDB != nil {
		if e := s.playerDB.Get(s.currentUsername()); e != nil {
			if e.Data == nil {
				e.Data = make(map[string]string)
			}
			e.Data["lang"] = lang
			s.playerDB.Save(e)
		}
	}
}

// Tr returns a translated message for this player's language.
func (s *session) Tr(key string, args ...any) string {
	if s.i18n != nil {
		return s.i18n.Get(s.Language(), key, args...)
	}
	return key
}
