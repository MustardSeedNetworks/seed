package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/speedtest"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

// The flow reads and the speed-test record pinned at the wire and at the row,
// so moving their queries behind internal/app (D-SEED-30, #2750) cannot
// change what a client receives or what a report later reads.

// flowWireBody serves one request and returns its body without the window,
// whose bounds move with the clock.
func flowWireBody(t *testing.T, handler http.HandlerFunc, req *http.Request) string {
	t.Helper()
	w := httptest.NewRecorder()
	handler(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	delete(body, "window")
	out, err := json.Marshal(body)
	require.NoError(t, err)
	return string(out)
}

func TestFlowReadsWireShape(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	end := time.Now().UTC().Add(-10 * time.Minute)
	rec := func(src, dst string, proto uint8, dport uint16, bytes, pkts uint64) flow.Record {
		return flow.Record{
			Exporter: netip.MustParseAddr("192.0.2.254"), Format: flow.FormatIPFIX,
			Start: end.Add(-time.Second), End: end,
			SrcAddr: netip.MustParseAddr(src), DstAddr: netip.MustParseAddr(dst),
			SrcPort: 50000, DstPort: dport, Protocol: proto, Bytes: bytes, Packets: pkts,
		}
	}
	require.NoError(t, s.db().FlowRecords().InsertFlows(t.Context(), []flow.Record{
		rec("10.0.0.1", "10.0.0.2", 6, 443, 900, 3),
		rec("10.0.0.3", "10.0.0.1", 17, 53, 50, 1),
	}))

	require.JSONEq(t,
		`{"by":"bytes","talkers":[`+
			`{"addr":"10.0.0.1","bytes":950,"packets":4},`+
			`{"addr":"10.0.0.2","bytes":900,"packets":3},`+
			`{"addr":"10.0.0.3","bytes":50,"packets":1}]}`,
		flowWireBody(t, s.handleFlowTopTalkers, historyRequest(t, "/api/v1/flows/top-talkers?range=1h")))
	require.JSONEq(t,
		`{"by":"packets","conversations":[`+
			`{"addrA":"10.0.0.1","addrB":"10.0.0.2","protocol":6,"bytes":900,"packets":3}]}`,
		flowWireBody(t, s.handleFlowTopConversations,
			historyRequest(t, "/api/v1/flows/top-conversations?by=packets&limit=1")))
	require.JSONEq(t,
		`{"by":"bytes","applications":[`+
			`{"name":"https","bytes":900,"packets":3},`+
			`{"name":"dns","bytes":50,"packets":1}]}`,
		flowWireBody(t, s.handleFlowTopApplications, historyRequest(t, "/api/v1/flows/top-applications")))
}

func TestFlowSettingsWireShape(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	put := func(path, body string) *http.Request {
		return httptest.NewRequest(http.MethodPut, path, strings.NewReader(body))
	}

	require.JSONEq(t,
		`{"source":"custom","signatures":[{"name":"erp","description":"ERP","protocol":6,"ports":["8800"]}]}`,
		flowWireBody(t, s.handleAppSignatures, put("/api/v1/flows/application-signatures",
			`{"signatures":[{"name":"erp","description":"ERP","protocol":6,"ports":["8800"]}]}`)))
	reset := flowWireBody(t, s.handleAppSignatures,
		httptest.NewRequest(http.MethodDelete, "/api/v1/flows/application-signatures", nil))
	require.Contains(t, reset, `"source":"builtin"`)

	require.JSONEq(t, `{"indicators":["203.0.113.0/24"]}`,
		flowWireBody(t, s.handleFlowIndicators, put("/api/v1/flows/threat-indicators",
			`{"indicators":["203.0.113.0/24"]}`)))
	require.JSONEq(t, `{"indicators":["203.0.113.0/24"]}`,
		flowWireBody(t, s.handleFlowIndicators,
			httptest.NewRequest(http.MethodGet, "/api/v1/flows/threat-indicators", nil)))
}

// A finished speed test is kept as one speedtest_results row, the source of
// the reports' bandwidth figures; with no database it is dropped quietly.
func TestRecordSpeedtestStoresTheResult(t *testing.T) {
	t.Parallel()

	(&Server{}).recordSpeedtest(t.Context(), &speedtest.Result{Download: 1})

	s := newHistoryServer(t)
	at := time.Date(2026, 10, 7, 9, 30, 0, 0, time.FixedZone("CEST", 2*60*60))
	s.recordSpeedtest(t.Context(), &speedtest.Result{
		Download: 94.5, Upload: 21.25, Latency: 7.5,
		Server: "s1", Location: "Oslo", Host: "s1.example.net", Timestamp: at,
	})

	var (
		iface, server, location, ts string
		down, up, latency           float64
	)
	require.NoError(t, s.db().QueryRow(t.Context(),
		`SELECT interface_name, server_name, server_location, download_mbps, upload_mbps,
			latency_ms, timestamp FROM speedtest_results`).
		Scan(&iface, &server, &location, &down, &up, &latency, &ts))
	require.Empty(t, iface, "no network manager, so no interface")
	require.Equal(t, "s1", server)
	require.Equal(t, "Oslo", location)
	require.InDelta(t, 94.5, down, 0)
	require.InDelta(t, 21.25, up, 0)
	require.InDelta(t, 7.5, latency, 0)
	require.Equal(t, "2026-10-07T07:30:00Z", ts)
}
