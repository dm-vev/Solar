//go:build plugin

package main

import "testing"

func TestRollRange(t *testing.T) {
	min, max, err := rollRange([]string{"10", "2"})
	if err != nil || min != 2 || max != 10 {
		t.Fatalf("rollRange = %d, %d, %v", min, max, err)
	}
	if _, _, err := rollRange([]string{"nope"}); err == nil {
		t.Fatal("rollRange accepted invalid integer")
	}
}
