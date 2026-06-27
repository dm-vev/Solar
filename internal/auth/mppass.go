// Package auth implements Minecraft Classic/ClassiCube name verification.
package auth

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	// SaltLength is the default generated salt length in hexadecimal characters.
	SaltLength    = 32
	MinSaltLength = 16
	MaxSaltLength = 128
)

// GenerateSalt returns a cryptographically random salt suitable for ClassiCube heartbeat authentication.
func GenerateSalt() (string, error) {
	bytes := make([]byte, SaltLength/2)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("read random salt: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// ValidSalt reports whether salt is safe to publish and reuse in Classic mppass verification.
func ValidSalt(salt string) bool {
	if len(salt) < MinSaltLength || len(salt) > MaxSaltLength {
		return false
	}
	for _, r := range salt {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

// Mppass calculates the Classic verification token for username and salt.
func Mppass(username, salt string) string {
	sum := md5.Sum([]byte(salt + username))
	return hex.EncodeToString(sum[:])
}

// ValidMppass checks a client-supplied Classic verification token.
func ValidMppass(username, salt, supplied string) bool {
	if username == "" || salt == "" {
		return false
	}

	expected := Mppass(username, salt)
	got := strings.ToLower(strings.TrimSpace(supplied))
	if len(got) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}
