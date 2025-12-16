package api

import (
	"net/http"
	"time"

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
	"github.com/krisarmstrong/seed/internal/vlan"
	"github.com/krisarmstrong/seed/internal/wifi"
)

// NewTestServer creates a minimal server instance for testing.
// This is used by integration tests to verify auth and routing behavior.
func NewTestServer() *Server {
	// Create minimal test config
	testConfig := &config.Config{
		Server: config.ServerConfig{
			Port:  8443,
			HTTPS: false, // HTTP for testing
		},
		Interface: config.InterfaceConfig{
			Default: "lo", // Use loopback for testing
		},
		Auth: config.AuthConfig{
			JWTSecret:           "test-secret-key-for-testing-only",
			SessionTimeout:      24 * time.Hour,
			DefaultUsername:     "testadmin",
			DefaultPasswordHash: "$2a$10$test.hash.for.testing.only",
		},
		DNS: config.DNSConfig{
			TestHostname: "example.com",
		},
		Speedtest: config.SpeedtestConfig{
			ServerID: "",
		},
	}

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
	}

	// Initialize WebSocket hub
	s.wsHub = NewHub()

	// Setup routes
	s.setupRoutes()

	return s
}
