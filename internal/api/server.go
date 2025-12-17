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
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/cable"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/dns"
	"github.com/krisarmstrong/seed/internal/gateway"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/publicip"
	"github.com/krisarmstrong/seed/internal/speedtest"
	"github.com/krisarmstrong/seed/internal/survey"
	"github.com/krisarmstrong/seed/internal/vlan"
	"github.com/krisarmstrong/seed/internal/wifi"
	"github.com/krisarmstrong/seed/web"
)

// indexHTMLPath is the path to the SPA entry point.
const indexHTMLPath = "/index.html"

// Server represents the HTTP/HTTPS server.
type Server struct {
	config              *config.Config
	configPath          string
	logPath             string
	httpServer          *http.Server
	authManager         *auth.Manager
	loginRateLimiter    *RateLimiter
	endpointRateLimiter *EndpointRateLimiter // Rate limiter for expensive endpoints (fixes #530)
	wsHub               *Hub
	logBroadcaster      *logging.LogBroadcaster // Log broadcaster for real-time log streaming
	mux                 *http.ServeMux
	netManager          *network.Manager
	linkMonitor         *network.LinkMonitor
	discoveryManager    *discovery.Manager         // Legacy: LLDP/CDP/EDP protocol capture
	deviceDiscovery     *discovery.DeviceDiscovery // Legacy: device aggregation
	discoveryService    *discovery.Service         // New unified discovery orchestrator
	dnsTester           *dns.Tester
	dhcpMonitor         *dhcp.Monitor
	rogueDetector       *dhcp.RogueDetector
	gatewayTester       *gateway.Tester
	vlanManager         *vlan.Manager
	vlanTrafficMonitor  *vlan.TrafficMonitor
	wifiManager         *wifi.Manager
	wifiScanner         *wifi.Scanner
	cableTester         *cable.Tester
	speedtestTester     *speedtest.Tester
	iperfManager        *iperf.Manager
	surveyManager       *survey.Manager
	vulnScanner         *discovery.VulnerabilityScanner
	publicipChecker     *publicip.Checker
	icmpAvailable       bool         // Whether raw ICMP sockets are available
	startTime           time.Time    // Application start time for uptime tracking (fixes #540)
	redirectServer      *http.Server // HTTP→HTTPS redirect server (fixes #515)
	redirectServerErr   chan error   // Error channel for redirect server
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
		discoveryManager:    discovery.NewManager(cfg.Interface.Default),
		deviceDiscovery:     discovery.NewDeviceDiscoveryWithOUI(cfg.Interface.Default, cfg.NetworkDiscovery.OUIFilePath, cfg.NetworkDiscovery.OUIMaxAge),
		discoveryService:    discovery.NewService(cfg, cfg.Interface.Default),
		dnsTester:           dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds()),
		dhcpMonitor:         dhcp.NewMonitor(cfg.Interface.Default),
		rogueDetector: dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
			Interface:        cfg.Interface.Default,
			KnownServers:     cfg.DHCP.RogueDetection.KnownServers,
			AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
		}),
		gatewayTester:      gateway.NewTester(gateway.DefaultThresholds()),
		vlanManager:        vlan.NewManager(cfg.Interface.Default),
		vlanTrafficMonitor: vlan.NewTrafficMonitor(cfg.Interface.Default),
		wifiManager:        wifi.NewManager(cfg.Interface.Default),
		wifiScanner:        wifi.NewScanner(cfg.Interface.Default),
		cableTester:        cable.NewTester(cfg.Interface.Default),
		speedtestTester:    speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID),
		iperfManager:       iperf.NewManager(),
		publicipChecker:    publicip.NewChecker(),
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
				slog.Warn("Failed to set additional subnets", "error", err)
			} else {
				slog.Info("Configured additional subnets for scanning", "count", len(enabledCIDRs))
			}
		}
	}

	// Initialize survey manager
	surveyStoragePath := "data/surveys"
	s.surveyManager = survey.NewManager(surveyStoragePath, s.wifiScanner, s.wifiManager, s.iperfManager)
	if err := s.surveyManager.LoadSurveys(); err != nil {
		slog.Warn("Failed to load surveys", "error", err)
	}

	// Initialize WebSocket hub (fixes #512)
	s.wsHub = NewHub()
	// Start hub before setupRoutes to prevent race condition
	go s.wsHub.Run()

	// Initialize log broadcaster for real-time log streaming
	s.logBroadcaster = logging.InitBroadcaster(1000) // Buffer last 1000 log entries
	s.logBroadcaster.SetBroadcaster(&logBroadcastAdapter{hub: s.wsHub})
	slog.Info("Log broadcaster initialized", "buffer_size", 1000)

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
			slog.Warn("Failed to initialize vulnerability scanner", "error", err)
		} else {
			s.vulnScanner = vulnScanner
			slog.Info("Vulnerability scanner initialized",
				"cve_database", scannerCfg.CVEDatabase, "threshold", scannerCfg.SeverityThreshold)
		}
	}

	// Configure security: allowed origins for CORS/WebSocket
	SetAllowedOrigins(cfg.Security.AllowedOrigins)
	if len(cfg.Security.AllowedOrigins) > 0 {
		slog.Info("Configured explicit allowed origins for CORS/WebSocket", "count", len(cfg.Security.AllowedOrigins))
	} else {
		slog.Info("Using default RFC 1918 private network origins for CORS/WebSocket")
	}

	// Setup routes (wsHub already initialized and running above)
	s.setupRoutes()

	return s
}

// onLinkStateChange handles link up/down events.
func (s *Server) onLinkStateChange(event network.LinkEvent) {
	slog.Info("Link state change", "interface", event.Interface, "state", event.State)

	switch event.State {
	case network.LinkStateUp:
		// Link came up - restart discovery to catch LLDP/CDP frames
		slog.Info("Link up - restarting discovery capture")
		s.discoveryManager.Stop()
		if err := s.discoveryManager.Start(); err != nil {
			slog.Warn("Failed to restart discovery", "error", err)
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
		slog.Info("Link down - notifying clients")
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]interface{}{
				"interface": event.Interface,
				"state":     "down",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})
	case network.LinkStateUnknown:
		// Unknown state - log but don't take action
		slog.Warn("Link state unknown", "interface", event.Interface)
	}
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	// API routes
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/auth/refresh", s.handleRefreshToken) // Token refresh (fixes #478)
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
	s.mux.HandleFunc("/api/wifi/scan", s.handleWiFiScan)
	s.mux.HandleFunc("/api/wifi/status", s.handleWiFiStatus)
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
	// Enhanced logging API endpoints (comprehensive logging enhancement)
	s.mux.HandleFunc("/api/logs/client", s.handleClientLogs) // Receive frontend logs
	s.mux.HandleFunc("/api/logs/query", s.handleLogsQuery)   // Query logs with filters
	s.mux.HandleFunc("/api/logs/stats", s.handleLogsStats)   // Get log statistics
	s.mux.HandleFunc("/api/logs/recent", s.handleLogsRecent) // Get recent logs
	s.mux.HandleFunc("/api/health", s.handleHealth)          // Simple liveness check (fixes #540)
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
	s.mux.HandleFunc("/api/survey/settings", s.updateSurveySettings)
	s.mux.HandleFunc("/api/survey/import/airmapper", s.importAirMapper)

	// Config backup/restore routes (implements #494)
	s.mux.HandleFunc("/api/config/backups", s.handleConfigBackups)
	s.mux.HandleFunc("/api/config/backup", s.handleConfigBackupCreate)
	s.mux.HandleFunc("/api/config/backup/delete", s.handleConfigBackupDelete)
	s.mux.HandleFunc("/api/config/restore", s.handleConfigRestore)
	s.mux.HandleFunc("/api/config/version", s.handleConfigVersion)

	// Setup routes (no auth required for initial setup)
	s.mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("/api/setup/complete", s.handleSetupComplete)

	// WebSocket
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Static files (frontend) - use embedded FS in production, filesystem in dev
	frontendFS, err := web.GetFS()
	if err != nil {
		slog.Warn("Failed to get embedded frontend FS, falling back to disk", "error", err)
		s.mux.Handle("/", http.FileServer(http.Dir("web/dist")))
	} else {
		slog.Info("Serving frontend from embedded filesystem", "embedded", web.IsEmbedded())
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
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
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

// Body size limits for different request types.
const (
	MaxBodySizeAuth      int64 = 1 * 1024         // 1KB for auth requests
	MaxBodySizeConfig    int64 = 64 * 1024        // 64KB for config updates
	MaxBodySizeJSON      int64 = 256 * 1024       // 256KB for general JSON
	MaxBodySizeFloorPlan int64 = 10 * 1024 * 1024 // 10MB for floor plan uploads
	MaxBodySizeAirMapper int64 = 50 * 1024 * 1024 // 50MB for AirMapper imports
	MaxBodySizeDefault   int64 = 1 * 1024 * 1024  // 1MB default
)

// bodyLimitMiddleware enforces request body size limits to prevent DoS attacks.
// Different endpoints have different limits based on expected payload size.
func bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip body limit for GET, HEAD, OPTIONS (no body expected)
		if r.Method == http.MethodGet ||
			r.Method == http.MethodHead ||
			r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		// Determine limit based on endpoint
		var limit int64
		path := r.URL.Path

		switch {
		case strings.HasPrefix(path, "/api/auth/"):
			limit = MaxBodySizeAuth
		case strings.HasPrefix(path, "/api/config/"):
			limit = MaxBodySizeConfig
		case path == "/api/survey/floorplan":
			limit = MaxBodySizeFloorPlan
		case path == "/api/survey/import/airmapper":
			limit = MaxBodySizeAirMapper
		case strings.HasPrefix(path, "/api/"):
			limit = MaxBodySizeJSON
		default:
			limit = MaxBodySizeDefault
		}

		// Wrap the body with a limit reader
		r.Body = http.MaxBytesReader(w, r.Body, limit)

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
				slog.Error("PANIC in handler",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err,
					"stack", string(debug.Stack()))
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
			path = indexHTMLPath
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
			path = indexHTMLPath
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
			indexPath := strings.TrimSuffix(path, "/") + indexHTMLPath
			f2, err := fsys.Open(indexPath)
			if err != nil {
				// No index.html in directory - serve root index.html
				path = indexHTMLPath
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

	// Apply middleware stack: panic recovery → request ID → logging → security headers → body limit → CORS → i18n → auth (fixes #519)
	// Panic recovery is outermost to catch all panics
	// Request ID middleware generates unique IDs for request correlation in logs
	// Logging middleware logs all HTTP requests with timing, status, and request IDs
	// Body limit middleware enforces request body size limits
	// i18n middleware extracts Accept-Language and attaches localizer to context
	handler := recoverMiddleware(
		logging.RequestIDMiddleware(
			logging.LoggingMiddleware(
				securityHeadersMiddleware(
					bodyLimitMiddleware(
						corsMiddleware(
							i18n.Middleware()(
								s.authManager.Middleware(s.mux))))))))

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
		slog.Warn("Link monitor failed to start", "error", err)
	} else {
		slog.Info("Link monitor started",
			"interface", s.config.Interface.Default,
			"state", s.linkMonitor.GetState())
	}

	// Start unified discovery service (applies profile-based configuration)
	if err := s.discoveryService.Start(); err != nil {
		slog.Warn("Discovery service failed to start (may require root)", "error", err)
	} else {
		status := s.discoveryService.GetStatus()
		slog.Info("Discovery service started",
			"profile", status.Profile,
			"methods", status.ActiveMethods)
	}

	// Legacy: Start protocol capture for backward compatibility
	if err := s.discoveryManager.Start(); err != nil {
		slog.Warn("Discovery capture failed to start (may require root)", "error", err)
	} else {
		slog.Info("Discovery capture started")
	}

	// Start VLAN traffic monitor (requires root/CAP_NET_RAW)
	if err := s.vlanTrafficMonitor.Start(); err != nil {
		slog.Warn("VLAN traffic monitor failed to start (may require root)", "error", err)
	} else {
		slog.Info("VLAN traffic monitor started")
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
	slog.Info("Starting HTTP server", "addr", s.httpServer.Addr)
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
	slog.Info("Starting HTTP→HTTPS redirect server", "addr", addr)

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
		slog.Error("HTTP redirect server error", "error", err)
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

	slog.Info("Starting HTTPS server", "addr", s.httpServer.Addr, "tls_version", "1.3")
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
		slog.Warn("ACME: Using Let's Encrypt STAGING server (certificates will not be trusted)")
	}

	// Configure TLS with ACME
	tlsConfig := manager.TLSConfig()
	tlsConfig.MinVersion = tls.VersionTLS13

	s.httpServer.TLSConfig = tlsConfig

	slog.Info("Starting HTTPS server with ACME",
		"addr", s.httpServer.Addr,
		"domain", s.config.Server.ACME.Domain)

	// Start HTTP-01 challenge handler on port 80
	// This is required for Let's Encrypt domain validation
	go func() {
		h := manager.HTTPHandler(nil)
		slog.Info("Starting HTTP-01 challenge handler", "addr", ":80")
		// HTTP-01 handler only serves ACME challenges, timeouts not critical
		challengeServer := &http.Server{
			Addr:              ":80",
			Handler:           h,
			ReadHeaderTimeout: 10 * time.Second,
		}
		if err := challengeServer.ListenAndServe(); err != nil {
			slog.Error("HTTP-01 handler error", "error", err)
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
			Organization: []string{"The Seed"},
			CommonName:   "The Seed Self-Signed",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost", "seed.local"},
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

	slog.Info("Generated self-signed certificate", "cert_file", certFile)
	return certFile, keyFile, nil
}

// Shutdown gracefully shuts down the server (fixes #515, #524).
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down server...")

	// Shutdown HTTP redirect server if running (fixes #515)
	if s.redirectServer != nil {
		slog.Info("Shutting down HTTP redirect server...")
		if err := s.redirectServer.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down redirect server", "error", err)
		}
	}

	// Stop all services (fixes #524 - services will complete gracefully)
	slog.Info("Stopping WebSocket hub...")
	s.wsHub.Shutdown()

	slog.Info("Stopping link monitor...")
	s.linkMonitor.Stop()

	slog.Info("Stopping discovery service...")
	s.discoveryService.Stop()

	slog.Info("Stopping discovery manager...")
	s.discoveryManager.Stop()

	slog.Info("Stopping VLAN traffic monitor...")
	s.vlanTrafficMonitor.Stop()

	slog.Info("Stopping rate limiters...")
	s.loginRateLimiter.Stop()
	s.endpointRateLimiter.Stop()

	// Shutdown main HTTP server
	slog.Info("Shutting down main HTTP server...")
	return s.httpServer.Shutdown(ctx)
}

// Hub returns the WebSocket hub.
func (s *Server) Hub() *Hub {
	return s.wsHub
}
