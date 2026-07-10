//go:build plugin

// Package main is a .so plugin example for Solar.
//
// Build with:
//
//	go build -buildmode=plugin -o data/plugins/soplug.so ./plugins/soplug
//
// The server loads it when [plugins] enabled = true in server.toml.
// init() runs during plugin.Open and calls plugin.Register — same
// registration mechanism as compile-time plugins.
package main

import (
	"fmt"
	"strings"

	"github.com/solar-mc/solar/plugin"
)

type SoPlug struct{}

func (SoPlug) Name() string { return "soplug" }
func (SoPlug) Init() error  { return nil }

func (SoPlug) Enable(s plugin.Server) error {
	if _, err := s.PluginDataDir("soplug"); err != nil {
		return err
	}
	plugin.OnPlayerChat.Register(func(ctx *plugin.Context, data plugin.PlayerChatData) {
		if strings.Contains(strings.ToLower(*data.Message), "pizza") {
			data.Player.Message("&cNo pizza talk!")
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)

	plugin.OnPlayerConnect.Register(func(ctx *plugin.Context, data plugin.PlayerConnectData) {
		data.Player.Message("&a.so plugin says hi, " + data.Player.Name() + "!")
	}, plugin.PriorityNormal)

	if !s.RegisterCommandSpec(plugin.CommandSpec{
		Name:    "so",
		Aliases: []string{"soplug"},
		Help:    "usage: /so",
		MinRank: plugin.RankBuilder,
		Handler: func(p plugin.Player, args []string) string {
			return "&eLoaded from .so: " + p.Name()
		},
	}) {
		return fmt.Errorf("register /so")
	}
	if !s.RegisterCommandSpec(plugin.CommandSpec{
		Name:    "soselect",
		Help:    "usage: /soselect",
		MinRank: plugin.RankBuilder,
		Handler: func(p plugin.Player, args []string) string {
			if !p.SelectBlocks(2, func(marks []plugin.BlockPos) {
				p.Message(fmt.Sprintf("&aSelected %d,%d,%d to %d,%d,%d",
					marks[0].X, marks[0].Y, marks[0].Z,
					marks[1].X, marks[1].Y, marks[1].Z))
			}) {
				return "&cCould not start block selection"
			}
			return "&ePlace or delete two blocks to mark the region"
		},
	}) {
		return fmt.Errorf("register /soselect")
	}
	return nil
}

func (SoPlug) Disable() error { return nil }

func init() {
	plugin.Register("soplug", SoPlug{})
}
