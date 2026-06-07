package api

// server_lifecycle.go contains the HTTP/HTTPS server lifecycle: Start,
// ACME-managed HTTPS, and the self-signed fallback certificate generator.

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
	"net/http"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Start starts the HTTP/HTTPS server.
// startBackgroundEngines fires up every engine registered with the
// service container's engine.Registry — probe + retention today,
// snmp-poller + listeners as they land. Lifecycle ordering is
// established at registration time; Registry.Start brings them up
// in that order and rolls back already-started engines if any one
// fails. Non-fatal: a failed Start logs a warning and the API
// surface stays available. Extracted from Start() to keep that
// function under the gocognit complexity limit.
//
// V1.0 NMS expansion — Stage A3.5d.
func (s *Server) startBackgroundEngines() {
	if s.services.Engines == nil {
		return
	}
	if err := s.services.Engines.Start(context.Background()); err != nil {
		logging.GetLogger().Warn("engine registry start failed", "error", err)
		return
	}
	logging.GetLogger().Info("engine registry started",
		"engines", engineNames(s.services.Engines.Engines()))
}

// engineNames extracts Name() from each engine for structured-log
// emission without leaking the engine pointers.
func engineNames(engines []engine.Engine) []string {
	out := make([]string, 0, len(engines))
	for _, e := range engines {
		out = append(out, e.Name())
	}
	return out
}

// Handler returns the fully composed HTTP handler: the route mux wrapped in the
// complete middleware stack. Both Start (production) and characterization tests
// use this so they exercise the identical chain.
//
// Stack (outermost → innermost): panic recovery → request ID → logging →
// security headers → body limit → CORS → i18n → API-token → auth (JWT) → CSRF
// → mux (fixes #519). apiTokenMiddleware sits in front of the JWT middleware so
// an `Authorization: Bearer sd_pat_…` resolves a personal-access token before
// the JWT middleware runs; otherwise it falls through.
func (s *Server) Handler() http.Handler {
	return recoverMiddleware(
		logging.RequestIDMiddleware(
			logging.LoggingMiddleware(
				securityHeadersMiddleware(
					bodyLimitMiddleware(
						corsMiddleware(
							i18n.Middleware()(
								apiTokenMiddleware(s.services.Auth.APITokens,
									s.authManager().Middleware(
										s.csrfManager().CSRFMiddleware(s.mux))))))))))
}

func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      s.Handler(),
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

	// Start the multi-interface monitor pool (Pro multi_interface fan-out,
	// seed#1192 / follow-up #1214). The pool was reconciled to the active
	// profile's interface set in NewServer; Start polls each child monitor
	// concurrently so the runtime can observe state changes across N
	// interfaces. A single monitor (Default) is the common Free / Starter
	// case — the pool gracefully handles that with one child.
	if pool := s.services.Network.LinkMonitorPool; pool != nil {
		if err := pool.Start(); err != nil {
			logging.GetLogger().Warn("Link monitor pool: partial start", "error", err)
		} else {
			logging.GetLogger().Info("Link monitor pool started",
				"interfaces", pool.Interfaces())
		}
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
	// This ensures /api/security/devices/status returns valid subnet info on first call
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

	s.startBackgroundEngines()

	if s.config.Server.HTTPS {
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
