// Package antispam implements configurable rate limiting for chat,
// block changes, and commands.
//
// Each player has a per-category sliding window counter. When the
// count exceeds the configured max within the time window, the
// configured action is triggered (kick, mute, or warn).
//
// The checker is safe for concurrent use. Player state is tracked
// in a map keyed by normalized username.

package player

import (
	"strings"
	"sync"
	"time"
)

// Category identifies which rate limit was exceeded.
type Category int

const (
	CatChat Category = iota
	CatBlock
	CatCommand
)

// Action is the response when a rate limit is exceeded.
type Action string

const (
	ActionKick Action = "kick"
	ActionMute Action = "mute"
	ActionWarn Action = "warn"
)

// Config holds the anti-spam configuration.
type Config struct {
	Enabled      bool
	ChatMax      int
	ChatWindow   time.Duration
	BlockMax     int
	BlockWindow  time.Duration
	CmdMax       int
	CmdWindow    time.Duration
	Action       Action
	MuteDuration time.Duration
}

// Checker tracks per-player rate limits.
type Checker struct {
	mu      sync.Mutex
	config  Config
	players map[string]*playerState
}

type playerState struct {
	chatTimes  []time.Time
	blockTimes []time.Time
	cmdTimes   []time.Time
	muted      bool
	mutedUntil time.Time
}

// New creates a SpamChecker with the given configuration.
func New(cfg Config) *Checker {
	return &Checker{
		config:  cfg,
		players: make(map[string]*playerState),
	}
}

// Result holds the outcome of a rate check.
type Result struct {
	Exceeded bool
	Category Category
	Action   Action
	Count    int
	Max      int
}

// CheckChat records a chat message and returns whether the rate limit
// was exceeded.
func (c *Checker) CheckChat(name string) Result {
	return c.check(name, CatChat, c.config.ChatMax, c.config.ChatWindow)
}

// CheckBlock records a block change and returns whether the rate limit
// was exceeded.
func (c *Checker) CheckBlock(name string) Result {
	return c.check(name, CatBlock, c.config.BlockMax, c.config.BlockWindow)
}

// CheckCommand records a command and returns whether the rate limit
// was exceeded.
func (c *Checker) CheckCommand(name string) Result {
	return c.check(name, CatCommand, c.config.CmdMax, c.config.CmdWindow)
}

// IsMuted reports whether the player is currently muted by anti-spam.
func (c *Checker) IsMuted(name string) bool {
	name = strings.ToLower(name)
	c.mu.Lock()
	defer c.mu.Unlock()
	ps := c.players[name]
	if ps == nil {
		return false
	}
	if ps.muted && time.Now().After(ps.mutedUntil) {
		ps.muted = false
	}
	return ps.muted
}

// Reset clears all state for a player (called on disconnect).
func (c *Checker) Reset(name string) {
	name = strings.ToLower(name)
	c.mu.Lock()
	delete(c.players, name)
	c.mu.Unlock()
}

func (c *Checker) check(name string, cat Category, max int, window time.Duration) Result {
	if !c.config.Enabled || max <= 0 || window <= 0 {
		return Result{Exceeded: false}
	}

	name = strings.ToLower(name)
	c.mu.Lock()
	defer c.mu.Unlock()

	ps := c.players[name]
	if ps == nil {
		ps = &playerState{}
		c.players[name] = ps
	}

	now := time.Now()
	var times *[]time.Time
	switch cat {
	case CatChat:
		times = &ps.chatTimes
	case CatBlock:
		times = &ps.blockTimes
	case CatCommand:
		times = &ps.cmdTimes
	}

	// Prune timestamps outside the window.
	cutoff := now.Add(-window)
	pruned := (*times)[:0]
	for _, t := range *times {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	*times = pruned

	// Add current timestamp.
	*times = append(*times, now)
	count := len(*times)

	if count <= max {
		return Result{Exceeded: false, Count: count, Max: max}
	}

	// Exceeded — apply action.
	switch c.config.Action {
	case ActionMute:
		ps.muted = true
		ps.mutedUntil = now.Add(c.config.MuteDuration)
	case ActionWarn:
		// Warning only — no state change needed.
	}

	return Result{
		Exceeded: true,
		Category: cat,
		Action:   c.config.Action,
		Count:    count,
		Max:      max,
	}
}
