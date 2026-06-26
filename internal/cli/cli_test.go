package cli

import (
	"strings"
	"testing"
)

func TestParseStartConfig(t *testing.T) {
	cmd, err := Parse([]string{"start", "--config", "tmp/server.toml"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Name != "start" {
		t.Fatalf("Name = %q, want start", cmd.Name)
	}
	if cmd.ConfigPath != "tmp/server.toml" {
		t.Fatalf("ConfigPath = %q, want tmp/server.toml", cmd.ConfigPath)
	}
}

func TestParseStartPprof(t *testing.T) {
	cmd, err := Parse([]string{"start", "--pprof", "127.0.0.1:6060"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.PprofAddress != "127.0.0.1:6060" {
		t.Fatalf("PprofAddress = %q, want 127.0.0.1:6060", cmd.PprofAddress)
	}
}

func TestParseLoadTest(t *testing.T) {
	cmd, err := Parse([]string{
		"loadtest",
		"--address",
		"127.0.0.1:25566",
		"--clients",
		"8",
		"--duration",
		"15s",
		"--prefix",
		"bot",
		"--scenario",
		"mixed",
		"--cpe",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Name != "loadtest" {
		t.Fatalf("Name = %q, want loadtest", cmd.Name)
	}
	if cmd.Address != "127.0.0.1:25566" {
		t.Fatalf("Address = %q, want 127.0.0.1:25566", cmd.Address)
	}
	if cmd.Clients != 8 {
		t.Fatalf("Clients = %d, want 8", cmd.Clients)
	}
	if cmd.Duration.String() != "15s" {
		t.Fatalf("Duration = %s, want 15s", cmd.Duration)
	}
	if cmd.UsernamePrefix != "bot" {
		t.Fatalf("UsernamePrefix = %q, want bot", cmd.UsernamePrefix)
	}
	if cmd.Scenario != "mixed" {
		t.Fatalf("Scenario = %q, want mixed", cmd.Scenario)
	}
	if !cmd.CPE {
		t.Fatal("CPE = false, want true")
	}
}

func TestHelpIncludesAllCommands(t *testing.T) {
	help := Help()
	for _, want := range []string{
		"solar start",
		"solar loadtest",
		"solar version",
		"solar help",
	} {
		if !strings.Contains(help, want) {
			t.Fatalf("Help() missing %q in %q", want, help)
		}
	}
}
