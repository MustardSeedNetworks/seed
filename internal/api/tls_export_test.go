package api

import (
	"context"
	"net/http"
	"path/filepath"

	"github.com/go-webauthn/webauthn/webauthn"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver"
	"github.com/MustardSeedNetworks/foundation/pkg/passkey"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// CompleteWebAuthnLoginForTest exercises finalization after protocol verification.
func CompleteWebAuthnLoginForTest(
	s *Server, w http.ResponseWriter, r *http.Request, username string, credential *webauthn.Credential,
) {
	s.completeWebAuthnLogin(w, r, username, credential)
}

// EnsureSelfSignedCert generates the self-signed pair startHTTPS would, in the
// working directory's certs/, and returns its paths, so a test can trust the
// certificate before the listener is up.
func EnsureSelfSignedCert() (string, string, error) {
	certFile := filepath.Join(selfSignedCertDir, httpserver.DefaultCertFileName)
	keyFile := filepath.Join(selfSignedCertDir, httpserver.DefaultKeyFileName)
	if _, err := httpserver.EnsureCertificate(nil, certFile, keyFile, selfSignedCertOptions()); err != nil {
		return "", "", err
	}
	return certFile, keyFile, nil
}

// StartHTTPSForTest starts the production TLS listener with a focused handler.
func (s *Server) StartHTTPSForTest(handler http.Handler, initialized chan<- struct{}) error {
	s.httpServer = &http.Server{Handler: handler}
	close(initialized)
	return s.startHTTPS()
}

// ShutdownHTTPSForTest stops the focused production listener.
func (s *Server) ShutdownHTTPSForTest(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// PasskeysEnabledForTest reports whether the bound listener initialized passkeys.
func (s *Server) PasskeysEnabledForTest() bool {
	return s.passkeys != nil
}

// SetPasskeyServiceForTest installs the passkey service the HTTPS listener
// would otherwise build at bind time.
func (s *Server) SetPasskeyServiceForTest(service *passkey.Service) {
	s.passkeys = service
}

// ExportFingerprintFromPEM exposes fingerprintFromPEM for testing.
func ExportFingerprintFromPEM(pemData []byte) (string, error) {
	return fingerprintFromPEM(pemData)
}

// ExportComputeCertFingerprint exposes computeCertFingerprint for testing.
func ExportComputeCertFingerprint(path string) (string, error) {
	return computeCertFingerprint(path)
}

// ExportFormatFingerprint exposes formatFingerprint for testing.
func ExportFormatFingerprint(digest []byte) string {
	return formatFingerprint(digest)
}

// ErrEmptyCertPath exposes errEmptyCertPath for testing.
var ErrEmptyCertPath = errEmptyCertPath

// ErrNoCertificateBlock exposes errNoCertificateBlock for testing.
var ErrNoCertificateBlock = errNoCertificateBlock

// TLSFingerprintCache is the exported type alias for tlsFingerprintCache,
// used in tests that exercise the cache behaviour directly.
type TLSFingerprintCache = tlsFingerprintCache

// ExportPasskeyConfigFromServer exposes relying-party derivation for testing.
func ExportPasskeyConfigFromServer(cfg *config.Config, listenerPort int) passkey.Config {
	return passkeyConfigFromServer(cfg, listenerPort)
}
