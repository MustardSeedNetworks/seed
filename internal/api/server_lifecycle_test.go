package api_test

import (
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/krisarmstrong/seed/internal/api"
)

// TestHTTPRedirectUsesStatus308 verifies that the HTTP→HTTPS redirect
// returns 308 Permanent Redirect (not 301 Moved Permanently) so the HTTP
// method is preserved across the redirect.
func TestHTTPRedirectUsesStatus308(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	// HTTPS listening port from the test config informs the Location header.
	cfg := server.Config()
	cfg.Server.Port = 8443
	server.SetConfig(cfg)

	handler := server.HTTPToHTTPSRedirectHandler()

	cases := []struct {
		method string
	}{
		{http.MethodGet},
		{http.MethodPost},
		{http.MethodPut},
		{http.MethodDelete},
		{http.MethodPatch},
	}
	for _, tc := range cases {
		t.Run(tc.method, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "http://localhost/api/anything?x=1", http.NoBody)
			req.Host = "localhost"
			req.RequestURI = "/api/anything?x=1"
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusPermanentRedirect {
				t.Errorf("method=%s: status = %d, want %d (308)",
					tc.method, rec.Code, http.StatusPermanentRedirect)
			}
			loc := rec.Header().Get("Location")
			want := "https://localhost:8443/api/anything?x=1"
			if loc != want {
				t.Errorf("method=%s: Location = %q, want %q", tc.method, loc, want)
			}
		})
	}
}

// TestHTTPRedirectStripsPortForStandardHTTPS verifies the Location header
// drops the port when HTTPS listens on the standard 443.
func TestHTTPRedirectStripsPortForStandardHTTPS(t *testing.T) {
	server := api.NewTestServer()
	defer server.Close()

	cfg := server.Config()
	cfg.Server.Port = 443
	server.SetConfig(cfg)

	handler := server.HTTPToHTTPSRedirectHandler()

	req := httptest.NewRequest(http.MethodGet, "http://example.com:80/path", http.NoBody)
	req.Host = "example.com:80"
	req.RequestURI = "/path"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusPermanentRedirect {
		t.Errorf("status = %d, want 308", rec.Code)
	}
	const want = "https://example.com/path"
	if got := rec.Header().Get("Location"); got != want {
		t.Errorf("Location = %q, want %q", got, want)
	}
}

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
