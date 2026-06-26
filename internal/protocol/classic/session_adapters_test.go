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

	codec.BroadcastPacket([]byte{0x01})
	assertPacketCount(t, alice, 1)
	assertPacketCount(t, bob, 1)

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
