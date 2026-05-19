package api

// server_lifecycle.go contains the HTTP/HTTPS server lifecycle: Start, the
// HTTP→HTTPS redirect server, ACME-managed HTTPS, and the self-signed
// fallback certificate generator.

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
)

// Start starts the HTTP/HTTPS server.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	// Apply middleware stack: panic recovery → request ID → logging → security headers → body limit → CORS → i18n → auth → CSRF (fixes #519)
	// Panic recovery is outermost to catch all panics
	// Request ID middleware generates unique IDs for request correlation in logs
	// Logging middleware logs all HTTP requests with timing, status, and request IDs
	// Body limit middleware enforces request body size limits
	// i18n middleware extracts Accept-Language and attaches localizer to context
	// CSRF middleware validates tokens on state-changing requests (POST, PUT, DELETE)
	handler := recoverMiddleware(
		logging.RequestIDMiddleware(
			logging.LoggingMiddleware(
				securityHeadersMiddleware(
					bodyLimitMiddleware(
						corsMiddleware(
							i18n.Middleware()(
								s.authManager().Middleware(
									s.csrfManager().CSRFMiddleware(s.mux)))))))))

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  serverReadTimeoutSec * time.Second,
		WriteTimeout: serverWriteTimeoutMin * time.Minute, // Increased for large file downloads/exports (fixes #529)
		IdleTimeout:  serverIdleTimeoutSec * time.Second,
	}

	// WebSocket hub already running (started in NewServer to fix #512 race condition)
	// Start WebSocket broadcast loop
	s.startBroadcastLoop()

	// Start link state monitor
	if err := s.linkMonitor().Start(); err != nil {
		logging.GetLogger().Warn("Link monitor failed to start", "error", err)
	} else {
		logging.GetLogger().Info("Link monitor started",
			"interface", s.config.Interface.Default,
			"state", s.linkMonitor().GetState())
	}

	// Start unified discovery service.
	if err := s.discoveryService().Start(); err != nil {
		logging.GetLogger().
			Warn("Discovery service failed to start (may require root)", "error", err)
	} else {
		status := s.discoveryService().GetStatus()
		logging.GetLogger().Info("Discovery service started",
			"methods", status.ActiveMethods)
	}

	// Trigger initial device discovery scan to populate subnet info immediately
	// This ensures /api/shell/devices/status returns valid subnet info on first call
	// without requiring a manual scan trigger from the frontend
	if s.config.NetworkDiscovery.Enabled {
		go func() {
			ctx, cancel := context.WithTimeout(
				context.Background(),
				s.config.NetworkDiscovery.ScanTimeout,
			)
			defer cancel()
			logging.GetLogger().Info("Triggering initial device discovery scan on startup")
			if err := s.deviceDiscovery().Scan(ctx); err != nil {
				logging.GetLogger().Warn("Initial device discovery scan failed", "error", err)
			} else {
				logging.GetLogger().Info("Initial device discovery scan completed",
					"deviceCount", s.deviceDiscovery().Count())
			}
		}()
	}

	// Start VLAN traffic monitor (requires root/CAP_NET_RAW)
	if err := s.vlanTrafficMonitor().Start(); err != nil {
		logging.GetLogger().
			Warn("VLAN traffic monitor failed to start (may require root)", "error", err)
	} else {
		logging.GetLogger().Info("VLAN traffic monitor started")
	}

	if s.config.Server.HTTPS {
		// Start HTTP→HTTPS redirect server if configured
		if s.config.Server.HTTPRedirectPort > 0 {
			go s.startHTTPRedirect(s.config.Server.HTTPRedirectPort)
		}
		return s.startHTTPS()
	}
	return s.startHTTP()
}

// startHTTP starts the server in HTTP mode.
//
// Uses bindWithFallback so that a busy canonical port falls back to
// port+1..+9 instead of failing outright. The actual bound port is
// reflected back into s.httpServer.Addr so /__version and log lines
// match reality (fixes #69).
func (s *Server) startHTTP() error {
	ln, actualPort, err := bindWithFallback(context.Background(), "", s.config.Server.Port)
	if err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	s.httpServer.Addr = fmt.Sprintf(":%d", actualPort)
	logging.GetLogger().Info("Starting HTTP server", "addr", s.httpServer.Addr)
	if serveErr := s.httpServer.Serve(ln); serveErr != nil {
		return fmt.Errorf("http server: %w", serveErr)
	}
	return nil
}

// httpToHTTPSRedirectHandler returns an [http.Handler] that permanently
// redirects every request to the equivalent HTTPS URL. Extracted from
// startHTTPRedirect so it can be exercised in tests without bringing up
// a real listener.
func (s *Server) httpToHTTPSRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build HTTPS URL preserving the host and path
		host := r.Host
		// Remove port from host if present (to avoid localhost:80 → https://localhost:80:8443)
		if colonPos := strings.LastIndex(host, ":"); colonPos != -1 {
			host = host[:colonPos]
		}

		// If HTTPS is on standard port 443, don't include it in URL
		httpsPort := s.config.Server.Port
		var httpsURL string
		if httpsPort == httpsDefaultPort {
			httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
		} else {
			httpsURL = "https://" + net.JoinHostPort(host, strconv.Itoa(httpsPort)) + r.RequestURI
		}

		// #nosec G710 -- httpsURL is server-controlled: scheme/port from our config, host stripped to its
		// bare form before re-joining; user-supplied r.RequestURI is appended as the path/query only.
		//
		// Use 308 Permanent Redirect (RFC 7538) rather than 301 so the HTTP
		// method and body are preserved across the redirect. 301 allows
		// clients to downgrade POST to GET, which would silently break
		// state-changing API calls that arrive on the HTTP listener.
		http.Redirect(w, r, httpsURL, http.StatusPermanentRedirect)
	})
}

// startHTTPRedirect starts an HTTP server that redirects all requests to HTTPS (fixes #515).
// Properly tracks the server and allows shutdown.
func (s *Server) startHTTPRedirect(port int) {
	redirectHandler := s.httpToHTTPSRedirectHandler()

	// Bind via the port-fallback helper so a busy :80 (or whatever is
	// configured) walks up to +9 instead of killing the redirect goroutine.
	ln, actualPort, bindErr := bindWithFallback(context.Background(), "", port)
	if bindErr != nil {
		logging.GetLogger().Error("HTTP redirect server bind failed", "error", bindErr)
		if s.redirectServerErr == nil {
			s.redirectServerErr = make(chan error, 1)
		}
		s.redirectServerErr <- bindErr
		return
	}
	addr := fmt.Sprintf(":%d", actualPort)
	logging.GetLogger().Info("Starting HTTP→HTTPS redirect server", "addr", addr)

	// Store redirect server for proper shutdown (fixes #515)
	s.redirectServer = &http.Server{
		Addr:         addr,
		Handler:      redirectHandler,
		ReadTimeout:  redirectReadWriteTimeoutSec * time.Second,
		WriteTimeout: redirectReadWriteTimeoutSec * time.Second,
	}

	// Create error channel if not already created
	if s.redirectServerErr == nil {
		s.redirectServerErr = make(chan error, 1)
	}

	// Run server and report errors
	err := s.redirectServer.Serve(ln)
	if err != nil && err != http.ErrServerClosed {
		logging.GetLogger().Error("HTTP redirect server error", "error", err)
		s.redirectServerErr <- err
	}
}

// startHTTPS starts the server in HTTPS mode.
func (s *Server) startHTTPS() error {
	// Priority 1: ACME/Let's Encrypt automatic certificates
	if s.config.Server.ACME.Enabled {
		if s.config.Server.ACME.Domain == "" {
			return errors.New("ACME enabled but no domain specified")
		}
		return s.startHTTPSWithACME()
	}

	// Priority 2: Manual certificates from config
	certFile := s.config.Server.CertFile
	keyFile := s.config.Server.KeyFile

	// Priority 3: Self-signed certificate (fallback)
	if certFile == "" || keyFile == "" {
		var err error
		certFile, keyFile, err = s.ensureSelfSignedCert()
		if err != nil {
			return fmt.Errorf("failed to generate self-signed certificate: %w", err)
		}
	}

	// Configure TLS 1.3 (fixes #523)
	// CipherSuites is not set because TLS 1.3 uses its own mandatory cipher suites:
	// - TLS_AES_128_GCM_SHA256
	// - TLS_AES_256_GCM_SHA384
	// - TLS_CHACHA20_POLY1305_SHA256
	// Setting CipherSuites with TLS 1.3 is misleading as Go ignores them.
	// If you need to control ciphers, use MinVersion: tls.VersionTLS12
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}
	s.httpServer.TLSConfig = tlsConfig

	ln, actualPort, bindErr := bindWithFallback(context.Background(), "", s.config.Server.Port)
	if bindErr != nil {
		return fmt.Errorf("https server: %w", bindErr)
	}
	s.httpServer.Addr = fmt.Sprintf(":%d", actualPort)

	logging.GetLogger().
		Info("Starting HTTPS server", "addr", s.httpServer.Addr, "tls_version", "1.3")
	if err := s.httpServer.ServeTLS(ln, certFile, keyFile); err != nil {
		return fmt.Errorf("https server: %w", err)
	}
	return nil
}

// startHTTPSWithACME starts the server with automatic Let's Encrypt certificates.
func (s *Server) startHTTPSWithACME() error {
	cacheDir := s.config.Server.ACME.CacheDir
	if cacheDir == "" {
		cacheDir = "certs/acme"
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("failed to create ACME cache dir: %w", err)
	}

	manager := &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.config.Server.ACME.Domain),
		Cache:      autocert.DirCache(cacheDir),
		Email:      s.config.Server.ACME.Email,
	}

	// Use Let's Encrypt staging server for testing (certs won't be trusted by browsers)
	if s.config.Server.ACME.Staging {
		manager.Client = &acme.Client{
			DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory",
		}
		logging.GetLogger().
			Warn("ACME: Using Let's Encrypt STAGING server (certificates will not be trusted)")
	}

	// Configure TLS with ACME
	tlsConfig := manager.TLSConfig()
	tlsConfig.MinVersion = tls.VersionTLS13

	s.httpServer.TLSConfig = tlsConfig

	logging.GetLogger().Info("Starting HTTPS server with ACME",
		"addr", s.httpServer.Addr,
		"domain", s.config.Server.ACME.Domain)

	// Start HTTP-01 challenge handler on port 80
	// This is required for Let's Encrypt domain validation
	// Store reference so it can be shut down properly (fixes #837)
	s.acmeChallengeServer = &http.Server{
		Addr:              ":80",
		Handler:           manager.HTTPHandler(nil),
		ReadHeaderTimeout: acmeReadHeaderTimeoutSec * time.Second,
	}
	go func() {
		logging.GetLogger().Info("Starting HTTP-01 challenge handler", "addr", ":80")
		if err := s.acmeChallengeServer.ListenAndServe(); err != nil &&
			err != http.ErrServerClosed {
			logging.GetLogger().Error("HTTP-01 handler error", "error", err)
		}
	}()

	ln, actualPort, bindErr := bindWithFallback(context.Background(), "", s.config.Server.Port)
	if bindErr != nil {
		return fmt.Errorf("https server with ACME: %w", bindErr)
	}
	s.httpServer.Addr = fmt.Sprintf(":%d", actualPort)

	// ServeTLS with empty cert/key paths uses GetCertificate from TLSConfig.
	if err := s.httpServer.ServeTLS(ln, "", ""); err != nil {
		return fmt.Errorf("https server with ACME: %w", err)
	}
	return nil
}

// ensureSelfSignedCert generates a self-signed certificate if needed.
func (s *Server) ensureSelfSignedCert() (string, string, error) {
	certsDir := "certs"
	certFile := filepath.Join(certsDir, "server.crt")
	keyFile := filepath.Join(certsDir, "server.key")

	// Check if certs already exist
	if _, certErr := os.Stat(certFile); certErr == nil {
		if _, keyErr := os.Stat(keyFile); keyErr == nil {
			return certFile, keyFile, nil
		}
	}

	// Ensure certs directory exists
	if err := os.MkdirAll(certsDir, 0o700); err != nil {
		return "", "", fmt.Errorf("create certs directory: %w", err)
	}

	// Generate private key with 4096-bit RSA (fixes #533)
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyBits)
	if err != nil {
		return "", "", fmt.Errorf("generate RSA key: %w", err)
	}

	// Create certificate template.
	//
	// The cert is a single-tier self-signed CA: it acts as both the root
	// (Issuer == Subject) and the leaf the TLS listener serves. This lets
	// `seed install-ca` install the same file into the OS trust store so
	// browsers stop showing the self-signed warning. Without IsCA=true and
	// KeyUsageCertSign, OS trust stores will reject the cert as not
	// eligible to act as a root.
	//
	// Existing certs on disk are not regenerated automatically; they will
	// continue to work for TLS but cannot be installed as roots until they
	// are deleted and seed regenerates them.
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"The Seed"},
			CommonName:   "The Seed Self-Signed",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage: x509.KeyUsageKeyEncipherment |
			x509.KeyUsageDigitalSignature |
			x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost", "seed.local"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		return "", "", fmt.Errorf("create certificate: %w", err)
	}

	// Write certificate

	certOut, err := os.Create(certFile)
	if err != nil {
		return "", "", fmt.Errorf("create cert file: %w", err)
	}
	defer func() { _ = certOut.Close() }()
	if encodeErr := pem.Encode(certOut, &pem.Block{Type: pemCertBlockType, Bytes: certDER}); encodeErr != nil {
		return "", "", fmt.Errorf("encode certificate PEM: %w", encodeErr)
	}

	// Write private key

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", fmt.Errorf("create key file: %w", err)
	}
	defer func() { _ = keyOut.Close() }()
	if keyEncodeErr := pem.Encode(
		keyOut,
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)},
	); keyEncodeErr != nil {
		return "", "", fmt.Errorf("encode private key PEM: %w", keyEncodeErr)
	}

	logging.GetLogger().Info("Generated self-signed certificate", "cert_file", certFile)
	return certFile, keyFile, nil
}
