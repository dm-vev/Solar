// Package command defines the CommandHandler function signature that
// plugins use to register custom commands.
package command

import "github.com/solar-mc/solar/plugin/player"

// CommandHandler is the function signature for custom plugin commands.
// The player is the command sender, args are the space-split arguments
// after the command name. Return a reply string (empty for no reply).
//
//nolint:revive // intentional: re-exported as plugin.X
type CommandHandler func(p player.Player, args []string) string

// CommandSpec describes a plugin command and its permission metadata.
// Name and Aliases omit the leading slash. MinRank uses Solar permission
// levels (guest=0, builder=30, advbuilder=50, operator=80, admin=100, owner=120).
type CommandSpec struct {
	Name    string
	Aliases []string
	Help    string
	MinRank int
	Handler CommandHandler
}
