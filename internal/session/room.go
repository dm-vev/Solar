package session

import (
	"strings"
	"sync"
)

// Participant is the minimal online session shape needed by Room.
type Participant interface {
	RoomEntityID() uint32
	RoomUsername() string
}

// Room tracks online participants for broadcast fan-out.
type Room[T Participant] struct {
	mu       sync.RWMutex
	sessions map[uint32]T
}

// NewRoom creates an empty online room.
func NewRoom[T Participant]() *Room[T] {
	return &Room[T]{sessions: make(map[uint32]T)}
}

// Join adds a participant and returns the existing peers.
func (r *Room[T]) Join(participant T) []T {
	r.mu.Lock()
	peers := make([]T, 0, len(r.sessions))
	for _, peer := range r.sessions {
		peers = append(peers, peer)
	}
	r.sessions[participant.RoomEntityID()] = participant
	r.mu.Unlock()
	return peers
}

// Leave removes a participant and returns the remaining peers.
func (r *Room[T]) Leave(entityID uint32) []T {
	r.mu.Lock()
	delete(r.sessions, entityID)
	peers := make([]T, 0, len(r.sessions))
	for _, peer := range r.sessions {
		peers = append(peers, peer)
	}
	r.mu.Unlock()
	return peers
}

// PeersExcept returns all peers except the participant with entityID.
func (r *Room[T]) PeersExcept(entityID uint32) []T {
	r.mu.RLock()
	peers := make([]T, 0, len(r.sessions))
	for id, peer := range r.sessions {
		if id == entityID {
			continue
		}
		peers = append(peers, peer)
	}
	r.mu.RUnlock()
	return peers
}

// ForEachPeerExcept calls fn for each peer except the participant with
// entityID, without allocating a slice. This is the preferred method
// for broadcast fan-out in hot paths.
func (r *Room[T]) ForEachPeerExcept(entityID uint32, fn func(peer T)) {
	r.mu.RLock()
	for id, peer := range r.sessions {
		if id != entityID {
			fn(peer)
		}
	}
	r.mu.RUnlock()
}

// ForEachPeer calls fn for every participant in the room.
func (r *Room[T]) ForEachPeer(fn func(peer T)) {
	r.mu.RLock()
	for _, peer := range r.sessions {
		fn(peer)
	}
	r.mu.RUnlock()
}

// FindByName returns the first participant with a case-insensitive username match.
func (r *Room[T]) FindByName(name string) (T, bool) {
	key := strings.TrimSpace(name)
	if key == "" {
		var zero T
		return zero, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, participant := range r.sessions {
		if strings.EqualFold(participant.RoomUsername(), key) {
			return participant, true
		}
	}
	var zero T
	return zero, false
}

// Snapshot returns a slice of all current participants. The caller may
// iterate it without holding the room lock.
func (r *Room[T]) Snapshot() []T {
	r.mu.RLock()
	peers := make([]T, 0, len(r.sessions))
	for _, peer := range r.sessions {
		peers = append(peers, peer)
	}
	r.mu.RUnlock()
	return peers
}
