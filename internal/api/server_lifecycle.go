package api

// server_lifecycle.go contains the HTTPS server lifecycle and the self-signed
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
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/supervise"

	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Start starts the HTTPS server.
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
	if s.engines == nil {
		return
	}
	if err := s.engines.Start(context.Background()); err != nil {
		logging.GetLogger().Warn("engine registry start failed", "error", err)
		return
	}
	logging.GetLogger().Info("engine registry started",
		"engines", engineNames(s.engines.Engines()))
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
								apiTokenMiddleware(s.apiTokens, s.resolveClientID,
									s.authManager().Middleware(
										s.csrfManager().CSRFMiddleware(s.mux))))))))))
}

// Start brings the server fully online: it starts the link monitor(s),
// discovery service, VLAN traffic monitor, and background engines, kicks off
// an initial discovery scan when network discovery is enabled, then blocks
// serving HTTPS via startHTTPS until the listener stops or errors. A failure
// to start any individual subsystem is logged and does not abort startup;
// only the final HTTPS listen error is returned.
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
	if pool := s.linkMonPool; pool != nil {
		if err := pool.Start(); err != nil {
			logging.GetLogger().Warn("Link monitor pool: partial start", "error", err)
		} else {
			logging.GetLogger().Info("Link monitor pool started",
				"interfaces", pool.Interfaces())
		}
	}

	// Read every sweep the service drives: its startup sweep, the rescan
	// ticker and an interface change — learning target networks (seed#2695)
	// and promoting the devices that answered SNMP (seed#2692). Registered
	// before Start so the startup sweep is already observed.
	s.discoveryService().SetSweepObserver(s.afterSweep)

	// Start unified discovery service.
	if err := s.discoveryService().Start(); err != nil {
		logging.GetLogger().
			Warn("Discovery service failed to start (may require root)", "error", err)
	} else {
		status := s.discoveryService().GetStatus()
		logging.GetLogger().Info("Discovery service started",
			"methods", status.ActiveMethods)
	}

	// A running service has already swept the shared registry at startup
	// (seed#2831); a second sweep here only collided with it. Sweep directly
	// when the service could not start, so /api/security/devices/status still
	// reports a subnet without a manual scan.
	if s.config.NetworkDiscovery.Enabled && !s.discoveryService().IsRunning() {
		s.startInitialScan()
	}

	s.startBackgroundEngines()

	return s.startHTTPS()
}

// startInitialScan runs the one-shot startup sweep under its own supervisor
// (#2748) so a panic in the scan path is logged and contained instead of
// killing the daemon.
//
// Its own group, not the background components' one: the scan is a one-shot
// with its own timeout, and supervise.Fatal on a shared group would stop the
// outbox relay, the report scheduler and the Wi-Fi loops because a sweep
// faulted — worse than the crash it replaces. Nothing waits on it at shutdown
// either, exactly as the bare goroutine it replaces did not: the scan's own
// timeout bounds it, and blocking shutdown on a sweep that ignores its context
// would trade a crash for a hang.
func (s *Server) startInitialScan() {
	group := supervise.New(logging.GetLogger())
	group.Add("initial-scan", supervise.Fatal, func(workerCtx context.Context) error {
		// The timeout is built inside the worker: supervise.Start detaches the
		// context it is given with context.WithoutCancel, which strips the
		// DEADLINE as well as the cancellation, and the scan path sizes its
		// per-probe timeouts from ctx.Deadline().
		scanCtx, cancel := context.WithTimeout(workerCtx, s.config.NetworkDiscovery.ScanTimeout)
		defer cancel()

		logging.GetLogger().InfoContext(scanCtx, "Triggering initial device discovery scan on startup")
		if err := s.deviceDiscovery().Scan(scanCtx); err != nil {
			logging.GetLogger().WarnContext(scanCtx, "Initial device discovery scan failed", "error", err)
			return nil
		}
		logging.GetLogger().InfoContext(scanCtx, "Initial device discovery scan completed",
			"deviceCount", s.deviceDiscovery().Count())
		return nil
	})
	group.Start(context.Background())
}

// SetBoundPortObserver registers the callback that receives the port the
// listener actually bound, once the +1..+9 fallback has settled. It is how
// cmd/seed publishes the running port on the single-instance lock.
//
// Registered before Start; there is no lock because nothing reads or writes it
// afterwards.
func (s *Server) SetBoundPortObserver(observe func(int)) {
	s.boundPort = observe
}

// startHTTPS starts the server with an operator-provided certificate or a
// generated self-signed certificate.
func (s *Server) startHTTPS() error {
	certFile := s.config.Server.CertFile
	keyFile := s.config.Server.KeyFile
	if (certFile == "") != (keyFile == "") {
		return errors.New("server.cert_file and server.key_file must be configured together")
	}

	// Generate a self-signed certificate when the operator did not provide one.
	if certFile == "" {
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
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
	}
	s.httpServer.TLSConfig = tlsConfig

	ln, actualPort, bindErr := bindWithFallback(context.Background(), "", s.config.Server.Port)
	if bindErr != nil {
		return fmt.Errorf("https server: %w", bindErr)
	}
	s.httpServer.Addr = fmt.Sprintf(":%d", actualPort)
	s.initWebAuthn(actualPort)
	if s.boundPort != nil {
		s.boundPort(actualPort)
	}

	logging.GetLogger().
		Info("Starting HTTPS server", "addr", s.httpServer.Addr, "tls_version", "1.3")
	if err := s.httpServer.ServeTLS(ln, certFile, keyFile); err != nil {
		return fmt.Errorf("https server: %w", err)
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
