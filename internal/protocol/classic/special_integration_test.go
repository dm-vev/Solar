package classic

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/protocol/wire"
)

// connectAndLogin creates a test session, logs in, and returns the session,
// client conn, and done channel.
func connectSpecialSession(t *testing.T) (*session, net.Conn, chan struct{}) {
	t.Helper()
	codec := newTestCodec()
	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()
	loginAndDrain(t, client, 5, "tester", opcodePing)
	p := codec.FindPlayer("tester")
	if p == nil {
		t.Fatal("player not found after login")
	}
	return p.(*session), client, done
}

// ─── @p replacement in message blocks ───

// TestSpecial_MessageBlockAtPReplacement verifies that @p is replaced with
// the player's username when a message block fires.
func TestSpecial_MessageBlockAtPReplacement(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	// Register a message block with @p at the player's feet.
	s.specialBlocks.Set(1, 1, 1, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "Hello @p, welcome!",
	})

	// Set lastSpecialBlock to a different position so the check fires.
	s.lastSpecialBlock = [3]int{0, 0, 0}

	// Trigger checkSpecialBlocks at the message block position.
	s.checkSpecialBlocks(1, 1, 1)

	// Read the message packet.
	msg := make([]byte, 66)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read message: %v", err)
	}
	if msg[0] != wire.OpcodeMessage {
		t.Fatalf("opcode = %d, want %d", msg[0], wire.OpcodeMessage)
	}
	text := string(bytesTrimRight(msg[2:66], " \x00"))
	if text != "Hello tester, welcome!" {
		t.Fatalf("message = %q, want 'Hello tester, welcome!'", text)
	}
}

// ─── Plain message block (no @p) ───

func TestSpecial_MessageBlockPlain(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	s.specialBlocks.Set(2, 2, 2, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "Welcome to the zone!",
	})
	s.lastSpecialBlock = [3]int{0, 0, 0}
	s.checkSpecialBlocks(2, 2, 2)

	msg := make([]byte, 66)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read message: %v", err)
	}
	text := string(bytesTrimRight(msg[2:66], " \x00"))
	if text != "Welcome to the zone!" {
		t.Fatalf("message = %q, want 'Welcome to the zone!'", text)
	}
}

// ─── Message block dedup ───

func TestSpecial_MessageBlockDedup(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	s.specialBlocks.Set(3, 3, 3, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "You should see this once!",
	})
	s.lastSpecialBlock = [3]int{0, 0, 0}

	// First trigger — should fire.
	s.checkSpecialBlocks(3, 3, 3)

	// Read the first message.
	msg := make([]byte, 66)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read first message: %v", err)
	}

	// Second trigger at same position — should NOT fire.
	s.checkSpecialBlocks(3, 3, 3)

	// Verify no second message arrives.
	if err := client.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	var buf [1]byte
	if _, err := client.Read(buf[:]); err == nil {
		t.Fatal("second checkSpecialBlocks at same position should not send a message")
	}
}

// ─── Door toggle preserves registry entry ───

func TestSpecial_DoorTogglePreservesEntry(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	// Register a door at (5, 5, 5) with DoorBlock=Log(17).
	s.specialBlocks.Set(5, 5, 5, &blocks.SpecialEntry{
		Type:      blocks.SpecialDoor,
		DoorBlock: 17, // Log
	})

	// Simulate door toggle: door block (201) → air.
	// applyBlockChange with blockID=0 (air), placing=false.
	// The door entry should survive because existing.Type == SpecialDoor.
	s.applyBlockChange(5, 5, 5, 0, true)

	entry := s.specialBlocks.Get(5, 5, 5)
	if entry == nil {
		t.Fatal("door entry was removed after toggling to air — should be preserved")
	}
	if entry.Type != blocks.SpecialDoor {
		t.Fatalf("entry type = %d, want SpecialDoor", entry.Type)
	}

	// Simulate second toggle: air → Log (17).
	// applyBlockChange with blockID=17 (Log), placing=true.
	// The door entry should survive because existing.Type == SpecialDoor.
	s.applyBlockChange(5, 5, 5, 17, true)

	entry = s.specialBlocks.Get(5, 5, 5)
	if entry == nil {
		t.Fatal("door entry was removed after toggling to solid — should be preserved")
	}
	if entry.Type != blocks.SpecialDoor {
		t.Fatalf("entry type = %d, want SpecialDoor", entry.Type)
	}
}

// ─── Door entry removed when player manually deletes block ───

func TestSpecial_DoorEntryRemovedOnManualDelete(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	// Place a message block (not a door) at (3, 3, 3).
	s.specialBlocks.Set(3, 3, 3, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "test",
	})

	// Manually delete the block (air).
	s.applyBlockChange(3, 3, 3, 0, true)

	// The message block entry should be removed (not a door).
	entry := s.specialBlocks.Get(3, 3, 3)
	if entry != nil {
		t.Fatal("message block entry should be removed when block is deleted")
	}
}

// ─── Portal same-level teleport ───

func TestSpecial_PortalSameLevelTeleport(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	// Register a same-level portal at (1, 1, 1) → teleport to (50, 60, 70).
	s.specialBlocks.Set(1, 1, 1, &blocks.SpecialEntry{
		Type:      blocks.SpecialPortal,
		PortalDst: [3]int{50, 60, 70},
	})
	s.lastSpecialBlock = [3]int{0, 0, 0}

	// Trigger the portal.
	s.checkSpecialBlocks(1, 1, 1)

	// The player should receive a teleport packet to the destination.
	tp := make([]byte, 10)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, tp); err != nil {
		t.Fatalf("read teleport: %v", err)
	}
	if tp[0] != wire.OpcodeEntityTeleport {
		t.Fatalf("opcode = %d, want %d (EntityTeleport)", tp[0], wire.OpcodeEntityTeleport)
	}
	wantX := uint16(50 * 32)
	gotX := uint16(tp[2])<<8 | uint16(tp[3])
	if gotX != wantX {
		t.Fatalf("teleport X = %d, want %d", gotX, wantX)
	}
}

// ─── Message block command execution ───

func TestSpecial_MessageBlockCommandExecution(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	// Register a message block with a /where command.
	s.specialBlocks.Set(1, 1, 1, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "/where",
	})
	s.lastSpecialBlock = [3]int{0, 0, 0}

	// Trigger the message block.
	s.checkSpecialBlocks(1, 1, 1)

	// The /where command should produce a reply message.
	msg := make([]byte, 66)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, msg); err != nil {
		t.Fatalf("read command reply: %v", err)
	}
	if msg[0] != wire.OpcodeMessage {
		t.Fatalf("opcode = %d, want %d (Message)", msg[0], wire.OpcodeMessage)
	}
	// The reply should contain the translation key for /where.
	text := bytesTrimRight(msg[2:66], " \x00")
	if !contains(string(text), "command.where") {
		t.Fatalf("command reply = %q, want 'command.where.position' key", text)
	}
}

// ─── Piped commands in message block ───

func TestSpecial_MessageBlockPipedCommands(t *testing.T) {
	t.Parallel()
	s, client, done := connectSpecialSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	// Register a message block with text + piped command.
	s.specialBlocks.Set(1, 1, 1, &blocks.SpecialEntry{
		Type:    blocks.SpecialMessage,
		Message: "Hello |/where",
	})
	s.lastSpecialBlock = [3]int{0, 0, 0}

	// Trigger the message block.
	s.checkSpecialBlocks(1, 1, 1)

	// First packet: the text "Hello".
	msg1 := make([]byte, 66)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, msg1); err != nil {
		t.Fatalf("read text message: %v", err)
	}
	if msg1[0] != wire.OpcodeMessage {
		t.Fatalf("first packet opcode = %d, want Message", msg1[0])
	}
	text1 := bytesTrimRight(msg1[2:66], " \x00")
	if string(text1) != "Hello" {
		t.Fatalf("first message = %q, want 'Hello'", text1)
	}

	// Second packet: the /where command reply.
	msg2 := make([]byte, 66)
	if err := client.SetReadDeadline(time.Now().Add(500 * time.Millisecond)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, err := io.ReadFull(client, msg2); err != nil {
		t.Fatalf("read command reply: %v", err)
	}
	if msg2[0] != wire.OpcodeMessage {
		t.Fatalf("second packet opcode = %d, want Message", msg2[0])
	}
}

// ─── helpers ───

func bytesTrimRight(b []byte, cutset string) []byte {
	for len(b) > 0 {
		found := false
		c := b[len(b)-1]
		for i := 0; i < len(cutset); i++ {
			if c == cutset[i] {
				b = b[:len(b)-1]
				found = true
				break
			}
		}
		if !found {
			break
		}
	}
	return b
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
