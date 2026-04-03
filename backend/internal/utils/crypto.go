package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Encrypt encrypts plaintext using AES-GCM with the provided key.
// The key should be a base64 encoded string of 32 bytes (for AES-256).
// Returns a base64 encoded string containing the nonce and ciphertext.
func Encrypt(plaintext, keyBase64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", fmt.Errorf("invalid encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts a base64 encoded ciphertext using AES-GCM with the provided key.
// The key should be a base64 encoded string of 32 bytes.
func Decrypt(ciphertextBase64, keyBase64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", fmt.Errorf("invalid encryption key: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// LoadOrGenerateKey attempts to load the encryption key from an environment variable,
// then a file. If neither exists, it generates a new key and saves it to the file.
func LoadOrGenerateKey(envVarName, keyFilePath string) (string, error) {
	// 1. Check Env Var
	val := os.Getenv(envVarName)
	if val != "" {
		slog.Info("Loaded encryption key from environment variable", "var", envVarName)
		return val, nil
	}

	// 2. Check File
	// Ensure directory exists
	dir := filepath.Dir(keyFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create key directory: %w", err)
	}

	content, err := os.ReadFile(keyFilePath)
	if err == nil {
		slog.Info("Loaded encryption key from file", "path", keyFilePath)
		return strings.TrimSpace(string(content)), nil
	}

	// 3. Generate
	slog.Info("Encryption key not found. Generating new key...", "path", keyFilePath)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	if err := os.WriteFile(keyFilePath, []byte(keyBase64), 0600); err != nil {
		return "", fmt.Errorf("failed to write key to file: %w", err)
	}

	return keyBase64, nil
}
