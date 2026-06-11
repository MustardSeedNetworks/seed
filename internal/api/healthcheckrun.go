package api

// healthcheckrun.go implements the /telemetry/health-checks/run endpoint on
// the probe engine (ADR-0027 P3). It loads the operator's configured
// health-check probes from the probes table, dispatches each through
// Engine.RunNow (sharing the same checker → breach → anomaly path a scheduled
// run uses), and maps the probe Results into the response shape the
// HealthCheckCard renders. The legacy run*Tests()/Run*Checks() parallel stack
// it replaces is deleted.
//
// The response DTO mirrors the frontend's hand-written HealthCheckData
// (ui/src/components/cards/healthCheckCardTypes.ts) field-for-field — the card
// fetches this endpoint and casts the JSON directly, so the Go json tags are
// the wire contract. Only fields the card actually renders are emitted.

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"
	"sync"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/i18n"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/probe"
	"github.com/MustardSeedNetworks/seed/internal/probe/checkers"
)

// maxConcurrentRunProbes bounds how many configured probes dispatch at once
// when /run fans out. The engine itself is not driving this fan-out (each
// RunNow is a synchronous dispatch), so the handler bounds it.
const maxConcurrentRunProbes = 8

// HealthCheckRunResponse is the /run response. It mirrors the frontend
// HealthCheckData interface; field json tags are the wire contract.
type HealthCheckRunResponse struct {
	PingResults []TestResult `json:"pingResults"`
	TCPResults  []TestResult `json:"tcpResults"`
	UDPResults  []TestResult `json:"udpResults"`
	HTTPResults []TestResult `json:"httpResults"`
	HasTests    bool         `json:"hasTests"`

	EnterpriseResults *EnterpriseResults `json:"enterpriseResults,omitempty"`
	VideoResults      *VideoResults      `json:"videoResults,omitempty"`
	MedicalResults    *MedicalResults    `json:"medicalResults,omitempty"`
	EducationResults  *EducationResults  `json:"educationResults,omitempty"`
	IndustrialResults *IndustrialResults `json:"industrialResults,omitempty"`
}

// TestResult is the ping/tcp/udp/http result the card renders. Only the
// rendered fields are present (see the card-consumption audit): host/port/url,
// minLatency/maxLatency, and certCommonName are intentionally omitted.
type TestResult struct {
	Name       string  `json:"name"`
	Success    bool    `json:"success"`
	Latency    float64 `json:"latency"`
	Error      string  `json:"error,omitempty"`
	Status     int     `json:"status,omitempty"`
	TestStatus string  `json:"testStatus,omitempty"`
	// Extended ping (emitted only when the probe supplies them; the probe
	// engine samples once per interval, so these are usually absent — see
	// ADR-0027 P3a non-port note).
	PacketLoss float64 `json:"packetLoss,omitempty"`
	Jitter     float64 `json:"jitter,omitempty"`
	// Per-phase HTTP timings + derived statuses.
	DNSLatency  float64 `json:"dnsLatency,omitempty"`
	TCPConnect  float64 `json:"tcpConnect,omitempty"`
	TLSLatency  float64 `json:"tlsLatency,omitempty"`
	TTFBLatency float64 `json:"ttfbLatency,omitempty"`
	DNSStatus   string  `json:"dnsStatus,omitempty"`
	TCPStatus   string  `json:"tcpStatus,omitempty"`
	TLSStatus   string  `json:"tlsStatus,omitempty"`
	TTFBStatus  string  `json:"ttfbStatus,omitempty"`
	// Certificate summary (https).
	CertDaysLeft int    `json:"certDaysLeft,omitempty"`
	CertStatus   string `json:"certStatus,omitempty"`
	CertExpiry   string `json:"certExpiry,omitempty"`
	CertIssuer   string `json:"certIssuer,omitempty"`
	TLSVersion   string `json:"tlsVersion,omitempty"`
}

// EnterpriseResults groups SQL/FileShare/LDAP results.
type EnterpriseResults struct {
	SQLResults       []SQLTestResult       `json:"sqlResults,omitempty"`
	FileShareResults []FileShareTestResult `json:"fileShareResults,omitempty"`
	LDAPResults      []LDAPTestResult      `json:"ldapResults,omitempty"`
}

// VideoResults groups RTSP results.
type VideoResults struct {
	RTSPResults []RTSPTestResult `json:"rtspResults,omitempty"`
}

// MedicalResults groups DICOM/HL7/FHIR results.
type MedicalResults struct {
	DICOMResults []DICOMTestResult `json:"dicomResults,omitempty"`
	HL7Results   []HL7TestResult   `json:"hl7Results,omitempty"`
	FHIRResults  []FHIRTestResult  `json:"fhirResults,omitempty"`
}

// EducationResults groups LTI results.
type EducationResults struct {
	LTIResults []LTITestResult `json:"ltiResults,omitempty"`
}

// IndustrialResults groups OPC-UA/Modbus results.
type IndustrialResults struct {
	OPCUAResults  []OPCUATestResult  `json:"opcuaResults,omitempty"`
	ModbusResults []ModbusTestResult `json:"modbusResults,omitempty"`
}

// SQLTestResult mirrors the card's SqlTestResult (rendered fields only).
type SQLTestResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	Driver        string  `json:"driver"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	ServerVersion string  `json:"serverVersion,omitempty"`
	Error         string  `json:"error,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	ConnectTimeMs float64 `json:"connectTimeMs,omitempty"`
	QueryTimeMs   float64 `json:"queryTimeMs,omitempty"`
}

// FileShareTestResult mirrors the card's FileShareTestResult.
type FileShareTestResult struct {
	Name           string  `json:"name"`
	Success        bool    `json:"success"`
	Protocol       string  `json:"protocol"`
	Host           string  `json:"host"`
	Share          string  `json:"share"`
	Error          string  `json:"error,omitempty"`
	ConnectTimeMs  float64 `json:"connectTimeMs"`
	ReadSpeedMbps  float64 `json:"readSpeedMbps,omitempty"`
	WriteSpeedMbps float64 `json:"writeSpeedMbps,omitempty"`
}

// LDAPTestResult mirrors the card's LdapTestResult.
type LDAPTestResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	UseTLS        bool    `json:"useTls"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	ServerInfo    string  `json:"serverInfo,omitempty"`
	Error         string  `json:"error,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	ConnectTimeMs float64 `json:"connectTimeMs,omitempty"`
	BindTimeMs    float64 `json:"bindTimeMs,omitempty"`
}

// RTSPTestResult mirrors the card's RtspTestResult.
type RTSPTestResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	URL           string  `json:"url"`
	Codec         string  `json:"codec,omitempty"`
	Resolution    string  `json:"resolution,omitempty"`
	Error         string  `json:"error,omitempty"`
	ConnectTimeMs float64 `json:"connectTimeMs"`
}

// DICOMTestResult mirrors the card's DicomTestResult.
type DICOMTestResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	AETitle       string  `json:"aeTitle"`
	ServerAETitle string  `json:"serverAeTitle,omitempty"`
	Error         string  `json:"error,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	EchoTimeMs    float64 `json:"echoTimeMs,omitempty"`
}

// HL7TestResult mirrors the card's Hl7TestResult.
type HL7TestResult struct {
	Name           string  `json:"name"`
	Success        bool    `json:"success"`
	Host           string  `json:"host"`
	Port           int     `json:"port"`
	AckCode        string  `json:"ackCode,omitempty"`
	ServerVersion  string  `json:"serverVersion,omitempty"`
	Error          string  `json:"error,omitempty"`
	TotalTimeMs    float64 `json:"totalTimeMs"`
	ResponseTimeMs float64 `json:"responseTimeMs,omitempty"`
}

// FHIRTestResult mirrors the card's FhirTestResult.
type FHIRTestResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	BaseURL       string  `json:"baseUrl"`
	FHIRVersion   string  `json:"fhirVersion,omitempty"`
	ServerName    string  `json:"serverName,omitempty"`
	Error         string  `json:"error,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	ResourceCount int     `json:"resourceCount,omitempty"`
}

// LTITestResult mirrors the card's LtiTestResult.
type LTITestResult struct {
	Name        string  `json:"name"`
	Success     bool    `json:"success"`
	LaunchURL   string  `json:"launchUrl"`
	LTIVersion  string  `json:"ltiVersion,omitempty"`
	Error       string  `json:"error,omitempty"`
	TotalTimeMs float64 `json:"totalTimeMs"`
}

// OPCUATestResult mirrors the card's OpcuaTestResult.
type OPCUATestResult struct {
	Name         string  `json:"name"`
	Success      bool    `json:"success"`
	EndpointURL  string  `json:"endpointUrl"`
	SecurityMode string  `json:"securityMode,omitempty"`
	ProductName  string  `json:"productName,omitempty"`
	ServerState  string  `json:"serverState,omitempty"`
	Error        string  `json:"error,omitempty"`
	TotalTimeMs  float64 `json:"totalTimeMs"`
}

// ModbusTestResult mirrors the card's ModbusTestResult.
type ModbusTestResult struct {
	Name          string  `json:"name"`
	Success       bool    `json:"success"`
	Host          string  `json:"host"`
	Port          int     `json:"port"`
	UnitID        int     `json:"unitId"`
	Error         string  `json:"error,omitempty"`
	TotalTimeMs   float64 `json:"totalTimeMs"`
	RegisterValue int     `json:"registerValue,omitempty"`
}

// runProbeResult pairs a dispatched probe with its Result for mapping.
type runProbeResult struct {
	probe  *database.Probe
	result probe.Result
}

// handleHealthChecks runs every configured health-check probe now and returns
// the results in the card's shape. The probes table is the source of record
// (ADR-0027 P2); dispatch goes through Engine.RunNow so an on-demand run
// raises and clears the same anomalies a scheduled run would (ADR-0025 §1).
func (s *Server) handleHealthChecks(w http.ResponseWriter, r *http.Request) {
	logger := logging.FromContext(r.Context())
	localizer := i18n.FromRequest(r)

	if r.Method != http.MethodGet {
		sendErrorResponseWithDetails(
			w, logger, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed,
			localizer.T("errors.api.methodNotAllowed"), "",
		)
		return
	}

	probes, err := s.healthCheckProbes(r.Context())
	if err != nil {
		logger.ErrorContext(r.Context(), "Failed to load health-check probes", "error", err)
		sendErrorResponseWithDetails(
			w, logger, http.StatusInternalServerError, ErrCodeInternal,
			localizer.T("errors.settings.loadFailed"), "",
		)
		return
	}

	dispatched := s.dispatchProbes(r.Context(), probes)

	var resp HealthCheckRunResponse
	resp.HasTests = len(probes) > 0
	thresholds := &s.config.Thresholds.CustomTests
	for _, d := range dispatched {
		mapRunResult(&resp, d.probe, d.result, thresholds)
	}
	sendJSONResponse(w, logger, http.StatusOK, resp)
}

// healthCheckProbes lists the operator's configured probes of the health-check
// kinds (ping/tcp/udp/http/rtsp/dicom + the eight verticals), in a stable
// display order.
func (s *Server) healthCheckProbes(ctx context.Context) ([]*database.Probe, error) {
	all, err := s.db().Probes().ListProbes(ctx, database.DefaultClientID, "")
	if err != nil {
		return nil, err
	}
	out := make([]*database.Probe, 0, len(all))
	for _, p := range all {
		if isHealthCheckKind(p.Kind) {
			out = append(out, p)
		}
	}
	return out, nil
}

// isHealthCheckKind reports whether kind is one of the fourteen kinds the
// health-check surface owns.
func isHealthCheckKind(kind string) bool {
	return slices.Contains(healthCheckKinds(), kind)
}

// dispatchProbes runs each probe via Engine.RunNow with bounded concurrency
// and returns the (probe, result) pairs in the input order. A probe whose
// dispatch errors (e.g. storage hiccup) still yields its error as a failed
// Result so the card shows it rather than silently dropping the row.
func (s *Server) dispatchProbes(ctx context.Context, probes []*database.Probe) []runProbeResult {
	out := make([]runProbeResult, len(probes))
	sem := make(chan struct{}, maxConcurrentRunProbes)
	var wg sync.WaitGroup
	for i, p := range probes {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, pr *database.Probe) {
			defer wg.Done()
			defer func() { <-sem }()
			res, err := s.probeEngine.RunNow(ctx, pr.ID)
			if err != nil {
				res = probe.Result{Kind: pr.Kind, Success: false, Error: err.Error()}
			}
			out[idx] = runProbeResult{probe: pr, result: res}
		}(i, p)
	}
	wg.Wait()
	return out
}

// mapRunResult dispatches a single (probe, result) into the right slot of the
// response by kind. The per-kind mappers live in healthcheckrunmappers.go.
func mapRunResult(resp *HealthCheckRunResponse, p *database.Probe, r probe.Result, th *config.CustomThresholds) {
	switch p.Kind {
	case probe.KindPing:
		resp.PingResults = append(resp.PingResults, mapPingResult(p, r))
	case probe.KindTCP:
		resp.TCPResults = append(resp.TCPResults, mapPortResult(p, r, th.TCP))
	case probe.KindUDP:
		resp.UDPResults = append(resp.UDPResults, mapPortResult(p, r, th.UDP))
	case probe.KindHTTP, probe.KindHTTPS:
		resp.HTTPResults = append(resp.HTTPResults, mapHTTPResult(p, r, th))
	case probe.KindRTSP:
		resp.video().RTSPResults = append(resp.video().RTSPResults, mapRTSPResult(p, r))
	case probe.KindDICOM:
		resp.medical().DICOMResults = append(resp.medical().DICOMResults, mapDICOMResult(p, r))
	case probe.KindHL7:
		resp.medical().HL7Results = append(resp.medical().HL7Results, mapHL7Result(p, r))
	case probe.KindFHIR:
		resp.medical().FHIRResults = append(resp.medical().FHIRResults, mapFHIRResult(p, r))
	case probe.KindSQL:
		resp.enterprise().SQLResults = append(resp.enterprise().SQLResults, mapSQLResult(p, r))
	case probe.KindFileShare:
		resp.enterprise().FileShareResults = append(resp.enterprise().FileShareResults, mapFileShareResult(p, r))
	case probe.KindLDAP:
		resp.enterprise().LDAPResults = append(resp.enterprise().LDAPResults, mapLDAPResult(p, r))
	case probe.KindLTI:
		resp.education().LTIResults = append(resp.education().LTIResults, mapLTIResult(p, r))
	case probe.KindOPCUA:
		resp.industrial().OPCUAResults = append(resp.industrial().OPCUAResults, mapOPCUAResult(p, r))
	case probe.KindMODBUS:
		resp.industrial().ModbusResults = append(resp.industrial().ModbusResults, mapModbusResult(p, r))
	}
}

// The grouping accessors lazily allocate the optional sub-structs so they
// serialize as present only when at least one result of that family ran.
func (resp *HealthCheckRunResponse) enterprise() *EnterpriseResults {
	if resp.EnterpriseResults == nil {
		resp.EnterpriseResults = &EnterpriseResults{}
	}
	return resp.EnterpriseResults
}

func (resp *HealthCheckRunResponse) video() *VideoResults {
	if resp.VideoResults == nil {
		resp.VideoResults = &VideoResults{}
	}
	return resp.VideoResults
}

func (resp *HealthCheckRunResponse) medical() *MedicalResults {
	if resp.MedicalResults == nil {
		resp.MedicalResults = &MedicalResults{}
	}
	return resp.MedicalResults
}

func (resp *HealthCheckRunResponse) education() *EducationResults {
	if resp.EducationResults == nil {
		resp.EducationResults = &EducationResults{}
	}
	return resp.EducationResults
}

func (resp *HealthCheckRunResponse) industrial() *IndustrialResults {
	if resp.IndustrialResults == nil {
		resp.IndustrialResults = &IndustrialResults{}
	}
	return resp.IndustrialResults
}

// mapPingResult maps a ping probe Result. Ping reports reachability + dial
// latency; extended loss/jitter stats are not produced by the scheduled
// probe (ADR-0027 P3a non-port), so those fields stay zero/absent.
func mapPingResult(p *database.Probe, r probe.Result) TestResult {
	return baseTestResult(p, r)
}

// mapPortResult maps a tcp/udp probe Result, deriving testStatus from the
// connect-latency threshold for that kind.
func mapPortResult(p *database.Probe, r probe.Result, th config.Threshold) TestResult {
	out := baseTestResult(p, r)
	if r.Success {
		out.TestStatus = getTestStatus(r.LatencyMs, th.Warning.Milliseconds(), th.Critical.Milliseconds())
	}
	return out
}

// mapHTTPResult maps an http/https probe Result: status code, per-phase
// timing breakdown + derived statuses, and (for https) the cert summary.
func mapHTTPResult(p *database.Probe, r probe.Result, th *config.CustomThresholds) TestResult {
	out := baseTestResult(p, r)
	var meta checkers.HTTPRunMetadata
	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &meta)
	}
	out.Status = meta.StatusCode

	out.DNSLatency = meta.Timings.DNS
	out.TCPConnect = meta.Timings.TCP
	out.TLSLatency = meta.Timings.TLS
	out.TTFBLatency = meta.Timings.TTFB
	if r.Success {
		applyHTTPTimingStatuses(&out, th)
	}

	if meta.TLS != nil {
		out.CertDaysLeft = meta.TLS.DaysRemaining
		out.CertExpiry = meta.TLS.NotAfter
		out.CertIssuer = meta.TLS.Issuer
		out.TLSVersion = meta.TLS.TLSVersion
		out.CertStatus = certStatusFromDays(meta.TLS.DaysRemaining, th.CertExpiry)
		if out.CertStatus == statusError && out.TestStatus != statusError {
			out.TestStatus = statusError
		} else if out.CertStatus == statusWarning && out.TestStatus == statusSuccess {
			out.TestStatus = statusWarning
		}
	}
	return out
}

// applyHTTPTimingStatuses sets the per-phase statuses and rolls them up into
// testStatus, mirroring the legacy evaluateHTTPTimings: any phase error/warn
// dominates, else the overall HTTP latency threshold decides.
func applyHTTPTimingStatuses(out *TestResult, th *config.CustomThresholds) {
	t := th.HTTPTimings
	out.DNSStatus = getTestStatus(out.DNSLatency, t.DNS.Warning.Milliseconds(), t.DNS.Critical.Milliseconds())
	out.TCPStatus = getTestStatus(out.TCPConnect, t.TCP.Warning.Milliseconds(), t.TCP.Critical.Milliseconds())
	out.TLSStatus = getTestStatus(out.TLSLatency, t.TLS.Warning.Milliseconds(), t.TLS.Critical.Milliseconds())
	out.TTFBStatus = getTestStatus(out.TTFBLatency, t.TTFB.Warning.Milliseconds(), t.TTFB.Critical.Milliseconds())
	switch {
	case out.DNSStatus == statusError || out.TCPStatus == statusError ||
		out.TLSStatus == statusError || out.TTFBStatus == statusError:
		out.TestStatus = statusError
	case out.DNSStatus == statusWarning || out.TCPStatus == statusWarning ||
		out.TLSStatus == statusWarning || out.TTFBStatus == statusWarning:
		out.TestStatus = statusWarning
	default:
		out.TestStatus = getTestStatus(out.Latency, th.HTTP.Warning.Milliseconds(), th.HTTP.Critical.Milliseconds())
	}
}

// certStatusFromDays maps days-until-expiry to a status using the cert-expiry
// thresholds (warning/critical in days; fewer days is worse).
func certStatusFromDays(days int, th config.CertExpiryThreshold) string {
	switch {
	case days <= th.Critical:
		return statusError
	case days <= th.Warning:
		return statusWarning
	default:
		return statusSuccess
	}
}

// baseTestResult fills the fields common to every kind: the display name,
// success, latency, and error.
func baseTestResult(p *database.Probe, r probe.Result) TestResult {
	out := TestResult{
		Name:    p.DisplayName,
		Success: r.Success,
		Latency: r.LatencyMs,
		Error:   r.Error,
	}
	if r.Success {
		out.TestStatus = statusSuccess
	} else {
		out.TestStatus = statusError
	}
	return out
}
