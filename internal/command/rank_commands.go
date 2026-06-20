// rank_commands.go implements rank management commands.
//
// /setrank <name> <rank>  — set a player's rank (operator)
// /rankinfo <name>        — show a player's current rank
// /viewranks              — list all available ranks

package command

import (
	"strings"
)

// setRankCommand — /setrank <name> <rank>
func setRankCommand(ctx Context, args []string) (string, bool) {
	if ctx.Ranks == nil {
		return ctx.tr("command.rank.unavailable"), true
	}
	if len(args) != 2 {
		return ctx.tr("command.setrank.usage"), true
	}
	playerName := args[0]
	rankName := strings.ToLower(args[1])

	rank := ctx.Ranks.Get(rankName)
	if rank == nil {
		return ctx.tr("command.setrank.not_found", rankName), true
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
		parts = append(parts, r.Color+r.Name+"("+itoa(r.Permission)+")")
	}
	return "&aRanks: &7" + strings.Join(parts, " "), true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
