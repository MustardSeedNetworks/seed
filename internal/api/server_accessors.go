package api

// server_accessors.go holds the Server service accessors: the exported read
// interface and the lowercase in-package accessors the ADR-0020 use-cases bind
// to as method values.

import (
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/iperf"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/oauth"
	"github.com/MustardSeedNetworks/seed/internal/pipeline/publicip"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
)

// Service accessors — the in-package read interface and the lazy method
// values the ADR-0020 use-cases bind to (D1, formerly #888).

// GetConfig returns the server configuration.
func (s *Server) GetConfig() *config.Config { return s.config }

// AuthManager returns the authentication manager.
func (s *Server) AuthManager() *auth.Manager { return s.authMgr }

// CSRFManager returns the CSRF token manager.
func (s *Server) CSRFManager() *csrf.Manager { return s.csrf }

// RecoveryManager returns the password recovery token manager.
func (s *Server) RecoveryManager() *auth.RecoveryTokenManager { return s.recovery }

// OAuthManager returns the OAuth manager.
func (s *Server) OAuthManager() *oauth.Manager { return s.oauthMgr }

// TrustedProxies returns the trusted proxies configuration.
func (s *Server) TrustedProxies() *TrustedProxies { return s.proxies }

// LoginRateLimiter returns the login rate limiter.
func (s *Server) LoginRateLimiter() *RateLimiter { return s.loginLimiter }

// EndpointRateLimiter returns the endpoint rate limiter.
func (s *Server) EndpointRateLimiter() *EndpointRateLimiter { return s.endpointLimiter }

// NetManager returns the network manager.
func (s *Server) NetManager() *netif.Manager { return s.netMgr }

// LinkMonitor returns the link monitor.
func (s *Server) LinkMonitor() *netif.LinkMonitor { return s.linkMon }

// DeviceDiscovery returns the device discovery service.
func (s *Server) DeviceDiscovery() *enumerate.DeviceDiscovery { return s.deviceDisc }

// DiscoveryService returns the unified discovery service.
func (s *Server) DiscoveryService() *enumerate.Service { return s.discoverySvc }

// VulnScanner returns the vulnerability scanner.
func (s *Server) VulnScanner() *vuln.VulnerabilityScanner { return s.vulnScan }

// DNSTester returns the DNS tester.
func (s *Server) DNSTester() *dns.Tester { return s.dnsTest }

// DHCPMonitor returns the DHCP monitor.
func (s *Server) DHCPMonitor() *dhcp.Monitor { return s.dhcpMon }

// RogueDetector returns the rogue DHCP detector.
func (s *Server) RogueDetector() *dhcp.RogueDetector { return s.rogueDet }

// GatewayTester returns the gateway tester.
func (s *Server) GatewayTester() *gateway.Tester { return s.gatewayTest }

// VLANManager returns the VLAN manager.
func (s *Server) VLANManager() *vlan.Manager { return s.vlanMgr }

// SpeedtestTester returns the speedtest tester.
func (s *Server) SpeedtestTester() *speedtest.Tester { return s.speedtestTest }

// IperfManager returns the iperf manager.
func (s *Server) IperfManager() *iperf.Manager { return s.iperf.mgr }

// CableTester returns the cable tester.
func (s *Server) CableTester() *cable.Tester { return s.cableTest }

// PublicIPChecker returns the public IP checker.
func (s *Server) PublicIPChecker() *publicip.Checker { return s.publicIP }

// WiFiManager returns the WiFi manager.
func (s *Server) WiFiManager() *wifi.Manager { return s.wifiMgr }

// WiFiScanner returns the WiFi scanner.
func (s *Server) WiFiScanner() *wifi.Scanner { return s.wifiScan }

// SSEHub returns the SSE hub.
func (s *Server) SSEHub() *SSEHub { return s.sse }

// LogBroadcaster returns the log broadcaster.
func (s *Server) LogBroadcaster() *logging.LogBroadcaster { return s.logBroadcast }

// MibDB returns the MIB database for SNMP OID resolution.
func (s *Server) MibDB() *mibdb.DB { return s.mibDB }

// Lowercase accessors — the unexported in-package read interface; many are
// also taken as method values by the ADR-0020 use-case wiring, so they must
// stay methods (not be inlined to field access).

func (s *Server) authManager() *auth.Manager                  { return s.authMgr }
func (s *Server) csrfManager() *csrf.Manager                  { return s.csrf }
func (s *Server) setupTokenManager() *SetupTokenManager       { return s.setupToken }
func (s *Server) recoveryManager() *auth.RecoveryTokenManager { return s.recovery }
func (s *Server) oauthManager() *oauth.Manager                { return s.oauthMgr }
func (s *Server) trustedProxies() *TrustedProxies             { return s.proxies }
func (s *Server) loginRateLimiter() *RateLimiter              { return s.loginLimiter }
func (s *Server) endpointRateLimiter() *EndpointRateLimiter   { return s.endpointLimiter }
func (s *Server) netManager() *netif.Manager                  { return s.netMgr }
func (s *Server) defaultInterface() string                    { return s.config.Interface.Default }
func (s *Server) discoveryScanTimeout() time.Duration         { return s.config.NetworkDiscovery.ScanTimeout }
func (s *Server) linkMonitor() *netif.LinkMonitor             { return s.linkMon }
func (s *Server) deviceDiscovery() *enumerate.DeviceDiscovery { return s.deviceDisc }
func (s *Server) discoveryService() *enumerate.Service        { return s.discoverySvc }
func (s *Server) discoveryEngine() *discovery.Engine          { return s.discoveryEng }
func (s *Server) problemDetector() *discovery.ProblemDetector { return s.problemDet }

// resolveWiFiInterface picks the Wi-Fi interface for a request: an explicit query
// override, else the configured Wi-Fi interface, else the default interface.
func (s *Server) resolveWiFiInterface(r *http.Request) string {
	if iface := s.getInterfaceFromRequest(r); iface != "" {
		return iface
	}
	if wifi := s.config.Interface.WiFi; wifi != "" {
		return wifi
	}
	return s.config.Interface.Default
}

// anomalyEngine is the shared, server-owned anomaly engine (ADR-0029), exposed so
// the store-read adapters can re-derive the catalog-static Impact / FollowUps the
// store does not persist. Nil when the merged catalog failed to build or no DB is
// wired (the test harness) — enrichment then degrades to a passthrough.
func (s *Server) anomalyEngine() *anomaly.Engine {
	if s.anomalyCoord == nil {
		return nil
	}
	return s.anomalyCoord.Engine()
}

func (s *Server) bluetoothScanner() *enumerate.BluetoothScanner { return s.bluetoothScan }
func (s *Server) vulnScanner() *vuln.VulnerabilityScanner       { return s.vulnScan }
func (s *Server) dnsTester() *dns.Tester                        { return s.dnsTest }
func (s *Server) dhcpMonitor() *dhcp.Monitor                    { return s.dhcpMon }
func (s *Server) rogueDetector() *dhcp.RogueDetector            { return s.rogueDet }
func (s *Server) gatewayTester() *gateway.Tester                { return s.gatewayTest }
func (s *Server) vlanManager() *vlan.Manager                    { return s.vlanMgr }
func (s *Server) speedtestTester() *speedtest.Tester            { return s.speedtestTest }
func (s *Server) iperfManager() *iperf.Manager                  { return s.iperf.mgr }
func (s *Server) cableTester() *cable.Tester                    { return s.cableTest }
func (s *Server) publicipChecker() *publicip.Checker            { return s.publicIP }
func (s *Server) wifiManager() *wifi.Manager                    { return s.wifiMgr }
func (s *Server) wifiScanner() *wifi.Scanner                    { return s.wifiScan }
func (s *Server) sseHub() *SSEHub                               { return s.sse }
func (s *Server) logBroadcaster() *logging.LogBroadcaster       { return s.logBroadcast }
func (s *Server) eventBus() *events.Bus                         { return s.bus }
func (s *Server) jobsRunner() *jobs.Runner                      { return s.jobRunner }
func (s *Server) jobIdempotency() jobIdempotencyStore           { return s.jobIdemp }
func (s *Server) persistence() *app.Persistence                 { return s.persist }
func (s *Server) licenseManager() *license.Manager              { return s.licenseMgr }
func (s *Server) engineRegistry() *engine.Registry              { return s.engines }
