package httpapi

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy/wifi"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/roots/publicip"
	"github.com/krisarmstrong/seed/internal/sap/cable"
	"github.com/krisarmstrong/seed/internal/sap/dns"
	"github.com/krisarmstrong/seed/internal/sap/gateway"
	"github.com/krisarmstrong/seed/internal/sap/speedtest"
	"github.com/krisarmstrong/seed/internal/sap/vlan"
	"github.com/krisarmstrong/seed/internal/testutil"
)

// NewTestServer creates a minimal server instance for testing.
// This is used by integration tests to verify auth and routing behavior.
func NewTestServer() *Server {
	// Use testutil for consistent test configuration
	testConfig := testutil.NewConfigBuilder().Build()

	return NewTestServerWithConfig(testConfig)
}

// GetAuthenticatedHandler returns the server's handler with auth middleware applied.
// This is used by tests to get the full middleware stack.
func (s *Server) GetAuthenticatedHandler() http.Handler {
	return corsMiddleware(s.authManager().Middleware(s.mux))
}

// NewTestServerWithConfig creates a test server with a specific config.
// This allows tests to customize the server configuration.
func NewTestServerWithConfig(cfg *config.Config) *Server {
	// Create test network manager (minimal)
	netMgr, err := network.NewManager(cfg.Interface.Default)
	if err != nil {
		// Use a nil manager for testing - handlers should handle this gracefully
		netMgr = nil
	}

	// Create server with ServiceContainer (#888)
	s := &Server{
		config:        cfg,
		configPath:    "/tmp/test-config.yaml",
		logPath:       "/tmp/test.log",
		mux:           http.NewServeMux(),
		icmpAvailable: true,
		services:      NewServiceContainer(),
	}

	// Initialize services in container
	s.services.Network.Manager = netMgr
	s.services.Network.LinkMonitor = network.NewLinkMonitor(cfg.Interface.Default)

	s.services.Auth.Manager = auth.NewManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.SessionTimeout,
		cfg.Auth.DefaultUsername,
		cfg.Auth.DefaultPasswordHash,
	)
	s.services.Auth.CSRF = auth.NewCSRFManager()
	s.services.Auth.SetupToken = NewSetupTokenManager()
	s.services.Auth.TrustedProxies = NewTrustedProxies("") // Empty for testing

	s.services.RateLimit.Login = NewRateLimiter(DefaultRateLimitConfig())
	s.services.RateLimit.Endpoint = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())

	s.services.Discovery.Device = discovery.NewDeviceDiscovery(cfg.Interface.Default)
	s.services.Discovery.Service = discovery.NewService(cfg, cfg.Interface.Default, nil)

	s.services.Sap.DNS = dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	s.services.Sap.DNSSecurity = dns.NewSecurityScanner(dns.DefaultSecurityScanConfig())
	s.services.Sap.DHCP = dhcp.NewMonitor(cfg.Interface.Default)
	s.services.Sap.Gateway = gateway.NewTester(gateway.DefaultThresholds())
	s.services.Sap.VLAN = vlan.NewManager(cfg.Interface.Default)
	s.services.Sap.VLANTraffic = vlan.NewTrafficMonitor(cfg.Interface.Default)
	s.services.Sap.Speedtest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	s.services.Sap.Iperf = iperf.NewManager()
	s.services.Sap.Cable = cable.NewTester(cfg.Interface.Default)
	s.services.Sap.PublicIP = publicip.NewChecker()

	s.services.Canopy.WiFi = wifi.NewManager(cfg.Interface.Default)

	// Initialize Discovery Engine for unified discovery
	s.services.Discovery.Engine = discovery.NewEngine(nil)

	// Initialize WebSocket hub
	s.services.RealTime.WSHub = NewHub()

	// Setup routes
	s.setupRoutes()

	return s
}
