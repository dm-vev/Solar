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
	"strings"

	"github.com/solar-mc/solar/plugin"
)

type SoPlug struct{}

func (SoPlug) Name() string { return "soplug" }
func (SoPlug) Init() error  { return nil }

func (SoPlug) Enable(s plugin.Server) error {
	plugin.OnPlayerChat.Register(func(ctx *plugin.Context, data plugin.PlayerChatData) {
		if strings.Contains(strings.ToLower(*data.Message), "pizza") {
			data.Player.Message("&cNo pizza talk!")
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)

	plugin.OnPlayerConnect.Register(func(ctx *plugin.Context, data plugin.PlayerConnectData) {
		data.Player.Message("&a.so plugin says hi, " + data.Player.Name() + "!")
	}, plugin.PriorityNormal)

	s.RegisterCommand("so", "so plugin command", func(p plugin.Player, args []string) string {
		return "&eLoaded from .so: " + p.Name()
	})
	return nil
}

func (SoPlug) Disable() error { return nil }

func init() {
	plugin.Register("soplug", SoPlug{})
}
