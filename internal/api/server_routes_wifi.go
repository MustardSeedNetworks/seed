package api

// server_routes_wifi.go holds the Wi-Fi route table registered by setupRoutes.

import (
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
)

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
	s.routes.RegisterAll([]route.Route{
		// GET only, the one method handleWiFi serves; POST and PUT were declared
		// as well and reached the handler only to be refused with a 405 (#3261).
		{Path: APIVersionPrefix + "/wifi/wifi", Handler: s.handleWiFi, Methods: get, Auth: true},
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
