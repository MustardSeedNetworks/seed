// Package api provides the HTTP/WebSocket server.
package api

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/krisarmstrong/luminetiq/internal/auth"
	"github.com/krisarmstrong/luminetiq/internal/cable"
	"github.com/krisarmstrong/luminetiq/internal/config"
	"github.com/krisarmstrong/luminetiq/internal/dhcp"
	"github.com/krisarmstrong/luminetiq/internal/discovery"
	"github.com/krisarmstrong/luminetiq/internal/dns"
	"github.com/krisarmstrong/luminetiq/internal/gateway"
	"github.com/krisarmstrong/luminetiq/internal/iperf"
	"github.com/krisarmstrong/luminetiq/internal/network"
	"github.com/krisarmstrong/luminetiq/internal/publicip"
	"github.com/krisarmstrong/luminetiq/internal/speedtest"
	"github.com/krisarmstrong/luminetiq/internal/survey"
	"github.com/krisarmstrong/luminetiq/internal/vlan"
	"github.com/krisarmstrong/luminetiq/internal/wifi"
	"github.com/krisarmstrong/luminetiq/web"
)

// Server represents the HTTP/HTTPS server.
type Server struct {
	config           *config.Config
	configPath       string
	logPath          string
	httpServer          *http.Server
	authManager         *auth.Manager
	loginRateLimiter    *RateLimiter
	endpointRateLimiter *EndpointRateLimiter // Rate limiter for expensive endpoints (fixes #530)
	wsHub               *Hub
	mux              *http.ServeMux
	netManager       *network.Manager
	linkMonitor      *network.LinkMonitor
	discoveryManager *discovery.Manager         // Legacy: LLDP/CDP/EDP protocol capture
	deviceDiscovery  *discovery.DeviceDiscovery // Legacy: device aggregation
	discoveryService *discovery.Service         // New unified discovery orchestrator
	dnsTester        *dns.Tester
	dhcpMonitor      *dhcp.Monitor
	rogueDetector    *dhcp.RogueDetector
	gatewayTester    *gateway.Tester
	vlanManager        *vlan.Manager
	vlanTrafficMonitor *vlan.TrafficMonitor
	wifiManager        *wifi.Manager
	wifiScanner        *wifi.Scanner
	cableTester      *cable.Tester
	speedtestTester  *speedtest.Tester
	iperfManager     *iperf.Manager
	surveyManager    *survey.Manager
	vulnScanner      *discovery.VulnerabilityScanner
	publicipChecker   *publicip.Checker
	logAccessToken    string
	logAccessHeader   string
	requireLogToken   bool
	icmpAvailable     bool      // Whether raw ICMP sockets are available
	startTime         time.Time // Application start time for uptime tracking (fixes #540)
	redirectServer    *http.Server // HTTP→HTTPS redirect server (fixes #515)
	redirectServerErr chan error    // Error channel for redirect server
}

// NewServer creates a new server instance.
func NewServer(cfg *config.Config, configPath, logPath string, netMgr *network.Manager, icmpAvailable bool) *Server {
	s := &Server{
		config:        cfg,
		configPath:    configPath,
		logPath:       logPath,
		mux:           http.NewServeMux(),
		netManager:    netMgr,
		icmpAvailable: icmpAvailable,
		startTime:     time.Now(), // Track application start time (fixes #540)
		authManager: auth.NewManager(
			cfg.Auth.JWTSecret,
			cfg.Auth.SessionTimeout,
			cfg.Auth.DefaultUsername,
			cfg.Auth.DefaultPasswordHash,
		),
		loginRateLimiter:    NewRateLimiter(DefaultRateLimitConfig()),
		endpointRateLimiter: NewEndpointRateLimiter(DefaultEndpointRateLimitConfig()), // Rate limit expensive endpoints (fixes #530)
		linkMonitor:         network.NewLinkMonitor(cfg.Interface.Default),
		discoveryManager: discovery.NewManager(cfg.Interface.Default),
		deviceDiscovery:  discovery.NewDeviceDiscoveryWithOUI(cfg.Interface.Default, cfg.NetworkDiscovery.OUIFilePath, cfg.NetworkDiscovery.OUIMaxAge),
		discoveryService: discovery.NewService(cfg, cfg.Interface.Default),
		dnsTester:        dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds()),
		dhcpMonitor:      dhcp.NewMonitor(cfg.Interface.Default),
		rogueDetector:    dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
			Interface:       cfg.Interface.Default,
			KnownServers:    cfg.DHCP.RogueDetection.KnownServers,
			AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
		}),
		gatewayTester:    gateway.NewTester(gateway.DefaultThresholds()),
		vlanManager:        vlan.NewManager(cfg.Interface.Default),
		vlanTrafficMonitor: vlan.NewTrafficMonitor(cfg.Interface.Default),
		wifiManager:        wifi.NewManager(cfg.Interface.Default),
		wifiScanner:        wifi.NewScanner(cfg.Interface.Default),
		cableTester:      cable.NewTester(cfg.Interface.Default),
		speedtestTester:  speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID),
		iperfManager:     iperf.NewManager(),
		publicipChecker:  publicip.NewChecker(),
		logAccessToken:   cfg.Server.LogAccessToken,
		logAccessHeader:  cfg.Server.LogAccessHeader,
		requireLogToken:  cfg.Server.RequireLogAccess,
	}

	// Set up link state change callback
	s.linkMonitor.OnStateChange(s.onLinkStateChange)

	// Initialize DNS tester with configured servers from config
	if len(cfg.DNS.Servers) > 0 {
		configuredServers := make([]dns.ConfiguredServer, 0, len(cfg.DNS.Servers))
		for _, d := range cfg.DNS.Servers {
			configuredServers = append(configuredServers, dns.ConfiguredServer{
				Address: d.Address,
				Enabled: d.Enabled,
			})
		}
		s.dnsTester.SetConfiguredServers(configuredServers)
	}

	// Initialize device discovery with configured additional subnets
	if len(cfg.NetworkDiscovery.AdditionalSubnets) > 0 {
		enabledCIDRs := make([]string, 0)
		for _, subnet := range cfg.NetworkDiscovery.AdditionalSubnets {
			if subnet.Enabled {
				enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
			}
		}
		if len(enabledCIDRs) > 0 {
			if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
				log.Printf("Warning: Failed to set additional subnets: %v", err)
			} else {
				log.Printf("Configured %d additional subnets for scanning", len(enabledCIDRs))
			}
		}
	}

	// Initialize survey manager
	surveyStoragePath := "data/surveys"
	s.surveyManager = survey.NewManager(surveyStoragePath, s.wifiScanner, s.wifiManager, s.iperfManager)
	if err := s.surveyManager.LoadSurveys(); err != nil {
		log.Printf("Warning: Failed to load surveys: %v", err)
	}

	// Initialize WebSocket hub (fixes #512)
	s.wsHub = NewHub()
	// Start hub before setupRoutes to prevent race condition
	go s.wsHub.Run()

	// Initialize vulnerability scanner if enabled
	if cfg.Security.VulnerabilityScanning.Enabled {
		scannerCfg := &discovery.VulnerabilityScannerConfig{
			Enabled:           cfg.Security.VulnerabilityScanning.Enabled,
			CVEDatabase:       cfg.Security.VulnerabilityScanning.CVEDatabase,
			NVDAPIKey:         cfg.Security.VulnerabilityScanning.NVDAPIKey,
			UpdateInterval:    cfg.Security.VulnerabilityScanning.UpdateInterval,
			SeverityThreshold: cfg.Security.VulnerabilityScanning.SeverityThreshold,
			MaxConcurrent:     cfg.Security.VulnerabilityScanning.MaxConcurrent,
		}

		vulnScanner, err := discovery.NewVulnerabilityScanner(scannerCfg)
		if err != nil {
			log.Printf("Warning: Failed to initialize vulnerability scanner: %v", err)
		} else {
			s.vulnScanner = vulnScanner
			log.Printf("Vulnerability scanner initialized (CVE DB: %s, threshold: %s)",
				scannerCfg.CVEDatabase, scannerCfg.SeverityThreshold)
		}
	}

	// Configure security: allowed origins for CORS/WebSocket
	SetAllowedOrigins(cfg.Security.AllowedOrigins)
	if len(cfg.Security.AllowedOrigins) > 0 {
		log.Printf("Configured %d explicit allowed origins for CORS/WebSocket", len(cfg.Security.AllowedOrigins))
	} else {
		log.Println("Using default RFC 1918 private network origins for CORS/WebSocket")
	}

	// Setup routes (wsHub already initialized and running above)
	s.setupRoutes()

	return s
}

// onLinkStateChange handles link up/down events.
func (s *Server) onLinkStateChange(event network.LinkEvent) {
	log.Printf("Link state change: %s -> %s", event.Interface, event.State)

	switch event.State {
	case network.LinkStateUp:
		// Link came up - restart discovery to catch LLDP/CDP frames
		log.Println("Link up - restarting discovery capture")
		s.discoveryManager.Stop()
		if err := s.discoveryManager.Start(); err != nil {
			log.Printf("Warning: Failed to restart discovery: %v", err)
		}

		// Notify WebSocket clients
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]interface{}{
				"interface": event.Interface,
				"state":     "up",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})
	case network.LinkStateDown:
		// Link went down - notify clients
		log.Println("Link down - notifying clients")
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]interface{}{
				"interface": event.Interface,
				"state":     "down",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})
	}
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/settings", s.handleSettings)
	s.mux.HandleFunc("/api/interfaces", s.handleInterfaces)
	s.mux.HandleFunc("/api/interface", s.handleInterface)
	s.mux.HandleFunc("/api/export", s.handleExport)
	s.mux.HandleFunc("/api/link", s.handleLink)
	s.mux.HandleFunc("/api/ipconfig", s.handleIPConfig)
	s.mux.HandleFunc("/api/ipconfig/settings", s.handleIPSettings)
	s.mux.HandleFunc("/api/network/mtu", s.handleSetMTU)
	s.mux.HandleFunc("/api/discovery", s.handleDiscovery)
	s.mux.HandleFunc("/api/discovery/probe", s.handleTCPProbe)
	s.mux.HandleFunc("/api/discovery/traceroute", s.handleTraceroute)
	s.mux.HandleFunc("/api/discovery/portscan", s.handlePortScan)
	s.mux.HandleFunc("/api/dns", s.handleDNS)
	s.mux.HandleFunc("/api/dhcp/rogue", s.handleRogueDHCP)
	s.mux.HandleFunc("/api/dhcp/rogue/servers", s.handleRogueDHCPServers)
	s.mux.HandleFunc("/api/dhcp/rogue/config", s.handleRogueDHCPConfig)
	s.mux.HandleFunc("/api/gateway", s.handleGateway)
	s.mux.HandleFunc("/api/vlan", s.handleVLAN)
	s.mux.HandleFunc("/api/vlan/traffic", s.handleVLANTraffic)
	s.mux.HandleFunc("/api/vlan/interface", s.handleVLANInterface)
	s.mux.HandleFunc("/api/wifi", s.handleWiFi)
	s.mux.HandleFunc("/api/wifi/settings", s.handleWiFiSettings)
	s.mux.HandleFunc("/api/snmp/settings", s.handleSNMPSettings)
	s.mux.HandleFunc("/api/cable", s.handleCable)
	// Rate-limited expensive endpoints (fixes #530)
	s.mux.Handle("/api/speedtest", s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleSpeedtest)))
	s.mux.HandleFunc("/api/speedtest/status", s.handleSpeedtestStatus)
	s.mux.HandleFunc("/api/tests/settings", s.handleTestsSettings)
	s.mux.Handle("/api/tests/run", s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleCustomTests)))
	s.mux.HandleFunc("/api/iperf/info", s.handleIperfInfo)
	s.mux.Handle("/api/iperf/client", s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleIperfClient)))
	s.mux.HandleFunc("/api/iperf/client/status", s.handleIperfClientStatus)
	s.mux.HandleFunc("/api/iperf/server", s.handleIperfServer)
	s.mux.HandleFunc("/api/iperf/server/status", s.handleIperfServerStatus)
	s.mux.HandleFunc("/api/iperf/suggestions", s.handleIperfSuggestions)
	s.mux.HandleFunc("/api/devices", s.handleDevices)
	s.mux.Handle("/api/devices/scan", s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleDevicesScan)))
	s.mux.HandleFunc("/api/devices/status", s.handleDevicesStatus)
	s.mux.HandleFunc("/api/devices/settings", s.handleDevicesSettings)
	s.mux.HandleFunc("/api/devices/subnets", s.handleDevicesSubnets)
	s.mux.HandleFunc("/api/discovery/profile", s.handleDiscoveryProfile)
	s.mux.HandleFunc("/api/discovery/service/status", s.handleDiscoveryServiceStatus)
	s.mux.HandleFunc("/api/discovery/fingerprint", s.handleAdvancedFingerprint)
	s.mux.HandleFunc("/api/publicip", s.handlePublicIP)
	s.mux.HandleFunc("/api/logs", s.handleLogs)
	s.mux.HandleFunc("/api/health", s.handleHealth)             // Simple liveness check (fixes #540)
	s.mux.HandleFunc("/api/system/health", s.handleSystemHealth)

	// WiFi Survey routes
	s.mux.HandleFunc("/api/survey/create", s.createSurvey)
	s.mux.HandleFunc("/api/survey/list", s.listSurveys)
	s.mux.HandleFunc("/api/survey", s.getSurvey)
	s.mux.HandleFunc("/api/survey/delete", s.deleteSurvey)
	s.mux.HandleFunc("/api/survey/start", s.startSurvey)
	s.mux.HandleFunc("/api/survey/pause", s.pauseSurvey)
	s.mux.HandleFunc("/api/survey/complete", s.completeSurvey)
	s.mux.HandleFunc("/api/survey/sample", s.addSurveySample)

	// Vulnerability scanner routes
	s.mux.HandleFunc("/api/vulnerabilities/scan", s.handleVulnerabilityScan)
	s.mux.HandleFunc("/api/vulnerabilities/status", s.handleVulnerabilityStatus)
	s.mux.HandleFunc("/api/vulnerabilities/results", s.handleVulnerabilityResults)
	s.mux.HandleFunc("/api/vulnerabilities/device", s.handleDeviceVulnerabilities)
	s.mux.HandleFunc("/api/vulnerabilities/settings", s.handleVulnerabilitySettings)
	s.mux.HandleFunc("/api/survey/floorplan", s.updateSurveyFloorPlan)

	// Setup routes (no auth required for initial setup)
	s.mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("/api/setup/complete", s.handleSetupComplete)

	// WebSocket
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Static files (frontend) - use embedded FS in production, filesystem in dev
	frontendFS, err := web.GetFS()
	if err != nil {
		log.Printf("Warning: Failed to get embedded frontend FS: %v, falling back to disk", err)
		s.mux.Handle("/", http.FileServer(http.Dir("web/dist")))
	} else {
		log.Printf("Serving frontend from embedded filesystem (embedded=%v)", web.IsEmbedded())
		s.mux.Handle("/", spaHandler(http.FS(frontendFS)))
	}
}

// securityHeadersMiddleware adds security headers to all responses.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HSTS (HTTP Strict Transport Security) - only set over HTTPS
		if r.TLS != nil {
			// max-age=31536000 (1 year), includeSubDomains
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		// Prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// Prevent MIME sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// XSS protection (legacy header, but doesn't hurt)
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Referrer policy
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Content Security Policy - strict policy without unsafe-inline (fixes #532)
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers with origin validation.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Allow requests from same origin (no Origin header) or localhost for development
		if origin == "" || isAllowedOrigin(origin) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			// Cache preflight requests for 24 hours to reduce overhead (fixes #531)
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin checks if the origin is in the allowed list for CORS.
// Uses the same configurable origin checking as WebSocket.
func isAllowedOrigin(origin string) bool {
	return isAllowedWSOrigin(origin)
}

// spaHandler wraps a file server to support SPA (Single Page Application) routing.
// It serves index.html for any path that doesn't match a static file, enabling
// client-side routing in React/Vue/Angular apps.
// recoverMiddleware recovers from panics in HTTP handlers (fixes #519).
// Prevents a single panic from crashing the entire server.
func recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC in handler %s %s: %v\n%s",
					r.Method, r.URL.Path, err, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func spaHandler(fsys http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Normalize root path to index.html
		if path == "/" || path == "" {
			path = "/index.html"
		}

		// Try to open the file
		f, err := fsys.Open(path)
		if err != nil {
			// File doesn't exist - check if it's an API or WS route (shouldn't happen, but be safe)
			if strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws") {
				http.NotFound(w, r)
				return
			}
			// Serve index.html for SPA routing (client-side routes)
			path = "/index.html"
			f, err = fsys.Open(path)
			if err != nil {
				http.NotFound(w, r)
				return
			}
		}
		defer f.Close()

		// Check if it's a directory - serve index.html from it
		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		if stat.IsDir() {
			// Try to serve index.html from the directory
			indexPath := strings.TrimSuffix(path, "/") + "/index.html"
			f2, err := fsys.Open(indexPath)
			if err != nil {
				// No index.html in directory - serve root index.html
				path = "/index.html"
				f2, err = fsys.Open(path)
				if err != nil {
					http.NotFound(w, r)
					return
				}
			}
			f.Close()
			f = f2
			stat, err = f.Stat()
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		// Serve the file with proper content type
		rs, ok := f.(io.ReadSeeker)
		if !ok {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, stat.Name(), stat.ModTime(), rs)
	})
}

// Start starts the HTTP/HTTPS server.
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.config.Server.Port)

	// Apply middleware stack: panic recovery → security headers → CORS → auth (fixes #519)
	// Panic recovery is outermost to catch all panics
	handler := recoverMiddleware(securityHeadersMiddleware(corsMiddleware(s.authManager.Middleware(s.mux))))

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 5 * time.Minute, // Increased for large file downloads/exports (fixes #529)
		IdleTimeout:  60 * time.Second,
	}

	// WebSocket hub already running (started in NewServer to fix #512 race condition)
	// Start WebSocket broadcast loop
	s.startBroadcastLoop()

	// Start link state monitor
	if err := s.linkMonitor.Start(); err != nil {
		log.Printf("Warning: Link monitor failed to start: %v", err)
	} else {
		log.Printf("Link monitor started for %s (state: %s)",
			s.config.Interface.Default, s.linkMonitor.GetState())
	}

	// Start unified discovery service (applies profile-based configuration)
	if err := s.discoveryService.Start(); err != nil {
		log.Printf("Warning: Discovery service failed to start (may require root): %v", err)
	} else {
		status := s.discoveryService.GetStatus()
		log.Printf("Discovery service started with profile '%s' (methods: %v)",
			status.Profile, status.ActiveMethods)
	}

	// Legacy: Start protocol capture for backward compatibility
	if err := s.discoveryManager.Start(); err != nil {
		log.Printf("Warning: Discovery capture failed to start (may require root): %v", err)
	} else {
		log.Println("Discovery capture started")
	}

	// Start VLAN traffic monitor (requires root/CAP_NET_RAW)
	if err := s.vlanTrafficMonitor.Start(); err != nil {
		log.Printf("Warning: VLAN traffic monitor failed to start (may require root): %v", err)
	} else {
		log.Println("VLAN traffic monitor started")
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
func (s *Server) startHTTP() error {
	log.Printf("Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// startHTTPRedirect starts an HTTP server that redirects all requests to HTTPS (fixes #515).
// Properly tracks the server and allows shutdown.
func (s *Server) startHTTPRedirect(port int) {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Build HTTPS URL preserving the host and path
		host := r.Host
		// Remove port from host if present (to avoid localhost:80 → https://localhost:80:8443)
		if colonPos := strings.LastIndex(host, ":"); colonPos != -1 {
			host = host[:colonPos]
		}

		// If HTTPS is on standard port 443, don't include it in URL
		httpsPort := s.config.Server.Port
		var httpsURL string
		if httpsPort == 443 {
			httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
		} else {
			httpsURL = fmt.Sprintf("https://%s:%d%s", host, httpsPort, r.RequestURI)
		}

		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting HTTP→HTTPS redirect server on %s", addr)

	// Store redirect server for proper shutdown (fixes #515)
	s.redirectServer = &http.Server{
		Addr:         addr,
		Handler:      redirectHandler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	// Create error channel if not already created
	if s.redirectServerErr == nil {
		s.redirectServerErr = make(chan error, 1)
	}

	// Run server and report errors
	err := s.redirectServer.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("HTTP redirect server error: %v", err)
		s.redirectServerErr <- err
	}
}

// startHTTPS starts the server in HTTPS mode.
func (s *Server) startHTTPS() error {
	// Priority 1: ACME/Let's Encrypt automatic certificates
	if s.config.Server.ACME.Enabled {
		if s.config.Server.ACME.Domain == "" {
			return fmt.Errorf("ACME enabled but no domain specified")
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

	log.Printf("Starting HTTPS server on %s with TLS 1.3", s.httpServer.Addr)
	return s.httpServer.ListenAndServeTLS(certFile, keyFile)
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
		log.Println("ACME: Using Let's Encrypt STAGING server (certificates will not be trusted)")
	}

	// Configure TLS with ACME
	tlsConfig := manager.TLSConfig()
	tlsConfig.MinVersion = tls.VersionTLS13

	s.httpServer.TLSConfig = tlsConfig

	log.Printf("Starting HTTPS server with ACME on %s (domain: %s)",
		s.httpServer.Addr, s.config.Server.ACME.Domain)

	// Start HTTP-01 challenge handler on port 80
	// This is required for Let's Encrypt domain validation
	go func() {
		h := manager.HTTPHandler(nil)
		log.Printf("Starting HTTP-01 challenge handler on :80")
		if err := http.ListenAndServe(":80", h); err != nil {
			log.Printf("HTTP-01 handler error: %v", err)
		}
	}()

	// ListenAndServeTLS with empty cert/key paths uses GetCertificate from TLSConfig
	return s.httpServer.ListenAndServeTLS("", "")
}

// ensureSelfSignedCert generates a self-signed certificate if needed.
func (s *Server) ensureSelfSignedCert() (certFile, keyFile string, err error) {
	certsDir := "certs"
	certFile = filepath.Join(certsDir, "server.crt")
	keyFile = filepath.Join(certsDir, "server.key")

	// Check if certs already exist
	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			return certFile, keyFile, nil
		}
	}

	// Ensure certs directory exists
	if err := os.MkdirAll(certsDir, 0o700); err != nil {
		return "", "", err
	}

	// Generate private key with 4096-bit RSA (fixes #533)
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return "", "", err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"LuminetIQ"},
			CommonName:   "LuminetIQ Self-Signed",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "luminetiq.local"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return "", "", err
	}

	// Write certificate
	//nolint:gosec // G304: certFile is from config for TLS certificate location
	certOut, err := os.Create(certFile)
	if err != nil {
		return "", "", err
	}
	defer certOut.Close()
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); err != nil {
		return "", "", err
	}

	// Write private key
	//nolint:gosec // G304: keyFile is from config for TLS private key location
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", err
	}
	defer keyOut.Close()
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); err != nil {
		return "", "", err
	}

	log.Printf("Generated self-signed certificate: %s", certFile)
	return certFile, keyFile, nil
}

// Shutdown gracefully shuts down the server (fixes #515, #524).
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Shutdown HTTP redirect server if running (fixes #515)
	if s.redirectServer != nil {
		log.Println("Shutting down HTTP redirect server...")
		if err := s.redirectServer.Shutdown(ctx); err != nil {
			log.Printf("Error shutting down redirect server: %v", err)
		}
	}

	// Stop all services (fixes #524 - services will complete gracefully)
	log.Println("Stopping WebSocket hub...")
	s.wsHub.Shutdown()

	log.Println("Stopping link monitor...")
	s.linkMonitor.Stop()

	log.Println("Stopping discovery service...")
	s.discoveryService.Stop()

	log.Println("Stopping discovery manager...")
	s.discoveryManager.Stop()

	log.Println("Stopping VLAN traffic monitor...")
	s.vlanTrafficMonitor.Stop()

	log.Println("Stopping rate limiters...")
	s.loginRateLimiter.Stop()
	s.endpointRateLimiter.Stop()

	// Shutdown main HTTP server
	log.Println("Shutting down main HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

// Hub returns the WebSocket hub.
func (s *Server) Hub() *Hub {
	return s.wsHub
}
