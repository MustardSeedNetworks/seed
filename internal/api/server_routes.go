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
	op := database.RoleOperator
	s.registerAll([]route{
		{path: APIVersionPrefix + "/auth/login", handler: s.handleLogin},
		{path: APIVersionPrefix + "/auth/logout", handler: s.handleLogout},
		{path: APIVersionPrefix + "/auth/refresh", handler: s.handleRefreshToken},
		{path: APIVersionPrefix + "/auth/csrf", handler: s.handleCSRFToken},
		// Wave 3 (#85): MFA + WebAuthn endpoints.
		{path: APIVersionPrefix + "/auth/login/totp", handler: s.handleLoginTOTP},
		{path: APIVersionPrefix + "/auth/totp/setup", handler: s.handleTOTPSetup},
		{path: APIVersionPrefix + "/auth/totp/verify", handler: s.handleTOTPVerify},
		{path: APIVersionPrefix + "/auth/totp/disable", handler: s.handleTOTPDisable},
		{path: APIVersionPrefix + "/auth/mfa/status", handler: s.handleMFAStatus},
		{path: APIVersionPrefix + "/auth/webauthn/register/begin", handler: s.handleWebAuthnRegisterBegin},
		{path: APIVersionPrefix + "/auth/webauthn/register/finish", handler: s.handleWebAuthnRegisterFinish},
		{path: APIVersionPrefix + "/auth/webauthn/login/begin", handler: s.handleWebAuthnLoginBegin},
		{path: APIVersionPrefix + "/auth/webauthn/login/finish", handler: s.handleWebAuthnLoginFinish},
		{path: APIVersionPrefix + "/status", handler: s.handleStatus},
		{path: APIVersionPrefix + "/settings", handler: s.handleSettings, minRole: op},
		{path: APIVersionPrefix + "/settings/defaults", handler: s.handleSettingsDefaults, minRole: op},
		{path: APIVersionPrefix + "/settings/link", handler: s.handleLinkSettings, minRole: op},
		{path: APIVersionPrefix + "/settings/cable", handler: s.handleCableTestSettings, minRole: op},
		{path: APIVersionPrefix + "/interfaces", handler: s.handleInterfaces},
		{path: APIVersionPrefix + "/interface", handler: s.handleInterface},
		{path: APIVersionPrefix + "/network/mtu", handler: s.handleSetMTU, minRole: op},
		{path: APIVersionPrefix + "/config/backups", handler: s.handleConfigBackups},
		{path: APIVersionPrefix + "/config/backup", handler: s.handleConfigBackupCreate, minRole: op},
		{path: APIVersionPrefix + "/config/backup/delete", handler: s.handleConfigBackupDelete, minRole: op},
		{path: APIVersionPrefix + "/config/restore", handler: s.handleConfigRestore, minRole: op},
		{path: APIVersionPrefix + "/config/version", handler: s.handleConfigVersion},
		{path: APIVersionPrefix + "/profiles", handler: s.handleProfiles, minRole: op},
		{path: APIVersionPrefix + "/profiles/active", handler: s.handleActiveProfile, minRole: op},
		{path: APIVersionPrefix + "/profiles/import", handler: s.handleImportProfiles, minRole: op},
		{path: APIVersionPrefix + "/profiles/export", handler: s.handleExportProfiles},
		{path: APIVersionPrefix + "/profiles/", handler: s.handleProfiles, minRole: op},
		{path: APIVersionPrefix + "/setup/status", handler: s.handleSetupStatus},
		{path: APIVersionPrefix + "/setup/complete", handler: s.handleSetupComplete},
		{path: APIVersionPrefix + "/recovery/status", handler: s.handleRecoveryStatus},
		{path: APIVersionPrefix + "/recovery/complete", handler: s.handleRecoveryComplete},
		{path: APIVersionPrefix + "/recovery/instructions", handler: s.handleRecoveryInstructions},
		{path: APIVersionPrefix + "/sso/providers", handler: s.handleSSOProviders},
		{path: APIVersionPrefix + "/sso/login", handler: s.handleSSOLogin},
		{path: APIVersionPrefix + "/sso/callback", handler: s.handleSSOCallback},
		{path: APIVersionPrefix + "/sso/settings", handler: s.handleSSOSettings, minRole: op},
		// SSO update is Pro-gated (requireFeature "sso", #1198) AND operator-gated.
		// NORMALIZATION (ADR-0002): the registry composes canonical order
		// requireFeature(writeGated(h)) — a viewer on Free now receives 402
		// (feature) where the prior hand-wrapped writeGated(requireFeature(...))
		// returned 403 (role) first. Operators and Pro tiers are unaffected.
		{path: APIVersionPrefix + "/sso/update", handler: s.handleSSOUpdate, minRole: op, feature: "sso"},
		{path: APIVersionPrefix + "/health", handler: s.handleHealth},
	})
}

// setupSAPRoutes registers SAP module routes (live telemetry).
func (s *Server) setupSAPRoutes() {
	op := database.RoleOperator
	s.registerAll([]route{
		{path: APIVersionPrefix + "/sap/link", handler: s.handleLink},
		{path: APIVersionPrefix + "/sap/cable", handler: s.handleCable},
		{path: APIVersionPrefix + "/sap/dns", handler: s.handleDNS},
		{path: APIVersionPrefix + "/sap/dns/security", handler: s.handleDNSSecurity},
		{path: APIVersionPrefix + "/sap/dns/security/settings", handler: s.handleDNSSecuritySettings, minRole: op},
		{path: APIVersionPrefix + "/sap/gateway", handler: s.handleGateway},
		{path: APIVersionPrefix + "/sap/dhcp/rogue", handler: s.handleRogueDHCP},
		{path: APIVersionPrefix + "/sap/dhcp/rogue/servers", handler: s.handleRogueDHCPServers},
		{path: APIVersionPrefix + "/sap/dhcp/rogue/config", handler: s.handleRogueDHCPConfig, minRole: op},
		{path: APIVersionPrefix + "/sap/vlan", handler: s.handleVLAN},
		{path: APIVersionPrefix + "/sap/vlan/traffic", handler: s.handleVLANTraffic},
		{path: APIVersionPrefix + "/sap/vlan/interface", handler: s.handleVLANInterface},
		{path: APIVersionPrefix + "/sap/speedtest", handler: s.handleSpeedtest, rateLimited: true},
		{path: APIVersionPrefix + "/sap/speedtest/status", handler: s.handleSpeedtestStatus},
		{path: APIVersionPrefix + "/sap/iperf/info", handler: s.handleIperfInfo},
		{path: APIVersionPrefix + "/sap/iperf/client", handler: s.handleIperfClient, rateLimited: true},
		{path: APIVersionPrefix + "/sap/iperf/client/status", handler: s.handleIperfClientStatus},
		{path: APIVersionPrefix + "/sap/iperf/server", handler: s.handleIperfServer},
		{path: APIVersionPrefix + "/sap/iperf/server/status", handler: s.handleIperfServerStatus},
		{path: APIVersionPrefix + "/sap/iperf/suggestions", handler: s.handleIperfSuggestions},
		{path: APIVersionPrefix + "/sap/health-checks/settings", handler: s.handleHealthChecksSettings, minRole: op},
		{path: APIVersionPrefix + "/sap/health-checks/run", handler: s.handleHealthChecks, rateLimited: true},
		{path: APIVersionPrefix + "/sap/health-checks/results", handler: s.handleHealthCheckResults},
		{path: APIVersionPrefix + "/sap/health-checks/history", handler: s.handleHealthCheckHistory},
		{path: APIVersionPrefix + "/sap/health-checks/scores", handler: s.handleHealthCheckScores},
		{path: APIVersionPrefix + "/sap/health-checks/sla", handler: s.handleHealthCheckSLA},
		{path: APIVersionPrefix + "/sap/health-checks/alerts", handler: s.handleHealthCheckAlerts},
		// Anomaly detection is Pro (LICENSE_STRATEGY §2); base results/history/alerts
		// stay open to all tiers — only the trend/anomaly analysis is paid.
		{
			path:    APIVersionPrefix + "/sap/health-checks/anomalies",
			handler: s.handleHealthCheckAnomalies,
			feature: "anomaly_detection",
		},
		{path: APIVersionPrefix + "/sap/snmp/settings", handler: s.handleSNMPSettings, minRole: op},
		{path: APIVersionPrefix + "/sap/system/health", handler: s.handleSystemHealth},
		{path: APIVersionPrefix + "/sap/ipconfig", handler: s.handleIPConfig},
		{path: APIVersionPrefix + "/sap/ipconfig/settings", handler: s.handleIPSettings, minRole: op},
		{path: APIVersionPrefix + "/sap/publicip", handler: s.handlePublicIP},
	})
}

// setupShellRoutes registers Shell module routes (security posture).
func (s *Server) setupShellRoutes() {
	op := database.RoleOperator
	s.registerAll([]route{
		{path: APIVersionPrefix + "/shell/discovery", handler: s.handleDiscovery},
		{path: APIVersionPrefix + "/shell/discovery/probe", handler: s.handleTCPProbe},
		{path: APIVersionPrefix + "/shell/discovery/portscan", handler: s.handlePortScan},
		{path: APIVersionPrefix + "/shell/discovery/options", handler: s.handleDiscoveryOptions},
		{path: APIVersionPrefix + "/shell/discovery/service/status", handler: s.handleDiscoveryServiceStatus},
		{path: APIVersionPrefix + "/shell/discovery/fingerprint", handler: s.handleAdvancedFingerprint},
		{path: APIVersionPrefix + "/shell/devices", handler: s.handleDevices},
		{path: APIVersionPrefix + "/shell/devices/scan", handler: s.handleDevicesScan, rateLimited: true},
		{path: APIVersionPrefix + "/shell/devices/status", handler: s.handleDevicesStatus},
		{path: APIVersionPrefix + "/shell/devices/settings", handler: s.handleDevicesSettings, minRole: op},
		{path: APIVersionPrefix + "/shell/devices/subnets", handler: s.handleDevicesSubnets},
		// Vulnerability scan + guest-audit run are compliance_advanced (Pro,
		// LICENSE_STRATEGY §2); read-only results/status/settings stay open so prior
		// scan output remains visible to lower tiers.
		{
			path:        APIVersionPrefix + "/shell/vulnerabilities/scan",
			handler:     s.handleVulnerabilityScan,
			feature:     "compliance_advanced",
			rateLimited: true,
		},
		{path: APIVersionPrefix + "/shell/vulnerabilities/status", handler: s.handleVulnerabilityStatus},
		{path: APIVersionPrefix + "/shell/vulnerabilities/results", handler: s.handleVulnerabilityResults},
		{path: APIVersionPrefix + "/shell/vulnerabilities/device", handler: s.handleDeviceVulnerabilities},
		{
			path:    APIVersionPrefix + "/shell/vulnerabilities/settings",
			handler: s.handleVulnerabilitySettings,
			minRole: op,
		},
		{path: APIVersionPrefix + "/shell/vulnerabilities/validate-api-key", handler: s.handleNVDAPIKeyValidate},
		// Guest-network isolation audit (#397).
		{path: APIVersionPrefix + "/shell/guest-audit/settings", handler: s.handleGuestAuditSettings, minRole: op},
		{
			path:        APIVersionPrefix + "/shell/guest-audit/run",
			handler:     s.handleGuestAuditRun,
			feature:     "compliance_advanced",
			rateLimited: true,
		},
		{path: APIVersionPrefix + "/shell/pipeline/status", handler: s.handlePipelineStatus},
		{path: APIVersionPrefix + "/shell/pipeline/start", handler: s.handlePipelineStart},
		{path: APIVersionPrefix + "/shell/pipeline/cancel", handler: s.handlePipelineCancel},
		{path: APIVersionPrefix + "/shell/pipeline/config", handler: s.handlePipelineConfigRoute, minRole: op},
		{path: APIVersionPrefix + "/shell/pipeline/port-intensity", handler: s.handlePipelinePortIntensityInfo},
		{path: APIVersionPrefix + "/shell/pipeline/timing-profiles", handler: s.handlePipelineTimingProfiles},
		// Network problem detection.
		{path: APIVersionPrefix + "/shell/problems", handler: s.handleNetworkProblems},
		{path: APIVersionPrefix + "/shell/problems/scan", handler: s.handleProblemScan},
		{path: APIVersionPrefix + "/shell/problems/thresholds", handler: s.handleProblemThresholds, minRole: op},
		// Bluetooth discovery.
		{path: APIVersionPrefix + "/shell/bluetooth/scan", handler: s.handleBluetoothScan},
		{path: APIVersionPrefix + "/shell/bluetooth/devices", handler: s.handleBluetoothDevices},
		{path: APIVersionPrefix + "/shell/bluetooth/stats", handler: s.handleBluetoothStats},
		{path: APIVersionPrefix + "/shell/bluetooth/status", handler: s.handleBluetoothStatus},
		// Enhanced WiFi discovery (unified).
		{path: APIVersionPrefix + "/shell/wifi/discovery/scan", handler: s.handleWiFiDiscoveryScan},
		{path: APIVersionPrefix + "/shell/wifi/discovery/networks", handler: s.handleWiFiDiscoveryNetworks},
		{path: APIVersionPrefix + "/shell/wifi/discovery/aps", handler: s.handleWiFiDiscoveryAPs},
		{path: APIVersionPrefix + "/shell/wifi/discovery/stats", handler: s.handleWiFiDiscoveryStats},
		// Discovery Engine (primary unified discovery system).
		{path: APIVersionPrefix + "/discovery/engine", handler: s.handleEngineDiscovery},
		{path: APIVersionPrefix + "/discovery/engine/scan", handler: s.handleEngineScan, rateLimited: true},
		{path: APIVersionPrefix + "/discovery/engine/quick", handler: s.handleEngineQuickScan, rateLimited: true},
		{path: APIVersionPrefix + "/discovery/engine/full", handler: s.handleEngineFullScan, rateLimited: true},
		{path: APIVersionPrefix + "/discovery/engine/stats", handler: s.handleEngineStats},
		{path: APIVersionPrefix + "/discovery/engine/capabilities", handler: s.handleEngineCapabilities},
		{path: APIVersionPrefix + "/discovery/engine/device/", handler: s.handleEngineDevice},
		{path: APIVersionPrefix + "/discovery/engine/events", handler: s.handleEngineEvents},
	})
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
