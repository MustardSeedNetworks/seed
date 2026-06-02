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

	alertpipeline "github.com/krisarmstrong/seed/internal/alerts/pipeline"
	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/dhcp"
	"github.com/krisarmstrong/seed/internal/diagnostics/cable"
	"github.com/krisarmstrong/seed/internal/diagnostics/dns"
	"github.com/krisarmstrong/seed/internal/diagnostics/gateway"
	"github.com/krisarmstrong/seed/internal/diagnostics/iperf"
	"github.com/krisarmstrong/seed/internal/diagnostics/speedtest"
	"github.com/krisarmstrong/seed/internal/diagnostics/vlan"
	"github.com/krisarmstrong/seed/internal/health"
	"github.com/krisarmstrong/seed/internal/license"
	listenersink "github.com/krisarmstrong/seed/internal/listener/sink"
	"github.com/krisarmstrong/seed/internal/listener/snmptrap"
	"github.com/krisarmstrong/seed/internal/listener/syslog"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/mibdb"
	"github.com/krisarmstrong/seed/internal/netif"
	"github.com/krisarmstrong/seed/internal/oauth"
	"github.com/krisarmstrong/seed/internal/paths"
	"github.com/krisarmstrong/seed/internal/pipeline/publicip"
	snmporchestrator "github.com/krisarmstrong/seed/internal/polling/snmp/orchestrator"
	"github.com/krisarmstrong/seed/internal/polling/snmp/snmpclient"
	"github.com/krisarmstrong/seed/internal/probe"
	"github.com/krisarmstrong/seed/internal/probe/checkers"
	"github.com/krisarmstrong/seed/internal/scheduler"
	"github.com/krisarmstrong/seed/internal/services/discovery"
	"github.com/krisarmstrong/seed/internal/timeseries/retention"
	"github.com/krisarmstrong/seed/internal/topology"
	"github.com/krisarmstrong/seed/internal/wifi"
	"github.com/krisarmstrong/seed/internal/wifi/survey"
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
// Refactored to use ServiceContainer for dependency injection (#888).
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

	// Service container - holds all domain services (#888)
	services *ServiceContainer

	// Runtime state
	icmpAvailable      bool                  // Whether raw ICMP sockets are available
	startTime          time.Time             // Application start time for uptime tracking (fixes #540)
	setupModeStartTime time.Time             // Security fix #891: Track when setup mode started
	background         *BackgroundComponents // Long-lived components with background lifecycle (report scheduler)
	tlsFingerprint     tlsFingerprintCache   // Cached SHA-256 fingerprint of the active TLS cert, exposed via /__version
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

	// Wave 3 (#85): initialise the WebAuthn manager. The relying-party
	// ID and origins are derived from the server config; failures here
	// are non-fatal because the rest of the auth surface still works
	// without passkeys.
	if wan, wanErr := auth.NewWebAuthnManager(webAuthnConfigFromServer(cfg)); wanErr != nil {
		logging.GetLogger().Warn("WebAuthn manager init failed; passkeys disabled",
			"error", wanErr)
	} else {
		services.Auth.WebAuthn = wan
	}

	// Initialize rate limiters
	services.RateLimit.Login = NewRateLimiter(DefaultRateLimitConfig())
	services.RateLimit.Endpoint = NewEndpointRateLimiter(DefaultEndpointRateLimitConfig())

	// Initialize network services
	services.Network.Manager = netMgr
	services.Network.LinkMonitor = netif.NewLinkMonitor(cfg.Interface.Default)
	// LinkMonitorPool tracks every interface in the multi_interface set
	// (Pro). Reconcile primes the pool from the active profile; the pool
	// itself is not started here — server_lifecycle.go owns Start/Stop.
	services.Network.LinkMonitorPool = netif.NewLinkMonitorPool()
	primaryInterfaces := append(cfg.Interface.AllEthernet(), cfg.Interface.AllWiFi()...)
	services.Network.LinkMonitorPool.Reconcile(primaryInterfaces)

	// Initialize discovery services
	services.Discovery.Device = discovery.NewDeviceDiscoveryWithOUI(
		cfg.Interface.Default,
		cfg.NetworkDiscovery.OUIFilePath,
		cfg.NetworkDiscovery.OUIMaxAge,
	)
	// Note: services.Discovery.Service is initialized after profiler is created (see below)

	// Initialize SAP services
	services.Diagnostics.DNS = dns.NewTester("", cfg.DNS.TestHostname, dns.DefaultThresholds())
	services.Diagnostics.DNSSecurity = dns.NewSecurityScanner(dns.DefaultSecurityScanConfig())
	services.Diagnostics.DHCP = dhcp.NewMonitor(cfg.Interface.Default)
	services.Diagnostics.RogueDetector = dhcp.NewRogueDetector(&dhcp.RogueDetectorConfig{
		Interface:        cfg.Interface.Default,
		KnownServers:     cfg.DHCP.RogueDetection.KnownServers,
		AlertOnDetection: cfg.DHCP.RogueDetection.AlertOnDetection,
	})
	services.Diagnostics.Gateway = gateway.NewTester(gateway.DefaultThresholds())
	services.Diagnostics.VLAN = vlan.NewManager(cfg.Interface.Default)
	services.Diagnostics.VLANTraffic = vlan.NewTrafficMonitor(cfg.Interface.Default)
	services.Diagnostics.Speedtest = speedtest.NewTesterWithConfig(cfg.Speedtest.ServerID)
	services.Diagnostics.Iperf = iperf.NewManager()
	services.Diagnostics.Cable = cable.NewTester(cfg.Interface.Default)
	services.Diagnostics.PublicIP = publicip.NewChecker()

	// Initialize Canopy services
	services.Wireless.WiFi = wifi.NewManager(cfg.Interface.Default)
	services.Wireless.Scanner = wifi.NewScanner(cfg.Interface.Default)

	// Initialize database services
	services.Database.DB = db

	initDatabaseDependentServices(services, db)

	s := &Server{
		config:        cfg,
		configPath:    configPath,
		logPath:       logPath,
		mux:           http.NewServeMux(),
		icmpAvailable: icmpAvailable,
		startTime:     time.Now(),
		background:    background,
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

// initDatabaseDependentServices wires every service that needs a
// live database connection. Called from NewServer after services.Database.DB
// is populated. Splits into per-concern helpers to keep each scope
// focused and to keep NewServer under the funlen limit.
func initDatabaseDependentServices(services *ServiceContainer, db *database.DB) {
	if db == nil {
		// Tests construct a Server without a DB; skip the
		// database-dependent wiring entirely rather than crash.
		return
	}
	initLicenseAndAPITokens(services, db)
	initHealthServices(services, db)
	initProbeEngine(services, db)
	initRetentionEngine(services, db)
	initListeners(services, db)
	initTopologyReconcilers(services, db)
	initAlertPipelines(services, db)
	initSNMPPoller(services, db)
}

// initLicenseAndAPITokens wires the Phase D-2 license manager + API
// token repository into the service container. The license manager is
// best-effort: failure to load isn't fatal, the mint endpoint just
// behaves as if no paid license is present (rejects with 402).
func initLicenseAndAPITokens(services *ServiceContainer, db *database.DB) {
	services.Auth.APITokens = database.NewAPITokenRepository(db)
	lm, lmErr := license.NewManager()
	if lmErr != nil {
		logging.GetLogger().Warn("license manager init failed; minting will be disabled",
			"error", lmErr)
		return
	}
	services.Auth.License = lm
}

// initProbeEngine constructs the unified probe.Engine, wires it to
// the probes table and a fresh scheduler, registers V1.0 baseline
// Checkers (DNS + TLS), and parks it in services.Probe for the
// lifecycle to Start. The engine is *not* started here — that
// happens in Server.Start so probes don't run during partial
// service-container construction.
//
// V1.0 NMS expansion — Stage A1.8.
func initProbeEngine(services *ServiceContainer, db *database.DB) {
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

	services.Probe.Engine = probeEngine
	services.Probe.Scheduler = sched
	if regErr := registerEngineIfLicensed(services, probeEngine); regErr != nil {
		logging.GetLogger().Warn("probe engine registry registration failed", "error", regErr)
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
func initListeners(services *ServiceContainer, db *database.DB) {
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
		} else if regErr := registerEngineIfLicensed(services, l); regErr != nil {
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
		} else if regErr := registerEngineIfLicensed(services, l); regErr != nil {
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
func initSNMPPoller(services *ServiceContainer, db *database.DB) {
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
	if regErr := registerEngineIfLicensed(services, poller); regErr != nil {
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
func initTopologyReconcilers(services *ServiceContainer, db *database.DB) {
	logger := logging.GetLogger()
	obs := db.SNMPObservations()
	topo := db.Topology()
	settings := db.Settings()

	if r, err := topology.NewSysInfoReconciler(topology.Config{
		Observations: obs, Nodes: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("sysinfo reconciler init failed", "error", err)
	} else if regErr := registerEngineIfLicensed(services, r); regErr != nil {
		logger.Warn("sysinfo reconciler registry registration failed", "error", regErr)
	}

	if r, err := topology.NewIfTableReconciler(topology.IfTableConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("iftable reconciler init failed", "error", err)
	} else if regErr := registerEngineIfLicensed(services, r); regErr != nil {
		logger.Warn("iftable reconciler registry registration failed", "error", regErr)
	}

	if r, err := topology.NewEdgeReconciler(topology.EdgeConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("edge reconciler init failed", "error", err)
	} else if regErr := registerEngineIfLicensed(services, r); regErr != nil {
		logger.Warn("edge reconciler registry registration failed", "error", regErr)
	}

	if r, err := topology.NewARPReconciler(topology.ARPConfig{
		Observations: obs, Store: topo, Settings: settings, Logger: logger,
	}); err != nil {
		logger.Warn("arp reconciler init failed", "error", err)
	} else if regErr := registerEngineIfLicensed(services, r); regErr != nil {
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
func initAlertPipelines(services *ServiceContainer, db *database.DB) {
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
	} else if regErr := registerEngineIfLicensed(services, p); regErr != nil {
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
	} else if regErr := registerEngineIfLicensed(services, p); regErr != nil {
		logger.Warn("observation alert pipeline registry registration failed", "error", regErr)
	}
}

// initRetentionEngine constructs the unified retention engine and
// registers V1.0 sources (probe_results, metrics). The engine is
// tier-aware — it reads license.Manager on each pass — so in-place
// license upgrades take effect on the next tick.
//
// V1.0 NMS expansion — Stage A2.
func initRetentionEngine(services *ServiceContainer, db *database.DB) {
	retentionEngine := retention.New(
		licenseTierAdapter{lm: services.Auth.License},
		logging.GetLogger(),
	)
	retentionEngine.Register(retention.NewProbeResultsSource(db))
	retentionEngine.Register(retention.NewMetricsSource(db))
	services.Probe.Retention = retentionEngine
	if regErr := registerEngineIfLicensed(services, retentionEngine); regErr != nil {
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

// initHealthServices wires the previously-dead health subsystem
// (Scorer, SLATracker, AnomalyDetector, DependencyMgr) into the
// service container. Stage A1.6 — these services existed in code
// since prior phases but were declared-but-never-assigned in
// services.HealthServices, so the health-check API endpoints
// returned HTTP 503 on every request. This wires them up.
//
// AlertManager and Repository are wired elsewhere (existing code);
// this function only handles the four previously-dead services.
func initHealthServices(services *ServiceContainer, db *database.DB) {
	services.Health.Repository = db.HealthChecks()

	logger := logging.GetLogger()
	services.Health.Scorer = health.NewScoringService(db, logger)

	services.Health.SLATracker = health.NewSLATracker(health.SLATrackerConfig{
		Repository: services.Health.Repository,
	})

	services.Health.AnomalyDetector = health.NewAnomalyDetector(health.AnomalyDetectorConfig{})

	services.Health.DependencyMgr = health.NewDependencyManager(health.DependencyManagerConfig{})
}

// Service accessors - provide backwards-compatible access to services (#888)

// GetConfig returns the server configuration.
func (s *Server) GetConfig() *config.Config { return s.config }

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
func (s *Server) DNSTester() *dns.Tester { return s.services.Diagnostics.DNS }

// DNSSecurityScanner returns the DNS security scanner.
func (s *Server) DNSSecurityScanner() *dns.SecurityScanner { return s.services.Diagnostics.DNSSecurity }

// DHCPMonitor returns the DHCP monitor.
func (s *Server) DHCPMonitor() *dhcp.Monitor { return s.services.Diagnostics.DHCP }

// RogueDetector returns the rogue DHCP detector.
func (s *Server) RogueDetector() *dhcp.RogueDetector { return s.services.Diagnostics.RogueDetector }

// GatewayTester returns the gateway tester.
func (s *Server) GatewayTester() *gateway.Tester { return s.services.Diagnostics.Gateway }

// VLANManager returns the VLAN manager.
func (s *Server) VLANManager() *vlan.Manager { return s.services.Diagnostics.VLAN }

// VLANTrafficMonitor returns the VLAN traffic monitor.
func (s *Server) VLANTrafficMonitor() *vlan.TrafficMonitor { return s.services.Diagnostics.VLANTraffic }

// SpeedtestTester returns the speedtest tester.
func (s *Server) SpeedtestTester() *speedtest.Tester { return s.services.Diagnostics.Speedtest }

// IperfManager returns the iperf manager.
func (s *Server) IperfManager() *iperf.Manager { return s.services.Diagnostics.Iperf }

// CableTester returns the cable tester.
func (s *Server) CableTester() *cable.Tester { return s.services.Diagnostics.Cable }

// PublicIPChecker returns the public IP checker.
func (s *Server) PublicIPChecker() *publicip.Checker { return s.services.Diagnostics.PublicIP }

// WiFiManager returns the WiFi manager.
func (s *Server) WiFiManager() *wifi.Manager { return s.services.Wireless.WiFi }

// WiFiScanner returns the WiFi scanner.
func (s *Server) WiFiScanner() *wifi.Scanner { return s.services.Wireless.Scanner }

// SurveyManager returns the survey manager.
func (s *Server) SurveyManager() *survey.Manager { return s.services.Wireless.Survey }

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
func (s *Server) webAuthnManager() *auth.WebAuthnManager      { return s.services.Auth.WebAuthn }
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
func (s *Server) dnsTester() *dns.Tester                   { return s.services.Diagnostics.DNS }
func (s *Server) dnsSecurityScanner() *dns.SecurityScanner { return s.services.Diagnostics.DNSSecurity }
func (s *Server) dhcpMonitor() *dhcp.Monitor               { return s.services.Diagnostics.DHCP }
func (s *Server) rogueDetector() *dhcp.RogueDetector       { return s.services.Diagnostics.RogueDetector }
func (s *Server) gatewayTester() *gateway.Tester           { return s.services.Diagnostics.Gateway }
func (s *Server) vlanManager() *vlan.Manager               { return s.services.Diagnostics.VLAN }
func (s *Server) vlanTrafficMonitor() *vlan.TrafficMonitor { return s.services.Diagnostics.VLANTraffic }
func (s *Server) speedtestTester() *speedtest.Tester       { return s.services.Diagnostics.Speedtest }
func (s *Server) iperfManager() *iperf.Manager             { return s.services.Diagnostics.Iperf }
func (s *Server) cableTester() *cable.Tester               { return s.services.Diagnostics.Cable }
func (s *Server) publicipChecker() *publicip.Checker       { return s.services.Diagnostics.PublicIP }
func (s *Server) wifiManager() *wifi.Manager               { return s.services.Wireless.WiFi }
func (s *Server) wifiScanner() *wifi.Scanner               { return s.services.Wireless.Scanner }
func (s *Server) surveyManager() *survey.Manager           { return s.services.Wireless.Survey }
func (s *Server) sseHub() *SSEHub                          { return s.services.RealTime.SSEHub }
func (s *Server) logBroadcaster() *logging.LogBroadcaster  { return s.services.RealTime.LogBroadcaster }
func (s *Server) db() *database.DB                         { return s.services.Database.DB }

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
