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

// TestValidateRejectsPlaintextSNMPCredential verifies the startup gate:
// Config.Validate() (via validateSNMPCredentials) refuses a config whose SNMP v3
// credential is plaintext rather than versioned DEK ciphertext, with a message
// pointing operators to the API/CLI. This is the enforcement point that stops the
// daemon from running with a plaintext secret at rest.
func TestValidateRejectsPlaintextSNMPCredential(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SNMP.V3Credentials = []config.SNMPv3Credential{
		{Name: "plain-cred", Username: "snmpuser", AuthProtocol: "SHA256", AuthPassword: "plaintext-secret"},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate should reject a plaintext SNMP v3 credential, got nil error")
	}
	if !strings.Contains(err.Error(), "must be set via the API/CLI") {
		t.Errorf("validation error should point operators to the API/CLI, got: %v", err)
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
