package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"time"
)

// ErrUsage is returned when the CLI arguments are invalid or incomplete.
var ErrUsage = errors.New("usage")

// Command is the parsed CLI command.
type Command struct {
	Name           string
	ConfigPath     string
	Address        string
	Clients        int
	Duration       time.Duration
	UsernamePrefix string
	Scenario       string
	CPE            bool
}

// Parse converts argv into a command.
func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{Name: "help"}, nil
	}

	switch args[0] {
	case "start":
		fs := flag.NewFlagSet("start", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		configPath := fs.String("config", "configs/server.toml", "config path")
		if err := fs.Parse(args[1:]); err != nil {
			return Command{}, fmt.Errorf("%w: start [--config path]", ErrUsage)
		}
		return Command{Name: "start", ConfigPath: *configPath}, nil
	case "loadtest":
		return parseLoadTest(args)
	case "version":
		return Command{Name: "version"}, nil
	case "help", "--help", "-h":
		return Command{Name: "help"}, nil
	default:
		return Command{}, fmt.Errorf("%w: unknown command %q", ErrUsage, args[0])
	}
}

// Help returns the CLI usage text.
func Help() string {
	return "usage:\n  solar start [--config path]\n  solar loadtest [--address host:port] [--clients n] [--duration 30s] [--prefix bot] [--scenario idle|chat|move|blocks|mixed] [--cpe]\n  solar version\n  solar help\n"
}
