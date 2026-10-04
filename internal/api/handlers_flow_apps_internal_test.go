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

	"github.com/MustardSeedNetworks/seed/internal/appid"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

func appSignaturesRequest(method, body string) *http.Request {
	return httptest.NewRequest(method, "/api/v1/flows/application-signatures", strings.NewReader(body))
}

func serveAppSignatures(t *testing.T, s *Server, req *http.Request) (int, AppSignaturesResponse) {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleAppSignatures(w, req)
	var resp AppSignaturesResponse
	if w.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	}
	return w.Code, resp
}

func TestFlowTopApplicationsServesTheRankedList(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	end := time.Now().UTC().Add(-10 * time.Minute)
	rec := func(proto uint8, sport, dport uint16, bytes, pkts uint64) flow.Record {
		return flow.Record{
			Exporter: netip.MustParseAddr("192.0.2.254"), Format: flow.FormatIPFIX,
			Start: end.Add(-time.Second), End: end,
			SrcAddr: netip.MustParseAddr("10.0.0.1"), DstAddr: netip.MustParseAddr("10.0.0.2"),
			SrcPort: sport, DstPort: dport, Protocol: proto, Bytes: bytes, Packets: pkts,
		}
	}
	require.NoError(t, s.db().FlowRecords().InsertFlows(t.Context(), []flow.Record{
		rec(6, 50000, 443, 900, 3),
		rec(17, 50001, 53, 100, 7),
		rec(6, 50002, 9999, 50, 1),
	}))

	w := httptest.NewRecorder()
	s.handleFlowTopApplications(w, historyRequest(t, "/api/v1/flows/top-applications?range=1h&by=packets"))
	require.Equal(t, http.StatusOK, w.Code)
	var resp FlowApplicationsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, historySourceRaw, resp.Window.Source)
	require.Equal(t, database.FlowRankPackets, resp.By)
	require.Equal(t, []database.FlowApplication{
		{Name: "dns", Bytes: 100, Packets: 7},
		{Name: "https", Bytes: 900, Packets: 3},
		{Name: appid.Unknown, Bytes: 50, Packets: 1},
	}, resp.Applications)

	w = httptest.NewRecorder()
	s.handleFlowTopApplications(w, historyRequest(t, "/api/v1/flows/top-applications?limit=0"))
	require.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAppSignaturesEditAndReset(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	code, resp := serveAppSignatures(t, s, appSignaturesRequest(http.MethodGet, ""))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, appSignaturesBuiltin, resp.Source)
	require.Equal(t, appid.Builtin().Signatures(), resp.Signatures)

	code, resp = serveAppSignatures(t, s, appSignaturesRequest(http.MethodPut,
		`{"signatures":[{"name":"historian","description":"Plant historian","protocol":6,"ports":["9999"]}]}`))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, appSignaturesCustom, resp.Source)
	require.Equal(t, []appid.Signature{
		{Name: "historian", Description: "Plant historian", Protocol: 6, Ports: []string{"9999"}},
	}, resp.Signatures)
	sigs, err := s.db().FlowRecords().AppSignatures(t.Context())
	require.NoError(t, err)
	require.Equal(t, "historian", sigs.Table.Classify(6, 50000, 9999))

	code, resp = serveAppSignatures(t, s, appSignaturesRequest(http.MethodDelete, ""))
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, appSignaturesBuiltin, resp.Source)
}

func TestAppSignaturesRejectsAnInvalidTable(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"empty table":       `{"signatures":[]}`,
		"unknown field":     `{"signatures":[{"name":"web","protocol":6,"port":["80"]}]}`,
		"claimed twice":     `{"signatures":[{"name":"a","protocol":6,"ports":["80"]},{"name":"b","protocol":6,"ports":["80"]}]}`,
		"not json":          `signatures`,
		"reserved name":     `{"signatures":[{"name":"unknown","protocol":6,"ports":["80"]}]}`,
		"port out of range": `{"signatures":[{"name":"web","protocol":6,"ports":["70000"]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			s := newHistoryServer(t)
			code, _ := serveAppSignatures(t, s, appSignaturesRequest(http.MethodPut, body))
			require.Equal(t, http.StatusBadRequest, code)

			sigs, err := s.db().FlowRecords().AppSignatures(t.Context())
			require.NoError(t, err)
			require.False(t, sigs.Custom, "a rejected table must not be stored")
		})
	}
}
