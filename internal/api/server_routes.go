package api

// server_routes.go contains the HTTP route table: setupRoutes plus the
// per-capability setup helpers (core auth/settings, telemetry, security, path,
// wifi, reporting) and the SSE + static file fallback.

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// setupRoutes configures all HTTP routes.
func (s *Server) setupRoutes() {
	s.mux.HandleFunc("/__version", s.handleBuildVersion)
	// /__capabilities exposes the route-policy manifest (ADR-0002). Registered
	// directly (not via register()) because it is infra introspection, not an
	// API surface — same as /__version. Reads s.manifest at request time, after
	// the module setups below have populated it.
	s.mux.HandleFunc("/__capabilities", s.handleCapabilities)
	s.setupCoreRoutes()
	s.setupAPITokenRoutes()
	s.registerUpdateRoutes()
	s.setupTelemetryRoutes()
	s.setupSecurityRoutes()
	s.setupPathRoutes()
	s.setupWiFiRoutes()
	s.setupReportingRoutes()
	s.setupTopologyRoutes()
	s.registerAll(s.jobsRoutes())
	s.setupSSEAndStatic()
}

// setupTopologyRoutes registers the Stage A5.1 read-only topology
// endpoints. All are GET-only and run through the same JWT/PAT auth
// middleware as the rest of /api/v1.
func (s *Server) setupTopologyRoutes() {
	op := database.RoleOperator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPutDelete := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	s.registerAll([]route{
		// /nodes must register BEFORE /nodes/ so the router doesn't treat the
		// list endpoint as a /nodes/{id} request.
		{path: APIVersionPrefix + "/topology/nodes", handler: s.handleTopologyNodes, methods: get},
		{
			path:    APIVersionPrefix + "/topology/nodes/",
			handler: s.handleTopologyNodeByID,
			methods: get,
		},
		{path: APIVersionPrefix + "/topology/links", handler: s.handleTopologyLinks, methods: get},
		{path: APIVersionPrefix + "/topology/arp", handler: s.handleTopologyARP, methods: get},
		// A5.2 alerts: GET read-only; the action endpoint is operator-gated.
		{path: APIVersionPrefix + "/alerts", handler: s.handleAlerts, methods: get},
		{
			path:    APIVersionPrefix + "/alerts/",
			handler: s.handleAlertAction,
			methods: post,
			minRole: op,
		},
		// A5.3 polling targets CRUD: both writeGated (collection accepts POST).
		{
			path:    APIVersionPrefix + "/polling-targets",
			handler: s.handlePollingTargets,
			methods: getPost,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/polling-targets/",
			handler: s.handlePollingTargetByID,
			methods: getPutDelete,
			minRole: op,
		},
		// A5.8 read-only engine registry surface.
		{path: APIVersionPrefix + "/engines", handler: s.handleEngines, methods: get},
		// A5.10 operator-defined alert rules: both writeGated.
		{
			path:    APIVersionPrefix + "/alert-rules",
			handler: s.handleAlertRules,
			methods: getPost,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/alert-rules/",
			handler: s.handleAlertRuleByID,
			methods: getPutDelete,
			minRole: op,
		},
	})
}

// setupAPITokenRoutes registers the Phase D-2 personal-access-token
// endpoints and the read-only license status endpoint the UI uses to
// know whether the mint button should be enabled.
func (s *Server) setupAPITokenRoutes() {
	op := database.RoleOperator
	get := []string{http.MethodGet}
	getPost := []string{http.MethodGet, http.MethodPost}
	del := []string{http.MethodDelete}
	getPatchDelete := []string{http.MethodGet, http.MethodPatch, http.MethodDelete}
	s.registerAll([]route{
		{
			path:    APIVersionPrefix + "/tokens",
			handler: s.handleAPITokens,
			methods: getPost,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/tokens/",
			handler: s.handleAPITokenByID,
			methods: del,
			minRole: op,
		},
		{path: APIVersionPrefix + "/license", handler: s.handleLicenseStatus, methods: get},
		// Users CRUD (#1191): /users/me registers before /users/ for path routing.
		// POST /users is admin-only AND Pro-gated, enforced inside the handler so the
		// response carries the right 403/402 FeatureGateResponse.
		{path: APIVersionPrefix + "/users/me", handler: s.handleCurrentUser, methods: get},
		{path: APIVersionPrefix + "/users", handler: s.handleUsers, methods: getPost},
		{path: APIVersionPrefix + "/users/", handler: s.handleUserByName, methods: getPatchDelete},
	})
}

// setupCoreRoutes registers auth, settings, config, and setup routes.
func (s *Server) setupCoreRoutes() {
	op := database.RoleOperator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	put := []string{http.MethodPut}
	del := []string{http.MethodDelete}
	getPut := []string{http.MethodGet, http.MethodPut}
	getPostPut := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	crud := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	authBody := MaxBodySizeAuth  // 1 KB — auth payloads are tiny.
	cfgBody := MaxBodySizeConfig // 64 KB — settings/config JSON.
	s.registerAll([]route{
		{
			path:         APIVersionPrefix + "/auth/login",
			handler:      s.handleLogin,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/logout",
			handler:      s.handleLogout,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/refresh",
			handler:      s.handleRefreshToken,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/csrf",
			handler:      s.handleCSRFToken,
			methods:      get,
			maxBodyBytes: authBody,
		},
		// Wave 3 (#85): MFA + WebAuthn endpoints.
		{
			path:         APIVersionPrefix + "/auth/login/totp",
			handler:      s.handleLoginTOTP,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/totp/setup",
			handler:      s.handleTOTPSetup,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/totp/verify",
			handler:      s.handleTOTPVerify,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/totp/disable",
			handler:      s.handleTOTPDisable,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/mfa/status",
			handler:      s.handleMFAStatus,
			methods:      get,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/webauthn/register/begin",
			handler:      s.handleWebAuthnRegisterBegin,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/webauthn/register/finish",
			handler:      s.handleWebAuthnRegisterFinish,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/webauthn/login/begin",
			handler:      s.handleWebAuthnLoginBegin,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{
			path:         APIVersionPrefix + "/auth/webauthn/login/finish",
			handler:      s.handleWebAuthnLoginFinish,
			methods:      post,
			maxBodyBytes: authBody,
		},
		{path: APIVersionPrefix + "/status", handler: s.handleStatus, methods: get},
		{
			path:         APIVersionPrefix + "/settings",
			handler:      s.handleSettings,
			methods:      getPut,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/settings/defaults",
			handler:      s.handleSettingsDefaults,
			methods:      get,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/settings/link",
			handler:      s.handleLinkSettings,
			methods:      getPut,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/settings/cable",
			handler:      s.handleCableTestSettings,
			methods:      getPut,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{path: APIVersionPrefix + "/interfaces", handler: s.handleInterfaces, methods: get},
		{
			path:    APIVersionPrefix + "/interface",
			handler: s.handleInterface,
			methods: getPut,
			minRole: op, // PUT persists the active NIC to disk (writeGated: operator+)
		},
		{
			path:    APIVersionPrefix + "/network/mtu",
			handler: s.handleSetMTU,
			methods: post,
			minRole: op,
		},
		{
			path:         APIVersionPrefix + "/config/backups",
			handler:      s.handleConfigBackups,
			methods:      get,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/config/backup",
			handler:      s.handleConfigBackupCreate,
			methods:      post,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/config/backup/delete",
			handler:      s.handleConfigBackupDelete,
			methods:      del,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/config/restore",
			handler:      s.handleConfigRestore,
			methods:      post,
			minRole:      op,
			maxBodyBytes: cfgBody,
		},
		{
			path:         APIVersionPrefix + "/config/version",
			handler:      s.handleConfigVersion,
			methods:      get,
			maxBodyBytes: cfgBody,
		},
		{
			path:    APIVersionPrefix + "/profiles",
			handler: s.handleProfiles,
			methods: crud,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/profiles/active",
			handler: s.handleActiveProfile,
			methods: getPostPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/profiles/import",
			handler: s.handleImportProfiles,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/profiles/export",
			handler: s.handleExportProfiles,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/profiles/",
			handler: s.handleProfiles,
			methods: crud,
			minRole: op,
		},
		{path: APIVersionPrefix + "/setup/status", handler: s.handleSetupStatus, methods: get},
		{path: APIVersionPrefix + "/setup/complete", handler: s.handleSetupComplete, methods: post},
		{
			path:    APIVersionPrefix + "/recovery/status",
			handler: s.handleRecoveryStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/recovery/complete",
			handler: s.handleRecoveryComplete,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/recovery/instructions",
			handler: s.handleRecoveryInstructions,
			methods: get,
		},
		{path: APIVersionPrefix + "/sso/providers", handler: s.handleSSOProviders, methods: get},
		{path: APIVersionPrefix + "/sso/login", handler: s.handleSSOLogin, methods: get},
		{path: APIVersionPrefix + "/sso/callback", handler: s.handleSSOCallback, methods: get},
		{
			path:    APIVersionPrefix + "/sso/settings",
			handler: s.handleSSOSettings,
			methods: get,
			minRole: op,
		},
		// SSO update is Pro-gated (requireFeature "sso", #1198) AND operator-gated.
		// NORMALIZATION (ADR-0002): the registry composes canonical order
		// requireFeature(writeGated(h)) — a viewer on Free now receives 402
		// (feature) where the prior hand-wrapped writeGated(requireFeature(...))
		// returned 403 (role) first. Operators and Pro tiers are unaffected.
		{
			path:    APIVersionPrefix + "/sso/update",
			handler: s.handleSSOUpdate,
			methods: put,
			minRole: op,
			feature: "sso",
		},
		{path: APIVersionPrefix + "/health", handler: s.handleHealth, methods: get},
	})
}

// setupTelemetryRoutes registers telemetry routes.
func (s *Server) setupTelemetryRoutes() {
	op := database.RoleOperator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPut := []string{http.MethodGet, http.MethodPut}
	getPostPut := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	s.registerAll([]route{
		{path: APIVersionPrefix + "/telemetry/link", handler: s.handleLink, methods: get},
		{path: APIVersionPrefix + "/telemetry/cable", handler: s.handleCable, methods: get},
		{path: APIVersionPrefix + "/telemetry/dns", handler: s.handleDNS, methods: getPost},
		{
			path:    APIVersionPrefix + "/telemetry/dns/security",
			handler: s.handleDNSSecurity,
			methods: getPost,
		},
		{
			path:    APIVersionPrefix + "/telemetry/dns/security/settings",
			handler: s.handleDNSSecuritySettings,
			methods: getPut,
			minRole: op,
		},
		{path: APIVersionPrefix + "/telemetry/gateway", handler: s.handleGateway, methods: get},
		{
			path:    APIVersionPrefix + "/telemetry/dhcp/rogue",
			handler: s.handleRogueDHCP,
			methods: getPost,
		},
		{
			path:    APIVersionPrefix + "/telemetry/dhcp/rogue/servers",
			handler: s.handleRogueDHCPServers,
			methods: getPost,
		},
		{
			path:    APIVersionPrefix + "/telemetry/dhcp/rogue/config",
			handler: s.handleRogueDHCPConfig,
			methods: getPut,
			minRole: op,
		},
		{path: APIVersionPrefix + "/telemetry/vlan", handler: s.handleVLAN, methods: get},
		{
			path:    APIVersionPrefix + "/telemetry/vlan/traffic",
			handler: s.handleVLANTraffic,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/vlan/interface",
			handler: s.handleVLANInterface,
			methods: getPostPut,
			minRole: op, // POST creates a live kernel VLAN sub-interface (writeGated: operator+)
		},
		{
			path:        APIVersionPrefix + "/telemetry/speedtest",
			handler:     s.handleSpeedtest,
			methods:     post,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/telemetry/speedtest/status",
			handler: s.handleSpeedtestStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/iperf/info",
			handler: s.handleIperfInfo,
			methods: get,
		},
		{
			path:        APIVersionPrefix + "/telemetry/iperf/client",
			handler:     s.handleIperfClient,
			methods:     post,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/telemetry/iperf/client/status",
			handler: s.handleIperfClientStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/iperf/server",
			handler: s.handleIperfServer,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/iperf/server/status",
			handler: s.handleIperfServerStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/iperf/suggestions",
			handler: s.handleIperfSuggestions,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/settings",
			handler: s.handleHealthChecksSettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:        APIVersionPrefix + "/telemetry/health-checks/run",
			handler:     s.handleHealthChecks,
			methods:     getPost,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/results",
			handler: s.handleHealthCheckResults,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/history",
			handler: s.handleHealthCheckHistory,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/scores",
			handler: s.handleHealthCheckScores,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/sla",
			handler: s.handleHealthCheckSLA,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/alerts",
			handler: s.handleHealthCheckAlerts,
			methods: get,
		},
		// Anomaly detection is Pro (LICENSE_STRATEGY §2); base results/history/alerts
		// stay open to all tiers — only the trend/anomaly analysis is paid.
		{
			path:    APIVersionPrefix + "/telemetry/health-checks/anomalies",
			handler: s.handleHealthCheckAnomalies,
			methods: get,
			feature: "anomaly_detection",
		},
		{
			path:    APIVersionPrefix + "/telemetry/snmp/settings",
			handler: s.handleSNMPSettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/telemetry/system/health",
			handler: s.handleSystemHealth,
			methods: get,
		},
		{path: APIVersionPrefix + "/telemetry/ipconfig", handler: s.handleIPConfig, methods: get},
		{
			path:    APIVersionPrefix + "/telemetry/ipconfig/settings",
			handler: s.handleIPSettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/telemetry/publicip",
			handler: s.handlePublicIP,
			methods: getPostPut,
		},
	})
}

// setupSecurityRoutes registers security routes.
func (s *Server) setupSecurityRoutes() {
	op := database.RoleOperator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPut := []string{http.MethodGet, http.MethodPut}
	crud := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	s.registerAll([]route{
		{path: APIVersionPrefix + "/security/discovery", handler: s.handleDiscovery, methods: get},
		{
			path:    APIVersionPrefix + "/security/discovery/probe",
			handler: s.handleTCPProbe,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/security/discovery/portscan",
			handler: s.handlePortScan,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/security/discovery/options",
			handler: s.handleDiscoveryOptions,
			methods: getPut,
			minRole: op, // PUT saves discovery options to disk (writeGated: operator+)
		},
		{
			path:    APIVersionPrefix + "/security/discovery/service/status",
			handler: s.handleDiscoveryServiceStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/discovery/fingerprint",
			handler: s.handleAdvancedFingerprint,
			methods: post,
		},
		{path: APIVersionPrefix + "/security/devices", handler: s.handleDevices, methods: getPost},
		{
			path:        APIVersionPrefix + "/security/devices/scan",
			handler:     s.handleDevicesScan,
			methods:     post,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/security/devices/status",
			handler: s.handleDevicesStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/devices/settings",
			handler: s.handleDevicesSettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/security/devices/subnets",
			handler: s.handleDevicesSubnets,
			methods: crud,
			minRole: op, // POST/PUT/DELETE mutate persisted subnet entries (writeGated: operator+)
		},
		// Vulnerability scan + guest-audit run are compliance_advanced (Pro,
		// LICENSE_STRATEGY §2); read-only results/status/settings stay open so prior
		// scan output remains visible to lower tiers.
		{
			path:        APIVersionPrefix + "/security/vulnerabilities/scan",
			handler:     s.handleVulnerabilityScan,
			methods:     post,
			feature:     "compliance_advanced",
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/security/vulnerabilities/status",
			handler: s.handleVulnerabilityStatus,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/vulnerabilities/results",
			handler: s.handleVulnerabilityResults,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/vulnerabilities/device",
			handler: s.handleDeviceVulnerabilities,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/vulnerabilities/settings",
			handler: s.handleVulnerabilitySettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/security/vulnerabilities/validate-api-key",
			handler: s.handleNVDAPIKeyValidate,
			methods: post,
		},
		// Guest-network isolation audit (#397).
		{
			path:    APIVersionPrefix + "/security/guest-audit/settings",
			handler: s.handleGuestAuditSettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:        APIVersionPrefix + "/security/guest-audit/run",
			handler:     s.handleGuestAuditRun,
			methods:     post,
			feature:     "compliance_advanced",
			rateLimited: true,
		},
		// Network problem detection.
		{
			path:    APIVersionPrefix + "/security/problems",
			handler: s.handleNetworkProblems,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/problems/scan",
			handler: s.handleProblemScan,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/security/problems/thresholds",
			handler: s.handleProblemThresholds,
			methods: getPut,
			minRole: op,
		},
		// Bluetooth discovery.
		{
			path:    APIVersionPrefix + "/security/bluetooth/scan",
			handler: s.handleBluetoothScan,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/security/bluetooth/devices",
			handler: s.handleBluetoothDevices,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/bluetooth/stats",
			handler: s.handleBluetoothStats,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/bluetooth/status",
			handler: s.handleBluetoothStatus,
			methods: get,
		},
		// Enhanced WiFi discovery (unified).
		{
			path:    APIVersionPrefix + "/security/wifi/discovery/scan",
			handler: s.handleWiFiDiscoveryScan,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/security/wifi/discovery/networks",
			handler: s.handleWiFiDiscoveryNetworks,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/wifi/discovery/aps",
			handler: s.handleWiFiDiscoveryAPs,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/security/wifi/discovery/stats",
			handler: s.handleWiFiDiscoveryStats,
			methods: get,
		},
		// Discovery Engine (primary unified discovery system).
		{
			path:    APIVersionPrefix + "/discovery/engine",
			handler: s.handleEngineDiscovery,
			methods: get,
		},
		{
			path:        APIVersionPrefix + "/discovery/engine/scan",
			handler:     s.handleEngineScan,
			methods:     post,
			rateLimited: true,
		},
		{
			path:        APIVersionPrefix + "/discovery/engine/quick",
			handler:     s.handleEngineQuickScan,
			methods:     post,
			rateLimited: true,
		},
		{
			path:        APIVersionPrefix + "/discovery/engine/full",
			handler:     s.handleEngineFullScan,
			methods:     post,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/discovery/engine/stats",
			handler: s.handleEngineStats,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/discovery/engine/capabilities",
			handler: s.handleEngineCapabilities,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/discovery/engine/device/",
			handler: s.handleEngineDevice,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/discovery/engine/events",
			handler: s.handleEngineEvents,
			methods: get,
		},
	})
}

// setupPathRoutes registers path-analysis routes.
// Both endpoints are gated on the `path_analysis` feature (Pro tier);
// Free / Starter receive 402 with an upgrade hint. The rate-limit
// middleware still wraps traceroute so abuse remains capped even for
// trial users.
func (s *Server) setupPathRoutes() {
	post := []string{http.MethodPost}
	s.registerAll([]route{
		{
			path:        APIVersionPrefix + "/path/traceroute",
			handler:     s.handleTraceroute,
			methods:     post,
			feature:     "path_analysis",
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/path/path",
			handler: s.handlePath,
			methods: post,
			feature: "path_analysis",
		},
	})
}

// setupWiFiRoutes registers Wi-Fi routes (Wi-Fi visibility &
// troubleshooting). First module on the declarative capability registry
// (ADR-0002): policy is data, composed by register() in one canonical order.
// Behavior is identical to the prior hand-wrapped form.
func (s *Server) setupWiFiRoutes() {
	op := database.RoleOperator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	put := []string{http.MethodPut}
	del := []string{http.MethodDelete}
	getPut := []string{http.MethodGet, http.MethodPut}
	getPostPut := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	floorBody := MaxBodySizeFloorPlan // 10 MB — floor-plan image uploads.
	airBody := MaxBodySizeAirMapper   // 50 MB — AirMapper survey imports.
	s.registerAll([]route{
		{path: APIVersionPrefix + "/wifi/wifi", handler: s.handleWiFi, methods: getPostPut},
		{path: APIVersionPrefix + "/wifi/wifi/scan", handler: s.handleWiFiScan, methods: get},
		{path: APIVersionPrefix + "/wifi/wifi/status", handler: s.handleWiFiStatus, methods: get},
		{
			path:    APIVersionPrefix + "/wifi/wifi/channel-graph",
			handler: s.handleWiFiChannelGraph,
			methods: get,
		},
		// Pro: monitor-mode airspace visibility + anomaly stream (W5). Read-only;
		// the capture source feeds the model out-of-band. Feature-gated so the
		// tree/forensics surface stays on the Pro tier (LICENSE_STRATEGY.md).
		{
			path:    APIVersionPrefix + "/wifi/airspace",
			handler: s.handleWiFiAirspace,
			methods: get,
			feature: "wifi_management_capture",
		},
		{
			path:    APIVersionPrefix + "/wifi/anomalies",
			handler: s.handleWiFiAnomalies,
			methods: get,
			feature: "wifi_association_forensics",
		},
		{
			path:    APIVersionPrefix + "/wifi/wifi/settings",
			handler: s.handleWiFiSettings,
			methods: getPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/wifi/connect",
			handler: s.handleWiFiConnect,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/wifi/disconnect",
			handler: s.handleWiFiDisconnect,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/wifi/saved",
			handler: s.handleWiFiSavedNetworks,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/wifi/wifi/forget",
			handler: s.handleWiFiForgetNetwork,
			methods: del,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/create",
			handler: s.createSurvey,
			methods: post,
			minRole: op,
		},
		{path: APIVersionPrefix + "/wifi/survey/list", handler: s.listSurveys, methods: get},
		{path: APIVersionPrefix + "/wifi/survey", handler: s.getSurvey, methods: get},
		{
			path:    APIVersionPrefix + "/wifi/survey/delete",
			handler: s.deleteSurvey,
			methods: del,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/start",
			handler: s.startSurvey,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/pause",
			handler: s.pauseSurvey,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/complete",
			handler: s.completeSurvey,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/sample",
			handler: s.addSurveySample,
			methods: post,
			minRole: op,
		},
		{
			path:         APIVersionPrefix + "/wifi/survey/floorplan",
			handler:      s.updateSurveyFloorPlan,
			methods:      put,
			minRole:      op,
			maxBodyBytes: floorBody,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/settings",
			handler: s.updateSurveySettings,
			methods: put,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/imported-data",
			handler: s.updateSurveyImportedData,
			methods: put,
			minRole: op,
		},
		// AirMapper baseline-diff (Pro, LICENSE_STRATEGY §2): imports an AirMapper
		// survey JSON and diffs it against the floor-plan baseline; rate-limited.
		{
			path:         APIVersionPrefix + "/wifi/survey/import/airmapper",
			handler:      s.importAirMapper,
			methods:      post,
			minRole:      op,
			feature:      "airmapper_baseline_diff",
			rateLimited:  true,
			maxBodyBytes: airBody,
		},
		{
			path:        APIVersionPrefix + "/wifi/survey/heatmap",
			handler:     s.getSurveyHeatmap,
			methods:     get,
			rateLimited: true,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/dead-zones",
			handler: s.getSurveyDeadZones,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/floors",
			handler: s.handleSurveyFloors,
			methods: getPostPut,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/floor",
			handler: s.handleSurveyFloor,
			methods: getPostPut,
			minRole: op,
		},
		{
			path:         APIVersionPrefix + "/wifi/survey/floor/floorplan",
			handler:      s.updateFloorFloorPlan,
			methods:      put,
			minRole:      op,
			maxBodyBytes: floorBody,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/floor/sample",
			handler: s.addFloorSample,
			methods: post,
			minRole: op,
		},
		{
			path:    APIVersionPrefix + "/wifi/survey/active-floor",
			handler: s.setActiveFloor,
			methods: put,
			minRole: op,
		},
		{
			path:        APIVersionPrefix + "/wifi/survey/report",
			handler:     s.generateSurveyReport,
			methods:     post,
			rateLimited: true,
		},
	})
}

// setupReportingRoutes registers reporting routes.
// /reporting/export is gated behind the `export_csv_json` feature
// (Starter or higher) per LICENSE_STRATEGY §2. Log endpoints stay
// ungated because operational visibility is a basic capability for
// every tier; only data extraction (the customer-facing reporting
// surface) is paid.
func (s *Server) setupReportingRoutes() {
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	s.registerAll([]route{
		{
			path:    APIVersionPrefix + "/reporting/export",
			handler: s.handleExport,
			methods: get,
			feature: "export_csv_json",
		},
		{path: APIVersionPrefix + "/reporting/logs", handler: s.handleLogs, methods: get},
		{
			path:    APIVersionPrefix + "/reporting/logs/client",
			handler: s.handleClientLogs,
			methods: post,
		},
		{
			path:    APIVersionPrefix + "/reporting/logs/query",
			handler: s.handleLogsQuery,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/reporting/logs/stats",
			handler: s.handleLogsStats,
			methods: get,
		},
		{
			path:    APIVersionPrefix + "/reporting/logs/recent",
			handler: s.handleLogsRecent,
			methods: get,
		},
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
	s.register(route{
		path:    APIVersionPrefix + "/events",
		handler: s.handleSSE,
		methods: []string{http.MethodGet},
		feature: "live_telemetry",
	})
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
