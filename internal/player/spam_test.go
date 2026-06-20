package player

import (
	"testing"
	"time"
)

func TestChatNotExceeded(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:    true,
		ChatMax:    5,
		ChatWindow: 3 * time.Second,
		Action:     SpamActionKick,
	})
	for i := 0; i < 5; i++ {
		r := c.CheckChat("alice")
		if r.Exceeded {
			t.Fatalf("check %d: unexpectedly exceeded", i)
		}
	}
}

func TestChatExceeded(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:    true,
		ChatMax:    3,
		ChatWindow: 3 * time.Second,
		Action:     SpamActionKick,
	})
	for i := 0; i < 3; i++ {
		c.CheckChat("bob")
	}
	r := c.CheckChat("bob")
	if !r.Exceeded {
		t.Fatal("should exceed after 4 messages")
	}
	if r.Action != SpamActionKick {
		t.Fatalf("action = %s, want kick", r.Action)
	}
}

func TestMuteAction(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:      true,
		ChatMax:      2,
		ChatWindow:   3 * time.Second,
		Action:       SpamActionMute,
		MuteDuration: 1 * time.Hour,
	})
	c.CheckChat("carol")
	c.CheckChat("carol")
	r := c.CheckChat("carol")
	if !r.Exceeded || r.Action != SpamActionMute {
		t.Fatalf("should mute, got %+v", r)
	}
	if !c.IsMuted("carol") {
		t.Fatal("should be muted")
	}
}

func TestWindowExpiry(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:    true,
		ChatMax:    2,
		ChatWindow: 50 * time.Millisecond,
		Action:     SpamActionKick,
	})
	c.CheckChat("dave")
	c.CheckChat("dave")
	time.Sleep(60 * time.Millisecond)
	r := c.CheckChat("dave")
	if r.Exceeded {
		t.Fatal("should not exceed after window expired")
	}
}

func TestDisabled(t *testing.T) {
	c := NewChecker(SpamConfig{Enabled: false, ChatMax: 1, ChatWindow: 1 * time.Second})
	r := c.CheckChat("eve")
	if r.Exceeded {
		t.Fatal("should never exceed when disabled")
	}
}

func TestBlockAndCommand(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:     true,
		BlockMax:    2,
		BlockWindow: 3 * time.Second,
		CmdMax:      1,
		CmdWindow:   3 * time.Second,
		Action:      SpamActionWarn,
	})
	c.CheckBlock("frank")
	c.CheckBlock("frank")
	r := c.CheckBlock("frank")
	if !r.Exceeded || r.Category != SpamCatBlock {
		t.Fatalf("block should exceed, got %+v", r)
	}
	c.CheckCommand("frank")
	r = c.CheckCommand("frank")
	if !r.Exceeded || r.Category != SpamCatCommand {
		t.Fatalf("command should exceed, got %+v", r)
	}
}

func TestReset(t *testing.T) {
	c := NewChecker(SpamConfig{
		Enabled:      true,
		ChatMax:      1,
		ChatWindow:   3 * time.Second,
		Action:       SpamActionMute,
		MuteDuration: 1 * time.Hour,
	})
	c.CheckChat("grace")
	c.CheckChat("grace") // exceeds, mutes
	if !c.IsMuted("grace") {
		t.Fatal("should be muted")
	}
	c.Reset("grace")
	if c.IsMuted("grace") {
		t.Fatal("should not be muted after reset")
	}
}
