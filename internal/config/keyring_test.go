package config_test

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// versionedPrefix is the on-the-wire prefix for DEK-encrypted credentials.
const versionedPrefix = "enc:v1:"

func newKeyedConfig(t *testing.T, dir string) *config.Config {
	t.Helper()
	cfg := &config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret-only-for-legacy-decrypt"},
	}
	if err := cfg.InitCredentialKeyring(dir); err != nil {
		t.Fatalf("InitCredentialKeyring failed: %v", err)
	}
	return cfg
}

func TestCredentialKeyringRoundtrip(t *testing.T) {
	cfg := newKeyedConfig(t, t.TempDir())

	enc, err := cfg.EncryptCredentialValue("s3cr3t-snmp-pass")
	if err != nil {
		t.Fatalf("EncryptCredentialValue failed: %v", err)
	}
	if !strings.HasPrefix(enc, versionedPrefix) {
		t.Fatalf("ciphertext should carry versioned prefix %q, got %q", versionedPrefix, enc)
	}

	got, err := cfg.DecryptSNMPPassword(enc)
	if err != nil {
		t.Fatalf("DecryptSNMPPassword failed: %v", err)
	}
	if got != "s3cr3t-snmp-pass" {
		t.Fatalf("roundtrip mismatch: got %q", got)
	}
}

// TestCredentialKeyringPersistsAcrossReload proves the DEK is persisted to the
// key file (not regenerated) so ciphertext survives a process restart.
func TestCredentialKeyringPersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	cfg1 := newKeyedConfig(t, dir)

	enc, err := cfg1.EncryptCredentialValue("persist-me")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// A fresh Config pointed at the same dir must decrypt the prior ciphertext.
	cfg2 := newKeyedConfig(t, dir)
	got, err := cfg2.DecryptSNMPPassword(enc)
	if err != nil {
		t.Fatalf("decrypt after reload failed: %v", err)
	}
	if got != "persist-me" {
		t.Fatalf("reload roundtrip mismatch: got %q", got)
	}
}

// TestCredentialKeyFilePerms verifies the key file is created 0600.
func TestCredentialKeyFilePerms(t *testing.T) {
	dir := t.TempDir()
	_ = newKeyedConfig(t, dir)

	info, err := os.Stat(filepath.Join(dir, "credential.key"))
	if err != nil {
		t.Fatalf("key file not created: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("key file perms = %o, want 0600", perm)
	}
}

// TestCredentialEncryptionDecoupledFromJWT is the core ADR-0015 guarantee:
// rotating Auth.JWTSecret must NOT make DEK-encrypted credentials undecryptable.
func TestCredentialEncryptionDecoupledFromJWT(t *testing.T) {
	dir := t.TempDir()
	cfg := newKeyedConfig(t, dir)

	enc, err := cfg.EncryptCredentialValue("decoupled")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Rotate the JWT signing secret — a routine auth operation.
	cfg.Auth.JWTSecret = "a-completely-different-rotated-jwt-secret"

	got, err := cfg.DecryptSNMPPassword(enc)
	if err != nil {
		t.Fatalf("decrypt after JWT rotation failed (still coupled?): %v", err)
	}
	if got != "decoupled" {
		t.Fatalf("decrypt after JWT rotation mismatch: got %q", got)
	}
}

// TestCredentialKeyEnvOverride verifies SEED_CREDENTIAL_KEY supplies the master
// and is never written to the key file.
func TestCredentialKeyEnvOverride(t *testing.T) {
	dir := t.TempDir()

	master := make([]byte, 32)
	if _, err := rand.Read(master); err != nil {
		t.Fatalf("rand: %v", err)
	}
	t.Setenv("SEED_CREDENTIAL_KEY", base64.StdEncoding.EncodeToString(master))

	cfg := newKeyedConfig(t, dir)
	enc, err := cfg.EncryptCredentialValue("byo-kms")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// The persisted key file must NOT contain the env-supplied master.
	raw, err := os.ReadFile(filepath.Join(dir, "credential.key"))
	if err != nil {
		t.Fatalf("read key file: %v", err)
	}
	var file struct {
		Master string `json:"master"`
	}
	if unmarshalErr := json.Unmarshal(raw, &file); unmarshalErr != nil {
		t.Fatalf("parse key file: %v", unmarshalErr)
	}
	if file.Master != "" {
		t.Fatalf("env-supplied master must not be written to disk, found %q", file.Master)
	}

	// Reload from env + on-disk salts must still decrypt.
	cfg2 := newKeyedConfig(t, dir)
	got, err := cfg2.DecryptSNMPPassword(enc)
	if err != nil {
		t.Fatalf("decrypt after env reload failed: %v", err)
	}
	if got != "byo-kms" {
		t.Fatalf("env reload roundtrip mismatch: got %q", got)
	}
}

// TestLegacyV0Migration proves a legacy JWT-derived ciphertext is transparently
// re-encrypted to the versioned DEK format and remains decryptable throughout.
func TestLegacyV0Migration(t *testing.T) {
	dir := t.TempDir()
	const jwt = "legacy-jwt-secret-used-for-v0-credentials"
	const plain = "legacy-priv-pass"

	// Craft a legacy v0 ciphertext exactly as the old code did.
	v0, err := config.EncryptCredential(plain, jwt)
	if err != nil {
		t.Fatalf("legacy encrypt failed: %v", err)
	}
	if strings.HasPrefix(v0, versionedPrefix) {
		t.Fatalf("legacy ciphertext should be unversioned, got %q", v0)
	}
	if !config.IsLegacyEncrypted(v0) {
		t.Fatalf("v0 ciphertext should be detected as legacy-encrypted")
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{JWTSecret: jwt},
		SNMP: config.SNMPConfig{
			V3Credentials: []config.SNMPv3Credential{
				{Name: "legacy", PrivPassword: v0},
			},
		},
	}
	if initErr := cfg.InitCredentialKeyring(dir); initErr != nil {
		t.Fatalf("InitCredentialKeyring failed: %v", initErr)
	}

	// Migration must still read v0 directly (read-only legacy path).
	pre, err := cfg.DecryptSNMPPassword(v0)
	if err != nil {
		t.Fatalf("legacy decrypt failed: %v", err)
	}
	if pre != plain {
		t.Fatalf("legacy decrypt mismatch: got %q", pre)
	}

	// EncryptSNMPCredentials re-encrypts v0 -> v1.
	if encErr := cfg.EncryptSNMPCredentials(); encErr != nil {
		t.Fatalf("EncryptSNMPCredentials failed: %v", encErr)
	}
	migrated := cfg.SNMP.V3Credentials[0].PrivPassword
	if !strings.HasPrefix(migrated, versionedPrefix) {
		t.Fatalf("credential should be migrated to versioned format, got %q", migrated)
	}

	// Decryptable after migration.
	post, err := cfg.DecryptSNMPPassword(migrated)
	if err != nil {
		t.Fatalf("post-migration decrypt failed: %v", err)
	}
	if post != plain {
		t.Fatalf("post-migration mismatch: got %q", post)
	}
}
