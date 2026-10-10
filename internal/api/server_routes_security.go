package api

// server_routes_security.go holds the security route table — discovery,
// devices, problems and reports — registered by setupRoutes.

import (
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

// setupSecurityRoutes registers security routes.
func (s *Server) setupSecurityRoutes() {
	op := roles.Operator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPut := []string{http.MethodGet, http.MethodPut}
	s.routes.RegisterAll([]route.Route{
		{Path: APIVersionPrefix + "/security/discovery", Handler: s.handleDiscovery, Methods: get, Auth: true},
		// probe and fingerprint connect to a caller-named host just as
		// portscan does, so they carry the same gate (#2635).
		{
			Path:    APIVersionPrefix + "/security/discovery/probe",
			Handler: s.handleTCPProbe,
			Methods: post,
			Scope:   op,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		// Port scanning is an active, outbound operation against an
		// operator-supplied target. Every sibling active scan in this file
		// carries rateLimited; this route had neither that nor a role gate, so
		// a viewer could scan arbitrary hosts at will. Operator+ matches the other routes that act on the network
		// rather than read from it (#347).
		{
			Path:    APIVersionPrefix + "/security/discovery/portscan",
			Handler: s.handlePortScan,
			Methods: post,
			Scope:   op,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/discovery/options",
			Handler: s.handleDiscoveryOptions,
			Methods: getPut,
			Scope:   op, // PUT saves discovery options to disk (writeGated: operator+)
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/discovery/service/status",
			Handler: s.handleDiscoveryServiceStatus,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/discovery/fingerprint",
			Handler: s.handleAdvancedFingerprint,
			Methods: post,
			Scope:   op,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/devices",
			Handler: s.handleDevices,
			Methods: getPost,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/devices/scan",
			Handler: s.handleDevicesScan,
			Methods: post,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/devices/status",
			Handler: s.handleDevicesStatus,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/devices/settings",
			Handler: s.handleDevicesSettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		// Reports (#2154). export_csv_json is Starter+ (license/policy.go), the
		// same gate ReportsPage already applies -- but the page gate is
		// cosmetic on its own, so it belongs here too.
		//
		// Generation sits on its own path rather than POST /reports because
		// rateLimited wraps the whole route and the shared endpoint limiter is
		// 5 requests/minute: on the collection it would throttle list reads.
		// ServeMux prefers the exact pattern over the /reports/ prefix.
		{
			Path:    APIVersionPrefix + "/reports",
			Handler: s.handleReports,
			Methods: get,
			Feature: "export_csv_json",
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/reports/generate",
			Handler: s.handleReportGenerate,
			Methods: post,
			Scope:   op,
			Feature: "export_csv_json",
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		// Scheduled reports are Pro (scheduled_reports). The more specific
		// /reports/schedules/ pattern wins over /reports/ in ServeMux.
		{
			Path:    APIVersionPrefix + "/reports/schedules",
			Handler: s.handleReportSchedules,
			Methods: getPost,
			Scope:   op,
			Feature: "scheduled_reports",
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/reports/schedules/",
			Handler: s.handleReportScheduleByID,
			Methods: []string{http.MethodGet, http.MethodPut, http.MethodDelete},
			Scope:   op,
			Feature: "scheduled_reports",
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/reports/",
			Handler: s.handleReportByID,
			Methods: []string{http.MethodGet, http.MethodDelete},
			Scope:   op, // gates DELETE only; GET stays open to viewers
			Feature: "export_csv_json",
			Auth:    true,
			CSRF:    true,
		},
		// Guest-network isolation audit (#397); the run is compliance_advanced
		// (Pro, LICENSE_STRATEGY §2).
		{
			Path:    APIVersionPrefix + "/security/guest-audit/settings",
			Handler: s.handleGuestAuditSettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/guest-audit/run",
			Handler: s.handleGuestAuditRun,
			Methods: post,
			Feature: "compliance_advanced",
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		// Network problem detection.
		{
			Path:    APIVersionPrefix + "/security/problems",
			Handler: s.handleNetworkProblems,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/problems/scan",
			Handler: s.handleProblemScan,
			Methods: post,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/problems/thresholds",
			Handler: s.handleProblemThresholds,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		// Enhanced WiFi discovery (unified).
		{
			Path:    APIVersionPrefix + "/security/wifi/discovery/scan",
			Handler: s.handleWiFiDiscoveryScan,
			Methods: post,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/wifi/discovery/networks",
			Handler: s.handleWiFiDiscoveryNetworks,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/wifi/discovery/aps",
			Handler: s.handleWiFiDiscoveryAPs,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/security/wifi/discovery/stats",
			Handler: s.handleWiFiDiscoveryStats,
			Methods: get,
			Auth:    true,
		},
		{
			// #364's Bonjour browse; reads the segment, so no role gate. A
			// literal, not a const: the route-consumer ratchet matches on the
			// string in this table (see /history/* above).
			Path:    APIVersionPrefix + "/discovery/bonjour",
			Handler: s.handleBonjourBrowse,
			Methods: get,
			Auth:    true,
		},
		// Discovery Engine (primary unified discovery system).
		{
			Path:    APIVersionPrefix + "/discovery/engine",
			Handler: s.handleEngineDiscovery,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/scan",
			Handler: s.handleEngineScan,
			Methods: post,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/quick",
			Handler: s.handleEngineQuickScan,
			Methods: post,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/full",
			Handler: s.handleEngineFullScan,
			Methods: post,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/stats",
			Handler: s.handleEngineStats,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/capabilities",
			Handler: s.handleEngineCapabilities,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/device/",
			Handler: s.handleEngineDevice,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/discovery/engine/events",
			Handler: s.handleEngineEvents,
			Methods: get,
			Auth:    true,
		},
	})
}
