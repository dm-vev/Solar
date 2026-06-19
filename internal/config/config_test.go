package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadParsesConfigFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"workers = 3\n" +
			"max_players = 24\n" +
			"connect_rate = 9\n" +
			"autosave_interval = \"15s\"\n" +
			"server_name = \"Test Solar\"\n" +
			"motd = \"Test MOTD\"\n" +
			"operators = [\"alice\", \"bob\"]\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ListenAddress != "127.0.0.1:25566" {
		t.Fatalf("ListenAddress = %q", cfg.ListenAddress)
	}
	if cfg.DataDir != dataDir {
		t.Fatalf("DataDir = %q", cfg.DataDir)
	}
	if cfg.Workers != 3 {
		t.Fatalf("Workers = %d", cfg.Workers)
	}
	if cfg.MaxPlayers != 24 {
		t.Fatalf("MaxPlayers = %d", cfg.MaxPlayers)
	}
	if cfg.ConnectRate != 9 {
		t.Fatalf("ConnectRate = %d", cfg.ConnectRate)
	}
	if cfg.Autosave != 15*time.Second {
		t.Fatalf("Autosave = %s", cfg.Autosave)
	}
	if cfg.DefaultGenerator != "Classic" {
		t.Fatalf("DefaultGenerator = %q, want Classic", cfg.DefaultGenerator)
	}
	if cfg.Name != "Test Solar" || cfg.MOTD != "Test MOTD" {
		t.Fatalf("Name/MOTD = %q/%q", cfg.Name, cfg.MOTD)
	}
	if len(cfg.Operators) != 2 || cfg.Operators[0] != "alice" || cfg.Operators[1] != "bob" {
		t.Fatalf("Operators = %#v", cfg.Operators)
	}
}

func TestLoadParsesDefaultGenerator(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"default_generator = \"Hills\"\n" +
			"server_name = \"Test Solar\"\n" +
			"motd = \"Test MOTD\"\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DefaultGenerator != "Hills" {
		t.Fatalf("DefaultGenerator = %q, want Hills", cfg.DefaultGenerator)
	}
}

func TestLoadRejectsUnknownConfigKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.toml")
	if err := os.WriteFile(path, []byte("unknown = \"value\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for unknown key")
	}
}

func TestAutosaveIntervalZeroIsAllowed(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.toml")
	contents := []byte("listen = \"127.0.0.1:25566\"\n" +
		"data_dir = \"" + t.TempDir() + "\"\n" +
		"autosave_interval = \"0s\"\n" +
		"server_name = \"Solar\"\n" +
		"motd = \"Test\"\n")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Autosave != 0 {
		t.Fatalf("Autosave = %s, want 0", cfg.Autosave)
	}
}

func TestAutosaveIntervalRejectsNegative(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.toml")
	contents := []byte("listen = \"127.0.0.1:25566\"\n" +
		"data_dir = \"" + t.TempDir() + "\"\n" +
		"autosave_interval = \"-1s\"\n" +
		"server_name = \"Solar\"\n" +
		"motd = \"Test\"\n")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for negative autosave interval")
	}
}
