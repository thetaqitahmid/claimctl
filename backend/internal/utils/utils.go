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
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

// GenerateStrongPassword returns a random password guaranteed to contain
// uppercase, lowercase, digit, and special characters.
func GenerateStrongPassword(length int) string {
	if length < 8 {
		length = 8
	}

	upper := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	lower := "abcdefghijklmnopqrstuvwxyz"
	digits := "0123456789"
	special := "@$!%*?&#"
	all := upper + lower + digits + special

	buf := make([]byte, length)
	rand.Read(buf)

	password := make([]byte, length)
	for i := range password {
		password[i] = all[int(buf[i])%len(all)]
	}

	password[0] = upper[int(buf[0])%len(upper)]
	password[1] = lower[int(buf[1])%len(lower)]
	password[2] = digits[int(buf[2])%len(digits)]
	password[3] = special[int(buf[3])%len(special)]

	for i := 3; i > 0; i-- {
		j := int(buf[length-i]) % (i + 1)
		password[i], password[j] = password[j], password[i]
	}

	return string(password)
}

// GetEnvAsBool retrieves an environment variable as a boolean or returns a default value
func GetEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		return strings.ToLower(value) == "true" || value == "1"
	}
	return defaultValue
}
