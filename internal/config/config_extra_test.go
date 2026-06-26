package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerAndFormatStringSlice(t *testing.T) {
	var buf bytes.Buffer
	logger := Config{Log: LogConfig{Level: "debug", Format: "json"}}.Logger(&buf)
	logger.Debug("debug message")
	if !strings.Contains(buf.String(), "debug message") {
		t.Fatalf("debug json log missing message: %q", buf.String())
	}

	buf.Reset()
	logger = Config{Log: LogConfig{Level: "warn", Format: "text"}}.Logger(&buf)
	logger.Info("hidden")
	logger.Warn("visible")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "visible") {
		t.Fatalf("warn text log = %q", buf.String())
	}

	for _, level := range []string{"error", "info", "unknown"} {
		buf.Reset()
		logger = Config{Log: LogConfig{Level: level}}.Logger(&buf)
		logger.LogAttrs(nil, slog.LevelError, "error message")
		if !strings.Contains(buf.String(), "error message") {
			t.Fatalf("%s logger did not emit error: %q", level, buf.String())
		}
	}

	if got := formatStringSlice(nil); got != "[]" {
		t.Fatalf("formatStringSlice nil = %q", got)
	}
	if got := formatStringSlice([]string{"alice", "bob"}); got != `["alice", "bob"]` {
		t.Fatalf("formatStringSlice = %q", got)
	}
}

func TestValidateRejectsEscapingStoragePaths(t *testing.T) {
	t.Parallel()

	cfg := minimalValidConfig(t)
	cfg.Storage.MainWorldName = "../main"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted escaping main world name")
	}

	cfg = minimalValidConfig(t)
	cfg.Storage.WorldsDir = "../worlds"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted escaping worlds dir")
	}

	cfg = minimalValidConfig(t)
	cfg.Storage.PolicyFile = "players/policy.json"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted policy file path")
	}

	cfg = minimalValidConfig(t)
	cfg.Storage.PlayersDir = `players\..\evil`
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate accepted backslash storage path")
	}
}

func minimalValidConfig(t *testing.T) Config {
	t.Helper()
	return Config{
		ListenAddress:    "127.0.0.1:25565",
		DataDir:          t.TempDir(),
		Workers:          1,
		MaxPlayers:       8,
		ConnectRate:      1,
		DefaultGenerator: "Classic",
		Name:             "Solar",
		MOTD:             "Test",
		Network: NetworkConfig{
			SessionOutbox:   1,
			WriteBatchSize:  1,
			SendTimeoutMode: "fixed",
		},
		Storage: StorageConfig{
			Backend:       "local",
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
			BlockDefsDir:  "blockdefs",
		},
		Player: PlayerConfig{MaxUsernameLength: 32},
		Log:    LogConfig{Level: "info", Format: "text"},
	}
}
