package storage

import (
	"path/filepath"
	"testing"
)

func TestLocalStorePaths(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewLocalStore(root)

	if got := store.WorldsDir(); got != filepath.Join(root, "worlds") {
		t.Fatalf("WorldsDir = %q, want %q", got, filepath.Join(root, "worlds"))
	}
	if got := store.PlayersDir(); got != filepath.Join(root, "players") {
		t.Fatalf("PlayersDir = %q, want %q", got, filepath.Join(root, "players"))
	}
}

func TestWorldFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewLocalStore(root)

	if got := store.WorldFile("main"); got != filepath.Join(root, "worlds", "main.json") {
		t.Fatalf("WorldFile(main) = %q, want %q", got, filepath.Join(root, "worlds", "main.json"))
	}
	if got := store.WorldFile("arena"); got != filepath.Join(root, "worlds", "arena.json") {
		t.Fatalf("WorldFile(arena) = %q, want %q", got, filepath.Join(root, "worlds", "arena.json"))
	}
}

func TestPlayerPolicyFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	store := NewLocalStore(root)

	if got := store.PlayerPolicyFile(); got != filepath.Join(root, "players", "policy.json") {
		t.Fatalf("PlayerPolicyFile = %q, want %q", got, filepath.Join(root, "players", "policy.json"))
	}
}
