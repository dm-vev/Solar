package session

import (
	"testing"
	"time"
)

type testParticipant struct {
	id   uint32
	name string
}

func (p testParticipant) RoomEntityID() uint32 { return p.id }
func (p testParticipant) RoomUsername() string { return p.name }

func TestRoomIterationAndSnapshot(t *testing.T) {
	room := NewRoom[testParticipant]()
	room.Join(testParticipant{id: 1, name: "alice"})
	room.Join(testParticipant{id: 2, name: "bob"})

	var except []string
	room.ForEachPeerExcept(1, func(peer testParticipant) {
		except = append(except, peer.name)
	})
	if len(except) != 1 || except[0] != "bob" {
		t.Fatalf("ForEachPeerExcept = %v", except)
	}

	count := 0
	room.ForEachPeer(func(peer testParticipant) {
		count++
	})
	if count != 2 {
		t.Fatalf("ForEachPeer count = %d, want 2", count)
	}

	snapshot := room.Snapshot()
	if len(snapshot) != 2 {
		t.Fatalf("Snapshot len = %d, want 2", len(snapshot))
	}
}

func TestRoomLookupAndLeave(t *testing.T) {
	room := NewRoom[testParticipant]()
	room.Join(testParticipant{id: 1, name: "alice"})
	room.Join(testParticipant{id: 2, name: "Bob"})
	room.Join(testParticipant{id: 3, name: "charlie"})

	peers := room.PeersExcept(2)
	if len(peers) != 2 {
		t.Fatalf("PeersExcept len = %d, want 2", len(peers))
	}
	for _, peer := range peers {
		if peer.id == 2 {
			t.Fatalf("PeersExcept returned excluded peer: %+v", peer)
		}
	}

	found, ok := room.FindByName(" bob ")
	if !ok || found.id != 2 {
		t.Fatalf("FindByName = %+v ok=%v, want id 2", found, ok)
	}
	if _, ok := room.FindByName(""); ok {
		t.Fatal("FindByName accepted blank name")
	}
	if _, ok := room.FindByName("missing"); ok {
		t.Fatal("FindByName found missing participant")
	}

	remaining := room.Leave(1)
	if len(remaining) != 2 {
		t.Fatalf("Leave remaining len = %d, want 2", len(remaining))
	}
	if _, ok := room.FindByName("alice"); ok {
		t.Fatal("Leave did not remove participant")
	}
}

func TestRoomCallbacksRunWithoutLock(t *testing.T) {
	room := NewRoom[testParticipant]()
	room.Join(testParticipant{id: 1, name: "alice"})

	done := make(chan struct{})
	go func() {
		room.ForEachPeer(func(testParticipant) {
			room.Join(testParticipant{id: 2, name: "bob"})
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("ForEachPeer held room lock while running callback")
	}
}
