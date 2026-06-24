package classic

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func connectTestSession(t *testing.T) (*session, net.Conn, chan struct{}) {
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

// TestSetAfkSetsAndClearsAfkSince verifies that SetAfk(true) records the
// AFK-since timestamp and SetAfk(false) clears it. This is the regression
// test for AFK kick timing: the kick must be measured from afkSince, not
// from lastAction.
func TestSetAfkSetsAndClearsAfkSince(t *testing.T) {
	t.Parallel()
	s, client, done := connectTestSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	if !s.AfkSince().IsZero() {
		t.Fatalf("afkSince should be zero before SetAfk, got %v", s.AfkSince())
	}

	s.SetAfk(true)
	if s.AfkSince().IsZero() {
		t.Fatal("afkSince should be non-zero after SetAfk(true)")
	}
	if !s.IsAfk() {
		t.Fatal("IsAfk should be true after SetAfk(true)")
	}

	s.SetAfk(false)
	if !s.AfkSince().IsZero() {
		t.Fatalf("afkSince should be zero after SetAfk(false), got %v", s.AfkSince())
	}
	if s.IsAfk() {
		t.Fatal("IsAfk should be false after SetAfk(false)")
	}
}

// TestTouchLastActionUpdatesTimestamp verifies that touchLastAction
// updates the lastAction timestamp. This is the regression test for the
// data-race fix: lastAction must only be written under stateMu.
func TestTouchLastActionUpdatesTimestamp(t *testing.T) {
	t.Parallel()
	s, client, done := connectTestSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	before := s.LastAction()
	time.Sleep(2 * time.Millisecond)
	s.touchLastAction()
	after := s.LastAction()
	if !after.After(before) {
		t.Fatalf("touchLastAction did not update timestamp: before=%v after=%v", before, after)
	}
}

// TestGetPlayerAFKStateReturnsCorrectValues verifies that
// GetPlayerAFKState returns the correct lastAction, afkSince, and afk
// values before and after SetAfk.
func TestGetPlayerAFKStateReturnsCorrectValues(t *testing.T) {
	t.Parallel()
	codec := newTestCodec()
	server, client := net.Pipe()
	done := make(chan struct{})
	go func() {
		codec.ServeConn(context.Background(), server)
		close(done)
	}()
	loginAndDrain(t, client, 5, "tester", opcodePing)
	defer func() {
		client.Close()
		<-done
	}()

	la, afkSince, afk := codec.GetPlayerAFKState("tester")
	if la.IsZero() {
		t.Fatal("lastAction should be non-zero after login")
	}
	if afk {
		t.Fatal("player should not be AFK right after login")
	}
	if !afkSince.IsZero() {
		t.Fatal("afkSince should be zero when not AFK")
	}

	p := codec.FindPlayer("tester").(*session)
	p.SetAfk(true)

	_, afkSince2, afk2 := codec.GetPlayerAFKState("tester")
	if !afk2 {
		t.Fatal("player should be AFK after SetAfk(true)")
	}
	if afkSince2.IsZero() {
		t.Fatal("afkSince should be non-zero when AFK")
	}
}

// TestLastActionConcurrentAccess verifies that concurrent reads and writes
// to lastAction and afkSince do not cause data races. Run with -race.
func TestLastActionConcurrentAccess(t *testing.T) {
	t.Parallel()
	s, client, done := connectTestSession(t)
	defer func() {
		client.Close()
		<-done
	}()

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			s.touchLastAction()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			_ = s.LastAction()
			_ = s.AfkSince()
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 1000; i++ {
			s.SetAfk(true)
			s.SetAfk(false)
		}
	}()

	wg.Wait()
}
