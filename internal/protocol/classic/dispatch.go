package classic

type packetHandler func() error

func (s *session) handleLoggedInPacket(_ byte, handler packetHandler) error {
	if !s.loggedIn {
		return nil
	}
	return handler()
}

func (s *session) handleCPEPacket(opcode byte) error {
	if !s.loggedIn {
		return nil
	}
	switch opcode {
	case opcodePlayerClick:
		return s.handlePlayerClick()
	case opcodePluginMessage:
		return s.handlePluginMessage()
	case opcodeNotifyAction:
		return s.handleNotifyAction()
	case opcodeNotifyPositionAction:
		return s.handleNotifyPositionAction()
	default:
		return nil
	}
}
