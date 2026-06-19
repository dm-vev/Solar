package config

import "time"

// Config provides read-write access to server configuration.
// Changes take effect immediately where applicable.
type Config interface {
	// Server
	Name() string
	SetName(name string)
	MOTD() string
	SetMOTD(motd string)
	MaxPlayers() int
	SetMaxPlayers(n int)
	ConnectRate() int
	SetConnectRate(rate int)

	// Network
	ReadTimeout() time.Duration
	WriteTimeout() time.Duration
	TCPNoDelay() bool
	SetTCPNoDelay(enable bool)

	// Simulation
	TickInterval() time.Duration
	SetTickInterval(d time.Duration)

	// World
	DefaultWidth() int
	DefaultHeight() int
	DefaultLength() int
	MaxBlocks() int

	// Autosave
	AutosaveInterval() time.Duration
	SetAutosaveInterval(d time.Duration)

	// Operators
	Operators() []string
	AddOperator(name string) bool
	RemoveOperator(name string) bool

	// Whitelist
	WhitelistEnabled() bool
	SetWhitelistEnabled(enabled bool)
}
