package classic

import (
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/ranks"
	roompkg "github.com/solar-mc/solar/internal/session"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

func TestSessionRankOperatorFallback(t *testing.T) {
	s := newCoverageSession(t, "alice")
	s.players.AddOperators("alice")

	if got := s.PlayerRank(); got != ranks.PermOperator {
		t.Fatalf("PlayerRank = %d, want operator", got)
	}
	if got := s.RankGetPlayer("alice"); got != ranks.PermOperator {
		t.Fatalf("RankGetPlayer = %d, want operator", got)
	}
}

func TestPluginPlayerSelectionAndBuildMetadata(t *testing.T) {
	s := newCoverageSession(t, "alice")
	var got []plugin.BlockPos
	if !s.SelectBlocks(2, func(marks []plugin.BlockPos) { got = marks }) {
		t.Fatal("SelectBlocks returned false")
	}
	s.selectionMu.Lock()
	callback := s.markState.callback
	s.selectionMu.Unlock()
	callback([]markPos{{X: 1, Y: 2, Z: 3}, {X: 4, Y: 5, Z: 6}})
	if len(got) != 2 || got[0] != (plugin.BlockPos{X: 1, Y: 2, Z: 3}) || got[1] != (plugin.BlockPos{X: 4, Y: 5, Z: 6}) {
		t.Fatalf("selection marks = %+v", got)
	}
	if s.LevelName() != s.worlds.Current().Name {
		t.Fatalf("LevelName = %q", s.LevelName())
	}
	if s.Rank() != s.PlayerRank() || s.DrawLimit() < 1 {
		t.Fatalf("rank=%d drawLimit=%d", s.Rank(), s.DrawLimit())
	}
	if !s.CancelSelection() || s.CancelSelection() {
		t.Fatal("CancelSelection did not report active state")
	}
}

func TestSessionTeleportUsesWireCoordinates(t *testing.T) {
	alice := newCoverageSession(t, "alice")
	bob := newCoverageSession(t, "bob")
	bob.worlds = alice.worlds
	bobID, ok := bob.entities.Add("bob", entity.Position{X: 5 * coordScale, Y: 6 * coordScale, Z: 7 * coordScale})
	if !ok {
		t.Fatal("add bob entity")
	}
	bob.setIdentity("bob", bobID, true)
	bob.players.Remove("bob")
	bob.players.Add("bob", bobID)

	room := roompkg.NewRoom[*session]()
	alice.room = room
	bob.room = room
	room.Join(alice)
	room.Join(bob)

	if !alice.TeleportToPlayer("bob") {
		t.Fatal("TeleportToPlayer returned false")
	}
	if !alice.BackToLastPos() {
		t.Fatal("BackToLastPos returned false")
	}
	snap, ok := alice.entities.Get(alice.currentEntityID())
	if !ok || snap.Pos.X != coordScale || snap.Pos.Y != 2*coordScale || snap.Pos.Z != 3*coordScale {
		t.Fatalf("BackToLastPos position = %+v ok=%v", snap, ok)
	}

	if !alice.SummonPlayer("bob") {
		t.Fatal("SummonPlayer returned false")
	}
	snap, ok = bob.entities.Get(bob.currentEntityID())
	if !ok || snap.Pos.X != coordScale || snap.Pos.Y != 2*coordScale || snap.Pos.Z != 3*coordScale {
		t.Fatalf("SummonPlayer position = %+v ok=%v", snap, ok)
	}
}

func TestSessionTPARequiresAcceptance(t *testing.T) {
	alice := newCoverageSession(t, "alice")
	bob := newCoverageSession(t, "bob")
	bob.worlds = alice.worlds
	bob.entities = alice.entities
	bob.players = alice.players
	bobID, ok := bob.entities.Add("bob", entity.Position{X: 5 * coordScale, Y: 6 * coordScale, Z: 7 * coordScale})
	if !ok {
		t.Fatal("add bob entity")
	}
	bob.entities.SetLocation(bobID, entity.Position{X: 5 * coordScale, Y: 6 * coordScale, Z: 7 * coordScale}, 9, 10)
	bob.setIdentity("bob", bobID, true)
	bob.players.Add("bob", bobID)

	room := roompkg.NewRoom[*session]()
	requests := newTPARequestStore()
	alice.room, bob.room = room, room
	alice.tpaRequests, bob.tpaRequests = requests, requests
	room.Join(alice)
	room.Join(bob)

	status, name := alice.RequestTeleport("bo")
	if status != command.TPARequestSent || name != "bob" {
		t.Fatalf("request = %v %q", status, name)
	}
	assertSessionBlockPosition(t, alice, 1, 2, 3)

	status, name = bob.RespondTeleport(false)
	if status != command.TPADenied || name != "alice" {
		t.Fatalf("deny = %v %q", status, name)
	}
	assertSessionBlockPosition(t, alice, 1, 2, 3)

	if status, _ = alice.RequestTeleport("bob"); status != command.TPARequestSent {
		t.Fatalf("second request = %v", status)
	}
	status, name = bob.RespondTeleport(true)
	if status != command.TPAAccepted || name != "alice" {
		t.Fatalf("accept = %v %q", status, name)
	}
	assertSessionBlockPosition(t, alice, 5, 6, 7)
	if !alice.BackToLastPos() {
		t.Fatal("back after accepted tpa failed")
	}
	assertSessionBlockPosition(t, alice, 1, 2, 3)
}

func TestTPARequestStoreIgnoresStaleTimeout(t *testing.T) {
	requests := newTPARequestStore()
	first, pending, _ := requests.add("alice", "bob", func(string, uint64) {})
	if first == nil || pending != "" {
		t.Fatalf("first request = %#v pending=%q", first, pending)
	}
	if _, ok := requests.take("bob"); !ok {
		t.Fatal("take first request")
	}
	second, pending, _ := requests.add("charlie", "bob", func(string, uint64) {})
	if second == nil || pending != "" {
		t.Fatalf("second request = %#v pending=%q", second, pending)
	}
	if _, ok := requests.expire("bob", first.id); ok {
		t.Fatal("stale timeout removed the new request")
	}
	got, ok := requests.take("bob")
	if !ok || got.requester != "charlie" {
		t.Fatalf("remaining request = %#v ok=%v", got, ok)
	}
}

func TestSessionTPACrossLevelBackReturnsToOrigin(t *testing.T) {
	alice := newCoverageSession(t, "alice")
	bob := newCoverageSession(t, "bob")
	origin := alice.worlds
	destination := world.NewManager()
	destination.SetCurrent(world.Level{Name: "other", Width: 4, Height: 4, Length: 4, Blocks: make([]byte, 64)})
	bob.worlds = destination
	bob.entities = alice.entities
	bob.players = alice.players
	bobID, ok := bob.entities.Add("bob", entity.Position{X: 2 * coordScale, Y: 2 * coordScale, Z: 2 * coordScale})
	if !ok {
		t.Fatal("add bob entity")
	}
	bob.setIdentity("bob", bobID, true)
	bob.players.Add("bob", bobID)

	room := roompkg.NewRoom[*session]()
	requests := newTPARequestStore()
	alice.room, bob.room = room, room
	alice.tpaRequests, bob.tpaRequests = requests, requests
	room.Join(alice)
	room.Join(bob)
	if status, _ := alice.RequestTeleport("bob"); status != command.TPARequestSent {
		t.Fatalf("request = %v", status)
	}
	if status, _ := bob.RespondTeleport(true); status != command.TPAAccepted {
		t.Fatalf("accept = %v", status)
	}
	if alice.CurrentWorldManager() != destination {
		t.Fatal("requester did not change to target level")
	}
	assertSessionBlockPosition(t, alice, 2, 2, 2)
	if !alice.BackToLastPos() {
		t.Fatal("cross-level back failed")
	}
	if alice.CurrentWorldManager() != origin {
		t.Fatal("back did not restore origin level")
	}
	assertSessionBlockPosition(t, alice, 1, 2, 3)
}

func assertSessionBlockPosition(t *testing.T, s *session, x, y, z int) {
	t.Helper()
	snapshot, ok := s.entities.Get(s.currentEntityID())
	if !ok || snapshot.Pos.X != x*coordScale || snapshot.Pos.Y != y*coordScale || snapshot.Pos.Z != z*coordScale {
		t.Fatalf("position = %+v ok=%v, want %d,%d,%d", snapshot.Pos, ok, x, y, z)
	}
}

func TestCodecBroadcastAndChangeMap(t *testing.T) {
	codec := NewCodec("Solar", "MOTD", world.NewManager(), player.NewRegistry(), entity.NewManager(), command.NewRegistry())
	alice := newCoverageSession(t, "alice")
	bob := newCoverageSession(t, "bob")
	bob.worlds = alice.worlds
	bobID, ok := bob.entities.Add("bob2", entity.Position{X: 2 * coordScale, Y: 2 * coordScale, Z: 2 * coordScale})
	if !ok {
		t.Fatal("add second bob entity")
	}
	bob.setIdentity("bob", bobID, true)
	codec.worlds = alice.worlds
	alice.room = codec.room
	bob.room = codec.room
	codec.room.Join(alice)
	codec.room.Join(bob)

	packet := []byte{0x01}
	codec.BroadcastPacket(packet)
	packet[0] = 0x02
	assertNextPacketByte(t, alice, 0x01)
	assertNextPacketByte(t, bob, 0x01)

	next := world.NewManager()
	next.SetCurrent(world.Level{
		Name:   "next",
		Width:  4,
		Height: 4,
		Length: 4,
		Blocks: make([]byte, 64),
		Spawn:  world.Spawn{X: 1, Y: 2, Z: 3},
	})
	alice.markJoined(true)
	if err := codec.ChangeMap(alice, next); err != nil {
		t.Fatalf("ChangeMap: %v", err)
	}
	assertPacketCountAtLeast(t, alice, 1)
}

func TestSessionCPEAccessor(t *testing.T) {
	s := newCoverageSession(t, "alice")
	if s.CPE() != s {
		t.Fatal("CPE did not return session")
	}
}

func newCoverageSession(t *testing.T, name string) *session {
	t.Helper()

	worlds := world.NewManager()
	entities := entity.NewManager()
	entityID, ok := entities.Add(name, entity.Position{X: 32, Y: 64, Z: 96})
	if !ok {
		t.Fatalf("add entity %s", name)
	}
	entities.SetLocation(entityID, entity.Position{X: 32, Y: 64, Z: 96}, 7, 8)

	players := player.NewRegistry()
	players.Add(name, entityID)
	var sendTimeout atomic.Int64
	sendTimeout.Store(int64(time.Second))

	s := &session{
		serverName:          "Solar",
		motd:                "MOTD",
		worlds:              worlds,
		players:             players,
		entities:            entities,
		commands:            command.NewRegistry(),
		logger:              slog.New(slog.NewTextHandler(io.Discard, nil)),
		outbox:              make(chan []byte, 128),
		stop:                make(chan struct{}),
		sendTimeoutVal:      &sendTimeout,
		writeBatchSize:      16,
		shutdownBatchSize:   16,
		buildCommandContext: testBuildContext,
		allowBuild:          true,
	}
	s.setIdentity(name, entityID, true)
	s.markLoggedIn()
	return s
}

func assertPacketCount(t *testing.T, s *session, want int) {
	t.Helper()
	got := drainPackets(s)
	if got != want {
		t.Fatalf("drained packets = %d, want %d", got, want)
	}
}

func assertNextPacketByte(t *testing.T, s *session, want byte) {
	t.Helper()
	select {
	case packet := <-s.outbox:
		if len(packet) != 1 || packet[0] != want {
			t.Fatalf("packet = %v, want [%d]", packet, want)
		}
	default:
		t.Fatal("outbox is empty")
	}
}

func assertPacketCountAtLeast(t *testing.T, s *session, want int) {
	t.Helper()
	got := drainPackets(s)
	if got < want {
		t.Fatalf("drained packets = %d, want at least %d", got, want)
	}
}

func drainPackets(s *session) int {
	count := 0
	for {
		select {
		case <-s.outbox:
			count++
		default:
			return count
		}
	}
}
