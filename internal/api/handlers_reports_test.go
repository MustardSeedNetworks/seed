package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// These tests drive Mux() rather than Handler(): the bare mux skips the auth
// and licence middleware, which is what lets them reach the handlers at all.
// Route-level gating (auth, minRole, feature) is asserted by the capabilities
// golden instead. Going through Handler() and asserting "not 200" would pass
// even if the handler were never reached, which proves nothing about it.

func reportsTestConfig(t *testing.T) (*config.Config, string) {
	t.Helper()

	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	configPath := filepath.Join(t.TempDir(), "test-config.json")
	require.NoError(t, cfg.Save(configPath))

	return cfg, configPath
}

func reportsTestDB(t *testing.T) *database.DB {
	t.Helper()

	db, err := database.Open(filepath.Join(t.TempDir(), "reports-test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// reportsTestServer builds a Server whose background components carry a real
// reporting service over a temp database, so these tests exercise the wiring
// cmd_serve uses rather than a stand-in.
func reportsTestServer(t *testing.T) *api.Server {
	t.Helper()

	cfg, configPath := reportsTestConfig(t)
	db := reportsTestDB(t)

	s := api.NewServer(cfg, configPath, "", nil, false, nil, db, &api.BackgroundComponents{
		Reporting: app.NewReporting(cfg, db),
	})

	// Reports are gated on export_csv_json, which is Starter+. A test server
	// comes up on Free, so the routes would answer 402 before the handler ran.
	// The trial grants the feature; that the gate is enforced at all is
	// asserted separately by TestReports_FeatureGatedOnFreeTier.
	// Reports are gated on export_csv_json (Starter+) and a test server comes
	// up on Free. The manager is built against a temp dir, never the real user
	// config directory: NewServer's own manager persists there, so starting a
	// trial through it would write to the developer's machine.
	mgr, err := license.NewManagerWithDir(t.TempDir())
	require.NoError(t, err)
	require.True(t, mgr.StartTrial().Success)
	s.SetLicenseManagerForTest(mgr)

	// POST and DELETE are operator-gated. Without a real operator the write
	// routes answer 401 and the handler never runs, so the tests below would
	// be asserting the middleware rather than the code they name.
	_, err = db.CreateUser(t.Context(), reportsOperator, "$2a$10$x", database.RoleOperator)
	require.NoError(t, err)

	return s
}

// reportsOperator is the identity the write-route tests present.
const reportsOperator = "reports-operator"

// TestReports_FeatureGatedOnFreeTier proves the gate lives on the route rather
// than only in ReportsPage: a Free-tier caller never reaches the handler.
func TestReports_FeatureGatedOnFreeTier(t *testing.T) {
	cfg, configPath := reportsTestConfig(t)
	db := reportsTestDB(t)
	s := api.NewServer(cfg, configPath, "", nil, false, nil, db, &api.BackgroundComponents{
		Reporting: app.NewReporting(cfg, db),
	})

	// An unactivated manager over a temp dir, so this asserts Free-tier
	// behaviour regardless of any licence on the machine running the test.
	mgr, err := license.NewManagerWithDir(t.TempDir())
	require.NoError(t, err)
	s.SetLicenseManagerForTest(mgr)

	for _, tc := range []struct{ method, path string }{
		{http.MethodGet, "/api/v1/reports"},
		{http.MethodPost, "/api/v1/reports/generate"},
		{http.MethodGet, "/api/v1/reports/some-id"},
	} {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			rec := reportsDo(t, s, tc.method, tc.path, "")

			assert.Equal(t, http.StatusPaymentRequired, rec.Code, "body: %s", rec.Body.String())
			assert.Contains(t, rec.Body.String(), "export_csv_json")
		})
	}
}

func reportsDo(t *testing.T, s *api.Server, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, http.NoBody)
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("X-Username", reportsOperator)

	rec := httptest.NewRecorder()
	s.Mux().ServeHTTP(rec, req)

	return rec
}

// TestReports_UnavailableWithoutReportingService covers the nil-service path:
// reporting is optional at construction, so every handler resolves it first.
func TestReports_UnavailableWithoutReportingService(t *testing.T) {
	cfg, configPath := reportsTestConfig(t)
	s := api.NewServer(cfg, configPath, "", nil, false, nil, nil, nil)

	rec := reportsDo(t, s, http.MethodGet, "/api/v1/reports", "")

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
}

func TestReports_ListIsEmptyBeforeAnyGeneration(t *testing.T) {
	s := reportsTestServer(t)

	rec := reportsDo(t, s, http.MethodGet, "/api/v1/reports", "")
	require.Equal(t, http.StatusOK, rec.Code, "body: %s", rec.Body.String())

	var got api.ReportsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	// Empty, not null: the UI maps over this without a nil guard.
	assert.NotNil(t, got.Reports)
	assert.Empty(t, got.Reports)
}

// TestReports_GenerateAcceptsAndAppearsInList is the round trip. Generation
// finishes asynchronously, so the useful assertion is that the record exists
// immediately with a non-terminal status, not that the file is ready.
func TestReports_GenerateAcceptsAndAppearsInList(t *testing.T) {
	s := reportsTestServer(t)

	rec := reportsDo(t, s, http.MethodPost, "/api/v1/reports/generate",
		`{"type":"executive","format":"json"}`)
	require.Equal(t, http.StatusAccepted, rec.Code, "body: %s", rec.Body.String())

	var created api.ReportInfo
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "executive", created.Type)
	assert.Equal(t, "json", created.Format)

	listed := reportsDo(t, s, http.MethodGet, "/api/v1/reports", "")
	require.Equal(t, http.StatusOK, listed.Code)

	var list api.ReportsResponse
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &list))
	require.Len(t, list.Reports, 1)
	assert.Equal(t, created.ID, list.Reports[0].ID)

	got := reportsDo(t, s, http.MethodGet, "/api/v1/reports/"+created.ID, "")
	require.Equal(t, http.StatusOK, got.Code)

	var one api.ReportInfo
	require.NoError(t, json.Unmarshal(got.Body.Bytes(), &one))
	assert.Equal(t, created.ID, one.ID)
}

func TestReports_DeleteRemovesTheRecord(t *testing.T) {
	s := reportsTestServer(t)

	rec := reportsDo(t, s, http.MethodPost, "/api/v1/reports/generate",
		`{"type":"executive","format":"json"}`)
	require.Equal(t, http.StatusAccepted, rec.Code)

	var created api.ReportInfo
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	del := reportsDo(t, s, http.MethodDelete, "/api/v1/reports/"+created.ID, "")
	require.Equal(t, http.StatusNoContent, del.Code)

	after := reportsDo(t, s, http.MethodGet, "/api/v1/reports/"+created.ID, "")
	assert.Equal(t, http.StatusNotFound, after.Code)
}

// TestReports_GenerateRejectsUnknownTypeAndFormat covers the validation added
// with the endpoint: without it the generator's format switch is the first
// thing to see an arbitrary string off the wire.
func TestReports_GenerateRejectsUnknownTypeAndFormat(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"unknown type", `{"type":"not-a-type","format":"json"}`},
		{"unknown format", `{"type":"executive","format":"not-a-format"}`},
		{"both empty", `{"type":"","format":""}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := reportsTestServer(t)

			rec := reportsDo(t, s, http.MethodPost, "/api/v1/reports/generate", tt.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())

			listed := reportsDo(t, s, http.MethodGet, "/api/v1/reports", "")
			var list api.ReportsResponse
			require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &list))
			assert.Empty(t, list.Reports, "a rejected request must not create a report")
		})
	}
}

// absentReportID is a syntactically valid report id that names no report, so a
// 404 here means "no such report" rather than "malformed id".
const absentReportID = "00000000-0000-0000-0000-000000000000"

func TestReports_UnknownIDIsNotFound(t *testing.T) {
	s := reportsTestServer(t)

	rec := reportsDo(t, s, http.MethodGet, "/api/v1/reports/"+absentReportID, "")

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// TestReports_ByIDRejectsMalformedPaths guards the id parsing: one prefix
// handler serves both /reports/{id} and /reports/{id}/download.
func TestReports_ByIDRejectsMalformedPaths(t *testing.T) {
	s := reportsTestServer(t)

	for _, path := range []string{
		"/api/v1/reports/",
		"/api/v1/reports/a/b",
		// Not a UUID: a report id names a uuid.New() value and nothing else.
		"/api/v1/reports/no-such-report",
		// r.URL.Path arrives percent-decoded, so %0A is a real newline by the
		// time the handler sees it. Logging that verbatim would let a caller
		// forge log lines (CodeQL go/log-injection).
		"/api/v1/reports/" + url.PathEscape("00000000-0000-0000-0000-000000000000\nFAKE audit entry"),
		"/api/v1/reports/" + url.PathEscape("../../etc/passwd"),
	} {
		t.Run(path, func(t *testing.T) {
			rec := reportsDo(t, s, http.MethodGet, path, "")

			assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
		})
	}
}

// TestReports_DownloadDistinguishesPendingFromMissing: a report that exists but
// has not finished has nothing to stream, which is not the same answer as a
// report that does not exist.
func TestReports_DownloadDistinguishesPendingFromMissing(t *testing.T) {
	s := reportsTestServer(t)

	rec := reportsDo(t, s, http.MethodPost, "/api/v1/reports/generate",
		`{"type":"executive","format":"json"}`)
	require.Equal(t, http.StatusAccepted, rec.Code)

	var created api.ReportInfo
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &created))

	dl := reportsDo(t, s, http.MethodGet, "/api/v1/reports/"+created.ID+"/download", "")
	// Ready (200) or not ready (409) are both correct depending on how far the
	// goroutine got. 404 would be wrong: the report exists.
	assert.Contains(t, []int{http.StatusOK, http.StatusConflict}, dl.Code,
		"body: %s", dl.Body.String())

	missing := reportsDo(t, s, http.MethodGet, "/api/v1/reports/"+absentReportID+"/download", "")
	assert.Equal(t, http.StatusNotFound, missing.Code)
}

// TestReportInfoOmitsFilePath is the disclosure guard: reporting.Report carries
// an absolute server path and the API type deliberately does not.
func TestReportInfoOmitsFilePath(t *testing.T) {
	blob, err := json.Marshal(api.ReportInfo{
		ID:     "r1",
		Name:   "Executive",
		Type:   string(reporting.ReportTypeExecutive),
		Format: string(reporting.FormatJSON),
		Status: string(reporting.StatusComplete),
	})
	require.NoError(t, err)

	assert.NotContains(t, strings.ToLower(string(blob)), "filepath")
	assert.NotContains(t, string(blob), "/")
}
