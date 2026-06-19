package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/solar-mc/solar/internal/generator"
	"github.com/solar-mc/solar/internal/world"
)

func (s *Server) loadState() (string, string, error) {
	if err := os.MkdirAll(s.store.WorldsDir(), 0o755); err != nil {
		return "", "", fmt.Errorf("create worlds directory: %w", err)
	}
	if err := os.MkdirAll(s.store.PlayersDir(), 0o755); err != nil {
		return "", "", fmt.Errorf("create players directory: %w", err)
	}

	worldPath := s.store.WorldFile("main")
	if _, err := os.Stat(worldPath); err != nil && errors.Is(err, os.ErrNotExist) {
		if err := s.generateDefaultWorld(worldPath); err != nil {
			return "", "", fmt.Errorf("generate default world %s: %w", worldPath, err)
		}
	} else if err != nil {
		return "", "", fmt.Errorf("stat world %s: %w", worldPath, err)
	} else {
		if err := s.worlds.Load(worldPath); err != nil {
			return "", "", fmt.Errorf("load world %s: %w", worldPath, err)
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

// defaultWorldDimensions are the dimensions for a generated default world.
const (
	defaultWorldWidth  = 128
	defaultWorldHeight = 64
	defaultWorldLength = 128
)

func (s *Server) generateDefaultWorld(worldPath string) error {
	gen, ok := generator.Find(s.cfg.DefaultGenerator)
	if !ok {
		return fmt.Errorf("unknown default generator %q", s.cfg.DefaultGenerator)
	}

	args, err := generator.ParseArgs("")
	if err != nil {
		return fmt.Errorf("parse default generator args: %w", err)
	}

	lvl, err := generator.Generate(gen, "main", defaultWorldWidth, defaultWorldHeight, defaultWorldLength, args)
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
		if err := s.worlds.Save(worldPath); err != nil {
			s.logger.Error("save world", "path", worldPath, "error", err)
		}
	}
	if policyPath != "" {
		if err := s.players.SavePolicy(policyPath); err != nil {
			s.logger.Error("save player policy", "path", policyPath, "error", err)
		}
	}
}

func (s *Server) autosaveLoop(ctx context.Context, worldPath, policyPath string) {
	interval := s.cfg.Autosave
	if interval <= 0 {
		interval = 60 * time.Second
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
