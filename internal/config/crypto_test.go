package config

import (
	"strings"
	"testing"
)

func TestEncryptDecryptCredential(t *testing.T) {
	masterSecret := "test-secret-key-for-encryption"
	plaintext := "mySecretPassword123!"

	// Test encryption
	encrypted, err := EncryptCredential(plaintext, masterSecret)
	if err != nil {
		t.Fatalf("EncryptCredential failed: %v", err)
	}

	// Verify encrypted value is different from plaintext
	if encrypted == plaintext {
		t.Error("Encrypted value should differ from plaintext")
	}

	// Verify encrypted value has prefix
	if !strings.HasPrefix(encrypted, encryptedPrefix) {
		t.Errorf("Encrypted value should have prefix %q, got %q", encryptedPrefix, encrypted)
	}

	// Test decryption
	decrypted, err := DecryptCredential(encrypted, masterSecret)
	if err != nil {
		t.Fatalf("DecryptCredential failed: %v", err)
	}

	// Verify decrypted matches original
	if decrypted != plaintext {
		t.Errorf("Decrypted value %q doesn't match original %q", decrypted, plaintext)
	}
}

func TestEncryptDecryptEmptyString(t *testing.T) {
	masterSecret := "test-secret-key"

	encrypted, err := EncryptCredential("", masterSecret)
	if err != nil {
		t.Fatalf("EncryptCredential with empty string failed: %v", err)
	}

	if encrypted != "" {
		t.Errorf("Empty string should encrypt to empty string, got %q", encrypted)
	}

	decrypted, err := DecryptCredential("", masterSecret)
	if err != nil {
		t.Fatalf("DecryptCredential with empty string failed: %v", err)
	}

	if decrypted != "" {
		t.Errorf("Empty string should decrypt to empty string, got %q", decrypted)
	}
}

func TestDecryptPlaintextBackwardCompatibility(t *testing.T) {
	masterSecret := "test-secret-key"
	plaintext := "oldPlaintextPassword"

	// Decrypting plaintext (no prefix) should return it as-is
	decrypted, err := DecryptCredential(plaintext, masterSecret)
	if err != nil {
		t.Fatalf("DecryptCredential with plaintext failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Plaintext should be returned as-is, got %q", decrypted)
	}
}

func TestEncryptAlreadyEncrypted(t *testing.T) {
	masterSecret := "test-secret-key"
	plaintext := "password"

	// First encryption
	encrypted1, err := EncryptCredential(plaintext, masterSecret)
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	// Second encryption of already encrypted value
	encrypted2, err := EncryptCredential(encrypted1, masterSecret)
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	// Should return same value (idempotent)
	if encrypted2 != encrypted1 {
		t.Error("Encrypting an already encrypted value should be idempotent")
	}
}

func TestDecryptInvalidCiphertext(t *testing.T) {
	masterSecret := "test-secret-key"

	testCases := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", encryptedPrefix + "not-valid-base64!!!"},
		{"too short", encryptedPrefix + "YWJj"}, // "abc" in base64, too short for nonce
		{"tampered", encryptedPrefix + "dGFtcGVyZWRkYXRhdGFtcGVyZWRkYXRh"}, // valid base64 but invalid ciphertext
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecryptCredential(tc.ciphertext, masterSecret)
			if err == nil {
				t.Error("Expected error for invalid ciphertext, got nil")
			}
		})
	}
}

func TestEncryptDecryptWithDifferentSecrets(t *testing.T) {
	plaintext := "password"
	secret1 := "secret1"
	secret2 := "secret2"

	// Encrypt with secret1
	encrypted, err := EncryptCredential(plaintext, secret1)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// Try to decrypt with secret2 (should fail)
	_, err = DecryptCredential(encrypted, secret2)
	if err == nil {
		t.Error("Expected error when decrypting with wrong secret, got nil")
	}
}

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
			result := IsEncrypted(tc.value)
			if result != tc.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tc.value, result, tc.expected)
			}
		})
	}
}

func TestEncryptSNMPCredentials(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			JWTSecret: "test-jwt-secret-for-encryption",
		},
		SNMP: SNMPConfig{
			V3Credentials: []SNMPv3Credential{
				{
					Name:         "test-cred",
					AuthPassword: "authPass123",
					PrivPassword: "privPass456",
				},
			},
		},
	}

	// Encrypt credentials
	err := cfg.EncryptSNMPCredentials()
	if err != nil {
		t.Fatalf("EncryptSNMPCredentials failed: %v", err)
	}

	// Verify passwords are encrypted
	cred := cfg.SNMP.V3Credentials[0]
	if !IsEncrypted(cred.AuthPassword) {
		t.Error("AuthPassword should be encrypted")
	}
	if !IsEncrypted(cred.PrivPassword) {
		t.Error("PrivPassword should be encrypted")
	}

	// Verify passwords are not the original values
	if cred.AuthPassword == "authPass123" {
		t.Error("AuthPassword should not be plaintext")
	}
	if cred.PrivPassword == "privPass456" {
		t.Error("PrivPassword should not be plaintext")
	}

	// Test decryption
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

func TestEncryptSNMPCredentialsIdempotent(t *testing.T) {
	cfg := &Config{
		Auth: AuthConfig{
			JWTSecret: "test-jwt-secret",
		},
		SNMP: SNMPConfig{
			V3Credentials: []SNMPv3Credential{
				{
					Name:         "test",
					AuthPassword: "password",
				},
			},
		},
	}

	// First encryption
	err := cfg.EncryptSNMPCredentials()
	if err != nil {
		t.Fatalf("First encryption failed: %v", err)
	}

	firstEncrypted := cfg.SNMP.V3Credentials[0].AuthPassword

	// Second encryption (should be idempotent)
	err = cfg.EncryptSNMPCredentials()
	if err != nil {
		t.Fatalf("Second encryption failed: %v", err)
	}

	secondEncrypted := cfg.SNMP.V3Credentials[0].AuthPassword

	if firstEncrypted != secondEncrypted {
		t.Error("EncryptSNMPCredentials should be idempotent")
	}
}
