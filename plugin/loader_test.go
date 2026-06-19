//go:build (linux || darwin) && !race

package plugin_test

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/solar-mc/solar/plugin"
)

// TestLoadDirectory builds the soplug example as a .so, loads it via
// LoadDirectory, and verifies the plugin registered itself.
func TestLoadDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping .so integration test in short mode")
	}

	tmp := t.TempDir()
	soPath := filepath.Join(tmp, "soplug.so")

	cmd := exec.Command("go", "build", "-buildmode=plugin", "-tags=plugin",
		"-o", soPath, "./plugins/soplug")
	cmd.Dir = ".." // repo root
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("build .so: %v", err)
	}

	if err := plugin.LoadDirectory(tmp, slog.Default()); err != nil {
		t.Fatalf("LoadDirectory: %v", err)
	}

	for _, p := range plugin.Registered() {
		if p.Name() == "soplug" {
			return
		}
	}
	t.Fatalf("soplug not registered after LoadDirectory")
}
