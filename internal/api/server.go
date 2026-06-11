// Package api provides the HTTP/REST/SSE server.
package api

// server.go holds the Server struct, NewServer constructor, and the public/
// lowercase service-accessor methods used throughout the api package. The
// initialisation helpers (NewServer composes), routes, middleware stack,
// SPA fallback, server lifecycle (Start/HTTPS/ACME/Shutdown), and data
// retention each live in sibling server_*.go files.

import (
	"net/http"
	"os"
	"strconv"
	"time"

	alertpipeline "github.com/MustardSeedNetworks/seed/internal/alerts/pipeline"
	"github.com/MustardSeedNetworks/seed/internal/alerts/rules"
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
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/bluetooth"
	"github.com/MustardSeedNetworks/seed/internal/discovery/devices"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/fingerprint"
	"github.com/MustardSeedNetworks/seed/internal/discovery/problems"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/engine"
	enginestatus "github.com/MustardSeedNetworks/seed/internal/engine/status"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
	ssosync "github.com/MustardSeedNetworks/seed/internal/identity/oauth"
	"github.com/MustardSeedNetworks/seed/internal/identity/tokens"
	"github.com/MustardSeedNetworks/seed/internal/identity/users"
	"github.com/MustardSeedNetworks/seed/internal/license"
	listenersink "github.com/MustardSeedNetworks/seed/internal/listener/sink"
	"github.com/MustardSeedNetworks/seed/internal/listener/snmptrap"
	"github.com/MustardSeedNetworks/seed/internal/listener/syslog"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/network/ipconfig"
	"github.com/MustardSeedNetworks/seed/internal/oauth"
	"github.com/MustardSeedNetworks/seed/internal/paths"
	"github.com/MustardSeedNetworks/seed/internal/pipeline/publicip"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	snmporchestrator "github.com/MustardSeedNetworks/seed/internal/polling/snmp/orchestrator"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/snmpclient"
	"github.com/MustardSeedNetworks/seed/internal/probe"
	probeanomaly "github.com/MustardSeedNetworks/seed/internal/probe/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
	"github.com/MustardSeedNetworks/seed/internal/profiles/catalog"
	"github.com/MustardSeedNetworks/seed/internal/scheduler"
	"github.com/MustardSeedNetworks/seed/internal/settings/persistence"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
	"github.com/MustardSeedNetworks/seed/internal/topology"
	"github.com/MustardSeedNetworks/seed/internal/update"
	"github.com/MustardSeedNetworks/seed/internal/update/lifecycle"
	"github.com/MustardSeedNetworks/seed/internal/wifi"
	"github.com/MustardSeedNetworks/seed/internal/wifi/survey"
	"github.com/MustardSeedNetworks/seed/internal/wifi/troubleshooting"
)

// indexHTMLPath is the path to the SPA entry point.
const indexHTMLPath = "/index.html"

// Server configuration constants.
const (
	// logBroadcasterBufferSize is the buffer size for log broadcaster entries.
	logBroadcasterBufferSize = 1000

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
	httpServer          *http.Server
	mux                 *http.ServeMux
	acmeChallengeServer *http.Server // HTTP-01 challenge server for ACME (fixes #837)

	// manifest records every route registered through register() (the
	// capability registry, ADR-0002). Exposed read-only via /__capabilities
	// for fleet policy audits.
	manifest []route

	// --- Auth & security services ---
	authMgr    *auth.Manager
	csrf       *auth.CSRFManager
	setupToken *SetupTokenManager
	recovery   *auth.RecoveryTokenManager
	oauthMgr   *oauth.Manager
	proxies    *TrustedProxies
	webAuthn   *auth.WebAuthnManager        // optional WebAuthn (passkeys), Wave 3 (#85)
	licenseMgr *license.Manager             // offline license manager (Phase D-2); nil in tests
	apiTokens  *database.APITokenRepository // personal-access tokens (Phase D-2)

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
	profiler      *discovery.DeviceProfiler // shared profiler for SNMP/ports/fingerprinting
	portScanner   *fingerprint.PortScanner  // TCP port scanner (fingerprint stage)
	discoveryEng  *discovery.Engine         // unified discovery engine (primary)

	// --- On-demand network diagnostics ---
	dnsTest       *dns.Tester
	dnsSec        *dns.SecurityScanner
	dhcpMon       *dhcp.Monitor
	rogueDet      *dhcp.RogueDetector
	gatewayTest   *gateway.Tester
	vlanMgr       *vlan.Manager
	vlanTraffic   *vlan.TrafficMonitor
	speedtestTest *speedtest.Tester
	iperfMgr      *iperf.Manager
	cableTest     *cable.Tester
	publicIP      *publicip.Checker

	// --- Unified probe engine + substrate (Stage A1.8) ---
	probeEngine     *probe.Engine
	probeScheduler  *scheduler.Scheduler
	probeAnomaly    *probeanomaly.Producer
	retentionEngine *retention.Engine

	// --- Wi-Fi visibility (scan, manage, survey) ---
	wifiMgr   *wifi.Manager
	wifiScan  *wifi.Scanner
	surveyMgr *survey.Manager

	// --- Real-time communication ---
	sse          *SSEHub                 // SSE hub for real-time updates
	logBroadcast *logging.LogBroadcaster // log streaming
	bus          *events.Bus             // in-process domain event bus (ADR-0004)
	jobRunner    *jobs.Runner            // unified async job runner (ADR-0005)
	jobIdemp     jobIdempotencyStore     // Idempotency-Key dedup for POST /jobs

	// --- Database ---
	dbConn          *database.DB
	mibDB           *mibdb.DB // MIB database for SNMP OID resolution
	retentionStopCh chan struct{}

	// --- Update service ---
	updateSvc *update.Service

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
	wifiQueries        *troubleshooting.Queries    // Wi-Fi visibility read use-case (ADR-0020)
	wifiManagement     *troubleshooting.Management // Wi-Fi settings/scan/status/connect use-case (ADR-0020)
	wifiDiscovery      *troubleshooting.Discovery  // Enhanced Wi-Fi discovery use-case (ADR-0020)
	settingsStore      *persistence.Service        // Settings-to-profile persistence use-case (ADR-0020)
	profiles           *catalog.Service            // Profile CRUD/active/import use-case (ADR-0020)
	networkIP          *ipconfig.Service           // IP-config + MTU use-case (ADR-0020)
	alertRules         *rules.Service              // Alert-rule CRUD use-case (ADR-0020)
	discoveryDevices   *devices.Service            // Unified-discovery (engine) use-case (ADR-0020)
	networkProblems    *problems.Service           // Network problem-detection use-case (ADR-0020)
	bluetoothScans     *bluetooth.Service          // Bluetooth-discovery use-case (ADR-0020)
	healthMonitoring   *monitoring.Service         // Health-monitoring use-case (ADR-0020)
	updateLifecycle    *lifecycle.Service          // Update-lifecycle use-case (ADR-0020)
	engineStatus       *enginestatus.Service       // Engine-status use-case (ADR-0020)
	identityUsers      *users.Service              // User-management use-case (ADR-0020, ADR-0024)
	identityTokens     *tokens.Service             // PAT mint/list/revoke use-case (ADR-0020, ADR-0024)
	identityOAuth      *ssosync.Service            // SSO identity-sync use-case (ADR-0020, ADR-0024)
	tlsFingerprint     tlsFingerprintCache         // Cached SHA-256 fingerprint of the active TLS cert, exposed via /__version
}

// NewServer creates a new server instance.
func NewServer(
	cfg *config.Config,
	configPath, logPath string,
	netMgr *netif.Manager,
	icmpAvailable bool,
	trustedProxies *TrustedProxies,
	db *database.DB,
	background *BackgroundComponents,
) *Server {
	s := &Server{
		config:        cfg,
		configPath:    configPath,
		logPath:       logPath,
		mux:           http.NewServeMux(),
		icmpAvailable: icmpAvailable,
		startTime:     time.Now(),
		background:    background,
		engines:       engine.NewRegistry(nil),
	}

	// Initialize auth services
	s.authMgr = auth.NewManager(
		cfg.Auth.JWTSecret,
		cfg.Auth.SessionTimeout,
		cfg.Auth.DefaultUsername,
		cfg.Auth.DefaultPasswordHash,
	)
	s.csrf = auth.NewCSRFManager()
	s.setupToken = NewSetupTokenManager()
	s.recovery = auth.NewRecoveryTokenManager(paths.Resolve(paths.ModeAuto).DataDir)
	s.proxies = trustedProxies

	// Wave 3 (#85): initialise the WebAuthn manager. The relying-party
	// ID and origins are derived from the server config; failures here
	// are non-fatal because the rest of the auth surface still works
	// without passkeys.
	if wan, wanErr := auth.NewWebAuthnManager(webAuthnConfigFromServer(cfg)); wanErr != nil {
		logging.GetLogger().Warn("WebAuthn manager init failed; passkeys disabled",
			"error", wanErr)
	} else {
		s.webAuthn = wan
	}

	// Initialize rate limiters
	s.loginLimiter = NewRateLimiter(DefaultRateLimitConfig())
	s.endpointLimiter = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())

	// Initialize network services
	s.netMgr = netMgr
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

	// Initialize telemetry services
	s.dnsTest = dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	s.dnsSec = dns.NewSecurityScanner(dns.DefaultSecurityScanConfig())
	s.gatewayTest = gateway.NewTester(gateway.DefaultThresholds())
	s.vlanMgr = vlan.NewManager(cfg.Interface.Default)
	s.speedtestTest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	s.iperfMgr = iperf.NewManager()
	s.cableTest = cable.NewTester(cfg.Interface.Default)
	s.publicIP = publicip.NewChecker()

	// Initialize Wi-Fi services
	s.wifiMgr = wifi.NewManager(cfg.Interface.Default)
	s.wifiScan = wifi.NewScanner(cfg.Interface.Default)

	// Initialize database services
	s.dbConn = db

	s.initDatabaseDependentServices(db)

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
	s.initDiscovery(cfg)

	// Wire the ADR-0020 use-cases now the discovery components exist.
	s.initUseCases()

	// Wire the settings-persistence use-case (ADR-0020). The composition root
	// builds the adapters; api passes its lazy db accessor + live config.
	s.settingsStore = app.NewSettings(s.db, s.config)

	// Wire the profiles catalog use-case (ADR-0020). The composition root builds
	// the adapter; api passes its lazy db accessor.
	s.profiles = app.NewProfiles(s.db)

	// Wire the network IP-config + MTU use-case (ADR-0020). The composition root
	// builds the adapters; api passes its lazy manager accessor + config.
	s.networkIP = app.NewNetworkIP(s.netManager, s.config, s.configPath)

	// Wire the alert-rule use-case (ADR-0020). The composition root builds the
	// adapter; api passes its lazy db accessor.
	s.alertRules = app.NewAlertRules(s.db)

	// Initialize vulnerability scanner if enabled
	s.initVulnerabilityScanner(cfg)

	// Configure security: allowed origins for CORS
	s.initSecurityOrigins(cfg)

	// Setup routes (sseHub already initialized and running above)
	s.setupRoutes()

	return s
}

// initCaptureServices constructs the services that perform live packet capture
// (device discovery via LLDP/CDP/EDP, DHCP monitoring, rogue-DHCP detection, and
// VLAN traffic) and injects the capture port adapter into each. The adapter is
// build-tagged (libpcap under CGO/Windows, a CGO-free no-op otherwise) so the
// domain packages stay CGO-free. See docs/architecture/CGO_BUILD_STRATEGY.md.
func (s *Server) initCaptureServices(cfg *config.Config) {
	captureOpener := defaultCaptureOpener()

	// s.discoverySvc is initialized later, after the shared profiler.
	s.deviceDisc = enumerate.NewDeviceDiscoveryWithOUI(
		cfg.Interface.Default,
		cfg.NetworkDiscovery.OUIFilePath,
		cfg.NetworkDiscovery.OUIMaxAge,
		enumerate.WithCapture(captureOpener),
	)
	s.dhcpMon = dhcp.NewMonitor(cfg.Interface.Default, dhcp.WithCapture(captureOpener))
	s.rogueDet = dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
		Interface:        cfg.Interface.Default,
		KnownServers:     cfg.DHCP.RogueDetection.KnownServers,
		AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
	}, dhcp.WithCapture(captureOpener))
	s.vlanTraffic = vlan.NewTrafficMonitor(
		cfg.Interface.Default, vlan.WithCapture(captureOpener),
	)
}

// initDatabaseDependentServices wires every service that needs a
// live database connection. Called from NewServer after s.dbConn
// is populated. Splits into per-concern helpers to keep each scope
// focused and to keep NewServer under the funlen limit.
func (s *Server) initDatabaseDependentServices(db *database.DB) {
	if db == nil {
		// Tests construct a Server without a DB; skip the
		// database-dependent wiring entirely rather than crash.
		return
	}
	s.initLicenseAndAPITokens(db)
	s.initProbeEngine(db)
	s.initRetentionEngine(db)
	s.initListeners(db)
	s.initTopologyReconcilers(db)
	s.initAlertPipelines(db)
	s.initSNMPPoller(db)
}

// initLicenseAndAPITokens wires the Phase D-2 license manager + API
// token repository onto the server. The license manager is
// best-effort: failure to load isn't fatal, the mint endpoint just
// behaves as if no paid license is present (rejects with 402).
func (s *Server) initLicenseAndAPITokens(db *database.DB) {
	s.apiTokens = database.NewAPITokenRepository(db)
	lm, lmErr := license.NewManager()
	if lmErr != nil {
		logging.GetLogger().Warn("license manager init failed; minting will be disabled",
			"error", lmErr)
		return
	}
	s.licenseMgr = lm
}

// initProbeEngine constructs the unified probe.Engine, wires it to
// the probes table and a fresh scheduler, registers V1.0 baseline
// Checkers (DNS + TLS), and parks it on s.probeEngine for the
// lifecycle to Start. The engine is *not* started here — that
// happens in Server.Start so probes don't run during partial
// server construction.
//
// V1.0 NMS expansion — Stage A1.8.
func (s *Server) initProbeEngine(db *database.DB) {
	sched := scheduler.New(probeSchedulerTick)

	probeEngine := probe.NewEngine(logging.GetLogger()).
		WithStorage(db.Probes(), sched)

	// Register V1.0 baseline checkers. Stage A1.7 will absorb the
	// remaining 11 internal/api/health_checks_*.go kinds.
	probeEngine.RegisterChecker(checkers.NewDNSChecker())
	probeEngine.RegisterChecker(checkers.NewTLSChecker())
	probeEngine.RegisterChecker(checkers.NewPingChecker())
	probeEngine.RegisterChecker(checkers.NewTCPChecker())
	probeEngine.RegisterChecker(checkers.NewUDPChecker())
	probeEngine.RegisterChecker(checkers.NewHTTPChecker())
	probeEngine.RegisterChecker(checkers.NewHTTPSChecker())
	probeEngine.RegisterChecker(checkers.NewRTSPChecker())
	probeEngine.RegisterChecker(checkers.NewDICOMChecker())

	s.probeEngine = probeEngine
	s.probeScheduler = sched
	if regErr := s.registerEngineIfLicensed(probeEngine); regErr != nil {
		logging.GetLogger().Warn("probe engine registry registration failed", "error", regErr)
	}

	// Wire the active-monitoring anomaly producer (ADR-0025): it subscribes to
	// the probe engine's ResultEvent channel here (so it does not miss events
	// before its own Start) and persists threshold breaches as anomalies under
	// source=probe. Subscription happens now; the consume/maintenance loops start
	// with the lifecycle. A catalog error is logged and skips the producer rather
	// than aborting probe startup.
	producer, prodErr := probeanomaly.New(
		probeEngine.Subscribe(), db.Anomalies(),
		probeanomaly.WithLogger(logging.GetLogger()),
	)
	if prodErr != nil {
		logging.GetLogger().Warn("probe anomaly producer not wired", "error", prodErr)
		return
	}
	s.probeAnomaly = producer
	if regErr := s.registerEngineIfLicensed(producer); regErr != nil {
		logging.GetLogger().Warn("probe anomaly producer registry registration failed", "error", regErr)
	}
}

// probeSchedulerTick is the scheduler's tick interval — how often
// it checks whether any registered Job is due. Production default
// 5s; tests can run faster via direct scheduler.New construction.
const probeSchedulerTick = 5 * time.Second

// initListeners wires the passive-ingress listeners (syslog UDP +
// SNMPv2c traps) into the engine registry. Both are opt-in via env
// variables — operators set SEED_SYSLOG_BIND / SEED_SNMP_TRAP_BIND
// (e.g. ":514", ":162") to enable them. Default is off because
// binding to <1024 requires elevated privileges and we don't want
// the server to crash out of the box when run as a non-root user.
//
// V1.0 NMS expansion — Stage A3.5e-4.
func (s *Server) initListeners(db *database.DB) {
	persistSink := listenersink.New(db.ListenerEvents(), logging.GetLogger(), nil)
	logger := logging.GetLogger()

	if addr := os.Getenv("SEED_SYSLOG_BIND"); addr != "" {
		l, err := syslog.New(syslog.Config{
			BindAddr: addr,
			Sink:     persistSink,
			Logger:   logger,
		})
		if err != nil {
			logger.Warn("syslog listener init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("syslog listener registry registration failed", "error", regErr)
		}
	}

	if addr := os.Getenv("SEED_SNMP_TRAP_BIND"); addr != "" {
		l, err := snmptrap.New(snmptrap.Config{
			BindAddr: addr,
			Sink:     persistSink,
			Logger:   logger,
		})
		if err != nil {
			logger.Warn("snmp trap listener init failed", "error", err)
		} else if regErr := s.registerEngineIfLicensed(l); regErr != nil {
			logger.Warn("snmp trap listener registry registration failed", "error", regErr)
		}
	}
}

// snmpPollerSchedulerTick is the cadence the snmp.Poller's
// scheduler wakes up at to dispatch due target jobs. The actual
// per-target cadence comes from polling_targets.poll_interval_seconds
// (default 300s); a 5s tick gives 1% scheduling-grain overhead on
// the default-cadence targets and is plenty fast for the 60s
// cadence operators typically use on high-priority devices.
const snmpPollerSchedulerTick = 5 * time.Second

// initSNMPPoller wires the orchestrator-built [*snmp.Poller] into
// the engine registry. Three things have to be true for the poller
// to do useful work:
//
//  1. The orchestrator needs a [snmp.ClientFactory] — we supply
//     the production gosnmp-backed one from internal/polling/snmp/
//     snmpclient.
//  2. There needs to be at least one row in polling_targets —
//     V1.0 operators populate this via the A5.3 CRUD API. With
//     zero rows the poller still starts (idempotent) but does
//     no work.
//  3. The scheduler needs a tick interval; snmpPollerSchedulerTick
//     defaults to 5s — see the doc comment for the rationale.
//
// V1.0 NMS expansion — Stage A5.4.
func (s *Server) initSNMPPoller(db *database.DB) {
	logger := logging.GetLogger()
	sched := scheduler.New(snmpPollerSchedulerTick)
	factory := snmpclient.NewFactory(snmpclient.Options{})
	poller, err := snmporchestrator.Build(snmporchestrator.Config{
		DB:            db,
		Scheduler:     sched,
		ClientFactory: factory,
		Logger:        logger,
	})
	if err != nil {
		logger.Warn("snmp poller init failed", "error", err)
		return
	}
	if regErr := s.registerEngineIfLicensed(poller); regErr != nil {
		logger.Warn("snmp poller registry registration failed", "error", regErr)
	}
}

// initTopologyReconcilers wires the four Stage A4 reconcilers
// (sysinfo, iftable, edge, arp) into the engine registry. They
// consume snmp_observations on a tick and maintain the fat-Node
// topology graph in topology_nodes / topology_interfaces /
// topology_links / topology_arp_bindings.
//
// V1.0 NMS expansion — Stage A4 wire-up.
func (s *Server) initTopologyReconcilers(db *database.DB) {
	logger := logging.GetLogger()
	obs := db.SNMPObservations()
	topo := db.Topology()
	settings := db.Settings()

	if r, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: obs, Nodes: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("sysinfo reconciler init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(r); regErr != nil {
		logger.Warn("sysinfo reconciler registry registration failed", "error", regErr)
	}

	if r, err := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("iftable reconciler init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(r); regErr != nil {
		logger.Warn("iftable reconciler registry registration failed", "error", regErr)
	}

	if r, err := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("edge reconciler init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(r); regErr != nil {
		logger.Warn("edge reconciler registry registration failed", "error", regErr)
	}

	if r, err := topology.NewARPReconciler(topology.ARPConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("arp reconciler init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(r); regErr != nil {
		logger.Warn("arp reconciler registry registration failed", "error", regErr)
	}
}

// initAlertPipelines wires the two Stage A4.5 / A4.6 alert
// pipelines into the engine registry. The listener pipeline scans
// listener_events for severe syslog + traps; the observation
// pipeline scans snmp_observations for state transitions
// (iface down, BGP flap, storage thresholds). Both write into the
// existing alerts table via the same Alert repository.
//
// V1.0 NMS expansion — Stage A4 wire-up.
func (s *Server) initAlertPipelines(db *database.DB) {
	logger := logging.GetLogger()
	settings := db.Settings()
	alerts := db.Alerts()
	suppressions := alertpipeline.NewDBSuppressionStore(db.AlertSuppressions())

	if p, err := alertpipeline.NewListenerPipeline(alertpipeline.ListenerConfig{
		Events:       db.ListenerEvents(),
		Alerts:       alerts,
		Settings:     settings,
		Logger:       logger,
		AlertRules:   db.AlertRules(),
		Suppressions: suppressions,
	}); err != nil {
		logger.Warn("listener alert pipeline init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(p); regErr != nil {
		logger.Warn("listener alert pipeline registry registration failed", "error", regErr)
	}

	if p, err := alertpipeline.NewObservationPipeline(alertpipeline.ObservationConfig{
		Observations: db.SNMPObservations(),
		Alerts:       alerts,
		Settings:     settings,
		Logger:       logger,
		Suppressions: suppressions,
	}); err != nil {
		logger.Warn("observation alert pipeline init failed", "error", err)
	} else if regErr := s.registerEngineIfLicensed(p); regErr != nil {
		logger.Warn("observation alert pipeline registry registration failed", "error", regErr)
	}
}

// initRetentionEngine constructs the unified retention engine and
// registers V1.0 sources (probe_results, metrics). The engine is
// tier-aware — it reads license.Manager on each pass — so in-place
// license upgrades take effect on the next tick.
//
// V1.0 NMS expansion — Stage A2.
func (s *Server) initRetentionEngine(db *database.DB) {
	retentionEngine := retention.New(
		licenseTierAdapter{lm: s.licenseMgr},
		logging.GetLogger(),
	)
	retentionEngine.Register(retention.NewProbeResultsSource(db))
	retentionEngine.Register(retention.NewMetricsSource(db))
	s.retentionEngine = retentionEngine
	if regErr := s.registerEngineIfLicensed(retentionEngine); regErr != nil {
		logging.GetLogger().Warn("retention engine registry registration failed", "error", regErr)
	}
}

// licenseTierAdapter satisfies retention.TierProvider by reading the
// active tier from license.Manager.GetState(). nil-safe — falls
// back to TierFree when no license manager is wired.
type licenseTierAdapter struct {
	lm *license.Manager
}

// GetTier returns the active tier, defaulting to Free when the
// license manager or its state is unavailable.
func (a licenseTierAdapter) GetTier() license.Tier {
	if a.lm == nil {
		return license.TierFree
	}
	state := a.lm.GetState()
	if state == nil {
		return license.TierFree
	}
	return state.Tier
}

// Service accessors — the in-package read interface and the lazy method
// values the ADR-0020 use-cases bind to (D1, formerly #888).

// GetConfig returns the server configuration.
func (s *Server) GetConfig() *config.Config { return s.config }

// AuthManager returns the authentication manager.
func (s *Server) AuthManager() *auth.Manager { return s.authMgr }

// CSRFManager returns the CSRF token manager.
func (s *Server) CSRFManager() *auth.CSRFManager { return s.csrf }

// SetupTokenManager returns the setup token manager.
func (s *Server) SetupTokenManager() *SetupTokenManager { return s.setupToken }

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

// DNSSecurityScanner returns the DNS security scanner.
func (s *Server) DNSSecurityScanner() *dns.SecurityScanner { return s.dnsSec }

// DHCPMonitor returns the DHCP monitor.
func (s *Server) DHCPMonitor() *dhcp.Monitor { return s.dhcpMon }

// RogueDetector returns the rogue DHCP detector.
func (s *Server) RogueDetector() *dhcp.RogueDetector { return s.rogueDet }

// GatewayTester returns the gateway tester.
func (s *Server) GatewayTester() *gateway.Tester { return s.gatewayTest }

// VLANManager returns the VLAN manager.
func (s *Server) VLANManager() *vlan.Manager { return s.vlanMgr }

// VLANTrafficMonitor returns the VLAN traffic monitor.
func (s *Server) VLANTrafficMonitor() *vlan.TrafficMonitor { return s.vlanTraffic }

// SpeedtestTester returns the speedtest tester.
func (s *Server) SpeedtestTester() *speedtest.Tester { return s.speedtestTest }

// IperfManager returns the iperf manager.
func (s *Server) IperfManager() *iperf.Manager { return s.iperfMgr }

// CableTester returns the cable tester.
func (s *Server) CableTester() *cable.Tester { return s.cableTest }

// PublicIPChecker returns the public IP checker.
func (s *Server) PublicIPChecker() *publicip.Checker { return s.publicIP }

// WiFiManager returns the WiFi manager.
func (s *Server) WiFiManager() *wifi.Manager { return s.wifiMgr }

// WiFiScanner returns the WiFi scanner.
func (s *Server) WiFiScanner() *wifi.Scanner { return s.wifiScan }

// SurveyManager returns the survey manager.
func (s *Server) SurveyManager() *survey.Manager { return s.surveyMgr }

// SSEHub returns the SSE hub.
func (s *Server) SSEHub() *SSEHub { return s.sse }

// LogBroadcaster returns the log broadcaster.
func (s *Server) LogBroadcaster() *logging.LogBroadcaster { return s.logBroadcast }

// DB returns the database connection.
func (s *Server) DB() *database.DB { return s.dbConn }

// MibDB returns the MIB database for SNMP OID resolution.
func (s *Server) MibDB() *mibdb.DB { return s.mibDB }

// Lowercase accessors — the unexported in-package read interface; many are
// also taken as method values by the ADR-0020 use-case wiring, so they must
// stay methods (not be inlined to field access).

func (s *Server) authManager() *auth.Manager                  { return s.authMgr }
func (s *Server) csrfManager() *auth.CSRFManager              { return s.csrf }
func (s *Server) setupTokenManager() *SetupTokenManager       { return s.setupToken }
func (s *Server) recoveryManager() *auth.RecoveryTokenManager { return s.recovery }
func (s *Server) oauthManager() *oauth.Manager                { return s.oauthMgr }
func (s *Server) trustedProxies() *TrustedProxies             { return s.proxies }
func (s *Server) webAuthnManager() *auth.WebAuthnManager      { return s.webAuthn }
func (s *Server) loginRateLimiter() *RateLimiter              { return s.loginLimiter }
func (s *Server) endpointRateLimiter() *EndpointRateLimiter   { return s.endpointLimiter }
func (s *Server) netManager() *netif.Manager                  { return s.netMgr }
func (s *Server) linkMonitor() *netif.LinkMonitor             { return s.linkMon }
func (s *Server) deviceDiscovery() *enumerate.DeviceDiscovery { return s.deviceDisc }
func (s *Server) discoveryService() *enumerate.Service        { return s.discoverySvc }
func (s *Server) discoveryEngine() *discovery.Engine          { return s.discoveryEng }
func (s *Server) problemDetector() *discovery.ProblemDetector { return s.problemDet }

// anomalyStore is the unified anomaly system of record (ADR-0021), the read
// source for the health-checks anomaly endpoint after the bespoke health
// detector was deleted. Nil-safe for the test harness (no DB wired).
func (s *Server) anomalyStore() *database.AnomalyRepository {
	if s.dbConn == nil {
		return nil
	}
	return s.dbConn.Anomalies()
}

func (s *Server) bluetoothScanner() *enumerate.BluetoothScanner { return s.bluetoothScan }
func (s *Server) vulnScanner() *vuln.VulnerabilityScanner       { return s.vulnScan }
func (s *Server) dnsTester() *dns.Tester                        { return s.dnsTest }
func (s *Server) dnsSecurityScanner() *dns.SecurityScanner      { return s.dnsSec }
func (s *Server) dhcpMonitor() *dhcp.Monitor                    { return s.dhcpMon }
func (s *Server) rogueDetector() *dhcp.RogueDetector            { return s.rogueDet }
func (s *Server) gatewayTester() *gateway.Tester                { return s.gatewayTest }
func (s *Server) vlanManager() *vlan.Manager                    { return s.vlanMgr }
func (s *Server) vlanTrafficMonitor() *vlan.TrafficMonitor      { return s.vlanTraffic }
func (s *Server) speedtestTester() *speedtest.Tester            { return s.speedtestTest }
func (s *Server) iperfManager() *iperf.Manager                  { return s.iperfMgr }
func (s *Server) cableTester() *cable.Tester                    { return s.cableTest }
func (s *Server) publicipChecker() *publicip.Checker            { return s.publicIP }
func (s *Server) wifiManager() *wifi.Manager                    { return s.wifiMgr }
func (s *Server) wifiScanner() *wifi.Scanner                    { return s.wifiScan }
func (s *Server) surveyManager() *survey.Manager                { return s.surveyMgr }
func (s *Server) sseHub() *SSEHub                               { return s.sse }
func (s *Server) logBroadcaster() *logging.LogBroadcaster       { return s.logBroadcast }
func (s *Server) eventBus() *events.Bus                         { return s.bus }
func (s *Server) jobsRunner() *jobs.Runner                      { return s.jobRunner }
func (s *Server) jobIdempotency() jobIdempotencyStore           { return s.jobIdemp }
func (s *Server) db() *database.DB                              { return s.dbConn }
func (s *Server) apiTokenRepo() *database.APITokenRepository    { return s.apiTokens }
func (s *Server) licenseManager() *license.Manager              { return s.licenseMgr }
func (s *Server) updateService() *update.Service                { return s.updateSvc }
func (s *Server) engineRegistry() *engine.Registry              { return s.engines }

// initWiFiUseCases wires the Wi-Fi troubleshooting use-cases (ADR-0020) from the
// composition root: the visibility-read, management, and discovery use-cases over
// the server's lazy accessors + live config. Called after initDiscovery so the
// discovery bridge the discovery use-case captures already exists.
func (s *Server) initWiFiUseCases() {
	s.wifiQueries = app.NewWiFiQueries(s.wifiVisibility)
	s.wifiManagement = app.NewWiFiManagement(s.wifiManager, s.wifiScanner, s.netManager, s.config, s.configPath)
	s.wifiDiscovery = app.NewWiFiDiscovery(s.wifiBridge)
}

// initDiscoveryUseCases wires the discovery use-cases (ADR-0020) from the
// composition root: the unified-discovery engine, the network problem detector,
// and the Bluetooth scanner, each over the server's lazy accessors so a nil or
// later-set collaborator (the test harness) is honored. The problem detector's
// scan reads the discovered devices through the device-discovery accessor.
func (s *Server) initDiscoveryUseCases() {
	s.discoveryDevices = app.NewDiscoveryDevices(s.discoveryEngine)
	s.networkProblems = app.NewProblems(s.problemDetector, s.discoveryService)
	s.bluetoothScans = app.NewBluetooth(s.bluetoothScanner)
}

// initHealthUseCases wires the health-monitoring use-case (ADR-0020) from the
// composition root over the server's lazy accessor for the unified anomaly store
// (the only remaining concern after the dead health_check_results read-path was
// deleted — ADR-0026), so a nil or later-set store (the test harness) is honored.
func (s *Server) initHealthUseCases() {
	s.healthMonitoring = app.NewHealthMonitoring(s.anomalyStore)
}

// initUpdateUseCases wires the update-lifecycle use-case (ADR-0020) from the
// composition root over the server's lazy accessor for the update service, so
// a nil or later-set service (the test harness) is honored.
func (s *Server) initUpdateUseCases() {
	s.updateLifecycle = app.NewUpdateLifecycle(s.updateService)
}

// initEngineUseCases wires the engine-status use-case (ADR-0020) from the
// composition root over the server's lazy accessor for the engine registry,
// so a nil or later-set registry (the api test harness) is honored.
func (s *Server) initEngineUseCases() {
	s.engineStatus = app.NewEngineStatus(s.engineRegistry)
}

// initIdentityUseCases wires the identity use-cases (ADR-0020, ADR-0024) from
// the composition root over the server's lazy accessors for the database, the
// token repository, and the license manager, so a nil or later-set collaborator
// (the api test harness) is honored.
func (s *Server) initIdentityUseCases() {
	s.identityUsers = app.NewIdentityUsers(s.db)
	s.identityTokens = app.NewIdentityTokens(s.apiTokenRepo, s.licenseManager)
	s.identityOAuth = app.NewIdentityOAuth(s.db)
}

// initUseCases wires the ADR-0020 application use-cases that depend on the
// discovery components existing: troubleshooting + discovery + health + update +
// identity + engine-status.
func (s *Server) initUseCases() {
	s.initWiFiUseCases()
	s.initDiscoveryUseCases()
	s.initHealthUseCases()
	s.initUpdateUseCases()
	s.initEngineUseCases()
	s.initIdentityUseCases()
}

// webAuthnConfigFromServer derives the relying-party config for the
// WebAuthn manager from the server config. The RPID is bound to the
// configured ACME domain when present (production) and falls back to
// the dev hostname otherwise (self-signed). Origins follow the same
// scheme + port: HTTPS uses https:// and the configured port, HTTP
// uses http://. Wave 3 (#85).
func webAuthnConfigFromServer(cfg *config.Config) auth.WebAuthnConfig {
	const (
		devHost     = "localhost"
		defaultPort = 8443
		schemeHTTPS = "https"
		schemeHTTP  = "http"
	)
	rpid := devHost
	if cfg != nil && cfg.Server.ACME.Domain != "" {
		rpid = cfg.Server.ACME.Domain
	}
	scheme := schemeHTTPS
	if cfg != nil && !cfg.Server.HTTPS {
		scheme = schemeHTTP
	}
	port := defaultPort
	if cfg != nil && cfg.Server.Port > 0 {
		port = cfg.Server.Port
	}
	origin := scheme + "://" + rpid
	if port != 443 && port != 80 {
		origin = origin + ":" + strconv.Itoa(port)
	}
	return auth.WebAuthnConfig{
		RPID:          rpid,
		RPDisplayName: "Seed",
		RPOrigins:     []string{origin},
	}
}
