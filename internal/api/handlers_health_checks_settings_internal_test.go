package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// TestHealthChecksSettingsRoundTrip exercises the ADR-0027 P2 store-of-record
// move: saving health-check settings persists the endpoint targets to the
// probes table, and reading them back reconstructs the same endpoints —
// including a vertical kind (HL7) that the pre-P2 save path silently dropped.
func TestHealthChecksSettingsRoundTrip(t *testing.T) {
	db := newTestDB(t)
	s := &Server{config: &config.Config{}, dbConn: db}

	req := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{
			{Name: "Google DNS", Host: "8.8.8.8", Enabled: true},
		},
		HL7Endpoints: []HL7EndpointResponse{
			{Name: "Lab MLLP", Host: "hl7.example", Port: 2575, SendingApp: "SEED", Enabled: true},
		},
	}

	// Save via the persistence path used by the PUT handler.
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/x", bytes.NewReader(body))
	require.True(t, s.saveHealthCheckProbes(w, r, &req),
		"saveHealthCheckProbes should succeed; got %d %s", w.Code, w.Body.String())

	// The probes table now holds exactly the two endpoints, as their kinds.
	probes, err := db.Probes().ListProbes(r.Context(), database.DefaultClientID, "")
	require.NoError(t, err)
	kinds := map[string]string{}
	for _, p := range probes {
		kinds[p.Kind] = p.Target
	}
	require.Equal(t, "8.8.8.8", kinds["ping"])
	require.Equal(t, "hl7.example", kinds["hl7"], "vertical HL7 endpoint must be persisted (pre-P2 dropped it)")

	// Read back through the GET handler.
	gw := httptest.NewRecorder()
	gr := httptest.NewRequest(http.MethodGet, "/x", http.NoBody)
	s.getHealthChecksSettings(gw, gr)
	require.Equal(t, http.StatusOK, gw.Code, "GET body: %s", gw.Body.String())

	var got TestsSettingsResponse
	require.NoError(t, json.Unmarshal(gw.Body.Bytes(), &got))
	require.Len(t, got.PingTargets, 1)
	require.Equal(t, "8.8.8.8", got.PingTargets[0].Host)
	require.True(t, got.PingTargets[0].Enabled)
	require.Len(t, got.HL7Endpoints, 1)
	require.Equal(t, "hl7.example", got.HL7Endpoints[0].Host)
	require.Equal(t, "SEED", got.HL7Endpoints[0].SendingApp, "vertical-specific params round-trip through params_json")
}

// TestSaveHealthCheckProbesReplacesPriorSet verifies that a second save
// replaces the prior health-check probes rather than appending.
func TestSaveHealthCheckProbesReplacesPriorSet(t *testing.T) {
	db := newTestDB(t)
	s := &Server{config: &config.Config{}, dbConn: db}

	first := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{
			{Name: "a", Host: "1.1.1.1", Enabled: true},
			{Name: "b", Host: "2.2.2.2", Enabled: true},
		},
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/x", http.NoBody)
	require.True(t, s.saveHealthCheckProbes(w, r, &first))

	second := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{{Name: "c", Host: "3.3.3.3", Enabled: true}},
	}
	require.True(t, s.saveHealthCheckProbes(httptest.NewRecorder(), r, &second))

	count, err := db.Probes().CountProbes(r.Context(), database.DefaultClientID, "ping")
	require.NoError(t, err)
	require.Equal(t, 1, count, "second save should replace, not append")
}
