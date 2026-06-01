package api

// server_routes.go contains the HTTP route table: setupRoutes plus the
// per-module setup helpers (core auth/settings, SAP telemetry, Shell, Roots,
// Canopy, Harvest) and the SSE + static file fallback.

import (
	"net/http"

	"github.com/krisarmstrong/seed/internal/database"
	"github.com/krisarmstrong/seed/internal/logging"
)

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/__version", s.handleBuildVersion)
	s.setupCoreRoutes()
	s.setupAPITokenRoutes()
	s.registerUpdateRoutes()
	s.setupSAPRoutes()
	s.setupShellRoutes()
	s.setupRootsRoutes()
	s.setupCanopyRoutes()
	s.setupHarvestRoutes()
	s.setupTopologyRoutes()
	s.setupSSEAndStatic()
}

// setupTopologyRoutes registers the Stage A5.1 read-only topology
// endpoints. All are GET-only and run through the same JWT/PAT auth
// middleware as the rest of /api/v1.
func (s *Server) setupTopologyRoutes() {
	op := database.RoleOperator
	s.registerAll([]route{
		// /nodes must register BEFORE /nodes/ so the router doesn't treat the
		// list endpoint as a /nodes/{id} request.
		{path: APIVersionPrefix + "/topology/nodes", handler: s.handleTopologyNodes},
		{path: APIVersionPrefix + "/topology/nodes/", handler: s.handleTopologyNodeByID},
		{path: APIVersionPrefix + "/topology/links", handler: s.handleTopologyLinks},
		{path: APIVersionPrefix + "/topology/arp", handler: s.handleTopologyARP},
		// A5.2 alerts: GET read-only; the action endpoint is operator-gated.
		{path: APIVersionPrefix + "/alerts", handler: s.handleAlerts},
		{path: APIVersionPrefix + "/alerts/", handler: s.handleAlertAction, minRole: op},
		// A5.3 polling targets CRUD: both writeGated (collection accepts POST).
		{path: APIVersionPrefix + "/polling-targets", handler: s.handlePollingTargets, minRole: op},
		{path: APIVersionPrefix + "/polling-targets/", handler: s.handlePollingTargetByID, minRole: op},
		// A5.8 read-only engine registry surface.
		{path: APIVersionPrefix + "/engines", handler: s.handleEngines},
		// A5.10 operator-defined alert rules: both writeGated.
		{path: APIVersionPrefix + "/alert-rules", handler: s.handleAlertRules, minRole: op},
		{path: APIVersionPrefix + "/alert-rules/", handler: s.handleAlertRuleByID, minRole: op},
	})
}

// setupAPITokenRoutes registers the Phase D-2 personal-access-token
// endpoints and the read-only license status endpoint the UI uses to
// know whether the mint button should be enabled.
func (s *Server) setupAPITokenRoutes() {
	op := database.RoleOperator
	s.registerAll([]route{
		{path: APIVersionPrefix + "/tokens", handler: s.handleAPITokens, minRole: op},
		{path: APIVersionPrefix + "/tokens/", handler: s.handleAPITokenByID, minRole: op},
		{path: APIVersionPrefix + "/license", handler: s.handleLicenseStatus},
		// Users CRUD (#1191): /users/me registers before /users/ for path routing.
		// POST /users is admin-only AND Pro-gated, enforced inside the handler so the
		// response carries the right 403/402 FeatureGateResponse.
		{path: APIVersionPrefix + "/users/me", handler: s.handleCurrentUser},
		{path: APIVersionPrefix + "/users", handler: s.handleUsers},
		{path: APIVersionPrefix + "/users/", handler: s.handleUserByName},
	})
}

// setupCoreRoutes registers auth, settings, config, and setup routes.
func (s *Server) setupCoreRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/auth/login", s.handleLogin)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/logout", s.handleLogout)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/refresh", s.handleRefreshToken)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/csrf", s.handleCSRFToken)
	// Wave 3 (#85): MFA + WebAuthn endpoints.
	s.mux.HandleFunc(APIVersionPrefix+"/auth/login/totp", s.handleLoginTOTP)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/totp/setup", s.handleTOTPSetup)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/totp/verify", s.handleTOTPVerify)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/totp/disable", s.handleTOTPDisable)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/mfa/status", s.handleMFAStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/webauthn/register/begin", s.handleWebAuthnRegisterBegin)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/webauthn/register/finish", s.handleWebAuthnRegisterFinish)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/webauthn/login/begin", s.handleWebAuthnLoginBegin)
	s.mux.HandleFunc(APIVersionPrefix+"/auth/webauthn/login/finish", s.handleWebAuthnLoginFinish)
	s.mux.HandleFunc(APIVersionPrefix+"/status", s.handleStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/settings", s.writeGated(s.handleSettings))
	s.mux.HandleFunc(APIVersionPrefix+"/settings/defaults", s.writeGated(s.handleSettingsDefaults))
	s.mux.HandleFunc(APIVersionPrefix+"/settings/link", s.writeGated(s.handleLinkSettings))
	s.mux.HandleFunc(APIVersionPrefix+"/settings/cable", s.writeGated(s.handleCableTestSettings))
	s.mux.HandleFunc(APIVersionPrefix+"/interfaces", s.handleInterfaces)
	s.mux.HandleFunc(APIVersionPrefix+"/interface", s.handleInterface)
	s.mux.HandleFunc(APIVersionPrefix+"/network/mtu", s.writeGated(s.handleSetMTU))
	s.mux.HandleFunc(APIVersionPrefix+"/config/backups", s.handleConfigBackups)
	s.mux.HandleFunc(APIVersionPrefix+"/config/backup", s.writeGated(s.handleConfigBackupCreate))
	s.mux.HandleFunc(APIVersionPrefix+"/config/backup/delete", s.writeGated(s.handleConfigBackupDelete))
	s.mux.HandleFunc(APIVersionPrefix+"/config/restore", s.writeGated(s.handleConfigRestore))
	s.mux.HandleFunc(APIVersionPrefix+"/config/version", s.handleConfigVersion)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles", s.writeGated(s.handleProfiles))
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/active", s.writeGated(s.handleActiveProfile))
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/import", s.writeGated(s.handleImportProfiles))
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/export", s.handleExportProfiles)
	s.mux.HandleFunc(APIVersionPrefix+"/profiles/", s.writeGated(s.handleProfiles))
	s.mux.HandleFunc(APIVersionPrefix+"/setup/status", s.handleSetupStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/setup/complete", s.handleSetupComplete)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/status", s.handleRecoveryStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/complete", s.handleRecoveryComplete)
	s.mux.HandleFunc(APIVersionPrefix+"/recovery/instructions", s.handleRecoveryInstructions)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/providers", s.handleSSOProviders)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/login", s.handleSSOLogin)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/callback", s.handleSSOCallback)
	s.mux.HandleFunc(APIVersionPrefix+"/sso/settings", s.writeGated(s.handleSSOSettings))
	// SSO settings PUT is Pro-gated via requireFeature("sso") — see seed#1198.
	// GET stays open so operators can inspect existing config on any tier
	// (a Pro→Free downgrade must not lock them out of reading current
	// state). Provider login + callback continue to work for already-
	// configured providers regardless of tier — only WRITES are blocked.
	s.mux.HandleFunc(APIVersionPrefix+"/sso/update", s.writeGated(s.requireFeature("sso", s.handleSSOUpdate)))
	s.mux.HandleFunc(APIVersionPrefix+"/health", s.handleHealth)
}

// setupSAPRoutes registers SAP module routes (live telemetry).
func (s *Server) setupSAPRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/sap/link", s.handleLink)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/cable", s.handleCable)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns", s.handleDNS)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns/security", s.handleDNSSecurity)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dns/security/settings", s.writeGated(s.handleDNSSecuritySettings))
	s.mux.HandleFunc(APIVersionPrefix+"/sap/gateway", s.handleGateway)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue", s.handleRogueDHCP)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue/servers", s.handleRogueDHCPServers)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/dhcp/rogue/config", s.writeGated(s.handleRogueDHCPConfig))
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
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/settings", s.writeGated(s.handleHealthChecksSettings))
	s.mux.Handle(
		APIVersionPrefix+"/sap/health-checks/run",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.handleHealthChecks)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/results", s.handleHealthCheckResults)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/history", s.handleHealthCheckHistory)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/scores", s.handleHealthCheckScores)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/sla", s.handleHealthCheckSLA)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/health-checks/alerts", s.handleHealthCheckAlerts)
	// Anomaly detection is a Pro feature (LICENSE_STRATEGY §2). Base
	// health-check results / history / alerts remain accessible to
	// all tiers — only the trend/anomaly analysis is paid.
	s.mux.HandleFunc(
		APIVersionPrefix+"/sap/health-checks/anomalies",
		s.requireFeature("anomaly_detection", s.handleHealthCheckAnomalies),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/snmp/settings", s.writeGated(s.handleSNMPSettings))
	s.mux.HandleFunc(APIVersionPrefix+"/sap/system/health", s.handleSystemHealth)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/ipconfig", s.handleIPConfig)
	s.mux.HandleFunc(APIVersionPrefix+"/sap/ipconfig/settings", s.writeGated(s.handleIPSettings))
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
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/settings", s.writeGated(s.handleDevicesSettings))
	s.mux.HandleFunc(APIVersionPrefix+"/shell/devices/subnets", s.handleDevicesSubnets)
	// Vulnerability scan + guest-audit run are compliance_advanced
	// (Pro) per LICENSE_STRATEGY §2. Read-only results / status /
	// settings endpoints stay open so existing scan output remains
	// visible to lower tiers (and so admins can downgrade then still
	// read prior reports).
	s.mux.Handle(
		APIVersionPrefix+"/shell/vulnerabilities/scan",
		s.endpointRateLimiter().RateLimitMiddleware(
			s.requireFeature("compliance_advanced", s.handleVulnerabilityScan),
		),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/status", s.handleVulnerabilityStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/results", s.handleVulnerabilityResults)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/device", s.handleDeviceVulnerabilities)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/settings", s.writeGated(s.handleVulnerabilitySettings))
	s.mux.HandleFunc(APIVersionPrefix+"/shell/vulnerabilities/validate-api-key", s.handleNVDAPIKeyValidate)
	// Guest-network isolation audit (#397).
	s.mux.HandleFunc(APIVersionPrefix+"/shell/guest-audit/settings", s.writeGated(s.handleGuestAuditSettings))
	s.mux.Handle(
		APIVersionPrefix+"/shell/guest-audit/run",
		s.endpointRateLimiter().RateLimitMiddleware(
			s.requireFeature("compliance_advanced", s.handleGuestAuditRun),
		),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/status", s.handlePipelineStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/start", s.handlePipelineStart)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/cancel", s.handlePipelineCancel)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/config", s.writeGated(s.handlePipelineConfigRoute))
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/port-intensity", s.handlePipelinePortIntensityInfo)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/pipeline/timing-profiles", s.handlePipelineTimingProfiles)

	// Network problem detection routes
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems", s.handleNetworkProblems)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems/scan", s.handleProblemScan)
	s.mux.HandleFunc(APIVersionPrefix+"/shell/problems/thresholds", s.writeGated(s.handleProblemThresholds))

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
// Both endpoints are gated on the `path_analysis` feature (Pro tier);
// Free / Starter receive 402 with an upgrade hint. The rate-limit
// middleware still wraps traceroute so abuse remains capped even for
// trial users.
func (s *Server) setupRootsRoutes() {
	s.registerAll([]route{
		{
			path:        APIVersionPrefix + "/roots/traceroute",
			handler:     s.handleTraceroute,
			feature:     "path_analysis",
			rateLimited: true,
		},
		{path: APIVersionPrefix + "/roots/path", handler: s.handlePath, feature: "path_analysis"},
	})
}

// setupCanopyRoutes registers Canopy module routes (Wi-Fi visibility &
// troubleshooting). First module on the declarative capability registry
// (ADR-0002): policy is data, composed by register() in one canonical order.
// Behavior is identical to the prior hand-wrapped form.
func (s *Server) setupCanopyRoutes() {
	op := database.RoleOperator
	s.registerAll([]route{
		{path: APIVersionPrefix + "/canopy/wifi", handler: s.handleWiFi},
		{path: APIVersionPrefix + "/canopy/wifi/scan", handler: s.handleWiFiScan},
		{path: APIVersionPrefix + "/canopy/wifi/status", handler: s.handleWiFiStatus},
		{path: APIVersionPrefix + "/canopy/wifi/channel-graph", handler: s.handleWiFiChannelGraph},
		{path: APIVersionPrefix + "/canopy/wifi/settings", handler: s.handleWiFiSettings, minRole: op},
		{path: APIVersionPrefix + "/canopy/wifi/connect", handler: s.handleWiFiConnect, minRole: op},
		{path: APIVersionPrefix + "/canopy/wifi/disconnect", handler: s.handleWiFiDisconnect, minRole: op},
		{path: APIVersionPrefix + "/canopy/wifi/saved", handler: s.handleWiFiSavedNetworks},
		{path: APIVersionPrefix + "/canopy/wifi/forget", handler: s.handleWiFiForgetNetwork, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/create", handler: s.createSurvey, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/list", handler: s.listSurveys},
		{path: APIVersionPrefix + "/canopy/survey", handler: s.getSurvey},
		{path: APIVersionPrefix + "/canopy/survey/delete", handler: s.deleteSurvey, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/start", handler: s.startSurvey, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/pause", handler: s.pauseSurvey, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/complete", handler: s.completeSurvey, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/sample", handler: s.addSurveySample, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/floorplan", handler: s.updateSurveyFloorPlan, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/settings", handler: s.updateSurveySettings, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/imported-data", handler: s.updateSurveyImportedData, minRole: op},
		// AirMapper baseline-diff (Pro, LICENSE_STRATEGY §2): imports an AirMapper
		// survey JSON and diffs it against the floor-plan baseline; rate-limited.
		{
			path:        APIVersionPrefix + "/canopy/survey/import/airmapper",
			handler:     s.importAirMapper,
			minRole:     op,
			feature:     "airmapper_baseline_diff",
			rateLimited: true,
		},
		{path: APIVersionPrefix + "/canopy/survey/heatmap", handler: s.getSurveyHeatmap, rateLimited: true},
		{path: APIVersionPrefix + "/canopy/survey/dead-zones", handler: s.getSurveyDeadZones},
		{path: APIVersionPrefix + "/canopy/survey/floors", handler: s.handleSurveyFloors, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/floor", handler: s.handleSurveyFloor, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/floor/floorplan", handler: s.updateFloorFloorPlan, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/floor/sample", handler: s.addFloorSample, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/active-floor", handler: s.setActiveFloor, minRole: op},
		{path: APIVersionPrefix + "/canopy/survey/report", handler: s.generateSurveyReport, rateLimited: true},
	})
}

// setupHarvestRoutes registers Harvest module routes (reporting).
// /harvest/export is gated behind the `export_csv_json` feature
// (Starter or higher) per LICENSE_STRATEGY §2. Log endpoints stay
// ungated because operational visibility is a basic capability for
// every tier; only data extraction (the customer-facing reporting
// surface) is paid.
func (s *Server) setupHarvestRoutes() {
	s.registerAll([]route{
		{path: APIVersionPrefix + "/harvest/export", handler: s.handleExport, feature: "export_csv_json"},
		{path: APIVersionPrefix + "/harvest/logs", handler: s.handleLogs},
		{path: APIVersionPrefix + "/harvest/logs/client", handler: s.handleClientLogs},
		{path: APIVersionPrefix + "/harvest/logs/query", handler: s.handleLogsQuery},
		{path: APIVersionPrefix + "/harvest/logs/stats", handler: s.handleLogsStats},
		{path: APIVersionPrefix + "/harvest/logs/recent", handler: s.handleLogsRecent},
	})
}

// setupSSEAndStatic registers SSE and static file handlers.
func (s *Server) setupSSEAndStatic() {
	// SSE endpoint for real-time updates. Gated on `live_telemetry`
	// (Pro tier) per FEATURE_TIER_MATRIX — the live stream of card
	// data (RSSI, link state, gateway latency, etc.) is the Pro-tier
	// real-time surface. Free / Starter get card data via the
	// per-endpoint REST handlers without the WebSocket-like stream.
	// `/discovery/engine/events` (the discovery-lifecycle SSE) is
	// intentionally NOT gated here — discovery is a Free-tier surface.
	s.mux.HandleFunc(APIVersionPrefix+"/events", s.requireFeature("live_telemetry", s.handleSSE))
	frontendFS, err := getUIFS()
	if err != nil {
		logging.GetLogger().
			Warn("Failed to get embedded frontend FS, falling back to disk", "error", err)
		s.mux.Handle("/", http.FileServer(http.Dir("internal/api/ui")))
	} else {
		logging.GetLogger().Info("Serving frontend from embedded filesystem", "embedded", isUIEmbedded())
		s.mux.Handle("/", spaHandler(http.FS(frontendFS)))
	}
}
