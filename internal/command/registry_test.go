package command

import "testing"

func TestRegistryExecuteDeniesAdminCommandsWithoutPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if got, handled := registry.Execute(Context{}, "/tp 1 2 3"); !handled || got != "permission denied" {
		t.Fatalf("Execute = %q handled=%v, want permission denied", got, handled)
	}
}

func TestRegistryExecuteAllowsAdminCommandsWithPermission(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	ctx := Context{
		Authority: testAuthority(true),
		World:     testWorld{t: t},
	}

	if got, handled := registry.Execute(ctx, "/tp 1 2 3 4 5"); !handled || got != "teleported to 1 2 3" {
		t.Fatalf("Execute = %q handled=%v, want teleport confirmation", got, handled)
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
