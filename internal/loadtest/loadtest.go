package loadtest

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// Config controls the load test harness.
type Config struct {
	Address        string
	Clients        int
	Duration       time.Duration
	UsernamePrefix string
	Scenario       string
	CPE            bool
	Logger         *slog.Logger
}

// Run connects multiple clients to the server and keeps them online for the duration.
func Run(ctx context.Context, cfg Config) error {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:25565"
	}
	if cfg.Clients < 1 {
		cfg.Clients = 1
	}
	if cfg.UsernamePrefix == "" {
		cfg.UsernamePrefix = "bot"
	}
	if cfg.Scenario == "" {
		cfg.Scenario = "idle"
	}
	if cfg.Duration < 0 {
		cfg.Duration = 0
	}

	runCtx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	logger.Info("loadtest starting",
		"address", cfg.Address,
		"clients", cfg.Clients,
		"duration", cfg.Duration.String(),
		"scenario", cfg.Scenario,
	)

	var successes int64
	var failures int64
	var wg sync.WaitGroup
	wg.Add(cfg.Clients)

	for i := 0; i < cfg.Clients; i++ {
		go func() {
			defer wg.Done()
			if err := runClient(runCtx, cfg, i); err != nil {
				atomic.AddInt64(&failures, 1)
				return
			}
			atomic.AddInt64(&successes, 1)
		}()
	}

	wg.Wait()

	logger.Info("loadtest complete",
		"connected", atomic.LoadInt64(&successes),
		"failed", atomic.LoadInt64(&failures),
	)
	if failures > 0 {
		return fmt.Errorf("loadtest failed: %d/%d clients could not join", failures, cfg.Clients)
	}
	return nil
}

func dialWithContext(ctx context.Context, address string) (net.Conn, error) {
	d := net.Dialer{Timeout: 5 * time.Second}
	return d.DialContext(ctx, "tcp", address)
}
