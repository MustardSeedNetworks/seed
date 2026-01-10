// Package httpapi provides the HTTP/WebSocket server.
package httpapi

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
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/oauth"
	"github.com/krisarmstrong/seed/internal/paths"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
	"github.com/krisarmstrong/seed/ui"
)

// indexHTMLPath is the path to the SPA entry point.
const indexHTMLPath = "/index.html"

// Server configuration constants.
const (
	// logBroadcasterBufferSize is the buffer size for log broadcaster entries.
	logBroadcasterBufferSize = 1000

	// httpsDefaultPort is the standard HTTPS port number.
	httpsDefaultPort = 443

	// rsaKeyBits is the RSA key size in bits for self-signed certificates.
	rsaKeyBits = 4096

	// serverReadTimeoutSec is the HTTP server read timeout in seconds.
	serverReadTimeoutSec = 15

	// serverWriteTimeoutMin is the HTTP server write timeout in minutes for large file transfers.
	serverWriteTimeoutMin = 5

	// serverIdleTimeoutSec is the HTTP server idle connection timeout in seconds.
	serverIdleTimeoutSec = 60

	// redirectReadWriteTimeoutSec is the timeout for HTTP redirect server operations.
	redirectReadWriteTimeoutSec = 5

	// acmeReadHeaderTimeoutSec is the timeout for reading ACME challenge request headers.
	acmeReadHeaderTimeoutSec = 10

	// setupModeTimeoutMin is how long setup mode remains active (security fix #891).
	// After this duration, setup is disabled and server restart is required.
	setupModeTimeoutMin = 15

	// retentionAlertsMultiplier is the multiplier for alerts retention (keep alerts longer).
	retentionAlertsMultiplier = 2

	// retentionAuditLogMultiplier is the multiplier for audit log retention (keep longest).
	retentionAuditLogMultiplier = 3

	// retentionInactiveDeviceMultiplier is the multiplier for inactive device retention.
	retentionInactiveDeviceMultiplier = 4
)

// Server represents the HTTP/HTTPS server.
type Server struct {
	config              *config.Config
	configPath          string
	logPath             string
	httpServer          *http.Server
	authManager         *auth.Manager
	csrfManager         *auth.CSRFManager          // CSRF token manager for state-changing requests (fixes contract review)
	setupTokenManager   *SetupTokenManager         // Setup token manager for secure initial setup (fixes #724, #758)
	recoveryManager     *auth.RecoveryTokenManager // Password recovery for headless machines
	loginRateLimiter    *RateLimiter
	endpointRateLimiter *EndpointRateLimiter // Rate limiter for expensive endpoints (fixes #530)
	wsHub               *Hub
	logBroadcaster      *logging.LogBroadcaster // Log broadcaster for real-time log streaming
	mux                 *http.ServeMux
	netManager          *network.Manager
	linkMonitor         *network.LinkMonitor
	deviceDiscovery     *discovery.DeviceDiscovery // Device aggregation (used by Service and Pipeline)
	discoveryService    *discovery.Service         // Unified discovery orchestrator
	dnsTester           *dns.Tester
	dnsSecurityScanner  *dns.SecurityScanner
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
	oauthManager        *oauth.Manager      // OAuth SSO provider manager
	db                  *database.DB        // SQLite database for persistence (#755)
	icmpAvailable       bool                // Whether raw ICMP sockets are available
	startTime           time.Time           // Application start time for uptime tracking (fixes #540)
	redirectServer      *http.Server        // HTTP→HTTPS redirect server (fixes #515)
	redirectServerErr   chan error          // Error channel for redirect server
	trustedProxies      *TrustedProxies     // Trusted proxy IPs for X-Forwarded-For handling (#H4)
	pipeline            *discovery.Pipeline // Phased discovery pipeline orchestrator
	acmeChallengeServer *http.Server        // HTTP-01 challenge server for ACME (fixes #837)
	retentionStopCh     chan struct{}       // Signals data retention goroutine to stop (fixes #848)
	modules             *Modules            // Application modules (Sap, Shell, Canopy, Roots, Harvest)
	setupModeStartTime  time.Time           // Security fix #891: Track when setup mode started
}

// NewServer creates a new server instance.
func NewServer(
	cfg *config.Config,
	configPath, logPath string,
	netMgr *network.Manager,
	icmpAvailable bool,
	trustedProxies *TrustedProxies,
	db *database.DB,
	modules *Modules,
) *Server {
	s := &Server{
		config:         cfg,
		configPath:     configPath,
		logPath:        logPath,
		mux:            http.NewServeMux(),
		netManager:     netMgr,
		icmpAvailable:  icmpAvailable,
		trustedProxies: trustedProxies, // Trusted proxy support (#H4)
		startTime:      time.Now(),     // Track application start time (fixes #540)
		db:             db,             // Database passed from cmd_serve.go
		modules:        modules,        // Modules passed from cmd_serve.go
		authManager: auth.NewManager(
			cfg.Auth.JWTSecret,
			cfg.Auth.SessionTimeout,
			cfg.Auth.DefaultUsername,
			cfg.Auth.DefaultPasswordHash,
		),
		csrfManager:      auth.NewCSRFManager(), // CSRF protection for state-changing requests
		loginRateLimiter: NewRateLimiter(DefaultRateLimitConfig()),
		endpointRateLimiter: NewEndpointRateLimiter(
			DefaultEndpointRateLimitConfig(),
		), // Rate limit expensive endpoints (fixes #530)
		setupTokenManager: NewSetupTokenManager(), // Setup token for secure initial setup (fixes #724, #758)
		recoveryManager:   auth.NewRecoveryTokenManager(paths.Resolve(paths.ModeAuto).DataDir),
		linkMonitor:       network.NewLinkMonitor(cfg.Interface.Default),
		deviceDiscovery: discovery.NewDeviceDiscoveryWithOUI(
			cfg.Interface.Default,
			cfg.NetworkDiscovery.OUIFilePath,
			cfg.NetworkDiscovery.OUIMaxAge,
		),
		// Note: discoveryService is initialized after profiler is created (see below)
		dnsTester:          dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds()),
		dnsSecurityScanner: dns.NewSecurityScanner(dns.DefaultSecurityScanConfig()),
		dhcpMonitor:        dhcp.NewMonitor(cfg.Interface.Default),
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

	// Security fix #891: Record setup mode start time
	if auth.IsDefaultPasswordHash(cfg.Auth.DefaultPasswordHash) {
		s.setupModeStartTime = time.Now()
	}

	// Set up link state change callback
	s.linkMonitor.OnStateChange(s.onLinkStateChange)

	// Initialize network services (DNS, device discovery subnets, survey manager)
	s.initNetworkServices(cfg)

	// Initialize OAuth manager for SSO
	s.initOAuthManager()

	// Configure database-backed services if db was passed in
	s.initDatabaseServices(cfg, db)

	// Initialize WebSocket hub and log broadcaster
	s.initWebSocketAndLogging(db)

	// Initialize discovery service and pipeline
	s.initDiscoveryPipeline(cfg)

	// Initialize vulnerability scanner if enabled
	s.initVulnerabilityScanner(cfg)

	// Configure security: allowed origins for CORS/WebSocket
	s.initSecurityOrigins(cfg)

	// Setup routes (wsHub already initialized and running above)
	s.setupRoutes()

	return s
}

// initNetworkServices initializes DNS servers, device discovery subnets, and survey manager.
func (s *Server) initNetworkServices(cfg *config.Config) {
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
	s.initAdditionalSubnets(cfg)

	// Initialize survey manager
	surveyStoragePath := "data/surveys"
	s.surveyManager = survey.NewManager(
		surveyStoragePath,
		s.wifiScanner,
		s.wifiManager,
		s.iperfManager,
	)
	if err := s.surveyManager.LoadSurveys(); err != nil {
		logging.GetLogger().Warn("Failed to load surveys", "error", err)
	}
}

// initAdditionalSubnets configures device discovery with additional subnets from config.
func (s *Server) initAdditionalSubnets(cfg *config.Config) {
	if len(cfg.NetworkDiscovery.AdditionalSubnets) == 0 {
		return
	}

	enabledCIDRs := s.collectEnabledSubnets(cfg)
	if len(enabledCIDRs) == 0 {
		return
	}

	if err := s.deviceDiscovery.SetAdditionalSubnets(enabledCIDRs); err != nil {
		logging.GetLogger().Warn("Failed to set additional subnets", "error", err)
		return
	}

	logging.GetLogger().Info("Configured additional subnets for scanning", "count", len(enabledCIDRs))
}

// collectEnabledSubnets extracts enabled subnet CIDRs from configuration.
func (s *Server) collectEnabledSubnets(cfg *config.Config) []string {
	enabledCIDRs := make([]string, 0, len(cfg.NetworkDiscovery.AdditionalSubnets))
	for _, subnet := range cfg.NetworkDiscovery.AdditionalSubnets {
		if subnet.Enabled {
			enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
		}
	}
	return enabledCIDRs
}

// initDatabaseServices configures database-backed services if db is available.
func (s *Server) initDatabaseServices(cfg *config.Config, db *database.DB) {
	if db == nil {
		return
	}

	// Set up database-backed user store for authentication
	userStore := database.NewUserStoreAdapter(db)
	s.authManager.SetUserStore(userStore)

	// Migrate admin user from config to database if needed
	// This ensures backward compatibility during the transition
	if cfg.Auth.DefaultPasswordHash != "" &&
		cfg.Auth.DefaultPasswordHash != auth.SetupModePlaceholder {
		if err := userStore.MigrateUserFromConfig(
			context.Background(),
			cfg.Auth.DefaultUsername,
			cfg.Auth.DefaultPasswordHash,
		); err != nil {
			logging.GetLogger().Error("Failed to migrate user from config", "error", err)
		} else {
			logging.GetLogger().Info("User migrated from config to database", "username", cfg.Auth.DefaultUsername)
		}
	}

	// Start data retention cleanup in background (fixes #848)
	if cfg.Database.RetentionDays > 0 {
		s.retentionStopCh = make(chan struct{})
		go s.startDataRetention(cfg.Database.RetentionDays)
	}
}

// initWebSocketAndLogging initializes the WebSocket hub and log broadcaster.
func (s *Server) initWebSocketAndLogging(db *database.DB) {
	// Initialize WebSocket hub (fixes #512)
	s.wsHub = NewHub()
	// Start hub before setupRoutes to prevent race condition
	go s.wsHub.Run()

	// Initialize log broadcaster for real-time log streaming
	s.logBroadcaster = logging.InitBroadcaster(logBroadcasterBufferSize)
	s.logBroadcaster.SetBroadcaster(&logBroadcastAdapter{hub: s.wsHub})

	// Wire up database persistence for logs if database is available
	if db != nil {
		s.logBroadcaster.SetDBWriter(&dbLogWriterAdapter{db: db})
		logging.GetLogger().
			Info("Log broadcaster initialized with database persistence", "buffer_size", logBroadcasterBufferSize)
	} else {
		logging.GetLogger().Info(
			"Log broadcaster initialized (memory-only, no database)",
			"buffer_size",
			logBroadcasterBufferSize,
		)
	}

	// Wire up database persistence for devices if database is available
	if db != nil {
		s.deviceDiscovery.SetDBWriter(&dbDeviceWriterAdapter{db: db})
		logging.GetLogger().Info("Device discovery initialized with database persistence")
	}
}

// initDiscoveryPipeline initializes the discovery service and pipeline.
func (s *Server) initDiscoveryPipeline(cfg *config.Config) {
	// Create SHARED DeviceProfiler - used by both Service and Pipeline
	// This ensures port scan results and SNMP data are consistent across the system
	sharedProfiler := discovery.NewDeviceProfiler(discovery.DefaultProfilerConfig(), &cfg.SNMP)

	// Initialize discovery service with the shared profiler
	s.discoveryService = discovery.NewService(cfg, cfg.Interface.Default, sharedProfiler)
	logging.GetLogger().Info("Discovery service initialized with shared profiler")

	// Initialize discovery pipeline with the SAME shared profiler
	pipelineCfg := discovery.PipelineConfigFromAdapter(&cfg.Pipeline)
	s.pipeline = discovery.NewPipeline(
		&pipelineCfg,
		s.deviceDiscovery,
		sharedProfiler, // Use the same profiler as Service
		&pipelineBroadcastAdapter{hub: s.wsHub},
	)

	// Link Service and Pipeline for coordination
	s.discoveryService.SetPipeline(s.pipeline)

	// Set up pipeline completion callback to sync results back to service
	s.discoveryService.SetOnPipelineComplete(func(devices []*discovery.DiscoveredDevice) {
		logging.GetLogger().Info(
			"Pipeline completed, syncing results to discovery service",
			"device_count",
			len(devices),
		)
	})

	logging.GetLogger().Info("Discovery pipeline initialized",
		"phases_enabled", s.pipeline.GetEnabledPhaseNames(),
		"port_scan_intensity", cfg.Pipeline.PortScan.Intensity,
		"shared_profiler", true)
}

// initVulnerabilityScanner initializes the vulnerability scanner if enabled.
func (s *Server) initVulnerabilityScanner(cfg *config.Config) {
	if !cfg.Security.VulnerabilityScanning.Enabled {
		return
	}

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
		logging.GetLogger().Warn("Failed to initialize vulnerability scanner", "error", err)
		return
	}
	s.vulnScanner = vulnScanner
	logging.GetLogger().Info("Vulnerability scanner initialized",
		"cve_database", scannerCfg.CVEDatabase, "threshold", scannerCfg.SeverityThreshold)
}

// initSecurityOrigins configures allowed origins for CORS/WebSocket.
func (s *Server) initSecurityOrigins(cfg *config.Config) {
	getWSState().setAllowedOrigins(cfg.Security.AllowedOrigins)

	if len(cfg.Security.AllowedOrigins) == 0 {
		logging.GetLogger().Info("Using default RFC 1918 private network origins for CORS/WebSocket")
		return
	}

	// Check for wildcard origin in production mode (fixes #715)
	// Production mode is inferred from HTTPS being enabled
	s.logWildcardOriginWarning(cfg)

	logging.GetLogger().Info(
		"Configured explicit allowed origins for CORS/WebSocket",
		"count",
		len(cfg.Security.AllowedOrigins),
	)
}

// logWildcardOriginWarning logs appropriate warnings for wildcard origin configuration.
func (s *Server) logWildcardOriginWarning(cfg *config.Config) {
	if !slices.Contains(cfg.Security.AllowedOrigins, "*") {
		return
	}

	if cfg.Server.HTTPS {
		logging.GetLogger().Warn(
			"SECURITY WARNING: Wildcard origin (*) allows all origins in production mode with HTTPS enabled",
			"recommendation",
			"Configure explicit allowed origins in Security.AllowedOrigins for production deployments",
		)
		return
	}

	logging.GetLogger().Info("Wildcard origin (*) configured - allows all origins (development mode)",
		"warning", "Not recommended for production use")
}

// getClientIP extracts the client IP from a request, considering trusted proxies.
// If trusted proxies are configured and the request comes from one, uses X-Forwarded-For.
// Otherwise, uses RemoteAddr (the only secure option).
func (s *Server) getClientIP(r *http.Request) string {
	return GetClientIPWithTrustedProxies(r, s.trustedProxies)
}

// onLinkStateChange handles link up/down events.
func (s *Server) onLinkStateChange(event network.LinkEvent) {
	logging.GetLogger().
		Info("Link state change", "interface", event.Interface, "state", event.State)

	switch event.State {
	case network.LinkStateUp:
		// Link came up - reload discovery service to restart protocol capture
		logging.GetLogger().Info("Link up - reloading discovery service")
		if err := s.discoveryService.Reload(); err != nil {
			logging.GetLogger().Warn("Failed to reload discovery service", "error", err)
		}

		// Notify WebSocket clients with linkState message
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]any{
				"interface": event.Interface,
				"state":     "up",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})

		// Also broadcast link card update immediately to trigger frontend auto-run tests.
		// The frontend listens for card_update messages on the "link" card to detect
		// link-up transitions and run speedtest/iperf tests.
		// Multi-interface support (#754): Include interface in broadcast.
		if linkData := s.collectLinkData(); linkData != nil {
			s.wsHub.BroadcastCardUpdateForInterface("link", linkData, event.Interface)
		}
	case network.LinkStateDown:
		// Link went down - notify clients
		logging.GetLogger().Info("Link down - notifying clients")
		s.wsHub.Broadcast(Message{
			Type: "linkState",
			Payload: map[string]any{
				"interface": event.Interface,
				"state":     "down",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		})

		// Also broadcast link card update for proper state tracking.
		// Frontend uses this to track DOWN state for detecting DOWN→UP transitions.
		// Multi-interface support (#754): Include interface in broadcast.
		if linkData := s.collectLinkData(); linkData != nil {
			s.wsHub.BroadcastCardUpdateForInterface("link", linkData, event.Interface)
		}
	case network.LinkStateUnknown:
		// Unknown state - log but don't take action
		logging.GetLogger().Warn("Link state unknown", "interface", event.Interface)
	}
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	s.setupCoreRoutes()
	s.setupSAPRoutes()
	s.setupShellRoutes()
	s.setupRootsRoutes()
	s.setupCanopyRoutes()
	s.setupHarvestRoutes()
	s.setupWebSocketAndStatic()
}

// setupCoreRoutes registers auth, settings, config, and setup routes.
func (s *Server) setupCoreRoutes() {
	s.mux.HandleFunc("/api/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/auth/refresh", s.handleRefreshToken)
	s.mux.HandleFunc("/api/auth/csrf", s.handleCSRFToken)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/settings", s.handleSettings)
	s.mux.HandleFunc("/api/settings/defaults", s.handleSettingsDefaults)
	s.mux.HandleFunc("/api/settings/link", s.handleLinkSettings)
	s.mux.HandleFunc("/api/settings/cable", s.handleCableTestSettings)
	s.mux.HandleFunc("/api/interfaces", s.handleInterfaces)
	s.mux.HandleFunc("/api/interface", s.handleInterface)
	s.mux.HandleFunc("/api/network/mtu", s.handleSetMTU)
	s.mux.HandleFunc("/api/config/backups", s.handleConfigBackups)
	s.mux.HandleFunc("/api/config/backup", s.handleConfigBackupCreate)
	s.mux.HandleFunc("/api/config/backup/delete", s.handleConfigBackupDelete)
	s.mux.HandleFunc("/api/config/restore", s.handleConfigRestore)
	s.mux.HandleFunc("/api/config/version", s.handleConfigVersion)
	s.mux.HandleFunc("/api/profiles", s.handleProfiles)
	s.mux.HandleFunc("/api/profiles/active", s.handleActiveProfile)
	s.mux.HandleFunc("/api/profiles/import", s.handleImportProfiles)
	s.mux.HandleFunc("/api/profiles/export", s.handleExportProfiles)
	s.mux.HandleFunc("/api/profiles/", s.handleProfiles)
	s.mux.HandleFunc("/api/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc("/api/setup/complete", s.handleSetupComplete)
	s.mux.HandleFunc("/api/recovery/status", s.handleRecoveryStatus)
	s.mux.HandleFunc("/api/recovery/complete", s.handleRecoveryComplete)
	s.mux.HandleFunc("/api/recovery/instructions", s.handleRecoveryInstructions)
	s.mux.HandleFunc("/api/sso/providers", s.handleSSOProviders)
	s.mux.HandleFunc("/api/sso/login", s.handleSSOLogin)
	s.mux.HandleFunc("/api/sso/callback", s.handleSSOCallback)
	s.mux.HandleFunc("/api/sso/settings", s.handleSSOSettings)
	s.mux.HandleFunc("/api/sso/update", s.handleSSOUpdate)
	s.mux.HandleFunc("/api/health", s.handleHealth)
}

// setupSAPRoutes registers SAP module routes (live telemetry).
func (s *Server) setupSAPRoutes() {
	s.mux.HandleFunc("/api/sap/link", s.handleLink)
	s.mux.HandleFunc("/api/sap/cable", s.handleCable)
	s.mux.HandleFunc("/api/sap/dns", s.handleDNS)
	s.mux.HandleFunc("/api/sap/dns/security", s.handleDNSSecurity)
	s.mux.HandleFunc("/api/sap/dns/security/settings", s.handleDNSSecuritySettings)
	s.mux.HandleFunc("/api/sap/gateway", s.handleGateway)
	s.mux.HandleFunc("/api/sap/dhcp/rogue", s.handleRogueDHCP)
	s.mux.HandleFunc("/api/sap/dhcp/rogue/servers", s.handleRogueDHCPServers)
	s.mux.HandleFunc("/api/sap/dhcp/rogue/config", s.handleRogueDHCPConfig)
	s.mux.HandleFunc("/api/sap/vlan", s.handleVLAN)
	s.mux.HandleFunc("/api/sap/vlan/traffic", s.handleVLANTraffic)
	s.mux.HandleFunc("/api/sap/vlan/interface", s.handleVLANInterface)
	s.mux.Handle(
		"/api/sap/speedtest",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleSpeedtest)),
	)
	s.mux.HandleFunc("/api/sap/speedtest/status", s.handleSpeedtestStatus)
	s.mux.HandleFunc("/api/sap/iperf/info", s.handleIperfInfo)
	s.mux.Handle(
		"/api/sap/iperf/client",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleIperfClient)),
	)
	s.mux.HandleFunc("/api/sap/iperf/client/status", s.handleIperfClientStatus)
	s.mux.HandleFunc("/api/sap/iperf/server", s.handleIperfServer)
	s.mux.HandleFunc("/api/sap/iperf/server/status", s.handleIperfServerStatus)
	s.mux.HandleFunc("/api/sap/iperf/suggestions", s.handleIperfSuggestions)
	s.mux.HandleFunc("/api/sap/health-checks/settings", s.handleHealthChecksSettings)
	s.mux.Handle(
		"/api/sap/health-checks/run",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleHealthChecks)),
	)
	s.mux.HandleFunc("/api/sap/snmp/settings", s.handleSNMPSettings)
	s.mux.HandleFunc("/api/sap/system/health", s.handleSystemHealth)
	s.mux.HandleFunc("/api/sap/ipconfig", s.handleIPConfig)
	s.mux.HandleFunc("/api/sap/ipconfig/settings", s.handleIPSettings)
	s.mux.HandleFunc("/api/sap/publicip", s.handlePublicIP)
}

// setupShellRoutes registers Shell module routes (security posture).
func (s *Server) setupShellRoutes() {
	s.mux.HandleFunc("/api/shell/discovery", s.handleDiscovery)
	s.mux.HandleFunc("/api/shell/discovery/probe", s.handleTCPProbe)
	s.mux.HandleFunc("/api/shell/discovery/portscan", s.handlePortScan)
	s.mux.HandleFunc("/api/shell/discovery/options", s.handleDiscoveryOptions)
	s.mux.HandleFunc("/api/shell/discovery/service/status", s.handleDiscoveryServiceStatus)
	s.mux.HandleFunc("/api/shell/discovery/fingerprint", s.handleAdvancedFingerprint)
	s.mux.HandleFunc("/api/shell/devices", s.handleDevices)
	s.mux.Handle(
		"/api/shell/devices/scan",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleDevicesScan)),
	)
	s.mux.HandleFunc("/api/shell/devices/status", s.handleDevicesStatus)
	s.mux.HandleFunc("/api/shell/devices/settings", s.handleDevicesSettings)
	s.mux.HandleFunc("/api/shell/devices/subnets", s.handleDevicesSubnets)
	s.mux.Handle(
		"/api/shell/vulnerabilities/scan",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleVulnerabilityScan)),
	)
	s.mux.HandleFunc("/api/shell/vulnerabilities/status", s.handleVulnerabilityStatus)
	s.mux.HandleFunc("/api/shell/vulnerabilities/results", s.handleVulnerabilityResults)
	s.mux.HandleFunc("/api/shell/vulnerabilities/device", s.handleDeviceVulnerabilities)
	s.mux.HandleFunc("/api/shell/vulnerabilities/settings", s.handleVulnerabilitySettings)
	s.mux.HandleFunc("/api/shell/vulnerabilities/validate-api-key", s.handleNVDAPIKeyValidate)
	s.mux.HandleFunc("/api/shell/pipeline/status", s.handlePipelineStatus)
	s.mux.HandleFunc("/api/shell/pipeline/start", s.handlePipelineStart)
	s.mux.HandleFunc("/api/shell/pipeline/cancel", s.handlePipelineCancel)
	s.mux.HandleFunc("/api/shell/pipeline/config", s.handlePipelineConfigRoute)
	s.mux.HandleFunc("/api/shell/pipeline/port-intensity", s.handlePipelinePortIntensityInfo)
	s.mux.HandleFunc("/api/shell/pipeline/timing-profiles", s.handlePipelineTimingProfiles)
}

// setupRootsRoutes registers Roots module routes (path analysis).
func (s *Server) setupRootsRoutes() {
	s.mux.Handle(
		"/api/roots/traceroute",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.handleTraceroute)),
	)
	s.mux.HandleFunc("/api/roots/path", s.handlePath)
}

// setupCanopyRoutes registers Canopy module routes (Wi-Fi planning).
func (s *Server) setupCanopyRoutes() {
	s.mux.HandleFunc("/api/canopy/wifi", s.handleWiFi)
	s.mux.HandleFunc("/api/canopy/wifi/scan", s.handleWiFiScan)
	s.mux.HandleFunc("/api/canopy/wifi/status", s.handleWiFiStatus)
	s.mux.HandleFunc("/api/canopy/wifi/channel-graph", s.handleWiFiChannelGraph)
	s.mux.HandleFunc("/api/canopy/wifi/settings", s.handleWiFiSettings)
	s.mux.HandleFunc("/api/canopy/wifi/connect", s.handleWiFiConnect)
	s.mux.HandleFunc("/api/canopy/wifi/disconnect", s.handleWiFiDisconnect)
	s.mux.HandleFunc("/api/canopy/wifi/saved", s.handleWiFiSavedNetworks)
	s.mux.HandleFunc("/api/canopy/wifi/forget", s.handleWiFiForgetNetwork)
	s.mux.HandleFunc("/api/canopy/survey/create", s.createSurvey)
	s.mux.HandleFunc("/api/canopy/survey/list", s.listSurveys)
	s.mux.HandleFunc("/api/canopy/survey", s.getSurvey)
	s.mux.HandleFunc("/api/canopy/survey/delete", s.deleteSurvey)
	s.mux.HandleFunc("/api/canopy/survey/start", s.startSurvey)
	s.mux.HandleFunc("/api/canopy/survey/pause", s.pauseSurvey)
	s.mux.HandleFunc("/api/canopy/survey/complete", s.completeSurvey)
	s.mux.HandleFunc("/api/canopy/survey/sample", s.addSurveySample)
	s.mux.HandleFunc("/api/canopy/survey/floorplan", s.updateSurveyFloorPlan)
	s.mux.HandleFunc("/api/canopy/survey/settings", s.updateSurveySettings)
	s.mux.Handle(
		"/api/canopy/survey/import/airmapper",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.importAirMapper)),
	)
	s.mux.Handle(
		"/api/canopy/survey/heatmap",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.getSurveyHeatmap)),
	)
	s.mux.HandleFunc("/api/canopy/survey/dead-zones", s.getSurveyDeadZones)
	s.mux.HandleFunc("/api/canopy/survey/floors", s.handleSurveyFloors)
	s.mux.HandleFunc("/api/canopy/survey/floor", s.handleSurveyFloor)
	s.mux.HandleFunc("/api/canopy/survey/floor/floorplan", s.updateFloorFloorPlan)
	s.mux.HandleFunc("/api/canopy/survey/floor/sample", s.addFloorSample)
	s.mux.HandleFunc("/api/canopy/survey/active-floor", s.setActiveFloor)
	s.mux.Handle(
		"/api/canopy/survey/report",
		s.endpointRateLimiter.RateLimitMiddleware(http.HandlerFunc(s.generateSurveyReport)),
	)
}

// setupHarvestRoutes registers Harvest module routes (reporting).
func (s *Server) setupHarvestRoutes() {
	s.mux.HandleFunc("/api/harvest/export", s.handleExport)
	s.mux.HandleFunc("/api/harvest/logs", s.handleLogs)
	s.mux.HandleFunc("/api/harvest/logs/client", s.handleClientLogs)
	s.mux.HandleFunc("/api/harvest/logs/query", s.handleLogsQuery)
	s.mux.HandleFunc("/api/harvest/logs/stats", s.handleLogsStats)
	s.mux.HandleFunc("/api/harvest/logs/recent", s.handleLogsRecent)
}

// setupWebSocketAndStatic registers WebSocket and static file handlers.
func (s *Server) setupWebSocketAndStatic() {
	s.mux.HandleFunc("/ws", s.handleWebSocket)
	frontendFS, err := ui.GetFS()
	if err != nil {
		logging.GetLogger().
			Warn("Failed to get embedded frontend FS, falling back to disk", "error", err)
		s.mux.Handle("/", http.FileServer(http.Dir("ui/dist")))
	} else {
		logging.GetLogger().Info("Serving frontend from embedded filesystem", "embedded", ui.IsEmbedded())
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
		w.Header().
			Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' ws: wss:; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")

		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers with origin validation.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// Reject null Origin header to prevent CORS bypass attacks (fixes #709)
		// Null origins can occur in sandboxed iframes or redirected requests
		if origin == "null" {
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusForbidden)
			} else {
				logger := logging.FromContext(r.Context())
				localizer := i18n.FromRequest(r)
				message := localizer.T("errors.security.nullOriginForbidden")
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusForbidden,
					ErrCodeForbidden,
					message,
					"",
				) // fixes #694
			}
			return
		}

		// Allow requests from same origin (no Origin header) or validated origins
		if origin == "" || isAllowedOrigin(origin) {
			if origin != "" {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().
				Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			// Cache preflight requests for 24 hours to reduce overhead (fixes #531)
			w.Header().Set("Access-Control-Max-Age", "86400")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// bodyLimitMiddleware enforces request body size limits to prevent DoS attacks.
// Different endpoints have different limits based on expected payload size.
func bodyLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
				logging.GetLogger().Error("PANIC in handler",
					"method", r.Method,
					"path", r.URL.Path,
					"error", err,
					"stack", string(debug.Stack()))
				logger := logging.FromContext(r.Context())
				localizer := i18n.FromRequest(r)
				sendErrorResponseWithDetails(
					w,
					logger,
					http.StatusInternalServerError,
					ErrCodeInternal,
					localizer.T("errors.security.panicRecovered"),
					"",
				) // fixes #694
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// spaFileResult holds the result of opening a file for SPA serving.
type spaFileResult struct {
	file http.File
	stat os.FileInfo
}

// normalizeSPAPath normalizes the request path for SPA handling.
func normalizeSPAPath(path string) string {
	if path == "/" || path == "" {
		return indexHTMLPath
	}
	return path
}

// isAPIOrWSRoute checks if the path is an API or WebSocket route.
func isAPIOrWSRoute(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/ws")
}

// openSPAFile attempts to open a file from the filesystem, falling back to index.html for SPA routes.
func openSPAFile(fsys http.FileSystem, path string) (http.File, error) {
	f, err := fsys.Open(path)
	if err == nil {
		return f, nil
	}

	// File doesn't exist - check if it's an API or WS route (shouldn't happen, but be safe)
	if isAPIOrWSRoute(path) {
		return nil, err
	}

	// Serve index.html for SPA routing (client-side routes)
	return fsys.Open(indexHTMLPath)
}

// handleDirectoryRequest handles requests for directories by serving their index.html.
func handleDirectoryRequest(fsys http.FileSystem, f http.File, path string) (*spaFileResult, error) {
	// Try to serve index.html from the directory
	indexPath := strings.TrimSuffix(path, "/") + indexHTMLPath
	f2, indexErr := fsys.Open(indexPath)
	if indexErr != nil {
		// No index.html in directory - serve root index.html
		f2, indexErr = fsys.Open(indexHTMLPath)
		if indexErr != nil {
			return nil, indexErr
		}
	}
	_ = f.Close()

	stat, err := f2.Stat()
	if err != nil {
		_ = f2.Close()
		return nil, err
	}
	return &spaFileResult{file: f2, stat: stat}, nil
}

func spaHandler(fsys http.FileSystem) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := normalizeSPAPath(r.URL.Path)

		f, err := openSPAFile(fsys, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }()

		stat, err := f.Stat()
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Handle directory requests
		if stat.IsDir() {
			result, dirErr := handleDirectoryRequest(fsys, f, path)
			if dirErr != nil {
				http.NotFound(w, r)
				return
			}
			defer func() { _ = result.file.Close() }()
			f = result.file
			stat = result.stat
		}

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
								s.authManager.Middleware(
									s.csrfManager.CSRFMiddleware(s.mux)))))))))

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
	if err := s.linkMonitor.Start(); err != nil {
		logging.GetLogger().Warn("Link monitor failed to start", "error", err)
	} else {
		logging.GetLogger().Info("Link monitor started",
			"interface", s.config.Interface.Default,
			"state", s.linkMonitor.GetState())
	}

	// Start unified discovery service.
	if err := s.discoveryService.Start(); err != nil {
		logging.GetLogger().
			Warn("Discovery service failed to start (may require root)", "error", err)
	} else {
		status := s.discoveryService.GetStatus()
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
			if err := s.deviceDiscovery.Scan(ctx); err != nil {
				logging.GetLogger().Warn("Initial device discovery scan failed", "error", err)
			} else {
				logging.GetLogger().Info("Initial device discovery scan completed",
					"deviceCount", s.deviceDiscovery.Count())
			}
		}()
	}

	// Start VLAN traffic monitor (requires root/CAP_NET_RAW)
	if err := s.vlanTrafficMonitor.Start(); err != nil {
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
func (s *Server) startHTTP() error {
	logging.GetLogger().Info("Starting HTTP server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("http server: %w", err)
	}
	return nil
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
		if httpsPort == httpsDefaultPort {
			httpsURL = fmt.Sprintf("https://%s%s", host, r.RequestURI)
		} else {
			httpsURL = "https://" + net.JoinHostPort(host, strconv.Itoa(httpsPort)) + r.RequestURI
		}

		http.Redirect(w, r, httpsURL, http.StatusMovedPermanently)
	})

	addr := fmt.Sprintf(":%d", port)
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
	err := s.redirectServer.ListenAndServe()
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

	logging.GetLogger().
		Info("Starting HTTPS server", "addr", s.httpServer.Addr, "tls_version", "1.3")
	if err := s.httpServer.ListenAndServeTLS(certFile, keyFile); err != nil {
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

	// ListenAndServeTLS with empty cert/key paths uses GetCertificate from TLSConfig
	if err := s.httpServer.ListenAndServeTLS("", ""); err != nil {
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
	if encodeErr := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certDER}); encodeErr != nil {
		return "", "", fmt.Errorf("encode certificate PEM: %w", encodeErr)
	}

	// Write private key

	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return "", "", fmt.Errorf("create key file: %w", err)
	}
	defer func() { _ = keyOut.Close() }()
	if keyEncodeErr := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)}); keyEncodeErr != nil {
		return "", "", fmt.Errorf("encode private key PEM: %w", keyEncodeErr)
	}

	logging.GetLogger().Info("Generated self-signed certificate", "cert_file", certFile)
	return certFile, keyFile, nil
}

// Shutdown gracefully shuts down the server (fixes #515, #524).
func (s *Server) Shutdown(ctx context.Context) error {
	logging.GetLogger().InfoContext(ctx, "Shutting down server...")

	// Shutdown HTTP redirect server if running (fixes #515)
	if s.redirectServer != nil {
		logging.GetLogger().InfoContext(ctx, "Shutting down HTTP redirect server...")
		if err := s.redirectServer.Shutdown(ctx); err != nil {
			logging.GetLogger().
				ErrorContext(ctx, "Error shutting down redirect server", "error", err)
		}
	}

	// Shutdown ACME HTTP-01 challenge server if running (fixes #837)
	if s.acmeChallengeServer != nil {
		logging.GetLogger().InfoContext(ctx, "Shutting down ACME challenge server...")
		if err := s.acmeChallengeServer.Shutdown(ctx); err != nil {
			logging.GetLogger().
				ErrorContext(ctx, "Error shutting down ACME challenge server", "error", err)
		}
	}

	// Stop all services (fixes #524 - services will complete gracefully)
	logging.GetLogger().InfoContext(ctx, "Stopping WebSocket hub...")
	s.wsHub.Shutdown()

	logging.GetLogger().InfoContext(ctx, "Stopping link monitor...")
	s.linkMonitor.Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping discovery service...")
	s.discoveryService.Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping VLAN traffic monitor...")
	s.vlanTrafficMonitor.Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping rate limiters...")
	s.loginRateLimiter.Stop()
	s.endpointRateLimiter.Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping CSRF manager...")
	s.csrfManager.Stop()

	// Stop data retention goroutine (fixes #848)
	if s.retentionStopCh != nil {
		logging.GetLogger().InfoContext(ctx, "Stopping data retention goroutine...")
		close(s.retentionStopCh)
		s.retentionStopCh = nil
	}

	// Close database connection (#755)
	if s.db != nil {
		logging.GetLogger().InfoContext(ctx, "Closing database connection...")
		if err := s.db.Close(); err != nil {
			logging.GetLogger().ErrorContext(ctx, "Error closing database", "error", err)
		}
	}

	// Shutdown main HTTP server
	logging.GetLogger().InfoContext(ctx, "Shutting down main HTTP server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown main server: %w", err)
	}
	return nil
}

// Hub returns the WebSocket hub.
func (s *Server) Hub() *Hub {
	return s.wsHub
}

// DB returns the database connection (#755).
func (s *Server) DB() *database.DB {
	return s.db
}

// startDataRetention runs periodic data cleanup based on retention policy (#755).
// The goroutine respects shutdown signals to avoid leaks (fixes #848).
func (s *Server) startDataRetention(retentionDays int) {
	// Run cleanup every hour
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	policy := database.RetentionPolicy{
		MetricsDays:        retentionDays,
		AlertsDays:         retentionDays * retentionAlertsMultiplier, // Keep alerts longer
		SpeedTestDays:      retentionDays,
		DNSResultDays:      retentionDays,
		GatewayResultDays:  retentionDays,
		AuditLogDays:       retentionDays * retentionAuditLogMultiplier,       // Keep audit logs longest
		InactiveDeviceDays: retentionDays * retentionInactiveDeviceMultiplier, // Keep inactive device records longer
	}

	for {
		select {
		case <-s.retentionStopCh:
			logging.GetLogger().Debug("Data retention goroutine shutting down")
			return
		case <-ticker.C:
			if s.db == nil {
				return
			}
			result, err := s.db.RunCleanup(context.Background(), policy)
			if err != nil {
				logging.GetLogger().Error("Data retention cleanup failed", "error", err)
				continue
			}
			totalDeleted := result.MetricsDeleted + result.AlertsDeleted +
				result.SpeedTestsDeleted + result.DNSResultsDeleted +
				result.GatewayResultsDeleted + result.AuditLogsDeleted +
				result.DevicesDeleted
			if totalDeleted > 0 {
				logging.GetLogger().Info("Data retention cleanup completed",
					"metrics_deleted", result.MetricsDeleted,
					"alerts_deleted", result.AlertsDeleted,
					"devices_deleted", result.DevicesDeleted,
					"speedtests_deleted", result.SpeedTestsDeleted,
					"dns_deleted", result.DNSResultsDeleted,
					"gateway_deleted", result.GatewayResultsDeleted,
					"audit_deleted", result.AuditLogsDeleted,
					"duration", result.Duration)
			}
		}
	}
}
