package classic

import (
	"fmt"
	"strings"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/world"
)

func (s *session) commandContext() command.Context {
	position, yaw, pitch := s.currentLocation()

	return command.Context{
		Username: s.currentUsername(),
		Position: command.Position{
			X: position.X,
			Y: position.Y,
			Z: position.Z,
		},
		Yaw:         yaw,
		Pitch:       pitch,
		Authority:   sessionAuthority{s},
		World:       sessionWorld{s},
		Persistence: sessionPersistence{s},
		Moderation:  sessionModeration{s},
		Players:     sessionDirectory{s},
	}
}

type sessionAuthority struct{ session *session }

func (a sessionAuthority) CanAdmin() bool {
	if a.session.players == nil {
		return false
	}
	return a.session.players.IsOperator(a.session.currentUsername())
}

type sessionWorld struct{ session *session }

func (w sessionWorld) SetBlock(x, y, z int, blockID byte) bool {
	return w.session.applyBlockChange(x, y, z, blockID, true) == nil
}

func (w sessionWorld) MovePlayer(x, y, z int, yaw, pitch byte) bool {
	return w.session.teleportSelf(x, y, z, yaw, pitch)
}

func (w sessionWorld) SetSpawn(x, y, z int, yaw, pitch byte) bool {
	if w.session.worlds == nil {
		return false
	}
	w.session.worlds.SetSpawn(world.Spawn{X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch})
	return true
}

func (w sessionWorld) GenerateWorld(name, theme string, width, height, length int, seed string) bool {
	if w.session.worlds == nil {
		return false
	}
	return w.session.generateWorld(name, theme, width, height, length, seed)
}

type sessionPersistence struct{ session *session }

func (p sessionPersistence) SaveState() bool {
	return p.session.saveState()
}

type sessionModeration struct{ session *session }

func (m sessionModeration) KickPlayer(name, reason string) bool {
	return m.session.kickPlayer(name, reason)
}

func (m sessionModeration) BanPlayer(name, reason string) bool {
	return m.session.banPlayer(name, reason)
}

func (m sessionModeration) UnbanPlayer(name string) bool {
	return m.session.unbanPlayer(name)
}

func (m sessionModeration) WhitelistEnabled() bool {
	if m.session.players == nil {
		return false
	}
	return m.session.players.WhitelistEnabled()
}

func (m sessionModeration) SetWhitelistEnabled(enabled bool) bool {
	if m.session.players == nil {
		return false
	}
	changed := m.session.players.SetWhitelistEnabled(enabled)
	_ = m.session.persistPlayerPolicy()
	return changed
}

func (m sessionModeration) WhitelistAdd(name string) bool {
	if m.session.players == nil {
		return false
	}
	changed := m.session.players.WhitelistAdd(name)
	_ = m.session.persistPlayerPolicy()
	return changed
}

func (m sessionModeration) WhitelistRemove(name string) bool {
	if m.session.players == nil {
		return false
	}
	changed := m.session.players.WhitelistRemove(name)
	_ = m.session.persistPlayerPolicy()
	return changed
}

type sessionDirectory struct{ session *session }

func (d sessionDirectory) ListPlayers() []string {
	if d.session.players == nil {
		return nil
	}
	return d.session.players.OnlineNames()
}

func (d sessionDirectory) ListWhitelisted() []string {
	if d.session.players == nil {
		return nil
	}
	return d.session.players.WhitelistNames()
}

func (s *session) currentLocation() (world.Spawn, byte, byte) {
	entityID := s.currentEntityID()
	if s.entities != nil && entityID != 0 {
		if entitySnapshot, ok := s.entities.Get(entityID); ok {
			return world.Spawn{
				X:     entitySnapshot.Pos.X,
				Y:     entitySnapshot.Pos.Y,
				Z:     entitySnapshot.Pos.Z,
				Yaw:   entitySnapshot.Yaw,
				Pitch: entitySnapshot.Pitch,
			}, entitySnapshot.Yaw, entitySnapshot.Pitch
		}
	}
	if s.worlds != nil {
		level := s.worlds.Current()
		return level.Spawn, level.Spawn.Yaw, level.Spawn.Pitch
	}
	return world.Spawn{}, 0, 0
}

func (s *session) teleportSelf(x, y, z int, yaw, pitch byte) bool {
	entityID := s.currentEntityID()
	if s.entities == nil || entityID == 0 {
		return false
	}

	position := entityPosition(x, y, z)
	if !s.entities.SetLocation(entityID, position, yaw, pitch) {
		return false
	}

	packet := encodeEntityTeleport(byte(entityID), position, yaw, pitch)
	if err := s.writePacket(packet); err != nil {
		return false
	}
	s.broadcastToPeers(packet)
	return true
}

func (s *session) saveState() bool {
	if s.worldPath == "" && s.policyPath == "" {
		return false
	}

	worldPath := s.worldPath
	policyPath := s.policyPath
	worlds := s.worlds
	players := s.players
	logger := s.logger

	go func() {
		if worldPath != "" && worlds != nil {
			if err := worlds.Save(worldPath); err != nil {
				logger.Error("save world", "path", worldPath, "error", err)
			}
		}
		if policyPath != "" && players != nil {
			if err := players.SavePolicy(policyPath); err != nil {
				logger.Error("save player policy", "path", policyPath, "error", err)
			}
		}
	}()
	return true
}

func (s *session) persistPlayerPolicy() bool {
	if s.players == nil || s.policyPath == "" {
		return false
	}
	policyPath := s.policyPath
	players := s.players
	logger := s.logger

	go func() {
		if err := players.SavePolicy(policyPath); err != nil {
			logger.Error("save player policy", "path", policyPath, "error", err)
		}
	}()
	return true
}

func (s *session) kickPlayer(name, reason string) bool {
	if s.room == nil {
		return false
	}

	target, ok := s.room.FindByName(name)
	if !ok {
		return false
	}
	if reason == "" {
		reason = fmt.Sprintf("kicked by %s", s.currentUsername())
	}
	target.disconnect(reason)
	return true
}

func (s *session) banPlayer(name, reason string) bool {
	if s.players == nil {
		return false
	}

	if reason == "" {
		reason = fmt.Sprintf("banned by %s", s.currentUsername())
	}
	changed := s.players.Ban(name, reason)
	_ = s.persistPlayerPolicy()
	if s.room != nil {
		if target, ok := s.room.FindByName(name); ok {
			target.disconnect(reason)
		}
	}
	return changed || strings.TrimSpace(name) != ""
}

func (s *session) unbanPlayer(name string) bool {
	if s.players == nil {
		return false
	}
	changed := s.players.Unban(name)
	_ = s.persistPlayerPolicy()
	return changed
}

func (s *session) generateWorld(name, theme string, width, height, length int, seed string) bool {
	if s.worlds == nil {
		return false
	}
	gen, ok := generator.Find(theme)
	if !ok {
		s.logger.Debug("unknown generator", "theme", theme)
		return false
	}

	args, err := generator.ParseArgs(seed)
	if err != nil {
		s.logger.Debug("parse generator args", "error", err)
		return false
	}

	lvl, err := generator.Generate(gen, name, width, height, length, args)
	if err != nil {
		s.logger.Debug("generate world", "error", err)
		return false
	}

	w := world.FromGeneratorLevel(lvl)
	s.worlds.SetCurrent(w)

	// Send the new level to the issuer; other players currently keep their
	// existing level stream until they reconnect.
	if err := s.sendLevel(w); err != nil {
		s.logger.Debug("send generated level", "error", err)
		return false
	}
	return true
}

func entityPosition(x, y, z int) entity.Position {
	return entity.Position{X: x, Y: y, Z: z}
}
