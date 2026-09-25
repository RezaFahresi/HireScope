package crypto

import (
	"testing"
)

func TestTokenEncryptor_EncryptDecrypt(t *testing.T) {
	enc, err := NewTokenEncryptor("my-secret-calendar-key-32-chars!")
	if err != nil {
		t.Fatalf("unexpected error creating encryptor: %v", err)
	}

	plaintext := "ya29.a0AfH6SMBySampleOAuthAccessToken12345"

	ciphertext, err := enc.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("unexpected error encrypting: %v", err)
	}

	if ciphertext == plaintext {
		t.Fatalf("ciphertext should not match plaintext")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("unexpected error decrypting: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("expected decrypted '%s', got '%s'", plaintext, decrypted)
	}
}

func TestTokenEncryptor_WrongKeyFails(t *testing.T) {
	enc1, _ := NewTokenEncryptor("key-number-one-super-secret-key-1")
	enc2, _ := NewTokenEncryptor("key-number-two-super-secret-key-2")

	ciphertext, err := enc1.Encrypt("confidential-refresh-token")
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	_, err = enc2.Decrypt(ciphertext)
	if err == nil {
		t.Fatalf("expected decryption failure with wrong key, got nil")
	}
}

func TestTokenEncryptor_EmptyStrings(t *testing.T) {
	enc, _ := NewTokenEncryptor("")
	encStr, err := enc.Encrypt("")
	if err != nil || encStr != "" {
		t.Fatalf("empty string encryption failed: encStr=%s, err=%v", encStr, err)
	}

	decStr, err := enc.Decrypt("")
	if err != nil || decStr != "" {
		t.Fatalf("empty string decryption failed: decStr=%s, err=%v", decStr, err)
	}
}
