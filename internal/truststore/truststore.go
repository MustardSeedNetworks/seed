// Package truststore installs and removes a PEM-encoded X.509 certificate
// in the host operating system's root trust store.
//
// The implementation is intentionally small: it shells out to the platform
// utility that is already required to maintain the trust store on each
// supported OS (the same approach mkcert takes). Compared to depending on
// mkcert's internals — which are not exposed as a library API — this keeps
// seed in control of error reporting and avoids pulling in mkcert's
// command-line surface.
//
// Supported platforms:
//
//   - macOS: `security add-trusted-cert -d -k /Library/Keychains/System.keychain`
//   - Linux (Debian/Ubuntu family): `update-ca-certificates` with the cert
//     copied into /usr/local/share/ca-certificates/.
//   - Linux (RHEL/Fedora family): `update-ca-trust extract` with the cert
//     copied into /etc/pki/ca-trust/source/anchors/.
//   - Linux (Arch): `trust extract-compat` with the cert in
//     /etc/ca-certificates/trust-source/anchors/.
//   - Windows: `certutil.exe -addstore Root` against the LocalMachine ROOT
//     store. Requires an elevated shell.
//
// The package does not write to NSS / Firefox / Chrome user trust stores.
// Those require `certutil` from libnss3-tools and per-profile installation;
// they are out of scope for Wave 1.
package truststore

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Result describes the outcome of an Install or Uninstall call. It is
// returned even on failure so the caller can show partial progress: which
// stores were touched and which were skipped.
type Result struct {
	// Stores is the list of trust stores that were modified (or attempted).
	Stores []string
	// Skipped lists trust stores that could not be modified (missing tool,
	// unsupported distribution, etc.) along with the reason.
	Skipped []string
}

// ErrUnsupportedPlatform is returned when the host OS is not one of the
// supported targets. install-ca prints this with a friendly hint.
var ErrUnsupportedPlatform = errors.New("trust store install is not supported on this platform")

// ErrInvalidCertificate is returned when the file at the given path is not
// a PEM-encoded X.509 certificate that the standard library can parse.
var ErrInvalidCertificate = errors.New("file is not a valid PEM-encoded X.509 certificate")

// ValidateCertFile reads the file at path and confirms it contains at least
// one CERTIFICATE PEM block that crypto/x509 can parse. Returns the parsed
// leaf certificate (the first block) so callers can derive a fingerprint
// or display Subject/Issuer info.
func ValidateCertFile(path string) (*x509.Certificate, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty path", ErrInvalidCertificate)
	}
	// #nosec G304 -- path is supplied by the operator running install-ca
	// (typically certs/server.crt), not by an untrusted source.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read certificate: %w", err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, ErrInvalidCertificate
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidCertificate, err)
	}
	return cert, nil
}

// Install adds the certificate at certPath to the host's root trust store.
// On Linux/macOS the command will typically require sudo; callers should
// invoke `seed install-ca` under sudo rather than re-implementing
// privilege escalation here.
func Install(ctx context.Context, certPath string) (Result, error) {
	if _, err := ValidateCertFile(certPath); err != nil {
		return Result{}, err
	}
	abs, err := filepath.Abs(certPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve cert path: %w", err)
	}
	return installPlatform(ctx, abs)
}

// Uninstall removes the certificate at certPath from the host's root
// trust store. Cert identity is platform-specific: macOS keys on cert
// data, Linux on the file name written under the anchors directory,
// Windows on subject + serial.
func Uninstall(ctx context.Context, certPath string) (Result, error) {
	if _, err := ValidateCertFile(certPath); err != nil {
		return Result{}, err
	}
	abs, err := filepath.Abs(certPath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve cert path: %w", err)
	}
	return uninstallPlatform(ctx, abs)
}

// trimError shortens combined-output blobs so error messages stay readable
// when surfaced to the operator.
func trimError(out []byte) string {
	s := strings.TrimSpace(string(out))
	const maxLen = 400
	if len(s) > maxLen {
		s = s[:maxLen] + "…"
	}
	return s
}
