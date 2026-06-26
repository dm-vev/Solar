package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

func TestPluginServerCoreConfigWorldAndLevels(t *testing.T) {
	host, srv := newPluginHostFixture(t)
	host.PostInit()
	if srv.flushBlockDBsFn == nil {
		t.Fatal("PostInit did not wire blockdb flush")
	}

	host.BroadcastMessage("hello")
	host.BroadcastMessageTo("all", nil, "hello")
	host.BroadcastMessageTo("level", nil, "hello")
	host.BroadcastMessageTo("ops", nil, "hello")
	host.BroadcastMessageTo("unknown", nil, "hello")
	if len(host.OnlinePlayers()) != 0 || host.OnlineCount() != 0 || host.MaxPlayers() != 8 {
		t.Fatal("online/max player adapters returned unexpected values")
	}
	if host.ServerName() != "Solar" || host.MOTD() != "MOTD" || host.FindPlayer("nobody") != nil {
		t.Fatal("server identity/find adapters returned unexpected values")
	}
	if host.World() == nil || host.Levels() == nil || host.Physics() == nil || host.Scheduler() == nil ||
		host.Entities() == nil || host.Config() == nil || host.PlayerDB() == nil {
		t.Fatal("plugin server handle returned nil subsystem")
	}
	if host.ChangeMap(nil, "missing") {
		t.Fatal("ChangeMap succeeded for missing level")
	}
	if host.RegisterCommand("", "", func(player plugin.Player, args []string) string { return "" }) {
		t.Fatal("RegisterCommand accepted empty name")
	}
	if !host.RegisterCommand("unit", "help", func(player plugin.Player, args []string) string { return "ok" }) ||
		!host.UnregisterCommand("unit") {
		t.Fatal("register/unregister command failed")
	}

	if !host.BanPlayer("bob", "bad") || !host.UnbanPlayer("bob") {
		t.Fatal("ban/unban failed")
	}
	host.SetWhitelistEnabled(true)
	if !host.IsWhitelistEnabled() || !host.WhitelistAdd("bob") || !host.WhitelistRemove("bob") {
		t.Fatal("whitelist adapters failed")
	}
	if !host.AddOperators("alice") || !host.IsOperator("alice") || len(host.OperatorNames()) != 1 {
		t.Fatal("operator adapters failed")
	}
	if !host.SaveState() {
		t.Fatal("SaveState returned false")
	}
	ctx, cancel := context.WithCancel(context.Background())
	srv.cancel = cancel
	host.Stop()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("Stop did not cancel server context")
	}

	db := host.BlockDB("main")
	if db == nil || host.BlockDB("main") != db || host.BlockDB("missing") != nil {
		t.Fatal("BlockDB lookup/cache failed")
	}
	host.flushAllBlockDBs()
	if host.MainLevelName() != "main" {
		t.Fatalf("MainLevelName = %q", host.MainLevelName())
	}
	callbackMgr := world.NewManager()
	callbackMgr.SetCurrent(world.Level{Name: "callback", Width: 4, Height: 4, Length: 4, Blocks: make([]byte, 64)})
	if err := callbackMgr.Save(srv.store.WorldFile("callback")); err != nil {
		t.Fatalf("save callback world: %v", err)
	}
	if !host.LoadLevelByName("callback") || len(host.ListLoadedLevels()) == 0 || len(host.ListLevelFiles()) == 0 {
		t.Fatal("level callback helpers failed")
	}
	if !host.UnloadLevelByName("callback") {
		t.Fatal("UnloadLevelByName failed")
	}

	cfg := host.Config()
	cfg.SetName("Solar2")
	cfg.SetMOTD("MOTD2")
	cfg.SetMaxPlayers(9)
	cfg.SetConnectRate(2)
	cfg.SetTCPNoDelay(false)
	cfg.SetTickInterval(25 * time.Millisecond)
	cfg.SetAutosaveInterval(time.Second)
	if cfg.Name() != "Solar2" || cfg.MOTD() != "MOTD2" || cfg.MaxPlayers() != 9 ||
		cfg.ConnectRate() != 2 || cfg.TCPNoDelay() || cfg.TickInterval() != 25*time.Millisecond ||
		cfg.AutosaveInterval() != time.Second {
		t.Fatal("config setters/getters returned unexpected values")
	}
	if cfg.ReadTimeout() != time.Second || cfg.WriteTimeout() != 2*time.Second ||
		cfg.DefaultWidth() != 4 || cfg.DefaultHeight() != 4 || cfg.DefaultLength() != 4 || cfg.MaxBlocks() != 1024 {
		t.Fatal("config read-only getters returned unexpected values")
	}
	if !cfg.AddOperator("charlie") || len(cfg.Operators()) == 0 || !cfg.RemoveOperator("charlie") {
		t.Fatal("config operator adapters failed")
	}
	cfg.SetWhitelistEnabled(false)
	if cfg.WhitelistEnabled() {
		t.Fatal("config whitelist disable failed")
	}

	testLevelManager(t, host)
	testPluginWorld(t, host.World())
}

func TestPluginPhysics(t *testing.T) {
	mgr := managerWithLevel("physics", 4, 4, 4)
	mgr.SetBlock(1, 1, 1, 7)
	physics := newPluginPhysics(mgr)
	physics.SetMode(plugin.PhysicsAdvanced)
	if physics.Mode() != plugin.PhysicsAdvanced {
		t.Fatalf("Mode = %d", physics.Mode())
	}
	called := false
	physics.RegisterHandler(func(block plugin.PhysicsBlock) bool {
		called = block.X == 1 && block.Y == 1 && block.Z == 1 && block.Block == 7 && block.Level == "physics"
		return false
	})
	physics.Schedule(1, 1, 1)
	physics.Tick()
	if !called {
		t.Fatal("physics handler was not called with expected block")
	}
	physics.Tick()
}

func testPluginWorld(t *testing.T, worldHandle plugin.World) {
	t.Helper()

	pw := worldHandle.(*pluginWorld)
	if !pw.SetBlock(0, 0, 0, 5) {
		t.Fatal("pluginWorld SetBlock failed")
	}
	if got, ok := pw.GetBlock(0, 0, 0); !ok || got != 5 {
		t.Fatalf("pluginWorld GetBlock = %d ok=%v", got, ok)
	}
	pw.SetSpawn(1, 2, 3, 4, 5)
	if x, y, z, yaw, pitch := pw.Spawn(); x != 1 || y != 2 || z != 3 || yaw != 4 || pitch != 5 {
		t.Fatalf("pluginWorld Spawn = %d %d %d %d %d", x, y, z, yaw, pitch)
	}
	if width, height, length := pw.Dimensions(); width != 4 || height != 4 || length != 4 {
		t.Fatalf("pluginWorld Dimensions = %d %d %d", width, height, length)
	}
	if pw.Name() != "main" || pw.PlayerCount() != 0 || len(pw.Players()) != 0 {
		t.Fatal("pluginWorld level info returned unexpected values")
	}
	pw.Message("level message")
	if err := pw.Save(); err != nil {
		t.Fatalf("pluginWorld Save: %v", err)
	}
	if err := pw.Resize(5, 4, 4); err != nil {
		t.Fatalf("pluginWorld Resize: %v", err)
	}
	if err := pw.Resize(0, 4, 4); err == nil {
		t.Fatal("pluginWorld Resize accepted zero dimension")
	}
	if err := pw.Reload(); err != nil {
		t.Fatalf("pluginWorld Reload: %v", err)
	}
	if err := pw.Copy("main_copy"); err != nil {
		t.Fatalf("pluginWorld Copy: %v", err)
	}
	if err := pw.Backup("main_backup"); err != nil {
		t.Fatalf("pluginWorld Backup: %v", err)
	}
	if got := pw.levelPath("derived"); filepath.Base(got) != "derived.swld" {
		t.Fatalf("pluginWorld levelPath = %q", got)
	}
	if err := pw.Rename("main_renamed"); err != nil {
		t.Fatalf("pluginWorld Rename: %v", err)
	}
	if pw.Name() != "main_renamed" {
		t.Fatalf("pluginWorld name after rename = %q", pw.Name())
	}
	if err := pw.Delete(); err != nil {
		t.Fatalf("pluginWorld Delete: %v", err)
	}
}

func testLevelManager(t *testing.T, host *pluginServer) {
	t.Helper()

	levels := host.Levels()
	current := levels.Current()
	if current == nil || current.Name() == "" {
		t.Fatal("LevelManager Current returned nil/empty")
	}
	if levels.Find("main") == nil || levels.Find("missing") != nil {
		t.Fatal("LevelManager Find returned unexpected values")
	}
	created, err := levels.Create("created", 4, 4, 4, "Flat", "2")
	if err != nil {
		t.Fatalf("LevelManager Create: %v", err)
	}
	if created.Name() != "created" {
		t.Fatalf("created level name = %q", created.Name())
	}
	if _, err := levels.Create("bad", 4, 4, 4, "MissingGenerator", ""); err == nil {
		t.Fatal("Create accepted missing generator")
	}
	if err := created.Save(); err != nil {
		t.Fatalf("created Save: %v", err)
	}
	if !created.SetBlock(0, 0, 0, 6) {
		t.Fatal("created SetBlock failed")
	}
	if got, ok := created.GetBlock(0, 0, 0); !ok || got != 6 {
		t.Fatalf("created GetBlock = %d ok=%v", got, ok)
	}
	created.SetSpawn(1, 1, 1, 2, 3)
	if x, y, z, yaw, pitch := created.Spawn(); x != 1 || y != 1 || z != 1 || yaw != 2 || pitch != 3 {
		t.Fatalf("created Spawn = %d %d %d %d %d", x, y, z, yaw, pitch)
	}
	if width, height, length := created.Dimensions(); width != 4 || height != 4 || length != 4 {
		t.Fatalf("created Dimensions = %d %d %d", width, height, length)
	}
	if created.PlayerCount() != 0 || len(created.Players()) != 0 {
		t.Fatal("created player list should be empty")
	}
	created.Message("hello")
	if err := created.Resize(5, 4, 4); err != nil {
		t.Fatalf("created Resize: %v", err)
	}
	if err := created.Resize(0, 4, 4); err == nil {
		t.Fatal("created Resize accepted zero dimension")
	}
	if err := created.Reload(); err != nil {
		t.Fatalf("created Reload: %v", err)
	}
	if err := created.Copy("created_copy"); err != nil {
		t.Fatalf("created Copy: %v", err)
	}
	if err := created.Backup("created_backup"); err != nil {
		t.Fatalf("created Backup: %v", err)
	}
	if got := created.(*pluginLevel).levelPath("derived"); filepath.Base(got) != "derived.swld" {
		t.Fatalf("pluginLevel levelPath = %q", got)
	}
	if err := created.Rename("created_renamed"); err != nil {
		t.Fatalf("created Rename: %v", err)
	}
	if created.Name() != "created_renamed" {
		t.Fatalf("created name after rename = %q", created.Name())
	}
	if err := created.Delete(); err != nil {
		t.Fatalf("created Delete: %v", err)
	}

	loadedMgr := world.NewManager()
	loadedMgr.SetCurrent(world.Level{Name: "loadme", Width: 4, Height: 4, Length: 4, Blocks: make([]byte, 64)})
	if err := loadedMgr.Save(host.server.store.WorldFile("loadme")); err != nil {
		t.Fatalf("save loadme: %v", err)
	}
	loaded, err := levels.Load("loadme")
	if err != nil {
		t.Fatalf("LevelManager Load: %v", err)
	}
	if loaded.Name() != "loadme" {
		t.Fatalf("loaded name = %q", loaded.Name())
	}
	if _, err := levels.Load("loadme"); err == nil {
		t.Fatal("Load accepted already loaded level")
	}
	if !levels.Unload("loadme") || levels.Unload("main") || levels.Unload("missing") {
		t.Fatal("Unload returned unexpected values")
	}

	if err := levels.SaveAll(); err != nil {
		t.Fatalf("SaveAll: %v", err)
	}
	if len(levels.List()) == 0 || len(levels.ListFiles()) == 0 {
		t.Fatal("level list/list files returned empty")
	}
	if err := levels.CopyLevel("main", "main_copy2"); err != nil {
		t.Fatalf("CopyLevel: %v", err)
	}
	if err := levels.BackupLevel("main", "main_backup2"); err != nil {
		t.Fatalf("BackupLevel: %v", err)
	}
	if err := levels.RenameLevel("main_copy2", "main_copy3"); err != nil {
		t.Fatalf("RenameLevel: %v", err)
	}
	if err := levels.DeleteLevel("main_copy3"); err != nil {
		t.Fatalf("DeleteLevel: %v", err)
	}
}

func newPluginHostFixture(t *testing.T) (*pluginServer, *Server) {
	t.Helper()

	dir := t.TempDir()
	store := storage.NewLocalStore(dir)
	if err := os.MkdirAll(store.WorldsDir(), 0o755); err != nil {
		t.Fatalf("mkdir worlds: %v", err)
	}
	if err := os.MkdirAll(store.PlayersDir(), 0o755); err != nil {
		t.Fatalf("mkdir players: %v", err)
	}

	worlds := managerWithLevel("main", 4, 4, 4)
	if err := worlds.Save(store.WorldFile("main")); err != nil {
		t.Fatalf("save main world: %v", err)
	}
	players := player.NewRegistry()
	entities := entity.NewManager()
	commands := command.NewRegistry()
	codec := classic.NewCodec("Solar", "MOTD", worlds, players, entities, commands)
	cfg := config.Config{
		MaxPlayers:  8,
		ConnectRate: 1,
		Name:        "Solar",
		MOTD:        "MOTD",
		Autosave:    0,
		Network: config.NetworkConfig{
			ReadTimeout:  1 * time.Second,
			WriteTimeout: 2 * time.Second,
			TCPNoDelay:   true,
		},
		Simulation: config.SimConfig{TickInterval: 50 * time.Millisecond},
		World: config.WorldConfig{
			DefaultWidth:  4,
			DefaultHeight: 4,
			DefaultLength: 4,
			MaxBlocks:     1024,
		},
		Storage: config.StorageConfig{
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
			BlockDefsDir:  "blockdefs",
		},
	}
	srv := New(cfg, nil, codec, worlds, players, entities, store, nil, testLogger)
	host := NewPluginServer(codec, worlds, commands, srv)
	return host, srv
}
