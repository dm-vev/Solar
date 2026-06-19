package app

import (
	"context"
	"fmt"
	"os"

	"github.com/solar-mc/solar/internal/cli"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/loadtest"
)

// Run executes the CLI entrypoint.
func Run(ctx context.Context, args []string) error {
	cmd, err := cli.Parse(args)
	if err != nil {
		return err
	}

	switch cmd.Name {
	case "start":
		cfg, err := config.Load(cmd.ConfigPath)
		if err != nil {
			return err
		}
		srv := buildServer(ctx, cfg)
		if cmd.PprofAddress != "" {
			srv.SetPprofAddress(cmd.PprofAddress)
		}
		return srv.Run(ctx)
	case "loadtest":
		return loadtest.Run(ctx, loadtest.Config{
			Address:        cmd.Address,
			Clients:        cmd.Clients,
			Duration:       cmd.Duration,
			UsernamePrefix: cmd.UsernamePrefix,
			Scenario:       cmd.Scenario,
			CPE:            cmd.CPE,
		})
	case "version":
		_, _ = fmt.Fprintln(os.Stdout, "solar pre-alpha")
		return nil
	case "help":
		_, _ = fmt.Fprint(os.Stdout, cli.Help())
		return nil
	default:
		return fmt.Errorf("unsupported command %q", cmd.Name)
	}
}
