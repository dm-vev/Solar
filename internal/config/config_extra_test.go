package config

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestLoggerAndFormatStringSlice(t *testing.T) {
	var buf bytes.Buffer
	logger := Config{Log: LogConfig{Level: "debug", Format: "json"}}.Logger(&buf)
	logger.Debug("debug message")
	if !strings.Contains(buf.String(), "debug message") {
		t.Fatalf("debug json log missing message: %q", buf.String())
	}

	buf.Reset()
	logger = Config{Log: LogConfig{Level: "warn", Format: "text"}}.Logger(&buf)
	logger.Info("hidden")
	logger.Warn("visible")
	if strings.Contains(buf.String(), "hidden") || !strings.Contains(buf.String(), "visible") {
		t.Fatalf("warn text log = %q", buf.String())
	}

	for _, level := range []string{"error", "info", "unknown"} {
		buf.Reset()
		logger = Config{Log: LogConfig{Level: level}}.Logger(&buf)
		logger.LogAttrs(nil, slog.LevelError, "error message")
		if !strings.Contains(buf.String(), "error message") {
			t.Fatalf("%s logger did not emit error: %q", level, buf.String())
		}
	}

	if got := formatStringSlice(nil); got != "[]" {
		t.Fatalf("formatStringSlice nil = %q", got)
	}
	if got := formatStringSlice([]string{"alice", "bob"}); got != `["alice", "bob"]` {
		t.Fatalf("formatStringSlice = %q", got)
	}
}
