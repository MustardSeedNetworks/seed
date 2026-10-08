package api

// server_init.go contains the per-subsystem initialisation helpers that
// NewServer composes: DNS/discovery, target networks, database +
// migration, the database-dependent engines, MIB DB, SSE + log broadcaster,
// discovery pipeline, vulnerability scanner, CORS origin policy, the
// retention engine, and the health and settings use-cases.

import (
	"context"
	"slices"

	alertdelivery "github.com/MustardSeedNetworks/seed/internal/alerts/delivery"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/export"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
	"github.com/MustardSeedNetworks/seed/internal/discovery/fingerprint"
	"github.com/MustardSeedNetworks/seed/internal/discovery/resolve"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
)

// initNetworkServices initializes DNS servers and device discovery subnets.
func (s *Server) initNetworkServices(cfg *config.Config) {
	// Initialize DNS tester with configured servers from config
	if len(cfg.DNS.Servers) > 0 {
		configuredServers := make([]dns.ConfiguredServer, 0, len(cfg.DNS.Servers))
		for _, d := range cfg.DNS.Servers {
			configuredServers = append(configuredServers, dns.ConfiguredServer{
				Address: d.Address,
				Enabled: d.Enabled,
			})
		}
		s.dnsTester().SetConfiguredServers(configuredServers)
	}
	s.dnsTester().SetConfiguredServerLimit(s.dnsServerLimit)

	// Initialize device discovery with configured target networks
	s.initTargetNetworks(cfg)
}

// initTargetNetworks configures device discovery with target networks from config.
func (s *Server) initTargetNetworks(cfg *config.Config) {
	if len(cfg.NetworkDiscovery.TargetNetworks) == 0 {
		return
	}

	enabledCIDRs := s.collectEnabledSubnets(cfg)
	if len(enabledCIDRs) == 0 {
		return
	}

	if err := s.deviceDiscovery().SetTargetNetworks(enabledCIDRs); err != nil {
		logging.GetLogger().Warn("Failed to set target networks", "error", err)
		return
	}

	logging.GetLogger().Info("Configured target networks for scanning", "count", len(enabledCIDRs))
}

// collectEnabledSubnets extracts enabled subnet CIDRs from configuration.
func (s *Server) collectEnabledSubnets(cfg *config.Config) []string {
	enabledCIDRs := make([]string, 0, len(cfg.NetworkDiscovery.TargetNetworks))
	for _, subnet := range cfg.NetworkDiscovery.TargetNetworks {
		if subnet.Enabled {
			enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
		}
	}
	return enabledCIDRs
}

// initDatabaseServices configures database-backed services if a database is
// wired.
func (s *Server) initDatabaseServices(cfg *config.Config) {
	if s.dbConn == nil {
		return
	}

	// Set up database-backed user store for authentication
	userStore := app.NewAuthUserStore(s.dbConn)
	s.authManager().SetUserStore(userStore)

	// Migrate admin user from config to database if needed
	// This ensures backward compatibility during the transition
	if cfg.Auth.DefaultPasswordHash != "" &&
		cfg.Auth.DefaultPasswordHash != auth.SetupModePlaceholder {
		if err := userStore.MigrateUserFromConfig(
			context.Background(),
			cfg.Auth.DefaultUsername,
			cfg.Auth.DefaultPasswordHash,
		); err != nil {
			logging.GetLogger().Error("Failed to migrate user from config", "error", err)
		} else {
			logging.GetLogger().Info("User migrated from config to database", "username", cfg.Auth.DefaultUsername)
		}
	}

	// Initialize MIB database for SNMP OID resolution
	s.initMibDatabase()

	// Start the background maintenance loop (fixes #848). s.dbConn is non-nil here
	// (the function returns early otherwise), so it always runs: it sweeps jobs
	// retention every tick (the runner map + jobs table always grow) and applies
	// the data-retention policy when a positive window is configured.
	s.retentionStopCh = make(chan struct{})
	go s.startMaintenance(cfg.Database.RetentionDays)
}

// initDatabaseDependentServices wires every service that needs a
// live database connection. Called from NewServer after s.dbConn
// is populated. Splits into per-concern helpers to keep each scope
// focused and to keep NewServer under the funlen limit.
func (s *Server) initDatabaseDependentServices() {
	db := s.dbConn
	if db == nil {
		// Tests construct a Server without a DB; skip the
		// database-dependent wiring entirely rather than crash.
		return
	}
	engines := app.NewEnginePersistence(db)
	s.initLicenseAndAPITokens(db)
	s.initAnomalyPlatform(engines.Anomalies)
	s.initProbeEngine(db)
	s.initRetentionEngine(engines.Rollups)
	s.initTelemetry(engines.Telemetry)
	s.initListeners(app.NewListenerPersistence(db))
	s.initTopologyReconcilers(db)
	s.initAlertPipelines(db)
	s.initSNMPPoller(engines.SNMPPoller)
}

// initMibDatabase initializes the MIB database and loads built-in OID definitions.
func (s *Server) initMibDatabase() {
	mibDB := app.NewMIBDatabase(s.dbConn)
	s.mibDB = mibDB

	// Load built-in OID definitions (918+ standard OIDs from RFC MIBs)
	if err := mibDB.LoadBuiltinOIDs(); err != nil {
		logging.GetLogger().Error("Failed to load built-in MIB OIDs", "error", err)
		return
	}

	// Log statistics
	stats, err := mibDB.Stats()
	if err != nil {
		logging.GetLogger().Warn("Failed to get MIB database stats", "error", err)
		return
	}
	logging.GetLogger().Info("MIB database initialized",
		"oid_entries", stats["oid_entries"],
		"mib_count", stats["mib_count"])
}

// initSSEAndLogging initializes the SSE hub and log broadcaster.
func (s *Server) initSSEAndLogging() {
	db := s.dbConn
	// Initialize SSE hub for real-time updates
	s.sse = NewSSEHub()
	go s.sseHub().Run()

	// Initialize log broadcaster for real-time log streaming
	s.logBroadcast = logging.InitBroadcaster(logBroadcasterBufferSize)

	// Initialize the in-process event bus and the unified job runner (ADR-0004 /
	// ADR-0005). The runner publishes job state changes onto the bus; the
	// /api/v1/jobs surface (POST/GET/DELETE + the /jobs/events SSE stream)
	// adapts it. No job kinds are registered yet — they arrive as the real
	// long-ops are migrated in a later slice; both Close() on shutdown.
	s.bus = events.New(logging.GetLogger())
	jobsCfg := jobs.Config{Retention: jobsRetention}
	var jobStore *app.JobStore
	if db != nil {
		// Durable backing (Phase 5c): the runner write-throughs lifecycle
		// transitions so a job survives a restart. Without a database the runner
		// stays in-memory only (the fail-cleanly v1).
		jobStore = app.NewJobStore(db)
		jobsCfg.Store = jobStore
	}
	s.jobRunner = jobs.New(
		s.bus, logging.GetLogger(), jobsCfg,
	)
	if jobStore != nil {
		// Durable Idempotency-Key dedup (Phase 5c-4): survives restart, so a
		// client retry across a restart still replays rather than duplicating.
		s.jobIdemp = newDBJobIdempotency(jobStore, logging.GetLogger())
	} else {
		s.jobIdemp = newJobIdempotencyCache(jobIdempotencyCapacity)
	}
	s.registerJobKinds()

	// Reconcile jobs left in-flight by a previous process: their handler
	// goroutines died with that process, so they can never complete and are
	// transitioned to failed. No-op when the runner has no durable store.
	if db != nil {
		if n, recErr := s.jobsRunner().Recover(context.Background()); recErr != nil {
			logging.GetLogger().Warn("job recovery failed", "error", recErr)
		} else if n > 0 {
			logging.GetLogger().Info("recovered interrupted jobs from a prior run", "count", n)
		}
	}

	// Transactional-outbox relay (ADR-0017): drains durable events written in a
	// domain transaction and republishes them post-commit onto the same bus. It
	// needs both the bus (built just above) and a durable store, so it is created
	// here and attached to the background components — started/stopped with them.
	// Dormant until a producer enqueues (no producer is rewired today; the jobs
	// runner keeps publishing directly). Subscribers register before Start, so a
	// future durable consumer never misses the startup replay.
	if db != nil && s.background != nil {
		s.background.Outbox = outbox.NewRelay(
			app.NewOutboxStore(db), s.eventBus(), logging.GetLogger(),
		)
	}

	// Wire up database persistence for logs if database is available
	if db != nil {
		s.logBroadcaster().SetDBWriter(app.NewLogWriter(db))
		logging.GetLogger().
			Info("Log broadcaster initialized with database persistence", "buffer_size", logBroadcasterBufferSize)
	} else {
		logging.GetLogger().Info(
			"Log broadcaster initialized (memory-only, no database)",
			"buffer_size",
			logBroadcasterBufferSize,
		)
	}

	// Wire up database persistence for devices if database is available
	if db != nil {
		s.deviceDiscovery().SetDBWriter(app.NewDeviceWriter(db))
		logging.GetLogger().Info("Device discovery initialized with database persistence")
	}
}

// initDiscovery initializes the shared discovery profiler, port scanner, and
// the discovery service. (The legacy pipeline orchestrator was retired in
// Phase 7 — discovery now runs through the engine + jobs spine.)
func (s *Server) initDiscovery(cfg *config.Config) {
	// Create SHARED DeviceProfiler - used by Service and Engine
	// This ensures port scan results and SNMP data are consistent across the system
	sharedProfiler := discovery.NewDeviceProfiler(discovery.DefaultProfilerConfig(), s.snmpCreds)
	s.profiler = sharedProfiler

	// Create PortScanner for Engine
	portScanner, err := fingerprint.NewPortScanner(portScannerTimeout)
	if err != nil {
		logging.GetLogger().Warn("Failed to create port scanner", "error", err)
	} else {
		s.portScanner = portScanner
	}

	// The service sweeps s.deviceDisc, the registry the API lists and the
	// discovery settings feed, never one of its own (seed#2831).
	if s.deviceDisc != nil {
		s.deviceDisc.SetSNMPCredentials(s.snmpCreds)
	}
	s.discoverySvc = enumerate.NewService(cfg, s.deviceDisc, sharedProfiler)
	logging.GetLogger().Info("Discovery service initialized with shared profiler")
}

// initVulnerabilityScanner initializes the vulnerability scanner if enabled.
func (s *Server) initVulnerabilityScanner(cfg *config.Config) {
	if !cfg.Security.VulnerabilityScanning.Enabled {
		return
	}

	scannerCfg := &vuln.VulnerabilityScannerConfig{
		Enabled:           cfg.Security.VulnerabilityScanning.Enabled,
		CVEDatabase:       cfg.Security.VulnerabilityScanning.CVEDatabase,
		NVDAPIKey:         cfg.Security.VulnerabilityScanning.NVDAPIKey,
		UpdateInterval:    cfg.Security.VulnerabilityScanning.UpdateInterval,
		SeverityThreshold: cfg.Security.VulnerabilityScanning.SeverityThreshold,
		MaxConcurrent:     cfg.Security.VulnerabilityScanning.MaxConcurrent,
	}

	vulnScanner, err := vuln.NewVulnerabilityScanner(scannerCfg)
	if err != nil {
		logging.GetLogger().Warn("Failed to initialize vulnerability scanner", "error", err)
		return
	}
	if s.dbConn != nil {
		vulnScanner.SetStore(app.NewVulnStore(s.dbConn))
	}
	s.vulnScan = vulnScanner
	logging.GetLogger().Info("Vulnerability scanner initialized",
		"cve_database", scannerCfg.CVEDatabase, "threshold", scannerCfg.SeverityThreshold)

	// Initialize problem detector for network issue detection
	s.problemDet = discovery.NewProblemDetector()
	logging.GetLogger().Info("Problem detector initialized")

	// Initialize Bluetooth scanner
	btConfig := enumerate.DefaultBluetoothScanConfig()
	var ouiDB *resolve.OUIDatabase
	if s.deviceDisc != nil {
		ouiDB = s.deviceDisc.GetOUIDatabase()
	}
	s.bluetoothScan = enumerate.NewBluetoothScanner("", btConfig, ouiDB)
	logging.GetLogger().Info("Bluetooth scanner initialized")

	// Initialize WiFi bridge connecting wifi to discovery
	if s.wifiScan != nil {
		wifiBridgeConfig := enumerate.DefaultWiFiBridgeConfig()
		s.wifiBridgeSvc = enumerate.NewWiFiBridge(
			s.wifiScan,
			s.wifiMgr,
			ouiDB,
			wifiBridgeConfig,
		)
		logging.GetLogger().Info("WiFi bridge initialized")
	}

	// Initialize Discovery Engine (primary unified discovery system)
	engineConfig := discovery.DefaultEngineConfig()
	s.discoveryEng = discovery.NewEngine(engineConfig)

	// Wire in all collectors
	if s.deviceDisc != nil {
		s.discoveryEng.SetWiredCollector(s.deviceDisc)
	}
	if s.wifiBridgeSvc != nil {
		s.discoveryEng.SetWiFiCollector(s.wifiBridgeSvc)
	}
	if s.bluetoothScan != nil {
		s.discoveryEng.SetBluetoothCollector(s.bluetoothScan)
	}
	if s.profiler != nil {
		s.discoveryEng.SetProfiler(s.profiler)
	}
	if s.portScanner != nil {
		s.discoveryEng.SetPortScanner(s.portScanner)
	}
	if s.vulnScan != nil {
		// ADR-0018: the vuln assessment stage is a subpackage adapter injected
		// as the engine's Assessor port, built over the engine's registry + bus.
		s.discoveryEng.SetAssessor(vuln.NewStage(
			s.vulnScan,
			s.discoveryEng.Registry(),
			s.discoveryEng.EventBus(),
		))
	}

	// Start the engine
	if startErr := s.discoveryEng.Start(context.Background()); startErr != nil {
		logging.GetLogger().Error("Failed to start discovery engine", "error", startErr)
	} else {
		logging.GetLogger().Info("Discovery engine started",
			"capabilities", s.discoveryEng.GetCapabilities(),
		)
	}
}

// initSecurityOrigins configures allowed origins for CORS.
func (s *Server) initSecurityOrigins(cfg *config.Config) {
	getOriginState().setAllowedOrigins(cfg.Security.AllowedOrigins)

	if len(cfg.Security.AllowedOrigins) == 0 {
		logging.GetLogger().Info("Using default RFC 1918 private network origins for CORS")
		return
	}

	// Check for wildcard origin (fixes #715).
	s.logWildcardOriginWarning(cfg)

	logging.GetLogger().Info(
		"Configured explicit allowed origins for CORS",
		"count",
		len(cfg.Security.AllowedOrigins),
	)
}

// logWildcardOriginWarning logs appropriate warnings for wildcard origin configuration.
func (s *Server) logWildcardOriginWarning(cfg *config.Config) {
	if !slices.Contains(cfg.Security.AllowedOrigins, "*") {
		return
	}

	logging.GetLogger().Warn(
		"SECURITY WARNING: Wildcard origin (*) allows all origins",
		"recommendation",
		"Configure explicit allowed origins in Security.AllowedOrigins",
	)
}

// newDiscoverySNMPCredentials builds discovery's vault-backed credential
// source (#2118). It returns nil — and discovery then probes no SNMP at all —
// when there is no database or no keyring to decrypt with, because the only
// other way to answer an SNMP probe would be the plaintext file-config
// communities #1799 removed.
func (s *Server) newDiscoverySNMPCredentials(cfg *config.Config) discovery.SNMPCredentialProvider {
	if cfg == nil || s.dbConn == nil {
		logging.GetLogger().Warn("SNMP discovery disabled: no credential vault available")
		return nil
	}
	keyring, err := cfg.CredentialKeyring()
	if err != nil {
		logging.GetLogger().Warn("SNMP discovery disabled: no credential keyring", "error", err)
		return nil
	}
	creds, err := app.NewDiscoverySNMPCredentials(s.dbConn, keyring, &cfg.SNMP)
	if err != nil {
		logging.GetLogger().Warn("SNMP discovery disabled", "error", err)
		return nil
	}
	return creds
}

// initRetentionEngine constructs the unified retention engine and
// registers its rollup sources. The engine is
// tier-aware — it reads license.Manager on each pass — so in-place
// license upgrades take effect on the next tick.
//
// V1.0 NMS expansion — Stage A2.
func (s *Server) initRetentionEngine(sources []retention.RollupSource) {
	retentionEngine := retention.New(
		licenseTierAdapter{lm: s.licenseMgr},
		logging.GetLogger(),
	)
	for _, src := range sources {
		retentionEngine.Register(src)
	}
	s.retentionEngine = retentionEngine
	if regErr := s.registerEngineIfLicensed(retentionEngine); regErr != nil {
		logging.GetLogger().Warn("retention engine registry registration failed", "error", regErr)
	}
}

// licenseTierAdapter satisfies retention.TierProvider with the tier the
// license manager grants now. nil-safe — falls back to TierFree when no
// license manager is wired.
type licenseTierAdapter struct {
	lm *license.Manager
}

// GetTier returns the granted tier, Free when there is no manager or no live
// grant.
func (a licenseTierAdapter) GetTier() license.Tier {
	if a.lm == nil {
		return license.TierFree
	}
	return license.EffectiveTier(a.lm)
}

// initHealthUseCases wires the health-monitoring use-case (ADR-0020) from the
// composition root over the server's lazy accessor for the unified anomaly store
// (the only remaining concern after the dead health_check_results read-path was
// deleted — ADR-0026), so a nil or later-set store (the test harness) is honored.
func (s *Server) initHealthUseCases() {
	s.healthMonitoring = app.NewHealthMonitoring(s.anomalyStore, s.anomalyEngine)
	s.healthSettings = app.NewHealthSettings(
		s.healthProbeRepo, s.rescheduleProbeEngine,
		s.config, s.configPath, s.dnsTester, s.speedtestTester,
		s.healthSettingsRepo,
	)
}

// initDiscoveryUseCases wires the discovery use-cases (ADR-0020) from the
// composition root: the unified-discovery engine, the network problem detector,
// and the Bluetooth scanner, each over the server's lazy accessors so a nil or
// later-set collaborator (the test harness) is honored. The problem detector's
// scan reads the discovered devices through the device-discovery accessor.
func (s *Server) initDiscoveryUseCases() {
	s.discoveryDevices = app.NewDiscoveryDevices(s.discoveryEngine)
	s.discoverySettings = app.NewDiscoverySettings(
		s.config, s.configPath, s.deviceDiscovery, s.discoveryService,
	)
	s.networkProblems = app.NewProblems(s.problemDetector, s.discoveryService)
	s.topologyQueries = app.NewTopologyQueries(s.db, topologyMaxLimit)
	s.exportService = export.NewService(serverExportSources{s: s})
	s.logQuery = app.NewLogQuery(s.db)
	s.historyQueries = app.NewHistory(s.db)
	s.vulnTriage = app.NewVulnTriage(s.db)
	s.flows = app.NewFlows(s.db)
	s.interfaceStats = app.NewInterfaceStats(s.db)
	s.pollingTargets = app.NewPollingTargets(s.db, s.pollingTargetLimit)
	// The credential vault needs the keyring that owns the DEK. Without a
	// config there is none, so the use-case stays nil and its handlers report
	// 503 — the alternative is a CRUD surface that would persist plaintext.
	if s.config != nil {
		if keyring, err := s.config.CredentialKeyring(); err == nil {
			if svc, credErr := app.NewDeviceCredentials(s.db, keyring); credErr == nil {
				s.deviceCredentials = svc
			}
		}
	}
	s.alertInbox = app.NewAlertInbox(s.db)
}

// initSettingsUseCases wires the ADR-0020 settings, profiles, network-IP, and
// alert-rule use-cases. The composition root builds the adapters; api passes
// its lazy db/manager accessors + live config. Split out of NewServer to
// keep it under the funlen limit.
func (s *Server) initSettingsUseCases() {
	s.settingsStore = app.NewSettings(s.db, s.config)
	s.settingsManagement = app.NewSettingsManagement(s.config, s.configPath,
		func() *alertdelivery.Manager { return s.alertDelivery })
	s.configBackups = app.NewConfigBackups(s.config, s.configPath,
		func() *alertdelivery.Manager { return s.alertDelivery })
	s.securitySettings = app.NewSecuritySettings(s.config, s.configPath, s.rogueDetector)
	s.profiles = app.NewProfiles(s.db, s.config, s.configPath)
	s.networkIP = app.NewNetworkIP(s.netManager, s.config, s.configPath)
	s.alertRules = app.NewAlertRules(s.db)
}
