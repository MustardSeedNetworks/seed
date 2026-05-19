package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeFixtureCert generates a self-signed PEM cert in a tempdir and
// returns the file path together with the expected SHA-256 fingerprint
// (uppercase colon-separated).
func writeFixtureCert(t *testing.T) (string, string) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(99),
		Subject:               pkix.Name{CommonName: "seed-install-ca-test"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	var buf bytes.Buffer
	if encErr := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); encErr != nil {
		t.Fatalf("pem encode: %v", encErr)
	}
	path := filepath.Join(t.TempDir(), "server.crt")
	if writeErr := os.WriteFile(path, buf.Bytes(), 0o600); writeErr != nil {
		t.Fatalf("write cert: %v", writeErr)
	}
	sum := sha256.Sum256(der)
	return path, formatColonHex(sum[:])
}

func TestInstallCACmdRegistered(t *testing.T) {
	state := newCLIState()
	initCommands(state)

	cmd, _, err := state.rootCmd.Find([]string{"install-ca"})
	if err != nil {
		t.Fatalf("install-ca subcommand not registered: %v", err)
	}
	if cmd.Name() != "install-ca" {
		t.Errorf("expected install-ca, got %q", cmd.Name())
	}

	// Required flags must exist.
	for _, name := range []string{"cert", "uninstall", "print-fingerprint"} {
		if cmd.Flag(name) == nil {
			t.Errorf("install-ca missing --%s flag", name)
		}
	}
}

func TestInstallCAHelpRuns(t *testing.T) {
	state := newCLIState()
	initCommands(state)

	state.rootCmd.SetArgs([]string{"install-ca", "--help"})
	state.rootCmd.SetOut(&bytes.Buffer{})
	state.rootCmd.SetErr(&bytes.Buffer{})
	if err := state.rootCmd.Execute(); err != nil {
		t.Fatalf("install-ca --help returned error: %v", err)
	}
}

func TestResolveCertPath_OK(t *testing.T) {
	t.Parallel()
	path, _ := writeFixtureCert(t)
	got, err := resolveCertPath(path)
	if err != nil {
		t.Fatalf("resolveCertPath(%q): %v", path, err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("expected absolute path, got %q", got)
	}
}

func TestResolveCertPath_Missing(t *testing.T) {
	t.Parallel()
	_, err := resolveCertPath(filepath.Join(t.TempDir(), "missing.crt"))
	if err == nil {
		t.Fatal("expected error for missing path, got nil")
	}
	if !strings.Contains(err.Error(), "Start seed once") {
		t.Errorf("expected hint about starting seed, got: %v", err)
	}
}

func TestResolveCertPath_Empty(t *testing.T) {
	t.Parallel()
	_, err := resolveCertPath("")
	if err == nil {
		t.Fatal("expected error for empty path, got nil")
	}
}

func TestCertFingerprintMatchesSha256(t *testing.T) {
	t.Parallel()
	path, want := writeFixtureCert(t)
	got, err := certFingerprint(path)
	if err != nil {
		t.Fatalf("certFingerprint: %v", err)
	}
	if got != want {
		t.Errorf("certFingerprint mismatch:\n  got  %s\n  want %s", got, want)
	}
}

func TestFormatColonHexUppercase(t *testing.T) {
	t.Parallel()
	got := formatColonHex([]byte{0x00, 0x0f, 0xff, 0xab})
	const want = "00:0F:FF:AB"
	if got != want {
		t.Errorf("formatColonHex = %q, want %q", got, want)
	}
}

func TestInstallCAFlagsDefaults(t *testing.T) {
	flags := installCAFlags{}
	if flags.certPath != "" {
		t.Errorf("certPath default should be empty in zero value, got %q", flags.certPath)
	}
	if flags.uninstall {
		t.Error("uninstall default should be false")
	}
	if flags.printFingerprint {
		t.Error("printFingerprint default should be false")
	}
}

func TestInstallCATimeoutPositive(t *testing.T) {
	if installCATimeoutSeconds <= 0 {
		t.Error("installCATimeoutSeconds should be positive")
	}
}

func TestDefaultCertPathConstant(t *testing.T) {
	if defaultCertPath == "" {
		t.Error("defaultCertPath should be non-empty")
	}
	if !strings.HasSuffix(defaultCertPath, ".crt") {
		t.Errorf("defaultCertPath should end in .crt, got %q", defaultCertPath)
	}
}
