package api

// server_routes.go contains the HTTP route table: setupRoutes plus the
// per-module setup helpers (core auth/settings, SAP telemetry, Shell, Roots,
// Canopy, Harvest) and the SSE + static file fallback.

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/ui"
)

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/__version", s.handleBuildVersion)
	s.setupCoreRoutes()
	s.registerUpdateRoutes()
	s.setupSAPRoutes()
	s.setupShellRoutes()
	s.setupRootsRoutes()
	s.setupCanopyRoutes()
	s.setupHarvestRoutes()
	s.setupSSEAndStatic()
}

// setupCoreRoutes registers auth, settings, config, and setup routes.
func (s *Server) setupCoreRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/auth/login", s.handleLogin)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/logout", s.handleLogout)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/refresh", s.handleRefreshToken)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/csrf", s.handleCSRFToken)
	s.mux.HandleFunc(APIVersionPrefix+"/status", s.handleStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/settings", s.handleSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/settings/defaults", s.handleSettingsDefaults)
	s.mux.HandleFunc(APIVersionPrefix+"/settings/link", s.handleLinkSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/settings/cable", s.handleCableTestSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/interfaces", s.handleInterfaces)
	s.mux.HandleFunc(APIVersionPrefix+"/interface", s.handleInterface)
	s.mux.HandleFunc(APIVersionPrefix+"/network/mtu", s.handleSetMTU)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backups", s.handleConfigBackups)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backup", s.handleConfigBackupCreate)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backup/delete", s.handleConfigBackupDelete)
	s.mux.HandleFunc(APIVersionPrefix+"/config/restore", s.handleConfigRestore)
	s.mux.HandleFunc(APIVersionPrefix+"/config/version", s.handleConfigVersion)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles", s.handleProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/active", s.handleActiveProfile)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/import", s.handleImportProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/export", s.handleExportProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/", s.handleProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/setup/complete", s.handleSetupComplete)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/status", s.handleRecoveryStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/complete", s.handleRecoveryComplete)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/instructions", s.handleRecoveryInstructions)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/providers", s.handleSSOProviders)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/login", s.handleSSOLogin)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/callback", s.handleSSOCallback)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/settings", s.handleSSOSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/update", s.handleSSOUpdate)
	s.mux.HandleFunc(APIVersionPrefix+"/health", s.handleHealth)
}

// setupSAPRoutes registers SAP module routes (live telemetry).
func (s *Server) setupSAPRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/sap/link", s.handleLink)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/cable", s.handleCable)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns", s.handleDNS)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns/security", s.handleDNSSecurity)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns/security/settings", s.handleDNSSecuritySettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/gateway", s.handleGateway)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue", s.handleRogueDHCP)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue/servers", s.handleRogueDHCPServers)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue/config", s.handleRogueDHCPConfig)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/vlan", s.handleVLAN)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/vlan/traffic", s.handleVLANTraffic)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/vlan/interface", s.handleVLANInterface)
	s.mux.Handle(
		APIVersionPrefix+"/sap/speedtest",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleSpeedtest)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/speedtest/status", s.handleSpeedtestStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/info", s.handleIperfInfo)
	s.mux.Handle(
		APIVersionPrefix+"/sap/iperf/client",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleIperfClient)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/client/status", s.handleIperfClientStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/server", s.handleIperfServer)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/server/status", s.handleIperfServerStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/iperf/suggestions", s.handleIperfSuggestions)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/settings", s.handleHealthChecksSettings)
	s.mux.Handle(
		APIVersionPrefix+"/sap/health-checks/run",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleHealthChecks)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/results", s.handleHealthCheckResults)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/history", s.handleHealthCheckHistory)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/scores", s.handleHealthCheckScores)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/sla", s.handleHealthCheckSLA)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/alerts", s.handleHealthCheckAlerts)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/anomalies", s.handleHealthCheckAnomalies)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/snmp/settings", s.handleSNMPSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/system/health", s.handleSystemHealth)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/ipconfig", s.handleIPConfig)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/ipconfig/settings", s.handleIPSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/publicip", s.handlePublicIP)
}

// setupShellRoutes registers Shell module routes (security posture).
func (s *Server) setupShellRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery", s.handleDiscovery)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/probe", s.handleTCPProbe)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/portscan", s.handlePortScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/options", s.handleDiscoveryOptions)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/service/status", s.handleDiscoveryServiceStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/discovery/fingerprint", s.handleAdvancedFingerprint)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices", s.handleDevices)
	s.mux.Handle(
		APIVersionPrefix+"/shell/devices/scan",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleDevicesScan)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/status", s.handleDevicesStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/settings", s.handleDevicesSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/subnets", s.handleDevicesSubnets)
	s.mux.Handle(
		APIVersionPrefix+"/shell/vulnerabilities/scan",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleVulnerabilityScan)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/status", s.handleVulnerabilityStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/results", s.handleVulnerabilityResults)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/device", s.handleDeviceVulnerabilities)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/settings", s.handleVulnerabilitySettings)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/validate-api-key", s.handleNVDAPIKeyValidate)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/status", s.handlePipelineStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/start", s.handlePipelineStart)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/cancel", s.handlePipelineCancel)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/config", s.handlePipelineConfigRoute)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/port-intensity", s.handlePipelinePortIntensityInfo)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/timing-profiles", s.handlePipelineTimingProfiles)

	// Network problem detection routes
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems", s.handleNetworkProblems)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems/scan", s.handleProblemScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems/thresholds", s.handleProblemThresholds)

	// Bluetooth discovery routes
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/scan", s.handleBluetoothScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/devices", s.handleBluetoothDevices)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/stats", s.handleBluetoothStats)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/bluetooth/status", s.handleBluetoothStatus)

	// Enhanced WiFi discovery routes (unified discovery)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/scan", s.handleWiFiDiscoveryScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/networks", s.handleWiFiDiscoveryNetworks)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/aps", s.handleWiFiDiscoveryAPs)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/wifi/discovery/stats", s.handleWiFiDiscoveryStats)

	// Discovery Engine routes (primary unified discovery system)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine", s.handleEngineDiscovery)
	s.mux.Handle(
		APIVersionPrefix+"/discovery/engine/scan",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleEngineScan)),
	)
	s.mux.Handle(
		APIVersionPrefix+"/discovery/engine/quick",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleEngineQuickScan)),
	)
	s.mux.Handle(
		APIVersionPrefix+"/discovery/engine/full",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleEngineFullScan)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/stats", s.handleEngineStats)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/capabilities", s.handleEngineCapabilities)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/device/", s.handleEngineDevice)
	s.mux.HandleFunc(APIVersionPrefix+"/discovery/engine/events", s.handleEngineEvents)
}

// setupRootsRoutes registers Roots module routes (path analysis).
func (s *Server) setupRootsRoutes() {
	s.mux.Handle(
		APIVersionPrefix+"/roots/traceroute",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleTraceroute)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/roots/path", s.handlePath)
}

// setupCanopyRoutes registers Canopy module routes (Wi-Fi planning).
func (s *Server) setupCanopyRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi", s.handleWiFi)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/scan", s.handleWiFiScan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/status", s.handleWiFiStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/channel-graph", s.handleWiFiChannelGraph)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/settings", s.handleWiFiSettings)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/connect", s.handleWiFiConnect)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/disconnect", s.handleWiFiDisconnect)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/saved", s.handleWiFiSavedNetworks)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/forget", s.handleWiFiForgetNetwork)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/create", s.createSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/list", s.listSurveys)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey", s.getSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/delete", s.deleteSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/start", s.startSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/pause", s.pauseSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/complete", s.completeSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/sample", s.addSurveySample)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floorplan", s.updateSurveyFloorPlan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/settings", s.updateSurveySettings)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/import/airmapper",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.importAirMapper)),
	)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/heatmap",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.getSurveyHeatmap)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/dead-zones", s.getSurveyDeadZones)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floors", s.handleSurveyFloors)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor", s.handleSurveyFloor)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor/floorplan", s.updateFloorFloorPlan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor/sample", s.addFloorSample)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/active-floor", s.setActiveFloor)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/report",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.generateSurveyReport)),
	)
}

// setupHarvestRoutes registers Harvest module routes (reporting).
func (s *Server) setupHarvestRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/export", s.handleExport)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs", s.handleLogs)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/client", s.handleClientLogs)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/query", s.handleLogsQuery)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/stats", s.handleLogsStats)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/recent", s.handleLogsRecent)
}

// setupSSEAndStatic registers SSE and static file handlers.
func (s *Server) setupSSEAndStatic() {
	// SSE endpoint for real-time updates
	s.mux.HandleFunc(APIVersionPrefix+"/events", s.handleSSE)
	frontendFS, err := ui.GetFS()
	if err != nil {
		logging.GetLogger().
			Warn("Failed to get embedded frontend FS, falling back to disk", "error", err)
		s.mux.Handle("/", http.FileServer(http.Dir("ui/dist")))
	} else {
		logging.GetLogger().Info("Serving frontend from embedded filesystem", "embedded", ui.IsEmbedded())
		s.mux.Handle("/", spaHandler(http.FS(frontendFS)))
	}
}
