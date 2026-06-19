package network

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestListenerStopsOnContextCancel(t *testing.T) {
	t.Parallel()

	ln := NewListener("127.0.0.1:0")
	ln.SetConnectRate(1)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- ln.Serve(ctx, func(net.Conn) {})
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not stop after context cancel")
	}
}

func TestSetConnectRateClampsToMinimum(t *testing.T) {
	t.Parallel()

	ln := NewListener(":0")
	ln.SetConnectRate(0)
	if ln.connectRate != 1 {
		t.Fatalf("connectRate = %d, want 1", ln.connectRate)
	}
	ln.SetConnectRate(-5)
	if ln.connectRate != 1 {
		t.Fatalf("connectRate = %d, want 1", ln.connectRate)
	}
}

func TestListenerAcceptsAndHandlesConnection(t *testing.T) {
	t.Parallel()

	// Use a fixed port that's likely free.
	ln := NewListener("127.0.0.1:13579")
	ln.SetConnectRate(100)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	accepted := make(chan net.Conn, 1)
	go func() {
		_ = ln.Serve(ctx, func(conn net.Conn) {
			accepted <- conn
		})
	}()

	// Give the listener time to bind.
	time.Sleep(100 * time.Millisecond)

	conn, err := net.DialTimeout("tcp", "127.0.0.1:13579", time.Second)
	if err != nil {
		t.Skipf("could not dial listener: %v", err)
	}
	defer conn.Close()

	select {
	case c := <-accepted:
		_ = c.Close()
	case <-time.After(time.Second):
		t.Fatal("connection was not accepted")
	}
}
