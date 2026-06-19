package classic

import (
	"fmt"

	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/world"
)

// SessionBackend exposes the session operations that command adapters need.
// This interface decouples the protocol session from the command layer:
// the command package depends only on its own interfaces, and the adapter
// implementation lives in the app package.
type SessionBackend interface {
	CurrentUsername() string
	CurrentLocation() (world.Spawn, byte, byte)
	IsOperator() bool

	ApplyBlockChange(x, y, z int, blockID byte, echo bool) error
	TeleportSelf(x, y, z int, yaw, pitch byte) bool
	SetSpawn(spawn world.Spawn) bool
	GenerateWorld(name, theme string, width, height, length int, seed string) bool
	SaveState() bool
	PersistPlayerPolicy() bool

	KickPlayer(name, reason string) bool
	BanPlayer(name, reason string) bool
	UnbanPlayer(name string) bool
	WhitelistEnabled() bool
	WhitelistAdd(name string) bool
	WhitelistRemove(name string) bool
	SetWhitelistEnabled(enabled bool) bool

	OnlineNames() []string
	WhitelistNames() []string

	AddBlockDef(def blockdef.BlockDefinition) bool
	RemoveBlockDef(id byte) bool
	GetBlockDef(id byte) (blockdef.BlockDefinition, bool)
	ListBlockDefs() []blockdef.BlockDefinition
	FreeBlockID() byte
}

// --- SessionBackend implementation on *session ---

func (s *session) CurrentUsername() string {
	return s.currentUsername()
}

func (s *session) CurrentLocation() (world.Spawn, byte, byte) {
	return s.currentLocation()
}

func (s *session) IsOperator() bool {
	if s.players == nil {
		return false
	}
	return s.players.IsOperator(s.currentUsername())
}

func (s *session) ApplyBlockChange(x, y, z int, blockID byte, echo bool) error {
	return s.applyBlockChange(x, y, z, blockID, echo)
}

func (s *session) TeleportSelf(x, y, z int, yaw, pitch byte) bool {
	return s.teleportSelf(x, y, z, yaw, pitch)
}

func (s *session) SetSpawn(spawn world.Spawn) bool {
	if s.worlds == nil {
		return false
	}
	s.worlds.SetSpawn(spawn)
	return true
}

func (s *session) GenerateWorld(name, theme string, width, height, length int, seed string) bool {
	return s.generateWorld(name, theme, width, height, length, seed)
}

func (s *session) SaveState() bool {
	return s.saveState()
}

func (s *session) PersistPlayerPolicy() bool {
	return s.persistPlayerPolicy()
}

func (s *session) KickPlayer(name, reason string) bool {
	return s.kickPlayer(name, reason)
}

func (s *session) BanPlayer(name, reason string) bool {
	return s.banPlayer(name, reason)
}

func (s *session) UnbanPlayer(name string) bool {
	return s.unbanPlayer(name)
}

func (s *session) WhitelistEnabled() bool {
	if s.players == nil {
		return false
	}
	return s.players.WhitelistEnabled()
}

func (s *session) WhitelistAdd(name string) bool {
	if s.players == nil {
		return false
	}
	changed := s.players.WhitelistAdd(name)
	if !s.persistPlayerPolicy() {
		return false
	}
	return changed
}

func (s *session) WhitelistRemove(name string) bool {
	if s.players == nil {
		return false
	}
	changed := s.players.WhitelistRemove(name)
	if !s.persistPlayerPolicy() {
		return false
	}
	return changed
}

func (s *session) SetWhitelistEnabled(enabled bool) bool {
	if s.players == nil {
		return false
	}
	changed := s.players.SetWhitelistEnabled(enabled)
	if !s.persistPlayerPolicy() {
		return false
	}
	return changed
}

func (s *session) OnlineNames() []string {
	if s.players == nil {
		return nil
	}
	return s.players.OnlineNames()
}

func (s *session) WhitelistNames() []string {
	if s.players == nil {
		return nil
	}
	return s.players.WhitelistNames()
}

// --- Internal session methods (not part of SessionBackend) ---

func (s *session) currentLocation() (world.Spawn, byte, byte) {
	entityID := s.currentEntityID()
	if s.entities != nil && entityID != 0 {
		if entitySnapshot, ok := s.entities.Get(entityID); ok {
			return world.Spawn{
				X:     entitySnapshot.Pos.X / coordScale,
				Y:     entitySnapshot.Pos.Y / coordScale,
				Z:     entitySnapshot.Pos.Z / coordScale,
				Yaw:   entitySnapshot.Yaw,
				Pitch: entitySnapshot.Pitch,
			}, entitySnapshot.Yaw, entitySnapshot.Pitch
		}
	}
	if s.worlds != nil {
		spawn := s.worlds.Spawn()
		return spawn, spawn.Yaw, spawn.Pitch
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
	return s.writePacket(packet) == nil
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

	save := func() {
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
	}

	if s.workers != nil {
		if !s.workers.Submit(save) {
			logger.Error("failed to queue save", "world_path", worldPath, "policy_path", policyPath)
			return false
		}
	} else {
		go save()
	}
	return true
}

func (s *session) persistPlayerPolicy() bool {
	if s.players == nil || s.policyPath == "" {
		return true
	}
	policyPath := s.policyPath
	players := s.players
	logger := s.logger

	save := func() {
		if err := players.SavePolicy(policyPath); err != nil {
			logger.Error("save player policy", "path", policyPath, "error", err)
		}
	}

	if s.workers != nil {
		if !s.workers.Submit(save) {
			logger.Error("failed to queue player policy save", "path", policyPath)
			return false
		}
	} else {
		go save()
	}
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
	persisted := s.persistPlayerPolicy()
	if s.room != nil {
		if target, ok := s.room.FindByName(name); ok {
			target.disconnect(reason)
		}
	}
	if !persisted {
		return false
	}
	return changed
}

func (s *session) unbanPlayer(name string) bool {
	if s.players == nil {
		return false
	}
	changed := s.players.Unban(name)
	if !s.persistPlayerPolicy() {
		return false
	}
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

	if err := s.sendLevel(s.currentSupportsFastMap()); err != nil {
		s.logger.Debug("send generated level", "error", err)
		return false
	}
	return true
}

func entityPosition(x, y, z int) entity.Position {
	return entity.Position{X: x * coordScale, Y: y * coordScale, Z: z * coordScale}
}

// --- Block definition methods ---

func (s *session) AddBlockDef(def blockdef.BlockDefinition) bool {
	if s.blockDefs == nil {
		return false
	}
	s.blockDefs.Add(def)
	s.broadcastBlockDef(def)
	return true
}

func (s *session) RemoveBlockDef(id byte) bool {
	if s.blockDefs == nil {
		return false
	}
	if !s.blockDefs.Remove(id) {
		return false
	}
	s.broadcastUndefineBlock(id)
	return true
}

func (s *session) GetBlockDef(id byte) (blockdef.BlockDefinition, bool) {
	if s.blockDefs == nil {
		return blockdef.BlockDefinition{}, false
	}
	return s.blockDefs.Get(id)
}

func (s *session) ListBlockDefs() []blockdef.BlockDefinition {
	if s.blockDefs == nil {
		return nil
	}
	return s.blockDefs.All()
}

func (s *session) FreeBlockID() byte {
	if s.blockDefs == nil {
		return 0
	}
	return s.blockDefs.FreeID()
}
