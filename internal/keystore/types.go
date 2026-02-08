package keystore

import (
	"strings"
	"time"
)

// Key represents a Gemini API key record stored in SQLite.
type Key struct {
	ID        int64
	Key       string
	Note      string
	Enabled   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// MaskKey hides most of the key for display purposes.
func MaskKey(raw string) string {
	key := strings.TrimSpace(raw)
	if len(key) <= 8 {
		return key
	}
	return key[:4] + "..." + key[len(key)-4:]
}
