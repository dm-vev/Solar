package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

func TestPluginServerPostInitAndLevelLoad(t *testing.T) {
	host, srv := newPluginHostFixture(t)
	host.PostInit()
	if srv.flushBlockDBsFn == nil {
		t.Fatal("PostInit did not wire blockdb flush")
	}

	manager := world.NewManager()
	manager.SetCurrent(world.Level{Name: "loadme", Width: 4, Height: 4, Length: 4, Blocks: make([]byte, 64)})
	if err := manager.Save(srv.store.WorldFile("loadme")); err != nil {
		t.Fatalf("save loadme: %v", err)
	}
	if !host.LoadLevelByName("loadme") {
		t.Fatal("LoadLevelByName returned false")
	}
	if !host.UnloadLevelByName("loadme") {
		t.Fatal("UnloadLevelByName returned false")
	}
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
		t.Fatal("physics handler was not called")
	}
}

func TestPluginConfigRejectsMaxPlayersAboveClassicLimit(t *testing.T) {
	host, _ := newPluginHostFixture(t)
	cfg := host.Config()

	cfg.SetMaxPlayers(int(entity.MaxClassicEntityID))
	if cfg.MaxPlayers() != int(entity.MaxClassicEntityID) {
		t.Fatalf("MaxPlayers = %d, want %d", cfg.MaxPlayers(), entity.MaxClassicEntityID)
	}

	cfg.SetMaxPlayers(int(entity.MaxClassicEntityID) + 1)
	if cfg.MaxPlayers() != int(entity.MaxClassicEntityID) {
		t.Fatalf("MaxPlayers changed to %d after invalid update", cfg.MaxPlayers())
	}

	cfg.SetMaxPlayers(0)
	if cfg.MaxPlayers() != int(entity.MaxClassicEntityID) {
		t.Fatalf("MaxPlayers changed to %d after zero update", cfg.MaxPlayers())
	}
}

func TestPluginLevelAPIsRejectEscapingNames(t *testing.T) {
	host, _ := newPluginHostFixture(t)
	levels := host.Levels()

	if _, err := levels.Create("../evil", 4, 4, 4, "Classic", ""); err == nil {
		t.Fatal("Create accepted escaping level name")
	}
	if _, err := levels.Load("../evil"); err == nil {
		t.Fatal("Load accepted escaping level name")
	}
	if err := levels.RenameLevel("main", "../evil"); err == nil {
		t.Fatal("RenameLevel accepted escaping destination")
	}
	if err := levels.DeleteLevel("../evil"); err == nil {
		t.Fatal("DeleteLevel accepted escaping name")
	}
	if err := levels.CopyLevel("../evil", "copy"); err == nil {
		t.Fatal("CopyLevel accepted escaping source")
	}
	if err := levels.BackupLevel("main", "../evil"); err == nil {
		t.Fatal("BackupLevel accepted escaping destination")
	}
	worldAPI, ok := host.World().(*pluginWorld)
	if !ok {
		t.Fatal("World did not return pluginWorld")
	}
	if err := worldAPI.Copy("../evil"); err == nil {
		t.Fatal("World.Copy accepted escaping destination")
	}
}

func TestCopyFileWritesDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.swld")
	dst := filepath.Join(dir, "nested", "dst.swld")
	if err := os.WriteFile(src, []byte("level-data"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "level-data" {
		t.Fatalf("dst = %q, want level-data", data)
	}
}

func newPluginHostFixture(t *testing.T) (*pluginServer, *Server) {
	t.Helper()

	store := storage.NewLocalStore(t.TempDir())
	worlds := managerWithLevel("main", 4, 4, 4)
	if err := worlds.Save(store.WorldFile("main")); err != nil {
		t.Fatalf("save main world: %v", err)
	}

	players := player.NewRegistry()
	entities := entity.NewManager()
	commands := command.NewRegistry()
	codec := classic.NewCodec("Solar", "MOTD", worlds, players, entities, commands)
	cfg := config.Config{
		MaxPlayers: 8,
		Name:       "Solar",
		MOTD:       "MOTD",
		Storage: config.StorageConfig{
			WorldsDir:     "worlds",
			PlayersDir:    "players",
			PolicyFile:    "policy.json",
			WorldFileExt:  ".swld",
			MainWorldName: "main",
		},
	}
	srv := New(cfg, nil, codec, worlds, players, entities, store, nil, testLogger)
	return NewPluginServer(codec, worlds, commands, srv), srv
}
