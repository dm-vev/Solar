package color

// Color represents a Classic Minecraft color code.
// Classic uses &-prefixed codes: &0-&9, &a-&f (16 colors).
// Special system codes: &S (server default), &H (help text),
// &T (help syntax), &I (IRC), &W (warning).
type Color string

// Standard color constants.
const (
	ColorBlack  Color = "&0"
	ColorNavy   Color = "&1"
	ColorGreen  Color = "&2"
	ColorTeal   Color = "&3"
	ColorMaroon Color = "&4"
	ColorPurple Color = "&5"
	ColorGold   Color = "&6"
	ColorSilver Color = "&7"
	ColorGray   Color = "&8"
	ColorBlue   Color = "&9"
	ColorLime   Color = "&a"
	ColorAqua   Color = "&b"
	ColorRed    Color = "&c"
	ColorPink   Color = "&d"
	ColorYellow Color = "&e"
	ColorWhite  Color = "&f"
)

// System color constants.
const (
	ColorServer  Color = "&S" // server default
	ColorHelp    Color = "&H" // help description
	ColorSyntax  Color = "&T" // help syntax
	ColorIRC     Color = "&I" // IRC color
	ColorWarning Color = "&W" // warning/error
)

// Colorize wraps text with the given color code.
func Colorize(c Color, text string) string {
	return string(c) + text + "&f"
}

// StripColor removes all color codes from a string.
func StripColor(text string) string {
	var out []byte
	for i := 0; i < len(text); i++ {
		if text[i] == '&' && i+1 < len(text) {
			c := text[i+1]
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') ||
				c == 'S' || c == 'H' || c == 'T' || c == 'I' || c == 'W' ||
				c == 's' || c == 'h' || c == 't' || c == 'i' || c == 'w' {
				i++
				continue
			}
		}
		out = append(out, text[i])
	}
	return string(out)
}

// ParseColor converts a color name (e.g. "red", "yellow") to a Color code.
// Returns ColorWhite if the name is not recognized.
func ParseColor(name string) Color {
	switch name {
	case "black":
		return ColorBlack
	case "navy", "darkblue":
		return ColorNavy
	case "green", "darkgreen":
		return ColorGreen
	case "teal", "darkaqua", "darkcyan":
		return ColorTeal
	case "maroon", "darkred":
		return ColorMaroon
	case "purple", "darkpurple":
		return ColorPurple
	case "gold", "orange", "darkyellow":
		return ColorGold
	case "silver", "lightgray", "lightgrey":
		return ColorSilver
	case "gray", "grey", "darkgray", "darkgrey":
		return ColorGray
	case "blue":
		return ColorBlue
	case "lime", "lightgreen":
		return ColorLime
	case "aqua", "cyan", "lightblue":
		return ColorAqua
	case "red":
		return ColorRed
	case "pink", "magenta":
		return ColorPink
	case "yellow":
		return ColorYellow
	case "white":
		return ColorWhite
	default:
		return ColorWhite
	}
}

// ColorName converts a Color code back to its name.
//
//nolint:revive // intentional: re-exported as plugin.X
func ColorName(c Color) string {
	switch c {
	case ColorBlack:
		return "black"
	case ColorNavy:
		return "navy"
	case ColorGreen:
		return "green"
	case ColorTeal:
		return "teal"
	case ColorMaroon:
		return "maroon"
	case ColorPurple:
		return "purple"
	case ColorGold:
		return "gold"
	case ColorSilver:
		return "silver"
	case ColorGray:
		return "gray"
	case ColorBlue:
		return "blue"
	case ColorLime:
		return "lime"
	case ColorAqua:
		return "aqua"
	case ColorRed:
		return "red"
	case ColorPink:
		return "pink"
	case ColorYellow:
		return "yellow"
	case ColorWhite:
		return "white"
	case ColorServer:
		return "server"
	case ColorHelp:
		return "help"
	case ColorSyntax:
		return "syntax"
	case ColorIRC:
		return "irc"
	case ColorWarning:
		return "warning"
	default:
		return "white"
	}
}
