package app

import (
	"context"
	"testing"
)

func TestRunVersionCommand(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"version"})
	if err != nil {
		t.Fatalf("Run version returned error: %v", err)
	}
}

func TestRunHelpCommand(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"help"})
	if err != nil {
		t.Fatalf("Run help returned error: %v", err)
	}
}

func TestRunNoArgsReturnsHelp(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("Run with no args returned error: %v", err)
	}
}

func TestRunUnknownCommandReturnsError(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRunLoadtestWithInvalidClients(t *testing.T) {
	t.Parallel()

	err := Run(context.Background(), []string{"loadtest", "--clients", "0"})
	if err == nil {
		t.Fatal("expected error for zero clients")
	}
}
