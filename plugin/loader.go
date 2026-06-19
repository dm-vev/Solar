//go:build linux || darwin

package plugin

import (
	"log/slog"
	"os"
	"path/filepath"
	"plugin"
	"runtime"
	"sort"
	"strings"
)

// LoadDirectory opens all *.so files in dir. Each .so's init() should
// call plugin.Register to add itself to the registry.
//
// Files are loaded in sorted (alphabetical) order for deterministic
// behaviour. Errors loading individual files (including panics from
// init()) are logged but do not stop loading others.
//
// The server's Go version is logged before loading so version mismatches
// (the most common .so failure) are easier to diagnose.
func LoadDirectory(dir string, logger *slog.Logger) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("plugin directory does not exist, skipping", "dir", dir)
			return nil
		}
		return err
	}

	// Collect and sort .so filenames for deterministic load order.
	var soFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".so") {
			soFiles = append(soFiles, entry.Name())
		}
	}
	sort.Strings(soFiles)

	if len(soFiles) == 0 {
		return nil
	}

	logger.Info("loading .so plugins",
		"dir", dir,
		"count", len(soFiles),
		"go_version", runtime.Version(),
	)

	for _, name := range soFiles {
		path := filepath.Join(dir, name)
		func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("plugin panicked during load", "path", path, "panic", r)
				}
			}()
			if _, err := plugin.Open(path); err != nil {
				hint := ""
				if strings.Contains(err.Error(), "different version") {
					hint = "rebuild with same Go version: go build -buildmode=plugin -o <file> <source>"
				}
				logger.Error("plugin load failed", "path", path, "error", err, "hint", hint)
				return
			}
			logger.Info("plugin loaded", "path", path)
		}()
	}
	return nil
}
