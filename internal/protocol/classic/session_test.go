package classic

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin/playerdb"
)

func TestServeConnMatchesClassiCubeFlow(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		version       byte
		handshakeTail byte
		wantRespSize  int
	}{
		{name: "classic-0.0.19", version: 5, handshakeTail: opcodePing, wantRespSize: 130},
		{name: "classic-0.30", version: 7, handshakeTail: 0, wantRespSize: 131},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			codec := newTestCodec()
			server, client := net.Pipe()
			done := make(chan struct{})

			go func() {
				codec.ServeConn(context.Background(), server)
				close(done)
			}()

			if _, err := client.Write(encodeClientHandshake(tc.version, "tester", tc.handshakeTail)); err != nil {
				t.Fatalf("write handshake: %v", err)
			}

			motd := make([]byte, tc.wantRespSize)
			if _, err := io.ReadFull(client, motd); err != nil {
				t.Fatalf("read motd: %v", err)
			}
			if motd[0] != opcodeHandshake {
				t.Fatalf("motd opcode = %d, want %d", motd[0], opcodeHandshake)
			}
			if motd[1] != tc.version {
				t.Fatalf("motd version = %d, want %d", motd[1], tc.version)
			}
			if got := bytes.TrimRight(motd[2:66], " \x00"); string(got) != "Solar" {
				t.Fatalf("server name = %q, want Solar", got)
			}
			if got := bytes.TrimRight(motd[66:130], " \x00"); string(got) != "CLI-only classic server" {
				t.Fatalf("motd = %q, want CLI-only classic server", got)
			}
			if tc.wantRespSize == 131 && motd[130] != 0 {
				t.Fatalf("user type = %d, want 0", motd[130])
			}

			stream := readLevelStream(t, client, false)
			level := codec.worlds.Current()
			if len(stream.Payload) != 4+len(level.Blocks) {
				t.Fatalf("decompressed size = %d, want %d", len(stream.Payload), 4+len(level.Blocks))
			}
			volume := int(stream.Payload[0])<<24 | int(stream.Payload[1])<<16 | int(stream.Payload[2])<<8 | int(stream.Payload[3])
			if volume != level.Volume() {
				t.Fatalf("map volume = %d, want %d", volume, level.Volume())
			}
			if !bytes.Equal(stream.Payload[4:], level.Blocks) {
				t.Fatalf("block payload does not match world snapshot")
			}
			if stream.Finalise[0] != opcodeLevelFinalize {
				t.Fatalf("level finalise opcode = %d, want %d", stream.Finalise[0], opcodeLevelFinalize)
			}
			if stream.Finalise[1] != byte(level.Width>>8) || stream.Finalise[2] != byte(level.Width) ||
				stream.Finalise[3] != byte(level.Height>>8) || stream.Finalise[4] != byte(level.Height) ||
				stream.Finalise[5] != byte(level.Length>>8) || stream.Finalise[6] != byte(level.Length) {
				t.Fatalf("level finalise dims = %v, want %dx%dx%d", stream.Finalise[1:], level.Width, level.Height, level.Length)
			}
			if stream.Teleport[0] != opcodeEntityTeleport || stream.Teleport[1] != 255 {
				t.Fatalf("teleport header = %v, want opcode 8 and self id 255", stream.Teleport[:2])
			}
			if stream.Teleport[2] != byte(level.Spawn.X*32>>8) || stream.Teleport[3] != byte(level.Spawn.X*32) ||
				stream.Teleport[4] != byte((level.Spawn.Y*32+51)>>8) || stream.Teleport[5] != byte(level.Spawn.Y*32+51) ||
				stream.Teleport[6] != byte(level.Spawn.Z*32>>8) || stream.Teleport[7] != byte(level.Spawn.Z*32) ||
				stream.Teleport[8] != level.Spawn.Yaw || stream.Teleport[9] != level.Spawn.Pitch {
				t.Fatalf("teleport packet = %v, want spawn %+v", stream.Teleport, level.Spawn)
			}

			if err := client.Close(); err != nil {
				t.Fatalf("close client: %v", err)
			}
			<-done
		})
	}
}

func TestServeConnIgnoresClientPingAfterLogin(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		newTestCodec().ServeConn(context.Background(), server)
		close(done)
	}()

	if _, err := client.Write(encodeClientHandshake(5, "tester", opcodePing)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	if _, err := io.ReadFull(client, make([]byte, 130)); err != nil {
		t.Fatalf("read motd: %v", err)
	}
	_ = readLevelStream(t, client, false)
	if err := client.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var buf [1]byte
	if n, err := client.Read(buf[:]); err == nil {
		t.Fatalf("read after login ping returned %d bytes, want timeout", n)
	} else {
		var ne net.Error
		if !errors.As(err, &ne) || !ne.Timeout() {
			t.Fatalf("read after login ping error = %v, want timeout", err)
		}
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnKicksInvalidUsername(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		newTestCodec().ServeConn(context.Background(), server)
		close(done)
	}()

	if _, err := client.Write(encodeClientHandshake(7, "", 0)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	kick := make([]byte, 65)
	if _, err := io.ReadFull(client, kick); err != nil {
		t.Fatalf("read kick: %v", err)
	}
	if kick[0] != opcodeKick {
		t.Fatalf("kick opcode = %d, want %d", kick[0], opcodeKick)
	}
	if got := bytes.TrimRight(kick[1:65], " \x00"); string(got) != "invalid username" {
		t.Fatalf("kick message = %q, want invalid username", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnKicksUsernameWithControlCharacters(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		newTestCodec().ServeConn(context.Background(), server)
		close(done)
	}()

	if _, err := client.Write(encodeClientHandshake(7, "bad\nname", 0)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	kick := make([]byte, 65)
	if _, err := io.ReadFull(client, kick); err != nil {
		t.Fatalf("read kick: %v", err)
	}
	if kick[0] != opcodeKick {
		t.Fatalf("kick opcode = %d, want %d", kick[0], opcodeKick)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnUsesWorldSnapshot(t *testing.T) {
	t.Parallel()

	worlds := world.NewManager()
	players := player.NewRegistry()
	players.AddOperators("tester")
	entities := entity.NewManager()
	worlds.SetCurrent(world.Level{
		Name:   "arena",
		Width:  2,
		Height: 3,
		Length: 4,
		Blocks: []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
		Spawn: world.Spawn{
			X:     64,
			Y:     96,
			Z:     128,
			Yaw:   17,
			Pitch: 29,
		},
	})

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		codec := NewCodec("Solar", "CLI-only classic server", worlds, players, entities, command.NewRegistry())
		codec.SetCommandContextBuilder(testBuildContext)
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	if _, err := client.Write(encodeClientHandshake(7, "tester", 0)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	if _, err := io.ReadFull(client, make([]byte, 131)); err != nil {
		t.Fatalf("read motd: %v", err)
	}
	stream := readLevelStream(t, client, false)
	level := worlds.Current()
	if len(stream.Payload) != 4+len(level.Blocks) {
		t.Fatalf("decompressed size = %d, want %d", len(stream.Payload), 4+len(level.Blocks))
	}
	volume := int(stream.Payload[0])<<24 | int(stream.Payload[1])<<16 | int(stream.Payload[2])<<8 | int(stream.Payload[3])
	if volume != level.Volume() {
		t.Fatalf("map volume = %d, want %d", volume, level.Volume())
	}
	if !bytes.Equal(stream.Payload[4:], level.Blocks) {
		t.Fatalf("block payload does not match world snapshot")
	}
	if stream.Finalise[1] != 0 || stream.Finalise[2] != 2 ||
		stream.Finalise[3] != 0 || stream.Finalise[4] != 3 ||
		stream.Finalise[5] != 0 || stream.Finalise[6] != 4 {
		t.Fatalf("level finalise dims = %v, want 2x3x4", stream.Finalise[1:])
	}
	wantX := level.Spawn.X * 32
	wantY := level.Spawn.Y*32 + 51
	wantZ := level.Spawn.Z * 32
	if stream.Teleport[2] != byte(wantX>>8) || stream.Teleport[3] != byte(wantX) ||
		stream.Teleport[4] != byte(wantY>>8) || stream.Teleport[5] != byte(wantY) ||
		stream.Teleport[6] != byte(wantZ>>8) || stream.Teleport[7] != byte(wantZ) {
		t.Fatalf("teleport position = %v, want %d,%d,%d (raw %d,%d,%d)", stream.Teleport[2:8],
			level.Spawn.X, level.Spawn.Y, level.Spawn.Z, wantX, wantY, wantZ)
	}
	if stream.Teleport[8] != 17 || stream.Teleport[9] != 29 {
		t.Fatalf("teleport rotation = %v, want 17,29", stream.Teleport[8:10])
	}

	if player, ok := players.Get("tester"); !ok {
		t.Fatalf("player not tracked after login")
	} else if !player.Spawned {
		t.Fatalf("player not marked spawned")
	} else if player.EntityID != 1 {
		t.Fatalf("player entity id = %d, want 1", player.EntityID)
	}
	if entity, ok := entities.Get(1); !ok {
		t.Fatalf("entity not tracked after login")
	} else if entity.Name != "tester" {
		t.Fatalf("entity name = %q, want tester", entity.Name)
	} else if entity.Pos.X != 2048 || entity.Pos.Y != 3072 || entity.Pos.Z != 4096 {
		t.Fatalf("entity pos = %+v, want 2048,3072,4096 (wire units)", entity.Pos)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
	if players.Count() != 0 {
		t.Fatalf("players count = %d, want 0 after disconnect", players.Count())
	}
	if entities.Count() != 0 {
		t.Fatalf("entities count = %d, want 0 after disconnect", entities.Count())
	}
}

func encodeClientHandshake(version byte, username string, userType byte) []byte {
	packet := make([]byte, 131)
	packet[0] = opcodeHandshake
	packet[1] = version
	writeFixedString(packet[2:66], username)
	writeFixedString(packet[66:130], "password")
	packet[130] = userType
	return packet
}

func newTestCodec() *Codec {
	return newTestCodecWithOperators()
}

func newTestCodecWithOperators(names ...string) *Codec {
	players := player.NewRegistry()
	players.AddOperators(names...)

	codec := NewCodec(
		"Solar",
		"CLI-only classic server",
		world.NewManager(),
		players,
		entity.NewManager(),
		command.NewRegistry(),
	)
	codec.SetCommandContextBuilder(testBuildContext)
	return codec
}

// testBuildContext builds a command.Context for test sessions. It mirrors
// what app.buildCommandContext does but is self-contained for tests.
func testBuildContext(backend SessionBackend) command.Context {
	position, yaw, pitch := backend.CurrentLocation()
	return command.Context{
		Username: backend.CurrentUsername(),
		Position: command.Position{
			X: position.X,
			Y: position.Y,
			Z: position.Z,
		},
		Yaw:         yaw,
		Pitch:       pitch,
		Authority:   testAuthority{backend: backend},
		World:       testWorldSvc{backend: backend},
		Persistence: testPersistence{backend: backend},
		Moderation:  testModeration{backend: backend},
		Players:     testDirectory{backend: backend},
		Tr:          backend.Translate,
	}
}

type testAuthority struct{ backend SessionBackend }

func (a testAuthority) CanAdmin() bool { return a.backend.IsOperator() }

type testWorldSvc struct{ backend SessionBackend }

func (w testWorldSvc) SetBlock(x, y, z int, blockID byte) bool {
	return w.backend.ApplyBlockChange(x, y, z, blockID, true) == nil
}
func (w testWorldSvc) MovePlayer(x, y, z int, yaw, pitch byte) bool {
	return w.backend.TeleportSelf(x, y, z, yaw, pitch)
}
func (w testWorldSvc) SetSpawn(x, y, z int, yaw, pitch byte) bool {
	return w.backend.SetSpawn(world.Spawn{X: x, Y: y, Z: z, Yaw: yaw, Pitch: pitch})
}
func (w testWorldSvc) GenerateWorld(name, theme string, width, height, length int, seed string) bool {
	return w.backend.GenerateWorld(name, theme, width, height, length, seed)
}

type testPersistence struct{ backend SessionBackend }

func (p testPersistence) SaveState() bool { return p.backend.SaveState() }

type testModeration struct{ backend SessionBackend }

func (m testModeration) KickPlayer(name, reason string) bool {
	return m.backend.KickPlayer(name, reason)
}
func (m testModeration) BanPlayer(name, reason string) bool { return m.backend.BanPlayer(name, reason) }
func (m testModeration) UnbanPlayer(name string) bool       { return m.backend.UnbanPlayer(name) }
func (m testModeration) WhitelistEnabled() bool             { return m.backend.WhitelistEnabled() }
func (m testModeration) SetWhitelistEnabled(enabled bool) bool {
	return m.backend.SetWhitelistEnabled(enabled)
}
func (m testModeration) WhitelistAdd(name string) bool       { return m.backend.WhitelistAdd(name) }
func (m testModeration) WhitelistRemove(name string) bool    { return m.backend.WhitelistRemove(name) }
func (m testModeration) MutePlayer(name string) bool         { return m.backend.MutePlayer(name) }
func (m testModeration) UnmutePlayer(name string) bool       { return m.backend.UnmutePlayer(name) }
func (m testModeration) FreezePlayer(name string) bool       { return m.backend.FreezePlayer(name) }
func (m testModeration) UnfreezePlayer(name string) bool     { return m.backend.UnfreezePlayer(name) }
func (m testModeration) ToggleAFK(name string) (bool, bool)  { return m.backend.ToggleAFK(name) }
func (m testModeration) ToggleHide(name string) (bool, bool) { return m.backend.ToggleHide(name) }

func (m testModeration) SpawnPoint() (int, int, int, byte, byte) { return 0, 0, 0, 0, 0 }
func (m testModeration) TeleportToPlayer(name string) bool       { return true }
func (m testModeration) SummonPlayer(name string) bool           { return true }
func (m testModeration) BackToLastPos() bool                     { return false }
func (m testModeration) MeAction(action string)                  {}
func (m testModeration) WhisperTo(target, msg string) bool       { return true }
func (m testModeration) IgnorePlayer(name string) (bool, bool)   { return true, true }

func (m testModeration) StartSelection(markCount int, callback func([][3]int)) bool { return true }
func (m testModeration) GetBlockAt(x, y, z int) (byte, bool)                        { return 0, true }
func (m testModeration) PlaceBlock(x, y, z int, block byte) bool                    { return true }
func (m testModeration) LevelDims() (int, int, int)                                 { return 128, 64, 128 }

func (m testModeration) PlayerDBLookup(name string) *playerdb.PlayerEntry {
	return m.backend.PlayerDBLookup(name)
}
func (m testModeration) ServerName() string          { return m.backend.ServerName() }
func (m testModeration) ServerMOTD() string          { return m.backend.ServerMOTD() }
func (m testModeration) OnlinePlayerCount() int      { return m.backend.OnlinePlayerCount() }
func (m testModeration) MaxPlayersCount() int        { return m.backend.MaxPlayersCount() }
func (m testModeration) LoadedLevelCount() int       { return m.backend.LoadedLevelCount() }
func (m testModeration) ServerUptime() time.Duration { return m.backend.ServerUptime() }

type testDirectory struct{ backend SessionBackend }

func (d testDirectory) ListPlayers() []string     { return d.backend.OnlineNames() }
func (d testDirectory) ListWhitelisted() []string { return d.backend.WhitelistNames() }

func TestServeConnUpdatesSelfEntityPosition(t *testing.T) {
	t.Parallel()

	worlds := world.NewManager()
	players := player.NewRegistry()
	players.AddOperators("tester")
	entities := entity.NewManager()

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		codec := NewCodec("Solar", "CLI-only classic server", worlds, players, entities, command.NewRegistry())
		codec.SetCommandContextBuilder(testBuildContext)
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	loginAndDrain(t, client, 5, "tester", opcodePing)

	if _, err := client.Write(encodeClientEntityTeleport(selfID, entity.Position{X: 640, Y: 384, Z: 576}, 33, 44)); err != nil {
		t.Fatalf("write entity teleport: %v", err)
	}

	waitForEntity(t, entities, 1, entity.Position{X: 640, Y: 384, Z: 576}, 33, 44)

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnAppliesClientBlockUpdate(t *testing.T) {
	t.Parallel()

	worlds := world.NewManager()
	players := player.NewRegistry()
	entities := entity.NewManager()

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		codec := NewCodec("Solar", "CLI-only classic server", worlds, players, entities, command.NewRegistry())
		codec.SetCommandContextBuilder(testBuildContext)
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	loginAndDrain(t, client, 5, "tester", opcodePing)

	if _, err := client.Write(encodeClientSetBlock(0, 0, 0, true, 7)); err != nil {
		t.Fatalf("write set block: %v", err)
	}

	packet := make([]byte, 8)
	if _, err := io.ReadFull(client, packet); err != nil {
		t.Fatalf("read server block update: %v", err)
	}
	if packet[0] != opcodeSetBlock {
		t.Fatalf("block update opcode = %d, want %d", packet[0], opcodeSetBlock)
	}
	if packet[7] != 7 {
		t.Fatalf("block id = %d, want 7", packet[7])
	}

	if got, ok := worlds.BlockAt(0, 0, 0); !ok {
		t.Fatal("block not present after update")
	} else if got != 7 {
		t.Fatalf("block at origin = %d, want 7", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnExecutesChatCommand(t *testing.T) {
	t.Parallel()

	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		newTestCodec().ServeConn(context.Background(), server)
		close(done)
	}()

	loginAndDrain(t, client, 5, "tester", opcodePing)

	if _, err := client.Write(encodeClientMessage("/where")); err != nil {
		t.Fatalf("write chat command: %v", err)
	}

	reply := make([]byte, 66)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("read command reply: %v", err)
	}
	if reply[0] != opcodeMessage {
		t.Fatalf("reply opcode = %d, want %d", reply[0], opcodeMessage)
	}
	if reply[1] != selfID {
		t.Fatalf("reply type = %d, want %d", reply[1], selfID)
	}
	text := bytes.TrimRight(reply[2:66], " \x00")
	if !strings.Contains(string(text), "command.where.position") {
		t.Fatalf("reply text = %q, want command.where.position", text)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnBroadcastsJoinAndBlockChanges(t *testing.T) {
	t.Parallel()

	codec := newTestCodecWithOperators("tester")

	serverA, clientA := net.Pipe()
	doneA := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), serverA)
		close(doneA)
	}()

	loginAndDrain(t, clientA, 5, "alice", opcodePing)

	serverB, clientB := net.Pipe()
	doneB := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), serverB)
		close(doneB)
	}()

	aliceJoin := make(chan []byte, 1)
	go func() {
		packet := make([]byte, 74)
		_, err := io.ReadFull(clientA, packet)
		if err != nil {
			t.Errorf("read alice join packet: %v", err)
			aliceJoin <- nil
			return
		}
		aliceJoin <- packet
	}()

	loginAndDrain(t, clientB, 5, "bob", opcodePing)

	bobJoin := make([]byte, 74)
	if _, err := io.ReadFull(clientB, bobJoin); err != nil {
		t.Fatalf("read bob join packet: %v", err)
	}
	if bobJoin[0] != opcodeAddEntity || bobJoin[1] != 1 {
		t.Fatalf("bob join header = %v, want opcode 7 and id 1", bobJoin[:2])
	}
	if got := bytes.TrimRight(bobJoin[2:66], " \x00"); string(got) != "alice" {
		t.Fatalf("bob join name = %q, want alice", got)
	}

	alicePacket := <-aliceJoin
	if alicePacket == nil {
		t.Fatal("alice join packet was not received")
	}
	if alicePacket[0] != opcodeAddEntity || alicePacket[1] != 2 {
		t.Fatalf("alice join header = %v, want opcode 7 and id 2", alicePacket[:2])
	}
	if got := bytes.TrimRight(alicePacket[2:66], " \x00"); string(got) != "bob" {
		t.Fatalf("alice join name = %q, want bob", got)
	}

	bobBlock := make(chan []byte, 1)
	go func() {
		packet := make([]byte, 8)
		_, err := io.ReadFull(clientA, packet)
		if err != nil {
			t.Errorf("read alice block packet: %v", err)
			bobBlock <- nil
			return
		}
		bobBlock <- packet
	}()

	if _, err := clientB.Write(encodeClientSetBlock(0, 0, 0, true, 7)); err != nil {
		t.Fatalf("write set block: %v", err)
	}

	selfBlock := make([]byte, 8)
	if _, err := io.ReadFull(clientB, selfBlock); err != nil {
		t.Fatalf("read bob self block packet: %v", err)
	}
	if selfBlock[0] != opcodeSetBlock || selfBlock[7] != 7 {
		t.Fatalf("bob block packet = %v, want opcode 6 and block 7", selfBlock)
	}

	aliceBlock := <-bobBlock
	if aliceBlock == nil {
		t.Fatal("alice block packet was not received")
	}
	if aliceBlock[0] != opcodeSetBlock || aliceBlock[7] != 7 {
		t.Fatalf("alice block packet = %v, want opcode 6 and block 7", aliceBlock)
	}

	if err := clientA.Close(); err != nil {
		t.Fatalf("close alice client: %v", err)
	}
	if err := clientB.Close(); err != nil {
		t.Fatalf("close bob client: %v", err)
	}
	<-doneA
	<-doneB
}

func TestServeConnExecutesTeleportAndSetSpawnCommands(t *testing.T) {
	t.Parallel()

	worlds := world.NewManager()
	players := player.NewRegistry()
	players.AddOperators("tester")
	entities := entity.NewManager()

	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec := NewCodec("Solar", "CLI-only classic server", worlds, players, entities, command.NewRegistry())
		codec.SetCommandContextBuilder(testBuildContext)
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	loginAndDrain(t, client, 5, "tester", opcodePing)

	if _, err := client.Write(encodeClientMessage("/tp 24 32 40 11 22")); err != nil {
		t.Fatalf("write tp command: %v", err)
	}

	var reply []byte
	first := make([]byte, 1)
	if _, err := io.ReadFull(client, first); err != nil {
		t.Fatalf("read first tp packet byte: %v", err)
	}

	switch first[0] {
	case opcodeEntityTeleport:
		teleport := make([]byte, 9)
		if _, err := io.ReadFull(client, teleport); err != nil {
			t.Fatalf("read teleport packet: %v", err)
		}
		if teleport[0] != 1 {
			t.Fatalf("teleport packet id = %d, want 1", teleport[0])
		}
		waitForEntity(t, entities, 1, entity.Position{X: 768, Y: 1024, Z: 1280}, 11, 22)

		reply = make([]byte, 66)
		if _, err := io.ReadFull(client, reply); err != nil {
			t.Fatalf("read tp reply: %v", err)
		}
		if got := bytes.TrimRight(reply[2:66], " \x00"); !strings.Contains(string(got), "command.teleport.done") {
			t.Fatalf("tp reply = %q, want command.teleport.done key", got)
		}
	case opcodeMessage:
		reply = make([]byte, 65)
		if _, err := io.ReadFull(client, reply); err != nil {
			t.Fatalf("read tp reply: %v", err)
		}
		if got := bytes.TrimRight(reply[1:65], " \x00"); !strings.Contains(string(got), "command.teleport.done") {
			t.Fatalf("tp reply = %q, want command.teleport.done key", got)
		}

		if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
			t.Fatalf("set read deadline: %v", err)
		}
		teleport := make([]byte, 10)
		teleport[0] = opcodeEntityTeleport
		if _, err := io.ReadFull(client, teleport[1:]); err != nil {
			t.Fatalf("read teleport packet: %v", err)
		}
		if teleport[0] != opcodeEntityTeleport {
			t.Fatalf("teleport packet opcode = %d, want %d", teleport[0], opcodeEntityTeleport)
		}
		if teleport[1] != 1 {
			t.Fatalf("teleport packet id = %d, want 1", teleport[1])
		}
		waitForEntity(t, entities, 1, entity.Position{X: 768, Y: 1024, Z: 1280}, 11, 22)
	default:
		t.Fatalf("first tp packet opcode = %d, want teleport or message", first[0])
	}

	if _, err := client.Write(encodeClientMessage("/setspawn 1 2 3 4 5")); err != nil {
		t.Fatalf("write setspawn command: %v", err)
	}
	reply = make([]byte, 66)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("read setspawn reply: %v", err)
	}
	if got := bytes.TrimRight(reply[2:66], " \x00"); !strings.Contains(string(got), "command.setspawn.done") {
		t.Fatalf("setspawn reply = %q, want command.setspawn.done", got)
	}
	if got := worlds.Current().Spawn; got.X != 1 || got.Y != 2 || got.Z != 3 || got.Yaw != 4 || got.Pitch != 5 {
		t.Fatalf("world spawn = %+v, want 1 2 3 4 5", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnKicksPlayerByName(t *testing.T) {
	t.Parallel()

	codec := newTestCodecWithOperators("alice")

	serverA, clientA := net.Pipe()
	doneA := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), serverA)
		close(doneA)
	}()
	loginAndDrain(t, clientA, 5, "alice", opcodePing)

	serverB, clientB := net.Pipe()
	doneB := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), serverB)
		close(doneB)
	}()
	loginAndDrain(t, clientB, 5, "bob", opcodePing)

	if _, err := clientA.Write(encodeClientMessage("/kick bob testing")); err != nil {
		t.Fatalf("write kick command: %v", err)
	}

	if err := clientB.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var header [1]byte
	if _, err := io.ReadFull(clientB, header[:]); err != nil {
		t.Fatalf("read packet header: %v", err)
	}
	kick := make([]byte, 65)
	switch header[0] {
	case 7:
		discard := make([]byte, 73)
		if _, err := io.ReadFull(clientB, discard); err != nil {
			t.Fatalf("read join packet: %v", err)
		}
		kick[0] = opcodeKick
		if _, err := io.ReadFull(clientB, kick[1:]); err != nil {
			t.Fatalf("read kick packet: %v", err)
		}
	case opcodeKick:
		kick[0] = opcodeKick
		if _, err := io.ReadFull(clientB, kick[1:]); err != nil {
			t.Fatalf("read kick packet: %v", err)
		}
	default:
		t.Fatalf("packet opcode = %d, want join or kick", header[0])
	}
	if kick[0] != opcodeKick {
		t.Fatalf("kick opcode = %d, want %d", kick[0], opcodeKick)
	}
	if got := bytes.TrimRight(kick[1:65], " \x00"); !strings.Contains(string(got), "testing") {
		t.Fatalf("kick message = %q, want testing reason", got)
	}

	if err := clientA.Close(); err != nil {
		t.Fatalf("close alice client: %v", err)
	}
	if err := clientB.Close(); err != nil {
		t.Fatalf("close bob client: %v", err)
	}
	<-doneA
	<-doneB
}

func TestServeConnRejectsBannedUsername(t *testing.T) {
	t.Parallel()

	worlds := world.NewManager()
	players := player.NewRegistry()
	players.Ban("tester", "nope")
	entities := entity.NewManager()

	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec := NewCodec("Solar", "CLI-only classic server", worlds, players, entities, command.NewRegistry())
		codec.SetCommandContextBuilder(testBuildContext)
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	if _, err := client.Write(encodeClientHandshake(7, "tester", 0)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	kick := make([]byte, 65)
	if _, err := io.ReadFull(client, kick); err != nil {
		t.Fatalf("read kick: %v", err)
	}
	if kick[0] != opcodeKick {
		t.Fatalf("kick opcode = %d, want %d", kick[0], opcodeKick)
	}
	if got := string(bytes.TrimRight(kick[1:65], " \x00")); got != "nope" {
		t.Fatalf("kick message = %q, want nope", got)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func loginAndDrain(t *testing.T, client net.Conn, version byte, username string, userType byte) {
	t.Helper()

	if _, err := client.Write(encodeClientHandshake(version, username, userType)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	motdSize := 130
	if version >= 6 {
		motdSize = 131
	}
	if _, err := io.ReadFull(client, make([]byte, motdSize)); err != nil {
		t.Fatalf("read motd: %v", err)
	}
	_ = readLevelStream(t, client, false)
}

func encodeClientEntityTeleport(entityID byte, pos entity.Position, yaw, pitch byte) []byte {
	packet := make([]byte, 10)
	packet[0] = opcodeEntityTeleport
	packet[1] = entityID
	binary.BigEndian.PutUint16(packet[2:4], uint16(pos.X))
	binary.BigEndian.PutUint16(packet[4:6], uint16(pos.Y+51))
	binary.BigEndian.PutUint16(packet[6:8], uint16(pos.Z))
	packet[8] = yaw
	packet[9] = pitch
	return packet
}

func encodeClientSetBlock(x, y, z int, place bool, blockID byte) []byte {
	packet := make([]byte, 9)
	packet[0] = opcodeSetBlockClient
	binary.BigEndian.PutUint16(packet[1:3], uint16(x))
	binary.BigEndian.PutUint16(packet[3:5], uint16(y))
	binary.BigEndian.PutUint16(packet[5:7], uint16(z))
	if place {
		packet[7] = 1
	}
	packet[8] = blockID
	return packet
}

func encodeClientMessage(text string) []byte {
	packet := make([]byte, 66)
	packet[0] = opcodeMessage
	packet[1] = 255
	writeFixedString(packet[2:66], text)
	return packet
}

func waitForEntity(
	t *testing.T,
	entities *entity.Manager,
	id uint32,
	wantPos entity.Position,
	wantYaw, wantPitch byte,
) {
	t.Helper()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		got, ok := entities.Get(id)
		if ok && got.Pos == wantPos && got.Yaw == wantYaw && got.Pitch == wantPitch {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}

	got, ok := entities.Get(id)
	if !ok {
		t.Fatal("entity not found after movement update")
	}
	t.Fatalf("entity state = %+v, want pos=%+v yaw=%d pitch=%d", got, wantPos, wantYaw, wantPitch)
}

func TestServeConnDropsIdleClientOnReadDeadline(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()
	codec.SetConnTimeouts(50*time.Millisecond, 0)

	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	if err := client.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	buf := make([]byte, 1)
	if _, err := client.Read(buf); err == nil {
		t.Fatal("expected read error from closed connection, got nil")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeConn did not return after read deadline expiry")
	}

	_ = client.Close()
}

func TestServeConnDropsClientOnWriteDeadline(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()
	codec.SetConnTimeouts(0, 50*time.Millisecond)

	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	if _, err := client.Write(encodeClientHandshake(7, "tester", 0)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	// Read the motd but then stop reading — the server's subsequent level
	// stream writes will block and hit the write deadline.
	motd := make([]byte, 131)
	if _, err := io.ReadFull(client, motd); err != nil {
		t.Fatalf("read motd: %v", err)
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ServeConn did not return after write deadline expiry")
	}

	_ = client.Close()
}

func TestCodecDefaultConnTimeouts(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()
	if codec.readDeadline != defaultReadDeadline {
		t.Fatalf("readDeadline = %v, want %v", codec.readDeadline, defaultReadDeadline)
	}
	if codec.writeDeadline != defaultWriteDeadline {
		t.Fatalf("writeDeadline = %v, want %v", codec.writeDeadline, defaultWriteDeadline)
	}

	codec.SetConnTimeouts(5*time.Second, 3*time.Second)
	if codec.readDeadline != 5*time.Second {
		t.Fatalf("readDeadline = %v, want 5s", codec.readDeadline)
	}
	if codec.writeDeadline != 3*time.Second {
		t.Fatalf("writeDeadline = %v, want 3s", codec.writeDeadline)
	}
}

func TestServeConnSaveFailsWithClosedWorkerPool(t *testing.T) {
	t.Parallel()

	codec := newTestCodecWithOperators("tester")
	pool := worker.NewPool(context.Background(), 1)
	pool.Close()
	codec.SetWorkerPool(pool)
	codec.SetPersistencePaths(
		filepath.Join(t.TempDir(), "world.swld"),
		filepath.Join(t.TempDir(), "policy.json"),
	)

	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	loginAndDrain(t, client, 5, "tester", opcodePing)

	if _, err := client.Write(encodeClientMessage("/save")); err != nil {
		t.Fatalf("write save command: %v", err)
	}

	reply := make([]byte, 66)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("read save reply: %v", err)
	}
	text := bytes.TrimRight(reply[2:66], " \x00")
	if !strings.Contains(string(text), "command.save.failed") {
		t.Fatalf("save reply = %q, want command.save.failed key", text)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnBanFailsWithClosedWorkerPool(t *testing.T) {
	t.Parallel()

	codec := newTestCodecWithOperators("alice")
	pool := worker.NewPool(context.Background(), 1)
	pool.Close()
	codec.SetWorkerPool(pool)
	codec.SetPersistencePaths("", filepath.Join(t.TempDir(), "policy.json"))

	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	loginAndDrain(t, client, 5, "alice", opcodePing)

	if _, err := client.Write(encodeClientMessage("/ban griefer")); err != nil {
		t.Fatalf("write ban command: %v", err)
	}

	reply := make([]byte, 66)
	if _, err := io.ReadFull(client, reply); err != nil {
		t.Fatalf("read ban reply: %v", err)
	}
	text := bytes.TrimRight(reply[2:66], " \x00")
	if !strings.Contains(string(text), "command.ban.failed") {
		t.Fatalf("ban reply = %q, want command.ban.failed key", text)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}
