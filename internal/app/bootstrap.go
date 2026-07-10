package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/solar-mc/solar/internal/auth"
	"github.com/solar-mc/solar/internal/blocks"
	"github.com/solar-mc/solar/internal/command"
	"github.com/solar-mc/solar/internal/config"
	"github.com/solar-mc/solar/internal/entity"
	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/generator/core"
	"github.com/solar-mc/solar/internal/i18n"
	"github.com/solar-mc/solar/internal/network"
	"github.com/solar-mc/solar/internal/player"
	"github.com/solar-mc/solar/internal/protocol/classic"
	"github.com/solar-mc/solar/internal/ranks"
	"github.com/solar-mc/solar/internal/server"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/lua"
)

func buildServer(ctx context.Context, cfg config.Config) (*server.Server, error) {
	logger := cfg.Logger(os.Stderr)
	if err := prepareAuthentication(&cfg, logger); err != nil {
		return nil, err
	}

	blockDefinitions, err := loadBlockDefinitions(cfg)
	if err != nil {
		return nil, err
	}

	store := storage.NewLocalStore(cfg.DataDir)
	store.Configure(
		cfg.Storage.WorldsDir,
		cfg.Storage.PlayersDir,
		cfg.Storage.PolicyFile,
		cfg.Storage.WorldFileExt,
	)

	commands := command.NewRegistry()
	worlds := world.NewManager()
	players := player.NewRegistry()
	players.SetWhitelistEnabled(cfg.Player.WhitelistEnabled)
	entities := entity.NewManager()
	listener := network.NewListener(cfg.ListenAddress)
	listener.SetConnectRate(cfg.ConnectRate)
	pool := worker.NewPool(ctx, cfg.Workers)

	world.SetMaxBlocks(cfg.World.MaxBlocks)
	core.SetMaxBlocks(cfg.World.MaxBlocks)

	translations := loadTranslations(cfg, logger)
	codec := buildCodec(
		cfg,
		store,
		worlds,
		players,
		entities,
		commands,
		pool,
		blockDefinitions,
		translations,
		logger,
	)

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
	if cfg.Debug.PprofAddress != "" {
		srv.SetPprofAddress(cfg.Debug.PprofAddress)
	}

	generator.RegisterDefaults()
	loadRuntimeExtensions(cfg, logger)
	wirePluginRuntime(codec, worlds, commands, srv, logger)

	return srv, nil
}

func loadBlockDefinitions(cfg config.Config) (*blocks.Registry, error) {
	blockDefinitionsDir := filepath.Join(cfg.DataDir, cfg.Storage.BlockDefsDir)
	registry := blocks.NewRegistry(blockDefinitionsDir)
	if err := registry.LoadGlobal(); err != nil {
		return nil, fmt.Errorf("load global block definitions: %w", err)
	}
	return registry, nil
}

func loadTranslations(cfg config.Config, logger *slog.Logger) *i18n.I18n {
	translations := i18n.New(cfg.Language.Default)
	if err := translations.Load(cfg.Language.File); err != nil {
		logger.Warn(
			"language file unavailable; continuing with message-key fallback",
			"path", cfg.Language.File,
			"error", err,
		)
		return translations
	}

	logger.Info("loaded translations", "languages", translations.Languages())
	return translations
}

func buildCodec(
	cfg config.Config,
	store *storage.LocalStore,
	worlds *world.Manager,
	players *player.Registry,
	entities *entity.Manager,
	commands *command.Registry,
	pool *worker.Pool,
	blockDefinitions *blocks.Registry,
	translations *i18n.I18n,
	logger *slog.Logger,
) *classic.Codec {
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
	codec.SetBlockDefinitions(blockDefinitions)
	codec.SetI18n(translations)
	codec.SetMaxPlayers(cfg.MaxPlayers)
	codec.SetAuthentication(cfg.Auth.Enabled, cfg.Auth.Salt)
	codec.SetSpamChecker(player.NewChecker(player.SpamConfig{
		Enabled:      cfg.AntiSpam.Enabled,
		ChatMax:      cfg.AntiSpam.ChatMax,
		ChatWindow:   cfg.AntiSpam.ChatWindow,
		BlockMax:     cfg.AntiSpam.BlockMax,
		BlockWindow:  cfg.AntiSpam.BlockWindow,
		CmdMax:       cfg.AntiSpam.CmdMax,
		CmdWindow:    cfg.AntiSpam.CmdWindow,
		Action:       player.SpamAction(cfg.AntiSpam.Action),
		MuteDuration: cfg.AntiSpam.MuteDuration,
	}))
	return codec
}

func prepareAuthentication(cfg *config.Config, logger *slog.Logger) error {
	if cfg.Auth.Salt != "" || (!cfg.Auth.Enabled && !cfg.Heartbeat.Enabled) {
		return nil
	}

	salt, err := auth.GenerateSalt()
	if err != nil {
		return fmt.Errorf("generate auth salt: %w", err)
	}
	cfg.Auth.Salt = salt
	if cfg.Auth.Enabled {
		logger.Info("generated runtime auth salt")
	}
	return nil
}

func loadRuntimeExtensions(cfg config.Config, logger *slog.Logger) {
	if cfg.Plugins.Enabled {
		pluginDir := filepath.Join(cfg.DataDir, cfg.Plugins.Dir)
		if err := plugin.LoadDirectory(pluginDir, logger); err != nil {
			logger.Error("plugin directory load failed", "dir", pluginDir, "error", err)
		}
	}

	if cfg.Lua.Enabled {
		luaDir := filepath.Join(cfg.DataDir, cfg.Lua.Dir)
		if err := lua.LoadLuaScripts(luaDir, logger); err != nil {
			logger.Error("lua script load failed", "dir", luaDir, "error", err)
		}
	}
}

func wirePluginRuntime(
	codec *classic.Codec,
	worlds *world.Manager,
	commands *command.Registry,
	srv *server.Server,
	logger *slog.Logger,
) {
	pluginServer := server.NewPluginServer(codec, worlds, commands, srv)
	pluginServer.PostInit()

	codec.SetPlayerDatabase(pluginServer.PlayerDB())
	codec.SetBlockDBLookup(pluginServer.BlockDB)
	codec.SetNameConverter(blocks.NewNameConverter())
	codec.SetLevelCallbacks(
		pluginServer.ChangeMap,
		pluginServer.MainLevelName,
		pluginServer.LoadLevelByName,
		pluginServer.UnloadLevelByName,
		pluginServer.ListLoadedLevels,
		pluginServer.ListLevelFiles,
	)
	codec.SetQueuePhysics(srv.QueueBlockPhysics)
	codec.SetPhysicsModeCallbacks(srv.BlockPhysicsMode, srv.SetBlockPhysicsMode)
	codec.SetSaveStateCallback(srv.SaveStateNow)

	rankRegistry := ranks.NewRegistry()
	rankRegistry.SetPlayerDB(pluginServer.PlayerDB())
	codec.SetRankRegistry(rankRegistry)

	if err := plugin.LoadAll(pluginServer, logger); err != nil {
		logger.Error("plugin load failed", "error", err)
	}
	plugin.OnPluginsLoaded.Fire(plugin.PluginsLoadedData{})
}
