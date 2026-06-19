package server

import (
	"context"
	"os"
	"testing"

	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
)

func TestMain(m *testing.M) {
	generator.RegisterDefaults()
	os.Exit(m.Run())
}

func TestLoadStateGeneratesDefaultWorld(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := config.Config{
		ListenAddress:    ":25565",
		DataDir:          dir,
		Workers:          1,
		MaxPlayers:       8,
		ConnectRate:      1,
		Autosave:         0,
		DefaultGenerator: "Classic",
		Name:             "Solar",
		MOTD:             "Test",
	}

	store := storage.NewLocalStore(dir)
	worlds := world.NewManager()
	players := player.NewRegistry()
	entities := entity.NewManager()
	listener := network.NewListener(":25565")
	codec := classic.NewCodec("Solar", "Test", worlds, players, entities, nil)

	srv := New(cfg, listener, codec, worlds, players, entities, store, worker.NewPool(context.Background(), 1))

	worldPath, _, err := srv.loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}

	if _, err := os.Stat(worldPath); err != nil {
		t.Fatalf("generated world file not found: %v", err)
	}

	level := worlds.Current()
	if level.Width != 128 || level.Height != 64 || level.Length != 128 {
		t.Fatalf("generated level size = %dx%dx%d, want 128x64x128", level.Width, level.Height, level.Length)
	}

	nonAir := 0
	for _, b := range level.Blocks {
		if b != 0 {
			nonAir++
		}
	}
	if nonAir == 0 {
		t.Fatal("default generator produced only air")
	}
}

func TestLoadStateUsesExistingWorld(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := config.Config{
		ListenAddress:    ":25565",
		DataDir:          dir,
		Workers:          1,
		MaxPlayers:       8,
		ConnectRate:      1,
		Autosave:         0,
		DefaultGenerator: "Classic",
		Name:             "Solar",
		MOTD:             "Test",
	}

	store := storage.NewLocalStore(dir)
	os.MkdirAll(store.WorldsDir(), 0o755)

	worldPath := store.WorldFile("main")
	emptyWorld := world.Level{
		Name:   "main",
		Width:  4,
		Height: 4,
		Length: 4,
		Blocks: make([]byte, 64),
		Spawn:  world.Spawn{X: 2, Y: 2, Z: 2},
	}
	emptyWorld.Blocks[0] = 1
	if err := emptyWorld.Save(worldPath); err != nil {
		t.Fatalf("save existing world: %v", err)
	}

	worlds := world.NewManager()
	players := player.NewRegistry()
	entities := entity.NewManager()
	listener := network.NewListener(":25565")
	codec := classic.NewCodec("Solar", "Test", worlds, players, entities, nil)

	srv := New(cfg, listener, codec, worlds, players, entities, store, worker.NewPool(context.Background(), 1))

	loadedPath, _, err := srv.loadState()
	if err != nil {
		t.Fatalf("loadState: %v", err)
	}
	if loadedPath != worldPath {
		t.Fatalf("worldPath = %q, want %q", loadedPath, worldPath)
	}

	level := worlds.Current()
	if level.Width != 4 {
		t.Fatalf("loaded level width = %d, want 4", level.Width)
	}
	if level.Blocks[0] != 1 {
		t.Fatalf("loaded level block[0] = %d, want 1", level.Blocks[0])
	}
}

func TestLoadStateRejectsUnknownDefaultGenerator(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := config.Config{
		ListenAddress:    ":25565",
		DataDir:          dir,
		Workers:          1,
		MaxPlayers:       8,
		ConnectRate:      1,
		Autosave:         0,
		DefaultGenerator: "VoidGenerator",
		Name:             "Solar",
		MOTD:             "Test",
	}

	store := storage.NewLocalStore(dir)
	worlds := world.NewManager()
	players := player.NewRegistry()
	entities := entity.NewManager()
	listener := network.NewListener(":25565")
	codec := classic.NewCodec("Solar", "Test", worlds, players, entities, nil)

	srv := New(cfg, listener, codec, worlds, players, entities, store, worker.NewPool(context.Background(), 1))

	_, _, err := srv.loadState()
	if err == nil {
		t.Fatal("expected error for unknown default generator")
	}
}
