package api

// server_lifecycle.go contains the HTTPS server lifecycle and the self-signed
// fallback certificate generator.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver"
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

// selfSignedCertDir is where foundation's httpserver writes the generated pair
// (server.crt / server.key) when the operator configured no certificate.
const selfSignedCertDir = "certs"

// selfSignedCertOptions describes the generated certificate. It is its own CA,
// so `seed install-ca` can put the same file in the OS trust store.
func selfSignedCertOptions() httpserver.CertOptions {
	return httpserver.CertOptions{
		CommonName: "The Seed Self-Signed",
		DNSNames:   []string{"localhost", "seed.local"},
	}
}

// startHTTPS binds foundation's HTTPS listener (port fallback per #69, TLS 1.3,
// the self-signed default and the same-port plaintext 308 redirect) and serves
// on it. It is the only listener: a plaintext request on the TLS port gets a
// redirect to https and nothing else.
func (s *Server) startHTTPS() error {
	certFile := s.config.Server.CertFile
	keyFile := s.config.Server.KeyFile
	if (certFile == "") != (keyFile == "") {
		return errors.New("server.cert_file and server.key_file must be configured together")
	}

	ln, listenErr := httpserver.Listen(context.Background(), httpserver.Config{
		Addr:     net.JoinHostPort("", strconv.Itoa(s.config.Server.Port)),
		CertFile: certFile,
		KeyFile:  keyFile,
		CertDir:  selfSignedCertDir,
		Cert:     selfSignedCertOptions(),
		Logger:   logging.GetLogger(),
	})
	if listenErr != nil {
		return fmt.Errorf("https server: %w", listenErr)
	}
	tcpAddr, isTCP := ln.Addr().(*net.TCPAddr)
	if !isTCP {
		_ = ln.Close()
		return fmt.Errorf("https server: listener address %s is not TCP", ln.Addr())
	}
	actualPort := tcpAddr.Port
	s.httpServer.Addr = ln.Addr().String()
	s.initWebAuthn(actualPort)
	if s.boundPort != nil {
		s.boundPort(actualPort)
	}

	logging.GetLogger().
		Info("Starting HTTPS server", "addr", s.httpServer.Addr, "tls_version", "1.3")
	if err := s.httpServer.Serve(ln); err != nil {
		return fmt.Errorf("https server: %w", err)
	}
	return nil
}
