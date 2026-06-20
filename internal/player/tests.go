package player

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestPolicyBanAndUnban(t *testing.T) {
	t.Parallel()

	p := NewPolicy()

	if !p.Ban("alice", "griefing") {
		t.Fatal("Ban alice returned false")
	}
	if allowed, reason := p.CanJoin("ALICE"); allowed || reason != "griefing" {
		t.Fatalf("banned CanJoin = %v %q", allowed, reason)
	}
	if !p.Unban("alice") {
		t.Fatal("Unban alice returned false")
	}
	if allowed, _ := p.CanJoin("alice"); !allowed {
		t.Fatal("alice should be allowed after unban")
	}
}

func TestPolicyWhitelist(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	p.SetWhitelistEnabled(true)

	if allowed, reason := p.CanJoin("bob"); allowed || reason != "server is whitelisted" {
		t.Fatalf("whitelist CanJoin bob = %v %q", allowed, reason)
	}
	if !p.WhitelistAdd("bob") {
		t.Fatal("WhitelistAdd bob returned false")
	}
	if allowed, _ := p.CanJoin("BOB"); !allowed {
		t.Fatal("whitelisted CanJoin bob should be allowed")
	}
	if !p.WhitelistRemove("bob") {
		t.Fatal("WhitelistRemove bob returned false")
	}
}

func TestPolicyOperators(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	if !p.AddOperators("root") {
		t.Fatal("AddOperators root returned false")
	}
	if !p.IsOperator("ROOT") {
		t.Fatal("IsOperator ROOT should be true")
	}
	if p.IsOperator("nobody") {
		t.Fatal("IsOperator nobody should be false")
	}
	if allowed, _ := p.CanJoin("root"); !allowed {
		t.Fatal("operator should always be allowed to join")
	}
}

func TestPolicyOperatorBypassesWhitelist(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	p.SetWhitelistEnabled(true)
	p.AddOperators("admin")

	if allowed, _ := p.CanJoin("admin"); !allowed {
		t.Fatal("operator should bypass whitelist")
	}
}

func TestPolicyInvalidName(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	if allowed, reason := p.CanJoin(""); allowed || reason != "invalid username" {
		t.Fatalf("CanJoin empty = %v %q", allowed, reason)
	}
	if allowed, reason := p.CanJoin("   "); allowed || reason != "invalid username" {
		t.Fatalf("CanJoin whitespace = %v %q", allowed, reason)
	}
}

func TestPolicyBanDefaultReason(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	p.Ban("griefer", "")
	if allowed, reason := p.CanJoin("griefer"); allowed || reason != "banned" {
		t.Fatalf("default ban reason = %v %q", allowed, reason)
	}
}

func TestPolicyWhitelistToggle(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	if p.WhitelistEnabled() {
		t.Fatal("whitelist should be disabled by default")
	}
	if !p.SetWhitelistEnabled(true) {
		t.Fatal("SetWhitelistEnabled(true) should return true (changed)")
	}
	if p.SetWhitelistEnabled(true) {
		t.Fatal("SetWhitelistEnabled(true) again should return false (no change)")
	}
}

func TestPolicyNames(t *testing.T) {
	t.Parallel()

	p := NewPolicy()
	p.Ban("Alice", "")
	p.WhitelistAdd("Bob")
	p.AddOperators("Carol")

	bans := p.BanNames()
	if len(bans) != 1 || bans[0] != "Alice" {
		t.Fatalf("BanNames = %v", bans)
	}
	wl := p.WhitelistNames()
	if len(wl) != 1 || wl[0] != "Bob" {
		t.Fatalf("WhitelistNames = %v", wl)
	}
	ops := p.OperatorNames()
	if len(ops) != 1 || ops[0] != "Carol" {
		t.Fatalf("OperatorNames = %v", ops)
	}
}

func TestRegistryOnlinePlayers(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Add("alice", 1)
	r.Add("bob", 2)

	if r.Count() != 2 {
		t.Fatalf("Count = %d, want 2", r.Count())
	}

	names := r.OnlineNames()
	if len(names) != 2 {
		t.Fatalf("OnlineNames len = %d, want 2", len(names))
	}

	player, ok := r.Get("alice")
	if !ok || player.Name != "alice" || player.EntityID != 1 {
		t.Fatalf("Get alice = %+v ok=%v", player, ok)
	}

	r.Remove("alice")
	if r.Count() != 1 {
		t.Fatalf("Count after remove = %d, want 1", r.Count())
	}
}

func TestRegistryMarkSpawned(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Add("alice", 1)
	r.MarkSpawned("alice")

	player, _ := r.Get("alice")
	if !player.Spawned {
		t.Fatal("player should be marked as spawned")
	}
}

func TestRegistryAddEmptyName(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	if r.Add("", 1) {
		t.Fatal("Add empty name should return false")
	}
	if r.Count() != 0 {
		t.Fatalf("Count = %d, want 0", r.Count())
	}
}

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
