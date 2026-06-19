package player

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyRoundTrip(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Ban("Alice", "griefing")
	registry.WhitelistAdd("Bob")
	registry.AddOperators("Root")
	registry.SetWhitelistEnabled(true)

	path := filepath.Join(t.TempDir(), "policy.json")
	if err := registry.SavePolicy(path); err != nil {
		t.Fatalf("SavePolicy returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("saved policy missing: %v", err)
	}

	reloaded := NewRegistry()
	if err := reloaded.LoadPolicy(path); err != nil {
		t.Fatalf("LoadPolicy returned error: %v", err)
	}

	if allowed, reason := reloaded.CanJoin("Alice"); allowed || reason != "griefing" {
		t.Fatalf("ban did not survive reload: allowed=%v reason=%q", allowed, reason)
	}
	if allowed, reason := reloaded.CanJoin("Bob"); !allowed || reason != "" {
		t.Fatalf("whitelist entry did not survive reload: allowed=%v reason=%q", allowed, reason)
	}
	if allowed, reason := reloaded.CanJoin("Root"); !allowed || reason != "" {
		t.Fatalf("operator entry did not survive reload: allowed=%v reason=%q", allowed, reason)
	}
	if allowed, reason := reloaded.CanJoin("Charlie"); allowed || reason != "server is whitelisted" {
		t.Fatalf("whitelist enforcement did not survive reload: allowed=%v reason=%q", allowed, reason)
	}
}
