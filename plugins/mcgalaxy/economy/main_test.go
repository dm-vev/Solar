//go:build plugin

package main

import (
	"path/filepath"
	"testing"
)

func TestEconomyPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "economy.json")
	var first economyState
	if err := first.load(path); err != nil {
		t.Fatal(err)
	}
	first.Accounts["alice"] = 42
	if err := first.flush(); err != nil {
		t.Fatal(err)
	}
	var second economyState
	if err := second.load(path); err != nil {
		t.Fatal(err)
	}
	if second.Accounts["alice"] != 42 || !second.Enabled || second.Currency != "coins" {
		t.Fatalf("reloaded economy = %+v", &second)
	}
}

func TestValidation(t *testing.T) {
	if !validPlayerName("Classic_User42") || validPlayerName("not valid") {
		t.Fatal("player name validation failed")
	}
	if amount, err := positiveAmount("100"); err != nil || amount != 100 {
		t.Fatalf("positiveAmount = %d, %v", amount, err)
	}
	if _, err := positiveAmount("-1"); err == nil {
		t.Fatal("positiveAmount accepted negative value")
	}
}
