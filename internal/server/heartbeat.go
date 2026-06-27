// heartbeat.go implements the ClassiCube.net heartbeat that publishes
// the server to the public server list.
//
// The heartbeat sends a POST request to https://www.classicube.net/heartbeat
// with the following form fields:
//   - port:     the server port
//   - max:      max players
//   - name:     server name (URL-encoded)
//   - public:   whether the server is publicly listed (true/false)
//   - version:  protocol version (7 for Classic)
//   - salt:     random salt for name verification
//   - users:    current online player count
//   - software: server software name + version (URL-encoded)
//   - web:      whether a web client is supported (false)
//
// The response contains a JSON object with a "response" field holding
// the public URL for the server. If the response contains "errors",
// the heartbeat failed and the error is logged.
//
// The heartbeat runs in a background goroutine started by the server.
// It retries up to 3 times per beat and logs failures without stopping.

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/solar-mc/solar/internal/auth"
)

const (
	heartbeatURL      = "https://www.classicube.net/heartbeat"
	heartbeatInterval = 15 * time.Second
	heartbeatTimeout  = 10 * time.Second
	heartbeatRetries  = 3
	heartbeatVersion  = "7"
)

// HeartbeatConfig holds the parameters for the ClassiCube heartbeat.
type HeartbeatConfig struct {
	Port        int
	MaxPlayers  int
	Name        string
	Public      bool
	Software    string
	Salt        string
	OnlineCount func() int
}

// StartHeartbeat runs the ClassiCube heartbeat loop in a background goroutine.
// It posts server info every 15 seconds until ctx is canceled.
// The public URL is sent to urlCh (buffered, size 1) when received.
func StartHeartbeat(ctx context.Context, cfg HeartbeatConfig, logger *slog.Logger, urlCh chan<- string) {
	salt := cfg.Salt
	if salt == "" {
		var err error
		salt, err = generateSalt()
		if err != nil {
			logger.Warn("heartbeat disabled; failed to generate auth salt", "error", err)
			return
		}
	}

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()

		// Send immediately on start, then every interval.
		beat(ctx, cfg, salt, logger, urlCh)
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				beat(ctx, cfg, salt, logger, urlCh)
			}
		}
	}()
}

func beat(ctx context.Context, cfg HeartbeatConfig, salt string, logger *slog.Logger, urlCh chan<- string) {
	data := url.Values{}
	data.Set("port", fmt.Sprintf("%d", cfg.Port))
	data.Set("max", fmt.Sprintf("%d", cfg.MaxPlayers))
	data.Set("name", cfg.Name)
	data.Set("public", fmt.Sprintf("%t", cfg.Public))
	data.Set("version", heartbeatVersion)
	data.Set("salt", salt)
	users := 0
	if cfg.OnlineCount != nil {
		users = cfg.OnlineCount()
	}
	data.Set("users", fmt.Sprintf("%d", users))
	data.Set("software", cfg.Software)
	data.Set("web", "false")

	body := data.Encode()

	var lastErr error
	for attempt := 0; attempt < heartbeatRetries; attempt++ {
		if ctx.Err() != nil {
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", heartbeatURL, strings.NewReader(body))
		if err != nil {
			lastErr = err
			continue
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{Timeout: heartbeatTimeout}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			lastErr = err
			continue
		}

		text := strings.TrimSpace(string(respBody))
		if strings.Contains(text, `"errors"`) {
			errMsg := parseHeartbeatError(text)
			logger.Warn("heartbeat error", "error", errMsg)
			return
		}

		// Success — extract the URL from the response.
		respURL := strings.TrimSpace(text)
		if respURL != "" && urlCh != nil {
			select {
			case urlCh <- respURL:
			default:
			}
		}
		logger.Debug("heartbeat ok", "url", respURL)
		return
	}

	if lastErr != nil {
		logger.Warn("heartbeat failed", "error", lastErr)
	}
}

// parseHeartbeatError extracts the first error message from a
// ClassiCube heartbeat error JSON response.
func parseHeartbeatError(jsonStr string) string {
	var resp struct {
		Errors [][]string `json:"errors"`
		Status string     `json:"status"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		return jsonStr
	}
	if len(resp.Errors) > 0 && len(resp.Errors[0]) > 0 {
		return resp.Errors[0][0]
	}
	return "unknown heartbeat error"
}

// generateSalt produces a random salt for name verification.
func generateSalt() (string, error) {
	return auth.GenerateSalt()
}
