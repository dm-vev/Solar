package session

import "testing"

type testParticipant struct {
	id   uint32
	name string
}

func (p testParticipant) RoomEntityID() uint32 { return p.id }
func (p testParticipant) RoomUsername() string { return p.name }

func TestRoomJoinLeaveAndLookup(t *testing.T) {
	t.Parallel()

	room := NewRoom[testParticipant]()
	alice := testParticipant{id: 1, name: "Alice"}
	bob := testParticipant{id: 2, name: "Bob"}

	if peers := room.Join(alice); len(peers) != 0 {
		t.Fatalf("first join peers = %d, want 0", len(peers))
	}
	if peers := room.Join(bob); len(peers) != 1 || peers[0].id != alice.id {
		t.Fatalf("second join peers = %#v", peers)
	}
	if got, ok := room.FindByName("alice"); !ok || got.id != alice.id {
		t.Fatalf("FindByName alice = %#v ok=%v", got, ok)
	}
	if peers := room.PeersExcept(alice.id); len(peers) != 1 || peers[0].id != bob.id {
		t.Fatalf("PeersExcept alice = %#v", peers)
	}
	if peers := room.Leave(alice.id); len(peers) != 1 || peers[0].id != bob.id {
		t.Fatalf("Leave alice peers = %#v", peers)
	}
	if _, ok := room.FindByName("Alice"); ok {
		t.Fatal("FindByName found removed participant")
	}
}
