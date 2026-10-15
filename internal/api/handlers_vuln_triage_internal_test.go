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

	"github.com/MustardSeedNetworks/foundation/pkg/httpserver/route"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/auth"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/identity/roles"
	"github.com/MustardSeedNetworks/seed/internal/security/vulntriage"
)

// newTriageServer scans one device carrying the fixture CVE and returns the
// server, the device (to rescan) and the finding's id.
func newTriageServer(t *testing.T) (*Server, func(osGuess string), int64, *database.DB) {
	t.Helper()
	db := newTestDB(t)
	s := &Server{persist: app.NewPersistence(db)}
	s.vulnTriage = app.NewVulnTriage(s.persistence)
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

	findings, err := db.Vulnerabilities().ListFindings(context.Background(), vulntriage.ListOptions{})
	require.NoError(t, err)
	require.Len(t, findings, 1)
	return s, rescan, findings[0].ID, db
}

func triage(t *testing.T, s *Server, id int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	return triageTarget(s, strconv.FormatInt(id, 10), body)
}

// triageTarget posts body to the status action of the finding named target,
// as alice.
func triageTarget(s *Server, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost,
		vulnFindingsPath+"/"+target+"/status", strings.NewReader(body))
	req = req.WithContext(auth.WithUsername(req.Context(), "alice"))
	w := httptest.NewRecorder()
	routed(vulnFindingStatusPath, s.handleVulnFindingStatus)(w, req)
	return w
}

func findingHistoryRaw(s *Server, id int64) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet,
		vulnFindingsPath+"/"+strconv.FormatInt(id, 10)+"/history", http.NoBody)
	w := httptest.NewRecorder()
	routed(vulnFindingHistoryPath, s.handleVulnFindingHistory)(w, req)
	return w
}

func findingHistory(t *testing.T, s *Server, id int64) []VulnStatusChangeResponse {
	t.Helper()
	w := findingHistoryRaw(s, id)
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
	s, rescan, id, db := newTriageServer(t)
	metrics := database.NewReportMetricsRepository(db)
	openCritical := func() int {
		counts, err := metrics.VulnerabilitySeverityCounts(ctx, time.Now().Add(-time.Hour))
		require.NoError(t, err)
		return counts["critical"]
	}
	status := func() vulntriage.Status {
		findings, err := db.Vulnerabilities().ListFindings(ctx, vulntriage.ListOptions{})
		require.NoError(t, err)
		require.Len(t, findings, 1)
		return findings[0].Status
	}

	w := triage(t, s, id, `{"status":"acknowledged"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, openCritical(), "an acknowledged finding is still open")
	rescan("Linux 5.4")
	require.Equal(t, vulntriage.StatusAcknowledged, status())

	w = triage(t, s, id, `{"status":"ignored","reason":"  kernel module not loaded  "}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 0, openCritical(), "an ignored finding leaves the counts")
	issues, err := metrics.TopIssues(ctx)
	require.NoError(t, err)
	require.Empty(t, issues)
	rescan("Linux 5.4")
	require.Equal(t, vulntriage.StatusIgnored, status(), "a rescan does not undo a triage decision")

	rescan("Windows 10")
	require.Equal(t, vulntriage.StatusResolved, status())
	rescan("Linux 5.4")
	require.Equal(t, vulntriage.StatusNew, status())
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
			body: `{"status":"ignored","reason":"` + strings.Repeat("x", vulntriage.ReasonMaxLen+1) + `"}`,
		},
		{name: "unknown finding", target: "999999", body: `{"status":"acknowledged"}`, want: http.StatusNotFound},
		{name: "malformed id", target: "abc", body: `{"status":"acknowledged"}`, want: http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, rescan, id, _ := newTriageServer(t)
			if tc.setup != nil {
				tc.setup(t, s, rescan, id)
			}
			target := strconv.FormatInt(id, 10)
			if tc.target != "" {
				target = tc.target
			}
			w := triageTarget(s, target, tc.body)
			require.Equal(t, tc.want, w.Code, w.Body.String())
			require.Len(t, findingHistory(t, s, id), tc.history, "a refused change writes no history")
		})
	}
}

func TestVulnFindingsList(t *testing.T) {
	s, _, id, _ := newTriageServer(t)
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

// TestVulnTriageStoreUnavailable pins the reply without a database: every
// route answers 503 rather than reaching a nil store.
func TestVulnTriageStoreUnavailable(t *testing.T) {
	t.Parallel()
	s := &Server{}
	w := httptest.NewRecorder()
	s.handleVulnFindings(w, httptest.NewRequest(http.MethodGet, vulnFindingsPath, http.NoBody))
	require.Equal(t, http.StatusServiceUnavailable, w.Code, vulnFindingsPath)
	require.Equal(t, http.StatusServiceUnavailable,
		triageTarget(s, "1", `{"status":"acknowledged"}`).Code, "status")
	require.Equal(t, http.StatusServiceUnavailable, findingHistoryRaw(s, 1).Code, "history")
}

// TestVulnTriageRequiresOperator pins the role gate: triage is a persistent
// write, while the list and the history stay readable by viewers.
func TestVulnTriageRequiresOperator(t *testing.T) {
	s := newRoutePolicyServer(t)
	byPath := make(map[string]route.Policy, len(s.routes.Policies()))
	for _, rt := range s.routes.Policies() {
		byPath[rt.Path] = rt
	}
	require.Equal(t, roles.Operator, byPath[vulnFindingStatusPath].Scope)
	require.Empty(t, byPath[vulnFindingHistoryPath].Scope)
	require.Equal(t, []string{http.MethodGet}, byPath[vulnFindingsPath].Methods)
}

// TestVulnFindingActionBodies pins the triage and history bodies key by key,
// so typing them cannot move the wire.
func TestVulnFindingActionBodies(t *testing.T) {
	s, rescan, id, _ := newTriageServer(t)
	keys := func(raw []byte) []string {
		t.Helper()
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(raw, &m))
		out := make([]string, 0, len(m))
		for k := range m {
			out = append(out, k)
		}
		return out
	}

	w := triage(t, s, id, `{"status":"acknowledged","reason":"tracked"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.JSONEq(t, `{"id":`+strconv.FormatInt(id, 10)+`,"status":"acknowledged"}`, w.Body.String())

	rescan("Windows 10")

	w = findingHistoryRaw(s, id)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.ElementsMatch(t, []string{"id", "history"}, keys(w.Body.Bytes()))
	var resp struct {
		History []json.RawMessage `json:"history"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Len(t, resp.History, 2)
	// The operator's triage, then the scanner's resolve with no actor or reason.
	require.ElementsMatch(t, []string{"from", "to", "actor", "reason", "changedAt"}, keys(resp.History[0]))
	require.ElementsMatch(t, []string{"from", "to", "changedAt"}, keys(resp.History[len(resp.History)-1]))
}
