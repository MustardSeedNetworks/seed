package api

// tls_fingerprint.go computes and caches the SHA-256 fingerprint of the
// active TLS certificate. The fingerprint is exposed via /__version as
// `tlsFingerprint` so operators can verify the cert their browser sees
// matches the one the server is serving (matters for self-signed certs
// installed via `seed install-ca`).

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

// pemCertBlockType is the PEM block header used for X.509 certificates.
// The standard library does not expose a constant for this string; defining
// one here lets the cert-template encoder and the fingerprint decoder share
// a single source of truth.
const pemCertBlockType = "CERTIFICATE"

// errEmptyCertPath is returned when the configured cert path is empty
// (i.e. the server is running in HTTP mode and no cert exists).
var errEmptyCertPath = errors.New("no certificate configured")

// errNoCertificateBlock is returned when the PEM-encoded file does not
// contain a CERTIFICATE block.
var errNoCertificateBlock = errors.New("no CERTIFICATE block in PEM data")

// tlsFingerprintCache caches the active TLS certificate fingerprint so
// repeated /__version calls do not re-read disk. The cache key is the
// cert file path; this lets the cache stay valid across a server's
// lifetime (cert file is not rotated at runtime — a restart picks up a
// new fingerprint via cache miss on a different path or first access).
type tlsFingerprintCache struct {
	mu          sync.RWMutex
	path        string
	fingerprint string
}

// Get returns the fingerprint for the given cert file path, computing
// and caching it on first access. An empty path returns an empty
// fingerprint without error (HTTP mode is a supported configuration).
func (c *tlsFingerprintCache) Get(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	c.mu.RLock()
	if c.path == path && c.fingerprint != "" {
		fp := c.fingerprint
		c.mu.RUnlock()
		return fp, nil
	}
	c.mu.RUnlock()

	fp, err := computeCertFingerprint(path)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.path = path
	c.fingerprint = fp
	c.mu.Unlock()

	return fp, nil
}

// computeCertFingerprint reads a PEM-encoded certificate file and
// returns its SHA-256 fingerprint formatted as 32 uppercase hex pairs
// separated by colons (standard browser display format).
func computeCertFingerprint(path string) (string, error) {
	if path == "" {
		return "", errEmptyCertPath
	}
	// #nosec G304 -- path is server-controlled (config.Server.CertFile or
	// the self-signed default at certs/server.crt), not user input.
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read certificate: %w", err)
	}
	return fingerprintFromPEM(data)
}

// fingerprintFromPEM extracts the first CERTIFICATE block from PEM data
// and returns its SHA-256 fingerprint as colon-separated uppercase hex.
func fingerprintFromPEM(pemData []byte) (string, error) {
	var certDER []byte
	rest := pemData
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type == pemCertBlockType {
			certDER = block.Bytes
			break
		}
	}
	if certDER == nil {
		return "", errNoCertificateBlock
	}
	// Parse to confirm it is a valid certificate before fingerprinting.
	if _, err := x509.ParseCertificate(certDER); err != nil {
		return "", fmt.Errorf("parse certificate: %w", err)
	}
	sum := sha256.Sum256(certDER)
	return formatFingerprint(sum[:]), nil
}

// formatFingerprint converts a raw hash digest into the colon-separated
// uppercase hex format used by browsers ("Show Certificate" dialogs).
func formatFingerprint(digest []byte) string {
	const hex = "0123456789ABCDEF"
	out := make([]byte, 0, len(digest)*3-1)
	for i, b := range digest {
		if i > 0 {
			out = append(out, ':')
		}
		out = append(out, hex[b>>4], hex[b&0x0f])
	}
	return string(out)
}

// activeCertPath returns the cert file path the server will use, or "" if
// the server is running in HTTP mode. Mirrors the priority order of
// startHTTPS so /__version reports the same cert that is actually served.
func (s *Server) activeCertPath() string {
	if !s.config.Server.HTTPS {
		return ""
	}
	if s.config.Server.ACME.Enabled {
		// ACME certs live in the autocert cache; they are not a single
		// stable file path we can fingerprint here. Return empty rather
		// than guessing.
		return ""
	}
	if s.config.Server.CertFile != "" {
		return s.config.Server.CertFile
	}
	// Fall back to the self-signed default path used by ensureSelfSignedCert.
	return defaultSelfSignedCertPath
}

// defaultSelfSignedCertPath is the path used by ensureSelfSignedCert.
// Kept here so /__version can fingerprint the same file even before the
// HTTPS listener has been started.
const defaultSelfSignedCertPath = "certs/server.crt"

// tlsFingerprintForResponse returns the cached fingerprint (computing it
// on first call). Errors are swallowed and reported as an empty string so
// /__version always returns a stable shape even if the cert is missing or
// unreadable; an empty value is a signal to the operator to investigate.
func (s *Server) tlsFingerprintForResponse() string {
	path := s.activeCertPath()
	if path == "" {
		return ""
	}
	fp, err := s.tlsFingerprint.Get(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(fp)
}
