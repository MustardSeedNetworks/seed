// Package api provides the HTTP/WebSocket server.
// handlers_dns.go contains DNS testing and security scanning handlers.
// Split from handlers_health_checks.go for code organization (Plan F).
package api

import (
	"encoding/json"
	"net/http"

	"github.com/krisarmstrong/seed/internal/i18n"
	"github.com/krisarmstrong/seed/internal/logging"
	"github.com/krisarmstrong/seed/internal/sap/dns"
)

// ============================================================================
// DNS Testing Types
// ============================================================================

// DNSLookupResult represents a DNS lookup result for the API.
type DNSLookupResult struct {
	Result   string   `json:"result"`
	Time     int64    `json:"time"` // ms (deprecated, use timeMs)
	TimeMs   int64    `json:"timeMs"`
	Status   string   `json:"status"`
	Error    string   `json:"error,omitempty"`
	Resolved []string `json:"resolved,omitempty"`
}

// DNSServerTestResult represents per-server DNS test results for the API.
type DNSServerTestResult struct {
	Server      string           `json:"server"`
	Forward     *DNSLookupResult `json:"forward,omitempty"`
	ForwardIpv6 *DNSLookupResult `json:"forwardIpv6,omitempty"`
	Status      string           `json:"status"`
	AvgTimeMs   int64            `json:"avgTimeMs"`
}

// DNSResponse represents the DNS test results for the API.
type DNSResponse struct {
	Interface        string                 `json:"interface"`
	Server           string                 `json:"server"`
	Servers          []string               `json:"servers"`
	TestHostname     string                 `json:"testHostname"`
	Forward          *DNSLookupResult       `json:"forward,omitempty"`
	ForwardIpv6      *DNSLookupResult       `json:"forwardIpv6,omitempty"`
	Reverse          *DNSLookupResult       `json:"reverse,omitempty"`
	ReverseIpv6      *DNSLookupResult       `json:"reverseIpv6,omitempty"`
	PerServerResults []*DNSServerTestResult `json:"perServerResults,omitempty"`
}

// ============================================================================
// DNS Testing Handlers
// ============================================================================

// convertDNSLookup converts dns.LookupResult to DNSLookupResult API type.
func convertDNSLookup(src *dns.LookupResult) *DNSLookupResult {
	if src == nil {
		return nil
	}
	return &DNSLookupResult{
		Result:   src.Result,
		Time:     src.TimeMs,
		TimeMs:   src.TimeMs,
		Status:   string(src.Status),
		Error:    src.Error,
		Resolved: src.Resolved,
	}
}

// buildDNSResponse builds the DNSResponse from dns.TestResult.
func buildDNSResponse(result *dns.TestResult, iface string) DNSResponse {
	resp := DNSResponse{
		Interface:    iface,
		Server:       result.Server,
		Servers:      result.Servers,
		TestHostname: result.TestHostname,
		Forward:      convertDNSLookup(result.Forward),
		ForwardIpv6:  convertDNSLookup(result.ForwardIPv6),
		Reverse:      convertDNSLookup(result.Reverse),
		ReverseIpv6:  convertDNSLookup(result.ReverseIPv6),
	}

	for _, sr := range result.PerServerResults {
		resp.PerServerResults = append(resp.PerServerResults, &DNSServerTestResult{
			Server:      sr.Server,
			Status:      string(sr.Status),
			AvgTimeMs:   sr.AvgTimeMs,
			Forward:     convertDNSLookup(sr.Forward),
			ForwardIpv6: convertDNSLookup(sr.ForwardIPv6),
		})
	}
	return resp
}

// handleDNS performs DNS testing and returns results.
func (s *Server) handleDNS(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
		return
	}

	if s.dnsTester == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.health.dnsNotAvailable"), "")
		return
	}

	currentIface := s.getInterfaceFromRequest(r)
	result := s.dnsTester.Test(r.Context())
	resp := buildDNSResponse(result, currentIface)

	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// ============================================================================
// DNS Security Scanning Types and Handlers
// ============================================================================

// DNSSecurityScanRequest represents a request to scan DNS servers for security issues.
type DNSSecurityScanRequest struct {
	Servers []string `json:"servers"`
}

// handleDNSSecurity handles DNS security scanning operations.
// POST - Scan specific DNS servers for security issues.
// GET - Get results of previous scans.
func (s *Server) handleDNSSecurity(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.dnsSecurityScanner == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.health.dnsSecurityNotAvailable"), "")
		return
	}

	switch r.Method {
	case http.MethodPost:
		// Trigger a security scan
		var req DNSSecurityScanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Warn("Invalid request body for DNS security scan", "error", err)
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
			return
		}

		if len(req.Servers) == 0 {
			// Use configured DNS servers from config
			s.config.RLock()
			for _, srv := range s.config.DNS.Servers {
				if srv.Enabled {
					req.Servers = append(req.Servers, srv.Address)
				}
			}
			s.config.RUnlock()
		}

		if len(req.Servers) == 0 {
			sendErrorResponseWithDetails(
				w,
				logger,
				http.StatusBadRequest,
				ErrCodeBadRequest,
				localizer.T("errors.health.noServersToScan"),
				"No DNS servers provided or configured",
			)
			return
		}

		// Check if already running
		if s.dnsSecurityScanner.IsRunning() {
			sendErrorResponseWithDetails(w, logger, http.StatusConflict,
				ErrCodeConflict, localizer.T("errors.health.scanInProgress"), "")
			return
		}

		// Run concurrent scans
		results, err := s.dnsSecurityScanner.ScanServers(r.Context(), req.Servers, 5)
		if err != nil {
			logger.Error("DNS security scan failed", "error", err)
			sendErrorResponseWithDetails(w, logger, http.StatusInternalServerError,
				ErrCodeInternal, localizer.T("errors.health.scanFailed"), "")
			return
		}

		sendJSONResponse(w, logger, http.StatusOK, results)

	case http.MethodGet:
		// Return cached results
		results := s.dnsSecurityScanner.GetResults()
		sendJSONResponse(w, logger, http.StatusOK, results)

	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	}
}

// handleDNSSecuritySettings handles DNS security scanner settings.
// GET - Get current settings.
// PUT - Update settings.
func (s *Server) handleDNSSecuritySettings(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if s.dnsSecurityScanner == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.health.dnsSecurityNotAvailable"), "")
		return
	}

	switch r.Method {
	case http.MethodGet:
		scanConfig := s.dnsSecurityScanner.GetConfig()
		sendJSONResponse(w, logger, http.StatusOK, scanConfig)

	case http.MethodPut:
		var newConfig dns.SecurityScanConfig
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			logger.Warn("Invalid request body for DNS security config", "error", err)
			sendErrorResponseWithDetails(w, logger, http.StatusBadRequest,
				ErrCodeBadRequest, localizer.T("errors.api.invalidRequestBody"), "")
			return
		}

		s.dnsSecurityScanner.SetConfig(newConfig)

		sendJSONResponse(w, logger, http.StatusOK, map[string]string{
			"status":  "success",
			"message": "DNS security settings updated",
		})

	default:
		sendErrorResponseWithDetails(w, logger, http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed, localizer.T("errors.api.methodNotAllowed"), "")
	}
}
