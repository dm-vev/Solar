//go:build lua

package plugin_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/solar-mc/solar/plugin"
)

// TestLoadLuaScripts loads the example.lua script and verifies it
// registers as a plugin.
func TestLoadLuaScripts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping lua integration test in short mode")
	}

	tmp := t.TempDir()
	src, err := os.ReadFile("../plugins/example.lua")
	if err != nil {
		t.Fatalf("read example.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "example.lua"), src, 0o644); err != nil {
		t.Fatalf("write lua file: %v", err)
	}

	if err := plugin.LoadLuaScripts(tmp, slog.Default()); err != nil {
		t.Fatalf("LoadLuaScripts: %v", err)
	}

	for _, p := range plugin.Registered() {
		if p.Name() == "example" {
			if err := p.Init(); err != nil {
				t.Fatalf("Init: %v", err)
			}
			return
		}
	}
	t.Fatalf("example lua plugin not registered")
}
