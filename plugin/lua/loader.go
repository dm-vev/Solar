//go:build lua

package lua

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
)

// LoadLuaScripts loads all *.lua files from dir and registers each as
// a plugin. Each file becomes a separate luaPlugin with its own Lua state.
func LoadLuaScripts(dir string, logger *slog.Logger) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("lua script directory does not exist, skipping", "dir", dir)
			return nil
		}
		return err
	}

	var luaFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.HasSuffix(entry.Name(), ".lua") {
			luaFiles = append(luaFiles, entry.Name())
		}
	}
	sort.Strings(luaFiles)

	if len(luaFiles) == 0 {
		return nil
	}

	logger.Info("loading lua scripts", "dir", dir, "count", len(luaFiles))

	for _, name := range luaFiles {
		path := filepath.Join(dir, name)
		pluginName := strings.TrimSuffix(name, ".lua")
		plugin.Register(pluginName, &luaPlugin{
			name: pluginName,
			path: path,
		})
		logger.Info("lua script registered", "path", path, "name", pluginName)
	}
	return nil
}

// luaPlugin implements plugin.Plugin by running a Lua script.
type luaPlugin struct {
	name string
	path string
	L    *glua.LState
}

func (p *luaPlugin) Name() string { return p.name }
func (p *luaPlugin) Init() error  { return nil }

func (p *luaPlugin) Enable(s plugin.Server) error {
	p.L = glua.NewState()
	api := newAPI(p.L, s)
	api.register()
	if err := p.L.DoFile(p.path); err != nil {
		return fmt.Errorf("lua %s: %w", p.name, err)
	}
	return nil
}

func (p *luaPlugin) Disable() error {
	if p.L != nil {
		p.L.Close()
		p.L = nil
	}
	return nil
}
