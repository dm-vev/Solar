// bug_test.go reproduces and verifies fixes for specific bugs.

package player

import (
	"path/filepath"
	"testing"
	"time"
)

// BUG: Undo then redo then undo should work (3-step cycle).
func TestBug_UndoRedoCycle(t *testing.T) {
	u := NewUndoStack(10)
	u.Push([]BlockChange{{0, 0, 0, 1, 2}})
	u.Push([]BlockChange{{1, 0, 0, 3, 4}})

	// Undo both.
	u.Undo()
	u.Undo()
	if u.CanUndo() {
		t.Fatal("should not be able to undo after 2 undos")
	}

	// Redo one.
	r := u.Redo()
	if len(r) != 1 || r[0].X != 0 {
		t.Fatalf("redo returned wrong batch: %+v", r)
	}

	// Undo the redo.
	r = u.Undo()
	if len(r) != 1 || r[0].X != 0 {
		t.Fatalf("undo after redo returned wrong batch: %+v", r)
	}
}

// BUG: Policy ban then unban should allow join.
func TestBug_BanUnbanCycle(t *testing.T) {
	p := NewPolicy()
	p.Ban("alice", "testing")
	allowed, _ := p.CanJoin("alice")
	if allowed {
		t.Fatal("banned player should not be allowed")
	}
	p.Unban("alice")
	allowed, _ = p.CanJoin("alice")
	if !allowed {
		t.Fatal("unbanned player should be allowed")
	}
}

// BUG: Operator should bypass whitelist.
func TestBug_OperatorBypassWhitelist(t *testing.T) {
	p := NewPolicy()
	p.SetWhitelistEnabled(true)
	p.AddOperators("admin")
	allowed, _ := p.CanJoin("admin")
	if !allowed {
		t.Fatal("operator should bypass whitelist")
	}
	allowed, _ = p.CanJoin("guest")
	if allowed {
		t.Fatal("non-whitelisted guest should be rejected")
	}
}

// BUG: PlayerProps persistence — save and load should preserve all fields.
func TestBug_PlayerPropsPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "policy.json")
	r := NewRegistry()
	r.SetProps("alice", PlayerProps{
		Color:  "&e",
		Model:  "creeper",
		Frozen: true,
		Muted:  true,
		AFK:    false,
	})
	ab := true
	r.SetProps("alice", PlayerProps{
		Color:      "&e",
		Model:      "creeper",
		Frozen:     true,
		Muted:      true,
		AFK:        false,
		AllowBuild: &ab,
	})
	if err := r.SavePolicy(path); err != nil {
		t.Fatalf("SavePolicy: %v", err)
	}

	r2 := NewRegistry()
	if err := r2.LoadPolicy(path); err != nil {
		t.Fatalf("LoadPolicy: %v", err)
	}
	props := r2.GetProps("alice")
	if props.Color != "&e" {
		t.Fatalf("Color: got %q, want &e", props.Color)
	}
	if props.Model != "creeper" {
		t.Fatalf("Model: got %q, want creeper", props.Model)
	}
	if !props.Frozen {
		t.Fatal("Frozen should be true")
	}
	if !props.Muted {
		t.Fatal("Muted should be true")
	}
	if props.AllowBuild == nil || *props.AllowBuild != true {
		t.Fatal("AllowBuild should be true")
	}
}

// BUG: SpamChecker mute should auto-expire.
func TestBug_SpamMuteExpiry(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:      true,
		ChatMax:      1,
		ChatWindow:   10 * time.Second,
		Action:       SpamActionMute,
		MuteDuration: 50 * time.Millisecond,
	})
	c.CheckChat("player") // 1st message, ok
	c.CheckChat("player") // 2nd message, exceeds → mute
	if !c.IsMuted("player") {
		t.Fatal("should be muted")
	}
	time.Sleep(60 * time.Millisecond)
	if c.IsMuted("player") {
		t.Fatal("mute should have expired")
	}
}

// BUG: SpamChecker block check should not trigger on normal build rate.
func TestBug_SpamBlockNormalRate(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:     true,
		BlockMax:    100,
		BlockWindow: 2 * time.Second,
		Action:      SpamActionKick,
	})
	for i := 0; i < 99; i++ {
		r := c.CheckBlock("builder")
		if r.Exceeded {
			t.Fatalf("block check %d should not exceed (limit 100)", i)
		}
	}
}
