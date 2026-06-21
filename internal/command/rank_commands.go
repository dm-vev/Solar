// rank_commands.go implements rank management commands.
//
// /setrank <name> <rank>  — set a player's rank (operator)
// /rankinfo <name>        — show a player's current rank
// /viewranks              — list all available ranks

package command

import (
	"strconv"
	"strings"
)

// setRankCommand — /setrank <name> <rank>
// Safety checks (matching MCGalaxy):
//   - Cannot rank yourself
//   - Cannot set rank to >= your own rank
//   - Cannot set to banned rank (use /ban instead)
func setRankCommand(ctx Context, args []string) (string, bool) {
	if ctx.Ranks == nil {
		return ctx.tr("command.rank.unavailable"), true
	}
	if len(args) != 2 {
		return ctx.tr("command.setrank.usage"), true
	}
	playerName := args[0]
	rankName := strings.ToLower(args[1])

	// Cannot rank yourself.
	if strings.EqualFold(playerName, ctx.Username) {
		return ctx.tr("command.setrank.self"), true
	}

	rank := ctx.Ranks.Get(rankName)
	if rank == nil {
		return ctx.tr("command.setrank.not_found", rankName), true
	}

	// Cannot set to banned rank.
	if rank.Permission < 0 {
		return ctx.tr("command.setrank.banned"), true
	}

	// Cannot set to rank >= your own.
	myRank := 0
	if ctx.RankLevel != nil {
		myRank = ctx.RankLevel()
	}
	if rank.Permission >= myRank {
		return ctx.tr("command.setrank.too_high"), true
	}

	// Cannot change rank of someone >= your own rank (matching MCGalaxy CheckRank).
	targetRank := ctx.Ranks.GetPlayerRank(playerName)
	if targetRank >= myRank {
		return ctx.tr("command.setrank.target_too_high"), true
	}

	if !ctx.Ranks.SetPlayerRank(playerName, rank.Permission) {
		return ctx.tr("command.setrank.failed", playerName), true
	}
	return ctx.tr("command.setrank.done", playerName, rank.Name), true
}

// rankInfoCommand — /rankinfo [name]
func rankInfoCommand(ctx Context, args []string) (string, bool) {
	if ctx.Ranks == nil {
		return ctx.tr("command.rank.unavailable"), true
	}
	name := ctx.Username
	if len(args) == 1 {
		name = args[0]
	}
	perm := ctx.Ranks.GetPlayerRank(name)
	rank := ctx.Ranks.GetByPerm(perm)
	if rank == nil {
		return ctx.tr("command.rankinfo.none", name), true
	}
	return ctx.tr("command.rankinfo.result", name, rank.Color+rank.Name, rank.Permission), true
}

// viewRanksCommand — /viewranks
func viewRanksCommand(ctx Context, args []string) (string, bool) {
	if ctx.Ranks == nil {
		return ctx.tr("command.rank.unavailable"), true
	}
	all := ctx.Ranks.All()
	var parts []string
	for _, r := range all {
		parts = append(parts, r.Color+r.Name+"("+strconv.Itoa(r.Permission)+")")
	}
	return "&aRanks: &7" + strings.Join(parts, " "), true
}
