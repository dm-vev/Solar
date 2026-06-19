//go:build !lua

package plugin

import "log/slog"

// LoadLuaScripts is a no-op when the server is built without -tags=lua.
// The gopher-lua dependency is excluded from the binary entirely.
func LoadLuaScripts(dir string, logger *slog.Logger) error {
	logger.Warn("lua scripting not compiled in; build with -tags=lua to enable",
		"dir", dir)
	return nil
}
