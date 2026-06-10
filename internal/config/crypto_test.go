package config_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

func TestIsEncrypted(t *testing.T) {
	testCases := []struct {
		value    string
		expected bool
	}{
		{"enc:base64data", true},
		{"plaintext", false},
		{"", false},
		{"enc:", true},
		{"ENC:base64", false}, // case-sensitive
	}

	for _, tc := range testCases {
		t.Run(tc.value, func(t *testing.T) {
			result := config.IsEncrypted(tc.value)
			if result != tc.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

// TestEncryptSNMPCredentials verifies that already-versioned DEK ciphertext is
// left unchanged (idempotent) by EncryptSNMPCredentials.
func TestEncryptSNMPCredentials(t *testing.T) {
	cfg := newKeyedConfigForCrypto(t)

	// Encrypt with the API path first.
	authEnc, err := cfg.EncryptCredentialValue("authPass123")
	if err != nil {
		t.Fatalf("EncryptCredentialValue auth: %v", err)
	}
	privEnc, err := cfg.EncryptCredentialValue("privPass456")
	if err != nil {
		t.Fatalf("EncryptCredentialValue priv: %v", err)
	}

	cfg.SNMP.V3Credentials = []config.SNMPv3Credential{
		{Name: "test-cred", AuthPassword: authEnc, PrivPassword: privEnc},
	}

	err = cfg.EncryptSNMPCredentials()
	if err != nil {
		t.Fatalf("EncryptSNMPCredentials failed: %v", err)
	}

	cred := cfg.SNMP.V3Credentials[0]
	if !config.IsEncrypted(cred.AuthPassword) {
		t.Error("AuthPassword should be encrypted")
	}
	if !config.IsEncrypted(cred.PrivPassword) {
		t.Error("PrivPassword should be encrypted")
	}

	authPass, err := cfg.DecryptSNMPPassword(cred.AuthPassword)
	if err != nil {
		t.Fatalf("Failed to decrypt auth password: %v", err)
	}
	if authPass != "authPass123" {
		t.Errorf("Decrypted auth password = %q, want %q", authPass, "authPass123")
	}

	privPass, err := cfg.DecryptSNMPPassword(cred.PrivPassword)
	if err != nil {
		t.Fatalf("Failed to decrypt priv password: %v", err)
	}
	if privPass != "privPass456" {
		t.Errorf("Decrypted priv password = %q, want %q", privPass, "privPass456")
	}
}

// TestEncryptSNMPCredentialsIdempotent verifies that calling EncryptSNMPCredentials
// twice on already-versioned ciphertext does not change the stored value.
func TestEncryptSNMPCredentialsIdempotent(t *testing.T) {
	cfg := newKeyedConfigForCrypto(t)

	authEnc, err := cfg.EncryptCredentialValue("password")
	if err != nil {
		t.Fatalf("EncryptCredentialValue: %v", err)
	}
	cfg.SNMP.V3Credentials = []config.SNMPv3Credential{
		{Name: "test", AuthPassword: authEnc},
	}

	err = cfg.EncryptSNMPCredentials()
	if err != nil {
		t.Fatalf("First EncryptSNMPCredentials failed: %v", err)
	}
	firstEncrypted := cfg.SNMP.V3Credentials[0].AuthPassword

	err = cfg.EncryptSNMPCredentials()
	if err != nil {
		t.Fatalf("Second EncryptSNMPCredentials failed: %v", err)
	}
	secondEncrypted := cfg.SNMP.V3Credentials[0].AuthPassword

	if firstEncrypted != secondEncrypted {
		t.Error("EncryptSNMPCredentials should be idempotent for already-encrypted values")
	}
}

// TestEncryptSNMPCredentialsRejectsPlaintext verifies that EncryptSNMPCredentials
// returns an error when a credential is still in plaintext. Credentials must be
// set via the API/CLI; silent on-save encryption is no longer accepted.
func TestEncryptSNMPCredentialsRejectsPlaintext(t *testing.T) {
	cfg := newKeyedConfigForCrypto(t)
	cfg.SNMP.V3Credentials = []config.SNMPv3Credential{
		{Name: "bad-cred", AuthPassword: "plaintext-password"},
	}

	err := cfg.EncryptSNMPCredentials()
	if err == nil {
		t.Fatal("EncryptSNMPCredentials should reject plaintext credential, got nil error")
	}
	if !errors.Is(err, config.ErrPlaintextCredential) {
		t.Errorf("expected ErrPlaintextCredential, got: %v", err)
	}
}

// TestDecryptSNMPPasswordRejectsPlaintext verifies that DecryptSNMPPassword
// returns an error for a non-encrypted value.
func TestDecryptSNMPPasswordRejectsPlaintext(t *testing.T) {
	cfg := newKeyedConfigForCrypto(t)

	_, err := cfg.DecryptSNMPPassword("plaintext-value")
	if err == nil {
		t.Fatal("DecryptSNMPPassword should reject plaintext, got nil error")
	}
	if !errors.Is(err, config.ErrPlaintextCredential) {
		t.Errorf("expected ErrPlaintextCredential, got: %v", err)
	}
}

// TestDecryptSNMPPasswordRejectsLegacyCiphertext verifies that the legacy
// v0/JWT-derived "enc:..." format (unversioned) is rejected by DecryptSNMPPassword.
// The v0 read path has been removed; operators must re-set such credentials via
// the API/CLI.
func TestDecryptSNMPPasswordRejectsLegacyCiphertext(t *testing.T) {
	cfg := newKeyedConfigForCrypto(t)

	// Craft a value that looks like legacy v0 ciphertext: has "enc:" prefix
	// but no version segment ("enc:v<N>:").
	legacyLike := config.EncryptedPrefix + "dGhpcyBpcyBmYWtlIGxlZ2FjeSBkYXRh"

	if !config.IsEncrypted(legacyLike) {
		t.Fatalf("test setup: value should be detected as encrypted: %q", legacyLike)
	}

	_, err := cfg.DecryptSNMPPassword(legacyLike)
	if err == nil {
		t.Fatal("DecryptSNMPPassword should reject legacy v0 ciphertext, got nil error")
	}
	if !errors.Is(err, config.ErrPlaintextCredential) {
		t.Errorf("expected ErrPlaintextCredential, got: %v", err)
	}
	if !strings.Contains(err.Error(), "legacy v0") {
		t.Errorf("error message should mention legacy v0, got: %v", err)
	}
}

// newKeyedConfigForCrypto creates a minimal Config with an ephemeral keyring for
// crypto_test.go tests (no JWTSecret dependency).
func newKeyedConfigForCrypto(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	if err := cfg.InitCredentialKeyring(t.TempDir()); err != nil {
		t.Fatalf("InitCredentialKeyring: %v", err)
	}
	return cfg
}
