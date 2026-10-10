package api

// server_routes_telemetry.go holds the telemetry route table, registered by
// setupRoutes.

import (
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

// setupTelemetryRoutes registers telemetry routes.
func (s *Server) setupTelemetryRoutes() {
	op := roles.Operator
	get := []string{http.MethodGet}
	post := []string{http.MethodPost}
	getPost := []string{http.MethodGet, http.MethodPost}
	getPut := []string{http.MethodGet, http.MethodPut}
	getPostPut := []string{http.MethodGet, http.MethodPost, http.MethodPut}
	s.routes.RegisterAll([]route.Route{
		{Path: APIVersionPrefix + "/telemetry/link", Handler: s.handleLink, Methods: get, Auth: true},
		{Path: APIVersionPrefix + "/telemetry/cable", Handler: s.handleCable, Methods: get, Auth: true},
		// The NIC driver's own error counters (#416). Linux only; the handler
		// refuses elsewhere with a 501 that names the reason.
		{
			Path:    APIVersionPrefix + "/telemetry/interface/driver-stats",
			Handler: s.handleDriverStats,
			Methods: get,
			Auth:    true,
		},
		{Path: APIVersionPrefix + "/telemetry/dns", Handler: s.handleDNS, Methods: getPost, Auth: true, CSRF: true},
		{Path: APIVersionPrefix + "/telemetry/gateway", Handler: s.handleGateway, Methods: get, Auth: true},
		// Reads the lease this host already holds — no DISCOVER is sent and no
		// target is supplied, so it is a read of local state rather than an
		// active scan. Rate limited all the same: it shells out to a platform
		// command, and there is no reason to allow that in a tight loop.
		{
			Path:    APIVersionPrefix + "/telemetry/dhcp/lease",
			Handler: s.handleDHCPLease,
			Methods: getPost,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			// Operator+ and rate-limited: this restarts the DHCP client on a
			// live interface, so it is a persistent write in every sense that
			// matters even though nothing is written to disk (#170).
			Path:    APIVersionPrefix + "/telemetry/dhcp/renew",
			Handler: s.handleRenewDHCPLease,
			Methods: post,
			Scope:   op,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/dhcp/rogue",
			Handler: s.handleRogueDHCP,
			Methods: getPost,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/dhcp/rogue/servers",
			Handler: s.handleRogueDHCPServers,
			Methods: getPost,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/dhcp/rogue/config",
			Handler: s.handleRogueDHCPConfig,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{Path: APIVersionPrefix + "/telemetry/vlan", Handler: s.handleVLAN, Methods: get, Auth: true},
		{
			Path:    APIVersionPrefix + "/telemetry/vlan/interface",
			Handler: s.handleVLANInterface,
			Methods: getPostPut,
			Scope:   op, // POST creates a live kernel VLAN sub-interface (writeGated: operator+)
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/speedtest",
			Handler: s.handleSpeedtest,
			Methods: post,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/speedtest/status",
			Handler: s.handleSpeedtestStatus,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/iperf/info",
			Handler: s.handleIperfInfo,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/iperf/client",
			Handler: s.handleIperfClient,
			Methods: post,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/iperf/client/status",
			Handler: s.handleIperfClientStatus,
			Methods: get,
			Auth:    true,
		},
		{
			// POST, not GET: handleIperfServer answers 405 to anything else, and
			// the route declared `get`, so the methodGate and the handler
			// refused opposite halves and NOTHING worked — start and stop were
			// both dead on every role. Operator+, because it binds a listener.
			Path:    APIVersionPrefix + "/telemetry/iperf/server",
			Handler: s.handleIperfServer,
			Methods: post,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/iperf/server/status",
			Handler: s.handleIperfServerStatus,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/iperf/suggestions",
			Handler: s.handleIperfSuggestions,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/probes/settings",
			Handler: s.handleHealthChecksSettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/probes/run",
			Handler: s.handleHealthChecks,
			Methods: getPost,
			Limiter: limitEndpoint,
			Auth:    true,
			CSRF:    true,
		},
		// Anomaly detection is Pro (LICENSE_STRATEGY §2). It reads the unified
		// anomaly store's source=probe slice (ADR-0021/0025); the legacy
		// results/history/scores/sla/alerts read-path over health_check_results
		// was deleted as dead code (ADR-0026).
		{
			Path:    APIVersionPrefix + "/telemetry/probes/anomalies",
			Handler: s.handleHealthCheckAnomalies,
			Methods: get,
			Feature: "anomaly_detection",
			Auth:    true,
		},
		// #175's bounded history read surface. Reads only, so no role gate;
		// the window each licence tier can answer is resolved per request
		// from the retention horizons rather than gated as a feature, so a
		// Free deployment gets its 7 days rather than a 402.
		{
			// Spelled as a literal, not the historyProbesPathPrefix const the
			// handler trims with: scripts/check-route-consumers.py matches on
			// the string in this table, so a const here hides the route from
			// the ratchet entirely.
			Path:    APIVersionPrefix + "/history/probes/",
			Handler: s.handleProbeHistory,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/history/anomalies",
			Handler: s.handleAnomalyHistory,
			Methods: get,
			Auth:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/snmp/settings",
			Handler: s.handleSNMPSettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/system/health",
			Handler: s.handleSystemHealth,
			Methods: get,
			Auth:    true,
		},
		{Path: APIVersionPrefix + "/telemetry/ipconfig", Handler: s.handleIPConfig, Methods: get, Auth: true},
		{
			Path:    APIVersionPrefix + "/telemetry/ipconfig/settings",
			Handler: s.handleIPSettings,
			Methods: getPut,
			Scope:   op,
			Auth:    true,
			CSRF:    true,
		},
		{
			Path:    APIVersionPrefix + "/telemetry/publicip",
			Handler: s.handlePublicIP,
			Methods: getPostPut,
			Auth:    true,
			CSRF:    true,
		},
	})
}
