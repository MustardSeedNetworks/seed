package api_test

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/api"
)

// TestEnsureSelfSignedCertIsCAEligible verifies the generated self-signed
// cert carries IsCA=true and KeyUsageCertSign so it can be installed into
// the OS trust store by `seed install-ca`.
func TestEnsureSelfSignedCertIsCAEligible(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	// ensureSelfSignedCert writes to "certs/" relative to CWD. Use a temp
	// directory so the test does not litter the repo and runs hermetically.
	// t.Chdir restores the original directory automatically.
	dir := t.TempDir()
	t.Chdir(dir)

	certFile, keyFile, err := server.EnsureSelfSignedCert()
	if err != nil {
		t.Fatalf("EnsureSelfSignedCert: %v", err)
	}
	if certFile == "" || keyFile == "" {
		t.Fatalf("empty paths: cert=%q key=%q", certFile, keyFile)
	}

	pemBytes, readErr := os.ReadFile(filepath.Join(dir, certFile))
	if readErr != nil {
		t.Fatalf("read generated cert: %v", readErr)
	}
	block, _ := pem.Decode(pemBytes)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("PEM decode: block=%v", block)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}

	if !cert.IsCA {
		t.Error("expected cert.IsCA = true so the cert can act as a root")
	}
	if !cert.BasicConstraintsValid {
		t.Error("expected BasicConstraintsValid = true")
	}
	if cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		t.Error("expected KeyUsageCertSign so the cert can sign certificates as a root")
	}
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("expected KeyUsageDigitalSignature to remain set for TLS handshake")
	}
	if cert.KeyUsage&x509.KeyUsageKeyEncipherment == 0 {
		t.Error("expected KeyUsageKeyEncipherment to remain set for TLS RSA key exchange")
	}
	// Self-signed: Subject must equal Issuer (DER-encoded comparison).
	if string(cert.RawSubject) != string(cert.RawIssuer) {
		t.Error("expected Subject == Issuer for a self-signed root")
	}
}
