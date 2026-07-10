//go:build plugin

// Package main provides MCGalaxy economy commands as an independent Solar plugin.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/solar-mc/solar/plugin"
)

const maxMoney = 16_777_215

type economyPlugin struct {
	server     plugin.Server
	registered []string
	state      economyState
}

type economyState struct {
	mu       sync.Mutex
	path     string
	Enabled  bool           `json:"enabled"`
	Currency string         `json:"currency"`
	Accounts map[string]int `json:"accounts"`
}

func (p *economyPlugin) Name() string { return "mcgalaxy-economy" }
func (p *economyPlugin) Init() error  { return nil }

func (p *economyPlugin) Enable(server plugin.Server) error {
	dir, err := server.PluginDataDir("mcgalaxy-economy")
	if err != nil {
		return err
	}
	p.server = server
	if err := p.state.load(filepath.Join(dir, "economy.json")); err != nil {
		return err
	}
	commands := []plugin.CommandSpec{
		{Name: "balance", Aliases: []string{"money"}, Help: "/balance [player] - shows an account balance", Handler: p.balance},
		{Name: "pay", Help: "/pay [player] [amount] - transfers money", Handler: p.pay},
		{Name: "give", Aliases: []string{"gib"}, Help: "/give [player] [amount] - gives money", MinRank: plugin.RankAdmin, Handler: p.give},
		{Name: "take", Help: "/take [player] [amount|all] - takes money", MinRank: plugin.RankAdmin, Handler: p.take},
		{Name: "economy", Aliases: []string{"eco"}, Help: "/economy [enable|disable|currency NAME]", MinRank: plugin.RankOperator, Handler: p.configure},
	}
	for _, spec := range commands {
		if !server.RegisterCommandSpec(spec) {
			p.unregister()
			return fmt.Errorf("register /%s", spec.Name)
		}
		p.registered = append(p.registered, spec.Name)
	}
	return nil
}

func (p *economyPlugin) Disable() error {
	p.unregister()
	return p.state.flush()
}

func (p *economyPlugin) unregister() {
	if p.server == nil {
		return
	}
	for _, name := range p.registered {
		p.server.UnregisterCommand(name)
	}
	p.registered = nil
}

func (e *economyState) load(path string) error {
	e.path, e.Enabled, e.Currency, e.Accounts = path, true, "coins", make(map[string]int)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read economy: %w", err)
	}
	if err := json.Unmarshal(data, e); err != nil {
		return fmt.Errorf("decode economy: %w", err)
	}
	if e.Currency == "" {
		e.Currency = "coins"
	}
	if e.Accounts == nil {
		e.Accounts = make(map[string]int)
	}
	return nil
}

func (e *economyState) flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.saveLocked()
}

func (e *economyState) saveLocked() error {
	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(e.path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(e.path), "economy-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(data)
	}
	if err == nil {
		err = tmp.Sync()
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpPath, e.path)
	}
	return err
}

func (p *economyPlugin) balance(player plugin.Player, args []string) string {
	if len(args) > 1 {
		return "&WUsage: /balance [player]"
	}
	name := player.Name()
	if len(args) == 1 {
		name = args[0]
	}
	if !validPlayerName(name) {
		return "&WInvalid player name"
	}
	p.state.mu.Lock()
	money, currency, enabled := p.state.Accounts[strings.ToLower(name)], p.state.Currency, p.state.Enabled
	p.state.mu.Unlock()
	if !enabled {
		return "&WEconomy is disabled"
	}
	return fmt.Sprintf("&SEconomy stats for %s&S: &f%d &3%s", name, money, currency)
}

func (p *economyPlugin) pay(player plugin.Player, args []string) string {
	if len(args) != 2 || !validPlayerName(args[0]) {
		return "&WUsage: /pay [player] [amount]"
	}
	amount, err := positiveAmount(args[1])
	if err != nil {
		return "&W" + err.Error()
	}
	from, to := strings.ToLower(player.Name()), strings.ToLower(args[0])
	if from == to {
		return "&WYou cannot pay yourself"
	}
	e := &p.state
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.Enabled {
		return "&WEconomy is disabled"
	}
	if e.Accounts[from] < amount {
		return "&WYou do not have enough " + e.Currency
	}
	if e.Accounts[to] > maxMoney-amount {
		return "&WPlayers cannot have over 16777215 " + e.Currency
	}
	e.Accounts[from] -= amount
	e.Accounts[to] += amount
	if err := e.saveLocked(); err != nil {
		e.Accounts[from] += amount
		e.Accounts[to] -= amount
		return "&WCould not save economy data"
	}
	if target := p.server.FindPlayer(args[0]); target != nil {
		target.Message(fmt.Sprintf("&a%s paid you %d &3%s", player.Name(), amount, e.Currency))
	}
	return fmt.Sprintf("&aPaid %s %d &3%s", args[0], amount, e.Currency)
}

func (p *economyPlugin) give(_ plugin.Player, args []string) string {
	if len(args) != 2 || !validPlayerName(args[0]) {
		return "&WUsage: /give [player] [amount]"
	}
	amount, err := positiveAmount(args[1])
	if err != nil {
		return "&W" + err.Error()
	}
	e := &p.state
	e.mu.Lock()
	defer e.mu.Unlock()
	key := strings.ToLower(args[0])
	if e.Accounts[key] > maxMoney-amount {
		return "&WPlayers cannot have over 16777215 " + e.Currency
	}
	e.Accounts[key] += amount
	if err := e.saveLocked(); err != nil {
		e.Accounts[key] -= amount
		return "&WCould not save economy data"
	}
	return fmt.Sprintf("&aGave %s %d &3%s", args[0], amount, e.Currency)
}

func (p *economyPlugin) take(_ plugin.Player, args []string) string {
	if len(args) != 2 || !validPlayerName(args[0]) {
		return "&WUsage: /take [player] [amount|all]"
	}
	e := &p.state
	e.mu.Lock()
	defer e.mu.Unlock()
	key, old := strings.ToLower(args[0]), e.Accounts[strings.ToLower(args[0])]
	amount := old
	if !strings.EqualFold(args[1], "all") {
		var err error
		amount, err = positiveAmount(args[1])
		if err != nil {
			return "&W" + err.Error()
		}
		if amount > old {
			amount = old
		}
	}
	e.Accounts[key] = old - amount
	if err := e.saveLocked(); err != nil {
		e.Accounts[key] = old
		return "&WCould not save economy data"
	}
	return fmt.Sprintf("&aTook %d &3%s &afrom %s", amount, e.Currency, args[0])
}

func (p *economyPlugin) configure(_ plugin.Player, args []string) string {
	e := &p.state
	e.mu.Lock()
	defer e.mu.Unlock()
	oldEnabled, oldCurrency := e.Enabled, e.Currency
	switch {
	case len(args) == 1 && strings.EqualFold(args[0], "enable"):
		e.Enabled = true
	case len(args) == 1 && strings.EqualFold(args[0], "disable"):
		e.Enabled = false
	case len(args) == 2 && strings.EqualFold(args[0], "currency") && validCurrency(args[1]):
		e.Currency = args[1]
	default:
		return "&WUsage: /economy [enable|disable|currency NAME]"
	}
	if err := e.saveLocked(); err != nil {
		e.Enabled, e.Currency = oldEnabled, oldCurrency
		return "&WCould not save economy data"
	}
	return fmt.Sprintf("&aEconomy enabled=%t, currency=%s", e.Enabled, e.Currency)
}

func positiveAmount(raw string) (int, error) {
	amount, err := strconv.Atoi(raw)
	if err != nil || amount <= 0 || amount > maxMoney {
		return 0, fmt.Errorf("amount must be between 1 and %d", maxMoney)
	}
	return amount, nil
}

func validPlayerName(name string) bool {
	if len(name) < 1 || len(name) > 16 {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

func validCurrency(name string) bool {
	if len(name) < 1 || len(name) > 16 {
		return false
	}
	for _, r := range name {
		if (r < 'A' || r > 'Z') && (r < 'a' || r > 'z') {
			return false
		}
	}
	return true
}

func init() { plugin.Register("mcgalaxy-economy", &economyPlugin{}) }
