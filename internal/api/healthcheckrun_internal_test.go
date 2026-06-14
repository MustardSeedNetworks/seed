package api

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

// thresholds returns a CustomThresholds with generous bounds (everything
// passes as success) unless a test overrides a field.
func lenientThresholds() config.CustomThresholds {
	wide := config.Threshold{Warning: time.Second, Critical: 5 * time.Second}
	return config.CustomThresholds{
		Ping: wide, TCP: wide, UDP: wide, HTTP: wide,
		HTTPTimings: config.HTTPTimingThresholds{DNS: wide, TCP: wide, TLS: wide, TTFB: wide},
		CertExpiry:  config.CertExpiryThreshold{Warning: 30, Critical: 7},
	}
}

// TestMapHTTPResult_TimingsAndCert verifies the http mapper lifts the
// per-phase timings and cert summary out of Result.Metadata and derives a
// success testStatus when every phase is within bounds.
func TestMapHTTPResult_TimingsAndCert(t *testing.T) {
	th := lenientThresholds()
	p := probe.Probe{Kind: probe.KindHTTPS, DisplayName: "api", Target: "https://api.example"}
	meta, _ := json.Marshal(map[string]any{
		"status_code": 200,
		"timings_ms":  map[string]float64{"dns": 1, "tcp": 2, "tls": 3, "ttfb": 10},
		"tls": map[string]any{
			"issuer": "Example CA", "not_after": "2027-01-01T00:00:00Z",
			"days_remaining": 200, "tls_version": "TLS 1.3",
		},
	})
	r := probe.Result{Kind: probe.KindHTTPS, Success: true, LatencyMs: 12, Metadata: meta}

	out := mapHTTPResult(p, r, &th)

	require.Equal(t, "api", out.Name)
	require.True(t, out.Success)
	require.Equal(t, 200, out.Status)
	require.InDelta(t, 1, out.DNSLatency, 0.001)
	require.InDelta(t, 10, out.TTFBLatency, 0.001)
	require.Equal(t, statusSuccess, out.DNSStatus)
	require.Equal(t, statusSuccess, out.TestStatus)
	require.Equal(t, 200, out.CertDaysLeft)
	require.Equal(t, "Example CA", out.CertIssuer)
	require.Equal(t, "TLS 1.3", out.TLSVersion)
	require.Equal(t, statusSuccess, out.CertStatus)
}

// TestMapHTTPResult_CertNearExpiryWarns verifies a soon-to-expire cert
// downgrades both certStatus and the overall testStatus to warning.
func TestMapHTTPResult_CertNearExpiryWarns(t *testing.T) {
	th := lenientThresholds()
	p := probe.Probe{Kind: probe.KindHTTPS, DisplayName: "api", Target: "https://api.example"}
	meta, _ := json.Marshal(map[string]any{
		"status_code": 200,
		"timings_ms":  map[string]float64{"ttfb": 5},
		"tls":         map[string]any{"days_remaining": 10, "tls_version": "TLS 1.2"},
	})
	r := probe.Result{Kind: probe.KindHTTPS, Success: true, LatencyMs: 6, Metadata: meta}

	out := mapHTTPResult(p, r, &th)

	require.Equal(t, statusWarning, out.CertStatus, "10 days < 30-day warning threshold")
	require.Equal(t, statusWarning, out.TestStatus, "cert warning elevates the overall status")
}

// TestMapPortResult_DerivesStatusFromThreshold verifies tcp/udp status comes
// from the connect-latency threshold.
func TestMapPortResult_DerivesStatusFromThreshold(t *testing.T) {
	th := config.Threshold{Warning: 10 * time.Millisecond, Critical: 50 * time.Millisecond}
	p := probe.Probe{Kind: probe.KindTCP, DisplayName: "db", Target: "db:5432"}

	warn := mapPortResult(p, probe.Result{Success: true, LatencyMs: 20}, th)
	require.Equal(t, statusWarning, warn.TestStatus)

	fail := mapPortResult(p, probe.Result{Success: false, Error: "refused"}, th)
	require.Equal(t, statusError, fail.TestStatus)
	require.Equal(t, "refused", fail.Error)
}

// TestMapRunResult_GroupsByFamily verifies kinds are filed into the correct
// nested result group matching the card's HealthCheckData shape.
func TestMapRunResult_GroupsByFamily(t *testing.T) {
	th := lenientThresholds()
	var resp HealthCheckRunResponse

	hl7Params, _ := json.Marshal(config.HL7Endpoint{Host: "hl7.example", Port: 2575})
	hl7Meta, _ := json.Marshal(map[string]any{"ack_code": "AA"})
	mapRunResult(&resp,
		probe.Probe{Kind: probe.KindHL7, DisplayName: "lab", Params: json.RawMessage(hl7Params)},
		probe.Result{Kind: probe.KindHL7, Success: true, LatencyMs: 8, Metadata: hl7Meta}, &th)

	sqlParams, _ := json.Marshal(config.SQLEndpoint{Driver: "postgres", Host: "db", Port: 5432})
	mapRunResult(&resp,
		probe.Probe{Kind: probe.KindSQL, DisplayName: "pg", Params: json.RawMessage(sqlParams)},
		probe.Result{Kind: probe.KindSQL, Success: true, LatencyMs: 4}, &th)

	mapRunResult(&resp,
		probe.Probe{Kind: probe.KindPing, DisplayName: "gw", Target: "1.1.1.1"},
		probe.Result{Kind: probe.KindPing, Success: true, LatencyMs: 2}, &th)

	require.Len(t, resp.PingResults, 1)
	require.NotNil(t, resp.MedicalResults)
	require.Len(t, resp.MedicalResults.HL7Results, 1)
	require.Equal(t, "AA", resp.MedicalResults.HL7Results[0].AckCode)
	require.Equal(t, "hl7.example", resp.MedicalResults.HL7Results[0].Host)
	require.NotNil(t, resp.EnterpriseResults)
	require.Len(t, resp.EnterpriseResults.SQLResults, 1)
	require.Equal(t, "postgres", resp.EnterpriseResults.SQLResults[0].Driver)
	require.Nil(t, resp.IndustrialResults, "no industrial probe ran → group omitted")
}
