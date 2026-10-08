package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

func getInterfaceStats(s *Server) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/interfaces", http.NoBody)
	w := httptest.NewRecorder()
	s.handleInterfaceStats(w, req)
	return w
}

func TestHandleInterfaceStatsWithoutDatabase(t *testing.T) {
	t.Parallel()
	s := &Server{}
	s.interfaceStats = app.NewInterfaceStats(s.db)

	w := getInterfaceStats(s)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
}

func TestHandleInterfaceStats(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	s := &Server{dbConn: db}
	s.interfaceStats = app.NewInterfaceStats(s.db)
	ctx := context.Background()

	w := getInterfaceStats(s)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"interfaces":[],"count":0}`, w.Body.String())

	target := &polling.Target{
		ClientID: database.DefaultClientID, Name: "edge-sw", IPAddress: "10.0.0.9",
		SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: true, CollectorChain: []string{"if_table"},
	}
	require.NoError(t, db.PollingTargets().Create(ctx, target))
	seen := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	_, err := db.Topology().Upsert(ctx, &topology.Node{
		ID: "node-e", ClientID: database.DefaultClientID, IdentityHash: "h-e", DisplayName: "edge-sw", LastSeen: seen,
	})
	require.NoError(t, err)
	require.NoError(t, db.Topology().UpsertTargetNode(ctx, database.DefaultClientID, target.ID, "node-e", seen))
	for _, idx := range []uint32{1, 2} {
		require.NoError(t, db.Topology().UpsertInterface(ctx, &topology.Interface{
			NodeID: "node-e", IfIndex: idx, IfName: fmt.Sprintf("port%d", idx), IfOperStatus: 1, LastSeen: seen,
		}))
	}
	require.NoError(t, db.Metrics().RecordInterfaceRates(ctx, []ifrate.Rate{
		{ClientID: database.DefaultClientID, TargetID: target.ID, IfIndex: 1, At: seen, InErrors: 0.1},
		{ClientID: database.DefaultClientID, TargetID: target.ID, IfIndex: 2, At: seen, InErrors: 3},
	}))

	w = getInterfaceStats(s)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body InterfaceStatsListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 2, body.Count)
	// The service orders by error rate, so the worse port leads.
	require.Equal(t, "port2", body.Interfaces[0].Name)
	require.Equal(t, "up", string(body.Interfaces[0].OperStatus))
	require.NotNil(t, body.Interfaces[0].Rates)
	require.InDelta(t, 3.0, body.Interfaces[0].Rates.InErrors, 0)
	require.Equal(t, "port1", body.Interfaces[1].Name)
}

func getInterfaceHistory(s *Server, query string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/topology/interfaces/history?"+query, http.NoBody)
	w := httptest.NewRecorder()
	s.handleInterfaceHistory(w, req)
	return w
}

func TestHandleInterfaceHistoryRejectsBadQuery(t *testing.T) {
	t.Parallel()
	s := &Server{dbConn: newTestDB(t)}
	s.interfaceStats = app.NewInterfaceStats(s.db)
	for _, query := range []string{
		"ifIndex=1", "target=t1", "target=t1&ifIndex=-1", "target=t1&ifIndex=4294967296",
		"target=t1&ifIndex=one", "target=t1&ifIndex=1&range=30d", "target=t1&ifIndex=1&range=2h",
	} {
		w := getInterfaceHistory(s, query)
		require.Equal(t, http.StatusBadRequest, w.Code, query)
	}
}

func TestHandleInterfaceHistoryWithoutDatabase(t *testing.T) {
	t.Parallel()
	s := &Server{}
	s.interfaceStats = app.NewInterfaceStats(s.db)

	w := getInterfaceHistory(s, "target=t1&ifIndex=1")
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
}

func TestHandleInterfaceHistory(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	s := &Server{dbConn: db}
	s.interfaceStats = app.NewInterfaceStats(s.db)

	w := getInterfaceHistory(s, "target=t1&ifIndex=2&range=1h")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var empty InterfaceHistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &empty))
	require.NotNil(t, empty.Points, "no rated poll is an empty list, not null")
	require.Empty(t, empty.Points)
	require.InDelta(t, 15.0, empty.BucketSeconds, 0, "1h over 240 buckets")

	recent := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	require.NoError(t, db.Metrics().RecordInterfaceRates(context.Background(), []ifrate.Rate{
		{ClientID: database.DefaultClientID, TargetID: "t1", IfIndex: 2, At: recent.Add(-2 * time.Hour), InErrors: 9},
		{
			ClientID: database.DefaultClientID, TargetID: "t1", IfIndex: 2, At: recent,
			Octets: &ifrate.Octets{In: 125, Out: 25}, OutErrors: 2,
		},
	}))

	// The default range is 24h, which holds both polls.
	w = getInterfaceHistory(s, "target=t1&ifIndex=2")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var day InterfaceHistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &day))
	require.Equal(t, "24h", string(day.Range))
	require.Equal(t, "t1", day.TargetID)
	require.Equal(t, uint32(2), day.IfIndex)
	require.Len(t, day.Points, 2)
	require.InDelta(t, 9.0, day.Points[0].InErrors, 0)

	// The last hour holds only the recent one.
	w = getInterfaceHistory(s, "target=t1&ifIndex=2&range=1h")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var hour InterfaceHistoryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &hour))
	require.Len(t, hour.Points, 1)
	require.True(t, hour.Points[0].SampledAt.Equal(recent))
	require.NotNil(t, hour.Points[0].InOctets)
	require.InDelta(t, 125.0, *hour.Points[0].InOctets, 0)
	require.InDelta(t, 2.0, hour.Points[0].OutErrors, 0)
}
