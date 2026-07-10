//go:build plugin

// Package main provides MCGalaxy chat commands as an independent Solar plugin.
package main

import (
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/solar-mc/solar/plugin"
)

type chatPlugin struct {
	server     plugin.Server
	registered []string
	eightMu    sync.Mutex
	nextEight  time.Time
}

func (p *chatPlugin) Name() string { return "mcgalaxy-chat" }
func (p *chatPlugin) Init() error  { return nil }

func (p *chatPlugin) Enable(server plugin.Server) error {
	p.server = server
	commands := []plugin.CommandSpec{
		{Name: "roll", Help: "/roll [min] [max] - rolls a random number", Handler: p.roll},
		{Name: "8ball", Help: "/8ball [question] - asks the all-knowing 8-Ball", Handler: p.eightBall},
		{Name: "high5", Help: "/high5 [player] - high five someone", Handler: p.highFive},
		{Name: "hug", Help: "/hug [player] [loving|friendly|creepy] - hugs someone", Handler: p.hug},
		{Name: "eat", Help: "/eat - eats a random snack", Handler: p.eat},
		{Name: "say", Aliases: []string{"broadcast"}, Help: "/say [message] - broadcasts a server message", MinRank: plugin.RankOperator, Handler: p.say},
		{Name: "clear", Aliases: []string{"cls", "playercls"}, Help: "/clear [global] - clears chat", Handler: p.clear},
		{Name: "color", Aliases: []string{"colour"}, Help: "/color [color] - sets your name color", MinRank: plugin.RankOperator, Handler: p.color},
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

func (p *chatPlugin) Disable() error {
	p.unregister()
	return nil
}

func (p *chatPlugin) unregister() {
	if p.server == nil {
		return
	}
	for _, name := range p.registered {
		p.server.UnregisterCommand(name)
	}
	p.registered = nil
}

func (p *chatPlugin) roll(player plugin.Player, args []string) string {
	min, max, err := rollRange(args)
	if err != nil {
		return "&W" + err.Error()
	}
	n := min + rand.Int64N(max-min+1)
	p.server.BroadcastMessage(fmt.Sprintf("%s &Srolled a &a%d &S(%d|%d)", player.Name(), n, min, max))
	return ""
}

func rollRange(args []string) (int64, int64, error) {
	if len(args) > 2 {
		return 0, 0, errors.New("usage: /roll [min] [max]")
	}
	min, max := int64(1), int64(6)
	var err error
	if len(args) == 1 {
		max, err = strconv.ParseInt(args[0], 10, 32)
	} else if len(args) == 2 {
		min, err = strconv.ParseInt(args[0], 10, 32)
		if err == nil {
			max, err = strconv.ParseInt(args[1], 10, 32)
		}
	}
	if err != nil {
		return 0, 0, errors.New("min and max must be integers")
	}
	if min > max {
		min, max = max, min
	}
	return min, max, nil
}

func (p *chatPlugin) eightBall(player plugin.Player, args []string) string {
	question := strings.TrimSpace(plugin.StripColor(strings.Join(args, " ")))
	if question == "" {
		return "&WUsage: /8ball [yes or no question]"
	}
	p.eightMu.Lock()
	remaining := time.Until(p.nextEight)
	if remaining > 0 {
		p.eightMu.Unlock()
		return fmt.Sprintf("&WThe 8-Ball is recharging for %d seconds", int((remaining+time.Second-1)/time.Second))
	}
	p.nextEight = time.Now().Add(12 * time.Second)
	p.eightMu.Unlock()
	p.server.BroadcastMessage(player.Name() + " &Sasked the &b8-Ball: &f" + question)
	p.server.Scheduler().After(2*time.Second, func() {
		answers := [...]string{"Yes.", "No.", "Definitely.", "Very doubtful.", "Ask again later.", "Without a doubt.", "Cannot predict now.", "Signs point to yes."}
		h := fnv.New32a()
		_, _ = h.Write([]byte(strings.ToLower(question)))
		p.server.BroadcastMessage("The &b8-Ball &Ssays: &f" + answers[int(h.Sum32())%len(answers)])
	})
	return ""
}

func (p *chatPlugin) highFive(player plugin.Player, args []string) string {
	target, err := p.target(args)
	if err != nil {
		return "&W" + err.Error()
	}
	p.server.BroadcastMessage(player.Name() + " &Sjust highfived " + target.Name())
	return ""
}

func (p *chatPlugin) hug(player plugin.Player, args []string) string {
	if len(args) < 1 || len(args) > 2 {
		return "&WUsage: /hug [player] [loving|friendly|creepy]"
	}
	target := p.server.FindPlayer(args[0])
	if target == nil {
		return "&WPlayer not found"
	}
	typeName := ""
	if len(args) == 2 {
		typeName = strings.ToLower(args[1])
		if typeName != "loving" && typeName != "friendly" && typeName != "creepy" {
			return "&WUnknown hug type"
		}
		typeName += " "
	}
	p.server.BroadcastMessage(player.Name() + " &Sgave " + target.Name() + " &Sa " + typeName + "hug")
	return ""
}

func (p *chatPlugin) eat(player plugin.Player, args []string) string {
	if len(args) != 0 {
		return "&WUsage: /eat"
	}
	snacks := [...]string{"apple", "cookie", "sandwich", "cake", "melon", "mushroom stew"}
	p.server.BroadcastMessage(player.Name() + " &Sate a &f" + snacks[rand.IntN(len(snacks))])
	return ""
}

func (p *chatPlugin) say(_ plugin.Player, args []string) string {
	message := strings.Join(args, " ")
	if message == "" {
		return "&WUsage: /say [message]"
	}
	p.server.BroadcastMessage(message)
	return ""
}

func (p *chatPlugin) clear(player plugin.Player, args []string) string {
	global := len(args) == 1 && strings.EqualFold(args[0], "global")
	if len(args) > 1 || (len(args) == 1 && !global) {
		return "&WUsage: /clear [global]"
	}
	if global && player.Rank() < plugin.RankAdmin {
		return "&WYou do not have permission to clear global chat"
	}
	targets := []plugin.Player{player}
	if global {
		targets = p.server.OnlinePlayers()
	}
	for _, target := range targets {
		for range 30 {
			target.Message(" ")
		}
	}
	if global {
		p.server.BroadcastMessage("&4Global chat cleared.")
		return ""
	}
	return "&4Chat cleared."
}

func (p *chatPlugin) color(player plugin.Player, args []string) string {
	if len(args) != 1 {
		return "&WUsage: /color [color]"
	}
	name := strings.ToLower(args[0])
	if !validColor(name) {
		return "&WUnknown color"
	}
	color := plugin.ParseColor(name)
	player.SetColor(string(color))
	return "&SYour color is now " + string(color) + plugin.ColorName(color)
}

func validColor(name string) bool {
	const names = " black navy darkblue green darkgreen teal darkaqua darkcyan maroon darkred purple darkpurple gold orange darkyellow silver lightgray lightgrey gray grey darkgray darkgrey blue lime lightgreen aqua cyan lightblue red pink magenta yellow white "
	return strings.Contains(names, " "+name+" ")
}

func (p *chatPlugin) target(args []string) (plugin.Player, error) {
	if len(args) != 1 {
		return nil, errors.New("usage: /high5 [player]")
	}
	target := p.server.FindPlayer(args[0])
	if target == nil {
		return nil, errors.New("player not found")
	}
	return target, nil
}

func init() { plugin.Register("mcgalaxy-chat", &chatPlugin{}) }
