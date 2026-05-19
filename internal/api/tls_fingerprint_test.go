package api_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/api"
)

// makeTestCertPEM generates a self-signed X.509 certificate for testing
// fingerprint logic. Returns the PEM-encoded certificate and the raw DER
// bytes (used to derive the expected SHA-256 fingerprint).
func makeTestCertPEM(t *testing.T) ([]byte, []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(42),
		Subject:               pkix.Name{CommonName: "seed-test-fp"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, certErr := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if certErr != nil {
		t.Fatalf("create certificate: %v", certErr)
	}
	var buf bytes.Buffer
	if encErr := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); encErr != nil {
		t.Fatalf("pem encode: %v", encErr)
	}
	return buf.Bytes(), der
}

func TestFingerprintFromPEM(t *testing.T) {
	t.Parallel()

	pemBytes, der := makeTestCertPEM(t)

	got, err := api.ExportFingerprintFromPEM(pemBytes)
	if err != nil {
		t.Fatalf("ExportFingerprintFromPEM: unexpected error: %v", err)
	}

	sum := sha256.Sum256(der)
	want := api.ExportFormatFingerprint(sum[:])
	if got != want {
		t.Errorf("fingerprint mismatch:\n  got  %s\n  want %s", got, want)
	}

	// Format validation: 32 uppercase hex pairs separated by colons.
	parts := strings.Split(got, ":")
	if len(parts) != sha256.Size {
		t.Errorf("expected %d colon-separated pairs, got %d (%q)", sha256.Size, len(parts), got)
	}
	for i, p := range parts {
		if len(p) != 2 {
			t.Errorf("pair %d has length %d, want 2 (%q)", i, len(p), p)
			continue
		}
		for _, r := range p {
			isUpperHex := (r >= '0' && r <= '9') || (r >= 'A' && r <= 'F')
			if !isUpperHex {
				t.Errorf("pair %d %q contains non-uppercase-hex rune %q", i, p, r)
				break
			}
		}
	}
}

func TestFingerprintFromPEM_NoCertificateBlock(t *testing.T) {
	t.Parallel()

	_, err := api.ExportFingerprintFromPEM([]byte("not a pem\n"))
	if !errors.Is(err, api.ErrNoCertificateBlock) {
		t.Errorf("expected api.ErrNoCertificateBlock, got %v", err)
	}
}

func TestComputeCertFingerprint_EmptyPath(t *testing.T) {
	t.Parallel()

	_, err := api.ExportComputeCertFingerprint("")
	if !errors.Is(err, api.ErrEmptyCertPath) {
		t.Errorf("expected api.ErrEmptyCertPath, got %v", err)
	}
}

func TestComputeCertFingerprint_ReadsFile(t *testing.T) {
	t.Parallel()

	pemBytes, der := makeTestCertPEM(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "server.crt")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write fixture cert: %v", err)
	}

	got, err := api.ExportComputeCertFingerprint(path)
	if err != nil {
		t.Fatalf("ExportComputeCertFingerprint: %v", err)
	}
	sum := sha256.Sum256(der)
	want := api.ExportFormatFingerprint(sum[:])
	if got != want {
		t.Errorf("fingerprint mismatch:\n  got  %s\n  want %s", got, want)
	}
}

func TestTLSFingerprintCache_CachesAfterFirstRead(t *testing.T) {
	t.Parallel()

	pemBytes, _ := makeTestCertPEM(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "server.crt")
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatalf("write fixture cert: %v", err)
	}

	cache := &api.TLSFingerprintCache{}

	fp1, err := cache.Get(path)
	if err != nil {
		t.Fatalf("first Get: %v", err)
	}
	if fp1 == "" {
		t.Fatal("expected non-empty fingerprint on first Get")
	}

	// Remove the file. A second Get must still succeed because the cache is
	// populated; it must not re-read the disk.
	if rmErr := os.Remove(path); rmErr != nil {
		t.Fatalf("remove file: %v", rmErr)
	}
	fp2, err := cache.Get(path)
	if err != nil {
		t.Fatalf("second Get after file removal: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("cached fingerprint changed: %s vs %s", fp1, fp2)
	}
}

func TestTLSFingerprintCache_EmptyPathReturnsEmpty(t *testing.T) {
	t.Parallel()

	cache := &api.TLSFingerprintCache{}
	fp, err := cache.Get("")
	if err != nil {
		t.Fatalf("Get(\"\"): %v", err)
	}
	if fp != "" {
		t.Errorf("expected empty fingerprint for empty path, got %q", fp)
	}
}

func TestFormatFingerprint(t *testing.T) {
	t.Parallel()

	in := []byte{0xaa, 0xbb, 0xcc, 0x01, 0x23, 0x45}
	got := api.ExportFormatFingerprint(in)
	const want = "AA:BB:CC:01:23:45"
	if got != want {
		t.Errorf("ExportFormatFingerprint(%x) = %q, want %q", in, got, want)
	}
}
