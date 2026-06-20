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

// SpamSpamCategory identifies which rate limit was exceeded.
type SpamSpamCategory int

const (
	SpamCatChat SpamSpamCategory = iota
	SpamCatBlock
	SpamCatCommand
)

// SpamAction is the response when a rate limit is exceeded.
type SpamAction string

const (
	SpamActionKick SpamAction = "kick"
	SpamActionMute SpamAction = "mute"
	SpamActionWarn SpamAction = "warn"
)

// SpamConfig holds the anti-spam configuration.
type SpamConfig struct {
	Enabled      bool
	ChatMax      int
	ChatWindow   time.Duration
	BlockMax     int
	BlockWindow  time.Duration
	CmdMax       int
	CmdWindow    time.Duration
	Action       SpamAction
	MuteDuration time.Duration
}

// SpamChecker tracks per-player rate limits.
type SpamChecker struct {
	mu      sync.Mutex
	config  SpamConfig
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
func NewChecker(cfg SpamConfig) *SpamChecker {
	return &SpamChecker{
		config:  cfg,
		players: make(map[string]*playerState),
	}
}

// SpamResult holds the outcome of a rate check.
type SpamResult struct {
	Exceeded bool
	Category SpamSpamCategory
	Action   SpamAction
	Count    int
	Max      int
}

// CheckChat records a chat message and returns whether the rate limit
// was exceeded.
func (c *SpamChecker) CheckChat(name string) SpamResult {
	return c.check(name, SpamCatChat, c.config.ChatMax, c.config.ChatWindow)
}

// CheckBlock records a block change and returns whether the rate limit
// was exceeded.
func (c *SpamChecker) CheckBlock(name string) SpamResult {
	return c.check(name, SpamCatBlock, c.config.BlockMax, c.config.BlockWindow)
}

// CheckCommand records a command and returns whether the rate limit
// was exceeded.
func (c *SpamChecker) CheckCommand(name string) SpamResult {
	return c.check(name, SpamCatCommand, c.config.CmdMax, c.config.CmdWindow)
}

// IsMuted reports whether the player is currently muted by anti-spam.
func (c *SpamChecker) IsMuted(name string) bool {
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
func (c *SpamChecker) Reset(name string) {
	name = strings.ToLower(name)
	c.mu.Lock()
	delete(c.players, name)
	c.mu.Unlock()
}

func (c *SpamChecker) check(name string, cat SpamSpamCategory, max int, window time.Duration) SpamResult {
	if !c.config.Enabled || max <= 0 || window <= 0 {
		return SpamResult{Exceeded: false}
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
	case SpamCatChat:
		times = &ps.chatTimes
	case SpamCatBlock:
		times = &ps.blockTimes
	case SpamCatCommand:
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
		return SpamResult{Exceeded: false, Count: count, Max: max}
	}

	// Exceeded — apply action.
	switch c.config.Action {
	case SpamActionMute:
		ps.muted = true
		ps.mutedUntil = now.Add(c.config.MuteDuration)
	case SpamActionWarn:
		// Warning only — no state change needed.
	}

	return SpamResult{
		Exceeded: true,
		Category: cat,
		Action:   c.config.Action,
		Count:    count,
		Max:      max,
	}
}
