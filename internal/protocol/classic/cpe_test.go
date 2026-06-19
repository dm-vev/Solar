package classic

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/entity"
)

func TestServeConnBroadcastsExtPlayerListJoinAndLeave(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()

	serverA, clientA := net.Pipe()
	doneA := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), serverA)
		close(doneA)
	}()

	loginWithCPE(t, clientA, "alice", true)

	serverB, clientB := net.Pipe()
	doneB := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), serverB)
		close(doneB)
	}()

	loginWithCPE(t, clientB, "bob", true)

	readJoinSet(t, clientA, 2, "bob")
	readJoinSet(t, clientB, 1, "alice")

	if err := clientB.Close(); err != nil {
		t.Fatalf("close bob client: %v", err)
	}
	<-doneB

	remove := make([]byte, 2)
	if _, err := io.ReadFull(clientA, remove); err != nil {
		t.Fatalf("read alice remove entity: %v", err)
	}
	if remove[0] != opcodeRemoveEntity || remove[1] != 2 {
		t.Fatalf("alice remove entity = %v, want opcode %d id 2", remove, opcodeRemoveEntity)
	}

	tabRemove := make([]byte, 3)
	if _, err := io.ReadFull(clientA, tabRemove); err != nil {
		t.Fatalf("read alice remove tab entry: %v", err)
	}
	if tabRemove[0] != opcodeExtRemovePlayerName || tabRemove[1] != 2 {
		t.Fatalf("alice remove tab entry = %v, want opcode %d id 2", tabRemove, opcodeExtRemovePlayerName)
	}

	if err := clientA.Close(); err != nil {
		t.Fatalf("close alice client: %v", err)
	}
	<-doneA
}

func TestServeConnUsesFastMapAndTwoWayPing(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()
	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	loginWithCPE(t, client, "tester", false)

	stream := readLevelStream(t, client, true)
	level := codec.worlds.Current()
	if len(stream.Begin) != 5 {
		t.Fatalf("fastmap begin size = %d, want 5", len(stream.Begin))
	}
	volume := int(stream.Begin[1])<<24 | int(stream.Begin[2])<<16 | int(stream.Begin[3])<<8 | int(stream.Begin[4])
	if volume != level.Volume() {
		t.Fatalf("fastmap level volume = %d, want %d", volume, level.Volume())
	}
	if len(stream.Payload) != 4+len(level.Blocks) {
		t.Fatalf("fastmap decompressed size = %d, want %d", len(stream.Payload), 4+len(level.Blocks))
	}
	if mapVolume := int(stream.Payload[0])<<24 | int(stream.Payload[1])<<16 |
		int(stream.Payload[2])<<8 | int(stream.Payload[3]); mapVolume != level.Volume() {
		t.Fatalf("fastmap map volume = %d, want %d", mapVolume, level.Volume())
	}
	if !bytes.Equal(stream.Payload[4:], level.Blocks) {
		t.Fatalf("fastmap blocks do not match world snapshot")
	}
	if stream.Finalise[0] != opcodeLevelFinalize {
		t.Fatalf("fastmap level finalise opcode = %d, want %d", stream.Finalise[0], opcodeLevelFinalize)
	}
	if stream.Teleport[0] != opcodeEntityTeleport || stream.Teleport[1] != 255 {
		t.Fatalf("fastmap teleport header = %v, want opcode 8 and self id 255", stream.Teleport[:2])
	}

	if _, err := client.Write(encodeTwoWayPing(false, 77)); err != nil {
		t.Fatalf("write two way ping: %v", err)
	}

	if err := client.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	echo := make([]byte, 4)
	if _, err := io.ReadFull(client, echo); err != nil {
		t.Fatalf("read two way ping echo: %v", err)
	}
	if echo[0] != opcodeTwoWayPing || echo[1] != 0 || binary.BigEndian.Uint16(echo[2:4]) != 77 {
		t.Fatalf("two way ping echo = %v, want opcode 43 false 77", echo)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func TestServeConnNegotiatesAllCPEExtensions(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()
	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	// Send CPE handshake
	if _, err := client.Write(encodeClientHandshake(7, "tester", 0x42)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	// Read server ExtInfo
	serverExtInfo := make([]byte, 67)
	if _, err := io.ReadFull(client, serverExtInfo); err != nil {
		t.Fatalf("read server ext info: %v", err)
	}
	if serverExtInfo[0] != opcodeExtInfo {
		t.Fatalf("server ext info opcode = %d, want %d", serverExtInfo[0], opcodeExtInfo)
	}
	extCount := int(binary.BigEndian.Uint16(serverExtInfo[65:67]))
	if extCount != len(serverExtensions) {
		t.Fatalf("server ext count = %d, want %d", extCount, len(serverExtensions))
	}

	// Read all server ExtEntry packets and verify they match
	for i := 0; i < extCount; i++ {
		entry := make([]byte, 69)
		if _, err := io.ReadFull(client, entry); err != nil {
			t.Fatalf("read server ext entry %d: %v", i, err)
		}
		if entry[0] != opcodeExtEntry {
			t.Fatalf("server ext entry %d opcode = %d, want %d", i, entry[0], opcodeExtEntry)
		}
		name := bytes.TrimRight(entry[1:65], " \x00")
		version := binary.BigEndian.Uint32(entry[65:69])
		if string(name) != serverExtensions[i].name {
			t.Fatalf("server ext entry %d name = %q, want %q", i, name, serverExtensions[i].name)
		}
		if version != serverExtensions[i].version {
			t.Fatalf("server ext entry %d version = %d, want %d", i, version, serverExtensions[i].version)
		}
	}

	// Client responds supporting all extensions
	if _, err := client.Write(encodeClientExtInfo("ClassiCube", len(serverExtensions))); err != nil {
		t.Fatalf("write client ext info: %v", err)
	}
	for _, ext := range serverExtensions {
		if _, err := client.Write(encodeClientExtEntry(ext.name, ext.version)); err != nil {
			t.Fatalf("write client ext entry %s: %v", ext.name, err)
		}
	}

	// Read MOTD
	if _, err := io.ReadFull(client, make([]byte, 131)); err != nil {
		t.Fatalf("read motd: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

// classiCubeExtensions lists all extensions ClassiCube client sends.
var classiCubeExtensions = []struct {
	name    string
	version uint32
}{
	{"ClickDistance", 1}, {"CustomBlocks", 1}, {"HeldBlock", 1}, {"EmoteFix", 1},
	{"TextHotKey", 1}, {"ExtPlayerList", 2}, {"EnvColors", 1}, {"SelectionCuboid", 1},
	{"BlockPermissions", 1}, {"ChangeModel", 1}, {"EnvMapAppearance", 2}, {"EnvWeatherType", 1},
	{"MessageTypes", 1}, {"HackControl", 1}, {"PlayerClick", 1}, {"FullCP437", 1},
	{"LongerMessages", 1}, {"BlockDefinitions", 1}, {"BlockDefinitionsExt", 2}, {"BulkBlockUpdate", 1},
	{"TextColors", 1}, {"EnvMapAspect", 2}, {"EntityProperty", 1}, {"ExtEntityPositions", 1},
	{"TwoWayPing", 1}, {"InventoryOrder", 1}, {"InstantMOTD", 1}, {"FastMap", 1},
	{"SetHotbar", 1}, {"SetSpawnpoint", 1}, {"VelocityControl", 1}, {"CustomParticles", 1},
	{"PluginMessages", 1}, {"ExtEntityTeleport", 1}, {"LightingMode", 1}, {"CinematicGui", 1},
	{"NotifyAction", 1}, {"ToggleBlockList", 1},
}

func TestServeConnClassiCubeFullFlow(t *testing.T) {
	t.Parallel()

	codec := newTestCodec()
	server, client := net.Pipe()
	done := make(chan struct{})

	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()

	// Send CPE handshake like ClassiCube
	if _, err := client.Write(encodeClientHandshake(7, "tester", 0x42)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	// Read server ExtInfo
	serverExtInfo := make([]byte, 67)
	if _, err := io.ReadFull(client, serverExtInfo); err != nil {
		t.Fatalf("read server ext info: %v", err)
	}
	serverExtCount := int(binary.BigEndian.Uint16(serverExtInfo[65:67]))

	// Read all server ExtEntry packets
	for i := 0; i < serverExtCount; i++ {
		entry := make([]byte, 69)
		if _, err := io.ReadFull(client, entry); err != nil {
			t.Fatalf("read server ext entry %d: %v", i, err)
		}
		// Verify server does NOT advertise ExtEntityPositions
		name := bytes.TrimRight(entry[1:65], " \x00")
		if string(name) == "ExtEntityPositions" {
			t.Fatalf("server must not advertise ExtEntityPositions")
		}
	}

	// Client sends all its extensions (including ExtEntityPositions)
	if _, err := client.Write(encodeClientExtInfo("ClassiCube", len(classiCubeExtensions))); err != nil {
		t.Fatalf("write client ext info: %v", err)
	}
	for _, ext := range classiCubeExtensions {
		if _, err := client.Write(encodeClientExtEntry(ext.name, ext.version)); err != nil {
			t.Fatalf("write client ext entry %s: %v", ext.name, err)
		}
	}

	// Read MOTD
	if _, err := io.ReadFull(client, make([]byte, 131)); err != nil {
		t.Fatalf("read motd: %v", err)
	}

	// Read level stream (FastMap since client supports it)
	stream := readLevelStream(t, client, true)
	if stream.Teleport[0] != opcodeEntityTeleport {
		t.Fatalf("initial teleport opcode = %d, want %d", stream.Teleport[0], opcodeEntityTeleport)
	}

	// Send position update with selfID=255 (standard)
	if _, err := client.Write(encodeClientEntityTeleport(255, entity.Position{X: 320, Y: 640, Z: 960}, 0, 0)); err != nil {
		t.Fatalf("write teleport self: %v", err)
	}

	// Send position update with a block ID (HeldBlock extension)
	// Client sends held block ID instead of 255 — server must treat as self
	if _, err := client.Write(encodeClientEntityTeleport(3, entity.Position{X: 352, Y: 640, Z: 960}, 0, 0)); err != nil {
		t.Fatalf("write teleport heldblock: %v", err)
	}

	// Wait for entity update to process
	time.Sleep(50 * time.Millisecond)

	// Connection should still be alive — send a ping
	if _, err := client.Write([]byte{opcodePing}); err != nil {
		t.Fatalf("write ping after teleport: %v", err)
	}

	// If connection is dead, close will fail or done will not fire
	if err := client.Close(); err != nil {
		t.Fatalf("close client: %v", err)
	}
	<-done
}

func loginWithCPE(t *testing.T, client net.Conn, username string, drainLevel bool, extensions ...string) {
	t.Helper()

	if len(extensions) == 0 {
		extensions = []string{cpeExtPlayerListName, cpeExtTwoWayPingName, cpeExtFastMapName}
	}

	if _, err := client.Write(encodeClientHandshake(7, username, 0x42)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	// Read server ExtInfo
	serverExtInfo := make([]byte, 67)
	if _, err := io.ReadFull(client, serverExtInfo); err != nil {
		t.Fatalf("read server ext info: %v", err)
	}
	if serverExtInfo[0] != opcodeExtInfo {
		t.Fatalf("server ext info opcode = %d, want %d", serverExtInfo[0], opcodeExtInfo)
	}
	serverExtCount := int(binary.BigEndian.Uint16(serverExtInfo[65:67]))

	// Read all server ExtEntry packets
	for i := 0; i < serverExtCount; i++ {
		entry := make([]byte, 69)
		if _, err := io.ReadFull(client, entry); err != nil {
			t.Fatalf("read server ext entry %d: %v", i, err)
		}
	}

	// Client responds with its supported extensions
	if _, err := client.Write(encodeClientExtInfo("ClassiCube", len(extensions))); err != nil {
		t.Fatalf("write client ext info: %v", err)
	}
	for _, name := range extensions {
		version := uint32(1)
		if name == cpeExtPlayerListName {
			version = 2
		}
		if _, err := client.Write(encodeClientExtEntry(name, version)); err != nil {
			t.Fatalf("write client ext entry %s: %v", name, err)
		}
	}

	if _, err := io.ReadFull(client, make([]byte, 131)); err != nil {
		t.Fatalf("read motd: %v", err)
	}
	if !drainLevel {
		return
	}
	_ = readLevelStream(t, client, hasExtension(extensions, cpeExtFastMapName))
}

func readJoinSet(t *testing.T, client net.Conn, wantID byte, wantName string) {
	t.Helper()

	packet := make([]byte, 74)
	if _, err := io.ReadFull(client, packet); err != nil {
		t.Fatalf("read join packet: %v", err)
	}
	if packet[0] != opcodeAddEntity {
		t.Fatalf("join packet opcode = %d, want %d", packet[0], opcodeAddEntity)
	}
	if packet[1] != wantID {
		t.Fatalf("join packet id = %d, want %d", packet[1], wantID)
	}
	if got := bytes.TrimRight(packet[2:66], " \x00"); string(got) != wantName {
		t.Fatalf("join packet name = %q, want %s", got, wantName)
	}

	tabPacket := make([]byte, 196)
	if _, err := io.ReadFull(client, tabPacket); err != nil {
		t.Fatalf("read tab packet: %v", err)
	}
	if tabPacket[0] != opcodeExtAddPlayerName {
		t.Fatalf("tab packet opcode = %d, want %d", tabPacket[0], opcodeExtAddPlayerName)
	}
	if tabPacket[1] != wantID {
		t.Fatalf("tab packet id = %d, want %d", tabPacket[1], wantID)
	}
	if got := bytes.TrimRight(tabPacket[2:66], " \x00"); string(got) != wantName {
		t.Fatalf("tab packet player name = %q, want %s", got, wantName)
	}
	if got := bytes.TrimRight(tabPacket[66:130], " \x00"); string(got) != wantName {
		t.Fatalf("tab packet list name = %q, want %s", got, wantName)
	}
	if got := bytes.TrimRight(tabPacket[130:194], " \x00"); string(got) != "Players" {
		t.Fatalf("tab packet group name = %q, want Players", got)
	}
	if tabPacket[194] != 0 {
		t.Fatalf("tab packet group rank = %d, want 0", tabPacket[194])
	}
}

func encodeClientExtInfo(appName string, count int) []byte {
	packet := make([]byte, 67)
	packet[0] = opcodeExtInfo
	writeFixedString(packet[1:65], appName)
	binary.BigEndian.PutUint16(packet[65:67], uint16(count))
	return packet
}

func encodeClientExtEntry(name string, version uint32) []byte {
	packet := make([]byte, 69)
	packet[0] = opcodeExtEntry
	writeFixedString(packet[1:65], name)
	binary.BigEndian.PutUint32(packet[65:69], version)
	return packet
}

func hasExtension(extensions []string, name string) bool {
	for _, extension := range extensions {
		if extension == name {
			return true
		}
	}
	return false
}
