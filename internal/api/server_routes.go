package api

// server_routes.go contains the HTTP route table: setupRoutes plus the
// per-capability setup helpers (core auth/settings, API tokens, path, Wi-Fi,
// reporting, topology) and the SSE + static file fallback. The telemetry and
// security tables are in server_routes_telemetry.go and
// server_routes_security.go.

import (
	"net/http"
	"slices"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// setupRoutes registers every route on one Registrar. Auth and CSRF are
// declared per route: Auth on every /api route except the pre-session steps
// (sign-in, refresh, first-run setup, recovery, the OAuth handshake), CSRF on
// every authenticated route that takes a state-changing method except logout
// and the client log sink.
func (s *Server) setupRoutes() {
	s.routes = s.newRegistrar()
	get := []string{http.MethodGet}
	s.routes.RegisterAll([]route.Route{
		{Path: "/__version", Handler: s.handleBuildVersion, Methods: get},
		// The route-policy manifest (ADR-0002): a deployment and audit
		// surface like /__version, naming no secret.
		{Path: "/__capabilities", Handler: s.routes.ServeManifest, Methods: get},
	})
	s.setupCoreRoutes()
	s.setupAPITokenRoutes()
	s.setupTelemetryRoutes()
	s.setupSecurityRoutes()
	s.setupPathRoutes()
	s.setupWiFiRoutes()
	s.setupReportingRoutes()
	s.setupTopologyRoutes()
	s.routes.RegisterAll(slices.Concat(s.alertRoutes(), s.vulnerabilityRoutes(), s.flowRoutes(),
		s.dashboardRoutes(), s.jobsRoutes(), s.captureRoutes(), s.targetNetworkRoutes(),
		s.interfaceStatsRoutes()))
	s.setupSSEAndStatic()
}

// setupTopologyRoutes registers the Stage A5.1 read-only topology
// endpoints. All are GET-only and run through the same JWT/PAT auth
// middleware as the rest of /api/v1.
func (s *Server) setupTopologyRoutes() {
	op := roles.Operator
	get := []string{http.MethodGet}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPutDelete := []string{http.MethodGet, http.MethodPut, http.MethodDelete}
	s.routes.RegisterAll([]route.Route{
		// /nodes must register BEFORE /nodes/ so the router doesn't treat the
		// list endpoint as a /nodes/{id} request.
		{Path: APIVersionPrefix + "/topology/nodes", Handler: s.handleTopologyNodes, Methods: get, Auth: true},
		{
			Path:    APIVersionPrefix + "/topology/nodes/",
			Handler: s.handleTopologyNodeByID,
			Methods: get,
			Auth:    true,
		},
		{Path: APIVersionPrefix + "/topology/links", Handler: s.handleTopologyLinks, Methods: get, Auth: true},
		{Path: APIVersionPrefix + "/topology/arp", Handler: s.handleTopologyARP, Methods: get, Auth: true},
		// This device's own neighbour cache, not a remote node's (#328). The
		// path deliberately does not sit under /topology/ — the two are easy
		// to confuse and answer different questions.
		{
			Path:    APIVersionPrefix + "/network/neighbours",
			Handler: s.handleNeighbourCache,
			Methods: get,
			Auth:    true,
		},
		// A5.3 polling targets CRUD: both writeGated (collection accepts POST).
		{
			Path:    APIVersionPrefix + "/polling-targets",
			Handler: s.handlePollingTargets,
			Methods: getPost,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/polling-targets/",
			Handler: s.handlePollingTargetByID,
			Methods: getPutDelete,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		// Device-credential vault CRUD (#1799). Operator+ on the mutating
		// routes, matching /polling-targets: a credential is polling
		// configuration, and an operator who can add targets but not the
		// credentials they reference cannot do the job. Secrets go in and
		// never come back out.
		{
			Path:    APIVersionPrefix + "/device-credentials",
			Handler: s.handleDeviceCredentials,
			Methods: getPost,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/device-credentials/",
			Handler: s.handleDeviceCredentialByID,
			Methods: getPutDelete,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		// A5.8 read-only engine registry surface.
		{Path: APIVersionPrefix + "/engines", Handler: s.handleEngines, Methods: get, Auth: true},
		// A5.10 operator-defined alert rules: both writeGated.
		{
			Path:    APIVersionPrefix + "/alert-rules",
			Handler: s.handleAlertRules,
			Methods: getPost,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/alert-rules/",
			Handler: s.handleAlertRuleByID,
			Methods: getPutDelete,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
	})
}

// setupAPITokenRoutes registers the Phase D-2 personal-access-token
// endpoints and the read-only license status endpoint the UI uses to
// know whether the mint button should be enabled.
func (s *Server) setupAPITokenRoutes() {
	op := roles.Operator
	get := []string{http.MethodGet}
	getPost := []string{http.MethodGet, http.MethodPost}
	del := []string{http.MethodDelete}
	getPatchDelete := []string{http.MethodGet, http.MethodPatch, http.MethodDelete}
	s.routes.RegisterAll([]route.Route{
		{
			Path:    APIVersionPrefix + "/tokens",
			Handler: s.handleAPITokens,
			Methods: getPost,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/tokens/",
			Handler: s.handleAPITokenByID,
			Methods: del,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{Path: APIVersionPrefix + "/license", Handler: s.handleLicenseStatus, Methods: get, Auth: true},
		// Users CRUD (#1191): /users/me registers before /users/ for path routing.
		// POST /users is admin-only AND Pro-gated, enforced inside the handler so the
		// response carries the right 403/402 FeatureGateResponse.
		{Path: APIVersionPrefix + "/users/me", Handler: s.handleCurrentUser, Methods: get, Auth: true},
		{Path: APIVersionPrefix + "/users", Handler: s.handleUsers, Methods: getPost, Auth: true, CSRF: true},
		{
			Path:    APIVersionPrefix + "/users/",
			Handler: s.handleUserByName,
			Methods: getPatchDelete,
			Auth:    true,
			CSRF:    true,
		},
	})
}

// setupCoreRoutes registers auth, settings, config, and setup routes.
func (s *Server) setupCoreRoutes() {
	op := roles.Operator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	put := []string{http.MethodPut}
	del := []string{http.MethodDelete}
	getPut := []string{http.MethodGet, http.MethodPut}
	getPostPut := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	crud := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete}
	authBody := MaxBodySizeAuth  // 1 KB — auth payloads are tiny.
	cfgBody := MaxBodySizeConfig // 64 KB — settings/config JSON.
	s.routes.RegisterAll([]route.Route{
		{
			Path:         APIVersionPrefix + "/auth/login",
			Handler:      s.handleLogin,
			Methods:      post,
			MaxBodyBytes: authBody,
		},
		{
			Path:         APIVersionPrefix + "/auth/logout",
			Handler:      s.handleLogout,
			Methods:      post,
			MaxBodyBytes: authBody,
			Auth:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/refresh",
			Handler:      s.handleRefreshToken,
			Methods:      post,
			MaxBodyBytes: authBody,
		},
		{
			Path:         APIVersionPrefix + "/auth/csrf",
			Handler:      s.handleCSRFToken,
			Methods:      get,
			MaxBodyBytes: authBody,
			Auth:         true,
		},
		// Wave 3 (#85): MFA + WebAuthn endpoints.
		{
			Path:         APIVersionPrefix + "/auth/login/totp",
			Handler:      s.handleLoginTOTP,
			Methods:      post,
			MaxBodyBytes: authBody,
		},
		{
			Path:         APIVersionPrefix + "/auth/totp/setup",
			Handler:      s.handleTOTPSetup,
			Methods:      post,
			MaxBodyBytes: authBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/totp/verify",
			Handler:      s.handleTOTPVerify,
			Methods:      post,
			MaxBodyBytes: authBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/totp/disable",
			Handler:      s.handleTOTPDisable,
			Methods:      post,
			MaxBodyBytes: authBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/mfa/status",
			Handler:      s.handleMFAStatus,
			Methods:      get,
			MaxBodyBytes: authBody,
			Auth:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/webauthn/register/begin",
			Handler:      s.handleWebAuthnRegisterBegin,
			Methods:      post,
			MaxBodyBytes: authBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/webauthn/register/finish",
			Handler:      s.handleWebAuthnRegisterFinish,
			Methods:      post,
			MaxBodyBytes: authBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/auth/webauthn/login/begin",
			Handler:      s.handleWebAuthnLoginBegin,
			Methods:      post,
			MaxBodyBytes: authBody,
		},
		{
			Path:         APIVersionPrefix + "/auth/webauthn/login/finish",
			Handler:      s.handleWebAuthnLoginFinish,
			Methods:      post,
			MaxBodyBytes: authBody,
		},
		{Path: APIVersionPrefix + "/status", Handler: s.handleStatus, Methods: get, Auth: true},
		{
			Path:         APIVersionPrefix + "/settings",
			Handler:      s.handleSettings,
			Methods:      getPut,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/settings/defaults",
			Handler:      s.handleSettingsDefaults,
			Methods:      get,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
		},
		{
			Path:         APIVersionPrefix + "/settings/link",
			Handler:      s.handleLinkSettings,
			Methods:      getPut,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/settings/cable",
			Handler:      s.handleCableTestSettings,
			Methods:      getPut,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
			CSRF:         true,
		},
		{Path: APIVersionPrefix + "/interfaces", Handler: s.handleInterfaces, Methods: get, Auth: true},
		{
			Path:    APIVersionPrefix + "/interface",
			Handler: s.handleInterface,
			Methods: getPut,
			Scope:   op, // PUT persists the active NIC to disk (writeGated: operator+)
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/network/mtu",
			Handler: s.handleSetMTU,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:         APIVersionPrefix + "/config/backups",
			Handler:      s.handleConfigBackups,
			Methods:      get,
			MaxBodyBytes: cfgBody,
			Auth:         true,
		},
		{
			Path:         APIVersionPrefix + "/config/backup",
			Handler:      s.handleConfigBackupCreate,
			Methods:      post,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/config/backup/delete",
			Handler:      s.handleConfigBackupDelete,
			Methods:      del,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/config/restore",
			Handler:      s.handleConfigRestore,
			Methods:      post,
			Scope:        op,
			MaxBodyBytes: cfgBody,
			Auth:         true,
			CSRF:         true,
		},
		{
			Path:         APIVersionPrefix + "/config/version",
			Handler:      s.handleConfigVersion,
			Methods:      get,
			MaxBodyBytes: cfgBody,
			Auth:         true,
		},
		{
			Path:    APIVersionPrefix + "/profiles",
			Handler: s.handleProfiles,
			Methods: crud,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/profiles/active",
			Handler: s.handleActiveProfile,
			Methods: getPostPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/profiles/import",
			Handler: s.handleImportProfiles,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/profiles/export",
			Handler: s.handleExportProfiles,
			Methods: get,
			Auth:    true,
		},
		{
			// The UI switches profiles by posting {profileId} here, which is
			// exactly what /profiles/active accepts. Registered as its own
			// route rather than handled inside the prefix route because a
			// route the registry can see is a route the capability manifest
			// and the route-consumer gate can see.
			Path:    APIVersionPrefix + "/profiles/switch",
			Handler: s.handleSetActiveProfile,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			// PATCH is here and not in crud because only this route accepts
			// one: /profiles/{id}/settings is a partial update, and the
			// collection routes have nothing to patch.
			Path:    APIVersionPrefix + "/profiles/",
			Handler: s.handleProfiles,
			Methods: append(append([]string{}, crud...), http.MethodPatch),
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{Path: APIVersionPrefix + "/setup/status", Handler: s.handleSetupStatus, Methods: get},
		{Path: APIVersionPrefix + "/setup/complete", Handler: s.handleSetupComplete, Methods: post},
		{
			Path:    APIVersionPrefix + "/recovery/status",
			Handler: s.handleRecoveryStatus,
			Methods: get,
		},
		{
			Path:    APIVersionPrefix + "/recovery/complete",
			Handler: s.handleRecoveryComplete,
			Methods: post,
		},
		{
			Path:    APIVersionPrefix + "/recovery/instructions",
			Handler: s.handleRecoveryInstructions,
			Methods: get,
		},
		{Path: APIVersionPrefix + "/sso/providers", Handler: s.handleSSOProviders, Methods: get},
		{Path: APIVersionPrefix + "/sso/login", Handler: s.handleSSOLogin, Methods: get},
		{Path: APIVersionPrefix + "/sso/callback", Handler: s.handleSSOCallback, Methods: get},
		{
			Path:    APIVersionPrefix + "/sso/settings",
			Handler: s.handleSSOSettings,
			Methods: get,
			Scope:   op,
			Auth:    true,
		},
		// SSO update is Pro-gated (requireFeature "sso", #1198) AND operator-gated.
		// NORMALIZATION (ADR-0002): the registry composes canonical order
		// requireFeature(writeGated(h)) — a viewer on Free now receives 402
		// (feature) where the prior hand-wrapped writeGated(requireFeature(...))
		// returned 403 (role) first. Operators and Pro tiers are unaffected.
		{
			Path:    APIVersionPrefix + "/sso/update",
			Handler: s.handleSSOUpdate,
			Methods: put,
			Scope:   op,
			Feature: "sso",
			Auth:    true,
			CSRF:    true,
		},
		{Path: APIVersionPrefix + "/health", Handler: s.handleHealth, Methods: get, Auth: true},
	})
}

// setupPathRoutes registers path-analysis routes.
// Both endpoints are gated on the `path_analysis` feature (Pro tier);
// Free / Starter receive 402 with an upgrade hint. The rate-limit
// middleware still wraps traceroute so abuse remains capped even for
// trial users.
func (s *Server) setupPathRoutes() {
	post := []string{http.MethodPost}
	s.routes.RegisterAll([]route.Route{
		{
			Path:    APIVersionPrefix + "/path/traceroute",
			Handler: s.handleTraceroute,
			Methods: post,
			Feature: "path_analysis",
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/path/path",
			Handler: s.handlePath,
			Methods: post,
			Feature: "path_analysis",
			Auth:    true,
			CSRF:    true,
		},
	})
}

// setupWiFiRoutes registers Wi-Fi routes (Wi-Fi visibility &
// troubleshooting). First module on the declarative capability registry
// (ADR-0002): policy is data, composed by register() in one canonical order.
// Behavior is identical to the prior hand-wrapped form.
func (s *Server) setupWiFiRoutes() {
	op := roles.Operator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	del := []string{http.MethodDelete}
	getPut := []string{http.MethodGet, http.MethodPut}
	getPostPut := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	s.routes.RegisterAll([]route.Route{
		{Path: APIVersionPrefix + "/wifi/wifi", Handler: s.handleWiFi, Methods: getPostPut, Auth: true, CSRF: true},
		{Path: APIVersionPrefix + "/wifi/wifi/scan", Handler: s.handleWiFiScan, Methods: get, Auth: true},
		{Path: APIVersionPrefix + "/wifi/wifi/status", Handler: s.handleWiFiStatus, Methods: get, Auth: true},
		{
			Path:    APIVersionPrefix + "/wifi/wifi/channel-graph",
			Handler: s.handleWiFiChannelGraph,
			Methods: get,
			Auth:    true,
		},
		// Starter: the airspace tree + anomaly stream (#2351). Read-only; the
		// scan and capture sources feed the model out-of-band. Free keeps the
		// raw scan above; the clients inside the tree are Pro (see the handler).
		{
			Path:    APIVersionPrefix + "/wifi/airspace",
			Handler: s.handleWiFiAirspace,
			Methods: get,
			Feature: "wifi_analysis",
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/wifi/anomalies",
			Handler: s.handleWiFiAnomalies,
			Methods: get,
			Feature: "wifi_analysis",
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/wifi/wifi/settings",
			Handler: s.handleWiFiSettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/wifi/wifi/connect",
			Handler: s.handleWiFiConnect,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/wifi/wifi/disconnect",
			Handler: s.handleWiFiDisconnect,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/wifi/wifi/saved",
			Handler: s.handleWiFiSavedNetworks,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/wifi/wifi/forget",
			Handler: s.handleWiFiForgetNetwork,
			Methods: del,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
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
	s.routes.RegisterAll([]route.Route{
		{
			Path:    APIVersionPrefix + "/reporting/export",
			Handler: s.handleExport,
			Methods: get,
			Feature: "export_csv_json",
			Auth:    true,
		},
		{Path: APIVersionPrefix + "/reporting/logs", Handler: s.handleLogs, Methods: get, Auth: true},
		{
			Path:    APIVersionPrefix + "/reporting/logs/client",
			Handler: s.handleClientLogs,
			Methods: post,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/reporting/logs/query",
			Handler: s.handleLogsQuery,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/reporting/logs/stats",
			Handler: s.handleLogsStats,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/reporting/logs/recent",
			Handler: s.handleLogsRecent,
			Methods: get,
			Auth:    true,
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
	s.routes.Register(route.Route{
		Path:    APIVersionPrefix + "/events",
		Handler: s.handleSSE,
		Methods: []string{http.MethodGet},
		Feature: "live_telemetry",
		Auth:    true,
	})
	var ui http.Handler
	frontendFS, err := getUIFS()
	if err != nil {
		logging.GetLogger().
			Warn("Failed to get embedded frontend FS, falling back to disk", "error", err)
		ui = http.FileServer(http.Dir("internal/api/ui"))
	} else {
		logging.GetLogger().Info("Serving frontend from embedded filesystem", "embedded", isUIEmbedded())
		ui = spaHandler(http.FS(frontendFS))
	}
	// Catch-alls, hidden from the API document. An unknown /api path still
	// meets Auth and CSRF before its 404, so an unauthenticated caller cannot
	// tell a route that exists from one that does not.
	s.routes.RegisterAll([]route.Route{
		{Path: "/api/", Handler: ui.ServeHTTP, Auth: true, CSRF: true, Hidden: true},
		{Path: "/", Handler: ui.ServeHTTP, Hidden: true},
	})
}
