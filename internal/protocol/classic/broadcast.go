package classic

import (
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/plugin"
)

func (s *session) joinRoom() {
	if s.room == nil {
		return
	}

	_, entityID, _ := s.sessionIdentity()
	_, joined := s.sessionFlags()
	if joined || entityID == 0 {
		return
	}

	selfState, ok := s.entitySnapshot()
	if !ok {
		return
	}

	peers := s.room.Join(s)

	username := s.currentUsername()
	selfPacket := encodeAddEntity(byte(entityID), username, selfState.Pos, selfState.Yaw, selfState.Pitch)
	myWorld := s.CurrentWorldManager()
	for _, peer := range peers {
		// Only spawn entities for peers on the same level.
		if peer.CurrentWorldManager() != myWorld {
			continue
		}
		peerState, ok := peer.entitySnapshot()
		if ok {
			peerUsername, peerEntityID, _ := peer.sessionIdentity()
			if s.canSee(peer) {
				if err := s.writePacket(encodeAddEntity(byte(peerEntityID), peerUsername, peerState.Pos, peerState.Yaw, peerState.Pitch)); err != nil {
					s.logger.Debug("send peer join packet", "username", s.currentUsername(), "peer", peerUsername, "error", err)
				}
				if s.supportsExt(cpeExtPlayerListName) {
					if err := s.writePacket(encodeExtAddPlayerName(byte(peerEntityID), peerUsername)); err != nil {
						s.logger.Debug("send peer list packet", "username", s.currentUsername(), "peer", peerUsername, "error", err)
					}
				}
			}
		}
		if peer.canSee(s) {
			if err := peer.writePacket(selfPacket); err != nil {
				s.logger.Debug("broadcast join packet", "username", username, "peer", peer.currentUsername(), "error", err)
			}
			if peer.currentSupportsExtPlayerList() {
				if err := peer.writePacket(encodeExtAddPlayerName(byte(entityID), username)); err != nil {
					s.logger.Debug("broadcast player list packet", "username", username, "peer", peer.currentUsername(), "error", err)
				}
			}
		}
	}

	s.markJoined(true)
}

// canSee reports whether s can see peer, consulting OnGettingCanSee handlers.
// ponytail: fires the event per check; join-only path is not hot.
// If no handlers are registered, returns true (default visible).
func (s *session) canSee(peer *session) bool {
	if !plugin.OnGettingCanSee.HasHandlers() {
		return true
	}
	canSee := true
	plugin.OnGettingCanSee.Fire(plugin.GettingCanSeeData{
		Player: s,
		Target: peer,
		CanSee: &canSee,
	})
	return canSee
}

func (s *session) leaveRoom() {
	if s.room == nil {
		return
	}

	_, entityID, _ := s.sessionIdentity()
	_, joined := s.sessionFlags()
	if !joined || entityID == 0 {
		return
	}

	peers := s.room.Leave(entityID)
	myWorld := s.CurrentWorldManager()
	packet := encodeRemoveEntity(byte(entityID))
	for _, peer := range peers {
		if peer.CurrentWorldManager() != myWorld {
			continue
		}
		if err := peer.writePacket(packet); err != nil {
			s.logger.Debug("broadcast leave packet", "entity_id", entityID, "peer", peer.currentUsername(), "error", err)
		}
		if peer.currentSupportsExtPlayerList() {
			if err := peer.writePacket(encodeExtRemovePlayerName(byte(entityID))); err != nil {
				s.logger.Debug("broadcast player list removal", "entity_id", entityID, "peer", peer.currentUsername(), "error", err)
			}
		}
	}
	s.markJoined(false)
}

func (s *session) broadcastToPeers(packet []byte) {
	if s.room == nil {
		return
	}

	_, entityID, _ := s.sessionIdentity()
	if entityID == 0 {
		return
	}

	myWorld := s.CurrentWorldManager()

	s.room.ForEachPeerExcept(entityID, func(peer *session) {
		// ponytail: filter by world manager pointer identity — peers on
		// other levels don't see entity updates from this level.
		if peer.CurrentWorldManager() != myWorld {
			return
		}
		if err := peer.writePacketNoCopy(packet); err != nil {
			s.logger.Debug("broadcast packet", "entity_id", entityID, "peer", peer.currentUsername(), "error", err)
		}
	})
}

func (s *session) entitySnapshot() (entity.Entity, bool) {
	if s.entities == nil {
		return entity.Entity{}, false
	}
	entityID := s.currentEntityID()
	if entityID == 0 {
		return entity.Entity{}, false
	}
	return s.entities.Get(entityID)
}
