package player

import "testing"

func TestPolicyBanWhitelistAndOperators(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if allowed, reason := registry.CanJoin("alice"); !allowed || reason != "" {
		t.Fatalf("CanJoin alice = %v %q", allowed, reason)
	}
	if !registry.Ban("alice", "bad") {
		t.Fatal("Ban alice returned false")
	}
	if allowed, reason := registry.CanJoin("ALICE"); allowed || reason != "bad" {
		t.Fatalf("banned CanJoin alice = %v %q", allowed, reason)
	}
	if !registry.Unban("alice") {
		t.Fatal("Unban alice returned false")
	}
	registry.SetWhitelistEnabled(true)
	if allowed, reason := registry.CanJoin("bob"); allowed || reason != "server is whitelisted" {
		t.Fatalf("whitelist CanJoin bob = %v %q", allowed, reason)
	}
	if !registry.WhitelistAdd("bob") {
		t.Fatal("WhitelistAdd bob returned false")
	}
	if allowed, reason := registry.CanJoin("BOB"); !allowed || reason != "" {
		t.Fatalf("whitelisted CanJoin bob = %v %q", allowed, reason)
	}
	if !registry.AddOperators("root") {
		t.Fatal("AddOperators root returned false")
	}
	if allowed, reason := registry.CanJoin("root"); !allowed || reason != "" {
		t.Fatalf("operator CanJoin root = %v %q", allowed, reason)
	}
}
