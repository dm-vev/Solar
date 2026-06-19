package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds the server bootstrap settings.
type Config struct {
	ListenAddress    string        `toml:"listen"`
	DataDir          string        `toml:"data_dir"`
	Workers          int           `toml:"workers"`
	MaxPlayers       int           `toml:"max_players"`
	ConnectRate      int           `toml:"connect_rate"`
	Autosave         time.Duration `toml:"autosave_interval"`
	DefaultGenerator string        `toml:"default_generator"`
	Name             string        `toml:"server_name"`
	MOTD             string        `toml:"motd"`
	Operators        []string      `toml:"operators"`
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
# SOLAR_OPERATORS="alice,bob" seeds admin usernames.
`, cfg.ListenAddress, cfg.DataDir, cfg.Workers, cfg.MaxPlayers,
		cfg.ConnectRate, cfg.Autosave.String(), cfg.DefaultGenerator,
		cfg.Name, cfg.MOTD)

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
