//go:build (linux || darwin) && !race && !lua

package plugin_test

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/solar-mc/solar/plugin"
)

// TestLoadDirectory builds multiple .so plugins, loads them from one directory,
// and verifies that they register together.
func TestLoadDirectory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping .so integration test in short mode")
	}

	tmp := t.TempDir()
	plugins := map[string]string{
		"soplug":           "./plugins/soplug",
		"mcgalaxy-chat":    "./plugins/mcgalaxy/chat",
		"mcgalaxy-cpe":     "./plugins/mcgalaxy/cpe",
		"mcgalaxy-economy": "./plugins/mcgalaxy/economy",
	}
	for name, source := range plugins {
		cmd := exec.Command("go", "build", "-buildmode=plugin", "-tags=plugin",
			"-o", filepath.Join(tmp, name+".so"), source)
		cmd.Dir = ".." // repo root
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("build %s.so: %v", name, err)
		}
	}

	if err := plugin.LoadDirectory(tmp, slog.Default()); err != nil {
		t.Fatalf("LoadDirectory: %v", err)
	}

	registered := make(map[string]bool)
	for _, loaded := range plugin.Registered() {
		registered[loaded.Name()] = true
	}
	for name := range plugins {
		if !registered[name] {
			t.Errorf("%s not registered after LoadDirectory", name)
		}
	}
}
