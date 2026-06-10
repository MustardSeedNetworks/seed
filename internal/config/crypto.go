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

// ErrPlaintextCredential is returned when a credential value is found in
// plaintext (not versioned DEK ciphertext) at read time. Credentials must be
// set via the API or CLI so they are encrypted at rest; hand-editing plaintext
// into the config is rejected.
var ErrPlaintextCredential = errors.New("plaintext credential value is not accepted; " +
	"credentials must be set via the API/CLI so they are encrypted at rest")

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
// version (ADR-0015). This is the one legitimate path from operator-supplied
// plaintext to stored ciphertext; it is called by the API and CLI handlers when
// an operator sets a credential. All other credential read paths expect versioned
// DEK ciphertext and reject anything else.
func (c *Config) EncryptCredentialValue(plaintext string) (string, error) {
	kr, err := c.ensureKeyring()
	if err != nil {
		return "", err
	}
	return kr.EncryptValue(plaintext)
}

// reEncryptCredential ensures a stored credential is in the active versioned DEK
// format. Already-versioned values are returned unchanged (idempotent). Empty
// values are returned unchanged. Any other value — including bare plaintext or
// the legacy v0/JWT-derived format — is an error: credentials must be set via
// the API/CLI (EncryptCredentialValue), not hand-edited or silently coerced.
func (c *Config) reEncryptCredential(value string) (string, error) {
	if value == "" || isVersionedCiphertext(value) {
		return value, nil
	}
	// Neither empty nor versioned — this is plaintext or legacy ciphertext.
	// Reject: the caller must set the credential via the API/CLI.
	if IsEncrypted(value) {
		// Legacy v0 (JWT-derived) unversioned ciphertext.
		return "", fmt.Errorf("SNMP v3 credential: legacy v0/JWT-derived ciphertext is no longer "+
			"supported; re-set the credential via the API/CLI so it is re-encrypted at rest: %w",
			ErrPlaintextCredential)
	}
	return "", fmt.Errorf("SNMP v3 credential: %w", ErrPlaintextCredential)
}

// EncryptSNMPCredentials re-encrypts all SNMP v3 credentials that are already
// in versioned DEK format (idempotent) and returns an error for any credential
// that is still in plaintext or the legacy v0/JWT-derived format. Callers must
// set such credentials via the API/CLI (EncryptCredentialValue) before calling
// this function.
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

// DecryptSNMPPassword decrypts an SNMP password for use (ADR-0015).
// Only versioned DEK ciphertext ("enc:v<N>:...") is accepted. Empty values
// return empty. Plaintext or legacy v0/JWT-derived ciphertext is rejected:
// credentials must be set via the API/CLI so they are encrypted at rest.
func (c *Config) DecryptSNMPPassword(encrypted string) (string, error) {
	if encrypted == "" {
		return "", nil
	}
	if !isVersionedCiphertext(encrypted) {
		if IsEncrypted(encrypted) {
			// Legacy v0 (JWT-derived) unversioned "enc:..." format.
			return "", fmt.Errorf("SNMP v3 credential: legacy v0/JWT-derived ciphertext is no longer "+
				"supported; re-set the credential via the API/CLI so it is re-encrypted at rest: %w",
				ErrPlaintextCredential)
		}
		// Bare plaintext.
		return "", fmt.Errorf("SNMP v3 credential: %w", ErrPlaintextCredential)
	}

	kr, err := c.ensureKeyring()
	if err != nil {
		return "", err
	}
	return kr.DecryptValue(encrypted)
}
