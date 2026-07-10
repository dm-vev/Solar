package command

import (
	"testing"

	"github.com/solar-mc/solar/internal/ranks"
	"github.com/solar-mc/solar/plugin/playerdb"
)

// ─── Admin commands: /tp ───

// MCGalaxy: /tp <x> <y> <z> [yaw] [pitch] — teleports to coordinates.
func TestMCG_TP_ValidCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Tr:        testTr,
	}
	got, ok := teleportCommand(ctx, []string{"10", "20", "30"})
	if !ok || got != "command.teleport.done" {
		t.Fatalf("tp 10 20 30: got %q ok=%v", got, ok)
	}
}

func TestMCG_TP_WithYawPitch(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Tr:        testTr,
	}
	got, ok := teleportCommand(ctx, []string{"10", "20", "30", "45", "90"})
	if !ok || got != "command.teleport.done" {
		t.Fatalf("tp with yaw/pitch: got %q ok=%v", got, ok)
	}
}

func TestMCG_TP_WrongArgCount(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := teleportCommand(ctx, []string{"1", "2"})
	if got != "command.teleport.unavailable" {
		t.Fatalf("tp with 2 args: got %q, want unavailable", got)
	}
}

func TestMCG_TP_InvalidCoords(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := teleportCommand(ctx, []string{"abc", "20", "30"})
	if got != "command.teleport.unavailable" {
		t.Fatalf("tp invalid x: got %q, want unavailable", got)
	}
}

// ─── Admin commands: /kick ───

// MCGalaxy: /kick <name> [reason] — disconnects a player.
func TestMCG_KickWithReason(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := kickCommand(ctx, []string{"alice", "griefing"})
	if got != "command.kick.done_reason" {
		t.Fatalf("kick with reason: got %q, want done_reason", got)
	}
}

func TestMCG_KickWithoutReason(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := kickCommand(ctx, []string{"alice"})
	if got != "command.kick.done" {
		t.Fatalf("kick without reason: got %q, want done", got)
	}
}

func TestMCG_KickPlayerNotFound(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationFail{}, Tr: testTr}
	got, _ := kickCommand(ctx, []string{"nobody"})
	if got != "command.shared.player_not_found" {
		t.Fatalf("kick not found: got %q, want player_not_found", got)
	}
}

// ─── Admin commands: /ban ───

func TestMCG_BanWithReason(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{banned: true}, Tr: testTr}
	got, _ := banCommand(ctx, []string{"griefer", "spam"})
	if got != "command.ban.done_reason" {
		t.Fatalf("ban with reason: got %q, want done_reason", got)
	}
}

func TestMCG_BanWithoutReason(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{banned: true}, Tr: testTr}
	got, _ := banCommand(ctx, []string{"griefer"})
	if got != "command.ban.done" {
		t.Fatalf("ban without reason: got %q, want done", got)
	}
}

func TestMCG_BanFailed(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationFail{}, Tr: testTr}
	got, _ := banCommand(ctx, []string{"griefer"})
	if got != "command.ban.failed" {
		t.Fatalf("ban failed: got %q, want failed", got)
	}
}

// ─── Admin commands: /unban ───

func TestMCG_UnbanSuccess(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := unbanCommand(ctx, []string{"alice"})
	if got != "command.unban.done" {
		t.Fatalf("unban: got %q, want done", got)
	}
}

func TestMCG_UnbanNotBanned(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationFail{}, Tr: testTr}
	got, _ := unbanCommand(ctx, []string{"alice"})
	if got != "command.unban.not_banned" {
		t.Fatalf("unban not banned: got %q, want not_banned", got)
	}
}

// ─── Admin commands: /save ───

func TestMCG_SaveSuccess(t *testing.T) {
	t.Parallel()
	ctx := Context{Persistence: stubPersistence{}, Tr: testTr}
	got, _ := saveCommand(ctx, nil)
	if got != "command.save.queued" {
		t.Fatalf("save: got %q, want queued", got)
	}
}

func TestMCG_SaveFailed(t *testing.T) {
	t.Parallel()
	ctx := Context{Persistence: failingPersistence{}, Tr: testTr}
	got, _ := saveCommand(ctx, nil)
	if got != "command.save.failed" {
		t.Fatalf("save failed: got %q, want failed", got)
	}
}

// ─── Admin commands: /setspawn ───

func TestMCG_SetSpawnDefault(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Position:  Position{X: 10, Y: 20, Z: 30},
		Yaw:       45,
		Pitch:     90,
		Tr:        testTr,
	}
	got, _ := setSpawnCommand(ctx, nil)
	if got != "command.setspawn.done" {
		t.Fatalf("setspawn default: got %q, want done", got)
	}
}

func TestMCG_SetSpawnExplicit(t *testing.T) {
	t.Parallel()
	ctx := Context{
		World:     stubWorld{},
		Authority: testAuthority(true),
		Tr:        testTr,
	}
	got, _ := setSpawnCommand(ctx, []string{"1", "2", "3", "4", "5"})
	if got != "command.setspawn.done" {
		t.Fatalf("setspawn explicit: got %q, want done", got)
	}
}

// ─── Moderation: /mute, /unmute, /freeze, /unfreeze ───

func TestMCG_Mute(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := muteCommand(ctx, []string{"alice"})
	if got != "command.mute.done" {
		t.Fatalf("mute: got %q, want done", got)
	}
}

func TestMCG_MuteNoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := muteCommand(ctx, nil)
	if got != "command.moderation.unavailable" {
		t.Fatalf("mute no args: got %q, want unavailable", got)
	}
}

func TestMCG_Unmute(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := unmuteCommand(ctx, []string{"alice"})
	if got != "command.unmute.done" {
		t.Fatalf("unmute: got %q, want done", got)
	}
}

func TestMCG_Freeze(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := freezeCommand(ctx, []string{"alice"})
	if got != "command.freeze.done" {
		t.Fatalf("freeze: got %q, want done", got)
	}
}

func TestMCG_Unfreeze(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Tr: testTr}
	got, _ := unfreezeCommand(ctx, []string{"alice"})
	if got != "command.unfreeze.done" {
		t.Fatalf("unfreeze: got %q, want done", got)
	}
}

// ─── Moderation: /afk, /hide ───

func TestMCG_AFKToggle(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Username: "alice", Tr: testTr}
	got, _ := afkCommand(ctx, nil)
	if got != "command.afk.on" {
		t.Fatalf("afk toggle: got %q, want on", got)
	}
}

func TestMCG_AFKOff(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModerationOff{}, Username: "alice", Tr: testTr}
	got, _ := afkCommand(ctx, nil)
	if got != "command.afk.off" {
		t.Fatalf("afk off: got %q, want off", got)
	}
}

func TestMCG_HideToggle(t *testing.T) {
	t.Parallel()
	ctx := Context{Moderation: stubModeration{}, Username: "alice", Tr: testTr}
	got, _ := hideCommand(ctx, nil)
	if got != "command.hide.on" {
		t.Fatalf("hide toggle: got %q, want on", got)
	}
}

// ─── Info: /where, /players, /seen, /whois ───

func TestMCG_Where(t *testing.T) {
	t.Parallel()
	ctx := Context{Username: "alice", Position: Position{X: 10, Y: 20, Z: 30}, Tr: testTr}
	got, ok := whereCommand(ctx, nil)
	if !ok || got != "command.where.position" {
		t.Fatalf("where: got %q ok=%v", got, ok)
	}
}

func TestMCG_PlayersList(t *testing.T) {
	t.Parallel()
	ctx := Context{Players: stubDirectory{names: []string{"alice", "bob"}}, Tr: testTr}
	got, _ := playersCommand(ctx, nil)
	if got != "command.players.list" {
		t.Fatalf("players: got %q, want list", got)
	}
}

func TestMCG_PlayersEmpty(t *testing.T) {
	t.Parallel()
	ctx := Context{Players: stubDirectory{}, Tr: testTr}
	got, _ := playersCommand(ctx, nil)
	if got != "command.players.none" {
		t.Fatalf("players empty: got %q, want none", got)
	}
}

func TestMCG_SeenFound(t *testing.T) {
	t.Parallel()
	ctx := Context{PlayerDB: stubPlayerDB{}, Tr: testTr}
	got, _ := seenCommand(ctx, []string{"alice"})
	if got != "command.seen.last" {
		t.Fatalf("seen found: got %q, want last", got)
	}
}

func TestMCG_SeenNever(t *testing.T) {
	t.Parallel()
	ctx := Context{PlayerDB: stubPlayerDB{}, Tr: testTr}
	got, _ := seenCommand(ctx, []string{"nobody"})
	if got != "command.seen.never" {
		t.Fatalf("seen never: got %q, want never", got)
	}
}

func TestMCG_WhoisFound(t *testing.T) {
	t.Parallel()
	ctx := Context{PlayerDB: stubPlayerDB{}, Tr: testTr}
	got, ok := whoisCommand(ctx, []string{"alice"})
	if !ok {
		t.Fatal("whois should be handled")
	}
	// whois returns a multiline string — just verify it starts with the name key.
	if !contains(got, "command.whois.name") {
		t.Fatalf("whois: got %q, want it to contain 'command.whois.name'", got)
	}
}

func TestMCG_WhoisNotFound(t *testing.T) {
	t.Parallel()
	ctx := Context{PlayerDB: stubPlayerDB{}, Tr: testTr}
	got, _ := whoisCommand(ctx, []string{"nobody"})
	if got != "command.seen.never" {
		t.Fatalf("whois not found: got %q, want never", got)
	}
}

// ─── Info: /help filters by rank ───

// MCGalaxy: /help only shows commands the player can use.
func TestMCG_HelpFiltersByRank(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	handler := helpCommand(registry)

	// Use a formatter so we can check the command names in the output.
	formatTr := func(key string, args ...any) string {
		if key == "command.help.list" && len(args) > 0 {
			if s, ok := args[0].(string); ok {
				return s
			}
		}
		return key
	}

	// Guest should not see /tp (rank 80).
	guestCtx := Context{Tr: formatTr, RankLevel: func() int { return 0 }}
	got, _ := handler(guestCtx, nil)
	if contains(got, "/tp,") || contains(got, "/tp ") {
		t.Fatal("guest /help should not show /tp")
	}

	// Operator should see /tp.
	opCtx := Context{Tr: formatTr, RankLevel: func() int { return 80 }}
	got, _ = handler(opCtx, nil)
	if !contains(got, "/tp") {
		t.Fatalf("operator /help should show /tp: got %s", got)
	}
}

// ─── Info: /time, /rules ───

func TestMCG_Time(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, ok := timeCommand(ctx, nil)
	if !ok {
		t.Fatal("time should be handled")
	}
	// Time format is "2006-01-02 15:04:05" — just verify it's not empty.
	if got == "" {
		t.Fatal("time should return non-empty string")
	}
}

func TestMCG_Rules(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, ok := rulesCommand(ctx, nil)
	if !ok || got != "command.rules.text" {
		t.Fatalf("rules: got %q ok=%v", got, ok)
	}
}

// ─── Teleport/chat: /spawn, /back, /tpa, /me, /whisper, /ignore ───

func TestMCG_Spawn(t *testing.T) {
	t.Parallel()
	ctx := Context{Teleport: stubTeleport{}, World: stubWorld{}, Tr: testTr}
	got, _ := spawnCommand(ctx, nil)
	if got != "command.spawn.done" {
		t.Fatalf("spawn: got %q, want done", got)
	}
}

func TestMCG_BackNoLastPos(t *testing.T) {
	t.Parallel()
	ctx := Context{Teleport: stubTeleport{}, Tr: testTr}
	got, _ := backCommand(ctx, nil)
	if got != "command.back.none" {
		t.Fatalf("back no last pos: got %q, want none", got)
	}
}

func TestMCG_TPA(t *testing.T) {
	t.Parallel()
	ctx := Context{Teleport: stubTeleport{}, Tr: testTr}
	got, _ := tpaCommand(ctx, []string{"bob"})
	if got != "command.tpa.sent" {
		t.Fatalf("tpa: got %q, want sent", got)
	}
}

func TestMCG_TPAPlayerNotFound(t *testing.T) {
	t.Parallel()
	ctx := Context{Teleport: stubTeleportFail{}, Tr: testTr}
	got, _ := tpaCommand(ctx, []string{"nobody"})
	if got != "command.moderation.player_not_found" {
		t.Fatalf("tpa not found: got %q, want player_not_found", got)
	}
}

func TestMCG_Me(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	_, ok := meCommand(ctx, []string{"waves"})
	if ok {
		t.Fatal("me should return handled=false (broadcast by chat service)")
	}
}

func TestMCG_Whisper(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	_, ok := whisperCommand(ctx, []string{"bob", "hello"})
	if ok {
		t.Fatal("whisper should return handled=false (sent by chat service)")
	}
}

func TestMCG_Ignore(t *testing.T) {
	t.Parallel()
	ctx := Context{Chat: stubChat{}, Tr: testTr}
	got, _ := ignoreCommand(ctx, []string{"bob"})
	if got != "command.ignore.on" {
		t.Fatalf("ignore: got %q, want on", got)
	}
}

// ─── Ranks: /setrank ───

// MCGalaxy: /setrank safety checks — cannot rank yourself, cannot set to banned,
// cannot set to rank >= your own, cannot change rank of someone >= your own.
func TestMCG_SetRankCannotRankSelf(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 },
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, []string{"alice", "operator"})
	if got != "command.setrank.self" {
		t.Fatalf("setrank self: got %q, want self", got)
	}
}

func TestMCG_SetRankCannotSetBanned(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 },
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, []string{"bob", "banned"})
	if got != "command.setrank.banned" {
		t.Fatalf("setrank banned: got %q, want banned", got)
	}
}

func TestMCG_SetRankCannotSetTooHigh(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 50 }, // advbuilder
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, []string{"bob", "operator"})
	if got != "command.setrank.too_high" {
		t.Fatalf("setrank too high: got %q, want too_high", got)
	}
}

func TestMCG_SetRankSuccess(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	rr.SetPlayerDB(&stubRankPlayerDB{entries: map[string]*playerdb.PlayerEntry{
		"bob": {Name: "bob", Data: map[string]string{}},
	}})
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 100 }, // admin can set operator
		Tr:        testTr,
	}
	got, _ := setRankCommand(ctx, []string{"bob", "operator"})
	if got != "command.setrank.done" {
		t.Fatalf("setrank success: got %q, want done", got)
	}
}

// ─── Ranks: /rankinfo, /viewranks ───

func TestMCG_RankInfoSelf(t *testing.T) {
	t.Parallel()
	rr := ranks.NewRegistry()
	ctx := Context{
		Username:  "alice",
		Ranks:     stubRanks{rr: rr},
		RankLevel: func() int { return 80 },
		Tr:        testTr,
	}
	got, _ := rankInfoCommand(ctx, nil)
	if got != "command.rankinfo.result" {
		t.Fatalf("rankinfo: got %q, want result", got)
	}
}

// ─── Drawing: /cuboid, /line, /sphere, /fill ───

// MCGalaxy: drawing commands use mark selection. StartSelection is called
// with 2 marks for cuboid/line, 1 for sphere/fill.
func TestMCG_CuboidStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	cuboidCommand(ctx, []string{"1"})
	if draw.selectionStarted != 2 {
		t.Fatalf("cuboid should start 2-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_LineStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	lineCommand(ctx, []string{"1"})
	if draw.selectionStarted != 2 {
		t.Fatalf("line should start 2-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_SphereStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	sphereCommand(ctx, []string{"1"})
	if draw.selectionStarted != 1 {
		t.Fatalf("sphere should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_FillStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	fillCommand(ctx, []string{"1"})
	if draw.selectionStarted != 1 {
		t.Fatalf("fill should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_CuboidNoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := cuboidCommand(ctx, nil)
	if got != "command.cuboid.usage" {
		t.Fatalf("cuboid no args: got %q, want usage", got)
	}
}

func TestMCG_SphereDefaultRadius(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	sphereCommand(ctx, []string{"1"})
	if draw.selectionStarted != 1 {
		t.Fatalf("sphere should start selection even with default radius")
	}
}

// ─── Drawing: /copy, /paste ───

func TestMCG_CopyStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	copyCommand(ctx, nil)
	if draw.selectionStarted != 2 {
		t.Fatalf("copy should start 2-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_PasteNoClipboard(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr} // stubDraw.HasClipboard() = false
	got, _ := pasteCommand(ctx, nil)
	if got != "command.paste.empty" {
		t.Fatalf("paste no clipboard: got %q, want empty", got)
	}
}

// ─── Undo: /undo, /redo ───

func TestMCG_UndoEmpty(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr} // stubDraw.Undo() = nil
	got, _ := undoCommand(ctx, nil)
	if got != "command.undo.none" {
		t.Fatalf("undo empty: got %q, want none", got)
	}
}

func TestMCG_RedoEmpty(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr} // stubDraw.Redo() = nil
	got, _ := redoCommand(ctx, nil)
	if got != "command.redo.none" {
		t.Fatalf("redo empty: got %q, want none", got)
	}
}

// ─── Registry: permission enforcement ───

// MCGalaxy: commands above player's rank are denied.
func TestMCG_RegistryRejectsByRank(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{Tr: testTr, RankLevel: func() int { return 0 }}
	got, handled := registry.Execute(ctx, "/tp 1 2 3")
	if !handled || got != "command.shared.permission_denied" {
		t.Fatalf("guest /tp: got %q handled=%v, want permission_denied", got, handled)
	}
}

// MCGalaxy: admin commands require operator flag even if rank is sufficient.
func TestMCG_RegistryRejectsAdminWithoutOpFlag(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{
		Tr:        testTr,
		RankLevel: func() int { return 80 }, // operator rank
		Authority: testAuthority(false),     // but NOT an operator
	}
	got, handled := registry.Execute(ctx, "/kick bob")
	if !handled || got != "command.shared.permission_denied" {
		t.Fatalf("non-op /kick: got %q handled=%v, want permission_denied", got, handled)
	}
}

// MCGalaxy: guest commands work without any special permissions.
func TestMCG_RegistryGuestCommandsWork(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	ctx := Context{Tr: testTr, RankLevel: func() int { return 0 }}
	got, handled := registry.Execute(ctx, "/where")
	if !handled || got != "command.where.position" {
		t.Fatalf("guest /where: got %q handled=%v", got, handled)
	}
}

// ─── Special: /mb, /portal, /door ───

func TestMCG_MBStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	mbCommand(ctx, []string{"hello"})
	if draw.selectionStarted != 1 {
		t.Fatalf("mb should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_MBNoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := mbCommand(ctx, nil)
	if got != "command.mb.usage" {
		t.Fatalf("mb no args: got %q, want usage", got)
	}
}

func TestMCG_PortalStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	portalCommand(ctx, []string{"10", "20", "30"})
	if draw.selectionStarted != 1 {
		t.Fatalf("portal should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

func TestMCG_PortalNotEnoughArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Draw: stubDraw{}, Tr: testTr}
	got, _ := portalCommand(ctx, []string{"10", "20"})
	if got != "command.portal.usage" {
		t.Fatalf("portal 2 args: got %q, want usage", got)
	}
}

func TestMCG_DoorStartsSelection(t *testing.T) {
	t.Parallel()
	draw := &trackingDraw{}
	ctx := Context{Draw: draw, Tr: testTr}
	doorCommand(ctx, nil)
	if draw.selectionStarted != 1 {
		t.Fatalf("door should start 1-mark selection, got %d", draw.selectionStarted)
	}
}

// ─── World: /goto, /main, /levels, /physics ───

func TestMCG_GotoNoArgs(t *testing.T) {
	t.Parallel()
	ctx := Context{Tr: testTr}
	got, _ := gotoCommand(ctx, nil)
	if got != "command.level.unavailable" {
		t.Fatalf("goto no args: got %q, want unavailable", got)
	}
}

func TestMCG_PhysicsShowMode(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevels{}, Tr: testTr}
	got, _ := physicsCommand(ctx, nil)
	if got != "command.blocks.current" {
		t.Fatalf("physics show: got %q, want current", got)
	}
}

func TestMCG_PhysicsSetMode(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevels{}, Tr: testTr}
	got, _ := physicsCommand(ctx, []string{"2"})
	if got != "command.blocks.set" {
		t.Fatalf("physics set 2: got %q, want set", got)
	}
}

func TestMCG_PhysicsInvalidMode(t *testing.T) {
	t.Parallel()
	ctx := Context{Levels: stubLevels{}, Tr: testTr}
	got, _ := physicsCommand(ctx, []string{"99"})
	if got != "command.blocks.usage" {
		t.Fatalf("physics invalid: got %q, want usage", got)
	}
}

// ─── stubs for testing ───

type stubModerationFail struct{}

func (stubModerationFail) KickPlayer(_, _ string) bool      { return false }
func (stubModerationFail) BanPlayer(_, _ string) bool       { return false }
func (stubModerationFail) UnbanPlayer(_ string) bool        { return false }
func (stubModerationFail) WhitelistEnabled() bool           { return false }
func (stubModerationFail) SetWhitelistEnabled(_ bool) bool  { return false }
func (stubModerationFail) WhitelistAdd(_ string) bool       { return false }
func (stubModerationFail) WhitelistRemove(_ string) bool    { return false }
func (stubModerationFail) MutePlayer(_ string) bool         { return false }
func (stubModerationFail) UnmutePlayer(_ string) bool       { return false }
func (stubModerationFail) FreezePlayer(_ string) bool       { return false }
func (stubModerationFail) UnfreezePlayer(_ string) bool     { return false }
func (stubModerationFail) ToggleAFK(_ string) (bool, bool)  { return false, true }
func (stubModerationFail) ToggleHide(_ string) (bool, bool) { return false, true }

type stubModerationOff struct{}

func (stubModerationOff) KickPlayer(_, _ string) bool      { return true }
func (stubModerationOff) BanPlayer(_, _ string) bool       { return true }
func (stubModerationOff) UnbanPlayer(_ string) bool        { return true }
func (stubModerationOff) WhitelistEnabled() bool           { return false }
func (stubModerationOff) SetWhitelistEnabled(_ bool) bool  { return true }
func (stubModerationOff) WhitelistAdd(_ string) bool       { return true }
func (stubModerationOff) WhitelistRemove(_ string) bool    { return true }
func (stubModerationOff) MutePlayer(_ string) bool         { return true }
func (stubModerationOff) UnmutePlayer(_ string) bool       { return true }
func (stubModerationOff) FreezePlayer(_ string) bool       { return true }
func (stubModerationOff) UnfreezePlayer(_ string) bool     { return true }
func (stubModerationOff) ToggleAFK(_ string) (bool, bool)  { return false, true }
func (stubModerationOff) ToggleHide(_ string) (bool, bool) { return false, true }

type stubTeleportFail struct{}

func (stubTeleportFail) SpawnPoint() (int, int, int, byte, byte) { return 0, 0, 0, 0, 0 }
func (stubTeleportFail) TeleportToPlayer(name string) bool       { return false }
func (stubTeleportFail) RequestTeleport(name string) (TPAStatus, string) {
	return TPAPlayerNotFound, name
}
func (stubTeleportFail) RespondTeleport(accept bool) (TPAStatus, string) {
	return TPANoPending, ""
}
func (stubTeleportFail) SummonPlayer(name string) bool { return false }
func (stubTeleportFail) Back() bool                    { return false }

type stubRanks struct {
	rr *ranks.Registry
}

func (s stubRanks) GetPlayerRank(name string) int {
	return s.rr.GetPlayerRank(name)
}
func (s stubRanks) Get(name string) *RankInfo {
	r := s.rr.Get(name)
	if r == nil {
		return nil
	}
	return &RankInfo{Name: r.Name, Permission: r.Permission, Color: r.Color, DrawLimit: r.DrawLimit}
}
func (s stubRanks) GetByPerm(perm int) *RankInfo {
	r := s.rr.GetByPerm(perm)
	if r == nil {
		return nil
	}
	return &RankInfo{Name: r.Name, Permission: r.Permission, Color: r.Color, DrawLimit: r.DrawLimit}
}
func (s stubRanks) All() []RankInfo {
	ranks := s.rr.All()
	out := make([]RankInfo, len(ranks))
	for i, r := range ranks {
		out[i] = RankInfo{Name: r.Name, Permission: r.Permission, Color: r.Color, DrawLimit: r.DrawLimit}
	}
	return out
}
func (s stubRanks) SetPlayerRank(name string, perm int) bool {
	return s.rr.SetPlayerRank(name, perm)
}

type stubLevels struct{}

func (stubLevels) Goto(name string) bool        { return true }
func (stubLevels) MainLevel() string            { return "main" }
func (stubLevels) LoadLevel(name string) bool   { return true }
func (stubLevels) UnloadLevel(name string) bool { return true }
func (stubLevels) ReloadLevel() bool            { return true }
func (stubLevels) ListLevels() []string         { return []string{"main"} }
func (stubLevels) ListLevelFiles() []string     { return []string{"main"} }
func (stubLevels) PhysicsMode() int             { return 1 }
func (stubLevels) SetPhysicsMode(mode int)      {}

type stubRankPlayerDB struct {
	entries map[string]*playerdb.PlayerEntry
}

func (d *stubRankPlayerDB) Get(name string) *playerdb.PlayerEntry { return d.entries[name] }
func (d *stubRankPlayerDB) Save(entry *playerdb.PlayerEntry)      { d.entries[entry.Name] = entry }
func (d *stubRankPlayerDB) Delete(name string) bool {
	if _, ok := d.entries[name]; ok {
		delete(d.entries, name)
		return true
	}
	return false
}
func (d *stubRankPlayerDB) List() []*playerdb.PlayerEntry {
	out := make([]*playerdb.PlayerEntry, 0, len(d.entries))
	for _, e := range d.entries {
		out = append(out, e)
	}
	return out
}
func (d *stubRankPlayerDB) Search(prefix string) []*playerdb.PlayerEntry { return nil }
func (d *stubRankPlayerDB) Count() int                                   { return len(d.entries) }
func (d *stubRankPlayerDB) Flush() error                                 { return nil }

type trackingDraw struct {
	selectionStarted int
}

func (d *trackingDraw) StartSelection(markCount int, callback func([][3]int)) bool {
	d.selectionStarted = markCount
	return true
}
func (d *trackingDraw) GetBlockAt(x, y, z int) (byte, bool)                       { return 0, true }
func (d *trackingDraw) PlaceBlock(x, y, z int, block byte) bool                   { return true }
func (d *trackingDraw) LevelDims() (int, int, int)                                { return 128, 64, 128 }
func (d *trackingDraw) CopyRegion(min, max [3]int) bool                           { return true }
func (d *trackingDraw) HasClipboard() bool                                        { return true }
func (d *trackingDraw) PasteAt(origin [3]int, pasteAir bool) int                  { return 0 }
func (d *trackingDraw) SetSpecialBlock(x, y, z int, entry SpecialBlockEntry) bool { return true }
func (d *trackingDraw) BeginBatch()                                               {}
func (d *trackingDraw) RecordChange(x, y, z int, old, new byte)                   {}
func (d *trackingDraw) CommitBatch()                                              {}
func (d *trackingDraw) Undo() []UndoChange                                        { return nil }
func (d *trackingDraw) Redo() []UndoChange                                        { return nil }
func (d *trackingDraw) DrawLimit() int                                            { return 4096 }
func (d *trackingDraw) CanPlace(block byte) bool                                  { return true }
func (d *trackingDraw) CanDelete(block byte) bool                                 { return true }

// ─── MCGalaxy conformance: discrepancies ───

func TestMCGalaxy_TPARequiresAcceptance(t *testing.T) {
	service := &trackingTPA{}
	ctx := Context{Teleport: service, Tr: testTr}
	got, _ := tpaCommand(ctx, []string{"bob"})
	if got != "command.tpa.sent" || service.pending != "bob" {
		t.Fatalf("request: got %q pending=%q", got, service.pending)
	}
	got, _ = tpaCommand(ctx, []string{"accept"})
	if got != "command.tpa.accepted" || service.pending != "" {
		t.Fatalf("accept: got %q pending=%q", got, service.pending)
	}
	service.pending = "alice"
	got, _ = NewRegistry().Execute(ctx, "/tpdeny")
	if got != "command.tpa.denied" || service.pending != "" {
		t.Fatalf("tpdeny alias: got %q pending=%q", got, service.pending)
	}
}

type trackingTPA struct{ pending string }

func (*trackingTPA) SpawnPoint() (int, int, int, byte, byte) { return 0, 0, 0, 0, 0 }
func (t *trackingTPA) RequestTeleport(name string) (TPAStatus, string) {
	t.pending = name
	return TPARequestSent, name
}
func (t *trackingTPA) RespondTeleport(accept bool) (TPAStatus, string) {
	name := t.pending
	if name == "" {
		return TPANoPending, ""
	}
	t.pending = ""
	if accept {
		return TPAAccepted, name
	}
	return TPADenied, name
}
func (*trackingTPA) SummonPlayer(string) bool { return true }
func (*trackingTPA) Back() bool               { return false }

func TestMCGalaxy_ViewRanksNoArgsFormat(t *testing.T) {
	rr := ranks.NewRegistry()
	var list string
	ctx := Context{
		Ranks: stubRanks{rr: rr},
		Tr: func(key string, args ...any) string {
			if key == "command.viewranks.available" {
				list = args[0].(string)
			}
			return key
		},
	}
	got, _ := viewRanksCommand(ctx, nil)
	if got != "command.viewranks.available" {
		t.Fatalf("viewranks key: got %q", got)
	}
	want := "&8banned, &7guest, &2builder, &3advbuilder, &coperator, &eadmin, &4owner"
	if list != want {
		t.Fatalf("viewranks list: got %q, want %q", list, want)
	}
}
