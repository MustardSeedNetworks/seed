package api

// server_init.go contains the per-subsystem initialisation helpers that
// NewServer composes: DNS/discovery/survey, additional subnets, database +
// migration, MIB DB, SSE + log broadcaster, discovery pipeline, vulnerability
// scanner, and CORS origin policy.

import (
	"context"
	"slices"

	"github.com/krisarmstrong/seed/internal/auth"
	"github.com/krisarmstrong/seed/internal/canopy/survey"
	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/mibdb"
	"github.com/krisarmstrong/seed/internal/services/discovery"
	"github.com/krisarmstrong/seed/internal/services/dns"
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
	s.services.Canopy.Survey = survey.NewManager(
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

	// Start data retention cleanup in background (fixes #848)
	if cfg.Database.RetentionDays > 0 {
		s.services.Database.RetentionStopCh = make(chan struct{})
		go s.startDataRetention(cfg.Database.RetentionDays)
	}
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

// initDiscoveryPipeline initializes the discovery service and pipeline.
func (s *Server) initDiscoveryPipeline(cfg *config.Config) {
	// Create SHARED DeviceProfiler - used by Service, Pipeline, and Engine
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

	// Initialize discovery service with the shared profiler
	s.services.Discovery.Service = discovery.NewService(cfg, cfg.Interface.Default, sharedProfiler)
	logging.GetLogger().Info("Discovery service initialized with shared profiler")

	// Initialize discovery pipeline with the SAME shared profiler
	pipelineCfg := discovery.PipelineConfigFromAdapter(&cfg.Pipeline)
	s.services.Discovery.Pipeline = discovery.NewPipeline(
		&pipelineCfg,
		s.deviceDiscovery(),
		sharedProfiler, // Use the same profiler as Service
		&pipelineBroadcastAdapter{hub: s.sseHub()},
	)

	// Link Service and Pipeline for coordination
	s.discoveryService().SetPipeline(s.pipeline())

	// Set up pipeline completion callback to sync results back to service
	s.discoveryService().SetOnPipelineComplete(func(devices []*discovery.DiscoveredDevice) {
		logging.GetLogger().Info(
			"Pipeline completed, syncing results to discovery service",
			"device_count",
			len(devices),
		)
	})

	logging.GetLogger().Info("Discovery pipeline initialized",
		"phases_enabled", s.pipeline().GetEnabledPhaseNames(),
		"port_scan_intensity", cfg.Pipeline.PortScan.Intensity,
		"shared_profiler", true)
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
	if s.services.Canopy.Scanner != nil {
		wifiBridgeConfig := discovery.DefaultWiFiBridgeConfig()
		s.services.Discovery.WiFiBridge = discovery.NewWiFiBridge(
			s.services.Canopy.Scanner,
			s.services.Canopy.WiFi,
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
