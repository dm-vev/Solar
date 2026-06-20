package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTOML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "language.toml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestGet_Basic(t *testing.T) {
	path := writeTOML(t, `
[en]
muted = "You are muted."
kick = "Kicked by %s"
`)
	i := New("en")
	if err := i.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := i.Get("en", "muted"); got != "You are muted." {
		t.Fatalf("Get(en, muted) = %q", got)
	}
	if got := i.Get("en", "kick", "Admin"); got != "Kicked by Admin" {
		t.Fatalf("Get(en, kick, Admin) = %q", got)
	}
}

func TestGet_Fallback(t *testing.T) {
	path := writeTOML(t, `
[en]
muted = "You are muted."

[ru]
muted = "Вы в муте."
`)
	i := New("en")
	if err := i.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	// ru has muted
	if got := i.Get("ru", "muted"); got != "Вы в муте." {
		t.Fatalf("Get(ru, muted) = %q", got)
	}
	// ru doesn't have kick — falls back to en
	if got := i.Get("ru", "kick"); got != "kick" {
		t.Fatalf("Get(ru, kick) should fall back to key, got %q", got)
	}
}

func TestGet_MissingKey(t *testing.T) {
	i := New("en")
	if got := i.Get("en", "nonexistent"); got != "nonexistent" {
		t.Fatalf("Get missing key = %q, want 'nonexistent'", got)
	}
}

func TestLanguages(t *testing.T) {
	path := writeTOML(t, `
[en]
a = "x"

[ru]
a = "y"
`)
	i := New("en")
	if err := i.Load(path); err != nil {
		t.Fatalf("Load: %v", err)
	}
	langs := i.Languages()
	if len(langs) != 2 {
		t.Fatalf("Languages = %v, want 2", langs)
	}
}
