//go:build linux || darwin

package plugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"strings"
)

// LoadDirectory opens all *.so files in dir. Each .so's init() should
// call plugin.Register to add itself to the registry.
// Errors loading individual files are logged but do not stop loading others.
func LoadDirectory(dir string, logger *slog.Logger) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".so") {
			continue
		}
		path := filepath.Join(dir, name)
		if _, err := plugin.Open(path); err != nil {
			logger.Error("plugin load failed", "path", path, "error", err)
			continue
		}
		logger.Info("plugin loaded", "path", path)
	}
	return nil
}
