package network

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// Listener owns the TCP accept loop.
type Listener struct {
	address     string
	connectRate int
}

// NewListener creates a Classic-compatible TCP listener.
func NewListener(address string) *Listener {
	return &Listener{address: address, connectRate: 32}
}

// SetConnectRate configures the maximum accepted connections per second.
func (l *Listener) SetConnectRate(rate int) {
	if rate < 1 {
		rate = 1
	}
	l.connectRate = rate
}

// Serve starts accepting clients until the context is canceled.
func (l *Listener) Serve(ctx context.Context, handle func(net.Conn)) error {
	ln, err := net.Listen("tcp", l.address)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", l.address, err)
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	limiter := time.NewTicker(time.Second / time.Duration(l.connectRate))
	defer limiter.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-limiter.C:
		}

		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("accept connection: %w", err)
		}
		handle(conn)
	}
}
