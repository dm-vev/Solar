package color

import "testing"

func TestColorRoundtrip(t *testing.T) {
	for _, name := range []string{"black", "navy", "green", "teal", "maroon",
		"purple", "gold", "silver", "gray", "blue", "lime", "aqua", "red",
		"pink", "yellow", "white"} {
		c := ParseColor(name)
		if ColorName(c) != name {
			t.Fatalf("ParseColor(%q)=%q -> ColorName=%q, want %q", name, c, ColorName(c), name)
		}
	}
}

func TestStripColor(t *testing.T) {
	in := Colorize(ColorRed, "hello") + Colorize(ColorBlue, "world")
	want := "helloworld"
	if got := StripColor(in); got != want {
		t.Fatalf("StripColor(%q)=%q, want %q", in, got, want)
	}
	if got := StripColor("&S&H&T&I&W&s&h&t&i&w"); got != "" {
		t.Fatalf("StripColor system codes = %q, want empty", got)
	}
}

func TestParseColorUnknown(t *testing.T) {
	if ParseColor("nope") != ColorWhite {
		t.Fatal("unknown color should default to white")
	}
}
