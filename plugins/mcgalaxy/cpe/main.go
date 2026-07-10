//go:build plugin

// Package main provides MCGalaxy CPE commands as an independent Solar plugin.
package main

import (
	"fmt"
	"strconv"

	"github.com/solar-mc/solar/plugin"
)

type cpePlugin struct {
	server plugin.Server
}

func (p *cpePlugin) Name() string { return "mcgalaxy-cpe" }
func (p *cpePlugin) Init() error  { return nil }

func (p *cpePlugin) Enable(server plugin.Server) error {
	p.server = server
	if !server.RegisterCommandSpec(plugin.CommandSpec{
		Name: "reachdistance", Aliases: []string{"reach"},
		Help: "/reachdistance [0-1023] - sets your reach", MinRank: plugin.RankAdvBuilder,
		Handler: reach,
	}) {
		return fmt.Errorf("register /reachdistance")
	}
	return nil
}

func (p *cpePlugin) Disable() error {
	if p.server != nil {
		p.server.UnregisterCommand("reachdistance")
	}
	return nil
}

func reach(player plugin.Player, args []string) string {
	if len(args) != 1 {
		return "&WUsage: /reachdistance [0-1023]"
	}
	distance, err := strconv.ParseFloat(args[0], 64)
	if err != nil || distance < 0 || distance > 1023 {
		return "&WDistance must be between 0 and 1023"
	}
	if !player.SupportsCPE("ClickDistance") || player.CPE() == nil {
		return "&WYour client does not support ClickDistance"
	}
	player.CPE().SetClickDistance(distance)
	return fmt.Sprintf("&SSet your reach distance to %g blocks", distance)
}

func init() { plugin.Register("mcgalaxy-cpe", &cpePlugin{}) }
