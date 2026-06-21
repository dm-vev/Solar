// handshake.go implements the login handshake and level streaming.
//
// The handshake flow:
//  1. Read 129-byte handshake payload (version, username, userType)
//  2. Validate username, check ban/whitelist
//  3. Register player in registry, create entity
//  4. If CPE requested: negotiate extensions (ExtInfo + ExtEntry pairs)
//  5. Send MOTD, block definitions, and level data stream
//  6. Teleport player to spawn, join room, fire OnPlayerConnect
//
// changeMap switches the player to a different level:
//  1. Fire OnJoiningLevel (cancelable)
//  2. Leave room (despawn from old level peers)
//  3. Swap world Manager
//  4. Send new level stream + teleport
//  5. Re-join room (spawn for new level peers)
//  6. Fire OnJoinedLevel
//
// sendLevelFrom encodes the level as a gzip or fastmap stream,
// sends it in chunks, then sends LevelFinalize and a teleport packet.
// CPE environment packets (env colors, weather, map properties) are
// sent after the teleport based on level.Env settings.

package classic

import (
	"fmt"
	"io"
	"time"

	"github.com/solar-mc/solar/internal/entity"
	pdb "github.com/solar-mc/solar/internal/playerdb"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

func (s *session) handleHandshake() error {
	payload := make([]byte, 129)
	if _, err := io.ReadFull(s.reader, payload); err != nil {
		return fmt.Errorf("read handshake payload: %w", err)
	}

	version := payload[0]
	username := readFixedString(payload[1:65])
	cpeRequested := false

	if version >= 6 {
		userType, err := s.reader.ReadByte()
		if err != nil {
			return fmt.Errorf("read handshake usertype: %w", err)
		}
		cpeRequested = userType == 0x42
	}

	if username == "" {
		return s.writeKick("invalid username")
	}
	if !s.validUsername(username) {
		return s.writeKick("invalid username")
	}
	if s.players != nil {
		if allowed, reason := s.players.CanJoin(username); !allowed {
			return s.writeKick(reason)
		}
	}
	if s.room != nil {
		if old, ok := s.room.FindByName(username); ok {
			old.disconnect("reconnected from another client")
		}
	}
	spawn := s.worlds.Spawn()
	entityID := uint32(0)
	tracked := false
	if s.entities != nil {
		id, ok := s.entities.Add(username, entity.Position{
			X: spawn.X * coordScale,
			Y: spawn.Y * coordScale,
			Z: spawn.Z * coordScale,
		})
		if ok {
			entityID = id
		}
	}
	if s.players != nil {
		tracked = s.players.Add(username, entityID)
	}
	s.setIdentity(username, entityID, tracked)

	// Record login in PlayerDB and capture login time for playtime tracking.
	if s.playerDB != nil {
		ip := ""
		if s.conn != nil && s.conn.RemoteAddr() != nil {
			ip = s.conn.RemoteAddr().String()
		}
		pdb.EnsureEntry(s.playerDB, username, ip)
	}
	s.loginTime = time.Now()

	// Assign BlockDB player ID.
	if s.nameConv != nil {
		s.playerDBID = s.nameConv.Get(username)
	}

	// Load persisted player props (color, model, frozen, muted, afk, allow_build).
	if s.players != nil {
		props := s.players.GetProps(username)
		s.stateMu.Lock()
		if props.Color != "" {
			s.color = props.Color
		}
		if props.Model != "" {
			s.model = props.Model
		}
		s.frozen = props.Frozen
		s.muted = props.Muted
		s.afk = props.AFK
		if props.AllowBuild != nil {
			s.allowBuild = *props.AllowBuild
		}
		s.stateMu.Unlock()
	}

	// Load player language from PlayerDB.
	if s.playerDB != nil {
		if e := s.playerDB.Get(username); e != nil {
			if lang, ok := e.Data["lang"]; ok && lang != "" {
				s.SetLanguage(lang)
			}
		}
	}

	// Player rank is resolved via ranks.Registry on demand — no need
	// to load it here. PlayerRank() reads from PlayerDB.Data["rank"]
	// through the registry at query time.

	// Resolve BlockDB for the main level and player's DB ID.
	if s.blockDBForLevel != nil && s.worlds != nil {
		s.blockDB = s.blockDBForLevel(s.worlds.Current().Name)
	}

	if plugin.OnPlayerStartConnecting.HasHandlers() {
		plugin.OnPlayerStartConnecting.Fire(plugin.PlayerStartConnectingData{Player: s})
	}

	if cpeRequested {
		if err := s.negotiateCPE(); err != nil {
			return fmt.Errorf("negotiate cpe: %w", err)
		}
	}

	if plugin.OnPlayerFinishConnecting.HasHandlers() {
		plugin.OnPlayerFinishConnecting.Fire(plugin.PlayerFinishConnectingData{Player: s})
	}

	motd := s.motd
	if plugin.OnGettingMotd.HasHandlers() {
		plugin.OnGettingMotd.Fire(plugin.GettingMotdData{Player: s, Motd: &motd})
	}
	if plugin.OnSendingMotd.HasHandlers() {
		plugin.OnSendingMotd.Fire(plugin.SendingMotdData{Player: s, Motd: &motd})
	}
	if err := s.writePacket(encodeMotd(version, s.serverName, motd)); err != nil {
		return err
	}
	if err := s.sendBlockDefinitions(); err != nil {
		return err
	}
	if err := s.sendLevel(s.currentSupportsFastMap()); err != nil {
		return err
	}
	s.markLoggedIn()
	s.joinRoom()

	if plugin.OnPlayerConnect.HasHandlers() {
		plugin.OnPlayerConnect.Fire(plugin.PlayerConnectData{Player: s})
	}
	return nil
}

func (s *session) validUsername(username string) bool {
	if len(username) < 1 || len(username) > 32 {
		return false
	}
	for _, r := range username {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func encodeMotd(version byte, serverName, motd string) []byte {
	size := 130
	if version >= 6 {
		size = 131
	}

	packet := make([]byte, size)
	packet[0] = opcodeHandshake
	packet[1] = version
	writeFixedString(packet[2:66], serverName)
	writeFixedString(packet[66:130], motd)
	if version >= 6 {
		packet[130] = 0
	}
	return packet
}

func (s *session) sendLevel(fastMap bool) error {
	return s.sendLevelFrom(s.worlds, fastMap)
}

// sendLevelFrom streams a level from the given Manager to the client:
// level begin + data chunks + finalize + teleport to spawn.
func (s *session) sendLevelFrom(mgr *world.Manager, fastMap bool) error {
	stream, err := mgr.LevelStream(fastMap)
	if err != nil {
		return err
	}
	level := stream.Level

	if err := s.writePacket(encodeLevelBegin(level.Volume(), fastMap)); err != nil {
		return err
	}

	chunks := encodeLevelDataPackets(stream)
	for _, chunk := range chunks {
		if err := s.writePacket(chunk); err != nil {
			return err
		}
	}

	if err := s.writePacket([]byte{
		opcodeLevelFinalize,
		byte(level.Width >> 8), byte(level.Width),
		byte(level.Height >> 8), byte(level.Height),
		byte(level.Length >> 8), byte(level.Length),
	}); err != nil {
		return err
	}

	if username := s.currentUsername(); s.players != nil && username != "" {
		s.players.MarkSpawned(username)
	}

	spawnX, spawnY, spawnZ := level.Spawn.X, level.Spawn.Y, level.Spawn.Z
	spawnYaw, spawnPitch := level.Spawn.Yaw, level.Spawn.Pitch
	if plugin.OnPlayerSpawn.HasHandlers() {
		plugin.OnPlayerSpawn.Fire(plugin.PlayerSpawnData{
			Player: s,
			X:      &spawnX, Y: &spawnY, Z: &spawnZ,
			Yaw: &spawnYaw, Pitch: &spawnPitch,
		})
	}

	teleportPkt := []byte{
		opcodeEntityTeleport,
		selfID,
		byte(spawnX * coordScale >> 8), byte(spawnX * coordScale),
		byte((spawnY*coordScale + eyeHeight) >> 8), byte(spawnY*coordScale + eyeHeight),
		byte(spawnZ * coordScale >> 8), byte(spawnZ * coordScale),
		spawnYaw,
		spawnPitch,
	}
	if err := s.writePacket(teleportPkt); err != nil {
		return err
	}

	// Send environment CPE packets based on level.Env.
	s.sendEnv(level.Env)

	if plugin.OnSentMap.HasHandlers() {
		plugin.OnSentMap.Fire(plugin.SentMapData{Player: s})
	}
	return nil
}

// changeMap switches the session's active world Manager, sends the new
// level stream, and re-syncs entity visibility across levels.
func (s *session) changeMap(mgr *world.Manager) error {
	levelName := mgr.Current().Name
	prevName := ""
	if s.worlds != nil {
		prevName = s.worlds.Current().Name
	}

	if plugin.OnJoiningLevel.HasHandlers() {
		ctx := plugin.OnJoiningLevel.Fire(plugin.JoiningLevelData{
			Player:    s,
			LevelName: levelName,
		})
		if ctx.Cancelled() {
			return nil
		}
	}

	// Leave room before swapping worlds — removes entity from old level peers.
	s.leaveRoom()

	s.stateMu.Lock()
	s.worlds = mgr
	s.stateMu.Unlock()

	if err := s.sendLevelFrom(mgr, s.currentSupportsFastMap()); err != nil {
		return err
	}

	spawn := mgr.Spawn()
	entityID := s.currentEntityID()
	if s.entities != nil && entityID != 0 {
		pos := entityPosition(spawn.X, spawn.Y, spawn.Z)
		s.entities.SetLocation(entityID, pos, spawn.Yaw, spawn.Pitch)
	}

	// Re-join room — spawns entity for new level peers and vice versa.
	s.markJoined(false)
	s.joinRoom()

	// Switch BlockDB to the new level.
	if s.blockDBForLevel != nil {
		s.blockDB = s.blockDBForLevel(levelName)
	}

	if s.entities != nil && entityID != 0 {
		s.broadcastToPeers(encodeEntityTeleport(byte(entityID),
			entityPosition(spawn.X, spawn.Y, spawn.Z), spawn.Yaw, spawn.Pitch))
	}

	if plugin.OnJoinedLevel.HasHandlers() {
		plugin.OnJoinedLevel.Fire(plugin.JoinedLevelData{
			Player:    s,
			LevelName: levelName,
			PrevLevel: prevName,
		})
	}
	return nil
}

func encodeLevelBegin(volume int, fastMap bool) []byte {
	if !fastMap {
		return []byte{opcodeLevelInitialize}
	}

	packet := make([]byte, 5)
	packet[0] = opcodeLevelInitialize
	packet[1] = byte(volume >> 24)
	packet[2] = byte(volume >> 16)
	packet[3] = byte(volume >> 8)
	packet[4] = byte(volume)
	return packet
}
