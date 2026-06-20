package config

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds the server bootstrap settings.
type Config struct {
	ListenAddress    string         `toml:"listen"`
	DataDir          string         `toml:"data_dir"`
	Workers          int            `toml:"workers"`
	MaxPlayers       int            `toml:"max_players"`
	ConnectRate      int            `toml:"connect_rate"`
	Autosave         time.Duration  `toml:"autosave_interval"`
	DefaultGenerator string         `toml:"default_generator"`
	Name             string         `toml:"server_name"`
	MOTD             string         `toml:"motd"`
	Operators        []string       `toml:"operators"`
	Network          NetworkConfig  `toml:"network"`
	Simulation       SimConfig      `toml:"simulation"`
	World            WorldConfig    `toml:"world"`
	Storage          StorageConfig  `toml:"storage"`
	Commands         CommandsConfig `toml:"commands"`
	Player           PlayerConfig   `toml:"player"`
	CPE              CPEConfig      `toml:"cpe"`
	Debug            DebugConfig    `toml:"debug"`
	Log              LogConfig      `toml:"log"`
	Plugins          PluginsConfig  `toml:"plugins"`
	Lua              LuaConfig      `toml:"lua"`
	Language         LanguageConfig `toml:"language"`
}

// NetworkConfig controls per-session TCP I/O tuning.
type NetworkConfig struct {
	ReadTimeout     time.Duration `toml:"read_timeout"`
	WriteTimeout    time.Duration `toml:"write_timeout"`
	TCPNoDelay      bool          `toml:"tcp_nodelay"`
	SessionOutbox   int           `toml:"session_outbox_size"`
	WriteBatchSize  int           `toml:"write_batch_size"`
	SendTimeout     time.Duration `toml:"send_timeout"`
	SendTimeoutMode string        `toml:"send_timeout_mode"`
}

// SimConfig controls the world/entity simulation loop.
type SimConfig struct {
	TickInterval time.Duration `toml:"tick_interval"`
}

// WorldConfig controls default world generation and safety limits.
type WorldConfig struct {
	DefaultWidth  int `toml:"default_width"`
	DefaultHeight int `toml:"default_height"`
	DefaultLength int `toml:"default_length"`
	MaxBlocks     int `toml:"max_blocks"`
}

// DebugConfig controls the pprof/health HTTP server.
type DebugConfig struct {
	PprofAddress         string        `toml:"pprof_address"`
	PprofShutdownTimeout time.Duration `toml:"pprof_shutdown_timeout"`
}

// LogConfig controls structured logging output.
type LogConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

// StorageConfig controls world/player persistence layout. Backend is
// forward-compatible: only "local" is implemented today.
type StorageConfig struct {
	Backend       string `toml:"backend"`
	WorldsDir     string `toml:"worlds_dir"`
	PlayersDir    string `toml:"players_dir"`
	PolicyFile    string `toml:"policy_file"`
	WorldFileExt  string `toml:"world_file_ext"`
	MainWorldName string `toml:"main_world_name"`
	BlockDefsDir  string `toml:"blockdefs_dir"`
}

// CommandsConfig controls command permissions.
type CommandsConfig struct {
	AdminCommands []string `toml:"admin_commands"`
}

// PlayerConfig controls player policy defaults and validation.
type PlayerConfig struct {
	WhitelistEnabled  bool `toml:"whitelist_enabled"`
	MaxUsernameLength int  `toml:"max_username_length"`
}

// CPEConfig controls which CPE extensions the server advertises.
type CPEConfig struct {
	ExtPlayerList bool `toml:"ext_player_list"`
	FastMap       bool `toml:"fast_map"`
	TwoWayPing    bool `toml:"two_way_ping"`
}

// PluginsConfig controls runtime .so plugin loading.
// The server scans dir (relative to data_dir) for *.so files and opens
// each with the Go plugin package. Each .so's init() should call
// plugin.Register to add itself to the registry.
// ponytail: Go's plugin package requires the main binary and .so to be
// built with the same Go version and dependency versions.
type PluginsConfig struct {
	Enabled bool   `toml:"enabled"`
	Dir     string `toml:"dir"`
}

// LuaConfig controls Lua script loading via gopher-lua.
// Requires the server binary to be built with -tags=lua.
// Without the tag, LoadLuaScripts is a no-op and this config is ignored.
type LuaConfig struct {
	Enabled bool   `toml:"enabled"`
	Dir     string `toml:"dir"`
}

// LanguageConfig controls server-side internationalisation.
type LanguageConfig struct {
	Default string `toml:"default"`
	File    string `toml:"file"`
}

// Load returns bootstrap config and ensures base directories exist.
func Load(path string) (Config, error) {
	cfg := Config{
		ListenAddress:    ":25565",
		DataDir:          "data",
		Workers:          runtime.NumCPU(),
		MaxPlayers:       128,
		ConnectRate:      32,
		Autosave:         60 * time.Second,
		DefaultGenerator: "Classic",
		Name:             "Solar",
		MOTD:             "CLI-only classic server",
		Operators:        parseOperatorsEnv(),
		Network: NetworkConfig{
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    10 * time.Second,
			TCPNoDelay:      true,
			SessionOutbox:   256,
			WriteBatchSize:  32,
			SendTimeout:     50 * time.Millisecond,
			SendTimeoutMode: "fixed",
		},
		Simulation: SimConfig{
			TickInterval: 50 * time.Millisecond,
		},
		World: WorldConfig{
			DefaultWidth:  128,
			DefaultHeight: 64,
			DefaultLength: 128,
			MaxBlocks:     64 * 1024 * 1024,
		},
		Debug: DebugConfig{
			PprofAddress:         "",
			PprofShutdownTimeout: 5 * time.Second,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
		Storage: StorageConfig{
			Backend:       "local",
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
			BlockDefsDir:  "blockdefs",
		},
		Commands: CommandsConfig{
			AdminCommands: []string{"tp", "setspawn", "save", "kick", "ban", "unban", "whitelist", "newlvl", "gb", "lb"},
		},
		Player: PlayerConfig{
			WhitelistEnabled:  false,
			MaxUsernameLength: 32,
		},
		CPE: CPEConfig{
			ExtPlayerList: true,
			FastMap:       true,
			TwoWayPing:    true,
		},
		Plugins: PluginsConfig{
			Enabled: false,
			Dir:     "plugins",
		},
		Lua: LuaConfig{
			Enabled: false,
			Dir:     "lua",
		},
		Language: LanguageConfig{
			Default: "en",
			File:    "configs/language.toml",
		},
	}

	if err := loadFile(path, &cfg); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Config{}, fmt.Errorf("create config directory: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return Config{}, fmt.Errorf("create data directory: %w", err)
	}
	if err := ensurePlaceholder(path, cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Validate checks the configuration for basic runtime safety.
func (c Config) Validate() error {
	if strings.TrimSpace(c.ListenAddress) == "" {
		return fmt.Errorf("listen address is empty")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("data directory is empty")
	}
	if err := validateDataDir(c.DataDir); err != nil {
		return err
	}
	if c.Workers < 1 {
		return fmt.Errorf("workers must be at least 1")
	}
	if c.MaxPlayers < 1 {
		return fmt.Errorf("max players must be at least 1")
	}
	if c.ConnectRate < 1 {
		return fmt.Errorf("connect rate must be at least 1")
	}
	if c.Autosave < 0 {
		return fmt.Errorf("autosave interval cannot be negative")
	}
	if strings.TrimSpace(c.Name) == "" {
		return fmt.Errorf("server name is empty")
	}
	if strings.TrimSpace(c.MOTD) == "" {
		return fmt.Errorf("motd is empty")
	}
	if strings.TrimSpace(c.DefaultGenerator) == "" {
		return fmt.Errorf("default generator is empty")
	}
	if c.Network.ReadTimeout < 0 {
		return fmt.Errorf("network.read_timeout cannot be negative")
	}
	if c.Network.WriteTimeout < 0 {
		return fmt.Errorf("network.write_timeout cannot be negative")
	}
	if c.Network.SessionOutbox < 1 {
		return fmt.Errorf("network.session_outbox_size must be at least 1")
	}
	if c.Network.WriteBatchSize < 1 {
		return fmt.Errorf("network.write_batch_size must be at least 1")
	}
	if c.Network.SendTimeout < 0 {
		return fmt.Errorf("network.send_timeout cannot be negative")
	}
	switch c.Network.SendTimeoutMode {
	case "fixed", "adaptive", "":
	default:
		return fmt.Errorf("network.send_timeout_mode %q is not one of fixed|adaptive", c.Network.SendTimeoutMode)
	}
	if c.Simulation.TickInterval < 0 {
		return fmt.Errorf("simulation.tick_interval cannot be negative")
	}
	if c.World.DefaultWidth < 1 || c.World.DefaultHeight < 1 || c.World.DefaultLength < 1 {
		return fmt.Errorf("world default dimensions must be positive")
	}
	if c.World.MaxBlocks < 1 {
		return fmt.Errorf("world.max_blocks must be at least 1")
	}
	if c.Debug.PprofShutdownTimeout < 0 {
		return fmt.Errorf("debug.pprof_shutdown_timeout cannot be negative")
	}
	switch c.Log.Level {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("log.level %q is not one of debug|info|warn|error", c.Log.Level)
	}
	switch c.Log.Format {
	case "text", "json":
	default:
		return fmt.Errorf("log.format %q is not one of text|json", c.Log.Format)
	}
	if c.Storage.Backend != "local" {
		return fmt.Errorf("storage.backend %q is not supported (only \"local\")", c.Storage.Backend)
	}
	if strings.TrimSpace(c.Storage.WorldsDir) == "" {
		return fmt.Errorf("storage.worlds_dir is empty")
	}
	if strings.TrimSpace(c.Storage.PlayersDir) == "" {
		return fmt.Errorf("storage.players_dir is empty")
	}
	if strings.TrimSpace(c.Storage.PolicyFile) == "" {
		return fmt.Errorf("storage.policy_file is empty")
	}
	if strings.TrimSpace(c.Storage.WorldFileExt) == "" {
		return fmt.Errorf("storage.world_file_ext is empty")
	}
	if strings.TrimSpace(c.Storage.MainWorldName) == "" {
		return fmt.Errorf("storage.main_world_name is empty")
	}
	if c.Player.MaxUsernameLength < 1 || c.Player.MaxUsernameLength > 64 {
		return fmt.Errorf("player.max_username_length must be 1..64")
	}
	return nil
}

// validateDataDir rejects relative paths that traverse above the working
// directory. Absolute paths are allowed for production deployments.
func validateDataDir(dir string) error {
	cleaned := filepath.Clean(dir)
	if !filepath.IsAbs(cleaned) {
		for _, part := range strings.Split(filepath.ToSlash(cleaned), "/") {
			if part == ".." {
				return fmt.Errorf("data directory escapes working tree: %s", dir)
			}
		}
	}
	return nil
}

func ensurePlaceholder(path string, cfg Config) error {
	_, err := os.Stat(path)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat config %s: %w", path, err)
	}

	contents := fmt.Sprintf(`listen = "%s"
data_dir = "%s"
workers = %d
max_players = %d
connect_rate = %d
autosave_interval = "%s"
default_generator = "%s"
server_name = "%s"
motd = "%s"
# operators = ["alice", "bob"]
# SOLAR_OPERATORS="alice,bob" also seeds admin usernames.

[network]
read_timeout = "%s"
write_timeout = "%s"
tcp_nodelay = %v
session_outbox_size = %d
write_batch_size = %d
send_timeout = "%s"
send_timeout_mode = "%s"

[simulation]
tick_interval = "%s"

[world]
default_width = %d
default_height = %d
default_length = %d
max_blocks = %d

[storage]
backend = "%s"
worlds_dir = "%s"
players_dir = "%s"
policy_file = "%s"
world_file_ext = "%s"
main_world_name = "%s"
blockdefs_dir = "%s"

[commands]
admin_commands = %s

[player]
whitelist_enabled = %v
max_username_length = %d

[cpe]
ext_player_list = %v
fast_map = %v
two_way_ping = %v

[debug]
pprof_address = "%s"
pprof_shutdown_timeout = "%s"

[log]
level = "%s"
format = "%s"
`, cfg.ListenAddress, cfg.DataDir, cfg.Workers, cfg.MaxPlayers,
		cfg.ConnectRate, cfg.Autosave.String(), cfg.DefaultGenerator,
		cfg.Name, cfg.MOTD,
		cfg.Network.ReadTimeout.String(), cfg.Network.WriteTimeout.String(),
		cfg.Network.TCPNoDelay, cfg.Network.SessionOutbox, cfg.Network.WriteBatchSize,
		cfg.Network.SendTimeout.String(), cfg.Network.SendTimeoutMode,
		cfg.Simulation.TickInterval.String(),
		cfg.World.DefaultWidth, cfg.World.DefaultHeight, cfg.World.DefaultLength, cfg.World.MaxBlocks,
		cfg.Storage.Backend, cfg.Storage.WorldsDir, cfg.Storage.PlayersDir,
		cfg.Storage.PolicyFile, cfg.Storage.WorldFileExt, cfg.Storage.MainWorldName,
		cfg.Storage.BlockDefsDir,
		formatStringSlice(cfg.Commands.AdminCommands),
		cfg.Player.WhitelistEnabled, cfg.Player.MaxUsernameLength,
		cfg.CPE.ExtPlayerList, cfg.CPE.FastMap, cfg.CPE.TwoWayPing,
		cfg.Debug.PprofAddress, cfg.Debug.PprofShutdownTimeout.String(),
		cfg.Log.Level, cfg.Log.Format)

	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	return nil
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}

	if len(data) == 0 {
		return nil
	}

	meta, err := toml.Decode(string(data), cfg)
	if err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		keys := make([]string, len(undecoded))
		for i, k := range undecoded {
			keys[i] = k.String()
		}
		return fmt.Errorf("parse config %s: unknown keys: %s", path, strings.Join(keys, ", "))
	}
	return nil
}

// Logger builds a *slog.Logger from the log config.
func (c Config) Logger(w io.Writer) *slog.Logger {
	var level slog.Level
	switch c.Log.Level {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}
	if c.Log.Format == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}

func parseOperatorsEnv() []string {
	raw := strings.TrimSpace(os.Getenv("SOLAR_OPERATORS"))
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv("SOLAR_ADMIN"))
	}
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	operators := make([]string, 0, len(parts))
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name != "" {
			operators = append(operators, name)
		}
	}
	return operators
}

func formatStringSlice(ss []string) string {
	if len(ss) == 0 {
		return "[]"
	}
	quoted := make([]string, len(ss))
	for i, s := range ss {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
