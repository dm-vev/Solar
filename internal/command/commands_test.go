package command

import "testing"

func TestSetBlockCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		ctx   Context
		args  []string
		reply string
	}{
		{
			name:  "valid block",
			ctx:   Context{World: stubWorld{}, Authority: testAuthority(true)},
			args:  []string{"1", "2", "3", "7"},
			reply: "set block at 1 2 3 to 7",
		},
		{
			name:  "wrong arg count",
			ctx:   Context{World: stubWorld{}},
			args:  []string{"1", "2"},
			reply: "usage: /setblock x y z block",
		},
		{
			name:  "invalid x",
			ctx:   Context{World: stubWorld{}},
			args:  []string{"abc", "2", "3", "7"},
			reply: "invalid x",
		},
		{
			name:  "no world service",
			ctx:   Context{},
			args:  []string{"1", "2", "3", "7"},
			reply: "block updates are unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := setBlockCommand(tt.ctx, tt.args)
			if !ok {
				t.Fatalf("setBlockCommand returned handled=false")
			}
			if !containsStr(got, tt.reply) {
				t.Fatalf("got %q, want prefix %q", got, tt.reply)
			}
		})
	}
}

func TestWhereCommand(t *testing.T) {
	t.Parallel()

	ctx := Context{
		Username: "alice",
		Position: Position{X: 10, Y: 20, Z: 30},
	}
	got, ok := whereCommand(ctx, nil)
	if !ok || got != "alice is at 10 20 30" {
		t.Fatalf("whereCommand = %q handled=%v", got, ok)
	}
}

func TestHelpCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	handler := helpCommand(registry)
	got, ok := handler(Context{}, nil)
	if !ok || !containsStr(got, "commands:") {
		t.Fatalf("helpCommand = %q handled=%v", got, ok)
	}
}

func TestKickCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ctx  Context
		args []string
		want string
	}{
		{
			name: "no args",
			ctx:  Context{Moderation: stubModeration{}},
			args: nil,
			want: "usage: /kick name",
		},
		{
			name: "no moderation service",
			ctx:  Context{},
			args: []string{"alice"},
			want: "kick is unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := kickCommand(tt.ctx, tt.args)
			if !containsStr(got, tt.want) {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBanCommand(t *testing.T) {
	t.Parallel()

	ctx := Context{Moderation: stubModeration{banned: true}}
	got, _ := banCommand(ctx, []string{"alice", "griefing"})
	if !containsStr(got, "banned alice") {
		t.Fatalf("got %q", got)
	}
}

func TestUnbanCommand(t *testing.T) {
	t.Parallel()

	ctx := Context{Moderation: stubModeration{}}
	got, _ := unbanCommand(ctx, []string{"alice"})
	if !containsStr(got, "unbanned") {
		t.Fatalf("got %q", got)
	}
}

func TestWhitelistCommand(t *testing.T) {
	t.Parallel()

	t.Run("status", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Moderation: stubModeration{enabled: true}}
		got, _ := whitelistCommand(ctx, nil)
		if got != "whitelist: on" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no args no moderation", func(t *testing.T) {
		t.Parallel()
		ctx := Context{}
		got, _ := whitelistCommand(ctx, nil)
		if got != "whitelist status unavailable" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestPlayersCommand(t *testing.T) {
	t.Parallel()

	t.Run("with players", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Players: stubDirectory{names: []string{"alice", "bob"}}}
		got, _ := playersCommand(ctx, nil)
		if got != "players: alice, bob" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no players", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Players: stubDirectory{}}
		got, _ := playersCommand(ctx, nil)
		if got != "players: none" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no directory", func(t *testing.T) {
		t.Parallel()
		ctx := Context{}
		got, _ := playersCommand(ctx, nil)
		if got != "player list is unavailable" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestNewLevelCommand(t *testing.T) {
	t.Parallel()

	t.Run("not enough args", func(t *testing.T) {
		t.Parallel()
		ctx := Context{World: stubWorld{}}
		got, _ := newLevelCommand(ctx, []string{"a", "b"})
		if !containsStr(got, "usage:") {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no world service", func(t *testing.T) {
		t.Parallel()
		ctx := Context{}
		got, _ := newLevelCommand(ctx, nil)
		if got != "world generation is unavailable" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestSaveCommand(t *testing.T) {
	t.Parallel()

	t.Run("no persistence", func(t *testing.T) {
		t.Parallel()
		ctx := Context{}
		got, _ := saveCommand(ctx, nil)
		if got != "save is unavailable" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("save ok", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Persistence: stubPersistence{}}
		got, _ := saveCommand(ctx, nil)
		if got != "save queued" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestRegistryUnknownCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	got, handled := registry.Execute(Context{}, "/nonexistent")
	if !handled || !containsStr(got, "unknown command") {
		t.Fatalf("got %q handled=%v", got, handled)
	}
}

func TestRegistryEmptyCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	_, handled := registry.Execute(Context{}, "")
	if handled {
		t.Fatal("expected handled=false for empty command")
	}
}

func TestRegisterRejectsEmpty(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	registry.Register("", func(_ Context, _ []string) (string, bool) { return "", false })
	registry.Register("test", nil)

	got, _ := registry.Execute(Context{}, "/test")
	if !containsStr(got, "unknown command") {
		t.Fatalf("nil handler should not be registered; got %q", got)
	}
}

type stubWorld struct{}

func (stubWorld) SetBlock(_, _, _ int, _ byte) bool                     { return true }
func (stubWorld) MovePlayer(_, _, _ int, _, _ byte) bool                { return true }
func (stubWorld) SetSpawn(_, _, _ int, _, _ byte) bool                  { return true }
func (stubWorld) GenerateWorld(_, _ string, _, _, _ int, _ string) bool { return true }

type stubModeration struct {
	enabled bool
	banned  bool
}

func (m stubModeration) KickPlayer(_, _ string) bool     { return true }
func (m stubModeration) BanPlayer(_, _ string) bool      { return m.banned }
func (m stubModeration) UnbanPlayer(_ string) bool       { return true }
func (m stubModeration) WhitelistEnabled() bool          { return m.enabled }
func (m stubModeration) SetWhitelistEnabled(_ bool) bool { return true }
func (m stubModeration) WhitelistAdd(_ string) bool      { return true }
func (m stubModeration) WhitelistRemove(_ string) bool   { return true }

type stubDirectory struct {
	names       []string
	whitelisted []string
}

func (d stubDirectory) ListPlayers() []string     { return d.names }
func (d stubDirectory) ListWhitelisted() []string { return d.whitelisted }

type stubPersistence struct{}

func (stubPersistence) SaveState() bool { return true }

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
