//go:build !(linux || darwin)

package plugin

import (
	"log/slog"
	"runtime"
)

// LoadDirectory is a no-op on platforms without Go plugin support.
// Logs a warning so the operator knows .so plugins were requested but
// cannot be loaded. Does not return an error — the server continues.
func LoadDirectory(dir string, logger *slog.Logger) error {
	logger.Warn(".so plugin loading is not supported on this platform; "+
		"plugins will not be loaded",
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"dir", dir,
	)
	return nil
}
