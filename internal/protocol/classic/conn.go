// conn.go implements the connection write loop and lifecycle.
//
// The write loop runs in a goroutine started by Codec.ServeConn. It
// drains the outbox channel, batches packets into a single TCP write,
// and flushes the buffered writer. When the session is stopped, the
// loop drains remaining packets (shutdown batch) before closing the
// connection.
//
// writePacket is the primary entry point for sending packets. It copies
// the packet (because callers may reuse the buffer), then either sends
// immediately or blocks up to sendTimeout before disconnecting the client.
//
// writePacketNoCopy is used for broadcast packets where the caller
// guarantees the slice will not be modified. It drops the packet if the
// outbox is full (best-effort for broadcasts).

package classic

import (
	"io"
	"strings"
	"time"
)

func (s *session) writeKick(message string) error {
	return s.writePacket(encodeKick(message))
}

// writePacket queues a packet for asynchronous writing. The packet slice
// is copied because the caller may reuse the underlying buffer.
// Blocks up to sendTimeout if the outbox is full, then disconnects.
func (s *session) writePacket(packet []byte) error {
	packetCopy := append([]byte(nil), packet...)
	select {
	case s.outbox <- packetCopy:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	default:
	}
	timeout := time.Duration(s.sendTimeoutVal.Load())
	if timeout <= 0 {
		s.fail()
		return io.ErrShortWrite
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case s.outbox <- packetCopy:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	case <-timer.C:
		s.fail()
		return io.ErrShortWrite
	}
}

// writePacketNoCopy queues a broadcast packet without copying. The caller must
// guarantee that the packet slice will not be modified after this call.
// Drops the packet if the outbox is full — broadcast packets are best-effort.
func (s *session) writePacketNoCopy(packet []byte) error {
	select {
	case s.outbox <- packet:
		return nil
	case <-s.stop:
		return io.ErrClosedPipe
	default:
		return nil
	}
}

func (s *session) writeLoop() {
	defer close(s.writerDone)
	defer s.closeConn()

	for {
		select {
		case packet := <-s.outbox:
			if !s.writePacketBuffer(packet) {
				return
			}
			// Batch any additional packets already queued so that small
			// broadcasts do not trigger a syscall per packet.
			if !s.drainOutbox(s.writeBatchSize) {
				return
			}
			if !s.flushWriter() {
				return
			}
		case <-s.stop:
			if !s.drainOutbox(s.shutdownBatchSize) {
				return
			}
			if !s.flushWriter() {
				return
			}
			return
		}
	}
}

// writePacketBuffer writes a packet to the buffered writer without flushing.
func (s *session) writePacketBuffer(packet []byte) bool {
	if packet == nil {
		return false
	}
	if s.writeDeadline > 0 {
		if err := s.conn.SetWriteDeadline(time.Now().Add(s.writeDeadline)); err != nil {
			return false
		}
	}
	_, err := s.writer.Write(packet)
	return err == nil
}

// drainOutbox pulls up to max packets from the outbox and writes them without
// flushing. Returns false if a write failed and the loop should stop.
func (s *session) drainOutbox(max int) bool {
	for i := 0; i < max; i++ {
		select {
		case packet := <-s.outbox:
			if !s.writePacketBuffer(packet) {
				return false
			}
		default:
			return true
		}
	}
	return true
}

// flushWriter flushes the buffered writer. Returns false on error.
func (s *session) flushWriter() bool {
	return s.writer.Flush() == nil
}

func (s *session) fail() {
	s.closeStop()
}

func (s *session) disconnect(message string) {
	if strings.TrimSpace(message) == "" {
		message = "kicked"
	}
	_ = s.writeKick(message)
	s.fail()
}

func (s *session) closeStop() {
	s.stopOnce.Do(func() {
		close(s.stop)
	})
}

func (s *session) closeConn() {
	s.connOnce.Do(func() {
		if s.conn != nil {
			_ = s.conn.Close()
		}
	})
}
