package api

// server_routes.go contains the HTTP route table: setupRoutes plus the
// per-module setup helpers (core auth/settings, SAP telemetry, Shell, Roots,
// Canopy, Harvest) and the SSE + static file fallback.

import (
	"net/http"

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
	// /nodes path must register BEFORE /nodes/ so the router doesn't
	// treat the list endpoint as a /nodes/{id} request.
	s.mux.HandleFunc(APIVersionPrefix+"/topology/nodes", s.handleTopologyNodes)
	s.mux.HandleFunc(APIVersionPrefix+"/topology/nodes/", s.handleTopologyNodeByID)
	s.mux.HandleFunc(APIVersionPrefix+"/topology/links", s.handleTopologyLinks)

	// Stage A5.2 — alerts API. GET is read-only; the action endpoint
	// is writeGated so only operator+ can ack/resolve.
	s.mux.HandleFunc(APIVersionPrefix+"/alerts", s.handleAlerts)
	s.mux.HandleFunc(APIVersionPrefix+"/alerts/", s.writeGated(s.handleAlertAction))

	// Stage A5.3 — polling targets CRUD. Collection-level routes
	// (GET list, POST create) are method-dispatched in
	// handlePollingTargets; resource-level (GET, PUT, DELETE) in
	// handlePollingTargetByID. Both are writeGated because the
	// collection handler accepts POST and any mutating method
	// requires the operator+ role.
	s.mux.HandleFunc(APIVersionPrefix+"/polling-targets", s.writeGated(s.handlePollingTargets))
	s.mux.HandleFunc(APIVersionPrefix+"/polling-targets/", s.writeGated(s.handlePollingTargetByID))

	// Stage A5.8 — read-only engine registry surface. Useful for
	// the operator UI's "what's running" pane and for ops debugging.
	s.mux.HandleFunc(APIVersionPrefix+"/engines", s.handleEngines)

	// Stage A5.10 — operator-defined alert rules. Both endpoints
	// writeGated because the collection accepts POST and any
	// resource-level mutation requires the operator+ role.
	s.mux.HandleFunc(APIVersionPrefix+"/alert-rules", s.writeGated(s.handleAlertRules))
	s.mux.HandleFunc(APIVersionPrefix+"/alert-rules/", s.writeGated(s.handleAlertRuleByID))
}

// setupAPITokenRoutes registers the Phase D-2 personal-access-token
// endpoints and the read-only license status endpoint the UI uses to
// know whether the mint button should be enabled.
func (s *Server) setupAPITokenRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/tokens", s.writeGated(s.handleAPITokens))
	s.mux.HandleFunc(APIVersionPrefix+"/tokens/", s.writeGated(s.handleAPITokenByID))
	s.mux.HandleFunc(APIVersionPrefix+"/license", s.handleLicenseStatus)

	// Users CRUD (seed#1191 — multi_user). The /users/me endpoint must
	// register BEFORE /users/ so the path router doesn't treat "me"
	// as a {username} suffix. POST /users is admin-only AND Pro-gated
	// (the gate is checked inside the handler so a 403/402 carries the
	// appropriate FeatureGateResponse rather than a generic Pro 402).
	s.mux.HandleFunc(APIVersionPrefix+"/users/me", s.handleCurrentUser)
	s.mux.HandleFunc(APIVersionPrefix+"/users", s.handleUsers)
	s.mux.HandleFunc(APIVersionPrefix+"/users/", s.handleUserByName)
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
	s.mux.Handle(
		APIVersionPrefix+"/roots/traceroute",
		s.endpointRateLimiter().RateLimitMiddleware(
			s.requireFeature("path_analysis", s.handleTraceroute),
		),
	)
	s.mux.HandleFunc(
		APIVersionPrefix+"/roots/path",
		s.requireFeature("path_analysis", s.handlePath),
	)
}

// setupCanopyRoutes registers Canopy module routes (Wi-Fi planning).
func (s *Server) setupCanopyRoutes() {
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi", s.handleWiFi)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/scan", s.handleWiFiScan)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/status", s.handleWiFiStatus)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/channel-graph", s.handleWiFiChannelGraph)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/settings", s.writeGated(s.handleWiFiSettings))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/connect", s.writeGated(s.handleWiFiConnect))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/disconnect", s.writeGated(s.handleWiFiDisconnect))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/saved", s.handleWiFiSavedNetworks)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/wifi/forget", s.writeGated(s.handleWiFiForgetNetwork))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/create", s.writeGated(s.createSurvey))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/list", s.listSurveys)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey", s.getSurvey)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/delete", s.writeGated(s.deleteSurvey))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/start", s.writeGated(s.startSurvey))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/pause", s.writeGated(s.pauseSurvey))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/complete", s.writeGated(s.completeSurvey))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/sample", s.writeGated(s.addSurveySample))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floorplan", s.writeGated(s.updateSurveyFloorPlan))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/settings", s.writeGated(s.updateSurveySettings))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/imported-data", s.writeGated(s.updateSurveyImportedData))
	// AirMapper baseline-diff is a Pro feature (LICENSE_STRATEGY §2):
	// imports an existing AirMapper survey JSON and diffs it against
	// the current floor-plan baseline. The rate limiter still wraps
	// the gated handler so trial / Pro users can't abuse the endpoint.
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/import/airmapper",
		s.endpointRateLimiter().RateLimitMiddleware(
			s.requireFeature("airmapper_baseline_diff", s.writeGated(s.importAirMapper)),
		),
	)
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/heatmap",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.getSurveyHeatmap)),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/dead-zones", s.getSurveyDeadZones)
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floors", s.writeGated(s.handleSurveyFloors))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor", s.writeGated(s.handleSurveyFloor))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor/floorplan", s.writeGated(s.updateFloorFloorPlan))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/floor/sample", s.writeGated(s.addFloorSample))
	s.mux.HandleFunc(APIVersionPrefix+"/canopy/survey/active-floor", s.writeGated(s.setActiveFloor))
	s.mux.Handle(
		APIVersionPrefix+"/canopy/survey/report",
		s.endpointRateLimiter().RateLimitMiddleware(http.HandlerFunc(s.generateSurveyReport)),
	)
}

// setupHarvestRoutes registers Harvest module routes (reporting).
// /harvest/export is gated behind the `export_csv_json` feature
// (Starter or higher) per LICENSE_STRATEGY §2. Log endpoints stay
// ungated because operational visibility is a basic capability for
// every tier; only data extraction (the customer-facing reporting
// surface) is paid.
func (s *Server) setupHarvestRoutes() {
	s.mux.HandleFunc(
		APIVersionPrefix+"/harvest/export",
		s.requireFeature("export_csv_json", s.handleExport),
	)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs", s.handleLogs)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/client", s.handleClientLogs)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/query", s.handleLogsQuery)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/stats", s.handleLogsStats)
	s.mux.HandleFunc(APIVersionPrefix+"/harvest/logs/recent", s.handleLogsRecent)
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
