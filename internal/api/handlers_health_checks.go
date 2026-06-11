package api

// handlers_health_checks.go contains the core health-checks entry handler and
// the shared constants, result types, and small helpers used across the
// protocol-specific health-check files (ping, port, http, http_security, tls,
// rtsp, dicom). DNS, Speedtest, and iPerf handlers live in separate files.

import (
	"context"
	"net/http"

	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// Test status and protocol constants.
const (
	statusError      = "error"
	statusWarning    = "warning"
	statusSuccess    = "success"
	protoTCP         = "tcp"
	protoUDP         = "udp"
	errHTTPReqFailed = "HTTP request failed"
)

// URL and score constants.
const (
	// httpsSchemeLen is the minimum length to check for "https://" in URL.
	httpsSchemeLen = 8
	// percentMultiplier converts score ratio to percentage.
	percentMultiplier = 100
	// scoreThresholdGood is the minimum score for "success" status.
	scoreThresholdGood = 80
	// scoreThresholdWarn is the minimum score for "warning" status.
	scoreThresholdWarn = 50
)

// Health check timing and configuration constants.
const (
	// defaultPingCount is the number of ping probes per target for extended ping tests.
	defaultPingCount = 5

	// pingProbeTimeoutSec is the timeout in seconds for each individual ping probe.
	pingProbeTimeoutSec = 2

	// pingProbeDelayMs is the delay in milliseconds between consecutive ping probes.
	pingProbeDelayMs = 100

	// tcpTestTimeoutSec is the timeout in seconds for TCP connectivity tests.
	tcpTestTimeoutSec = 5

	// udpTestTimeoutSec is the timeout in seconds for UDP connectivity tests.
	udpTestTimeoutSec = 5

	// udpReadDeadlineSec is the deadline in seconds for reading UDP responses.
	udpReadDeadlineSec = 3

	// udpReadBufferBytes is the buffer size in bytes for reading UDP responses.
	udpReadBufferBytes = 1024

	// certCheckTimeoutSec is the timeout in seconds for TLS certificate checks.
	certCheckTimeoutSec = 5

	// hoursPerDay is the number of hours in a day for certificate expiry calculations.
	hoursPerDay = 24

	// percentageDivisor converts ratios to percentages (multiply by 100).
	percentageDivisor = 100

	// packetLossThresholdFull indicates complete packet loss (100%).
	packetLossThresholdFull = 100

	// packetLossThresholdHigh indicates severe packet loss threshold (50%).
	packetLossThresholdHigh = 50

	// packetLossThresholdLow indicates elevated packet loss threshold (10%).
	packetLossThresholdLow = 10

	// millisecondsPerSecond is the conversion factor from seconds to milliseconds.
	millisecondsPerSecond = 1000

	// dnsPort is the standard DNS service port.
	dnsPort = 53

	// httpClientTimeoutSec is the timeout in seconds for HTTP client requests.
	httpClientTimeoutSec = 10
)

// ============================================================================
// Health Checks Test Types
// ============================================================================

// CustomTestResult represents the result of a single custom test.
type CustomTestResult struct {
	Name        string  `json:"name"`
	Host        string  `json:"host"`
	Port        int     `json:"port,omitempty"`
	URL         string  `json:"url,omitempty"`
	Success     bool    `json:"success"`
	Latency     float64 `json:"latency"` // ms
	DNSLatency  float64 `json:"dnsLatency,omitempty"`
	TCPConnect  float64 `json:"tcpConnect,omitempty"`
	TLSLatency  float64 `json:"tlsLatency,omitempty"`
	TTFBLatency float64 `json:"ttfbLatency,omitempty"` // Time to first byte (server processing + wait)
	Error       string  `json:"error,omitempty"`
	Status      int     `json:"status,omitempty"`     // HTTP status code
	TestStatus  string  `json:"testStatus,omitempty"` // success, warning, error
	// Per-phase status fields for HTTP timing breakdown
	DNSStatus  string `json:"dnsStatus,omitempty"`  // success, warning, error
	TCPStatus  string `json:"tcpStatus,omitempty"`  // success, warning, error
	TLSStatus  string `json:"tlsStatus,omitempty"`  // success, warning, error
	TTFBStatus string `json:"ttfbStatus,omitempty"` // success, warning, error
	// Extended ping fields
	PacketLoss float64 `json:"packetLoss,omitempty"` // Percentage
	Jitter     float64 `json:"jitter,omitempty"`     // ms
	MinLatency float64 `json:"minLatency,omitempty"` // ms
	MaxLatency float64 `json:"maxLatency,omitempty"` // ms
	// Certificate fields
	CertDaysLeft   int    `json:"certDaysLeft,omitempty"`   // Days until cert expires
	CertStatus     string `json:"certStatus,omitempty"`     // success, warning, error
	CertExpiry     string `json:"certExpiry,omitempty"`     // Expiry date string
	CertCommonName string `json:"certCommonName,omitempty"` // Certificate CN
	TLSVersion     string `json:"tlsVersion,omitempty"`     // TLS 1.2, TLS 1.3, etc.
	CertIssuer     string `json:"certIssuer,omitempty"`     // Certificate issuer
	// HTTP enhancements (Health Checks 100x)
	BodyMatchSuccess bool             `json:"bodyMatchSuccess,omitempty"` // True if body matched pattern
	BodyMatchStatus  string           `json:"bodyMatchStatus,omitempty"`  // success, error
	ResponseSize     int64            `json:"responseSize,omitempty"`     // Response body size in bytes
	HTTPVersion      string           `json:"httpVersion,omitempty"`      // HTTP/1.1, HTTP/2, HTTP/3
	SecurityHeaders  *SecurityHeaders `json:"securityHeaders,omitempty"`  // Security headers check results
	RedirectChain    []RedirectHop    `json:"redirectChain,omitempty"`    // Redirect chain details
}

// SecurityHeaders contains results of security header checks.
type SecurityHeaders struct {
	HSTS              *HeaderCheck `json:"hsts,omitempty"`              // Strict-Transport-Security
	CSP               *HeaderCheck `json:"csp,omitempty"`               // Content-Security-Policy
	XFrameOptions     *HeaderCheck `json:"xFrameOptions,omitempty"`     // X-Frame-Options
	XContentType      *HeaderCheck `json:"xContentType,omitempty"`      // X-Content-Type-Options
	XSSProtection     *HeaderCheck `json:"xssProtection,omitempty"`     // X-XSS-Protection
	ReferrerPolicy    *HeaderCheck `json:"referrerPolicy,omitempty"`    // Referrer-Policy
	PermissionsPolicy *HeaderCheck `json:"permissionsPolicy,omitempty"` // Permissions-Policy
	OverallStatus     string       `json:"overallStatus"`               // success, warning, error
	Score             int          `json:"score"`                       // 0-100 security score
}

// HeaderCheck represents the check result for a single security header.
type HeaderCheck struct {
	Present bool   `json:"present"`           // Whether header is present
	Value   string `json:"value,omitempty"`   // Header value if present
	Status  string `json:"status"`            // success, warning, error
	Message string `json:"message,omitempty"` // Recommendation/warning message
}

// RedirectHop represents a single hop in a redirect chain.
type RedirectHop struct {
	URL        string  `json:"url"`
	StatusCode int     `json:"statusCode"`
	LatencyMs  float64 `json:"latencyMs"` // Time taken for this hop
}

// CustomTestsResult represents results from all custom tests.
type CustomTestsResult struct {
	PingResults  []CustomTestResult `json:"pingResults"`
	TCPResults   []CustomTestResult `json:"tcpResults"`
	UDPResults   []CustomTestResult `json:"udpResults"`
	HTTPResults  []CustomTestResult `json:"httpResults"`
	RTSPResults  []CustomTestResult `json:"rtspResults"`  // Issue #778
	DICOMResults []CustomTestResult `json:"dicomResults"` // Issue #777
	HasTests     bool               `json:"hasTests"`

	// Health Checks 100x - Vertical-specific protocols
	MedicalResults    *MedicalCheckResults    `json:"medicalResults,omitempty"`
	EnterpriseResults *EnterpriseCheckResults `json:"enterpriseResults,omitempty"`
	IndustryResults   *IndustryCheckResults   `json:"industryResults,omitempty"`
}

// ============================================================================
// Health Checks Test Handlers
// ============================================================================

// handleHealthChecks runs all configured health checks and returns results.
func (s *Server) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w,
			logger,
			http.StatusMethodNotAllowed,
			ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"),
			"",
		)
		return
	}

	// The probes table is the store of record for health-check targets
	// (ADR-0027 P2). Hydrate the in-memory config snapshot from it so the
	// run*Tests helpers below see the current set. P3 rewires /run onto
	// Engine.RunNow and deletes these helpers along with this hydration.
	s.hydrateHealthCheckTargets(r.Context())

	result := CustomTestsResult{
		PingResults:  s.runPingTests(),
		TCPResults:   s.runTCPTests(r.Context()),
		UDPResults:   s.runUDPTests(),
		HTTPResults:  s.runHTTPTests(r.Context(), logger),
		RTSPResults:  s.runRTSPTests(r.Context()),  // Issue #778
		DICOMResults: s.runDICOMTests(r.Context()), // Issue #777
	}

	// Health Checks 100x - Run vertical-specific protocol checks
	result.MedicalResults = s.RunMedicalChecks(r.Context())
	result.EnterpriseResults = s.RunEnterpriseChecks(r.Context())
	result.IndustryResults = s.RunIndustryChecks(r.Context())

	result.HasTests = len(s.config.HealthChecks.PingTargets) > 0 ||
		len(s.config.HealthChecks.TCPPorts) > 0 ||
		len(s.config.HealthChecks.UDPPorts) > 0 ||
		len(s.config.HealthChecks.HTTPEndpoints) > 0 ||
		len(s.config.HealthChecks.RTSPEndpoints) > 0 ||
		len(s.config.HealthChecks.DICOMEndpoints) > 0 ||
		len(s.config.HealthChecks.HL7Endpoints) > 0 ||
		len(s.config.HealthChecks.FHIREndpoints) > 0 ||
		len(s.config.HealthChecks.LDAPEndpoints) > 0 ||
		len(s.config.HealthChecks.LTIEndpoints) > 0 ||
		len(s.config.HealthChecks.OPCUAEndpoints) > 0 ||
		len(s.config.HealthChecks.ModbusEndpoints) > 0

	sendJSONResponse(w, logger, http.StatusOK, result)
}

// hydrateHealthCheckTargets refreshes the in-memory config's health-check
// endpoint lists from the probes table (the store of record, ADR-0027 P2),
// leaving the performance toggles untouched. Best-effort: a load failure is
// logged and the previous in-memory snapshot is left in place. P3 removes
// this once /run dispatches through Engine.RunNow directly.
func (s *Server) hydrateHealthCheckTargets(ctx context.Context) {
	hc, err := loadHealthCheckEndpoints(ctx, s.db().Probes())
	if err != nil {
		logging.FromContext(ctx).WarnContext(ctx, "Failed to hydrate health-check targets from probes", "error", err)
		return
	}
	s.config.Lock()
	defer s.config.Unlock()
	s.config.HealthChecks.PingTargets = hc.PingTargets
	s.config.HealthChecks.TCPPorts = hc.TCPPorts
	s.config.HealthChecks.UDPPorts = hc.UDPPorts
	s.config.HealthChecks.HTTPEndpoints = hc.HTTPEndpoints
	s.config.HealthChecks.RTSPEndpoints = hc.RTSPEndpoints
	s.config.HealthChecks.DICOMEndpoints = hc.DICOMEndpoints
	s.config.HealthChecks.HL7Endpoints = hc.HL7Endpoints
	s.config.HealthChecks.FHIREndpoints = hc.FHIREndpoints
	s.config.HealthChecks.SQLEndpoints = hc.SQLEndpoints
	s.config.HealthChecks.FileShareEndpoints = hc.FileShareEndpoints
	s.config.HealthChecks.LDAPEndpoints = hc.LDAPEndpoints
	s.config.HealthChecks.LTIEndpoints = hc.LTIEndpoints
	s.config.HealthChecks.OPCUAEndpoints = hc.OPCUAEndpoints
	s.config.HealthChecks.ModbusEndpoints = hc.ModbusEndpoints
}

// getTestStatus returns status based on latency and thresholds.
func getTestStatus(latencyMs float64, warningMs, criticalMs int64) string {
	if latencyMs < float64(warningMs) {
		return statusSuccess
	}
	if latencyMs < float64(criticalMs) {
		return statusWarning
	}
	return statusError
}
