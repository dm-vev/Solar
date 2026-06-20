package classic

import (
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/world"
)

func (s *session) setIdentity(username string, entityID uint32, tracked bool) {
	s.stateMu.Lock()
	s.username = username
	s.entityID = entityID
	s.tracked = tracked
	s.stateMu.Unlock()
}

func (s *session) markLoggedIn() {
	s.stateMu.Lock()
	s.loggedIn = true
	s.stateMu.Unlock()
}

func (s *session) markJoined(joined bool) {
	s.stateMu.Lock()
	s.joined = joined
	s.stateMu.Unlock()
}

func (s *session) setCPESupport(exts map[string]uint32) {
	s.stateMu.Lock()
	s.cpeExts = exts
	s.stateMu.Unlock()
}

func (s *session) sessionIdentity() (username string, entityID uint32, tracked bool) {
	s.stateMu.RLock()
	username = s.username
	entityID = s.entityID
	tracked = s.tracked
	s.stateMu.RUnlock()
	return
}

func (s *session) sessionFlags() (loggedIn, joined bool) {
	s.stateMu.RLock()
	loggedIn = s.loggedIn
	joined = s.joined
	s.stateMu.RUnlock()
	return
}

func (s *session) currentUsername() string {
	s.stateMu.RLock()
	username := s.username
	s.stateMu.RUnlock()
	return username
}

func (s *session) currentEntityID() uint32 {
	s.stateMu.RLock()
	entityID := s.entityID
	s.stateMu.RUnlock()
	return entityID
}

// supportsExt reports whether the client supports the given CPE extension.
func (s *session) supportsExt(name string) bool {
	s.stateMu.RLock()
	_, ok := s.cpeExts[name]
	s.stateMu.RUnlock()
	return ok
}

func (s *session) currentSupportsFastMap() bool {
	return s.supportsExt(cpeExtFastMapName)
}

// CurrentWorldManager returns the session's active world Manager.
func (s *session) CurrentWorldManager() *world.Manager {
	s.stateMu.RLock()
	w := s.worlds
	s.stateMu.RUnlock()
	return w
}

func (s *session) currentSupportsExtPlayerList() bool {
	return s.supportsExt(cpeExtPlayerListName)
}

func (s *session) lastBroadcast() (entity.Position, byte, byte) {
	s.stateMu.RLock()
	pos := s.lastPos
	yaw := s.lastYaw
	pitch := s.lastPitch
	s.stateMu.RUnlock()
	return pos, yaw, pitch
}

func (s *session) setLastBroadcast(pos entity.Position, yaw, pitch byte) {
	s.stateMu.Lock()
	s.lastPos = pos
	s.lastYaw = yaw
	s.lastPitch = pitch
	s.stateMu.Unlock()
}
