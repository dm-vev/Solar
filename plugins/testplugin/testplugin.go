// Package testplugin is a minimal example plugin for Solar.
// It demonstrates the plugin API: event subscription, command
// registration, and player interaction.
package testplugin

import (
	"strings"

	"github.com/solar-mc/solar/plugin"
)

type TestPlugin struct{}

func (TestPlugin) Name() string { return "test" }

func (TestPlugin) Init() error { return nil }

func (TestPlugin) Enable(s plugin.Server) error {
	// Subscribe to player chat — filter "badword"
	plugin.OnPlayerChat.Register(func(ctx *plugin.Context, data plugin.PlayerChatData) {
		if strings.Contains(strings.ToLower(*data.Message), "badword") {
			data.Player.Message("&cPlease don't use that word!")
			ctx.Cancel()
		}
	}, plugin.PriorityNormal)

	// Subscribe to block change — log placements
	plugin.OnBlockChanged.Register(func(ctx *plugin.Context, data plugin.BlockChangedData) {
		if data.Placing {
			s.BroadcastMessage("&a" + data.Player.Name() + " placed a block!")
		}
	}, plugin.PriorityLow)

	// Subscribe to player connect — welcome message
	plugin.OnPlayerConnect.Register(func(ctx *plugin.Context, data plugin.PlayerConnectData) {
		data.Player.Message("&eWelcome to the server, " + data.Player.Name() + "!")
	}, plugin.PriorityNormal)

	// Subscribe to tick — periodic announcement
	plugin.OnTick.Register(func(ctx *plugin.Context, data plugin.TickData) {
		if data.Tick%400 == 0 { // every 20 seconds at 20 TPS
			s.BroadcastMessage("&7[Server] Tick count: " + formatUint(data.Tick))
		}
	}, plugin.PriorityLow)

	// Register a custom command
	s.RegisterCommand("hello", "Say hello to the server", func(p plugin.Player, args []string) string {
		return "&eHello, " + p.Name() + "!"
	})

	return nil
}

func (TestPlugin) Disable() error { return nil }

func init() {
	plugin.Register("test", TestPlugin{})
}

func formatUint(n uint64) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
