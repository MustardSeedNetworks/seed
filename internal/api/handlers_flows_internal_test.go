package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/retention"
)

// Each window reads the table that holds it: raw inside the raw horizon,
// hourly rollups past it, daily rollups past the hourly-resolution span.
func TestFlowTierForWindow(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	pro := retention.HorizonsFor(license.TierPro)
	for _, tc := range []struct {
		name      string
		requested time.Duration
		h         retention.TierHorizons
		want      database.FlowTier
	}{
		{"24h is raw", 24 * time.Hour, pro, database.FlowTierRaw},
		{"7d is raw", 7 * 24 * time.Hour, pro, database.FlowTierRaw},
		{"14d is hourly", 14 * 24 * time.Hour, pro, database.FlowTierHourly},
		{"60d is daily", 60 * 24 * time.Hour, pro, database.FlowTierDaily},
		{"Free clamps 30d to raw", 30 * 24 * time.Hour, retention.HorizonsFor(license.TierFree), database.FlowTierRaw},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.want, flowTierFor(resolveHistoryWindow(now, tc.requested, tc.h)))
		})
	}
}

func TestFlowTopRejectsBadParameters(t *testing.T) {
	t.Parallel()

	for _, query := range []string{"by=octets", "limit=0", "limit=101", "limit=ten", "range=yesterday"} {
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			newHistoryServer(t).handleFlowTopTalkers(rec, historyRequest(t, "/api/v1/flows/top-talkers?"+query))
			require.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestFlowTopRefusesARequestWithNoClientClaim(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHistoryServer(t).handleFlowTopConversations(rec,
		httptest.NewRequest(http.MethodGet, "/api/v1/flows/top-conversations", nil))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFlowTopServesTheRankedLists(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	end := time.Now().UTC().Add(-10 * time.Minute)
	rec := func(src, dst string, proto uint8, bytes, pkts uint64) flow.Record {
		return flow.Record{
			Exporter: netip.MustParseAddr("192.0.2.254"), Format: flow.FormatNetFlow9,
			Start: end.Add(-time.Second), End: end,
			SrcAddr: netip.MustParseAddr(src), DstAddr: netip.MustParseAddr(dst),
			Protocol: proto, Bytes: bytes, Packets: pkts,
		}
	}
	require.NoError(t, s.db().FlowRecords().InsertFlows(t.Context(), []flow.Record{
		rec("10.0.0.1", "10.0.0.2", 6, 900, 3),
		rec("10.0.0.2", "10.0.0.1", 6, 100, 7),
		rec("10.0.0.3", "10.0.0.1", 17, 50, 1),
	}))

	w := httptest.NewRecorder()
	s.handleFlowTopTalkers(w, historyRequest(t, "/api/v1/flows/top-talkers?range=1h&by=packets&limit=2"))
	require.Equal(t, http.StatusOK, w.Code)
	var talkers FlowTalkersResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &talkers))
	require.Equal(t, historySourceRaw, talkers.Window.Source)
	require.Equal(t, database.FlowRankPackets, talkers.By)
	require.Equal(t, []database.FlowTalker{
		{Addr: "10.0.0.1", Bytes: 1050, Packets: 11},
		{Addr: "10.0.0.2", Bytes: 1000, Packets: 10},
	}, talkers.Talkers)

	w = httptest.NewRecorder()
	s.handleFlowTopConversations(w, historyRequest(t, "/api/v1/flows/top-conversations"))
	require.Equal(t, http.StatusOK, w.Code)
	var convs FlowConversationsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &convs))
	require.Equal(t, database.FlowRankBytes, convs.By)
	require.Equal(t, []database.FlowConversation{
		{AddrA: "10.0.0.1", AddrB: "10.0.0.2", Protocol: 6, Bytes: 1000, Packets: 10},
		{AddrA: "10.0.0.1", AddrB: "10.0.0.3", Protocol: 17, Bytes: 50, Packets: 1},
	}, convs.Conversations)

	w = httptest.NewRecorder()
	s.handleFlowTopTalkers(w, historyRequest(t, "/api/v1/flows/top-talkers?range=30d"))
	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"talkers":[]`,
		"a 30-day window reads hourly rollups, and none has been written yet")
}
