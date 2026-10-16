package api

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/autotest"
	"github.com/MustardSeedNetworks/seed/internal/license"
)

const (
	autotestResultPath       = autotestResultsPath + "/{resultId}"
	autotestResultExportPath = autotestResultPath + "/export"
)

func newAutotestResultsTestServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{}
	s.persist = app.NewPersistence(newTestDB(t))
	s.autotestResults = app.NewAutotestResults(s.persistence)
	return s
}

func keptRun(t *testing.T, s *Server, jack string) *autotest.Report {
	t.Helper()
	rep := autotestRun(t, jack)
	require.NoError(t, s.autotestResults.Insert(t.Context(), rep))
	return rep
}

// autotestRun is a finished run on eth0 that fails at DHCP.
func autotestRun(t *testing.T, jack string) *autotest.Report {
	t.Helper()
	stages := []autotest.Stage{
		{Name: autotest.StageLink, Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
			return autotest.StageReport{
				Outcome: autotest.OutcomePass, Reason: autotest.ReasonOK,
				Readings: []autotest.Reading{{Label: "speed", Value: "1000 Mb/s"}},
			}
		}},
		{Name: autotest.StageDHCP, Run: func(context.Context, string, autotest.Spec) autotest.StageReport {
			return autotest.StageReport{
				Outcome: autotest.OutcomeFail,
				Reason:  autotest.ReasonNoLeaseFound,
				Message: "no lease",
			}
		}},
	}
	return autotest.Run(
		t.Context(),
		"eth0",
		autotest.Spec{JackLabel: jack, AssetTag: "PP-0042"},
		stages,
		func(float64) {},
	)
}

func exportAutotest(t *testing.T, s *Server, id, format string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	routed(autotestResultExportPath, s.handleAutotestResultExport)(w, withClaim(httptest.NewRequest(
		http.MethodGet, autotestResultsPath+"/"+id+"/export?format="+format, http.NoBody)))
	return w
}

func importAutotest(t *testing.T, s *Server, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleAutotestResultImport(w, withClaim(httptest.NewRequest(
		http.MethodPost, autotestResultsPath+"/import", bytes.NewReader(body))))
	return w
}

// P-E2's acceptance: a completed run exports with its label fields populated
// and re-imports as the same record.
func TestAutotestResultExportReimportsAsTheSameRecord(t *testing.T) {
	src := newAutotestResultsTestServer(t)
	rep := keptRun(t, src, "B2-3F-R301-P07")

	w := exportAutotest(t, src, rep.ID, "json")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, `attachment; filename="autotest-`+rep.ID+`.json"`, w.Header().Get("Content-Disposition"))
	exported := w.Body.Bytes()
	var onWire autotest.Report
	require.NoError(t, json.Unmarshal(exported, &onWire))
	require.Equal(t, "B2-3F-R301-P07", onWire.JackLabel)
	require.Equal(t, "PP-0042", onWire.AssetTag)

	dst := newAutotestResultsTestServer(t)
	w = importAutotest(t, dst, exported)
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	require.Equal(t, autotestResultsPath+"/"+rep.ID, w.Header().Get("Location"))

	got, err := dst.autotestResults.Get(t.Context(), rep.ID)
	require.NoError(t, err)
	want, err := src.autotestResults.Get(t.Context(), rep.ID)
	require.NoError(t, err)
	require.Equal(t, want, got)

	w = exportAutotest(t, dst, rep.ID, "json")
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, string(exported), w.Body.String())

	w = importAutotest(t, dst, exported)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

func TestAutotestResultImportRefusesWhatARunCannotProduce(t *testing.T) {
	s := newAutotestResultsTestServer(t)
	good := keptRun(t, newAutotestResultsTestServer(t), "jack")
	valid, err := json.Marshal(good)
	require.NoError(t, err)

	mutate := func(f func(m map[string]any)) []byte {
		var m map[string]any
		require.NoError(t, json.Unmarshal(valid, &m))
		f(m)
		b, mErr := json.Marshal(m)
		require.NoError(t, mErr)
		return b
	}
	for name, body := range map[string][]byte{
		"not json":         []byte("id,interface\n"),
		"unknown field":    mutate(func(m map[string]any) { m["extra"] = 1 }),
		"id not a uuid":    mutate(func(m map[string]any) { m["id"] = "../etc" }),
		"verdict mismatch": mutate(func(m map[string]any) { m["verdict"] = "pass" }),
		"unknown stage":    mutate(func(m map[string]any) { m["stages"].([]any)[0].(map[string]any)["stage"] = "wifi" }),
		"label too long":   mutate(func(m map[string]any) { m["jackLabel"] = strings.Repeat("x", 65) }),
		"no interface":     mutate(func(m map[string]any) { delete(m, "interface") }),
	} {
		t.Run(name, func(t *testing.T) {
			w := importAutotest(t, s, body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}
	list, err := s.autotestResults.List(t.Context())
	require.NoError(t, err)
	require.Empty(t, list)
}

func TestAutotestResultCSVCarriesTheLabelsOnEveryStage(t *testing.T) {
	s := newAutotestResultsTestServer(t)
	rep := keptRun(t, s, "=HYPERLINK(\"x\")")

	w := exportAutotest(t, s, rep.ID, "csv")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "text/csv; charset=utf-8", w.Header().Get("Content-Type"))
	rows, err := csv.NewReader(w.Body).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, "jackLabel", rows[0][2])
	for _, row := range rows[1:] {
		require.Equal(t, rep.ID, row[0])
		require.Equal(t, `'=HYPERLINK("x")`, row[2], "a typed formula is neutralized")
		require.Equal(t, "PP-0042", row[3])
	}
	require.Equal(t, []string{"link", "pass", "ok", "", "speed=1000 Mb/s"}, rows[1][7:12])
	require.Equal(t, []string{"dhcp", "fail", "noLeaseFound", "no lease"}, rows[2][7:11])
}

func TestAutotestResultPDFIsPro(t *testing.T) {
	s := newAutotestResultsTestServer(t)
	rep := keptRun(t, s, "B2-3F-R301-P07")

	w := exportAutotest(t, s, rep.ID, "pdf")
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "application/pdf", w.Header().Get("Content-Type"))
	require.True(t, bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF-")))

	s.licenseMgr = licenseManagerAt(t, license.TierStarter)
	requireFeatureGate(t, exportAutotest(t, s, rep.ID, "pdf"), "audit_pdf")
	require.Equal(t, http.StatusOK, exportAutotest(t, s, rep.ID, "csv").Code)
}

func TestAutotestResultReads(t *testing.T) {
	s := newAutotestResultsTestServer(t)
	older := autotestRun(t, "first")
	older.StartedAt = older.StartedAt.Add(-time.Hour)
	require.NoError(t, s.autotestResults.Insert(t.Context(), older))
	newer := keptRun(t, s, "second")

	w := httptest.NewRecorder()
	s.handleAutotestResults(w, withClaim(httptest.NewRequest(http.MethodGet, autotestResultsPath, http.NoBody)))
	require.Equal(t, http.StatusOK, w.Code)
	var list AutotestResultListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Equal(t, 2, list.Count)
	require.Equal(t, []string{newer.ID, older.ID}, []string{list.Results[0].ID, list.Results[1].ID})

	for id, want := range map[string]int{
		older.ID:                               http.StatusOK,
		"0b6c3f0e-6a53-4c43-9a43-2f1d0f6b8a09": http.StatusNotFound,
		"not-a-uuid":                           http.StatusNotFound,
	} {
		w = httptest.NewRecorder()
		routed(autotestResultPath, s.handleAutotestResult)(w, withClaim(httptest.NewRequest(
			http.MethodGet, autotestResultsPath+"/"+id, http.NoBody)))
		require.Equal(t, want, w.Code, id)
	}
	require.Equal(t, http.StatusBadRequest, exportAutotest(t, s, older.ID, "xlsx").Code)
}
