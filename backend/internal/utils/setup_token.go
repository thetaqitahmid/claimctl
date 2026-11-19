package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// GenerateSetupToken returns a one-time first-run setup token.
func GenerateSetupToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate setup token: %w", err)
	}
	return "st_" + hex.EncodeToString(b), nil
}
