package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/pipeline/publicip"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// NewTestServer creates a minimal server instance for testing.
// This is used by integration tests to verify auth and routing behavior.
// IMPORTANT: Call defer server.Close() after creating the server to avoid goroutine leaks.
func NewTestServer() *Server {
	// Use testutil for consistent test configuration
	testConfig := testutil.NewConfigBuilder().Build()

	return NewTestServerWithConfig(testConfig)
}

// Close cleans up test server resources to prevent goroutine leaks.
// This should be called with defer after creating a test server.
func (s *Server) Close() {
	// Stop rate limiters
	if s.services.RateLimit.Login != nil {
		s.services.RateLimit.Login.Stop()
	}
	if s.services.RateLimit.Endpoint != nil {
		s.services.RateLimit.Endpoint.Stop()
	}

	// Stop CSRF manager
	if s.services.Auth.CSRF != nil {
		s.services.Auth.CSRF.Stop()
	}

	// Stop auth manager (token blacklist cleanup)
	if s.services.Auth.Manager != nil {
		s.services.Auth.Manager.Stop()
	}

	// Stop link monitor
	if s.services.Network.LinkMonitor != nil {
		s.services.Network.LinkMonitor.Stop()
	}

	// Stop discovery service
	if s.services.Discovery.Service != nil {
		s.services.Discovery.Service.Stop()
	}

	// Stop discovery engine (fixes EventBus goroutine leak)
	if s.services.Discovery.Engine != nil {
		s.services.Discovery.Engine.Stop()
	}

	// Stop SSE hub
	if s.services.RealTime.SSEHub != nil {
		s.services.RealTime.SSEHub.Shutdown()
	}
}

// GetAuthenticatedHandler returns the server's handler with auth middleware applied.
// This is used by tests to get the full middleware stack.
func (s *Server) GetAuthenticatedHandler() http.Handler {
	return corsMiddleware(s.authManager().Middleware(s.mux))
}

// SetTestDB injects a *database.DB into the test server. Wave 3 (#85)
// added MFA endpoints that require persistence; tests use this to
// attach a temp SQLite database without standing up the full
// NewServer dependency graph.
func SetTestDB(s *Server, db *database.DB) {
	s.services.Database.DB = db
}

// ResetMFAAttempts clears the package-level MFA rate-limit store.
// Tests call this in t.Cleanup or at the start of each case to avoid
// cross-test bleed-through (the store is process-global).
func ResetMFAAttempts() {
	mfaAttempts.Reset()
}

// NewTestServerWithConfig creates a test server with a specific config.
// This allows tests to customize the server configuration.
// Uses a mock network manager to avoid slow hardware detection while still
// allowing handlers to work properly with realistic interface data.
func NewTestServerWithConfig(cfg *config.Config) *Server {
	// Use mock network manager to avoid slow hardware detection.
	// The mock provides realistic interface data for handler testing.
	netMgr := netif.NewMockManager(netif.DefaultMockConfig())

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
	s.services.Network.LinkMonitor = netif.NewLinkMonitor(cfg.Interface.Default)

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

	// Skip slow discovery initialization (OUI database loading, EventBus goroutines)
	// Discovery.Device, Discovery.Service, Discovery.Engine are nil by default.
	// Handlers check for nil and return appropriate errors.

	// Initialize lightweight telemetry services (no slow I/O)
	s.services.Diagnostics.DNS = dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	s.services.Diagnostics.DNSSecurity = dns.NewSecurityScanner(dns.DefaultSecurityScanConfig())
	s.services.Diagnostics.DHCP = dhcp.NewMonitor(cfg.Interface.Default)
	s.services.Diagnostics.Gateway = gateway.NewTester(gateway.DefaultThresholds())
	s.services.Diagnostics.VLAN = vlan.NewManager(cfg.Interface.Default)
	s.services.Diagnostics.VLANTraffic = vlan.NewTrafficMonitor(cfg.Interface.Default)
	s.services.Diagnostics.Speedtest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	s.services.Diagnostics.Iperf = iperf.NewManager()
	s.services.Diagnostics.Cable = cable.NewTester(cfg.Interface.Default)
	s.services.Diagnostics.PublicIP = publicip.NewChecker()

	s.services.Wireless.WiFi = wifi.NewManager(cfg.Interface.Default)

	// Initialize SSE hub
	s.services.RealTime.SSEHub = NewSSEHub()

	// Wire the ADR-0016 use-cases the same way NewServer does, so handlers that
	// depend on them work under the test harness.
	s.initSettingsUseCase()
	s.initProfilesUseCase()
	s.initNetworkUseCase()

	// Setup routes
	s.setupRoutes()

	return s
}
