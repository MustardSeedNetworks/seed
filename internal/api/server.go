// Package api provides the HTTP/REST/SSE server.
package api

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
	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/mibdb"
	"github.com/krisarmstrong/seed/internal/netif"
	"github.com/krisarmstrong/seed/internal/oauth"
	"github.com/krisarmstrong/seed/internal/paths"
	"github.com/krisarmstrong/seed/internal/pipeline/publicip"
	"github.com/krisarmstrong/seed/internal/services/cable"
	"github.com/krisarmstrong/seed/internal/services/discovery"
	"github.com/krisarmstrong/seed/internal/services/dns"
	"github.com/krisarmstrong/seed/internal/services/gateway"
	"github.com/krisarmstrong/seed/internal/services/iperf"
	"github.com/krisarmstrong/seed/internal/services/speedtest"
	"github.com/krisarmstrong/seed/internal/services/vlan"
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

	// portScannerTimeout is the timeout for the port scanner.
	portScannerTimeout = 5 * time.Second

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

// API versioning constants (fixes #887).
const (
	// APIVersionPrefix is the version prefix for all API routes.
	// Allows graceful API evolution without breaking existing clients.
	APIVersionPrefix = "/api/v1"

	// APIBasePath is the base path for non-versioned routes (SSE).
	APIBasePath = "/api"
)

// Server represents the HTTP/HTTPS server.
// Refactored to use ServiceContainer for dependency injection (#888).
type Server struct {
	// Core configuration
	config     *config.Config
	configPath string
	logPath    string

	// HTTP server components
	httpServer          *http.Server
	mux                 *http.ServeMux
	redirectServer      *http.Server // HTTP→HTTPS redirect server (fixes #515)
	redirectServerErr   chan error   // Error channel for redirect server
	acmeChallengeServer *http.Server // HTTP-01 challenge server for ACME (fixes #837)

	// Service container - holds all domain services (#888)
	services *ServiceContainer

	// Runtime state
	icmpAvailable      bool      // Whether raw ICMP sockets are available
	startTime          time.Time // Application start time for uptime tracking (fixes #540)
	setupModeStartTime time.Time // Security fix #891: Track when setup mode started
	modules            *Modules  // Application modules (Sap, Shell, Canopy, Roots, Harvest)
}

// NewServer creates a new server instance.
// NewServer creates a new server instance.
func NewServer(
	cfg *config.Config,
	configPath, logPath string,
	netMgr *netif.Manager,
	icmpAvailable bool,
	trustedProxies *TrustedProxies,
	db *database.DB,
	modules *Modules,
) *Server {
	// Create service container (#888)
	services := NewServiceContainer()

	// Initialize auth services
	services.Auth.Manager = auth.NewManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.SessionTimeout,
		cfg.Auth.DefaultUsername,
		cfg.Auth.DefaultPasswordHash,
	)
	services.Auth.CSRF = auth.NewCSRFManager()
	services.Auth.SetupToken = NewSetupTokenManager()
	services.Auth.Recovery = auth.NewRecoveryTokenManager(paths.Resolve(paths.ModeAuto).DataDir)
	services.Auth.TrustedProxies = trustedProxies

	// Initialize rate limiters
	services.RateLimit.Login = NewRateLimiter(DefaultRateLimitConfig())
	services.RateLimit.Endpoint = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())

	// Initialize network services
	services.Network.Manager = netMgr
	services.Network.LinkMonitor = netif.NewLinkMonitor(cfg.Interface.Default)

	// Initialize discovery services
	services.Discovery.Device = discovery.NewDeviceDiscoveryWithOUI(
		cfg.Interface.Default,
		cfg.NetworkDiscovery.OUIFilePath,
		cfg.NetworkDiscovery.OUIMaxAge,
	)
	// Note: services.Discovery.Service is initialized after profiler is created (see below)

	// Initialize SAP services
	services.Sap.DNS = dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	services.Sap.DNSSecurity = dns.NewSecurityScanner(dns.DefaultSecurityScanConfig())
	services.Sap.DHCP = dhcp.NewMonitor(cfg.Interface.Default)
	services.Sap.RogueDetector = dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
		Interface:        cfg.Interface.Default,
		KnownServers:     cfg.DHCP.RogueDetection.KnownServers,
		AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
	})
	services.Sap.Gateway = gateway.NewTester(gateway.DefaultThresholds())
	services.Sap.VLAN = vlan.NewManager(cfg.Interface.Default)
	services.Sap.VLANTraffic = vlan.NewTrafficMonitor(cfg.Interface.Default)
	services.Sap.Speedtest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	services.Sap.Iperf = iperf.NewManager()
	services.Sap.Cable = cable.NewTester(cfg.Interface.Default)
	services.Sap.PublicIP = publicip.NewChecker()

	// Initialize Canopy services
	services.Canopy.WiFi = wifi.NewManager(cfg.Interface.Default)
	services.Canopy.Scanner = wifi.NewScanner(cfg.Interface.Default)

	// Initialize database services
	services.Database.DB = db

	s := &Server{
		config:        cfg,
		configPath:    configPath,
		logPath:       logPath,
		mux:           http.NewServeMux(),
		icmpAvailable: icmpAvailable,
		startTime:     time.Now(),
		modules:       modules,
		services:      services,
	}

	// Security fix #891: Record setup mode start time
	if auth.IsDefaultPasswordHash(cfg.Auth.DefaultPasswordHash) {
		s.setupModeStartTime = time.Now()
	}

	// Set up link state change callback
	s.linkMonitor().OnStateChange(s.onLinkStateChange)

	// Initialize network services (DNS, device discovery subnets, survey manager)
	s.initNetworkServices(cfg)

	// Initialize OAuth manager for SSO
	s.initOAuthManager()

	// Configure database-backed services if db was passed in
	s.initDatabaseServices(cfg, db)

	// Initialize SSE hub and log broadcaster
	s.initSSEAndLogging(db)

	// Initialize discovery service and pipeline
	s.initDiscoveryPipeline(cfg)

	// Initialize vulnerability scanner if enabled
	s.initVulnerabilityScanner(cfg)

	// Configure security: allowed origins for CORS
	s.initSecurityOrigins(cfg)

	// Setup routes (sseHub already initialized and running above)
	s.setupRoutes()

	return s
}

// Service accessors - provide backwards-compatible access to services (#888)

// AuthManager returns the authentication manager.
func (s *Server) AuthManager() *auth.Manager { return s.services.Auth.Manager }

// CSRFManager returns the CSRF token manager.
func (s *Server) CSRFManager() *auth.CSRFManager { return s.services.Auth.CSRF }

// SetupTokenManager returns the setup token manager.
func (s *Server) SetupTokenManager() *SetupTokenManager { return s.services.Auth.SetupToken }

// RecoveryManager returns the password recovery token manager.
func (s *Server) RecoveryManager() *auth.RecoveryTokenManager { return s.services.Auth.Recovery }

// OAuthManager returns the OAuth manager.
func (s *Server) OAuthManager() *oauth.Manager { return s.services.Auth.OAuth }

// TrustedProxies returns the trusted proxies configuration.
func (s *Server) TrustedProxies() *TrustedProxies { return s.services.Auth.TrustedProxies }

// LoginRateLimiter returns the login rate limiter.
func (s *Server) LoginRateLimiter() *RateLimiter { return s.services.RateLimit.Login }

// EndpointRateLimiter returns the endpoint rate limiter.
func (s *Server) EndpointRateLimiter() *EndpointRateLimiter { return s.services.RateLimit.Endpoint }

// NetManager returns the network manager.
func (s *Server) NetManager() *netif.Manager { return s.services.Network.Manager }

// LinkMonitor returns the link monitor.
func (s *Server) LinkMonitor() *netif.LinkMonitor { return s.services.Network.LinkMonitor }

// DeviceDiscovery returns the device discovery service.
func (s *Server) DeviceDiscovery() *discovery.DeviceDiscovery { return s.services.Discovery.Device }

// DiscoveryService returns the unified discovery service.
func (s *Server) DiscoveryService() *discovery.Service { return s.services.Discovery.Service }

// Pipeline returns the discovery pipeline.
func (s *Server) Pipeline() *discovery.Pipeline { return s.services.Discovery.Pipeline }

// VulnScanner returns the vulnerability scanner.
func (s *Server) VulnScanner() *discovery.VulnerabilityScanner {
	return s.services.Discovery.Vulnerability
}

// DNSTester returns the DNS tester.
func (s *Server) DNSTester() *dns.Tester { return s.services.Sap.DNS }

// DNSSecurityScanner returns the DNS security scanner.
func (s *Server) DNSSecurityScanner() *dns.SecurityScanner { return s.services.Sap.DNSSecurity }

// DHCPMonitor returns the DHCP monitor.
func (s *Server) DHCPMonitor() *dhcp.Monitor { return s.services.Sap.DHCP }

// RogueDetector returns the rogue DHCP detector.
func (s *Server) RogueDetector() *dhcp.RogueDetector { return s.services.Sap.RogueDetector }

// GatewayTester returns the gateway tester.
func (s *Server) GatewayTester() *gateway.Tester { return s.services.Sap.Gateway }

// VLANManager returns the VLAN manager.
func (s *Server) VLANManager() *vlan.Manager { return s.services.Sap.VLAN }

// VLANTrafficMonitor returns the VLAN traffic monitor.
func (s *Server) VLANTrafficMonitor() *vlan.TrafficMonitor { return s.services.Sap.VLANTraffic }

// SpeedtestTester returns the speedtest tester.
func (s *Server) SpeedtestTester() *speedtest.Tester { return s.services.Sap.Speedtest }

// IperfManager returns the iperf manager.
func (s *Server) IperfManager() *iperf.Manager { return s.services.Sap.Iperf }

// CableTester returns the cable tester.
func (s *Server) CableTester() *cable.Tester { return s.services.Sap.Cable }

// PublicIPChecker returns the public IP checker.
func (s *Server) PublicIPChecker() *publicip.Checker { return s.services.Sap.PublicIP }

// WiFiManager returns the WiFi manager.
func (s *Server) WiFiManager() *wifi.Manager { return s.services.Canopy.WiFi }

// WiFiScanner returns the WiFi scanner.
func (s *Server) WiFiScanner() *wifi.Scanner { return s.services.Canopy.Scanner }

// SurveyManager returns the survey manager.
func (s *Server) SurveyManager() *survey.Manager { return s.services.Canopy.Survey }

// SSEHub returns the SSE hub.
func (s *Server) SSEHub() *SSEHub { return s.services.RealTime.SSEHub }

// LogBroadcaster returns the log broadcaster.
func (s *Server) LogBroadcaster() *logging.LogBroadcaster { return s.services.RealTime.LogBroadcaster }

// DB returns the database connection.
func (s *Server) DB() *database.DB { return s.services.Database.DB }

// MibDB returns the MIB database for SNMP OID resolution.
func (s *Server) MibDB() *mibdb.DB { return s.services.Database.MibDB }

// Lowercase aliases for backwards compatibility with existing handler code (#888)
// These match the original field access pattern (e.g., s.authManager vs s.AuthManager())

func (s *Server) authManager() *auth.Manager                  { return s.services.Auth.Manager }
func (s *Server) csrfManager() *auth.CSRFManager              { return s.services.Auth.CSRF }
func (s *Server) setupTokenManager() *SetupTokenManager       { return s.services.Auth.SetupToken }
func (s *Server) recoveryManager() *auth.RecoveryTokenManager { return s.services.Auth.Recovery }
func (s *Server) oauthManager() *oauth.Manager                { return s.services.Auth.OAuth }
func (s *Server) trustedProxies() *TrustedProxies             { return s.services.Auth.TrustedProxies }
func (s *Server) loginRateLimiter() *RateLimiter              { return s.services.RateLimit.Login }
func (s *Server) endpointRateLimiter() *EndpointRateLimiter   { return s.services.RateLimit.Endpoint }
func (s *Server) netManager() *netif.Manager                  { return s.services.Network.Manager }
func (s *Server) linkMonitor() *netif.LinkMonitor             { return s.services.Network.LinkMonitor }
func (s *Server) deviceDiscovery() *discovery.DeviceDiscovery { return s.services.Discovery.Device }
func (s *Server) discoveryService() *discovery.Service        { return s.services.Discovery.Service }
func (s *Server) pipeline() *discovery.Pipeline               { return s.services.Discovery.Pipeline }
func (s *Server) vulnScanner() *discovery.VulnerabilityScanner {
	return s.services.Discovery.Vulnerability
}
func (s *Server) dnsTester() *dns.Tester                   { return s.services.Sap.DNS }
func (s *Server) dnsSecurityScanner() *dns.SecurityScanner { return s.services.Sap.DNSSecurity }
func (s *Server) dhcpMonitor() *dhcp.Monitor               { return s.services.Sap.DHCP }
func (s *Server) rogueDetector() *dhcp.RogueDetector       { return s.services.Sap.RogueDetector }
func (s *Server) gatewayTester() *gateway.Tester           { return s.services.Sap.Gateway }
func (s *Server) vlanManager() *vlan.Manager               { return s.services.Sap.VLAN }
func (s *Server) vlanTrafficMonitor() *vlan.TrafficMonitor { return s.services.Sap.VLANTraffic }
func (s *Server) speedtestTester() *speedtest.Tester       { return s.services.Sap.Speedtest }
func (s *Server) iperfManager() *iperf.Manager             { return s.services.Sap.Iperf }
func (s *Server) cableTester() *cable.Tester               { return s.services.Sap.Cable }
func (s *Server) publicipChecker() *publicip.Checker       { return s.services.Sap.PublicIP }
func (s *Server) wifiManager() *wifi.Manager               { return s.services.Canopy.WiFi }
func (s *Server) wifiScanner() *wifi.Scanner               { return s.services.Canopy.Scanner }
func (s *Server) surveyManager() *survey.Manager           { return s.services.Canopy.Survey }
func (s *Server) sseHub() *SSEHub                          { return s.services.RealTime.SSEHub }
func (s *Server) logBroadcaster() *logging.LogBroadcaster  { return s.services.RealTime.LogBroadcaster }
func (s *Server) db() *database.DB                         { return s.services.Database.DB }

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
		s.dnsTester().SetConfiguredServers(configuredServers)
	}

	// Initialize device discovery with configured additional subnets
	s.initAdditionalSubnets(cfg)

	// Initialize survey manager
	surveyStoragePath := "data/surveys"
	s.services.Canopy.Survey = survey.NewManager(
		surveyStoragePath,
		s.wifiScanner(),
		s.wifiManager(),
		s.iperfManager(),
	)
	if err := s.surveyManager().LoadSurveys(); err != nil {
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

	if err := s.deviceDiscovery().SetAdditionalSubnets(enabledCIDRs); err != nil {
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
	s.authManager().SetUserStore(userStore)

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

	// Initialize MIB database for SNMP OID resolution
	s.initMibDatabase(db)

	// Start data retention cleanup in background (fixes #848)
	if cfg.Database.RetentionDays > 0 {
		s.services.Database.RetentionStopCh = make(chan struct{})
		go s.startDataRetention(cfg.Database.RetentionDays)
	}
}

// initMibDatabase initializes the MIB database and loads built-in OID definitions.
func (s *Server) initMibDatabase(db *database.DB) {
	// Create MIB database interface using the underlying SQL connection
	mibDB := mibdb.New(db.Conn())
	s.services.Database.MibDB = mibDB

	// Load built-in OID definitions (918+ standard OIDs from RFC MIBs)
	if err := mibDB.LoadBuiltinOIDs(); err != nil {
		logging.GetLogger().Error("Failed to load built-in MIB OIDs", "error", err)
		return
	}

	// Log statistics
	stats, err := mibDB.Stats()
	if err != nil {
		logging.GetLogger().Warn("Failed to get MIB database stats", "error", err)
		return
	}
	logging.GetLogger().Info("MIB database initialized",
		"oid_entries", stats["oid_entries"],
		"mib_count", stats["mib_count"])
}

// initSSEAndLogging initializes the SSE hub and log broadcaster.
func (s *Server) initSSEAndLogging(db *database.DB) {
	// Initialize SSE hub for real-time updates
	s.services.RealTime.SSEHub = NewSSEHub()
	go s.sseHub().Run()

	// Initialize log broadcaster for real-time log streaming
	s.services.RealTime.LogBroadcaster = logging.InitBroadcaster(logBroadcasterBufferSize)
	s.logBroadcaster().SetBroadcaster(&sseLogBroadcastAdapter{hub: s.sseHub()})

	// Wire up database persistence for logs if database is available
	if db != nil {
		s.logBroadcaster().SetDBWriter(&dbLogWriterAdapter{db: db})
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
		s.deviceDiscovery().SetDBWriter(&dbDeviceWriterAdapter{db: db})
		logging.GetLogger().Info("Device discovery initialized with database persistence")
	}
}

// initDiscoveryPipeline initializes the discovery service and pipeline.
func (s *Server) initDiscoveryPipeline(cfg *config.Config) {
	// Create SHARED DeviceProfiler - used by Service, Pipeline, and Engine
	// This ensures port scan results and SNMP data are consistent across the system
	sharedProfiler := discovery.NewDeviceProfiler(discovery.DefaultProfilerConfig(), &cfg.SNMP)
	s.services.Discovery.Profiler = sharedProfiler

	// Create PortScanner for Engine
	portScanner, err := discovery.NewPortScanner(portScannerTimeout)
	if err != nil {
		logging.GetLogger().Warn("Failed to create port scanner", "error", err)
	} else {
		s.services.Discovery.PortScanner = portScanner
	}

	// Initialize discovery service with the shared profiler
	s.services.Discovery.Service = discovery.NewService(cfg, cfg.Interface.Default, sharedProfiler)
	logging.GetLogger().Info("Discovery service initialized with shared profiler")

	// Initialize discovery pipeline with the SAME shared profiler
	pipelineCfg := discovery.PipelineConfigFromAdapter(&cfg.Pipeline)
	s.services.Discovery.Pipeline = discovery.NewPipeline(
		&pipelineCfg,
		s.deviceDiscovery(),
		sharedProfiler, // Use the same profiler as Service
		&pipelineBroadcastAdapter{hub: s.sseHub()},
	)

	// Link Service and Pipeline for coordination
	s.discoveryService().SetPipeline(s.pipeline())

	// Set up pipeline completion callback to sync results back to service
	s.discoveryService().SetOnPipelineComplete(func(devices []*discovery.DiscoveredDevice) {
		logging.GetLogger().Info(
			"Pipeline completed, syncing results to discovery service",
			"device_count",
			len(devices),
		)
	})

	logging.GetLogger().Info("Discovery pipeline initialized",
		"phases_enabled", s.pipeline().GetEnabledPhaseNames(),
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
	s.services.Discovery.Vulnerability = vulnScanner
	logging.GetLogger().Info("Vulnerability scanner initialized",
		"cve_database", scannerCfg.CVEDatabase, "threshold", scannerCfg.SeverityThreshold)

	// Initialize problem detector for network issue detection
	s.services.Discovery.ProblemDetector = discovery.NewProblemDetector()
	logging.GetLogger().Info("Problem detector initialized")

	// Initialize Bluetooth scanner
	btConfig := discovery.DefaultBluetoothScanConfig()
	var ouiDB *discovery.OUIDatabase
	if s.services.Discovery.Device != nil {
		ouiDB = s.services.Discovery.Device.GetOUIDatabase()
	}
	s.services.Discovery.BluetoothScanner = discovery.NewBluetoothScanner("", btConfig, ouiDB)
	logging.GetLogger().Info("Bluetooth scanner initialized")

	// Initialize WiFi bridge connecting canopy/wifi to discovery
	if s.services.Canopy.Scanner != nil {
		wifiBridgeConfig := discovery.DefaultWiFiBridgeConfig()
		s.services.Discovery.WiFiBridge = discovery.NewWiFiBridge(
			s.services.Canopy.Scanner,
			s.services.Canopy.WiFi,
			ouiDB,
			wifiBridgeConfig,
		)
		logging.GetLogger().Info("WiFi bridge initialized")
	}

	// Initialize Discovery Engine (primary unified discovery system)
	engineConfig := discovery.DefaultEngineConfig()
	s.services.Discovery.Engine = discovery.NewEngine(engineConfig)

	// Wire in all collectors
	if s.services.Discovery.Device != nil {
		s.services.Discovery.Engine.SetWiredCollector(s.services.Discovery.Device)
	}
	if s.services.Discovery.WiFiBridge != nil {
		s.services.Discovery.Engine.SetWiFiCollector(s.services.Discovery.WiFiBridge)
	}
	if s.services.Discovery.BluetoothScanner != nil {
		s.services.Discovery.Engine.SetBluetoothCollector(s.services.Discovery.BluetoothScanner)
	}
	if s.services.Discovery.Profiler != nil {
		s.services.Discovery.Engine.SetProfiler(s.services.Discovery.Profiler)
	}
	if s.services.Discovery.PortScanner != nil {
		s.services.Discovery.Engine.SetPortScanner(s.services.Discovery.PortScanner)
	}
	if s.services.Discovery.Vulnerability != nil {
		s.services.Discovery.Engine.SetVulnScanner(s.services.Discovery.Vulnerability)
	}

	// Start the engine
	if startErr := s.services.Discovery.Engine.Start(context.Background()); startErr != nil {
		logging.GetLogger().Error("Failed to start discovery engine", "error", startErr)
	} else {
		logging.GetLogger().Info("Discovery engine started",
			"capabilities", s.services.Discovery.Engine.GetCapabilities(),
		)
	}
}

// initSecurityOrigins configures allowed origins for CORS.
func (s *Server) initSecurityOrigins(cfg *config.Config) {
	getOriginState().setAllowedOrigins(cfg.Security.AllowedOrigins)

	if len(cfg.Security.AllowedOrigins) == 0 {
		logging.GetLogger().Info("Using default RFC 1918 private network origins for CORS")
		return
	}

	// Check for wildcard origin in production mode (fixes #715)
	// Production mode is inferred from HTTPS being enabled
	s.logWildcardOriginWarning(cfg)

	logging.GetLogger().Info(
		"Configured explicit allowed origins for CORS",
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
	return GetClientIPWithTrustedProxies(r, s.trustedProxies())
}

// onLinkStateChange handles link up/down events.
func (s *Server) onLinkStateChange(event netif.LinkEvent) {
	logging.GetLogger().
		Info("Link state change", "interface", event.Interface, "state", event.State)

	switch event.State {
	case netif.LinkStateUp:
		// Link came up - reload discovery service to restart protocol capture
		logging.GetLogger().Info("Link up - reloading discovery service")
		if err := s.discoveryService().Reload(); err != nil {
			logging.GetLogger().Warn("Failed to reload discovery service", "error", err)
		}

		// Notify clients with linkState message (SSE primary, WebSocket for backwards compat)
		linkStateMsg := Message{
			Type: "linkState",
			Payload: map[string]any{
				"interface": event.Interface,
				"state":     "up",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		}
		s.sseHub().Broadcast(linkStateMsg)

		// Also broadcast link card update immediately to trigger frontend auto-run tests.
		// The frontend listens for card_update messages on the "link" card to detect
		// link-up transitions and run speedtest/iperf tests.
		// Multi-interface support (#754): Include interface in broadcast.
		if linkData := s.collectLinkData(); linkData != nil {
			s.sseHub().BroadcastCardUpdateForInterface("link", linkData, event.Interface)
		}
	case netif.LinkStateDown:
		// Link went down - notify clients
		logging.GetLogger().Info("Link down - notifying clients")
		linkStateMsg := Message{
			Type: "linkState",
			Payload: map[string]any{
				"interface": event.Interface,
				"state":     "down",
				"timestamp": event.Timestamp.Format(time.RFC3339),
			},
		}
		s.sseHub().Broadcast(linkStateMsg)

		// Also broadcast link card update for proper state tracking.
		// Frontend uses this to track DOWN state for detecting DOWN→UP transitions.
		// Multi-interface support (#754): Include interface in broadcast.
		if linkData := s.collectLinkData(); linkData != nil {
			s.sseHub().BroadcastCardUpdateForInterface("link", linkData, event.Interface)
		}
	case netif.LinkStateUnknown:
		// Unknown state - log but don't take action
		logging.GetLogger().Warn("Link state unknown", "interface", event.Interface)
	}
}

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/__version", s.handleBuildVersion)
	s.setupCoreRoutes()
	s.registerUpdateRoutes()
	s.setupSAPRoutes()
	s.setupShellRoutes()
	s.setupRootsRoutes()
	s.setupCanopyRoutes()
	s.setupHarvestRoutes()
	s.setupSSEAndStatic()
}

// setupCoreRoutes registers auth, settings, config, and setup routes.
func (s *Server) setupCoreRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/auth/login", s.handleLogin)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/logout", s.handleLogout)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/refresh", s.handleRefreshToken)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/csrf", s.handleCSRFToken)
	s.mux.HandleFunc(APIVersionPrefix+"/status", s.handleStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/settings", s.handleSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/settings/defaults", s.handleSettingsDefaults)
	s.mux.HandleFunc(APIVersionPrefix+"/settings/link", s.handleLinkSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/settings/cable", s.handleCableTestSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/interfaces", s.handleInterfaces)
	s.mux.HandleFunc(APIVersionPrefix+"/interface", s.handleInterface)
	s.mux.HandleFunc(APIVersionPrefix+"/network/mtu", s.handleSetMTU)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backups", s.handleConfigBackups)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backup", s.handleConfigBackupCreate)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backup/delete", s.handleConfigBackupDelete)
	s.mux.HandleFunc(APIVersionPrefix+"/config/restore", s.handleConfigRestore)
	s.mux.HandleFunc(APIVersionPrefix+"/config/version", s.handleConfigVersion)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles", s.handleProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/active", s.handleActiveProfile)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/import", s.handleImportProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/export", s.handleExportProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/", s.handleProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/setup/complete", s.handleSetupComplete)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/status", s.handleRecoveryStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/complete", s.handleRecoveryComplete)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/instructions", s.handleRecoveryInstructions)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/providers", s.handleSSOProviders)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/login", s.handleSSOLogin)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/callback", s.handleSSOCallback)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/settings", s.handleSSOSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/update", s.handleSSOUpdate)
	s.mux.HandleFunc(APIVersionPrefix+"/health", s.handleHealth)
}

// setupSAPRoutes registers SAP module routes (live telemetry).
func (s *Server) setupSAPRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/sap/link", s.handleLink)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/cable", s.handleCable)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns", s.handleDNS)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns/security", s.handleDNSSecurity)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns/security/settings", s.handleDNSSecuritySettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/gateway", s.handleGateway)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue", s.handleRogueDHCP)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue/servers", s.handleRogueDHCPServers)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue/config", s.handleRogueDHCPConfig)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/vlan", s.handleVLAN)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/vlan/traffic", s.handleVLANTraffic)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/vlan/interface", s.handleVLANInterface)
	s.mux.Handle(
		APIVersionPrefix+"/sap/speedtest",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleSpeedtest)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/speedtest/status", s.handleSpeedtestStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/info", s.handleIperfInfo)
	s.mux.Handle(
		APIVersionPrefix+"/sap/iperf/client",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleIperfClient)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/client/status", s.handleIperfClientStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/server", s.handleIperfServer)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/server/status", s.handleIperfServerStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/suggestions", s.handleIperfSuggestions)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/settings", s.handleHealthChecksSettings)
	s.mux.Handle(
		APIVersionPrefix+"/sap/health-checks/run",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleHealthChecks)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/results", s.handleHealthCheckResults)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/history", s.handleHealthCheckHistory)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/scores", s.handleHealthCheckScores)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/sla", s.handleHealthCheckSLA)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/alerts", s.handleHealthCheckAlerts)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/anomalies", s.handleHealthCheckAnomalies)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/snmp/settings", s.handleSNMPSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/system/health", s.handleSystemHealth)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/ipconfig", s.handleIPConfig)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/ipconfig/settings", s.handleIPSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/publicip", s.handlePublicIP)
}

// setupShellRoutes registers Shell module routes (security posture).
func (s *Server) setupShellRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery", s.handleDiscovery)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/probe", s.handleTCPProbe)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/portscan", s.handlePortScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/options", s.handleDiscoveryOptions)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/service/status", s.handleDiscoveryServiceStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/fingerprint", s.handleAdvancedFingerprint)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices", s.handleDevices)
	s.mux.Handle(
		APIVersionPrefix+"/shell/devices/scan",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleDevicesScan)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/status", s.handleDevicesStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/settings", s.handleDevicesSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/subnets", s.handleDevicesSubnets)
	s.mux.Handle(
		APIVersionPrefix+"/shell/vulnerabilities/scan",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleVulnerabilityScan)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/status", s.handleVulnerabilityStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/results", s.handleVulnerabilityResults)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/device", s.handleDeviceVulnerabilities)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/settings", s.handleVulnerabilitySettings)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/validate-api-key", s.handleNVDAPIKeyValidate)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/status", s.handlePipelineStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/start", s.handlePipelineStart)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/cancel", s.handlePipelineCancel)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/config", s.handlePipelineConfigRoute)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/port-intensity", s.handlePipelinePortIntensityInfo)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/timing-profiles", s.handlePipelineTimingProfiles)

	// Network problem detection routes
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems", s.handleNetworkProblems)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems/scan", s.handleProblemScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems/thresholds", s.handleProblemThresholds)

	// Bluetooth discovery routes
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/scan", s.handleBluetoothScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/devices", s.handleBluetoothDevices)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/stats", s.handleBluetoothStats)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/status", s.handleBluetoothStatus)

	// Enhanced WiFi discovery routes (unified discovery)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/scan", s.handleWiFiDiscoveryScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/networks", s.handleWiFiDiscoveryNetworks)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/aps", s.handleWiFiDiscoveryAPs)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/stats", s.handleWiFiDiscoveryStats)

	// Discovery Engine routes (primary unified discovery system)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine", s.handleEngineDiscovery)
	s.mux.Handle(
		APIVersionPrefix+"/discovery/engine/scan",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleEngineScan)),
	)
	s.mux.Handle(
		APIVersionPrefix+"/discovery/engine/quick",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleEngineQuickScan)),
	)
	s.mux.Handle(
		APIVersionPrefix+"/discovery/engine/full",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleEngineFullScan)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/stats", s.handleEngineStats)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/capabilities", s.handleEngineCapabilities)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/device/", s.handleEngineDevice)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/events", s.handleEngineEvents)
}

// setupRootsRoutes registers Roots module routes (path analysis).
func (s *Server) setupRootsRoutes() {
	s.mux.Handle(
		APIVersionPrefix+"/roots/traceroute",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleTraceroute)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/roots/path", s.handlePath)
}

// setupCanopyRoutes registers Canopy module routes (Wi-Fi planning).
func (s *Server) setupCanopyRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi", s.handleWiFi)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/scan", s.handleWiFiScan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/status", s.handleWiFiStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/channel-graph", s.handleWiFiChannelGraph)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/settings", s.handleWiFiSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/connect", s.handleWiFiConnect)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/disconnect", s.handleWiFiDisconnect)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/saved", s.handleWiFiSavedNetworks)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/forget", s.handleWiFiForgetNetwork)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/create", s.createSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/list", s.listSurveys)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey", s.getSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/delete", s.deleteSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/start", s.startSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/pause", s.pauseSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/complete", s.completeSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/sample", s.addSurveySample)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floorplan", s.updateSurveyFloorPlan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/settings", s.updateSurveySettings)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/import/airmapper",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.importAirMapper)),
	)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/heatmap",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.getSurveyHeatmap)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/dead-zones", s.getSurveyDeadZones)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floors", s.handleSurveyFloors)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor", s.handleSurveyFloor)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor/floorplan", s.updateFloorFloorPlan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor/sample", s.addFloorSample)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/active-floor", s.setActiveFloor)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/report",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.generateSurveyReport)),
	)
}

// setupHarvestRoutes registers Harvest module routes (reporting).
func (s *Server) setupHarvestRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/export", s.handleExport)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs", s.handleLogs)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/client", s.handleClientLogs)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/query", s.handleLogsQuery)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/stats", s.handleLogsStats)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/recent", s.handleLogsRecent)
}

// setupSSEAndStatic registers SSE and static file handlers.
func (s *Server) setupSSEAndStatic() {
	// SSE endpoint for real-time updates
	s.mux.HandleFunc(APIVersionPrefix+"/events", s.handleSSE)
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
		case strings.HasPrefix(path, APIVersionPrefix+"/auth/"):
			limit = MaxBodySizeAuth
		case strings.HasPrefix(path, APIVersionPrefix+"/config/"):
			limit = MaxBodySizeConfig
		case path == APIVersionPrefix+"/canopy/survey/floorplan":
			limit = MaxBodySizeFloorPlan
		case path == APIVersionPrefix+"/canopy/survey/import/airmapper":
			limit = MaxBodySizeAirMapper
		case strings.HasPrefix(path, APIVersionPrefix):
			limit = MaxBodySizeJSON
		default:
			limit = MaxBodySizeDefault
		}

		// Wrap the body with a limit reader
		r.Body = http.MaxBytesReader(w, r.Body, limit)

		next.ServeHTTP(w, r)
	})
}

// isAllowedOrigin is defined in messages.go for CORS origin checking.

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

// isAPIRoute checks if the path is an API or SSE route.
func isAPIRoute(path string) bool {
	return strings.HasPrefix(path, APIVersionPrefix) ||
		strings.HasPrefix(path, APIBasePath+"/events") // SSE endpoint
}

// openSPAFile attempts to open a file from the filesystem, falling back to index.html for SPA routes.
func openSPAFile(fsys http.FileSystem, path string) (http.File, error) {
	f, err := fsys.Open(path)
	if err == nil {
		return f, nil
	}

	// File doesn't exist - check if it's an API route (shouldn't happen, but be safe)
	if isAPIRoute(path) {
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
	if keyEncodeErr := pem.Encode(
		keyOut,
		&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)},
	); keyEncodeErr != nil {
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
	logging.GetLogger().InfoContext(ctx, "Stopping SSE hub...")
	s.sseHub().Shutdown()

	logging.GetLogger().InfoContext(ctx, "Stopping link monitor...")
	s.linkMonitor().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping discovery service...")
	s.discoveryService().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping VLAN traffic monitor...")
	s.vlanTrafficMonitor().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping rate limiters...")
	s.loginRateLimiter().Stop()
	s.endpointRateLimiter().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping CSRF manager...")
	s.csrfManager().Stop()

	logging.GetLogger().InfoContext(ctx, "Stopping auth manager (token blacklist)...")
	s.authManager().Stop()

	// Stop data retention goroutine (fixes #848)
	if s.services.Database.RetentionStopCh != nil {
		logging.GetLogger().InfoContext(ctx, "Stopping data retention goroutine...")
		close(s.services.Database.RetentionStopCh)
		s.services.Database.RetentionStopCh = nil
	}

	// Close database connection (#755)
	if s.db() != nil {
		logging.GetLogger().InfoContext(ctx, "Closing database connection...")
		if err := s.db().Close(); err != nil {
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

// These methods are now defined via accessor methods in the Server struct section above.

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
		case <-s.services.Database.RetentionStopCh:
			logging.GetLogger().Debug("Data retention goroutine shutting down")
			return
		case <-ticker.C:
			if s.db() == nil {
				return
			}
			result, err := s.db().RunCleanup(context.Background(), policy)
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
