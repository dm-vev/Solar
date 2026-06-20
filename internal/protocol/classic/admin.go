// admin.go implements all SessionBackend methods.
//
// These methods provide the operations that chat commands and the
// plugin API use to interact with the session and the server:
//
//   - Identity: CurrentUsername, CurrentLocation, IsOperator, Translate
//   - World ops: ApplyBlockChange, TeleportSelf, SetSpawn, GenerateWorld
//   - Persistence: SaveState, PersistPlayerPolicy
//   - Moderation: KickPlayer, BanPlayer, MutePlayer, FreezePlayer,
//     ToggleAFK, ToggleHide (all operate on online players via room lookup)
//   - Level ops: GotoLevel, LoadLevel, UnloadLevel, ReloadCurrentLevel
//   - BlockDB: ChangesAt, ChangesBy, Count, Enabled, Clear, RevertBlock
//   - Level env: GetEnvColor, SetLevelEnvColor, GetWeather, SetLevelWeather,
//     GetLevelMOTD, SetLevelMOTD
//   - Info: PlayerDBLookup, ServerName, OnlinePlayerCount, ServerUptime
//   - Block defs: AddBlockDef, RemoveBlockDef, GetBlockDef, ListBlockDefs
//   - Whitelist: WhitelistAdd, WhitelistRemove, SetWhitelistEnabled
//
// findTarget is a shared helper that looks up an online player by name
// (case-insensitive) via the session room.

package classic

import (
	"fmt"
	"strings"
	"time"

	"github.com/solar-mc/solar/internal/blockdb"
	"github.com/solar-mc/solar/internal/blockdef"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/drawing"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/playerdb"
)

// SessionBackend exposes the session operations that command adapters need.
// This interface decouples the protocol session from the command layer:
// the command package depends only on its own interfaces, and the adapter
// implementation lives in the app package.

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

func (s *session) Translate(key string, args ...any) string {
	if s.i18n != nil {
		return s.i18n.Get(s.Language(), key, args...)
	}
	return key
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

// ─── TeleportService methods ───

func (s *session) SpawnPoint() (x, y, z int, yaw, pitch byte) {
	if s.worlds == nil {
		return 0, 0, 0, 0, 0
	}
	sp := s.worlds.Spawn()
	return sp.X, sp.Y, sp.Z, sp.Yaw, sp.Pitch
}

func (s *session) TeleportToPlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	s.saveLastPos()
	tx, ty, tz := target.Position()
	targetYaw, targetPitch := target.Yaw(), target.Pitch()
	return s.teleportSelf(tx, ty, tz, targetYaw, targetPitch)
}

func (s *session) SummonPlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	target.saveLastPos()
	mx, my, mz := s.Position()
	myaw, mpitch := s.Yaw(), s.Pitch()
	return target.teleportSelf(mx, my, mz, myaw, mpitch)
}

func (s *session) BackToLastPos() bool {
	if !s.lastTeleportValid {
		return false
	}
	x, y, z := s.lastTeleportPos[0], s.lastTeleportPos[1], s.lastTeleportPos[2]
	yaw, pitch := s.Yaw(), s.Pitch()
	s.lastTeleportValid = false
	return s.teleportSelf(x, y, z, yaw, pitch)
}

func (s *session) saveLastPos() {
	x, y, z := s.Position()
	s.lastTeleportPos = [3]int{x, y, z}
	s.lastTeleportValid = true
}

// ─── ChatService methods ───

func (s *session) MeAction(action string) {
	msg := "* " + s.currentUsername() + " " + action
	pkt := encodeMessage(selfID, msg)
	_ = s.writePacket(pkt)
	s.broadcastToPeers(pkt)
}

func (s *session) WhisperTo(targetName, msg string) bool {
	target, ok := s.findTarget(targetName)
	if !ok {
		return false
	}
	target.Message("&7[whisper] &e" + s.currentUsername() + "&7: &f" + msg)
	s.Message("&7[to " + targetName + "] &f" + msg)
	return true
}

func (s *session) IgnorePlayer(name string) (bool, bool) {
	if s.ignoredPlayers == nil {
		s.ignoredPlayers = make(map[string]bool)
	}
	key := strings.ToLower(name)
	_, ok := s.findTarget(name)
	if !ok {
		return false, false
	}
	s.ignoredPlayers[key] = !s.ignoredPlayers[key]
	return s.ignoredPlayers[key], true
}

func (s *session) isIgnoring(name string) bool {
	if s.ignoredPlayers == nil {
		return false
	}
	return s.ignoredPlayers[strings.ToLower(name)]
}

// ─── DrawService methods ───

func (s *session) StartSelection(markCount int, callback func(marks [][3]int)) bool {
	if markCount < 1 || markCount > 3 {
		return false
	}
	s.markState = &markSelection{
		marks: make([]markPos, markCount),
		callback: func(marks []markPos) {
			out := make([][3]int, len(marks))
			for i, m := range marks {
				out[i] = [3]int{m.X, m.Y, m.Z}
			}
			callback(out)
		},
	}
	return true
}

func (s *session) GetBlockAt(x, y, z int) (byte, bool) {
	if s.worlds == nil {
		return 0, false
	}
	return s.worlds.BlockAt(x, y, z)
}

func (s *session) PlaceBlock(x, y, z int, block byte) bool {
	if s.worlds == nil {
		return false
	}
	if !s.worlds.SetBlock(x, y, z, block) {
		return false
	}
	pkt := encodeSetBlock(x, y, z, block)
	_ = s.writePacket(pkt)
	s.broadcastToPeers(pkt)
	return true
}

func (s *session) LevelDims() (width, height, length int) {
	if s.worlds == nil {
		return 0, 0, 0
	}
	lvl := s.worlds.Current()
	return lvl.Width, lvl.Height, lvl.Length
}

func (s *session) CopyRegion(min, max [3]int) bool {
	if s.worlds == nil {
		return false
	}
	w := max[0] - min[0] + 1
	h := max[1] - min[1] + 1
	l := max[2] - min[2] + 1
	if w < 1 || h < 1 || l < 1 {
		return false
	}
	cs := drawing.NewCopyState(w, h, l)
	for y := 0; y < h; y++ {
		for z := 0; z < l; z++ {
			for x := 0; x < w; x++ {
				b, _ := s.worlds.BlockAt(min[0]+x, min[1]+y, min[2]+z)
				cs.Set(x, y, z, b)
			}
		}
	}
	s.clipboard = cs
	return true
}

func (s *session) HasClipboard() bool {
	return s.clipboard != nil
}

func (s *session) PasteAt(origin [3]int, pasteAir bool) int {
	if s.clipboard == nil {
		return 0
	}
	count := 0
	s.clipboard.Paste(origin[0], origin[1], origin[2], pasteAir, func(x, y, z int, block byte) {
		if s.PlaceBlock(x, y, z, block) {
			count++
		}
	})
	return count
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

func (s *session) MutePlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	target.SetMuted(true)
	return true
}

func (s *session) UnmutePlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	target.SetMuted(false)
	return true
}

func (s *session) FreezePlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	target.SetFrozen(true)
	return true
}

func (s *session) UnfreezePlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	target.SetFrozen(false)
	return true
}

func (s *session) ToggleAFK(name string) (bool, bool) {
	target, ok := s.findTarget(name)
	if !ok {
		return false, false
	}
	newAfk := !target.IsAfk()
	target.SetAfk(newAfk)
	return newAfk, true
}

func (s *session) ToggleHide(name string) (bool, bool) {
	target, ok := s.findTarget(name)
	if !ok {
		return false, false
	}
	newHidden := !target.IsHidden()
	target.SetHidden(newHidden)
	return newHidden, true
}

func (s *session) findTarget(name string) (*session, bool) {
	if s.room == nil {
		return nil, false
	}
	return s.room.FindByName(name)
}

// ─── Info service methods ───

func (s *session) PlayerDBLookup(name string) *playerdb.PlayerEntry {
	if s.playerDB == nil {
		return nil
	}
	return s.playerDB.Get(name)
}

func (s *session) ServerName() string { return s.serverName }
func (s *session) ServerMOTD() string { return s.motd }
func (s *session) OnlinePlayerCount() int {
	if s.players == nil {
		return 0
	}
	return s.players.Count()
}
func (s *session) MaxPlayersCount() int {
	return s.maxPlayers
}
func (s *session) LoadedLevelCount() int {
	if s.listLoadedLevels == nil {
		return 1
	}
	return len(s.listLoadedLevels())
}
func (s *session) ServerUptime() time.Duration {
	return time.Since(StartTime)
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
	s.saveLastPos()
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

// ─── BlockDB methods ───

func (s *session) BlockDBChangesAt(x, y, z int) []command.BlockDBEntry {
	if s.blockDB == nil {
		return nil
	}
	entries := s.blockDB.ChangesAt(x, y, z)
	return convertEntries(entries, s.nameConv)
}

func (s *session) BlockDBChangesBy(playerName string, since time.Time, max int) []command.BlockDBEntry {
	if s.blockDB == nil {
		return nil
	}
	var pid int32
	if s.nameConv != nil {
		pid = s.nameConv.Get(playerName)
	}
	entries := s.blockDB.ChangesBy(pid, since, time.Time{}, max)
	return convertEntries(entries, s.nameConv)
}

func (s *session) BlockDBCount() int64 {
	if s.blockDB == nil {
		return 0
	}
	return s.blockDB.Count()
}

func (s *session) BlockDBEnabled() bool {
	if s.blockDB == nil {
		return false
	}
	return s.blockDB.Enabled()
}

func (s *session) BlockDBSetEnabled(enabled bool) {
	if s.blockDB != nil {
		s.blockDB.SetEnabled(enabled)
	}
}

func (s *session) BlockDBClear() error {
	if s.blockDB == nil {
		return fmt.Errorf("blockdb not available")
	}
	return s.blockDB.Clear()
}

func (s *session) BlockDBRevertBlock(x, y, z int, block byte) bool {
	return s.applyBlockChange(x, y, z, block, true) == nil
}

func convertEntries(entries []plugin.BlockEntry, nc *blockdb.NameConverter) []command.BlockDBEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]command.BlockDBEntry, len(entries))
	for i, e := range entries {
		out[i] = command.BlockDBEntry{
			Time: e.Time,
			X:    e.X, Y: e.Y, Z: e.Z,
			OldBlock: e.OldBlock,
			NewBlock: e.NewBlock,
			Flags:    uint16(e.Flags),
		}
		out[i].PlayerName = fmt.Sprintf("ID:%d", e.PlayerID)
	}
	return out
}

// ─── LevelService methods ───

func (s *session) GotoLevel(name string) bool {
	if s.gotoLevel == nil {
		return false
	}
	return s.gotoLevel(s, name)
}

func (s *session) MainLevelName() string {
	if s.mainLevelName != nil {
		return s.mainLevelName()
	}
	return ""
}

func (s *session) LoadLevel(name string) bool {
	if s.loadLevel == nil {
		return false
	}
	return s.loadLevel(name)
}

func (s *session) UnloadLevel(name string) bool {
	if s.unloadLevel == nil {
		return false
	}
	return s.unloadLevel(name)
}

func (s *session) ReloadCurrentLevel() bool {
	if s.worlds == nil {
		return false
	}
	worldPath := s.worldPath
	if worldPath == "" {
		return false
	}
	return s.worlds.Load(worldPath) == nil
}

func (s *session) ListLoadedLevels() []string {
	if s.listLoadedLevels == nil {
		return nil
	}
	return s.listLoadedLevels()
}

func (s *session) ListLevelFiles() []string {
	if s.listLevelFiles == nil {
		return nil
	}
	return s.listLevelFiles()
}

func (s *session) CurrentPhysicsMode() int {
	// ponytail: physics mode is on pluginServer, not session. Return 0 for now.
	return 0
}

func (s *session) SetCurrentPhysicsMode(mode int) {
	// ponytail: physics mode is on pluginServer, not session. No-op for now.
}

// ─── LevelEnvService methods ───

func (s *session) GetEnvColor(slot int) (r, g, b byte, set bool) {
	if s.worlds == nil || slot < 0 || slot >= 5 {
		return 0, 0, 0, false
	}
	env := s.worlds.Current().Env
	c := env.Colors[slot]
	return c.R, c.G, c.B, c.Set
}

func (s *session) SetLevelEnvColor(slot int, r, g, b byte) {
	if s.worlds == nil || slot < 0 || slot >= 5 {
		return
	}
	lvl := s.worlds.Current()
	lvl.Env.Colors[slot] = world.EnvColor{R: r, G: g, B: b, Set: true}
	s.worlds.SetCurrent(lvl)
	s.broadcastToPeers(encodeEnvColor(byte(slot), int16(r), int16(g), int16(b)))
	_ = s.writePacket(encodeEnvColor(byte(slot), int16(r), int16(g), int16(b)))
}

func (s *session) GetWeather() int {
	if s.worlds == nil {
		return 0
	}
	return int(s.worlds.Current().Env.Weather)
}

func (s *session) SetLevelWeather(weather int) {
	if s.worlds == nil {
		return
	}
	lvl := s.worlds.Current()
	lvl.Env.Weather = byte(weather)
	s.worlds.SetCurrent(lvl)
	pkt := encodeEnvWeatherType(byte(weather))
	s.broadcastToPeers(pkt)
	_ = s.writePacket(pkt)
}

func (s *session) GetLevelMOTD() string {
	if s.worlds == nil {
		return ""
	}
	return s.worlds.Current().Env.MOTD
}

func (s *session) SetLevelMOTD(motd string) {
	if s.worlds == nil {
		return
	}
	lvl := s.worlds.Current()
	lvl.Env.MOTD = motd
	s.worlds.SetCurrent(lvl)
}
