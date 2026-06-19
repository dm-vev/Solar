package app

import (
	"context"
	"os"

	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/generator/core"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/server"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
)

func buildServer(ctx context.Context, cfg config.Config) *server.Server {
	logger := cfg.Logger(os.Stderr)

	store := storage.NewLocalStore(cfg.DataDir)
	store.Configure(cfg.Storage.WorldsDir, cfg.Storage.PlayersDir, cfg.Storage.PolicyFile, cfg.Storage.WorldFileExt)
	commands := command.NewRegistry()
	commands.SetAdminCommands(cfg.Commands.AdminCommands)
	worlds := world.NewManager()
	players := player.NewRegistry()
	players.SetWhitelistEnabled(cfg.Player.WhitelistEnabled)
	entities := entity.NewManager()
	listener := network.NewListener(cfg.ListenAddress)
	listener.SetConnectRate(cfg.ConnectRate)
	pool := worker.NewPool(ctx, cfg.Workers)

	world.SetMaxBlocks(cfg.World.MaxBlocks)
	core.SetMaxBlocks(cfg.World.MaxBlocks)

	codec := classic.NewCodec(cfg.Name, cfg.MOTD, worlds, players, entities, commands)
	codec.SetLogger(logger)
	codec.SetPersistencePaths(store.WorldFile(cfg.Storage.MainWorldName), store.PlayerPolicyFile())
	codec.SetCommandContextBuilder(buildCommandContext)
	codec.SetWorkerPool(pool)
	codec.SetConnTimeouts(cfg.Network.ReadTimeout, cfg.Network.WriteTimeout)
	codec.SetOutboxSize(cfg.Network.SessionOutbox)
	codec.SetWriteBatchSize(cfg.Network.WriteBatchSize, cfg.Network.SessionOutbox)
	codec.SetTCPNoDelay(cfg.Network.TCPNoDelay)
	codec.SetSendTimeout(cfg.Network.SendTimeoutMode, cfg.Network.SendTimeout)

	srv := server.New(
		cfg,
		listener,
		codec,
		worlds,
		players,
		entities,
		store,
		pool,
		logger,
	)
	srv.SetLogger(logger)
	if cfg.Debug.PprofAddress != "" {
		srv.SetPprofAddress(cfg.Debug.PprofAddress)
	}

	generator.RegisterDefaults()
	return srv
}
