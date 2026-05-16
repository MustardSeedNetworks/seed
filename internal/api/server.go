// Package api provides the HTTP/REST/SSE server.
package api

// server.go holds the Server struct, NewServer constructor, and the public/
// lowercase service-accessor methods used throughout the api package. The
// initialisation helpers (NewServer composes), routes, middleware stack,
// SPA fallback, server lifecycle (Start/HTTPS/ACME/Shutdown), and data
// retention each live in sibling server_*.go files.

import (
	"net/http"
	"time"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
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
