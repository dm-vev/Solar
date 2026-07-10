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
	"path/filepath"
	"strings"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/ranks"
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
	manager := s.CurrentWorldManager()
	if manager == nil {
		return false
	}
	manager.SetSpawn(spawn)
	return true
}

// ─── TeleportService methods ───

func (s *session) SpawnPoint() (x, y, z int, yaw, pitch byte) {
	manager := s.CurrentWorldManager()
	if manager == nil {
		return 0, 0, 0, 0, 0
	}
	sp := manager.Spawn()
	return sp.X, sp.Y, sp.Z, sp.Yaw, sp.Pitch
}

func (s *session) TeleportToPlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	// Can't teleport to hidden players.
	if target.IsHidden() {
		return false
	}
	return s.teleportToSession(target)
}

func (s *session) RequestTeleport(name string) (command.TPAStatus, string) {
	target, ok, ambiguous := s.findTPATarget(name)
	if ambiguous {
		return command.TPAAmbiguous, name
	}
	if !ok || target.IsHidden() {
		return command.TPAPlayerNotFound, name
	}
	targetName := target.currentUsername()
	if target == s {
		return command.TPASelf, targetName
	}
	if target.isIgnoring(s.currentUsername()) {
		return command.TPARequestSent, targetName
	}
	if s.tpaRequests == nil {
		return command.TPAFailed, targetName
	}
	_, pending, busy := s.tpaRequests.add(s.currentUsername(), targetName, s.expireTeleportRequest)
	if pending != "" {
		if busy {
			return command.TPATargetBusy, pending
		}
		return command.TPAAlreadyPending, pending
	}
	if current, online := s.room.FindByName(targetName); !online || current != target {
		s.tpaRequests.cancel(s.currentUsername())
		return command.TPAPlayerNotFound, targetName
	}
	target.Message(target.Translate("command.tpa.received", s.currentUsername()))
	return command.TPARequestSent, targetName
}

func (s *session) RespondTeleport(accept bool) (command.TPAStatus, string) {
	if s.tpaRequests == nil {
		return command.TPAFailed, ""
	}
	req, ok := s.tpaRequests.take(s.currentUsername())
	if !ok {
		return command.TPANoPending, ""
	}
	if s.room == nil {
		return command.TPARequesterOffline, req.requester
	}
	requester, ok := s.room.FindByName(req.requester)
	if !ok {
		return command.TPARequesterOffline, req.requester
	}
	if !accept {
		requester.Message(requester.Translate("command.tpa.denied_by", s.currentUsername()))
		return command.TPADenied, req.requester
	}
	if !requester.teleportToSession(s) {
		return command.TPAFailed, req.requester
	}
	requester.Message(requester.Translate("command.tpa.accepted_by", s.currentUsername()))
	return command.TPAAccepted, req.requester
}

func (s *session) SummonPlayer(name string) bool {
	target, ok := s.findTarget(name)
	if !ok {
		return false
	}
	return target.teleportToSession(s)
}

func (s *session) BackToLastPos() bool {
	s.stateMu.RLock()
	valid := s.lastTeleportValid
	position := s.lastTeleportPos
	previousWorld := s.lastTeleportWorld
	s.stateMu.RUnlock()
	if !valid {
		return false
	}
	currentWorld := s.CurrentWorldManager()
	if previousWorld != nil && previousWorld != currentWorld {
		if err := s.prepareMapChange(previousWorld); err != nil {
			return false
		}
	}
	s.teleportMu.Lock()
	s.stateMu.RLock()
	unchanged := s.lastTeleportValid && s.lastTeleportPos == position && s.lastTeleportWorld == previousWorld
	s.stateMu.RUnlock()
	if !unchanged {
		s.teleportMu.Unlock()
		return false
	}
	changedMap := previousWorld != nil && previousWorld != s.CurrentWorldManager()
	prevName := ""
	if changedMap && currentWorld != nil {
		prevName = currentWorld.Current().Name
	}
	if previousWorld != nil && previousWorld != s.CurrentWorldManager() {
		if err := s.changeMapLocked(previousWorld); err != nil {
			s.teleportMu.Unlock()
			return false
		}
	}
	x, y, z := position[0], position[1], position[2]
	yaw, pitch := s.Yaw(), s.Pitch()
	entityID := s.currentEntityID()
	if s.entities == nil || entityID == 0 {
		s.teleportMu.Unlock()
		return false
	}
	pos := entityPosition(x, y, z)
	if !s.entities.SetLocation(entityID, pos, yaw, pitch) {
		s.teleportMu.Unlock()
		return false
	}
	if s.writePacket(encodeEntityTeleport(byte(entityID), pos, yaw, pitch)) != nil {
		s.teleportMu.Unlock()
		return false
	}
	s.stateMu.Lock()
	s.lastTeleportValid = false
	s.lastTeleportWorld = nil
	s.stateMu.Unlock()
	s.teleportMu.Unlock()
	if changedMap {
		s.finishMapChange(previousWorld.Current().Name, prevName)
	}
	return true
}

func (s *session) saveLastPosLocked() {
	x, y, z := s.Position()
	x, y, z = wireCoordsToBlocks(x, y, z)
	s.stateMu.Lock()
	s.lastTeleportPos = [3]int{x, y, z}
	s.lastTeleportWorld = s.worlds
	s.lastTeleportValid = true
	s.stateMu.Unlock()
}

func (s *session) findTPATarget(name string) (*session, bool, bool) {
	if target, ok := s.findTarget(name); ok {
		return target, true, false
	}
	if s.room == nil {
		return nil, false, false
	}
	key := tpaKey(name)
	var match *session
	for _, peer := range s.room.Snapshot() {
		if peer.IsHidden() || !strings.HasPrefix(tpaKey(peer.currentUsername()), key) {
			continue
		}
		if match != nil {
			return nil, false, true
		}
		match = peer
	}
	return match, match != nil, false
}

func (s *session) teleportToSession(target *session) bool {
	if target == nil {
		return false
	}
	origin := s.CurrentWorldManager()
	ox, oy, oz := s.Position()
	ox, oy, oz = wireCoordsToBlocks(ox, oy, oz)
	targetWorld := target.CurrentWorldManager()
	if origin != targetWorld {
		if err := s.prepareMapChange(targetWorld); err != nil {
			return false
		}
	}
	s.teleportMu.Lock()
	changedMap := origin != targetWorld
	if origin != targetWorld {
		if target.CurrentWorldManager() != targetWorld || s.changeMapLocked(targetWorld) != nil {
			s.teleportMu.Unlock()
			return false
		}
	}
	tx, ty, tz := target.Position()
	tx, ty, tz = wireCoordsToBlocks(tx, ty, tz)
	if !s.teleportSelfLocked(tx, ty, tz, target.Yaw(), target.Pitch()) {
		s.teleportMu.Unlock()
		return false
	}
	s.stateMu.Lock()
	s.lastTeleportPos = [3]int{ox, oy, oz}
	s.lastTeleportWorld = origin
	s.lastTeleportValid = true
	s.stateMu.Unlock()
	s.teleportMu.Unlock()
	if changedMap {
		prevName := ""
		if origin != nil {
			prevName = origin.Current().Name
		}
		s.finishMapChange(targetWorld.Current().Name, prevName)
	}
	return true
}

func (s *session) expireTeleportRequest(target string, id uint64) {
	if s.tpaRequests == nil {
		return
	}
	req, ok := s.tpaRequests.expire(target, id)
	if !ok || s.room == nil {
		return
	}
	if requester, ok := s.room.FindByName(req.requester); ok {
		requester.Message(requester.Translate("command.tpa.timed_out", req.target))
	}
	if recipient, ok := s.room.FindByName(req.target); ok {
		recipient.Message(recipient.Translate("command.tpa.incoming_timed_out", req.requester))
	}
}

func (s *session) cancelTeleportRequests() {
	if s.tpaRequests == nil || s.room == nil {
		return
	}
	name := s.currentUsername()
	for _, req := range s.tpaRequests.cancel(name) {
		other := req.requester
		if strings.EqualFold(other, name) {
			other = req.target
		}
		if peer, ok := s.room.FindByName(other); ok {
			peer.Message(peer.Translate("command.tpa.offline", name))
		}
	}
}

// ─── ChatService methods ───

func (s *session) MeAction(action string) {
	msg := s.Color() + "* " + s.currentUsername() + " " + action
	s.Message(msg)
	s.room.ForEachPeerExcept(s.currentEntityID(), func(peer *session) {
		if peer.CurrentWorldManager() != s.CurrentWorldManager() {
			return
		}
		if peer.isIgnoring(s.currentUsername()) {
			return
		}
		peer.Message(msg)
	})
}

func (s *session) WhisperTo(targetName, msg string) bool {
	target, ok := s.findTarget(targetName)
	if !ok {
		return false
	}
	// Check if target is ignoring the sender.
	if target.isIgnoring(s.currentUsername()) {
		return false
	}
	target.Message("&7[whisper] " + s.Color() + s.currentUsername() + "&7: &f" + msg)
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
	if markCount < 1 || markCount > 3 || callback == nil {
		return false
	}
	s.selectionMu.Lock()
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
	s.selectionMu.Unlock()
	return true
}

func (s *session) GetBlockAt(x, y, z int) (byte, bool) {
	manager := s.CurrentWorldManager()
	if manager == nil {
		return 0, false
	}
	return manager.BlockAt(x, y, z)
}

func (s *session) PlaceBlock(x, y, z int, block byte) bool {
	manager := s.CurrentWorldManager()
	if manager == nil {
		return false
	}
	// Check per-block permissions for both the new block (place) and
	// the old block (delete). A guest shouldn't be able to overwrite
	// TNT with stone any more than they can delete TNT directly.
	if !s.CanPlaceBlock(block) {
		return false
	}
	if old, ok := manager.BlockAt(x, y, z); ok && old != block {
		if !s.CanDeleteBlock(old) {
			return false
		}
	}
	if !manager.SetBlock(x, y, z, block) {
		return false
	}
	pkt := encodeSetBlock(x, y, z, block)
	_ = s.writePacket(pkt)
	s.broadcastToPeers(pkt)
	return true
}

func (s *session) LevelDims() (width, height, length int) {
	manager := s.CurrentWorldManager()
	if manager == nil {
		return 0, 0, 0
	}
	lvl := manager.Current()
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
	cs := blocks.NewCopyState(w, h, l)
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
		old, _ := s.worlds.BlockAt(x, y, z)
		if s.PlaceBlock(x, y, z, block) {
			s.RecordChange(x, y, z, old, block)
			count++
		}
	})
	return count
}

func (s *session) SetSpecialBlock(x, y, z int, entry command.SpecialBlockEntry) bool {
	manager := s.CurrentWorldManager()
	if manager == nil {
		return false
	}
	block := blocks.MBWhite
	if blocks.SpecialType(entry.Type) == blocks.SpecialPortal {
		block = blocks.PortalAir
	}
	if !s.CanPlaceBlock(block) {
		return false
	}
	if old, ok := manager.BlockAt(x, y, z); ok && old != block && !s.CanDeleteBlock(old) {
		return false
	}
	if !manager.SetBlockWithSpecial(x, y, z, block, &blocks.SpecialEntry{
		Type:        blocks.SpecialType(entry.Type),
		Message:     entry.Message,
		PortalDst:   [3]int{entry.PortalX, entry.PortalY, entry.PortalZ},
		PortalLevel: entry.PortalLevel,
		DoorBlock:   entry.DoorBlock,
	}) {
		return false
	}
	pkt := encodeSetBlock(x, y, z, block)
	_ = s.writePacket(pkt)
	s.broadcastToPeers(pkt)
	return true
}

// ─── Undo/redo methods ───

func (s *session) BeginBatch() {
	if s.undoStack == nil {
		s.undoStack = player.NewUndoStack(200)
	}
	s.batchChanges = nil
}

func (s *session) RecordChange(x, y, z int, oldBlock, newBlock byte) {
	s.batchChanges = append(s.batchChanges, player.BlockChange{X: x, Y: y, Z: z, Old: oldBlock, New: newBlock})
}

func (s *session) CommitBatch() {
	if s.undoStack != nil && len(s.batchChanges) > 0 {
		s.undoStack.Push(s.batchChanges)
	}
	s.batchChanges = nil
}

func (s *session) UndoBatch() []command.UndoChange {
	if s.undoStack == nil {
		return nil
	}
	changes := s.undoStack.Undo()
	if changes == nil {
		return nil
	}
	out := make([]command.UndoChange, len(changes))
	for i, c := range changes {
		out[i] = command.UndoChange{X: c.X, Y: c.Y, Z: c.Z, Old: c.Old, New: c.New}
	}
	return out
}

func (s *session) RedoBatch() []command.UndoChange {
	if s.undoStack == nil {
		return nil
	}
	changes := s.undoStack.Redo()
	if changes == nil {
		return nil
	}
	out := make([]command.UndoChange, len(changes))
	for i, c := range changes {
		out[i] = command.UndoChange{X: c.X, Y: c.Y, Z: c.Z, Old: c.Old, New: c.New}
	}
	return out
}

// ─── Rank methods ───

func (s *session) PlayerRank() int {
	perm := ranks.PermGuest
	if s.rankRegistry == nil {
		if s.players != nil && s.players.IsOperator(s.currentUsername()) {
			return ranks.PermOperator
		}
		return perm
	}
	perm = s.rankRegistry.GetPlayerRank(s.currentUsername())
	if perm < ranks.PermOperator && s.players != nil && s.players.IsOperator(s.currentUsername()) {
		return ranks.PermOperator
	}
	return perm
}

func (s *session) RankGetPlayer(name string) int {
	perm := ranks.PermGuest
	if s.rankRegistry == nil {
		if s.players != nil && s.players.IsOperator(name) {
			return ranks.PermOperator
		}
		return perm
	}
	perm = s.rankRegistry.GetPlayerRank(name)
	if perm < ranks.PermOperator && s.players != nil && s.players.IsOperator(name) {
		return ranks.PermOperator
	}
	return perm
}

func (s *session) RankGet(name string) *ranks.Rank {
	if s.rankRegistry == nil {
		return nil
	}
	return s.rankRegistry.Get(name)
}

func (s *session) RankGetByPerm(perm int) *ranks.Rank {
	if s.rankRegistry == nil {
		return nil
	}
	return s.rankRegistry.GetByPerm(perm)
}

func (s *session) RankAll() []*ranks.Rank {
	if s.rankRegistry == nil {
		return nil
	}
	return s.rankRegistry.All()
}

func (s *session) RankSetPlayer(name string, perm int) bool {
	if s.rankRegistry == nil {
		return false
	}
	return s.rankRegistry.SetPlayerRank(name, perm)
}

// ─── Draw limit + per-block permissions ───

func (s *session) DrawLimit() int {
	if s.rankRegistry == nil {
		return 1
	}
	perm := s.rankRegistry.GetPlayerRank(s.currentUsername())
	rank := s.rankRegistry.GetByPerm(perm)
	if rank == nil {
		return 1
	}
	return rank.DrawLimit
}

// blockPlacePerms holds per-block min rank for placing.
// ponytail: hardcoded defaults matching MCGalaxy's BlockPerms defaults.
// Configurable persistence can be added later.
var blockPlacePerms = [256]int{
	// Default: guest (0) for most blocks
	// Special blocks: operator (80)
	182: ranks.PermOperator,   // TNT_Small
	183: ranks.PermOperator,   // TNT_Big
	186: ranks.PermOperator,   // TNT_Nuke
	160: ranks.PermAdvBuilder, // Portal_Air
	161: ranks.PermAdvBuilder, // Portal_Water
	162: ranks.PermAdvBuilder, // Portal_Lava
	175: ranks.PermAdvBuilder, // Portal_Blue
	176: ranks.PermAdvBuilder, // Portal_Orange
	111: ranks.PermBuilder,    // Door_Log
	130: ranks.PermAdvBuilder, // MB_White
	131: ranks.PermAdvBuilder, // MB_Black
	132: ranks.PermAdvBuilder, // MB_Air
	133: ranks.PermAdvBuilder, // MB_Water
	134: ranks.PermAdvBuilder, // MB_Lava
	201: ranks.PermBuilder,    // Door_Log_air
}

func (s *session) CanPlaceBlock(block byte) bool {
	return s.PlayerRank() >= blockPlacePerms[block]
}

func (s *session) CanDeleteBlock(block byte) bool {
	return true // ponytail: all blocks deletable by guest (matching MCGalaxy defaults)
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
	s.teleportMu.Lock()
	defer s.teleportMu.Unlock()
	return s.teleportSelfLocked(x, y, z, yaw, pitch)
}

func (s *session) teleportSelfLocked(x, y, z int, yaw, pitch byte) bool {
	s.saveLastPosLocked()
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
	if s.saveServerState != nil {
		save := s.saveServerState
		if s.workers != nil {
			return s.workers.Submit(save)
		}
		go save()
		return true
	}
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
	if s.worldPath != "" && s.loadLevel != nil {
		mgr := world.NewManager()
		mgr.SetCurrent(w)
		path := filepath.Join(filepath.Dir(s.worldPath), name+filepath.Ext(s.worldPath))
		if err := mgr.Save(path); err != nil {
			s.logger.Debug("save generated level", "path", path, "error", err)
			return false
		}
		return s.loadLevel(name)
	}

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

func wireCoordsToBlocks(x, y, z int) (int, int, int) {
	return x / coordScale, y / coordScale, z / coordScale
}

// --- Block definition methods ---

func (s *session) AddBlockDef(def blocks.BlockDefinition) bool {
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

func (s *session) GetBlockDef(id byte) (blocks.BlockDefinition, bool) {
	if s.blockDefs == nil {
		return blocks.BlockDefinition{}, false
	}
	return s.blockDefs.Get(id)
}

func (s *session) ListBlockDefs() []blocks.BlockDefinition {
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

func convertEntries(entries []plugin.BlockEntry, nc *blocks.NameConverter) []command.BlockDBEntry {
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
	manager := s.CurrentWorldManager()
	if s.physicsMode == nil || manager == nil {
		return blocks.ModeOff
	}
	return s.physicsMode(manager)
}

func (s *session) SetCurrentPhysicsMode(mode int) {
	manager := s.CurrentWorldManager()
	if s.setPhysicsMode != nil && manager != nil {
		s.setPhysicsMode(manager, mode)
	}
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
