package classic

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

func (s *session) setSupports(extPlayerList, twoWayPing, fastMap bool) {
	s.stateMu.Lock()
	s.supportsExtPlayerList = extPlayerList
	s.supportsTwoWayPing = twoWayPing
	s.supportsFastMap = fastMap
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

func (s *session) sessionFlags() (loggedIn, joined, supportsExtPlayerList, supportsTwoWayPing, supportsFastMap bool) {
	s.stateMu.RLock()
	loggedIn = s.loggedIn
	joined = s.joined
	supportsExtPlayerList = s.supportsExtPlayerList
	supportsTwoWayPing = s.supportsTwoWayPing
	supportsFastMap = s.supportsFastMap
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

func (s *session) currentSupportsExtPlayerList() bool {
	s.stateMu.RLock()
	enabled := s.supportsExtPlayerList
	s.stateMu.RUnlock()
	return enabled
}

func (s *session) currentSupportsTwoWayPing() bool {
	s.stateMu.RLock()
	enabled := s.supportsTwoWayPing
	s.stateMu.RUnlock()
	return enabled
}

func (s *session) currentSupportsFastMap() bool {
	s.stateMu.RLock()
	enabled := s.supportsFastMap
	s.stateMu.RUnlock()
	return enabled
}
