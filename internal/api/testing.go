package api

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/cable"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/discovery"
	"github.com/krisarmstrong/seed/internal/dns"
	"github.com/krisarmstrong/seed/internal/gateway"
	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/network"
	"github.com/krisarmstrong/seed/internal/publicip"
	"github.com/krisarmstrong/seed/internal/speedtest"
	"github.com/krisarmstrong/seed/internal/testutil"
	"github.com/krisarmstrong/seed/internal/vlan"
	"github.com/krisarmstrong/seed/internal/wifi"
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
	return corsMiddleware(s.authManager.Middleware(s.mux))
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

	// Create server with all required managers
	s := &Server{
		config:              cfg,
		configPath:          "/tmp/test-config.yaml",
		logPath:             "/tmp/test.log",
		mux:                 http.NewServeMux(),
		netManager:          netMgr,
		icmpAvailable:       true,
		authManager:         auth.NewManager(cfg.Auth.JWTSecret, cfg.Auth.SessionTimeout, cfg.Auth.DefaultUsername, cfg.Auth.DefaultPasswordHash),
		loginRateLimiter:    NewRateLimiter(DefaultRateLimitConfig()),
		endpointRateLimiter: NewEndpointRateLimiter(DefaultEndpointRateLimitConfig()),
		linkMonitor:         network.NewLinkMonitor(cfg.Interface.Default),
		discoveryManager:    discovery.NewManager(cfg.Interface.Default),
		deviceDiscovery:     discovery.NewDeviceDiscovery(cfg.Interface.Default),
		discoveryService:    discovery.NewService(cfg, cfg.Interface.Default),
		dnsTester:           dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds()),
		dhcpMonitor:         dhcp.NewMonitor(cfg.Interface.Default),
		gatewayTester:       gateway.NewTester(gateway.DefaultThresholds()),
		vlanManager:         vlan.NewManager(cfg.Interface.Default),
		vlanTrafficMonitor:  vlan.NewTrafficMonitor(cfg.Interface.Default),
		wifiManager:         wifi.NewManager(cfg.Interface.Default),
		cableTester:         cable.NewTester(cfg.Interface.Default),
		speedtestTester:     speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID),
		iperfManager:        iperf.NewManager(),
		publicipChecker:     publicip.NewChecker(),
		setupTokenManager:   NewSetupTokenManager(), // Fixes test initialization
	}

	// Initialize WebSocket hub
	s.wsHub = NewHub()

	// Setup routes
	s.setupRoutes()

	return s
}
