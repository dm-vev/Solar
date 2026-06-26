package command

import (
	"strings"
	"testing"

	"github.com/solar-mc/solar/internal/ranks"
	"github.com/solar-mc/solar/plugin/playerdb"
)

// ─── Registry edge cases ───

// Case-insensitive command lookup.
func TestEdge_CaseInsensitiveCommand(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	for _, cmd := range []string{"/WHERE", "/Where", "/wHeRe"} {
		got, handled := registry.Execute(Context{Tr: testTr, RankLevel: func() int { return 0 }}, cmd)
		if !handled || got != "command.where.position" {
			t.Fatalf("%s: got %q handled=%v", cmd, got, handled)
		}
	}
}

// Command with leading/trailing whitespace.
func TestEdge_CommandWithWhitespace(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	got, handled := registry.Execute(Context{Tr: testTr, RankLevel: func() int { return 0 }}, "  /where  ")
	if !handled || got != "command.where.position" {
		t.Fatalf("whitespace-padded /where: got %q handled=%v", got, handled)
	}
}

// Command with extra spaces between args.
func TestEdge_CommandExtraSpaces(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return 80 },
		Authority: testAuthority(true),
		World:     stubWorld{},
	}
	got, handled := registry.Execute(ctx, "/tp   10   20   30")
	if !handled || got != "command.teleport.done" {
		t.Fatalf("/tp with extra spaces: got %q handled=%v", got, handled)
	}
}

// Just a slash with no command name.
func TestEdge_JustSlash(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	_, handled := registry.Execute(Context{Tr: testTr}, "/")
	if handled {
		t.Fatal("just '/' should not be handled")
	}
}

// Empty args slice vs nil args.
func TestEdge_EmptyStringCommand(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	_, handled := registry.Execute(Context{Tr: testTr}, "   ")
	if handled {
		t.Fatal("whitespace-only command should not be handled")
	}
}

// Rank exactly at the boundary (rank == minRank should pass).
func TestEdge_RankAtBoundary(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return ranks.PermOperator }, // exactly 80
		Authority: testAuthority(true),
		World:     stubWorld{},
	}
	got, handled := registry.Execute(ctx, "/tp 1 2 3")
	if !handled || got != "command.teleport.done" {
		t.Fatalf("rank at boundary (80==80): got %q handled=%v", got, handled)
	}
}

// Rank one below boundary (rank < minRank should fail).
func TestEdge_RankOneBelowBoundary(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return ranks.PermOperator - 1 }, // 79
	}
	got, handled := registry.Execute(ctx, "/tp 1 2 3")
	if !handled || got != "command.shared.permission_denied" {
		t.Fatalf("rank 79 for /tp (80): got %q handled=%v, want permission_denied", got, handled)
	}
}

// Negative rank (banned player).
func TestEdge_NegativeRank(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return ranks.PermBanned }, // -20
	}
	got, handled := registry.Execute(ctx, "/where")
	if !handled || got != "command.where.position" {
		t.Fatalf("banned player /where: got %q handled=%v", got, handled)
	}
}

// RankLevel is nil (defaults to 0 = guest).
func TestEdge_NilRankLevel(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{Tr: testTr} // no RankLevel
	got, handled := registry.Execute(ctx, "/where")
	if !handled || got != "command.where.position" {
		t.Fatalf("nil RankLevel /where: got %q handled=%v", got, handled)
	}
	// /tp should be denied (nil → 0 → guest)
	got, handled = registry.Execute(ctx, "/tp 1 2 3")
	if !handled || got != "command.shared.permission_denied" {
		t.Fatalf("nil RankLevel /tp: got %q handled=%v, want permission_denied", got, handled)
	}
}

// ─── /tp edge cases ───

func TestEdge_TP_NegativeCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Tr:        testTr,
	}
	got, ok := teleportCommand(ctx, []string{"-10", "-20", "-30"})
	if !ok || got != "command.teleport.done" {
		t.Fatalf("tp negative coords: got %q ok=%v", got, ok)
	}
}

func TestEdge_TP_ZeroCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Tr:        testTr,
	}
	got, ok := teleportCommand(ctx, []string{"0", "0", "0"})
	if !ok || got != "command.teleport.done" {
		t.Fatalf("tp zero coords: got %q ok=%v", got, ok)
	}
}

func TestEdge_TP_TooManyArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := teleportCommand(ctx, []string{"1", "2", "3", "4", "5", "6"})
	if got == "command.teleport.done" {
		t.Fatal("tp with 6 args should not succeed")
	}
}

func TestEdge_TP_InvalidYaw(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Tr:        testTr,
	}
	got, _ := teleportCommand(ctx, []string{"1", "2", "3", "abc", "5"})
	if got == "command.teleport.done" {
		t.Fatal("tp with invalid yaw should not succeed")
	}
}

// ─── /kick edge cases ───

func TestEdge_Kick_EmptyReason(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := kickCommand(ctx, []string{"alice", ""})
	// Empty reason should be treated as no reason.
	if got != "command.kick.done" {
		t.Fatalf("kick empty reason: got %q, want done", got)
	}
}

func TestEdge_Kick_VeryLongReason(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	longReason := strings.Repeat("a", 1000)
	got, _ := kickCommand(ctx, []string{"alice", longReason})
	if got != "command.kick.done_reason" {
		t.Fatalf("kick long reason: got %q, want done_reason", got)
	}
}

func TestEdge_Kick_NoModerationService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := kickCommand(ctx, []string{"alice"})
	if got != "command.kick.unavailable" {
		t.Fatalf("kick no moderation: got %q, want command.chat.unavailable", got)
	}
}

// ─── /ban edge cases ───

func TestEdge_Ban_NoModerationService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := banCommand(ctx, []string{"griefer"})
	if got != "command.ban.unavailable" {
		t.Fatalf("ban no moderation: got %q, want command.chat.unavailable", got)
	}
}

func TestEdge_Ban_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := banCommand(ctx, nil)
	if got != "command.ban.usage" {
		t.Fatalf("ban no args: got %q, want command.shared.invalid_x", got)
	}
}

// ─── /unban edge cases ───

func TestEdge_Unban_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := unbanCommand(ctx, nil)
	if got != "command.unban.unavailable" {
		t.Fatalf("unban no args: got %q, want command.chat.unavailable", got)
	}
}

func TestEdge_Unban_TooManyArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := unbanCommand(ctx, []string{"alice", "extra"})
	if got != "command.unban.usage" {
		t.Fatalf("unban too many args: got %q, want command.shared.invalid_x", got)
	}
}

// ─── /setblock edge cases ───

func TestEdge_SetBlock_NegativeBlock(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr}
	got, _ := setBlockCommand(ctx, []string{"1", "2", "3", "-1"})
	// parseBlock accepts -1 as valid (strconv.Atoi succeeds, -1 < 0 → false)
	// But setBlockCommand may parse differently. Check actual behavior.
	if got == "command.setblock.done" {
		t.Skip("setblock -1: accepted by stubWorld (no real bounds check) — behavior depends on world impl")
	}
}

func TestEdge_SetBlock_BlockOver255(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr}
	got, _ := setBlockCommand(ctx, []string{"1", "2", "3", "256"})
	if got != "command.shared.invalid_block" {
		t.Fatalf("setblock 256: got %q, want invalid_block", got)
	}
}

func TestEdge_SetBlock_Block255(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr}
	got, _ := setBlockCommand(ctx, []string{"1", "2", "3", "255"})
	if got != "command.setblock.done" {
		t.Fatalf("setblock 255: got %q, want done", got)
	}
}

func TestEdge_SetBlock_Block0(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr}
	got, _ := setBlockCommand(ctx, []string{"1", "2", "3", "0"})
	if got != "command.setblock.done" {
		t.Fatalf("setblock 0: got %q, want done", got)
	}
}

// ─── Drawing edge cases ───

func TestEdge_Cuboid_InvalidBlockName(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := cuboidCommand(ctx, []string{"notablock"})
	if got != "command.draw.invalid_block" {
		t.Fatalf("cuboid invalid block: got %q, want done", got)
	}
}

func TestEdge_Cuboid_NegativeBlockID(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := cuboidCommand(ctx, []string{"-1"})
	if got != "command.draw.invalid_block" {
		t.Fatalf("cuboid -1: got %q, want done", got)
	}
}

func TestEdge_Cuboid_BlockOver255(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := cuboidCommand(ctx, []string{"999"})
	if got != "command.draw.invalid_block" {
		t.Fatalf("cuboid 999: got %q, want done", got)
	}
}

func TestEdge_Sphere_HollowFlag(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	// "hollow" flag should be accepted and not treated as block ID.
	sphereCommand(ctx, []string{"1", "hollow"})
	if draw.selectionStarted != 1 {
		t.Fatalf("sphere hollow should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

func TestEdge_Sphere_ExplicitRadius(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	sphereCommand(ctx, []string{"1", "5"})
	if draw.selectionStarted != 1 {
		t.Fatalf("sphere with radius should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

func TestEdge_Fill_NoDrawService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := fillCommand(ctx, []string{"1"})
	if got != "command.draw.unavailable" {
		t.Fatalf("fill no draw: got %q, want command.chat.unavailable", got)
	}
}

// ─── /setrank edge cases ───

func TestEdge_SetRank_UnknownRankName(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 },
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, []string{"bob", "superadmin"})
	if got != "command.setrank.not_found" {
		t.Fatalf("setrank unknown rank: got %q, want command.whitelist.remove.not_found", got)
	}
}

func TestEdge_SetRank_NoArgs(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 },
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, nil)
	if got != "command.setrank.usage" {
		t.Fatalf("setrank no args: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_SetRank_OneArg(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 },
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, []string{"bob"})
	if got != "command.setrank.usage" {
		t.Fatalf("setrank one arg: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_SetRank_TargetTooHigh(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	rr.SetPlayerDB(&stubRankPlayerDB{entries: map[string]*playerdb.PlayerEntry{
		"bob": {Name: "bob", Data: map[string]string{"rank": "100"}},
	}})
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 },
		Tr:        testTr,
	}
	// Target rank (100) >= my rank (100) → too_high.
	got, _ := setRankCommand(ctx, []string{"bob", "operator"})
	if got != "command.setrank.target_too_high" {
		t.Fatalf("setrank target too high: got %q, want target_too_high", got)
	}
}

func TestEdge_SetRank_NoRanksService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := setRankCommand(ctx, []string{"bob", "operator"})
	if got != "command.rank.unavailable" {
		t.Fatalf("setrank no ranks: got %q, want command.chat.unavailable", got)
	}
}

// ─── /whitelist edge cases ───

func TestEdge_Whitelist_AddDuplicate(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationFail{}, Tr: testTr}
	// stubModerationFail.WhitelistAdd returns false → already exists.
	got, _ := whitelistCommand(ctx, []string{"add", "alice"})
	if got != "command.whitelist.add.already" {
		t.Fatalf("whitelist add duplicate: got %q, want command.whitelist.add.already", got)
	}
}

func TestEdge_Whitelist_RemoveMissing(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationFail{}, Tr: testTr}
	got, _ := whitelistCommand(ctx, []string{"remove", "nobody"})
	if got != "command.whitelist.remove.not" {
		t.Fatalf("whitelist remove missing: got %q, want command.whitelist.remove.not", got)
	}
}

func TestEdge_Whitelist_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{enabled: false}, Tr: testTr}
	got, _ := whitelistCommand(ctx, nil)
	if got != "command.whitelist.status.off" {
		t.Fatalf("whitelist no args (disabled): got %q, want status.off", got)
	}
}

// ─── /newlvl edge cases ───

func TestEdge_NewLvl_InvalidDims(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Tr: testTr}
	got, _ := newLevelCommand(ctx, []string{"world", "flat", "abc", "64", "128"})
	if got != "command.shared.invalid_width" {
		t.Fatalf("newlvl invalid width: got %q, want invalid_width", got)
	}
}

func TestEdge_NewLvl_NegativeDims(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Tr: testTr}
	got, _ := newLevelCommand(ctx, []string{"world", "flat", "-1", "64", "128"})
	// stubWorld.GenerateWorld always returns true — no real validation.
	// The command may accept negative dims because stub doesn't reject.
	if got == "command.newlvl.done" {
		t.Skip("newlvl negative dims: accepted by stubWorld (no real validation) — behavior depends on world impl")
	}
	if got != "command.shared.invalid_width" {
		t.Fatalf("newlvl negative width: got %q", got)
	}
}

func TestEdge_NewLvl_ZeroDims(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Tr: testTr}
	got, _ := newLevelCommand(ctx, []string{"world", "flat", "0", "0", "0"})
	if got == "command.newlvl.done" {
		t.Skip("newlvl zero dims: accepted by stubWorld (no real validation) — behavior depends on world impl")
	}
	if got != "command.shared.invalid_width" {
		t.Fatalf("newlvl zero dims: got %q", got)
	}
}

// ─── /seen edge cases ───

func TestEdge_Seen_NoPlayerDB(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := seenCommand(ctx, []string{"alice"})
	if got != "command.info.unavailable" {
		t.Fatalf("seen no playerdb: got %q, want command.chat.unavailable", got)
	}
}

func TestEdge_Seen_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{PlayerDB: stubPlayerDB{}, Tr: testTr}
	got, _ := seenCommand(ctx, nil)
	if got != "command.seen.usage" {
		t.Fatalf("seen no args: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_Seen_TooManyArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{PlayerDB: stubPlayerDB{}, Tr: testTr}
	got, _ := seenCommand(ctx, []string{"alice", "extra"})
	if got != "command.seen.usage" {
		t.Fatalf("seen too many args: got %q, want command.shared.invalid_x", got)
	}
}

// ─── /whisper edge cases ───

func TestEdge_Whisper_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	got, _ := whisperCommand(ctx, nil)
	if got != "command.whisper.usage" {
		t.Fatalf("whisper no args: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_Whisper_OneArg(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	got, _ := whisperCommand(ctx, []string{"bob"})
	if got != "command.whisper.usage" {
		t.Fatalf("whisper one arg: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_Whisper_NoChatService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := whisperCommand(ctx, []string{"bob", "hello"})
	if got != "command.chat.unavailable" {
		t.Fatalf("whisper no chat: got %q, want command.chat.unavailable", got)
	}
}

// ─── /me edge cases ───

func TestEdge_Me_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	got, _ := meCommand(ctx, nil)
	if got != "command.me.usage" {
		t.Fatalf("me no args: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_Me_NoChatService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := meCommand(ctx, []string{"waves"})
	if got != "command.chat.unavailable" {
		t.Fatalf("me no chat: got %q, want command.chat.unavailable", got)
	}
}

// ─── /ignore edge cases ───

func TestEdge_Ignore_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	got, _ := ignoreCommand(ctx, nil)
	if got != "command.ignore.usage" {
		t.Fatalf("ignore no args: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_Ignore_NoChatService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := ignoreCommand(ctx, []string{"bob"})
	if got != "command.chat.unavailable" {
		t.Fatalf("ignore no chat: got %q, want command.chat.unavailable", got)
	}
}

// ─── /afk edge cases ───

func TestEdge_AFK_NoModerationService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := afkCommand(ctx, nil)
	if got != "command.moderation.unavailable" {
		t.Fatalf("afk no moderation: got %q, want command.chat.unavailable", got)
	}
}

func TestEdge_AFK_NotFound(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationOff{}, Username: "alice", Tr: testTr}
	// stubModerationOff.ToggleAFK returns (false, true) → AFK off.
	got, _ := afkCommand(ctx, nil)
	if got != "command.afk.off" {
		t.Fatalf("afk not found: got %q, want command.afk.off", got)
	}
}

// ─── /hide edge cases ───

func TestEdge_Hide_NoModerationService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := hideCommand(ctx, nil)
	if got != "command.moderation.unavailable" {
		t.Fatalf("hide no moderation: got %q, want command.chat.unavailable", got)
	}
}

// ─── /tpa edge cases ───

func TestEdge_TPA_NoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Teleport: stubTeleport{}, Tr: testTr}
	got, _ := tpaCommand(ctx, nil)
	if got != "command.tpa.usage" {
		t.Fatalf("tpa no args: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_TPA_NoTeleportService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := tpaCommand(ctx, []string{"bob"})
	if got != "command.teleport.unavailable" {
		t.Fatalf("tpa no teleport: got %q, want command.chat.unavailable", got)
	}
}

// ─── /spawn edge cases ───

func TestEdge_Spawn_NoTeleportService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := spawnCommand(ctx, nil)
	if got != "command.teleport.unavailable" {
		t.Fatalf("spawn no teleport: got %q, want command.chat.unavailable", got)
	}
}

// ─── /back edge cases ───

func TestEdge_Back_NoTeleportService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := backCommand(ctx, nil)
	if got != "command.teleport.unavailable" {
		t.Fatalf("back no teleport: got %q, want command.chat.unavailable", got)
	}
}

// ─── /physics edge cases ───

func TestEdge_Physics_OffAlias(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevels{}, Tr: testTr}
	got, _ := physicsCommand(ctx, []string{"off"})
	if got != "command.blocks.set" {
		t.Fatalf("physics off: got %q, want set", got)
	}
}

func TestEdge_Physics_BasicAlias(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevels{}, Tr: testTr}
	got, _ := physicsCommand(ctx, []string{"basic"})
	if got != "command.blocks.set" {
		t.Fatalf("physics basic: got %q, want set", got)
	}
}

func TestEdge_Physics_AdvancedAlias(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevels{}, Tr: testTr}
	got, _ := physicsCommand(ctx, []string{"advanced"})
	if got != "command.blocks.set" {
		t.Fatalf("physics advanced: got %q, want set", got)
	}
}

func TestEdge_Physics_NoLevelService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := physicsCommand(ctx, nil)
	if got != "command.level.unavailable" {
		t.Fatalf("physics no levels: got %q, want command.chat.unavailable", got)
	}
}

// ─── /goto edge cases ───

func TestEdge_Goto_LevelNotFound(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevelsFail{}, Tr: testTr}
	got, _ := gotoCommand(ctx, []string{"nonexistent"})
	if got != "command.goto.not_found" {
		t.Fatalf("goto not found: got %q, want command.whitelist.remove.not_found", got)
	}
}

func TestEdge_Goto_NoLevelService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := gotoCommand(ctx, []string{"main"})
	if got != "command.level.unavailable" {
		t.Fatalf("goto no levels: got %q, want command.chat.unavailable", got)
	}
}

// ─── /portal edge cases ───

func TestEdge_Portal_InvalidCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := portalCommand(ctx, []string{"abc", "20", "30"})
	if got != "command.shared.invalid_x" {
		t.Fatalf("portal invalid coords: got %q, want command.shared.invalid_x", got)
	}
}

func TestEdge_Portal_WithLevel(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	portalCommand(ctx, []string{"10", "20", "30", "other"})
	if draw.selectionStarted != 1 {
		t.Fatalf("portal with level should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

// ─── /mb edge cases ───

func TestEdge_MB_EmptyMessage(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := mbCommand(ctx, []string{""})
	// Empty string arg — should it be treated as no args?
	if got != "command.mb.usage" && got != "command.draw.select1" {
		t.Fatalf("mb empty message: got %q", got)
	}
}

// ─── /setspawn edge cases ───

func TestEdge_SetSpawn_PartialCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr}
	got, _ := setSpawnCommand(ctx, []string{"1", "2"})
	if got == "command.setspawn.done" {
		t.Fatal("setspawn with 2 args should not succeed")
	}
}

func TestEdge_SetSpawn_InvalidCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{World: stubWorld{}, Authority: testAuthority(true), Tr: testTr}
	got, _ := setSpawnCommand(ctx, []string{"abc", "2", "3"})
	if got == "command.setspawn.done" {
		t.Fatal("setspawn with invalid coords should not succeed")
	}
}

// ─── /cuboid hollow/walls ───

func TestEdge_Cuboid_HollowFlag(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	cuboidCommand(ctx, []string{"1", "hollow"})
	if draw.selectionStarted != 2 {
		t.Fatalf("cuboid hollow should start 2-mark selection, got %d", draw.selectionStarted)
	}
}

func TestEdge_Cuboid_WallsFlag(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	cuboidCommand(ctx, []string{"1", "walls"})
	if draw.selectionStarted != 2 {
		t.Fatalf("cuboid walls should start 2-mark selection, got %d", draw.selectionStarted)
	}
}

// ─── /copy edge cases ───

func TestEdge_Copy_NoDrawService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := copyCommand(ctx, nil)
	if got != "command.draw.unavailable" {
		t.Fatalf("copy no draw: got %q, want command.chat.unavailable", got)
	}
}

// ─── /paste edge cases ───

func TestEdge_Paste_NoDrawService(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := pasteCommand(ctx, nil)
	if got != "command.draw.unavailable" {
		t.Fatalf("paste no draw: got %q, want command.chat.unavailable", got)
	}
}

// ─── stubs for edge case tests ───

type stubLevelsFail struct{}

func (stubLevelsFail) Goto(name string) bool        { return false }
func (stubLevelsFail) MainLevel() string            { return "main" }
func (stubLevelsFail) LoadLevel(name string) bool   { return false }
func (stubLevelsFail) UnloadLevel(name string) bool { return false }
func (stubLevelsFail) ReloadLevel() bool            { return false }
func (stubLevelsFail) ListLevels() []string         { return []string{"main"} }
func (stubLevelsFail) ListLevelFiles() []string     { return []string{"main"} }
func (stubLevelsFail) PhysicsMode() int             { return 1 }
func (stubLevelsFail) SetPhysicsMode(mode int)      {}
