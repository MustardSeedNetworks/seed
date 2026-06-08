package api

// server_init.go contains the per-subsystem initialisation helpers that
// NewServer composes: DNS/discovery/survey, additional subnets, database +
// migration, MIB DB, SSE + log broadcaster, discovery pipeline, vulnerability
// scanner, and CORS origin policy.

import (
	"context"
	"slices"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/mibdb"
	"github.com/MustardSeedNetworks/seed/internal/platform/events"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
	"github.com/MustardSeedNetworks/seed/internal/platform/outbox"
	"github.com/MustardSeedNetworks/seed/internal/wifi/survey"
)

// initNetworkServices initializes DNS servers, device discovery subnets, and survey manager.
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

	// Initialize device discovery with configured additional subnets
	s.initAdditionalSubnets(cfg)

	// Initialize survey manager
	surveyStoragePath := "data/surveys"
	s.services.Wireless.Survey = survey.NewManager(
		surveyStoragePath,
		s.wifiScanner(),
		s.wifiManager(),
		s.iperfManager(),
	)
	if err := s.surveyManager().LoadSurveys(); err != nil {
		logging.GetLogger().Warn("Failed to load surveys", "error", err)
	}
}

// initAdditionalSubnets configures device discovery with additional subnets from config.
func (s *Server) initAdditionalSubnets(cfg *config.Config) {
	if len(cfg.NetworkDiscovery.AdditionalSubnets) == 0 {
		return
	}

	enabledCIDRs := s.collectEnabledSubnets(cfg)
	if len(enabledCIDRs) == 0 {
		return
	}

	if err := s.deviceDiscovery().SetAdditionalSubnets(enabledCIDRs); err != nil {
		logging.GetLogger().Warn("Failed to set additional subnets", "error", err)
		return
	}

	logging.GetLogger().Info("Configured additional subnets for scanning", "count", len(enabledCIDRs))
}

// collectEnabledSubnets extracts enabled subnet CIDRs from configuration.
func (s *Server) collectEnabledSubnets(cfg *config.Config) []string {
	enabledCIDRs := make([]string, 0, len(cfg.NetworkDiscovery.AdditionalSubnets))
	for _, subnet := range cfg.NetworkDiscovery.AdditionalSubnets {
		if subnet.Enabled {
			enabledCIDRs = append(enabledCIDRs, subnet.CIDR)
		}
	}
	return enabledCIDRs
}

// initDatabaseServices configures database-backed services if db is available.
func (s *Server) initDatabaseServices(cfg *config.Config, db *database.DB) {
	if db == nil {
		return
	}

	// Set up database-backed user store for authentication
	userStore := database.NewUserStoreAdapter(db)
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
	s.initMibDatabase(db)

	// Start the background maintenance loop (fixes #848). db is non-nil here
	// (the function returns early otherwise), so it always runs: it sweeps jobs
	// retention every tick (the runner map + jobs table always grow) and applies
	// the data-retention policy when a positive window is configured.
	s.services.Database.RetentionStopCh = make(chan struct{})
	go s.startMaintenance(cfg.Database.RetentionDays)
}

// initMibDatabase initializes the MIB database and loads built-in OID definitions.
func (s *Server) initMibDatabase(db *database.DB) {
	// Create MIB database interface using the underlying SQL connection
	mibDB := mibdb.New(db.Conn())
	s.services.Database.MibDB = mibDB

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
func (s *Server) initSSEAndLogging(db *database.DB) {
	// Initialize SSE hub for real-time updates
	s.services.RealTime.SSEHub = NewSSEHub()
	go s.sseHub().Run()

	// Initialize log broadcaster for real-time log streaming
	s.services.RealTime.LogBroadcaster = logging.InitBroadcaster(logBroadcasterBufferSize)
	s.logBroadcaster().SetBroadcaster(&sseLogBroadcastAdapter{hub: s.sseHub()})

	// Initialize the in-process event bus and the unified job runner (ADR-0004 /
	// ADR-0005). The runner publishes job state changes onto the bus; the
	// /api/v1/jobs surface (POST/GET/DELETE + the /jobs/events SSE stream)
	// adapts it. No job kinds are registered yet — they arrive as the real
	// long-ops are migrated in a later slice; both Close() on shutdown.
	s.services.RealTime.EventBus = events.New(logging.GetLogger())
	jobsCfg := jobs.Config{Retention: jobsRetention}
	if db != nil {
		// Durable backing (Phase 5c): the runner write-throughs lifecycle
		// transitions so a job survives a restart. Without a database the runner
		// stays in-memory only (the fail-cleanly v1).
		jobsCfg.Store = newDBJobStore(db)
	}
	s.services.RealTime.Jobs = jobs.New(
		s.services.RealTime.EventBus, logging.GetLogger(), jobsCfg,
	)
	if db != nil {
		// Durable Idempotency-Key dedup (Phase 5c-4): survives restart, so a
		// client retry across a restart still replays rather than duplicating.
		s.services.RealTime.JobIdempotency = newDBJobIdempotency(db, logging.GetLogger())
	} else {
		s.services.RealTime.JobIdempotency = newJobIdempotencyCache(jobIdempotencyCapacity)
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
			newDBOutboxStore(db), s.eventBus(), logging.GetLogger(),
		)
	}

	// Wire up database persistence for logs if database is available
	if db != nil {
		s.logBroadcaster().SetDBWriter(&dbLogWriterAdapter{db: db})
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
		s.deviceDiscovery().SetDBWriter(&dbDeviceWriterAdapter{db: db})
		logging.GetLogger().Info("Device discovery initialized with database persistence")
	}
}

// initDiscovery initializes the shared discovery profiler, port scanner, and
// the discovery service. (The legacy pipeline orchestrator was retired in
// Phase 7 — discovery now runs through the engine + jobs spine.)
func (s *Server) initDiscovery(cfg *config.Config) {
	// Create SHARED DeviceProfiler - used by Service and Engine
	// This ensures port scan results and SNMP data are consistent across the system
	sharedProfiler := discovery.NewDeviceProfiler(discovery.DefaultProfilerConfig(), &cfg.SNMP)
	s.services.Discovery.Profiler = sharedProfiler

	// Create PortScanner for Engine
	portScanner, err := discovery.NewPortScanner(portScannerTimeout)
	if err != nil {
		logging.GetLogger().Warn("Failed to create port scanner", "error", err)
	} else {
		s.services.Discovery.PortScanner = portScanner
	}

	// Initialize discovery service with the shared profiler. WithCapture injects
	// the build-tagged capture adapter so the Service's internal device discovery
	// uses real libpcap capture in production (CGO-free no-op under CGO_ENABLED=0).
	s.services.Discovery.Service = discovery.NewService(
		cfg, cfg.Interface.Default, sharedProfiler, discovery.WithCapture(defaultCaptureOpener()),
	)
	logging.GetLogger().Info("Discovery service initialized with shared profiler")
}

// initVulnerabilityScanner initializes the vulnerability scanner if enabled.
func (s *Server) initVulnerabilityScanner(cfg *config.Config) {
	if !cfg.Security.VulnerabilityScanning.Enabled {
		return
	}

	scannerCfg := &discovery.VulnerabilityScannerConfig{
		Enabled:           cfg.Security.VulnerabilityScanning.Enabled,
		CVEDatabase:       cfg.Security.VulnerabilityScanning.CVEDatabase,
		NVDAPIKey:         cfg.Security.VulnerabilityScanning.NVDAPIKey,
		UpdateInterval:    cfg.Security.VulnerabilityScanning.UpdateInterval,
		SeverityThreshold: cfg.Security.VulnerabilityScanning.SeverityThreshold,
		MaxConcurrent:     cfg.Security.VulnerabilityScanning.MaxConcurrent,
	}

	vulnScanner, err := discovery.NewVulnerabilityScanner(scannerCfg)
	if err != nil {
		logging.GetLogger().Warn("Failed to initialize vulnerability scanner", "error", err)
		return
	}
	s.services.Discovery.Vulnerability = vulnScanner
	logging.GetLogger().Info("Vulnerability scanner initialized",
		"cve_database", scannerCfg.CVEDatabase, "threshold", scannerCfg.SeverityThreshold)

	// Initialize problem detector for network issue detection
	s.services.Discovery.ProblemDetector = discovery.NewProblemDetector()
	logging.GetLogger().Info("Problem detector initialized")

	// Initialize Bluetooth scanner
	btConfig := discovery.DefaultBluetoothScanConfig()
	var ouiDB *discovery.OUIDatabase
	if s.services.Discovery.Device != nil {
		ouiDB = s.services.Discovery.Device.GetOUIDatabase()
	}
	s.services.Discovery.BluetoothScanner = discovery.NewBluetoothScanner("", btConfig, ouiDB)
	logging.GetLogger().Info("Bluetooth scanner initialized")

	// Initialize WiFi bridge connecting canopy/wifi to discovery
	if s.services.Wireless.Scanner != nil {
		wifiBridgeConfig := discovery.DefaultWiFiBridgeConfig()
		s.services.Discovery.WiFiBridge = discovery.NewWiFiBridge(
			s.services.Wireless.Scanner,
			s.services.Wireless.WiFi,
			ouiDB,
			wifiBridgeConfig,
		)
		logging.GetLogger().Info("WiFi bridge initialized")
	}

	// Initialize Discovery Engine (primary unified discovery system)
	engineConfig := discovery.DefaultEngineConfig()
	s.services.Discovery.Engine = discovery.NewEngine(engineConfig)

	// Wire in all collectors
	if s.services.Discovery.Device != nil {
		s.services.Discovery.Engine.SetWiredCollector(s.services.Discovery.Device)
	}
	if s.services.Discovery.WiFiBridge != nil {
		s.services.Discovery.Engine.SetWiFiCollector(s.services.Discovery.WiFiBridge)
	}
	if s.services.Discovery.BluetoothScanner != nil {
		s.services.Discovery.Engine.SetBluetoothCollector(s.services.Discovery.BluetoothScanner)
	}
	if s.services.Discovery.Profiler != nil {
		s.services.Discovery.Engine.SetProfiler(s.services.Discovery.Profiler)
	}
	if s.services.Discovery.PortScanner != nil {
		s.services.Discovery.Engine.SetPortScanner(s.services.Discovery.PortScanner)
	}
	if s.services.Discovery.Vulnerability != nil {
		s.services.Discovery.Engine.SetVulnScanner(s.services.Discovery.Vulnerability)
	}

	// Start the engine
	if startErr := s.services.Discovery.Engine.Start(context.Background()); startErr != nil {
		logging.GetLogger().Error("Failed to start discovery engine", "error", startErr)
	} else {
		logging.GetLogger().Info("Discovery engine started",
			"capabilities", s.services.Discovery.Engine.GetCapabilities(),
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

	// Check for wildcard origin in production mode (fixes #715)
	// Production mode is inferred from HTTPS being enabled
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

	if cfg.Server.HTTPS {
		logging.GetLogger().Warn(
			"SECURITY WARNING: Wildcard origin (*) allows all origins in production mode with HTTPS enabled",
			"recommendation",
			"Configure explicit allowed origins in Security.AllowedOrigins for production deployments",
		)
		return
	}

	logging.GetLogger().Info("Wildcard origin (*) configured - allows all origins (development mode)",
		"warning", "Not recommended for production use")
}
