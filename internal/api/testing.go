package api

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/app"
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
	"github.com/MustardSeedNetworks/seed/internal/engine"
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
	if s.loginLimiter != nil {
		s.loginLimiter.Stop()
	}
	if s.endpointLimiter != nil {
		s.endpointLimiter.Stop()
	}

	// Stop CSRF manager
	if s.csrf != nil {
		s.csrf.Stop()
	}

	// Stop auth manager (token blacklist cleanup)
	if s.authMgr != nil {
		s.authMgr.Stop()
	}

	// Stop link monitor
	if s.linkMon != nil {
		s.linkMon.Stop()
	}

	// Stop discovery service
	if s.discoverySvc != nil {
		s.discoverySvc.Stop()
	}

	// Stop discovery engine (fixes EventBus goroutine leak)
	if s.discoveryEng != nil {
		s.discoveryEng.Stop()
	}

	// Stop SSE hub
	if s.sse != nil {
		s.sse.Shutdown()
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
	s.dbConn = db
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

	// Create server with the lightweight subset of services tests need (D1).
	s := &Server{
		config:        cfg,
		configPath:    "/tmp/test-config.yaml",
		logPath:       "/tmp/test.log",
		mux:           http.NewServeMux(),
		icmpAvailable: true,
		engines:       engine.NewRegistry(nil),
	}

	// Initialize services
	s.netMgr = netMgr
	s.linkMon = netif.NewLinkMonitor(cfg.Interface.Default)

	s.authMgr = auth.NewManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.SessionTimeout,
		cfg.Auth.DefaultUsername,
		cfg.Auth.DefaultPasswordHash,
	)
	s.csrf = auth.NewCSRFManager()
	s.setupToken = NewSetupTokenManager()
	s.proxies = NewTrustedProxies("") // Empty for testing

	s.loginLimiter = NewRateLimiter(DefaultRateLimitConfig())
	s.endpointLimiter = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())

	// Skip slow discovery initialization (OUI database loading, EventBus goroutines).
	// deviceDisc, discoverySvc, discoveryEng are nil by default; handlers check for
	// nil and return appropriate errors.

	// Initialize lightweight telemetry services (no slow I/O)
	s.dnsTest = dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	s.dnsSec = dns.NewSecurityScanner(dns.DefaultSecurityScanConfig())
	s.dhcpMon = dhcp.NewMonitor(cfg.Interface.Default)
	s.gatewayTest = gateway.NewTester(gateway.DefaultThresholds())
	s.vlanMgr = vlan.NewManager(cfg.Interface.Default)
	s.vlanTraffic = vlan.NewTrafficMonitor(cfg.Interface.Default)
	s.speedtestTest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	s.iperfMgr = iperf.NewManager()
	s.cableTest = cable.NewTester(cfg.Interface.Default)
	s.publicIP = publicip.NewChecker()

	s.wifiMgr = wifi.NewManager(cfg.Interface.Default)

	// Initialize SSE hub
	s.sse = NewSSEHub()

	// Wire the ADR-0020 use-cases the same way NewServer does, so handlers that
	// depend on them work under the test harness. The Wi-Fi use-cases degrade
	// gracefully here: no visibility component, scanner, or discovery bridge is
	// wired, so reads return empty results and management/discovery report the
	// adapter-absent states.
	s.settingsStore = app.NewSettings(s.db, s.config)
	s.settingsManagement = app.NewSettingsManagement(s.config, s.configPath)
	s.securitySettings = app.NewSecuritySettings(s.config, s.configPath, s.rogueDetector)
	s.profiles = app.NewProfiles(s.db)
	s.networkIP = app.NewNetworkIP(s.netManager, s.config, s.configPath)
	s.alertRules = app.NewAlertRules(s.db)
	s.initUseCases()

	// Setup routes
	s.setupRoutes()

	return s
}
