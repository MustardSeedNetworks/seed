package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

func getIPSLA(path string, handler http.HandlerFunc) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, APIVersionPrefix+path, http.NoBody)
	w := httptest.NewRecorder()
	handler(w, req)
	return w
}

func TestHandleIPSLAWithoutDatabase(t *testing.T) {
	t.Parallel()
	s := &Server{}
	s.ipSLA = app.NewIPSLA(s.db)

	w := getIPSLA("/ip-sla/operations", s.handleIPSLAOperations)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	w = getIPSLA("/ip-sla/operations/history?target=t1&index=10", s.handleIPSLAHistory)
	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
}

func storeIPSLAObservation(t *testing.T, db *database.DB, targetID string, at time.Time, ops ...ipsla.Operation) {
	t.Helper()
	payload, err := json.Marshal(ipsla.Observation{
		ClientID: database.DefaultClientID, TargetID: targetID, ObservedAt: at, Operations: ops,
	})
	require.NoError(t, err)
	require.NoError(t, db.SNMPObservations().Insert(context.Background(), &observation.SNMPObservation{
		ClientID: database.DefaultClientID, TargetID: targetID, Kind: ipsla.Name,
		ObservedAt: at, PayloadJSON: string(payload),
	}))
}

func TestHandleIPSLAOperations(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	s := &Server{dbConn: db}
	s.ipSLA = app.NewIPSLA(s.db)
	ctx := context.Background()

	w := getIPSLA("/ip-sla/operations", s.handleIPSLAOperations)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"operations":[],"count":0}`, w.Body.String())

	target := &polling.Target{
		ClientID: database.DefaultClientID, Name: "wan-rtr", IPAddress: "10.0.0.1",
		SNMPVersion: "v2c", PollIntervalSec: 60, Enabled: true, CollectorChain: []string{ipsla.Name},
	}
	require.NoError(t, db.PollingTargets().Create(ctx, target))
	seen := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	healthy := ipsla.Operation{
		Index: 10, Tag: "wan-gw", RttType: 1, Target: "192.0.2.1", ThresholdMs: 100, FrequencyS: 60, Status: 1,
		Latest: &ipsla.Result{Sense: ipsla.SenseOK, RTTMs: 7, RunUptime: 100},
	}
	// The older poll still saw operation 30 healthy; only the latest counts.
	storeIPSLAObservation(t, db, target.ID, seen.Add(-time.Minute), healthy,
		ipsla.Operation{Index: 30, Status: 1, Latest: &ipsla.Result{Sense: ipsla.SenseOK, RTTMs: 3, RunUptime: 50}})
	storeIPSLAObservation(t, db, target.ID, seen, healthy,
		ipsla.Operation{
			Index: 20, Tag: "voice-dc", RttType: 9, Target: "192.0.2.2", ThresholdMs: 300, Status: 1,
			Latest: &ipsla.Result{
				Sense: ipsla.SenseOK, RTTMs: 13, RunUptime: 100,
				Jitter: &ipsla.Jitter{AvgJitterMs: 3, Received: 95, Lost: 5, MOS: 4.34},
			},
		},
		ipsla.Operation{
			Index: 30, Tag: "dead-site", RttType: 1, Status: 1,
			Latest: &ipsla.Result{Sense: ipsla.SenseTimeout, RunUptime: 100},
		},
		ipsla.Operation{Index: 40, RttType: 6, Status: 2})
	// A target that no longer exists is not listed.
	storeIPSLAObservation(t, db, "deleted-target", seen, healthy)

	w = getIPSLA("/ip-sla/operations", s.handleIPSLAOperations)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var body IPSLAOperationsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 4, body.Count)

	var order []int
	for _, op := range body.Operations {
		require.Equal(t, target.ID, op.TargetID)
		require.Equal(t, "wan-rtr", op.TargetName)
		require.True(t, op.ObservedAt.Equal(seen))
		order = append(order, op.Index)
	}
	// The failed operation leads, the one never run trails.
	require.Equal(t, []int{30, 10, 20, 40}, order)
	require.False(t, body.Operations[0].Latest.Completed)
	require.Equal(t, ipsla.SenseTimeout, body.Operations[0].Latest.Sense)
	voice := body.Operations[2]
	require.Equal(t, "voice-dc", voice.Tag)
	require.True(t, voice.Latest.Completed)
	require.Equal(t, &IPSLAJitterResponse{AvgJitterMs: 3, Received: 95, Lost: 5, MOS: 4.34}, voice.Latest.Jitter)
	require.Nil(t, body.Operations[3].Latest)
}

func TestHandleIPSLAHistoryRejectsBadQuery(t *testing.T) {
	t.Parallel()
	s := &Server{dbConn: newTestDB(t)}
	s.ipSLA = app.NewIPSLA(s.db)
	for _, query := range []string{
		"index=10", "target=t1", "target=t1&index=0", "target=t1&index=-1",
		"target=t1&index=ten", "target=t1&index=10&range=30d",
	} {
		w := getIPSLA("/ip-sla/operations/history?"+query, s.handleIPSLAHistory)
		require.Equal(t, http.StatusBadRequest, w.Code, query)
	}
}

func TestHandleIPSLAHistory(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	s := &Server{dbConn: db}
	s.ipSLA = app.NewIPSLA(s.db)
	history := func(query string) IPSLAHistoryResponse {
		t.Helper()
		w := getIPSLA("/ip-sla/operations/history?"+query, s.handleIPSLAHistory)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var body IPSLAHistoryResponse
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
		return body
	}

	empty := history("target=t1&index=10&range=1h")
	require.NotNil(t, empty.Points, "no run is an empty list, not null")
	require.Empty(t, empty.Points)

	recent := time.Now().UTC().Add(-10 * time.Minute).Truncate(time.Second)
	tracker := sla.NewTracker()
	run := func(at time.Time, uptime uint32, latest ipsla.Result) {
		latest.RunUptime = uptime
		require.NoError(t, db.Metrics().RecordSLAResults(context.Background(), tracker.Observe(ipsla.Observation{
			ClientID: database.DefaultClientID, TargetID: "t1", ObservedAt: at,
			Operations: []ipsla.Operation{{Index: 10, Latest: &latest}},
		})))
	}
	run(recent.Add(-2*time.Hour), 1, ipsla.Result{Sense: ipsla.SenseTimeout})
	run(recent, 2, ipsla.Result{
		Sense: ipsla.SenseOK, RTTMs: 13,
		Jitter: &ipsla.Jitter{AvgJitterMs: 3, Received: 95, Lost: 5, MOS: 4.34},
	})

	day := history("target=t1&index=10")
	require.Equal(t, "24h", string(day.Range))
	require.Equal(t, "t1", day.TargetID)
	require.Equal(t, 10, day.Index)
	require.Len(t, day.Points, 2)
	require.InDelta(t, 0.0, day.Points[0].CompletedPct, 0)
	require.Nil(t, day.Points[0].RTTMs, "a failed run measured no RTT")

	hour := history("target=t1&index=10&range=1h")
	require.Len(t, hour.Points, 1)
	p := hour.Points[0]
	require.True(t, p.SampledAt.Equal(recent))
	require.Equal(t, 1, p.Runs)
	require.InDelta(t, 100.0, p.CompletedPct, 0)
	require.InDelta(t, 13.0, *p.RTTMs, 0)
	require.InDelta(t, 3.0, *p.JitterMs, 0)
	require.InDelta(t, 5.0, *p.LossPct, 1e-9)
	require.InDelta(t, 4.34, *p.MOS, 1e-9)

	// Another operation's runs are not this one's.
	require.Empty(t, history("target=t1&index=11").Points)
}
