package api

// handlers_dns.go contains the DNS testing handlers.
// Split from handlers_health_checks.go for code organization (Plan F).

import (
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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
	ServerScope      string                 `json:"serverScope"`
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
		ServerScope:  string(result.ServerScope),
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

	if s.dnsTester() == nil {
		sendErrorResponseWithDetails(w, logger, http.StatusServiceUnavailable,
			ErrCodeServiceUnavail, localizer.T("errors.health.dnsNotAvailable"), "")
		return
	}

	currentIface := s.getInterfaceFromRequest(r)
	result := s.dnsTester().Test(r.Context())
	resp := buildDNSResponse(result, currentIface)

	sendJSONResponse(w, logger, http.StatusOK, resp)
}
