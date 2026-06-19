//go:build !(linux || darwin)

package plugin

import (
	"fmt"
	"log/slog"
	"runtime"
)

// LoadDirectory is a stub for platforms without Go plugin support.
func LoadDirectory(_ string, logger *slog.Logger) error {
	logger.Warn(".so plugin loading not supported on this platform",
		"os", runtime.GOOS, "arch", runtime.GOARCH)
	return fmt.Errorf("plugin loading not supported on %s/%s",
		runtime.GOOS, runtime.GOARCH)
}
