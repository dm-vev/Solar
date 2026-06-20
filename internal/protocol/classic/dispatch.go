// dispatch.go contains the packet opcode dispatch logic.
//
// The read loop (session.run) reads a single byte opcode, then
// dispatches to the appropriate handler based on the value. Most
// handlers require the session to be logged in (handshake completed).
//
// CPE packets (PlayerClick, PluginMessage, NotifyAction,
// NotifyPositionAction) are dispatched to handleCPEPacket which
// further routes to the specific CPE handler.

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
