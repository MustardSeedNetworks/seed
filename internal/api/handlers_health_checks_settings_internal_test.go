package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

// wireHealthSettings wires the health-settings use-case onto s over its current
// config/db/testers. Used by the api-internal tests that drive the GET/PUT
// handlers on a bare Server (no probe engine / testers — those degrade to no-ops).
func wireHealthSettings(s *Server) {
	s.healthSettings = app.NewHealthSettings(
		s.healthProbeRepo, s.rescheduleProbeEngine,
		s.config, s.configPath, s.dnsTester, s.speedtestTester,
		s.healthSettingsRepo,
	)
}

// newHealthSettingsServer builds a minimal Server wired with the health-settings
// use-case over a real test DB and a temp config path (so the PUT's config save
// succeeds), exercising the strangled PUT/GET path end-to-end.
func newHealthSettingsServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		config:     &config.Config{},
		configPath: filepath.Join(t.TempDir(), "config.yaml"),
		dbConn:     newTestDB(t),
	}
	wireHealthSettings(s)
	return s
}

func putHealthChecksSettings(t *testing.T, s *Server, req TestsSettingsResponse) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(req)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPut, "/api/v1/settings/health-checks", bytes.NewReader(body))
	s.updateHealthChecksSettings(w, r)
	return w
}

// TestHealthChecksSettingsRoundTrip exercises the ADR-0027 P2 store-of-record
// move: saving health-check settings persists the endpoint targets to the probes
// table, and reading them back reconstructs the same endpoints — including a
// vertical kind (HL7) that the pre-P2 save path silently dropped.
func TestHealthChecksSettingsRoundTrip(t *testing.T) {
	s := newHealthSettingsServer(t)

	req := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{
			{Name: "Google DNS", Host: "8.8.8.8", Enabled: true},
		},
		HL7Endpoints: []HL7EndpointResponse{
			{Name: "Lab MLLP", Host: "hl7.example", Port: 2575, SendingApp: "SEED", Enabled: true},
		},
	}

	w := putHealthChecksSettings(t, s, req)
	require.Equal(t, http.StatusOK, w.Code, "PUT body: %s", w.Body.String())

	// The probes table now holds exactly the two endpoints, as their kinds.
	probes, err := s.dbConn.Probes().ListProbes(t.Context(), database.DefaultClientID, "")
	require.NoError(t, err)
	kinds := map[string]string{}
	for _, p := range probes {
		kinds[p.Kind] = p.Target
	}
	require.Equal(t, "8.8.8.8", kinds["ping"])
	require.Equal(t, "hl7.example", kinds["hl7"], "vertical HL7 endpoint must be persisted (pre-P2 dropped it)")

	// Read back through the GET handler.
	gw := httptest.NewRecorder()
	gr := httptest.NewRequest(http.MethodGet, "/api/v1/settings/health-checks", http.NoBody)
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

// TestHealthChecksSettingsReplacesPriorSet verifies that a second save replaces
// the prior health-check probes rather than appending.
func TestHealthChecksSettingsReplacesPriorSet(t *testing.T) {
	s := newHealthSettingsServer(t)

	first := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{
			{Name: "a", Host: "1.1.1.1", Enabled: true},
			{Name: "b", Host: "2.2.2.2", Enabled: true},
		},
	}
	require.Equal(t, http.StatusOK, putHealthChecksSettings(t, s, first).Code)

	second := TestsSettingsResponse{
		PingTargets: []PingTargetResponse{{Name: "c", Host: "3.3.3.3", Enabled: true}},
	}
	require.Equal(t, http.StatusOK, putHealthChecksSettings(t, s, second).Code)

	count, err := s.dbConn.Probes().CountProbes(t.Context(), database.DefaultClientID, "ping")
	require.NoError(t, err)
	require.Equal(t, 1, count, "second save should replace, not append")
}
