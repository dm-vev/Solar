package classic

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"testing"
	"time"
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

func loginWithCPE(t *testing.T, client net.Conn, username string, drainLevel bool, extensions ...string) {
	t.Helper()

	if len(extensions) == 0 {
		extensions = []string{cpeExtPlayerListName, cpeTwoWayPingName, cpeFastMapName}
	}

	if _, err := client.Write(encodeClientHandshake(7, username, 0x42)); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	serverExtInfo := make([]byte, 67)
	if _, err := io.ReadFull(client, serverExtInfo); err != nil {
		t.Fatalf("read server ext info: %v", err)
	}
	if serverExtInfo[0] != opcodeExtInfo {
		t.Fatalf("server ext info opcode = %d, want %d", serverExtInfo[0], opcodeExtInfo)
	}
	if got := bytes.TrimRight(serverExtInfo[1:65], " \x00"); string(got) != "Solar" {
		t.Fatalf("server ext info app name = %q, want Solar", got)
	}
	serverExtensions := []struct {
		name    string
		version uint32
	}{
		{name: cpeExtPlayerListName, version: cpeExtPlayerListVersion},
		{name: cpeTwoWayPingName, version: 1},
		{name: cpeFastMapName, version: 1},
	}
	if got := binary.BigEndian.Uint16(serverExtInfo[65:67]); got != uint16(len(serverExtensions)) {
		t.Fatalf("server ext count = %d, want %d", got, len(serverExtensions))
	}

	for _, expected := range serverExtensions {
		serverExtEntry := make([]byte, 69)
		if _, err := io.ReadFull(client, serverExtEntry); err != nil {
			t.Fatalf("read server ext entry: %v", err)
		}
		if serverExtEntry[0] != opcodeExtEntry {
			t.Fatalf("server ext entry opcode = %d, want %d", serverExtEntry[0], opcodeExtEntry)
		}
		if got := bytes.TrimRight(serverExtEntry[1:65], " \x00"); string(got) != expected.name {
			t.Fatalf("server ext name = %q, want %s", got, expected.name)
		}
		if got := binary.BigEndian.Uint32(serverExtEntry[65:69]); got != expected.version {
			t.Fatalf("server ext version = %d, want %d", got, expected.version)
		}
	}

	if _, err := client.Write(encodeClientExtInfo("ClassiCube", len(extensions))); err != nil {
		t.Fatalf("write client ext info: %v", err)
	}
	for _, name := range extensions {
		version := uint32(1)
		if name == cpeExtPlayerListName {
			version = cpeExtPlayerListVersion
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
	_ = readLevelStream(t, client, hasExtension(extensions, cpeFastMapName))
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
