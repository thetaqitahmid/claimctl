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
	"time"

	"golang.org/x/sys/unix"
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
	validate := func(value string) (string, error) {
		key, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(key) != 32 {
			return "", fmt.Errorf("encryption key must be base64 encoded 32 bytes")
		}
		return value, nil
	}
	val := os.Getenv(envVarName)
	if val != "" {
		slog.Info("Loaded encryption key from environment variable", "var", envVarName)
		return validate(val)
	}

	// 2. Check File
	// Ensure directory exists
	dir := filepath.Dir(keyFilePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create key directory: %w", err)
	}

	lock, err := os.OpenFile(keyFilePath+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	defer lock.Close()
	deadline := time.Now().Add(5 * time.Second)
	for {
		err := unix.Flock(int(lock.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			break
		}
		if !errors.Is(err, unix.EWOULDBLOCK) {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("encryption key initialization lock timed out")
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer unix.Flock(int(lock.Fd()), unix.LOCK_UN)
	content, err := os.ReadFile(keyFilePath)
	if err == nil {
		slog.Info("Loaded encryption key from file", "path", keyFilePath)
		return validate(strings.TrimSpace(string(content)))
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if _, statErr := os.Lstat(keyFilePath); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
		return "", fmt.Errorf("existing encryption key file cannot be read")
	}

	// 3. Generate
	slog.Info("Encryption key not found. Generating new key...", "path", keyFilePath)
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	tmp, err := os.CreateTemp(dir, ".app-key-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(keyBase64); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := os.Rename(tmp.Name(), keyFilePath); err != nil {
		return "", err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return "", err
	}
	return keyBase64, nil
}
