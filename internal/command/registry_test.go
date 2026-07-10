package command

import "testing"

func TestRegistryExecuteDeniesAdminCommandsWithoutPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if got, handled := registry.Execute(Context{Tr: testTr}, "/tp 1 2 3"); !handled || got != "command.shared.permission_denied" {
		t.Fatalf("Execute = %q handled=%v, want command.shared.permission_denied", got, handled)
	}
}

func TestRegistryExecuteAllowsAdminCommandsWithPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	ctx := Context{
		Authority: testAuthority(true),
		World:     testWorld{t: t},
		Tr:        testTr,
		RankLevel: func() int { return 80 }, // operator
	}

	if got, handled := registry.Execute(ctx, "/tp 1 2 3 4 5"); !handled || got != "command.teleport.done" {
		t.Fatalf("Execute = %q handled=%v, want command.teleport.done", got, handled)
	}
}

func TestRegistryRegisterManyIfAbsent(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	called := false
	handler := func(_ Context, args []string) (string, bool) {
		called = len(args) == 1 && args[0] == "value"
		return "ok", true
	}
	if !registry.RegisterManyIfAbsent([]string{"Example", "Ex"}, 30, "example help", handler) {
		t.Fatal("RegisterManyIfAbsent returned false")
	}
	if registry.RegisterManyIfAbsent([]string{"other", "help"}, 0, "", handler) {
		t.Fatal("registration overwrote a built-in command")
	}
	if _, exists := registry.handlers["other"]; exists {
		t.Fatal("failed atomic registration left a partial command")
	}

	guest := Context{Tr: testTr, RankLevel: func() int { return 0 }}
	if got, _ := registry.Execute(guest, "/ex value"); got != "command.shared.permission_denied" {
		t.Fatalf("guest alias result = %q", got)
	}
	builder := Context{Tr: testTr, RankLevel: func() int { return 30 }}
	if got, _ := registry.Execute(builder, "/ex value"); got != "ok" || !called {
		t.Fatalf("builder alias result = %q called=%v", got, called)
	}
	if got, _ := registry.Execute(builder, "/help example"); got != "example help" {
		t.Fatalf("plugin help = %q", got)
	}
	if !registry.UnregisterMany([]string{"example", "ex"}) {
		t.Fatal("UnregisterMany returned false")
	}
	if _, exists := registry.handlers["example"]; exists {
		t.Fatal("primary command remained registered")
	}
	if _, exists := registry.handlers["ex"]; exists {
		t.Fatal("alias remained registered")
	}
}

type testAuthority bool

func (a testAuthority) CanAdmin() bool { return bool(a) }

type testWorld struct{ t *testing.T }

func (w testWorld) SetBlock(_, _, _ int, _ byte) bool { return false }

func (w testWorld) MovePlayer(x, y, z int, yaw, pitch byte) bool {
	w.t.Helper()
	if x != 1 || y != 2 || z != 3 || yaw != 4 || pitch != 5 {
		w.t.Fatalf("MovePlayer got %d %d %d %d %d", x, y, z, yaw, pitch)
	}
	return true
}

func (w testWorld) SetSpawn(_, _, _ int, _, _ byte) bool { return false }

func (w testWorld) GenerateWorld(_, _ string, _, _, _ int, _ string) bool { return false }
