package command

import (
	"runtime"
	"strings"
	"time"
)

// seenCommand — /seen <name>
// Shows when a player was last online.
func seenCommand(ctx Context, args []string) (string, bool) {
	if ctx.PlayerDB == nil {
		return ctx.tr("command.info.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.seen.usage"), true
	}
	e := ctx.PlayerDB.Lookup(args[0])
	if e == nil {
		return ctx.tr("command.seen.never", args[0]), true
	}
	age := time.Since(e.LastLogin).Round(time.Second)
	return ctx.tr("command.seen.last", e.Name, age), true
}

// whoisCommand — /whois <name>
// Shows detailed player info from PlayerDB.
func whoisCommand(ctx Context, args []string) (string, bool) {
	if ctx.PlayerDB == nil {
		return ctx.tr("command.info.unavailable"), true
	}
	if len(args) != 1 {
		return ctx.tr("command.whois.usage"), true
	}
	e := ctx.PlayerDB.Lookup(args[0])
	if e == nil {
		return ctx.tr("command.seen.never", args[0]), true
	}
	playtime := e.TotalTime.Round(time.Minute)
	var sb strings.Builder
	sb.WriteString(ctx.tr("command.whois.name", e.Name))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.first_login", e.FirstLogin.Format("2006-01-02 15:04")))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.last_login", e.LastLogin.Format("2006-01-02 15:04")))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.playtime", playtime))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.logins", e.LoginCount))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.ip", e.IP))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.deaths", e.Deaths))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.whois.blocks", e.BlocksPlaced, e.BlocksDeleted))
	return sb.String(), true
}

// blocksCommand — /blocks [name]
// Shows block placement/deletion stats. Defaults to self.
func blocksCommand(ctx Context, args []string) (string, bool) {
	if ctx.PlayerDB == nil {
		return ctx.tr("command.info.unavailable"), true
	}
	name := ctx.Username
	if len(args) == 1 {
		name = args[0]
	}
	e := ctx.PlayerDB.Lookup(name)
	if e == nil {
		return ctx.tr("command.seen.never", name), true
	}
	return ctx.tr("command.blocks.stats", name, e.BlocksPlaced, e.BlocksDeleted), true
}

// mapinfoCommand — /mapinfo
// Shows info about the current level.
func mapinfoCommand(ctx Context, args []string) (string, bool) {
	if ctx.Levels == nil {
		return ctx.tr("command.level.unavailable"), true
	}
	levels := ctx.Levels.ListLevels()
	count := 0
	if levels != nil {
		count = len(levels)
	}
	var sb strings.Builder
	sb.WriteString(ctx.tr("command.mapinfo.levels", count))
	if count > 0 && len(levels) > 0 {
		sb.WriteString(" &7(")
		sb.WriteString(strings.Join(levels, ", "))
		sb.WriteString(")")
	}
	sb.WriteString("\n")
	if ctx.BlockDB != nil {
		sb.WriteString(ctx.tr("command.mapinfo.blockdb", ctx.BlockDB.Count()))
		sb.WriteString("\n")
	}
	if ctx.LevelEnv != nil {
		weather := "sunny"
		switch ctx.LevelEnv.Weather() {
		case 1:
			weather = "raining"
		case 2:
			weather = "snowing"
		}
		sb.WriteString(ctx.tr("command.mapinfo.weather", weather))
	}
	return sb.String(), true
}

// serverinfoCommand — /serverinfo
// Shows server-wide info.
func serverinfoCommand(ctx Context, args []string) (string, bool) {
	if ctx.ServerInfo == nil {
		return ctx.tr("command.info.unavailable"), true
	}
	si := ctx.ServerInfo
	uptime := si.Uptime().Round(time.Second)
	var sb strings.Builder
	sb.WriteString(ctx.tr("command.serverinfo.name", si.ServerName()))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.serverinfo.players", si.OnlineCount(), si.MaxPlayers()))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.serverinfo.levels", si.LevelCount()))
	sb.WriteString("\n")
	sb.WriteString(ctx.tr("command.serverinfo.uptime", uptime))
	sb.WriteString("\n")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	sb.WriteString(ctx.tr("command.serverinfo.memory", m.Alloc/1024/1024))
	return sb.String(), true
}

// timeCommand — /time
// Shows the current server time.
func timeCommand(ctx Context, args []string) (string, bool) {
	now := time.Now().Format("2006-01-02 15:04:05")
	return ctx.tr("command.time.current", now), true
}

// rulesCommand — /rules
// Shows server rules (from a configurable message).
func rulesCommand(ctx Context, args []string) (string, bool) {
	return ctx.tr("command.rules.text"), true
}
