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
	// encryptedPrefix identifies encrypted values in config.
	encryptedPrefix = "enc:"
)

// ErrInvalidCiphertext is returned when decryption fails due to invalid input.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

// deriveKey derives a 32-byte AES-256 key from the master secret using SHA-256.
// This provides a consistent key for encryption/decryption.
func deriveKey(masterSecret string) []byte {
	hash := sha256.Sum256([]byte(masterSecret))
	return hash[:]
}

// EncryptCredential encrypts a credential string using AES-256-GCM (fixes #518).
// The encrypted value is prefixed with "enc:" to identify it as encrypted
// Format: enc:base64(nonce||ciphertext||tag).
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
	if _, nonceErr := io.ReadFull(rand.Reader, nonce); nonceErr != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", nonceErr)
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
		return "", fmt.Errorf("%w: invalid base64: %w", ErrInvalidCiphertext, err)
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
		return "", fmt.Errorf("%w: authentication failed: %w", ErrInvalidCiphertext, err)
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a credential value is encrypted.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encryptedPrefix)
}

// InitCredentialKeyring loads or creates the credential DEK keyring in dir
// (ADR-0015). It must be called once during startup, before any credential
// encryption or decryption, so ciphertext is persisted and survives restart.
func (c *Config) InitCredentialKeyring(dir string) error {
	kr, err := LoadOrCreateKeyring(dir)
	if err != nil {
		return err
	}
	c.credentialKeyring = kr
	return nil
}

// ensureKeyring returns the configured keyring, lazily creating a non-persistent
// ephemeral one if InitCredentialKeyring was never called. Production startup
// always initialises a persistent keyring; the ephemeral fallback exists only
// so unit tests and incidental code paths round-trip within a process.
func (c *Config) ensureKeyring() (*Keyring, error) {
	if c.credentialKeyring != nil {
		return c.credentialKeyring, nil
	}
	kr, err := newEphemeralKeyring()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKeyringUnavailable, err)
	}
	c.credentialKeyring = kr
	return kr, nil
}

// EncryptCredentialValue encrypts a single credential value with the active DEK
// version (ADR-0015). It replaces the JWT-derived EncryptCredential call sites.
func (c *Config) EncryptCredentialValue(plaintext string) (string, error) {
	kr, err := c.ensureKeyring()
	if err != nil {
		return "", err
	}
	return kr.EncryptValue(plaintext)
}

// reEncryptCredential normalises a credential value to the active DEK version.
// Plaintext is encrypted; legacy v0 (JWT-derived) ciphertext is decrypted with
// Auth.JWTSecret and re-encrypted with the DEK; already-versioned values are
// returned unchanged. Legacy values are left intact (not lost) when JWTSecret is
// unavailable.
func (c *Config) reEncryptCredential(value string) (string, error) {
	if value == "" || isVersionedCiphertext(value) {
		return value, nil
	}

	plaintext := value
	if IsLegacyEncrypted(value) {
		if c.Auth.JWTSecret == "" {
			return value, nil // cannot migrate without the legacy key; keep intact
		}
		decrypted, err := DecryptCredential(value, c.Auth.JWTSecret)
		if err != nil {
			return value, fmt.Errorf("decrypt legacy credential: %w", err)
		}
		plaintext = decrypted
	}

	return c.EncryptCredentialValue(plaintext)
}

// EncryptSNMPCredentials encrypts (and migrates legacy v0 values for) all SNMP
// v3 credentials with the DEK keyring (fixes #518; ADR-0015).
func (c *Config) EncryptSNMPCredentials() error {
	for i := range c.SNMP.V3Credentials {
		cred := &c.SNMP.V3Credentials[i]

		authPw, err := c.reEncryptCredential(cred.AuthPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt auth password for %s: %w", cred.Name, err)
		}
		cred.AuthPassword = authPw

		privPw, err := c.reEncryptCredential(cred.PrivPassword)
		if err != nil {
			return fmt.Errorf("failed to encrypt priv password for %s: %w", cred.Name, err)
		}
		cred.PrivPassword = privPw
	}

	return nil
}

// DecryptSNMPPassword decrypts an SNMP password for use (fixes #518; ADR-0015).
// Versioned ciphertext is decrypted with the DEK keyring; legacy unversioned
// ciphertext falls back to the JWT-derived key (read-only migration path).
func (c *Config) DecryptSNMPPassword(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	if !IsEncrypted(encrypted) {
		return encrypted, nil // plaintext, returned as-is for backward compatibility
	}

	if isVersionedCiphertext(encrypted) {
		kr, err := c.ensureKeyring()
		if err != nil {
			return "", err
		}
		return kr.DecryptValue(encrypted)
	}

	// Legacy v0 (JWT-derived) — read-only path retained for migration.
	if c.Auth.JWTSecret == "" {
		return "", errors.New("JWT secret required to decrypt legacy credential")
	}
	return DecryptCredential(encrypted, c.Auth.JWTSecret)
}
