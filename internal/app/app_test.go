package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/solar-mc/solar/internal/config"
)

func TestRunVersionCommand(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"version"})
	if err != nil {
		t.Fatalf("Run version returned error: %v", err)
	}
}

func TestRunHelpCommand(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("Run help returned error: %v", err)
	}
}

func TestRunNoArgsReturnsHelp(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run with no args returned error: %v", err)
	}
}

func TestRunUnknownCommandReturnsError(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRunLoadtestWithInvalidClients(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"loadtest", "--clients", "0"})
	if err == nil {
		t.Fatal("expected error for zero clients")
	}
}

func TestLoadBlockDefinitionsAllowsMissingFile(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		DataDir: t.TempDir(),
		Storage: config.StorageConfig{BlockDefsDir: "blockdefs"},
	}
	registry, err := loadBlockDefinitions(cfg)
	if err != nil {
		t.Fatalf("loadBlockDefinitions returned error: %v", err)
	}
	if registry == nil {
		t.Fatal("loadBlockDefinitions returned a nil registry")
	}
}

func TestLoadBlockDefinitionsRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	blockDefinitionsDir := filepath.Join(dataDir, "blockdefs")
	if err := os.MkdirAll(blockDefinitionsDir, 0o755); err != nil {
		t.Fatalf("create block definitions directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(blockDefinitionsDir, "global.json"), []byte("{"), 0o644); err != nil {
		t.Fatalf("write invalid block definitions: %v", err)
	}

	cfg := config.Config{
		DataDir: dataDir,
		Storage: config.StorageConfig{BlockDefsDir: "blockdefs"},
	}
	if _, err := loadBlockDefinitions(cfg); err == nil {
		t.Fatal("expected invalid block definitions to fail startup")
	}
}

func TestBuildServerWiresBootstrapSystems(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg := loadBootstrapTestConfig(t)
	srv, err := buildServer(ctx, cfg)
	if err != nil {
		t.Fatalf("buildServer returned error: %v", err)
	}
	if srv == nil {
		t.Fatal("buildServer returned nil server")
	}
}

func TestLoadTranslationsFallbackForMissingFile(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	translations := loadTranslations(config.Config{
		Language: config.LanguageConfig{Default: "en", File: filepath.Join(t.TempDir(), "missing.toml")},
	}, logger)
	if translations.Get("en", "missing.key") != "missing.key" {
		t.Fatal("missing translation did not fall back to key")
	}
}

func loadBootstrapTestConfig(t *testing.T) config.Config {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	if err := os.WriteFile(path, []byte(fmt.Sprintf(`
listen = "127.0.0.1:0"
data_dir = %q
workers = 1
max_players = 4
`, dataDir)), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("load test config: %v", err)
	}
	return cfg
}
