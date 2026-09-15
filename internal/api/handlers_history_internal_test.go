package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

func newHistoryServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{}
	s.dbConn = newTestDB(t)
	return s
}

func historyRequest(t *testing.T, path string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	return req.WithContext(auth.WithClientID(req.Context(), database.DefaultClientID))
}

// A request that cannot be attributed to a tenant must not be answered with
// the default tenant's history.
func TestProbeHistoryRefusesARequestWithNoClientClaim(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHistoryServer(t).handleProbeHistory(rec,
		httptest.NewRequest(http.MethodGet, "/api/v1/history/probes/p1", nil))

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestProbeHistoryRejectsAnUnusableRange(t *testing.T) {
	t.Parallel()

	for _, raw := range []string{"yesterday", "-24h", "0h", "3000d", "7 d", "d", "1w"} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			rec := httptest.NewRecorder()
			newHistoryServer(t).handleProbeHistory(rec,
				historyRequest(t, "/api/v1/history/probes/p1?range="+url.QueryEscape(raw)))

			require.Equal(t, http.StatusBadRequest, rec.Code, "range=%q", raw)
		})
	}
}

// The probe id comes off the path; an empty one is a bad request rather than a
// read of every probe.
func TestProbeHistoryRequiresAProbeID(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHistoryServer(t).handleProbeHistory(rec, historyRequest(t, "/api/v1/history/probes/"))

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

// The window is the contract the UI half draws against: it must say which
// resolution it served, whether the last bucket is still filling, and — when
// the tier retains less than was asked for — that it was clamped.
func TestProbeHistoryReportsTheServedWindow(t *testing.T) {
	t.Parallel()

	s := newHistoryServer(t)
	now := time.Now().UTC()
	require.NoError(t, s.db().Probes().CreateProbe(t.Context(), &database.Probe{
		ID: "p-window", ClientID: database.DefaultClientID,
		Kind: "ping", Target: "10.0.0.1", Enabled: true,
	}))
	require.NoError(t, s.db().Probes().RecordResult(t.Context(), &database.ProbeResult{
		ProbeID: "p-window", Kind: "ping", Timestamp: now.Add(-10 * time.Minute),
		Success: true, LatencyMs: 12,
	}))

	rec := httptest.NewRecorder()
	s.handleProbeHistory(rec, historyRequest(t, "/api/v1/history/probes/p-window?range=24h"))
	require.Equal(t, http.StatusOK, rec.Code)

	var got ProbeHistoryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "p-window", got.ProbeID)
	require.Equal(t, historyResolutionHourly, got.Window.Resolution)
	require.Equal(t, historySourceRaw, got.Window.Source,
		"a 24h window is inside every tier's raw horizon, so the open bucket is included")
	require.False(t, got.Window.Clamped)
	require.Equal(t, 1, got.Window.Days)
	require.Len(t, got.Points, 1)
	require.InDelta(t, 12.0, got.Points[0].AvgLatencyMs, 0.001)
}

// An empty series is `[]`, not `null`: the client should not have to tell "no
// data" from "no field".
func TestProbeHistoryServesAnEmptySeriesAsAnArray(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHistoryServer(t).handleProbeHistory(rec,
		historyRequest(t, "/api/v1/history/probes/never-run"))

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"points":[]`)
}

// Anomaly history is a whole-deployment view, so it carries no client claim
// requirement, but it must still report its window.
func TestAnomalyHistoryReportsTheServedWindow(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	newHistoryServer(t).handleAnomalyHistory(rec,
		historyRequest(t, "/api/v1/history/anomalies?range=3d"))
	require.Equal(t, http.StatusOK, rec.Code)

	var got AnomalyHistoryResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, 3, got.Window.Days)
	require.Equal(t, historySourceRaw, got.Window.Source)
	require.Len(t, got.Days, 4, "every day in the inclusive range, quiet days included")
	for _, d := range got.Days {
		require.Zero(t, d.Count)
	}
}
