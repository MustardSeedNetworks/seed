package config

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// encryptedPrefix identifies encrypted values in config.
	encryptedPrefix = "enc:"
)

// ErrInvalidCiphertext is returned when decryption fails due to invalid input.
var ErrInvalidCiphertext = errors.New("invalid ciphertext")

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

// CredentialKeyring exposes the credential DEK keyring so components that
// decrypt stored secrets at use time — the SNMP poller's credential resolver —
// share the one keyring rather than each loading their own.
func (c *Config) CredentialKeyring() (*Keyring, error) {
	return c.ensureKeyring()
}
