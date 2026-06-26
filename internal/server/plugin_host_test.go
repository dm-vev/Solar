package server

import (
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
