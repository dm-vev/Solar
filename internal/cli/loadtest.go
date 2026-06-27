package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"
)

func parseLoadTest(args []string) (Command, error) {
	fs := flag.NewFlagSet("loadtest", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	address := fs.String("address", "127.0.0.1:25565", "server address")
	fs.StringVar(address, "addr", "127.0.0.1:25565", "server address")
	clients := fs.Int("clients", 16, "client count")
	duration := fs.Duration("duration", 30*time.Second, "test duration")
	prefix := fs.String("prefix", "bot", "username prefix")
	scenario := fs.String("scenario", "idle", "scenario name")
	cpe := fs.Bool("cpe", false, "enable CPE")
	authSalt := fs.String("auth-salt", "", "Classic mppass salt")
	if err := fs.Parse(args[1:]); err != nil {
		return Command{}, fmt.Errorf("%w: loadtest [--address host:port] [--clients n] [--duration 30s] [--prefix bot] [--scenario idle|chat|move|blocks|mixed] [--cpe] [--auth-salt salt]", ErrUsage)
	}
	parsedScenario := strings.ToLower(strings.TrimSpace(*scenario))
	switch parsedScenario {
	case "", "idle", "chat", "move", "blocks", "mixed":
		if parsedScenario == "" {
			parsedScenario = "idle"
		}
	default:
		return Command{}, fmt.Errorf("%w: loadtest --scenario idle|chat|move|blocks|mixed", ErrUsage)
	}
	if *clients < 1 {
		return Command{}, fmt.Errorf("%w: loadtest --clients n", ErrUsage)
	}
	if *duration < 0 {
		return Command{}, fmt.Errorf("%w: loadtest --duration 30s", ErrUsage)
	}
	return Command{
		Name:           "loadtest",
		Address:        *address,
		Clients:        *clients,
		Duration:       *duration,
		UsernamePrefix: *prefix,
		Scenario:       parsedScenario,
		CPE:            *cpe,
		AuthSalt:       *authSalt,
	}, nil
}
