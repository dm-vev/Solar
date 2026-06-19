package network

import (
	"context"
	"testing"
	"time"
)

func TestListenerStopsWhenContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	listener := NewListener("127.0.0.1:0")
	listener.SetConnectRate(1000)
	done := make(chan error, 1)
	go func() {
		done <- listener.Serve(ctx, nil)
	}()
	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve returned error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not stop after context cancellation")
	}
}
