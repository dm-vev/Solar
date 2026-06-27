package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/entity"
)

func TestLoadParsesConfigFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"workers = 3\n" +
			"max_players = 24\n" +
			"connect_rate = 9\n" +
			"autosave_interval = \"15s\"\n" +
			"server_name = \"Test Solar\"\n" +
			"motd = \"Test MOTD\"\n" +
			"operators = [\"alice\", \"bob\"]\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.ListenAddress != "127.0.0.1:25566" {
		t.Fatalf("ListenAddress = %q", cfg.ListenAddress)
	}
	if cfg.DataDir != dataDir {
		t.Fatalf("DataDir = %q", cfg.DataDir)
	}
	if cfg.Workers != 3 {
		t.Fatalf("Workers = %d", cfg.Workers)
	}
	if cfg.MaxPlayers != 24 {
		t.Fatalf("MaxPlayers = %d", cfg.MaxPlayers)
	}
	if cfg.ConnectRate != 9 {
		t.Fatalf("ConnectRate = %d", cfg.ConnectRate)
	}
	if cfg.Autosave != 15*time.Second {
		t.Fatalf("Autosave = %s", cfg.Autosave)
	}
	if cfg.DefaultGenerator != "Classic" {
		t.Fatalf("DefaultGenerator = %q, want Classic", cfg.DefaultGenerator)
	}
	if cfg.Name != "Test Solar" || cfg.MOTD != "Test MOTD" {
		t.Fatalf("Name/MOTD = %q/%q", cfg.Name, cfg.MOTD)
	}
	if len(cfg.Operators) != 2 || cfg.Operators[0] != "alice" || cfg.Operators[1] != "bob" {
		t.Fatalf("Operators = %#v", cfg.Operators)
	}
}

func TestLoadParsesDefaultGenerator(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"default_generator = \"Hills\"\n" +
			"server_name = \"Test Solar\"\n" +
			"motd = \"Test MOTD\"\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DefaultGenerator != "Hills" {
		t.Fatalf("DefaultGenerator = %q, want Hills", cfg.DefaultGenerator)
	}
}

func TestLoadRejectsUnknownConfigKey(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.toml")
	if err := os.WriteFile(path, []byte("unknown = \"value\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for unknown key")
	}
}

func TestAutosaveIntervalZeroIsAllowed(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.toml")
	contents := []byte("listen = \"127.0.0.1:25566\"\n" +
		"data_dir = \"" + t.TempDir() + "\"\n" +
		"autosave_interval = \"0s\"\n" +
		"server_name = \"Solar\"\n" +
		"motd = \"Test\"\n")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Autosave != 0 {
		t.Fatalf("Autosave = %s, want 0", cfg.Autosave)
	}
}

func TestAutosaveIntervalRejectsNegative(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "server.toml")
	contents := []byte("listen = \"127.0.0.1:25566\"\n" +
		"data_dir = \"" + t.TempDir() + "\"\n" +
		"autosave_interval = \"-1s\"\n" +
		"server_name = \"Solar\"\n" +
		"motd = \"Test\"\n")
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for negative autosave interval")
	}
}

func TestValidateRejectsMaxPlayersAboveClassicLimit(t *testing.T) {
	t.Parallel()

	cfg := Config{
		ListenAddress:    "127.0.0.1:25565",
		DataDir:          t.TempDir(),
		Workers:          1,
		MaxPlayers:       int(entity.MaxClassicEntityID) + 1,
		ConnectRate:      1,
		DefaultGenerator: "Classic",
		Name:             "Solar",
		MOTD:             "Test",
		Network: NetworkConfig{
			SessionOutbox:   1,
			WriteBatchSize:  1,
			SendTimeoutMode: "fixed",
		},
		Storage: StorageConfig{
			Backend:       "local",
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
		},
		Player: PlayerConfig{MaxUsernameLength: 32},
		Log:    LogConfig{Level: "info", Format: "text"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate returned nil error for max_players above Classic limit")
	}
}

func TestLoadParsesNestedTables(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[network]\n" +
			"read_timeout = \"45s\"\n" +
			"write_timeout = \"15s\"\n" +
			"tcp_nodelay = false\n" +
			"session_outbox_size = 512\n" +
			"write_batch_size = 64\n" +
			"\n[simulation]\n" +
			"tick_interval = \"100ms\"\n" +
			"\n[world]\n" +
			"default_width = 256\n" +
			"default_height = 128\n" +
			"default_length = 256\n" +
			"max_blocks = 33554432\n" +
			"\n[storage]\n" +
			"backend = \"local\"\n" +
			"worlds_dir = \"levels\"\n" +
			"players_dir = \"userdb\"\n" +
			"policy_file = \"bans.json\"\n" +
			"world_file_ext = \"bin\"\n" +
			"main_world_name = \"spawn\"\n" +
			"\n[commands]\n" +
			"admin_commands = [\"tp\", \"save\", \"ban\"]\n" +
			"\n[player]\n" +
			"whitelist_enabled = true\n" +
			"max_username_length = 16\n" +
			"\n[cpe]\n" +
			"ext_player_list = false\n" +
			"fast_map = false\n" +
			"two_way_ping = false\n" +
			"\n[auth]\n" +
			"enabled = true\n" +
			"salt = \"1234567890abcdef\"\n" +
			"\n[debug]\n" +
			"pprof_address = \"127.0.0.1:6060\"\n" +
			"pprof_shutdown_timeout = \"10s\"\n" +
			"\n[log]\n" +
			"level = \"debug\"\n" +
			"format = \"json\"\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Network.ReadTimeout != 45*time.Second {
		t.Fatalf("Network.ReadTimeout = %s", cfg.Network.ReadTimeout)
	}
	if cfg.Network.WriteTimeout != 15*time.Second {
		t.Fatalf("Network.WriteTimeout = %s", cfg.Network.WriteTimeout)
	}
	if cfg.Network.TCPNoDelay {
		t.Fatal("Network.TCPNoDelay should be false")
	}
	if cfg.Network.SessionOutbox != 512 {
		t.Fatalf("Network.SessionOutbox = %d", cfg.Network.SessionOutbox)
	}
	if cfg.Network.WriteBatchSize != 64 {
		t.Fatalf("Network.WriteBatchSize = %d", cfg.Network.WriteBatchSize)
	}
	if cfg.Simulation.TickInterval != 100*time.Millisecond {
		t.Fatalf("Simulation.TickInterval = %s", cfg.Simulation.TickInterval)
	}
	if cfg.World.DefaultWidth != 256 || cfg.World.DefaultHeight != 128 || cfg.World.DefaultLength != 256 {
		t.Fatalf("World dims = %d/%d/%d", cfg.World.DefaultWidth, cfg.World.DefaultHeight, cfg.World.DefaultLength)
	}
	if cfg.World.MaxBlocks != 33554432 {
		t.Fatalf("World.MaxBlocks = %d", cfg.World.MaxBlocks)
	}
	if cfg.Debug.PprofAddress != "127.0.0.1:6060" {
		t.Fatalf("Debug.PprofAddress = %q", cfg.Debug.PprofAddress)
	}
	if cfg.Debug.PprofShutdownTimeout != 10*time.Second {
		t.Fatalf("Debug.PprofShutdownTimeout = %s", cfg.Debug.PprofShutdownTimeout)
	}
	if cfg.Log.Level != "debug" || cfg.Log.Format != "json" {
		t.Fatalf("Log = %q/%q", cfg.Log.Level, cfg.Log.Format)
	}
	if cfg.Storage.Backend != "local" || cfg.Storage.WorldsDir != "levels" || cfg.Storage.PlayersDir != "userdb" {
		t.Fatalf("Storage = %+v", cfg.Storage)
	}
	if cfg.Storage.PolicyFile != "bans.json" || cfg.Storage.WorldFileExt != "bin" || cfg.Storage.MainWorldName != "spawn" {
		t.Fatalf("Storage = %+v", cfg.Storage)
	}
	if len(cfg.Commands.AdminCommands) != 3 || cfg.Commands.AdminCommands[0] != "tp" {
		t.Fatalf("Commands.AdminCommands = %#v", cfg.Commands.AdminCommands)
	}
	if !cfg.Player.WhitelistEnabled || cfg.Player.MaxUsernameLength != 16 {
		t.Fatalf("Player = %+v", cfg.Player)
	}
	if cfg.CPE.ExtPlayerList || cfg.CPE.FastMap || cfg.CPE.TwoWayPing {
		t.Fatalf("CPE = %+v, want all false", cfg.CPE)
	}
	if !cfg.Auth.Enabled || cfg.Auth.Salt != "1234567890abcdef" {
		t.Fatalf("Auth = %+v", cfg.Auth)
	}
}

func TestLoadAppliesNestedDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Network.ReadTimeout != 30*time.Second {
		t.Fatalf("default Network.ReadTimeout = %s", cfg.Network.ReadTimeout)
	}
	if cfg.Network.TCPNoDelay != true {
		t.Fatal("default Network.TCPNoDelay should be true")
	}
	if cfg.Simulation.TickInterval != 50*time.Millisecond {
		t.Fatalf("default Simulation.TickInterval = %s", cfg.Simulation.TickInterval)
	}
	if cfg.World.DefaultWidth != 128 || cfg.World.DefaultHeight != 64 {
		t.Fatalf("default World dims = %d/%d", cfg.World.DefaultWidth, cfg.World.DefaultHeight)
	}
	if cfg.Log.Level != "info" || cfg.Log.Format != "text" {
		t.Fatalf("default Log = %q/%q", cfg.Log.Level, cfg.Log.Format)
	}
	if cfg.Storage.Backend != "local" || cfg.Storage.WorldFileExt != ".swld" || cfg.Storage.MainWorldName != "main" {
		t.Fatalf("default Storage = %+v", cfg.Storage)
	}
	if len(cfg.Commands.AdminCommands) != 10 {
		t.Fatalf("default Commands.AdminCommands = %#v", cfg.Commands.AdminCommands)
	}
	if cfg.Player.WhitelistEnabled || cfg.Player.MaxUsernameLength != 32 {
		t.Fatalf("default Player = %+v", cfg.Player)
	}
	if !cfg.CPE.ExtPlayerList || !cfg.CPE.FastMap || !cfg.CPE.TwoWayPing {
		t.Fatalf("default CPE = %+v, want all true", cfg.CPE)
	}
	if cfg.Auth.Enabled || cfg.Auth.Salt != "" {
		t.Fatalf("default Auth = %+v", cfg.Auth)
	}
}

func TestLoadRejectsAuthWithoutSaltOrHeartbeat(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[auth]\n" +
			"enabled = true\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for auth without salt or heartbeat")
	}
}

func TestLoadAllowsAuthWithHeartbeatGeneratedSalt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[auth]\n" +
			"enabled = true\n" +
			"\n[heartbeat]\n" +
			"enabled = true\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !cfg.Auth.Enabled || !cfg.Heartbeat.Enabled || cfg.Auth.Salt != "" {
		t.Fatalf("Auth/Heartbeat = %+v/%+v", cfg.Auth, cfg.Heartbeat)
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[log]\n" +
			"level = \"verbose\"\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for invalid log level")
	}
}

func TestLoadRejectsUnknownNestedKey(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[network]\n" +
			"bogus = 123\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for unknown nested key")
	}
}

func TestLoadRejectsUnsupportedStorageBackend(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[storage]\n" +
			"backend = \"s3\"\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for unsupported storage backend")
	}
}

func TestLoadRejectsInvalidUsernameLength(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "server.toml")
	dataDir := filepath.Join(dir, "data")
	contents := []byte(
		"listen = \"127.0.0.1:25566\"\n" +
			"data_dir = \"" + dataDir + "\"\n" +
			"server_name = \"Test\"\n" +
			"motd = \"Test\"\n" +
			"\n[player]\n" +
			"max_username_length = 0\n",
	)
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load returned nil error for invalid max_username_length")
	}
}
