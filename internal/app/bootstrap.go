package app

import (
	"context"
	"os"
	"path/filepath"

	"github.com/solar-mc/solar/internal/antispam"
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
	"github.com/solar-mc/solar/internal/server"
	"github.com/solar-mc/solar/internal/storage"
	"github.com/solar-mc/solar/internal/worker"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/lua"
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

	blockDefsDir := filepath.Join(cfg.DataDir, cfg.Storage.BlockDefsDir)
	blockDefs := blocks.NewRegistry(blockDefsDir)
	if err := blockDefs.LoadGlobal(); err != nil {
		logger.Error("load block definitions", "error", err)
	}

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
	codec.SetBlockDefinitions(blockDefs)

	// Load internationalisation.
	i18nStore := i18n.New(cfg.Language.Default)
	if err := i18nStore.Load(cfg.Language.File); err != nil {
		logger.Error("load language file", "path", cfg.Language.File, "error", err)
	} else {
		logger.Info("loaded translations", "languages", i18nStore.Languages())
	}
	codec.SetI18n(i18nStore)

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

	// Load runtime .so plugins before enabling any registered plugins.
	// Each .so's init() calls plugin.Register; LoadAll then enables them
	// alongside compile-time plugins.
	// On non-linux/darwin platforms LoadDirectory is a no-op that logs a warning.
	if cfg.Plugins.Enabled {
		pluginDir := filepath.Join(cfg.DataDir, cfg.Plugins.Dir)
		if err := plugin.LoadDirectory(pluginDir, logger); err != nil {
			logger.Error("plugin directory load failed", "dir", pluginDir, "error", err)
		}
	}

	// Load Lua scripts. Requires -tags=lua at build time; without it
	// LoadLuaScripts is a no-op that logs a warning.
	if cfg.Lua.Enabled {
		luaDir := filepath.Join(cfg.DataDir, cfg.Lua.Dir)
		if err := lua.LoadLuaScripts(luaDir, logger); err != nil {
			logger.Error("lua script load failed", "dir", luaDir, "error", err)
		}
	}

	// Load plugins: create the server API handle and enable all registered plugins.
	pluginSrv := server.NewPluginServer(codec, worlds, commands, srv)
	pluginSrv.PostInit()
	codec.SetPlayerDatabase(pluginSrv.PlayerDB())
	codec.SetBlockDBLookup(pluginSrv.BlockDB)
	codec.SetNameConverter(blocks.NewNameConverter())
	codec.SetLevelCallbacks(
		pluginSrv.ChangeMap,
		pluginSrv.MainLevelName,
		pluginSrv.LoadLevelByName,
		pluginSrv.UnloadLevelByName,
		pluginSrv.ListLoadedLevels,
		pluginSrv.ListLevelFiles,
	)
	codec.SetQueuePhysics(func(x, y, z int) {
		if srv.BlockPhysics() != nil {
			srv.BlockPhysics().Queue(x, y, z)
		}
	})
	codec.SetMaxPlayers(cfg.MaxPlayers)
	codec.SetSpamChecker(antispam.New(antispam.Config{
		Enabled:      cfg.AntiSpam.Enabled,
		ChatMax:      cfg.AntiSpam.ChatMax,
		ChatWindow:   cfg.AntiSpam.ChatWindow,
		BlockMax:     cfg.AntiSpam.BlockMax,
		BlockWindow:  cfg.AntiSpam.BlockWindow,
		CmdMax:       cfg.AntiSpam.CmdMax,
		CmdWindow:    cfg.AntiSpam.CmdWindow,
		Action:       antispam.Action(cfg.AntiSpam.Action),
		MuteDuration: cfg.AntiSpam.MuteDuration,
	}))
	if err := plugin.LoadAll(pluginSrv, logger); err != nil {
		logger.Error("plugin load failed", "error", err)
	}
	plugin.OnPluginsLoaded.Fire(plugin.PluginsLoadedData{})

	return srv
}
