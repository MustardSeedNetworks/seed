package api

// server_routes_reporting.go holds the reporting route table registered by
// setupRoutes.

import (
	"net/http"

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"
)

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
