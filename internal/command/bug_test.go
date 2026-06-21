// bug_test.go reproduces and verifies fixes for specific bugs.

package command

import (
	"testing"
)

// BUG: /help should filter by rank — guest should not see operator commands.
func TestBug_HelpFiltersByRank(t *testing.T) {
	registry := NewRegistry()
	handler := helpCommand(registry)

	// Use a real formatter so we can check the command names.
	formatTr := func(key string, args ...any) string {
		if key == "command.help.list" && len(args) > 0 {
			if s, ok := args[0].(string); ok {
				return s
			}
		}
		return key
	}

	// Guest rank (0) — should not see /tp (but /tpa is ok)
	ctx := Context{Tr: formatTr, RankLevel: func() int { return 0 }}
	got, _ := handler(ctx, nil)
	t.Logf("guest help: %s", got)
	// Check for exact "/tp " or "/tp," — not "/tpa"
	if contains(got, "/tp,") || contains(got, "/tp ") || (got[len(got)-3:] == "/tp") {
		t.Fatal("guest /help should not show /tp")
	}

	// Operator rank (80) — should see /tp
	ctx = Context{Tr: formatTr, RankLevel: func() int { return 80 }}
	got, _ = handler(ctx, nil)
	if !contains(got, "/tp") {
		t.Fatal("operator /help should show /tp")
	}
}

// BUG: Execute should reject commands above player's rank.
func TestBug_ExecuteRejectsByRank(t *testing.T) {
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return 0 }, // guest
	}
	got, handled := registry.Execute(ctx, "/tp 1 2 3")
	if !handled {
		t.Fatal("should be handled (permission denied message)")
	}
	if got != "command.shared.permission_denied" {
		t.Fatalf("guest /tp: got %q, want permission_denied", got)
	}
}

// BUG: Execute should allow commands at player's rank.
func TestBug_ExecuteAllowsAtRank(t *testing.T) {
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return 80 }, // operator
		Authority: testAuthority(true),
		World:     testWorld{t: t},
	}
	got, handled := registry.Execute(ctx, "/tp 1 2 3 4 5")
	if !handled {
		t.Fatal("should be handled")
	}
	if got != "command.teleport.done" {
		t.Fatalf("operator /tp: got %q, want command.teleport.done", got)
	}
}

// BUG: Unknown command should return error message, not empty.
func TestBug_UnknownCommand(t *testing.T) {
	registry := NewRegistry()
	ctx := Context{Tr: testTr, RankLevel: func() int { return 0 }}
	got, handled := registry.Execute(ctx, "/nonexistent")
	if !handled {
		t.Fatal("unknown command should be handled=true")
	}
	if got != "command.shared.unknown" {
		t.Fatalf("unknown command: got %q, want command.shared.unknown", got)
	}
}

// BUG: Empty command should not be handled.
func TestBug_EmptyCommand(t *testing.T) {
	registry := NewRegistry()
	ctx := Context{Tr: testTr}
	_, handled := registry.Execute(ctx, "")
	if handled {
		t.Fatal("empty command should not be handled")
	}
}

// BUG: SetCommandRank should update existing command's rank requirement.
func TestBug_SetCommandRank(t *testing.T) {
	registry := NewRegistry()
	// /help is guest by default. Set it to operator.
	if !registry.SetCommandRank("help", 80) {
		t.Fatal("SetCommandRank should succeed for existing command")
	}
	ctx := Context{Tr: testTr, RankLevel: func() int { return 0 }}
	got, _ := registry.Execute(ctx, "/help")
	if got != "command.shared.permission_denied" {
		t.Fatalf("guest /help after SetCommandRank: got %q, want permission_denied", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// BUG: DrawService stubs need to track PlaceBlock calls for permission tests.
// This is a compile-time check — if the interface changes, stubs must update.
func TestBug_DrawServiceInterface(t *testing.T) {
	var _ DrawService = stubDraw{}
}
