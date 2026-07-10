// special_commands.go implements special block placement commands.
//
// /mb <message>              — place a message block (shows text on step)
// /portal <x> <y> <z> [level] — place a portal (teleports on step)
// /door                      — place a physics-driven log door

package command

import (
	"strconv"
	"strings"
)

// mbCommand — /mb <message>
// Places a message block. The next block click sets the location.
// When a player steps on it, the message is displayed.
func mbCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if len(args) < 1 {
		return ctx.tr("command.mb.usage"), true
	}
	msg := strings.Join(args, " ")

	ctx.Draw.StartSelection(1, func(marks [][3]int) {
		x, y, z := marks[0][0], marks[0][1], marks[0][2]
		ctx.Draw.SetSpecialBlock(x, y, z, SpecialBlockEntry{
			Type:    1, // SpecialMessage
			Message: msg,
		})
	})
	return ctx.tr("command.draw.select1"), true
}

// portalCommand — /portal <x> <y> <z> [level]
// Places a portal block. The next block click sets the portal location.
// When a player steps on it, they teleport to the destination.
func portalCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}
	if len(args) < 3 {
		return ctx.tr("command.portal.usage"), true
	}
	dx, err := strconv.Atoi(args[0])
	if err != nil {
		return ctx.tr("command.shared.invalid_x", err), true
	}
	dy, err := strconv.Atoi(args[1])
	if err != nil {
		return ctx.tr("command.shared.invalid_y", err), true
	}
	dz, err := strconv.Atoi(args[2])
	if err != nil {
		return ctx.tr("command.shared.invalid_z", err), true
	}
	level := ""
	if len(args) > 3 {
		level = args[3]
	}

	ctx.Draw.StartSelection(1, func(marks [][3]int) {
		x, y, z := marks[0][0], marks[0][1], marks[0][2]
		ctx.Draw.SetSpecialBlock(x, y, z, SpecialBlockEntry{
			Type:        2, // SpecialPortal
			PortalX:     dx,
			PortalY:     dy,
			PortalZ:     dz,
			PortalLevel: level,
		})
	})
	return ctx.tr("command.draw.select1"), true
}

// doorCommand — /door
// Places a door block. The next block click sets the location.
// Deleting the solid door activates it when physics is enabled.
func doorCommand(ctx Context, args []string) (string, bool) {
	if ctx.Draw == nil {
		return ctx.tr("command.draw.unavailable"), true
	}

	ctx.Draw.StartSelection(1, func(marks [][3]int) {
		x, y, z := marks[0][0], marks[0][1], marks[0][2]
		ctx.Draw.PlaceBlock(x, y, z, 111) // Door_Log
	})
	return ctx.tr("command.draw.select1"), true
}
