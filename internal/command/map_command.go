package command

import (
	"fmt"
	"strconv"
	"strings"
)

// mapCommand — /map [property] [value]
// Views or sets per-level environment properties.
// Properties: weather, sky, cloud, fog, ambient, diffuse, motd
func mapCommand(ctx Context, args []string) (string, bool) {
	if ctx.LevelEnv == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	if len(args) == 0 {
		return showMapInfo(ctx), true
	}

	prop := strings.ToLower(args[0])
	switch prop {
	case "weather":
		if len(args) != 2 {
			return ctx.tr("command.map.weather.usage"), true
		}
		w, err := strconv.Atoi(args[1])
		if err != nil || w < 0 || w > 2 {
			return ctx.tr("command.map.weather.invalid"), true
		}
		ctx.LevelEnv.SetWeather(w)
		return ctx.tr("command.map.weather.set", w), true

	case "sky", "cloud", "fog", "ambient", "diffuse":
		if len(args) != 4 {
			return ctx.tr("command.map.color.usage", prop), true
		}
		r, err := parseColorComp(args[1])
		if err != nil {
			return ctx.tr("command.map.color.invalid", "r"), true
		}
		g, err := parseColorComp(args[2])
		if err != nil {
			return ctx.tr("command.map.color.invalid", "g"), true
		}
		b, err := parseColorComp(args[3])
		if err != nil {
			return ctx.tr("command.map.color.invalid", "b"), true
		}
		slot := colorSlot(prop)
		ctx.LevelEnv.SetEnvColor(slot, r, g, b)
		return ctx.tr("command.map.color.set", prop, r, g, b), true

	case "motd":
		if len(args) < 2 {
			return ctx.tr("command.map.motd.usage"), true
		}
		motd := strings.Join(args[1:], " ")
		ctx.LevelEnv.SetMOTD(motd)
		return ctx.tr("command.map.motd.set"), true

	default:
		return ctx.tr("command.map.usage"), true
	}
}

func showMapInfo(ctx Context) string {
	weather := ctx.LevelEnv.Weather()
	motd := ctx.LevelEnv.MOTD()
	weatherName := "sunny"
	switch weather {
	case 1:
		weatherName = "raining"
	case 2:
		weatherName = "snowing"
	}
	var sb strings.Builder
	sb.WriteString(ctx.tr("command.map.info.weather", weatherName))
	sb.WriteString("\n")
	for _, slot := range []struct {
		name string
		idx  int
	}{
		{"sky", 0}, {"cloud", 1}, {"fog", 2}, {"ambient", 3}, {"diffuse", 4},
	} {
		r, g, b, set := ctx.LevelEnv.GetEnvColor(slot.idx)
		if set {
			sb.WriteString(fmt.Sprintf("&a%s: &7%d %d %d\n", slot.name, r, g, b))
		} else {
			sb.WriteString(fmt.Sprintf("&7%s: default\n", slot.name))
		}
	}
	if motd != "" {
		sb.WriteString(ctx.tr("command.map.info.motd", motd))
	} else {
		sb.WriteString("&7motd: default")
	}
	return sb.String()
}

func colorSlot(name string) int {
	switch name {
	case "sky":
		return 0
	case "cloud":
		return 1
	case "fog":
		return 2
	case "ambient":
		return 3
	case "diffuse":
		return 4
	}
	return 0
}

func parseColorComp(s string) (byte, error) {
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 || v > 255 {
		return 0, fmt.Errorf("invalid")
	}
	return byte(v), nil
}
