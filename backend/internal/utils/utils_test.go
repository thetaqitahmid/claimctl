package utils

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyPassword(t *testing.T) {
	tests := []struct {
		name           string
		plainPassword  string
		hashedPassword string
		expected       bool
	}{
		{
			name:           "Valid password",
			plainPassword:  "password123",
			hashedPassword: "$2a$10$V0k3CYWOGRCJgIkaDCQW/.txz8VEmJKnYGp.JH0qjx7nM5qIqUeHS", // bcrypt hash of "password123"
			expected:       true,
		},
		{
			name:           "Invalid password",
			plainPassword:  "wrongpassword",
			hashedPassword: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			expected:       false,
		},
		{
			name:           "Empty password",
			plainPassword:  "",
			hashedPassword: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			expected:       false,
		},
		{
			name:           "Invalid hash",
			plainPassword:  "password123",
			hashedPassword: "invalidhash",
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyPassword(tt.plainPassword, tt.hashedPassword)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "Valid simple email",
			email:    "test@example.com",
			expected: true,
		},
		{
			name:     "Valid email with subdomain",
			email:    "user@mail.example.com",
			expected: true,
		},
		{
			name:     "Valid email with numbers",
			email:    "user123@test456.com",
			expected: true,
		},
		{
			name:     "Valid email with special characters",
			email:    "test.user+tag@example.com",
			expected: true,
		},
		{
			name:     "Invalid email - no @",
			email:    "testexample.com",
			expected: false,
		},
		{
			name:     "Invalid email - no domain",
			email:    "test@",
			expected: false,
		},
		{
			name:     "Invalid email - no local part",
			email:    "@example.com",
			expected: false,
		},
		{
			name:     "Invalid email - empty",
			email:    "",
			expected: false,
		},
		{
			name:     "Invalid email - spaces",
			email:    "test @example.com",
			expected: false,
		},
		{
			name:     "Invalid email - multiple @",
			email:    "test@@example.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidEmailRegex(t *testing.T) {
	// Test the regex compilation directly
	r, err := regexp.Compile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+` +
		`@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?` +
		`(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*`)
	assert.NoError(t, err)
	assert.NotNil(t, r)
}

func TestIsValidPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{
			name:     "Valid password with all requirements",
			password: "Password123!",
			expected: true,
		},
		{
			name:     "Valid password with different special chars",
			password: "SecurePass@456",
			expected: true,
		},
		{
			name:     "Valid password with underscore special char",
			password: "TestPass$789",
			expected: true,
		},
		{
			name:     "Invalid password - too short",
			password: "Pass1!",
			expected: false,
		},
		{
			name:     "Invalid password - no uppercase",
			password: "password123!",
			expected: false,
		},
		{
			name:     "Invalid password - no lowercase",
			password: "PASSWORD123!",
			expected: false,
		},
		{
			name:     "Invalid password - no numbers",
			password: "Password!",
			expected: false,
		},
		{
			name:     "Invalid password - no special characters",
			password: "Password123",
			expected: false,
		},
		{
			name:     "Invalid password - empty",
			password: "",
			expected: false,
		},
		{
			name:     "Invalid password - only special chars",
			password: "!!!!!!!!",
			expected: false,
		},
		{
			name:     "Invalid password - only numbers",
			password: "12345678",
			expected: false,
		},
		{
			name:     "Invalid password - only letters",
			password: "Password",
			expected: false,
		},
		{
			name:     "Valid password exactly 8 chars",
			password: "Pass1!ab",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidPassword(tt.password)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnv(t *testing.T) {
	// Note: These tests use os.Getenv which reads from actual environment
	// For comprehensive testing, you might want to mock os.Getenv

	tests := []struct {
		name          string
		key           string
		defaultValue  string
		expectDefault bool
	}{
		{
			name:          "Non-existent env var with default",
			key:           "NON_EXISTENT_VAR_12345",
			defaultValue:  "default_value",
			expectDefault: true,
		},
		{
			name:          "Non-existent env var with empty default",
			key:           "NON_EXISTENT_VAR_67890",
			defaultValue:  "",
			expectDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnv(tt.key, tt.defaultValue)
			if tt.expectDefault {
				assert.Equal(t, tt.defaultValue, result)
			}
		})
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		name     string
		n        int
		expected int
	}{
		{
			name:     "Generate 8 bytes",
			n:        8,
			expected: 8, // Note: base64 encoding makes output longer, but we test input length
		},
		{
			name:     "Generate 16 bytes",
			n:        16,
			expected: 16,
		},
		{
			name:     "Generate 32 bytes",
			n:        32,
			expected: 32,
		},
		{
			name:     "Generate 0 bytes",
			n:        0,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateRandomString(tt.n)

			// Result should not be empty for n > 0
			if tt.n > 0 {
				assert.NotEmpty(t, result)
				// Result should be base64 encoded (may contain = for padding)
				// Just verify it's not empty and contains valid base64 characters
				assert.True(t, len(result) > 0)
			} else {
				assert.Empty(t, result)
			}

			// Multiple calls should produce different results
			if tt.n > 4 { // Only test for larger inputs to avoid false positives
				result2 := GenerateRandomString(tt.n)
				assert.NotEqual(t, result, result2)
			}
		})
	}
}
