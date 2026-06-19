package event

import "github.com/solar-mc/solar/plugin/player"

// Server event data types and global event instances.

// ConnectionReceivedData fires when a new TCP connection arrives.
// Cancelable — if cancelled, the connection is dropped.
type ConnectionReceivedData struct {
	RemoteAddr string
}

// TickData fires every server tick (default 50ms / 20 TPS).
type TickData struct {
	Tick uint64
}

// ShutdownData fires when the server is shutting down.
type ShutdownData struct {
	Restarting bool
	Reason     string
}

// PluginsLoadedData fires after all plugins have been enabled.
type PluginsLoadedData struct{}

// ConfigUpdatedData — server configuration has been updated
type ConfigUpdatedData struct{}

// ChatSysData — system chat message (no source player). Modifiable.
type ChatSysData struct{ Message *string }

// ChatFromData — chat message from a specific player. Modifiable.
type ChatFromData struct {
	Source  player.Player
	Message *string
}

// ChatData — all chat (fires for both system and player chat). Modifiable.
type ChatData struct {
	Source  player.Player
	Message *string
}

// CpePluginMessageData — player sends a CPE PluginMessage packet
type CpePluginMessageData struct {
	Player  player.Player
	Channel byte
	Data    []byte
}

var (
	OnConnectionReceived = NewEvent[ConnectionReceivedData]()
	OnTick               = NewEvent[TickData]()
	OnShutdown           = NewEvent[ShutdownData]()
	OnPluginsLoaded      = NewEvent[PluginsLoadedData]()
	OnConfigUpdated      = NewEvent[ConfigUpdatedData]()
	OnChatSys            = NewEvent[ChatSysData]()
	OnChatFrom           = NewEvent[ChatFromData]()
	OnChat               = NewEvent[ChatData]()
	OnPluginMessage      = NewEvent[CpePluginMessageData]()
)
