package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// VerifyPassword compares the hashed password against the real password
func VerifyPassword(plainPassword, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

// IsValidEmail verifies if an email address is valid
func IsValidEmail(email string) bool {
	r, err := regexp.Compile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+` +
		`@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?` +
		`(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*`)
	if err != nil {
		slog.Error("Invalid regex for email", "error", err)
		return false
	}
	return r.MatchString(email)
}

// IsValidPassword verfies if a password is valid and strong
func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	// Check for required character types
	hasLower := false
	hasUpper := false
	hasNumber := false
	hasSpecial := false
	specialChars := "@$!%*?&#"

	for _, char := range password {
		switch {
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsDigit(char):
			hasNumber = true
		case strings.ContainsRune(specialChars, char):
			hasSpecial = true
		}
	}

	return hasLower && hasUpper && hasNumber && hasSpecial
}

// GetEnv retrieves an environment variable or returns a default value
func GetEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// GetEnvAsInt retrieves an environment variable as an integer or returns a default value
func GetEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		var intValue int
		_, err := fmt.Sscanf(value, "%d", &intValue)
		if err == nil {
			return intValue
		}
		slog.Warn("Failed to parse environment variable as int, using default",
			"key", key,
			"value", value,
			"default", defaultValue)
	}
	return defaultValue
}

// GenerateRandomString returns a URL-safe random string of n bytes
func GenerateRandomString(n int) string {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to simpler random if crypto/rand fails (unlikely)
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GetEnvAsBool retrieves an environment variable as a boolean or returns a default value
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}
