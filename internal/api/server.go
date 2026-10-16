// Package api provides the HTTP/REST/SSE server.
package api

// server.go holds the Server struct, its Deps and the NewServer constructor.
// The service accessors (server_accessors.go), the initialisation helpers
// NewServer composes (server_wiring.go, server_init.go), routes, middleware
// stack, SPA fallback, server lifecycle (Start/HTTPS/Shutdown), and data
// retention each live in sibling server_*.go files.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/MustardSeedNetworks/foundation/pkg/csrf"
	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/alerts/inbox"
	"github.com/MustardSeedNetworks/seed/internal/alerts/rules"
	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/config/backups"
	"github.com/MustardSeedNetworks/seed/internal/deviceconfig"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/dhcp"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/cable"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/export"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/packetcapture"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/devices"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/fingerprint"
	"github.com/MustardSeedNetworks/seed/internal/discovery/problems"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	enginestatus "github.com/MustardSeedNetworks/seed/internal/engine/status"
	"github.com/MustardSeedNetworks/seed/internal/flows"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
	healthsettings "github.com/MustardSeedNetworks/seed/internal/health/settings"
	"github.com/MustardSeedNetworks/seed/internal/identity/mfa"
	ssosync "github.com/MustardSeedNetworks/seed/internal/identity/oauth"
	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/ipam"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/lifecycle"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	logquery "github.com/MustardSeedNetworks/seed/internal/logs/query"
	"github.com/MustardSeedNetworks/seed/internal/meraki"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/network/ipconfig"
	"github.com/MustardSeedNetworks/seed/internal/oauth"
	"github.com/MustardSeedNetworks/seed/internal/outbound"
	"github.com/MustardSeedNetworks/seed/internal/paloalto"
	"github.com/MustardSeedNetworks/seed/internal/pipeline/publicip"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	"github.com/MustardSeedNetworks/seed/internal/polling/credentials"
	snmppoller "github.com/MustardSeedNetworks/seed/internal/polling/snmp"
	"github.com/MustardSeedNetworks/seed/internal/polling/targets"
	"github.com/MustardSeedNetworks/seed/internal/probe"
	probeanomaly "github.com/MustardSeedNetworks/seed/internal/probe/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
	securitysettings "github.com/MustardSeedNetworks/seed/internal/security/settings"
	"github.com/MustardSeedNetworks/seed/internal/security/vulntriage"
	"github.com/MustardSeedNetworks/seed/internal/settings/management"
	"github.com/MustardSeedNetworks/seed/internal/settings/persistence"
	"github.com/MustardSeedNetworks/seed/internal/system"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/history"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
	"github.com/MustardSeedNetworks/seed/internal/topology"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
	"github.com/MustardSeedNetworks/seed/internal/wmi"
)

// indexHTMLPath is the path to the SPA entry point.
const indexHTMLPath = "/index.html"

// Server configuration constants.
const (
	// logBroadcasterBufferSize is the buffer size for log broadcaster entries.
	logBroadcasterBufferSize = 1000

	// portScannerTimeout is the timeout for the port scanner.
	portScannerTimeout = 5 * time.Second

	// serverReadTimeoutSec is the HTTP server read timeout in seconds.
	serverReadTimeoutSec = 15

	// serverWriteTimeoutMin is the HTTP server write timeout in minutes for large file transfers.
	serverWriteTimeoutMin = 5

	// serverIdleTimeoutSec is the HTTP server idle connection timeout in seconds.
	serverIdleTimeoutSec = 60

	// setupModeTimeoutMin is how long setup mode remains active (security fix #891).
	// After this duration, setup is disabled and server restart is required.
	setupModeTimeoutMin = 15
)

// API versioning constants (fixes #887).
const (
	// APIVersionPrefix is the version prefix for all API routes.
	// Allows graceful API evolution without breaking existing clients.
	APIVersionPrefix = "/api/v1"

	// APIBasePath is the base path for non-versioned routes (SSE).
	APIBasePath = "/api"
)

// Server represents the HTTPS server.
//
// The long-running domain services live as direct fields on Server (D1,
// ADR-0016/ADR-0020). The earlier ServiceContainer + grouped sub-structs
// (#888) were deleted: the indirection bought nothing once the ADR-0020
// use-cases became the handler interface, and a flat composition root is
// the idiomatic Go shape. The lowercase accessor methods below are the
// in-package read interface and the lazy method values the use-cases bind
// to; they now return these fields directly.
type Server struct {
	// Core configuration
	config     *config.Config
	configPath string
	logPath    string

	// HTTP server components
	httpServer *http.Server
	// routes is the capability registry (ADR-0002): every route and its
	// policy, served behind the global headers by Handler.
	routes *route.Registrar

	// boundPort receives the port startHTTPS bound (the +1..+9 fallback, #69).
	boundPort func(int)

	// --- Auth & security services ---
	authMgr    *auth.Manager
	csrf       *csrf.Manager
	setupToken *SetupTokenManager
	recovery   *auth.RecoveryTokenManager
	oauthMgr   *oauth.Manager
	proxies    *TrustedProxies
	webAuthn   *auth.WebAuthnManager // optional WebAuthn (passkeys), Wave 3 (#85)
	licenseMgr *license.Manager      // offline license manager (Phase D-2); nil in tests

	// renewSupportedFn/renewLeaseFn are the DHCP-renewal seam; vlanSeam is the
	// analogous VLAN seam (handlers_vlan.go). Nil/zero in production; a test
	// sets them since the real calls would modify the machine running the tests.
	renewSupportedFn func() bool
	renewLeaseFn     func(ctx context.Context, interfaceName string) error
	vlanSeam         vlanSeam

	// licenseDir is where activation state is persisted. Empty means the real
	// user config directory, which is what production wants and what a test
	// must never get: license.NewManager() resolves ~/.config/seed, so a test
	// that starts a trial writes to the developer's machine and every later
	// test in the run reads it back. Tests set this to t.TempDir().
	licenseDir string

	// --- Rate limiting ---
	loginLimiter    *RateLimiter
	endpointLimiter *EndpointRateLimiter

	// --- Network services ---
	// linkMon watches the single "primary" interface (cfg.Interface.Default).
	// linkMonPool watches every interface in the Pro multi_interface fan-out
	// (cfg.Interface.AllEthernet() ∪ cfg.Interface.AllWiFi()); it is reconciled
	// on profile change so it tracks the operator's current configuration.
	netMgr      *netif.Manager
	linkMon     *netif.LinkMonitor
	linkMonPool *netif.LinkMonitorPool

	// --- Device & network discovery ---
	deviceDisc    *enumerate.DeviceDiscovery
	discoverySvc  *enumerate.Service
	vulnScan      *vuln.VulnerabilityScanner
	problemDet    *discovery.ProblemDetector
	bluetoothScan *enumerate.BluetoothScanner
	wifiBridgeSvc *enumerate.WiFiBridge
	profiler      *discovery.DeviceProfiler        // shared profiler for SNMP/ports/fingerprinting
	snmpCreds     discovery.SNMPCredentialProvider // vault-backed SNMP credentials (#2118)
	portScanner   *fingerprint.PortScanner         // TCP port scanner (fingerprint stage)
	discoveryEng  *discovery.Engine                // unified discovery engine (primary)

	// --- On-demand network diagnostics ---
	dnsTest       *dns.Tester
	dhcpMon       *dhcp.Monitor
	rogueDet      *dhcp.RogueDetector
	gatewayTest   *gateway.Tester
	vlanMgr       *vlan.Manager
	speedtestTest *speedtest.Tester
	iperf         iperfDeps
	cableTest     *cable.Tester
	publicIP      *publicip.Checker

	// --- Unified probe engine + substrate (Stage A1.8) ---
	probeEngine     *probe.Engine
	probeScheduler  *scheduler.Scheduler
	probeAnomaly    *probeanomaly.Producer
	retentionEngine *retention.Engine

	// alertDelivery is the outbound alert webhook (#368); nil unless configured.
	alertDelivery *alertdelivery.Manager
	// alertWriter is the alert write chain (store, correlation, delivery)
	// the pipelines emit through, kept for use-cases that raise alerts of
	// their own; nil until initAlertPipelines runs.
	alertWriter alertdelivery.Writer
	// dhcpScopeWatch remembers each DHCP scope's exhaustion level between
	// sweeps, so a scope alerts once per climb (P-D5).
	dhcpScopeWatch ipam.Watch

	// anomalyCoord is the single, server-owned anomaly Coordinator every producer
	// shares (ADR-0029): one engine over the merged catalog + the unified store.
	// Built by initAnomalyPlatform before the producers; nil only if the merged
	// catalog is malformed (a programming error), in which case producers degrade
	// to off.
	anomalyCoord *anomaly.Coordinator

	// --- Wi-Fi visibility (scan, manage) ---
	wifiMgr  *wifi.Manager
	wifiScan *wifi.Scanner

	// wifiHelper is the macOS Wi-Fi helper socket, nil elsewhere. Held as an
	// io.Closer because its concrete type only exists on darwin.
	wifiHelper io.Closer

	// --- Real-time communication ---
	sse          *SSEHub                 // SSE hub for real-time updates
	logBroadcast *logging.LogBroadcaster // log streaming
	bus          *events.Bus             // in-process domain event bus (ADR-0004)
	jobRunner    *jobs.Runner            // unified async job runner (ADR-0005)
	jobIdemp     jobIdempotencyStore     // Idempotency-Key dedup for POST /jobs
	captures     *packetcapture.Store    // packet-capture job files, downloaded by ID
	tftp         tftpService             // admin-started TFTP session (P-D4)

	// --- Database ---
	persist         *app.Persistence
	mibDB           *mibdb.DB // MIB database for SNMP OID resolution
	retentionStopCh chan struct{}

	// --- Update service ---

	// engines is the lifecycle registry every long-running engine (probe,
	// retention, snmp-poller, listeners, discovery) registers with. Start
	// drives Registry.Start; Shutdown drives Registry.Stop in reverse
	// registration order. V1.0 NMS expansion — Stage A3.5d.
	engines *engine.Registry

	// Runtime state
	icmpAvailable      bool                        // Whether raw ICMP sockets are available
	startTime          time.Time                   // Application start time for uptime tracking (fixes #540)
	setupModeStartTime time.Time                   // Security fix #891: Track when setup mode started
	background         *BackgroundComponents       // Long-lived components with background lifecycle (report scheduler)
	health             *system.Collector           // Host health sampler; owns the CPU sampler and process cache
	wifiQueries        *troubleshooting.Queries    // Wi-Fi visibility read use-case (ADR-0020)
	wifiManagement     *troubleshooting.Management // Wi-Fi settings/scan/status/connect use-case (ADR-0020)
	wifiDiscovery      *troubleshooting.Discovery  // Enhanced Wi-Fi discovery use-case (ADR-0020)
	settingsStore      *persistence.Service        // Settings-to-profile persistence use-case (ADR-0020)
	settingsManagement *management.Service         // Main settings read/apply use-case (ADR-0020)
	securitySettings   *securitysettings.Service   // SNMP + rogue-DHCP settings use-case (ADR-0020)
	profiles           *catalog.Service            // Profile CRUD/active/import use-case (ADR-0020)
	networkIP          *ipconfig.Service           // IP-config + MTU use-case (ADR-0020)
	alertRules         *rules.Service              // Alert-rule CRUD use-case (ADR-0020)
	discoveryDevices   *devices.Service            // Unified-discovery (engine) use-case (ADR-0020)
	discoverySettings  *discoverysettings.Service  // Network-discovery settings use-case (ADR-0020)
	networkProblems    *problems.Service           // Network problem-detection use-case (ADR-0020)
	topologyQueries    *topology.Queries           // Topology read use-case (ADR-0020)
	pollingTargets     *targets.Service            // Polling-targets CRUD use-case (ADR-0020)
	snmpPoller         *snmppoller.Poller          // Live poller, reloaded when targets change (seed#2452)
	snmpBrowser        *snmppoller.Browser         // On-demand subtree walks, the MIB browser (P-E5)
	deviceCredentials  *credentials.Service        // Device-credential CRUD use-case (#1799)
	deviceConfigs      *deviceconfig.Service       // Device configuration backup use-case (P-D1)
	deviceGroups       *devicegroups.Service       // Device-group CRUD use-case (P-D10)
	wmiInventory       *wmi.Service                // Windows inventory use-case (P-D11)
	autotestResults    autotest.Store              // AutoTest run records (P-E2)
	alertInbox         *inbox.Service              // Alert-inbox (list/ack/resolve) use-case (ADR-0020)
	configBackups      *backups.Service            // Config backup/restore use-case (ADR-0020)
	exportService      *export.Service             // Diagnostic-export use-case (ADR-0020)
	logQuery           *logquery.Service           // Log-query use-case (ADR-0020)
	historyQueries     *history.Service            // Probe/anomaly history read use-case (#175, ADR-0020)
	vulnTriage         *vulntriage.Service         // Vulnerability-triage use-case (#899, ADR-0020)
	flows              *flows.Service              // Flow top-N reads and flow settings use-case (ADR-0020)
	lifecycleEoX       lifecycle.Store             // Imported EoX data for the lifecycle report (P-D9)
	lifecycleFeed      lifecycleFeedSettings       // Opt-in EoX feed settings (P-D9)
	lifecycleRefresher *lifecycle.Refresher        // Fetches EoX data while the feed is on (P-D9)
	interfaceStats     *ifstats.Service            // Interface rates read use-case (UI-SEED-21, #3191)
	ipSLA              *sla.Service                // IP-SLA operations and trend read use-case (P-D8, #3463)
	healthMonitoring   *monitoring.Service         // Health-monitoring use-case (ADR-0020)
	healthSettings     *healthsettings.Service     // Health-checks settings use-case (ADR-0020)
	engineStatus       *enginestatus.Service       // Engine-status use-case (ADR-0020)
	identityUsers      *users.Service              // User-management use-case (ADR-0020, ADR-0024)
	identityTokens     *tokens.Service             // PAT mint/list/revoke use-case (ADR-0020, ADR-0024)
	identityOAuth      *ssosync.Service            // SSO identity-sync use-case (ADR-0020, ADR-0024)
	identityMFA        mfa.Store                   // TOTP + WebAuthn factor store (ADR-0024)
	tlsFingerprint     tlsFingerprintCache         // Cached SHA-256 fingerprint of the active TLS cert, exposed via /__version

	// Opt-in vendor API integrations (P-D12): settings, and the readers that
	// read device state while an integration is on.
	meraki         vendorAPISettings
	merakiReader   *outbound.Reader[meraki.Device]
	paloAlto       vendorAPISettings
	paloAltoReader *outbound.Reader[paloalto.Device]
}

// Deps is everything NewServer needs from the composition root (cmd/seed).
// Config and ConfigPath are required; every other field may be left zero:
// a nil NetManager, Persistence or Background, or a nil TrustedProxies, is a
// server without that collaborator (the api tests run that way).
type Deps struct {
	Config         *config.Config
	ConfigPath     string
	LogPath        string
	NetManager     *netif.Manager
	ICMPAvailable  bool
	TrustedProxies *TrustedProxies
	Persistence    *app.Persistence
	Background     *BackgroundComponents
}

// ErrMissingDependency is returned by NewServer when a required Deps field is
// unset; the wrapped message names the field.
var ErrMissingDependency = errors.New("api: missing server dependency")

func (d Deps) validate() error {
	switch {
	case d.Config == nil:
		return fmt.Errorf("%w: Config", ErrMissingDependency)
	case d.ConfigPath == "":
		return fmt.Errorf("%w: ConfigPath", ErrMissingDependency)
	}
	return nil
}

// NewServer creates a new server instance.
func NewServer(d Deps) (*Server, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	cfg, persist := d.Config, d.Persistence
	s := &Server{
		config:        cfg,
		configPath:    d.ConfigPath,
		logPath:       d.LogPath,
		icmpAvailable: d.ICMPAvailable,
		startTime:     time.Now(),
		background:    d.Background,
		health:        system.NewCollector(),
		engines:       engine.NewRegistry(nil),
	}

	// Initialize auth, WebAuthn, and rate-limiting services
	s.initAuthSecurity(cfg, d.TrustedProxies)

	// Initialize network services
	s.netMgr = d.NetManager
	s.linkMon = netif.NewLinkMonitor(cfg.Interface.Default)
	// LinkMonitorPool tracks every interface in the multi_interface set
	// (Pro). Reconcile primes the pool from the active profile; the pool
	// itself is not started here — server_lifecycle.go owns Start/Stop.
	s.linkMonPool = netif.NewLinkMonitorPool()
	primaryInterfaces := append(cfg.Interface.AllEthernet(), cfg.Interface.AllWiFi()...)
	s.linkMonPool.Reconcile(primaryInterfaces)

	// Initialize discovery + capture-using diagnostics services. WithCapture
	// injects the build-tagged capture adapter (libpcap or CGO-free no-op) so the
	// domain packages stay CGO-free. See docs/architecture/CGO_BUILD_STRATEGY.md.
	s.initCaptureServices(cfg)

	// Initialize telemetry test tools + Wi-Fi services
	s.initTelemetryAndWiFiServices(cfg)

	// Initialize database services
	s.persist = persist

	// Discovery and the trap listener read SNMP credentials from the vault;
	// the listener is built in initDatabaseDependentServices.
	s.snmpCreds = s.newDiscoverySNMPCredentials(cfg)

	s.initDatabaseDependentServices()

	// Security fix #891: Record setup mode start time
	if auth.IsDefaultPasswordHash(cfg.Auth.DefaultPasswordHash) {
		s.setupModeStartTime = time.Now()
		s.issueSetupToken()
	}

	// Set up link state change callback
	s.linkMonitor().OnStateChange(s.onLinkStateChange)

	// Initialize network services (DNS, device discovery subnets)
	s.initNetworkServices(cfg)

	// Initialize OAuth manager for SSO
	s.initOAuthManager()

	// Configure database-backed services if db was passed in
	s.initDatabaseServices(cfg)

	// Initialize SSE hub and log broadcaster
	s.initSSEAndLogging()

	// Initialize discovery service and pipeline
	s.initDiscovery(cfg)

	// Wire the ADR-0020 use-cases now the discovery components exist.
	s.initUseCases()

	// Seed the factory health-check targets on first run (ADR-0027 follow-up).
	// Runs after initUseCases (so s.healthSettings is wired — the seed goes
	// through the use-case) and before initProbeEngine (so the engine loads the
	// seeded probes on start). The probes table is the store of record since P2;
	// without this a fresh install comes up with an empty health-check card.
	// Gated by a persistent marker, so a later delete-all is not re-seeded.
	// Non-fatal: a failure degrades to the empty card rather than aborting startup.
	if persist != nil {
		if err := s.seedDefaultHealthCheckProbes(context.Background(), cfg.HealthChecks); err != nil {
			logging.GetLogger().Error("Failed to seed default health-check probes", "error", err)
		}
	}

	// Wire the settings/profiles/network-IP/alert-rule use-cases (ADR-0020).
	s.initSettingsUseCases()

	// Initialize vulnerability scanner if enabled
	s.initVulnerabilityScanner(cfg)

	// Configure security: allowed origins for CORS
	s.initSecurityOrigins(cfg)

	// Setup routes (sseHub already initialized and running above)
	s.setupRoutes()

	return s, nil
}
