package app

import (
	"context"
	"log/slog"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/server"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
)

func buildServer(ctx context.Context, cfg config.Config) *server.Server {
	logger := slog.Default()

	store := storage.NewLocalStore(cfg.DataDir)
	commands := command.NewRegistry()
	worlds := world.NewManager()
	players := player.NewRegistry()
	entities := entity.NewManager()
	listener := network.NewListener(cfg.ListenAddress)
	listener.SetConnectRate(cfg.ConnectRate)
	codec := classic.NewCodec(cfg.Name, cfg.MOTD, worlds, players, entities, commands)
	codec.SetLogger(logger)
	codec.SetPersistencePaths(store.WorldFile("main"), store.PlayerPolicyFile())

	srv := server.New(
		cfg,
		listener,
		codec,
		worlds,
		players,
		entities,
		store,
		worker.NewPool(ctx, cfg.Workers),
	)
	srv.SetLogger(logger)

	generator.RegisterDefaults()
	return srv
}
