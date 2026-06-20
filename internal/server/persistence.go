package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/world"
	"github.com/solar-mc/solar/plugin"
)

func (s *Server) loadState() (string, string, error) {
	if err := os.MkdirAll(s.store.WorldsDir(), 0o755); err != nil {
		return "", "", fmt.Errorf("create worlds directory: %w", err)
	}
	if err := os.MkdirAll(s.store.PlayersDir(), 0o755); err != nil {
		return "", "", fmt.Errorf("create players directory: %w", err)
	}

	worldPath := s.store.WorldFile(s.cfg.Storage.MainWorldName)
	if _, err := os.Stat(worldPath); err != nil && errors.Is(err, os.ErrNotExist) {
		if err := s.generateDefaultWorld(worldPath); err != nil {
			return "", "", fmt.Errorf("generate default world %s: %w", worldPath, err)
		}
	} else if err != nil {
		return "", "", fmt.Errorf("stat world %s: %w", worldPath, err)
	} else {
		if plugin.OnLevelLoad.HasHandlers() {
			ctx := plugin.OnLevelLoad.Fire(plugin.LevelLoadData{Name: s.cfg.Storage.MainWorldName})
			if !ctx.Cancelled() {
				if err := s.worlds.Load(worldPath); err != nil {
					return "", "", fmt.Errorf("load world %s: %w", worldPath, err)
				}
			}
		} else {
			if err := s.worlds.Load(worldPath); err != nil {
				return "", "", fmt.Errorf("load world %s: %w", worldPath, err)
			}
		}
	}

	policyPath := s.store.PlayerPolicyFile()
	if err := s.players.LoadPolicy(policyPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", fmt.Errorf("load player policy %s: %w", policyPath, err)
	}
	if len(s.cfg.Operators) > 0 {
		if changed := s.players.AddOperators(s.cfg.Operators...); changed && policyPath != "" {
			if err := s.players.SavePolicy(policyPath); err != nil {
				return "", "", fmt.Errorf("save player policy %s: %w", policyPath, err)
			}
		}
	}

	return worldPath, policyPath, nil
}

func (s *Server) generateDefaultWorld(worldPath string) error {
	gen, ok := generator.Find(s.cfg.DefaultGenerator)
	if !ok {
		return fmt.Errorf("unknown default generator %q", s.cfg.DefaultGenerator)
	}

	args, err := generator.ParseArgs("")
	if err != nil {
		return fmt.Errorf("parse default generator args: %w", err)
	}

	lvl, err := generator.Generate(gen, s.cfg.Storage.MainWorldName, s.cfg.World.DefaultWidth, s.cfg.World.DefaultHeight, s.cfg.World.DefaultLength, args)
	if err != nil {
		return fmt.Errorf("generate default world: %w", err)
	}

	s.worlds.SetCurrent(world.FromGeneratorLevel(lvl))

	if err := s.worlds.Save(worldPath); err != nil {
		return fmt.Errorf("save generated world %s: %w", worldPath, err)
	}
	s.logger.Info("generated default world", "generator", s.cfg.DefaultGenerator, "path", worldPath)
	return nil
}

func (s *Server) saveState(worldPath, policyPath string) {
	if worldPath != "" {
		skipSave := false
		if plugin.OnLevelSave.HasHandlers() {
			ctx := plugin.OnLevelSave.Fire(plugin.LevelSaveData{})
			skipSave = ctx.Cancelled()
		}
		if !skipSave {
			if err := s.worlds.Save(worldPath); err != nil {
				s.logger.Error("save world", "path", worldPath, "error", err)
			}
		}
	}
	if policyPath != "" {
		if err := s.players.SavePolicy(policyPath); err != nil {
			s.logger.Error("save player policy", "path", policyPath, "error", err)
		}
	}
	if s.playerDB != nil {
		if err := s.playerDB.Flush(); err != nil {
			s.logger.Error("flush playerdb", "error", err)
		}
	}
	s.flushBlockDBs()
}

// SaveStateNow persists world and player policy using the store's configured
// paths. Safe to call at runtime; it re-derives the same paths as loadState.
func (s *Server) SaveStateNow() {
	s.saveState(s.store.WorldFile(s.cfg.Storage.MainWorldName), s.store.PlayerPolicyFile())
}

// worldSavePath returns the on-disk path of the main world snapshot.
func (s *Server) worldSavePath() string {
	return s.store.WorldFile(s.cfg.Storage.MainWorldName)
}

func (s *Server) autosaveLoop(ctx context.Context, worldPath, policyPath string) {
	interval := s.cfg.Autosave
	if interval <= 0 {
		// Autosave is disabled; the caller normally skips this goroutine.
		return
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.saveState(worldPath, policyPath)
		}
	}
}
