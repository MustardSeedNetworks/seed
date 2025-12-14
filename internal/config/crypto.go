// Package config provides configuration management with encryption support.
package config

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

const (
	// encryptedPrefix identifies encrypted values in config
	encryptedPrefix = "enc:"
)

var (
	// ErrInvalidCiphertext is returned when decryption fails due to invalid input
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// deriveKey derives a 32-byte AES-256 key from the master secret using SHA-256.
// This provides a consistent key for encryption/decryption.
func deriveKey(masterSecret string) []byte {
	hash := sha256.Sum256([]byte(masterSecret))
	return hash[:]
}

// EncryptCredential encrypts a credential string using AES-256-GCM (fixes #518).
// The encrypted value is prefixed with "enc:" to identify it as encrypted.
// Format: enc:base64(nonce||ciphertext||tag)
func EncryptCredential(plaintext, masterSecret string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	// Already encrypted
	if strings.HasPrefix(plaintext, encryptedPrefix) {
		return plaintext, nil
	}

	key := deriveKey(masterSecret)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 and add prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encryptedPrefix + encoded, nil
}

// DecryptCredential decrypts a credential string using AES-256-GCM (fixes #518).
// If the value doesn't have the "enc:" prefix, it's returned as-is (backward compatibility).
func DecryptCredential(encrypted, masterSecret string) (string, error) {
	if encrypted == "" {
		return "", nil
	}

	// Not encrypted, return as-is (backward compatibility during migration)
	if !strings.HasPrefix(encrypted, encryptedPrefix) {
		return encrypted, nil
	}

	// Remove prefix
	encoded := strings.TrimPrefix(encrypted, encryptedPrefix)

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("%w: invalid base64: %v", ErrInvalidCiphertext, err)
	}

	key := deriveKey(masterSecret)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("%w: authentication failed: %v", ErrInvalidCiphertext, err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a credential value is encrypted.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}

// EncryptSNMPCredentials encrypts all SNMP v3 credentials in the config (fixes #518).
func (c *Config) EncryptSNMPCredentials() error {
	if c.Auth.JWTSecret == "" {
		return errors.New("JWT secret required for credential encryption")
	}

	for i := range c.SNMP.V3Credentials {
		cred := &c.SNMP.V3Credentials[i]

		// Encrypt AuthPassword if not already encrypted
		if cred.AuthPassword != "" && !IsEncrypted(cred.AuthPassword) {
			encrypted, err := EncryptCredential(cred.AuthPassword, c.Auth.JWTSecret)
			if err != nil {
				return fmt.Errorf("failed to encrypt auth password for %s: %w", cred.Name, err)
			}
			cred.AuthPassword = encrypted
		}

		// Encrypt PrivPassword if not already encrypted
		if cred.PrivPassword != "" && !IsEncrypted(cred.PrivPassword) {
			encrypted, err := EncryptCredential(cred.PrivPassword, c.Auth.JWTSecret)
			if err != nil {
				return fmt.Errorf("failed to encrypt priv password for %s: %w", cred.Name, err)
			}
			cred.PrivPassword = encrypted
		}
	}

	return nil
}

// DecryptSNMPPassword decrypts an SNMP password for use (fixes #518).
func (c *Config) DecryptSNMPPassword(encrypted string) (string, error) {
	if c.Auth.JWTSecret == "" {
		return "", errors.New("JWT secret required for credential decryption")
	}

	return DecryptCredential(encrypted, c.Auth.JWTSecret)
}
