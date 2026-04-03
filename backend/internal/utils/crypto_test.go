package utils

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a valid 32-byte key encoded in base64
	key := make([]byte, 32)
	// fill with some dummy data for test
	for i := range key {
		key[i] = byte(i)
	}
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	originalText := "super_secret_value"

	encrypted, err := Encrypt(originalText, keyBase64)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == originalText {
		t.Fatal("Encrypted text should not match original text")
	}

	decrypted, err := Decrypt(encrypted, keyBase64)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != originalText {
		t.Errorf("Decrypted text '%s' does not match original '%s'", decrypted, originalText)
	}
}

func TestDecryptParams(t *testing.T) {
	key := make([]byte, 32)
	keyBase64 := base64.StdEncoding.EncodeToString(key)

	_, err := Decrypt("invalid_base64", keyBase64)
	if err == nil {
		t.Error("Expected error for invalid base64 ciphertext")
	}

	_, err = Decrypt(base64.StdEncoding.EncodeToString([]byte("short")), keyBase64)
	if err == nil {
		t.Error("Expected error for short ciphertext")
	}
}
