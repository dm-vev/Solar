package command

var adminCommands = map[string]struct{}{
	"tp":        {},
	"setspawn":  {},
	"save":      {},
	"kick":      {},
	"ban":       {},
	"unban":     {},
	"whitelist": {},
	"newlvl":    {},
}

func requiresAdmin(name string) bool {
	_, ok := adminCommands[name]
	return ok
}
