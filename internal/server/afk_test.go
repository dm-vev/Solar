package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/protocol/wire"
	"github.com/solar-mc/solar/internal/ranks"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin/playerdb"
)

// drainLogin writes a handshake, reads the motd + level stream + teleport,
// and returns. The world must be small to keep this fast.
func drainLogin(t *testing.T, client net.Conn, username string) {
	t.Helper()

	buf := make([]byte, 131)
	buf[0] = wire.OpcodeHandshake
	buf[1] = 7
	wire.WriteFixedString(buf[2:66], username)
	wire.WriteFixedString(buf[66:130], "password")
	if _, err := client.Write(buf); err != nil {
		t.Fatalf("write handshake: %v", err)
	}

	motd := make([]byte, 131)
	if _, err := io.ReadFull(client, motd); err != nil {
		t.Fatalf("read motd: %v", err)
	}

	for {
		var opcode [1]byte
		if _, err := io.ReadFull(client, opcode[:]); err != nil {
			t.Fatalf("read level opcode: %v", err)
		}
		switch opcode[0] {
		case wire.OpcodeLevelInitialize:
			continue
		case wire.OpcodeLevelData:
			chunk := make([]byte, 1027)
			if _, err := io.ReadFull(client, chunk); err != nil {
				t.Fatalf("read level chunk: %v", err)
			}
		case wire.OpcodeLevelFinalize:
			tail := make([]byte, 6)
			if _, err := io.ReadFull(client, tail); err != nil {
				t.Fatalf("read level finalize: %v", err)
			}
			teleport := make([]byte, 10)
			if _, err := io.ReadFull(client, teleport); err != nil {
				t.Fatalf("read teleport: %v", err)
			}
			return
		default:
			t.Fatalf("unexpected opcode %d during level stream", opcode[0])
		}
	}
}

// encodeRelPos builds a 5-byte relative-position packet (opcode 10).
func encodeRelPos(entityID byte, dx, dy, dz byte) []byte {
	return []byte{wire.OpcodeRelPos, entityID, dx, dy, dz}
}

// newAFKTestServer creates a Server with the given AFK config and a tiny
// world. Returns the server, codec, and player registry.
func newAFKTestServer(t *testing.T, afkCfg config.AFKConfig) (*Server, *classic.Codec, *player.Registry) {
	t.Helper()

	dir := t.TempDir()
	worlds := world.NewManager()
	worlds.SetCurrent(world.Level{
		Name:   "test",
		Width:  2,
		Height: 3,
		Length: 4,
		Blocks: make([]byte, 24),
		Spawn:  world.Spawn{X: 1, Y: 2, Z: 2},
	})
	players := player.NewRegistry()
	entities := entity.NewManager()
	listener := network.NewListener(":0")
	codec := classic.NewCodec("Solar", "Test", worlds, players, entities, nil)

	cfg := config.Config{
		ListenAddress: ":0",
		DataDir:       dir,
		MaxPlayers:    8,
		Name:          "Solar",
		MOTD:          "Test",
		AFK:           afkCfg,
		Storage: config.StorageConfig{
			Backend:       "local",
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
			BlockDefsDir:  "blockdefs",
		},
	}

	store := storage.NewLocalStore(dir)
	srv := New(cfg, listener, codec, worlds, players, entities, store,
		worker.NewPool(context.Background(), 1), testLogger)
	return srv, codec, players
}

// connectPlayer connects a player to the codec, logs in, and returns
// the client conn and a done channel.
func connectPlayer(t *testing.T, codec *classic.Codec, username string) (net.Conn, chan struct{}) {
	t.Helper()
	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()
	drainLogin(t, client, username)
	return client, done
}

// connectPlayerWithRank connects a player with the given rank via a
// mock PlayerDB + rank registry.
func connectPlayerWithRank(t *testing.T, codec *classic.Codec, username string, rank int) (net.Conn, chan struct{}) {
	t.Helper()
	rr := ranks.NewRegistry()
	rr.SetPlayerDB(&mockPlayerDB{
		entries: map[string]*playerdb.PlayerEntry{
			username: {Name: username, Data: map[string]string{"rank": fmt.Sprintf("%d", rank)}},
		},
	})
	codec.SetRankRegistry(rr)
	return connectPlayer(t, codec, username)
}

// mockPlayerDB is a minimal in-memory PlayerDB for rank-exempt tests.
type mockPlayerDB struct {
	mu      sync.Mutex
	entries map[string]*playerdb.PlayerEntry
}

func (m *mockPlayerDB) Get(name string) *playerdb.PlayerEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.entries[name]
}
func (m *mockPlayerDB) Save(entry *playerdb.PlayerEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries[entry.Name] = entry
}
func (m *mockPlayerDB) Delete(name string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.entries[name]; ok {
		delete(m.entries, name)
		return true
	}
	return false
}
func (m *mockPlayerDB) List() []*playerdb.PlayerEntry {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*playerdb.PlayerEntry, 0, len(m.entries))
	for _, e := range m.entries {
		out = append(out, e)
	}
	return out
}
func (m *mockPlayerDB) Search(prefix string) []*playerdb.PlayerEntry { return nil }
func (m *mockPlayerDB) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.entries)
}
func (m *mockPlayerDB) Flush() error { return nil }

// TestCheckAFKAutoMark verifies that checkAFK marks a player AFK when
// they have been inactive for AutoAfkTime.
func TestCheckAFKAutoMark(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: 5 * time.Millisecond,
	})
	client, done := connectPlayer(t, codec, "alice")
	defer func() {
		client.Close()
		<-done
	}()

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	p := codec.FindPlayer("alice")
	if p == nil {
		t.Fatal("player not found")
	}
	if !p.IsAfk() {
		t.Fatal("player should be AFK after inactivity")
	}

	msg := make([]byte, 66)
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read broadcast message: %v", err)
	}
	if msg[0] != wire.OpcodeMessage {
		t.Fatalf("broadcast opcode = %d, want %d", msg[0], wire.OpcodeMessage)
	}
}

// TestCheckAFKAutoUnmark verifies that checkAFK clears AFK when a player
// resumes activity. Regression test for the "sticky AFK" bug: AFK was
// never automatically cleared, so a returning player stayed AFK indefinitely.
func TestCheckAFKAutoUnmark(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: 50 * time.Millisecond,
	})
	client, done := connectPlayer(t, codec, "bob")
	defer func() {
		client.Close()
		<-done
	}()

	time.Sleep(60 * time.Millisecond)
	srv.checkAFK()

	p := codec.FindPlayer("bob")
	if p == nil {
		t.Fatal("player not found")
	}
	if !p.IsAfk() {
		t.Fatal("player should be AFK after inactivity")
	}

	msg := make([]byte, 66)
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read AFK broadcast: %v", err)
	}

	if _, err := client.Write(encodeRelPos(wire.SelfID, 0, 0, 0)); err != nil {
		t.Fatalf("write relpos: %v", err)
	}

	// Poll until the session has processed the movement packet
	// (lastAction becomes recent).
	deadline := time.Now().Add(200 * time.Millisecond)
	processed := false
	for time.Now().Before(deadline) {
		la, _, _ := codec.GetPlayerAFKState("bob")
		if !la.IsZero() && time.Since(la) < 10*time.Millisecond {
			processed = true
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if !processed {
		t.Fatal("movement packet was not processed within 200ms")
	}

	srv.checkAFK()

	if p.IsAfk() {
		t.Fatal("player should no longer be AFK after resuming activity")
	}

	msg2 := make([]byte, 66)
	if _, err := io.ReadFull(client, msg2); err != nil {
		t.Fatalf("read un-AFK broadcast: %v", err)
	}
	if msg2[0] != wire.OpcodeMessage {
		t.Fatalf("broadcast opcode = %d, want %d", msg2[0], wire.OpcodeMessage)
	}
}

func TestCheckAFKDoesNotClearManualAFKWithoutNewActivity(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: time.Hour,
	})
	client, done := connectPlayer(t, codec, "bob")
	defer func() {
		client.Close()
		<-done
	}()

	p := codec.FindPlayer("bob")
	if p == nil {
		t.Fatal("player not found")
	}
	p.SetAfk(true)
	srv.checkAFK()
	if !p.IsAfk() {
		t.Fatal("manual AFK should not clear without activity after afkSince")
	}
}

// TestCheckAFKKickFromAfkSince verifies that the AFK kick timer is
// measured from afkSince (the moment the player became AFK), not from
// lastAction. Regression test for the kick-timing bug.
func TestCheckAFKKickFromAfkSince(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: 5 * time.Millisecond,
		KickTime:    5 * time.Millisecond,
		KickMaxRank: 80,
	})
	client, done := connectPlayer(t, codec, "carol")
	defer func() {
		client.Close()
		<-done
	}()

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	p := codec.FindPlayer("carol")
	if p == nil {
		t.Fatal("player not found")
	}
	if !p.IsAfk() {
		t.Fatal("player should be AFK after inactivity")
	}

	msg := make([]byte, 66)
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read AFK broadcast: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	kick := make([]byte, 65)
	if _, err := io.ReadFull(client, kick); err != nil {
		t.Fatalf("read kick packet: %v", err)
	}
	if kick[0] != wire.OpcodeKick {
		t.Fatalf("kick opcode = %d, want %d", kick[0], wire.OpcodeKick)
	}
}

// TestCheckAFKNoKickBeforeKickTime verifies that a player is NOT kicked
// when afkSince has not yet reached KickTime. Regression test for the
// kick-timing bug: if the timer were measured from lastAction instead of
// afkSince, the kick would fire immediately (lastAction is older than
// KickTime by the time auto-AFK triggers).
func TestCheckAFKNoKickBeforeKickTime(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: 5 * time.Millisecond,
		KickTime:    500 * time.Millisecond,
		KickMaxRank: 80,
	})
	client, done := connectPlayer(t, codec, "eve")
	defer func() {
		client.Close()
		<-done
	}()

	// Trigger auto-AFK (lastAction is now ~15ms old, well past AutoAfkTime).
	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	p := codec.FindPlayer("eve")
	if p == nil {
		t.Fatal("player not found")
	}
	if !p.IsAfk() {
		t.Fatal("player should be AFK after inactivity")
	}

	// Drain the "is now AFK" broadcast.
	msg := make([]byte, 66)
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read AFK broadcast: %v", err)
	}

	// checkAFK immediately — afkSince was just set, KickTime not reached.
	srv.checkAFK()

	if !p.IsAfk() {
		t.Fatal("player should still be AFK (kick not due yet)")
	}

	// Verify no kick packet was sent — set a short read deadline.
	if err := client.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var buf [1]byte
	if _, err := client.Read(buf[:]); err == nil {
		t.Fatal("expected no kick (KickTime not reached), but got a packet")
	}
}

// TestCheckAFKKickExemptByRank verifies that players with rank >=
// KickMaxRank are not kicked. Regression test for the rank-exemption logic.
func TestCheckAFKKickExemptByRank(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: 5 * time.Millisecond,
		KickTime:    5 * time.Millisecond,
		KickMaxRank: 80,
	})
	client, done := connectPlayerWithRank(t, codec, "dave", 80)
	defer func() {
		client.Close()
		<-done
	}()

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	p := codec.FindPlayer("dave")
	if p == nil {
		t.Fatal("player not found")
	}
	if !p.IsAfk() {
		t.Fatal("player should be AFK after inactivity")
	}

	msg := make([]byte, 66)
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read AFK broadcast: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	if !p.IsAfk() {
		t.Fatal("player should still be AFK (not kicked, not un-AFKed)")
	}

	if err := client.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var buf [1]byte
	if _, err := client.Read(buf[:]); err == nil {
		t.Fatal("expected no data (player should not be kicked), but got a packet")
	}
}

// TestCheckAFKKickBelowMaxRank verifies that players with rank strictly
// below KickMaxRank ARE kicked. Complements TestCheckAFKKickExemptByRank
// to test the boundary (rank 79 vs KickMaxRank 80).
func TestCheckAFKKickBelowMaxRank(t *testing.T) {
	t.Parallel()

	srv, codec, _ := newAFKTestServer(t, config.AFKConfig{
		AutoAfkTime: 5 * time.Millisecond,
		KickTime:    5 * time.Millisecond,
		KickMaxRank: 80,
	})
	client, done := connectPlayerWithRank(t, codec, "frank", 79)
	defer func() {
		client.Close()
		<-done
	}()

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	p := codec.FindPlayer("frank")
	if p == nil {
		t.Fatal("player not found")
	}
	if !p.IsAfk() {
		t.Fatal("player should be AFK after inactivity")
	}

	msg := make([]byte, 66)
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read AFK broadcast: %v", err)
	}

	time.Sleep(15 * time.Millisecond)
	srv.checkAFK()

	kick := make([]byte, 65)
	if _, err := io.ReadFull(client, kick); err != nil {
		t.Fatalf("read kick packet: %v", err)
	}
	if kick[0] != wire.OpcodeKick {
		t.Fatalf("kick opcode = %d, want %d", kick[0], wire.OpcodeKick)
	}
}
