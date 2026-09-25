package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

var (
	ErrEmptyKey        = errors.New("encryption key cannot be empty")
	ErrInvalidPayload  = errors.New("invalid encrypted payload")
	ErrDecryptionFail  = errors.New("failed to decrypt ciphertext")
)

// TokenEncryptor handles AES-256-GCM authenticated encryption and decryption of secrets.
type TokenEncryptor struct {
	key []byte
}

// NewTokenEncryptor initializes a TokenEncryptor with a given key string.
// If the key is not exactly 32 bytes, SHA-256 is used to derive a deterministic 32-byte key.
func NewTokenEncryptor(keyStr string) (*TokenEncryptor, error) {
	keyStr = strings.TrimSpace(keyStr)
	if keyStr == "" {
		// Default development fallback key if not set, preventing crashes in local dev
		keyStr = "hirescope-default-dev-token-encryption-key-32b"
	}

	hasher := sha256.New()
	hasher.Write([]byte(keyStr))
	derivedKey := hasher.Sum(nil)

	return &TokenEncryptor{
		key: derivedKey,
	}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Returns a base64-encoded string: base64(nonce + ciphertext + tag).
func (e *TokenEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate random nonce: %w", err)
	}

	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt decrypts a base64-encoded payload produced by Encrypt.
func (e *TokenEncryptor) Decrypt(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: base64 decode failed", ErrInvalidPayload)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM cipher: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidPayload)
	}

	nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: authentication failed", ErrDecryptionFail)
	}

	return string(plaintext), nil
}
