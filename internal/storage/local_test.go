package storage

import (
	"path/filepath"
	"testing"
)

func TestLocalStoreConfigureAndBlockDBPaths(t *testing.T) {
	root := t.TempDir()
	store := NewLocalStore(root)
	store.Configure("maps", "users", "policy-custom.json", ".cw")

	if got := store.WorldsDir(); got != filepath.Join(root, "maps") {
		t.Fatalf("WorldsDir = %q", got)
	}
	if got := store.PlayersDir(); got != filepath.Join(root, "users") {
		t.Fatalf("PlayersDir = %q", got)
	}
	if got := store.WorldFile("main"); got != filepath.Join(root, "maps", "main.cw") {
		t.Fatalf("WorldFile = %q", got)
	}
	if got := store.PlayerPolicyFile(); got != filepath.Join(root, "users", "policy-custom.json") {
		t.Fatalf("PlayerPolicyFile = %q", got)
	}
	if got := store.BlockDBsDir(); got != filepath.Join(root, "blockdb") {
		t.Fatalf("BlockDBsDir = %q", got)
	}
	if got := store.BlockDBFile("main"); got != filepath.Join(root, "blockdb", "main.cbdb") {
		t.Fatalf("BlockDBFile = %q", got)
	}
}

func TestLocalStoreFileNamesCannotEscapeRoot(t *testing.T) {
	root := t.TempDir()
	store := NewLocalStore(root)

	if got := store.WorldFile("../evil"); got != filepath.Join(root, "worlds", "evil.swld") {
		t.Fatalf("WorldFile escaped or sanitized incorrectly: %q", got)
	}
	if got := store.BlockDBFile("../evil"); got != filepath.Join(root, "blockdb", "evil.cbdb") {
		t.Fatalf("BlockDBFile escaped or sanitized incorrectly: %q", got)
	}
	if ValidName("../evil") {
		t.Fatal("ValidName accepted traversal")
	}
	if !ValidName("main") {
		t.Fatal("ValidName rejected main")
	}
}
