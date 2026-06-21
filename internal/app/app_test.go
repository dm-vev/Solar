package app

import (
	"context"
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
