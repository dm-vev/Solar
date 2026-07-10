package command

import (
	"testing"
	"time"

	"github.com/solar-mc/solar/plugin/playerdb"
)

// testTr is a no-op translator for tests: returns the key as-is.
// This verifies commands use i18n keys, not hardcoded strings.
var testTr = func(key string, args ...any) string { return key }

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
			ctx:   Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr},
			args:  []string{"1", "2", "3", "7"},
			reply: "command.setblock.done",
		},
		{
			name:  "wrong arg count",
			ctx:   Context{World: stubWorld{}, Tr: testTr},
			args:  []string{"1", "2"},
			reply: "command.setblock.usage",
		},
		{
			name:  "invalid x",
			ctx:   Context{World: stubWorld{}, Tr: testTr},
			args:  []string{"abc", "2", "3", "7"},
			reply: "command.shared.invalid_x",
		},
		{
			name:  "no world service",
			ctx:   Context{Tr: testTr},
			args:  []string{"1", "2", "3", "7"},
			reply: "command.teleport.unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, ok := setBlockCommand(tt.ctx, tt.args)
			if !ok {
				t.Fatalf("setBlockCommand returned handled=false")
			}
			if got != tt.reply {
				t.Fatalf("got %q, want %q", got, tt.reply)
			}
		})
	}
}

func TestWhereCommand(t *testing.T) {
	t.Parallel()

	ctx := Context{
		Username: "alice",
		Position: Position{X: 10, Y: 20, Z: 30},
		Tr:       testTr,
	}
	got, ok := whereCommand(ctx, nil)
	if !ok || got != "command.where.position" {
		t.Fatalf("whereCommand = %q handled=%v", got, ok)
	}
}

func TestHelpCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	handler := helpCommand(registry)
	got, ok := handler(Context{Tr: testTr}, nil)
	if !ok || got != "command.help.list" {
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
			ctx:  Context{Moderation: stubModeration{}, Tr: testTr},
			args: nil,
			want: "command.kick.usage",
		},
		{
			name: "no moderation service",
			ctx:  Context{Tr: testTr},
			args: []string{"alice"},
			want: "command.kick.unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, _ := kickCommand(tt.ctx, tt.args)
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBanCommand(t *testing.T) {
	t.Parallel()

	ctx := Context{Moderation: stubModeration{banned: true}, Tr: testTr}
	got, _ := banCommand(ctx, []string{"alice", "griefing"})
	if got != "command.ban.done_reason" {
		t.Fatalf("got %q", got)
	}
}

func TestUnbanCommand(t *testing.T) {
	t.Parallel()

	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := unbanCommand(ctx, []string{"alice"})
	if got != "command.unban.done" {
		t.Fatalf("got %q", got)
	}
}

func TestWhitelistCommand(t *testing.T) {
	t.Parallel()

	t.Run("status", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Moderation: stubModeration{enabled: true}, Tr: testTr}
		got, _ := whitelistCommand(ctx, nil)
		if got != "command.whitelist.status.on" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no args no moderation", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Tr: testTr}
		got, _ := whitelistCommand(ctx, nil)
		if got != "command.whitelist.unavailable" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestPlayersCommand(t *testing.T) {
	t.Parallel()

	t.Run("with players", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Players: stubDirectory{names: []string{"alice", "bob"}}, Tr: testTr}
		got, _ := playersCommand(ctx, nil)
		if got != "command.players.list" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no players", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Players: stubDirectory{}, Tr: testTr}
		got, _ := playersCommand(ctx, nil)
		if got != "command.players.none" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no directory", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Tr: testTr}
		got, _ := playersCommand(ctx, nil)
		if got != "command.players.unavailable" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestNewLevelCommand(t *testing.T) {
	t.Parallel()

	t.Run("not enough args", func(t *testing.T) {
		t.Parallel()
		ctx := Context{World: stubWorld{}, Tr: testTr}
		got, _ := newLevelCommand(ctx, []string{"a", "b"})
		if got != "command.newlvl.usage" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("no world service", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Tr: testTr}
		got, _ := newLevelCommand(ctx, nil)
		if got != "command.newlvl.unavailable" {
			t.Fatalf("got %q", got)
		}
	})
}

func TestSaveCommand(t *testing.T) {
	t.Parallel()

	t.Run("no persistence", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Tr: testTr}
		got, _ := saveCommand(ctx, nil)
		if got != "command.save.unavailable" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("save ok", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Persistence: stubPersistence{}, Tr: testTr}
		got, _ := saveCommand(ctx, nil)
		if got != "command.save.queued" {
			t.Fatalf("got %q", got)
		}
	})

	t.Run("save failed", func(t *testing.T) {
		t.Parallel()
		ctx := Context{Persistence: failingPersistence{}, Tr: testTr}
		got, _ := saveCommand(ctx, nil)
		if got != "command.save.failed" {
			t.Fatalf("got %q, want 'command.save.failed'", got)
		}
	})
}

func TestRegistryUnknownCommand(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	got, handled := registry.Execute(Context{Tr: testTr}, "/nonexistent")
	if !handled || got != "command.shared.unknown" {
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

	got, _ := registry.Execute(Context{Tr: testTr}, "/test")
	if got != "command.shared.unknown" {
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

func (m stubModeration) KickPlayer(_, _ string) bool      { return true }
func (m stubModeration) BanPlayer(_, _ string) bool       { return m.banned }
func (m stubModeration) UnbanPlayer(_ string) bool        { return true }
func (m stubModeration) WhitelistEnabled() bool           { return m.enabled }
func (m stubModeration) SetWhitelistEnabled(_ bool) bool  { return true }
func (m stubModeration) WhitelistAdd(_ string) bool       { return true }
func (m stubModeration) WhitelistRemove(_ string) bool    { return true }
func (m stubModeration) MutePlayer(_ string) bool         { return true }
func (m stubModeration) UnmutePlayer(_ string) bool       { return true }
func (m stubModeration) FreezePlayer(_ string) bool       { return true }
func (m stubModeration) UnfreezePlayer(_ string) bool     { return true }
func (m stubModeration) ToggleAFK(_ string) (bool, bool)  { return true, true }
func (m stubModeration) ToggleHide(_ string) (bool, bool) { return true, true }

// stubPlayerDB implements PlayerLookup.
type stubPlayerDB struct{}

func (stubPlayerDB) Lookup(name string) *playerdb.PlayerEntry {
	if name == "alice" {
		return &playerdb.PlayerEntry{Name: "Alice", LoginCount: 5}
	}
	return nil
}

// stubServerInfo implements ServerInfo.
type stubServerInfo struct{}

func (stubServerInfo) ServerName() string    { return "TestServer" }
func (stubServerInfo) MOTD() string          { return "Test MOTD" }
func (stubServerInfo) OnlineCount() int      { return 1 }
func (stubServerInfo) MaxPlayers() int       { return 128 }
func (stubServerInfo) LevelCount() int       { return 1 }
func (stubServerInfo) Uptime() time.Duration { return 5 * time.Minute }

// stubTeleport implements TeleportService.
type stubTeleport struct{}

func (stubTeleport) SpawnPoint() (int, int, int, byte, byte) { return 0, 0, 0, 0, 0 }
func (stubTeleport) TeleportToPlayer(name string) bool       { return true }
func (stubTeleport) RequestTeleport(name string) (TPAStatus, string) {
	return TPARequestSent, name
}
func (stubTeleport) RespondTeleport(accept bool) (TPAStatus, string) {
	if accept {
		return TPAAccepted, "alice"
	}
	return TPADenied, "alice"
}
func (stubTeleport) SummonPlayer(name string) bool { return true }
func (stubTeleport) Back() bool                    { return false }

// stubChat implements ChatService.
type stubChat struct{}

func (stubChat) Me(action string)                {}
func (stubChat) Whisper(target, msg string) bool { return true }
func (stubChat) Ignore(name string) (bool, bool) { return true, true }

// stubDraw implements DrawService.
type stubDraw struct{}

func (stubDraw) StartSelection(markCount int, callback func([][3]int)) bool { return true }
func (stubDraw) GetBlockAt(x, y, z int) (byte, bool)                        { return 0, true }
func (stubDraw) PlaceBlock(x, y, z int, block byte) bool                    { return true }
func (stubDraw) LevelDims() (int, int, int)                                 { return 128, 64, 128 }
func (stubDraw) CopyRegion(min, max [3]int) bool                            { return true }
func (stubDraw) HasClipboard() bool                                         { return false }
func (stubDraw) PasteAt(origin [3]int, pasteAir bool) int                   { return 0 }
func (stubDraw) SetSpecialBlock(x, y, z int, entry SpecialBlockEntry) bool  { return true }
func (stubDraw) BeginBatch()                                                {}
func (stubDraw) RecordChange(x, y, z int, old, new byte)                    {}
func (stubDraw) CommitBatch()                                               {}
func (stubDraw) Undo() []UndoChange                                         { return nil }
func (stubDraw) Redo() []UndoChange                                         { return nil }
func (stubDraw) DrawLimit() int                                             { return 4096 }
func (stubDraw) CanPlace(block byte) bool                                   { return true }
func (stubDraw) CanDelete(block byte) bool                                  { return true }

type stubDirectory struct {
	names       []string
	whitelisted []string
}

func (d stubDirectory) ListPlayers() []string     { return d.names }
func (d stubDirectory) ListWhitelisted() []string { return d.whitelisted }

type stubPersistence struct{}

func (stubPersistence) SaveState() bool { return true }

type failingPersistence struct{}

func (failingPersistence) SaveState() bool { return false }
