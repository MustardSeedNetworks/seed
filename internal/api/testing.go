package api

import (
	"net/http"
	"time"

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
	"github.com/krisarmstrong/luminetiq/internal/vlan"
	"github.com/krisarmstrong/luminetiq/internal/wifi"
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

	// Create test network manager (minimal)
	netMgr, err := network.NewManager(testConfig.Interface.Default)
	if err != nil {
		// Use a nil manager for testing - handlers should handle this gracefully
		netMgr = nil
	}

	// Create server with all required managers
	s := &Server{
		config:               testConfig,
		configPath:           "/tmp/test-config.yaml",
		logPath:              "/tmp/test.log",
		mux:                  http.NewServeMux(),
		netManager:           netMgr,
		icmpAvailable:        true,
		authManager:          auth.NewManager(testConfig.Auth.JWTSecret, testConfig.Auth.SessionTimeout, testConfig.Auth.DefaultUsername, testConfig.Auth.DefaultPasswordHash),
		loginRateLimiter:     NewRateLimiter(DefaultRateLimitConfig()),
		endpointRateLimiter:  NewEndpointRateLimiter(DefaultEndpointRateLimitConfig()),
		linkMonitor:          network.NewLinkMonitor(testConfig.Interface.Default),
		discoveryManager:     discovery.NewManager(testConfig.Interface.Default),
		deviceDiscovery:      discovery.NewDeviceDiscovery(testConfig.Interface.Default),
		discoveryService:     discovery.NewService(testConfig, testConfig.Interface.Default),
		dnsTester:            dns.NewTester("", testConfig.DNS.TestHostname, dns.DefaultThresholds()),
		dhcpMonitor:          dhcp.NewMonitor(testConfig.Interface.Default),
		gatewayTester:        gateway.NewTester(gateway.DefaultThresholds()),
		vlanManager:          vlan.NewManager(testConfig.Interface.Default),
		vlanTrafficMonitor:   vlan.NewTrafficMonitor(testConfig.Interface.Default),
		wifiManager:          wifi.NewManager(testConfig.Interface.Default),
		cableTester:          cable.NewTester(testConfig.Interface.Default),
		speedtestTester:      speedtest.NewTesterWithConfig(testConfig.Speedtest.ServerID),
		iperfManager:         iperf.NewManager(),
		publicipChecker:      publicip.NewChecker(),
		logAccessToken:       "",
		logAccessHeader:      "X-Log-Token",
		requireLogToken:      false,
	}

	// Initialize WebSocket hub
	s.wsHub = NewHub()

	// Setup routes
	s.setupRoutes()

	return s
}

// GetAuthenticatedHandler returns the server's handler with auth middleware applied.
// This is used by tests to get the full middleware stack.
func (s *Server) GetAuthenticatedHandler() http.Handler {
	return corsMiddleware(s.authManager.Middleware(s.mux))
}
