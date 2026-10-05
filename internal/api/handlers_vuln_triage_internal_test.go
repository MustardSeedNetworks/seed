package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/reporting/store"
)

// newTriageServer scans one device carrying the fixture CVE and returns the
// server, the device (to rescan) and the finding's id.
func newTriageServer(t *testing.T) (*Server, func(osGuess string), int64) {
	t.Helper()
	db := newTestDB(t)
	s := &Server{dbConn: db}
	scanner := newStoredVulnScanner(t, db)
	device := &discovery.DiscoveredDevice{
		IP: "10.20.30.50", MAC: "02:00:00:00:00:05", Vendor: "Linux", OSGuess: "Linux 5.4",
	}
	rescan := func(osGuess string) {
		device.OSGuess = osGuess
		_, err := scanner.ScanDevice(context.Background(), device)
		require.NoError(t, err)
	}
	rescan("Linux 5.4")

	findings, err := db.Vulnerabilities().ListFindings(context.Background(), database.VulnListOptions{})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	return s, rescan, findings[0].ID
}

func triage(t *testing.T, s *Server, id int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost,
		vulnFindingsPathPrefix+strconv.FormatInt(id, 10)+"/status", strings.NewReader(body))
	req = req.WithContext(auth.WithUsername(req.Context(), "alice"))
	w := httptest.NewRecorder()
	s.handleVulnFindingAction(w, req)
	return w
}

func findingHistory(t *testing.T, s *Server, id int64) []VulnStatusChangeResponse {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet,
		vulnFindingsPathPrefix+strconv.FormatInt(id, 10)+"/history", http.NoBody)
	w := httptest.NewRecorder()
	s.handleVulnFindingAction(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		History []VulnStatusChangeResponse `json:"history"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp.History
}

// TestVulnTriageLifecycle is S6-2's #899 acceptance: an operator's decision
// holds across rescans that still report the CVE, an ignored finding leaves
// the report counts, the scanner resolves and reopens whatever the triage
// state, and every step is in the history with who made it.
func TestVulnTriageLifecycle(t *testing.T) {
	ctx := context.Background()
	s, rescan, id := newTriageServer(t)
	metrics := store.NewMetricsRepo(s.db())
	openCritical := func() int {
		counts, err := metrics.VulnerabilitySeverityCounts(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		return counts["critical"]
	}
	status := func() database.VulnStatus {
		findings, err := s.db().Vulnerabilities().ListFindings(ctx, database.VulnListOptions{})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		return findings[0].Status
	}

	w := triage(t, s, id, `{"status":"acknowledged"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, openCritical(), "an acknowledged finding is still open")
	rescan("Linux 5.4")
	require.Equal(t, database.VulnStatusAcknowledged, status())

	w = triage(t, s, id, `{"status":"ignored","reason":"  kernel module not loaded  "}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 0, openCritical(), "an ignored finding leaves the counts")
	issues, err := metrics.TopIssues(ctx)
	require.NoError(t, err)
	require.Empty(t, issues)
	rescan("Linux 5.4")
	require.Equal(t, database.VulnStatusIgnored, status(), "a rescan does not undo a triage decision")

	rescan("Windows 10")
	require.Equal(t, database.VulnStatusResolved, status())
	rescan("Linux 5.4")
	require.Equal(t, database.VulnStatusNew, status())
	require.Equal(t, 1, openCritical())

	history := findingHistory(t, s, id)
	type step struct{ from, to, actor, reason string }
	got := make([]step, 0, len(history))
	for _, c := range history {
		require.False(t, c.ChangedAt.IsZero())
		got = append(got, step{c.From, c.To, c.Actor, c.Reason})
	}
	require.Equal(t, []step{
		{"new", "acknowledged", "alice", ""},
		{"acknowledged", "ignored", "alice", "kernel module not loaded"},
		{"ignored", "resolved", "", ""},
		{"resolved", "new", "", ""},
	}, got)
}

func TestVulnTriageRefusals(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(t *testing.T, s *Server, rescan func(string), id int64)
		target string
		body   string
		want   int
		// history is the entries the setup leaves; a refusal adds none.
		history int
	}{
		{name: "operator cannot resolve", body: `{"status":"resolved"}`, want: http.StatusConflict},
		{name: "same status", body: `{"status":"new"}`, want: http.StatusConflict},
		{
			name: "resolved finding", body: `{"status":"acknowledged"}`, want: http.StatusConflict, history: 1,
			setup: func(_ *testing.T, _ *Server, rescan func(string), _ int64) { rescan("Windows 10") },
		},
		{name: "ignore without reason", body: `{"status":"ignored","reason":"  "}`, want: http.StatusBadRequest},
		{name: "unknown status", body: `{"status":"fixed"}`, want: http.StatusBadRequest},
		{name: "unknown field", body: `{"status":"acknowledged","by":"bob"}`, want: http.StatusBadRequest},
		{
			name: "reason too long", want: http.StatusBadRequest,
			body: `{"status":"ignored","reason":"` + strings.Repeat("x", vulnReasonMaxLen+1) + `"}`,
		},
		{name: "unknown finding", target: "999999", body: `{"status":"acknowledged"}`, want: http.StatusNotFound},
		{name: "malformed id", target: "abc", body: `{"status":"acknowledged"}`, want: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, rescan, id := newTriageServer(t)
			if tc.setup != nil {
				tc.setup(t, s, rescan, id)
			}
			target := strconv.FormatInt(id, 10)
			if tc.target != "" {
				target = tc.target
			}
			req := httptest.NewRequest(http.MethodPost,
				vulnFindingsPathPrefix+target+"/status", strings.NewReader(tc.body))
			req = req.WithContext(auth.WithUsername(req.Context(), "alice"))
			w := httptest.NewRecorder()
			s.handleVulnFindingAction(w, req)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.Len(t, findingHistory(t, s, id), tc.history, "a refused change writes no history")
		})
	}
}

func TestVulnFindingsList(t *testing.T) {
	s, _, id := newTriageServer(t)
	list := func(query string) (int, []VulnFindingResponse) {
		w := httptest.NewRecorder()
		s.handleVulnFindings(w, httptest.NewRequest(http.MethodGet, vulnFindingsPath+query, http.NoBody))
		var resp struct {
			Findings []VulnFindingResponse `json:"findings"`
		}
		if w.Code == http.StatusOK {
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		}
		return w.Code, resp.Findings
	}

	code, findings := list("")
	require.Equal(t, http.StatusOK, code)
	require.Len(t, findings, 1)
	f := findings[0]
	require.Equal(t, id, f.ID)
	require.Equal(t, "10.20.30.50", f.DeviceIP)
	require.Equal(t, "CVE-2099-0001", f.CVEID)
	require.Equal(t, "critical", f.Severity)
	require.Equal(t, "new", f.Status)
	require.Nil(t, f.ResolvedAt)

	_, findings = list("?status=ignored")
	require.Empty(t, findings)
	_, findings = list("?status=new&device_id=" + f.DeviceID)
	require.Len(t, findings, 1)
	_, findings = list("?offset=1")
	require.Empty(t, findings)

	for _, bad := range []string{"?status=open", "?limit=0", "?limit=1001", "?offset=-1", "?limit=x"} {
		code, _ = list(bad)
		require.Equal(t, http.StatusBadRequest, code, bad)
	}
}

func TestVulnFindingActionMethods(t *testing.T) {
	s, _, id := newTriageServer(t)
	base := vulnFindingsPathPrefix + strconv.FormatInt(id, 10)
	for _, tc := range []struct{ method, path, allow string }{
		{http.MethodGet, base + "/status", http.MethodPost},
		{http.MethodPost, base + "/history", http.MethodGet},
	} {
		w := httptest.NewRecorder()
		s.handleVulnFindingAction(w, httptest.NewRequest(tc.method, tc.path, http.NoBody))
		require.Equal(t, http.StatusMethodNotAllowed, w.Code, tc.path)
		require.Equal(t, tc.allow, w.Header().Get("Allow"))
	}
	w := httptest.NewRecorder()
	s.handleVulnFindingAction(w, httptest.NewRequest(http.MethodGet, base+"/notes", http.NoBody))
	require.Equal(t, http.StatusNotFound, w.Code)
}

// TestVulnTriageRequiresOperator pins the role gate: triage is a persistent
// write, while the list and the history stay readable by viewers.
func TestVulnTriageRequiresOperator(t *testing.T) {
	s := newRoutePolicyServer(t)
	byPath := make(map[string]route, len(s.manifest))
	for _, rt := range s.manifest {
		byPath[rt.path] = rt
	}
	require.Equal(t, roles.Operator, byPath[vulnFindingsPathPrefix].minRole)
	require.Equal(t, []string{http.MethodGet}, byPath[vulnFindingsPath].methods)
}
