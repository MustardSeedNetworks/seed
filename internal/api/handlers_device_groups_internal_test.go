package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/polling"
)

const (
	deviceGroupPath        = deviceGroupsPath + "/{groupId}"
	deviceGroupSummaryPath = deviceGroupPath + "/summary"
)

func newDeviceGroupsTestServer(t *testing.T) (*Server, *database.DB) {
	t.Helper()
	db := newTestDB(t)
	s := &Server{}
	s.persist = app.NewPersistence(db)
	s.deviceGroups = app.NewDeviceGroups(s.persistence)
	s.alertRules = app.NewAlertRules(s.persistence)
	return s, db
}

func groupRequest(t *testing.T, method, path string, body any) *http.Request {
	t.Helper()
	buf := &bytes.Buffer{}
	if body != nil {
		require.NoError(t, json.NewEncoder(buf).Encode(body))
	}
	return withClaim(httptest.NewRequest(method, path, buf))
}

func createGroup(t *testing.T, s *Server, body DeviceGroupRequest) DeviceGroupResponse {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleDeviceGroups(w, groupRequest(t, http.MethodPost, deviceGroupsPath, body))
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var g DeviceGroupResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &g))
	require.Equal(t, deviceGroupsPath+"/"+g.ID, w.Header().Get("Location"))
	return g
}

func TestDeviceGroupsCRUD(t *testing.T) {
	s, db := newDeviceGroupsTestServer(t)
	a := seedTarget(t, db, "core-a")
	b := seedTarget(t, db, "core-b")

	g := createGroup(t, s, DeviceGroupRequest{Name: "  core  ", Members: []string{b.ID, a.ID, b.ID}})
	require.Equal(t, "core", g.Name)
	require.ElementsMatch(t, []string{a.ID, b.ID}, g.Members)
	require.Len(t, g.Members, 2)

	w := httptest.NewRecorder()
	routed(deviceGroupPath, s.handleDeviceGroupByID)(
		w,
		groupRequest(
			t,
			http.MethodPut,
			deviceGroupsPath+"/"+g.ID,
			DeviceGroupRequest{Name: "core", Members: []string{a.ID}},
		),
	)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	w = httptest.NewRecorder()
	s.handleDeviceGroups(w, groupRequest(t, http.MethodGet, deviceGroupsPath, nil))
	require.Equal(t, http.StatusOK, w.Code)
	var list DeviceGroupListResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Equal(t, 1, list.Count)
	require.Equal(t, []string{a.ID}, list.Groups[0].Members)

	w = httptest.NewRecorder()
	routed(deviceGroupPath, s.handleDeviceGroupByID)(w,
		groupRequest(t, http.MethodDelete, deviceGroupsPath+"/"+g.ID, nil))
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	routed(deviceGroupPath, s.handleDeviceGroupByID)(w,
		groupRequest(t, http.MethodGet, deviceGroupsPath+"/"+g.ID, nil))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeviceGroupsRefuseBadInput(t *testing.T) {
	s, db := newDeviceGroupsTestServer(t)
	seedOtherClient(t, db)
	theirs := seedTargetFor(t, db, otherClientID, "theirs")
	createGroup(t, s, DeviceGroupRequest{Name: "taken"})

	tests := []struct {
		name string
		body any
		want int
	}{
		{"blank name", DeviceGroupRequest{Name: "  "}, http.StatusBadRequest},
		{"another client's target", DeviceGroupRequest{Name: "x", Members: []string{theirs.ID}}, http.StatusBadRequest},
		{"duplicate name", DeviceGroupRequest{Name: "taken"}, http.StatusConflict},
		{"unknown field", map[string]any{"name": "x", "clientId": otherClientID}, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			s.handleDeviceGroups(w, groupRequest(t, http.MethodPost, deviceGroupsPath, tt.body))
			require.Equal(t, tt.want, w.Code, w.Body.String())
		})
	}
}

func TestAlertRuleGroupMustBeTheCallersAndPinsTheGroup(t *testing.T) {
	s, _ := newDeviceGroupsTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core"})
	rule := map[string]any{
		"name": "core syslog", "enabled": true, "alertType": "system",
		"alertSeverity": "warning", "alertTitle": "core", "groupId": "grp-000000000000",
	}

	w := httptest.NewRecorder()
	s.handleAlertRules(w, groupRequest(t, http.MethodPost, alertRulesPath, rule))
	require.Equal(t, http.StatusBadRequest, w.Code, "a group the caller does not own: %s", w.Body.String())

	rule["groupId"] = g.ID
	w = httptest.NewRecorder()
	s.handleAlertRules(w, groupRequest(t, http.MethodPost, alertRulesPath, rule))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var created AlertRuleResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	require.Equal(t, g.ID, created.GroupID)

	w = httptest.NewRecorder()
	routed(deviceGroupPath, s.handleDeviceGroupByID)(w,
		groupRequest(t, http.MethodDelete, deviceGroupsPath+"/"+g.ID, nil))
	require.Equal(t, http.StatusConflict, w.Code, "a group a rule names must not be deletable")
}

func TestDeviceGroupSummaryShowsOnlyMembers(t *testing.T) {
	s, db := newDeviceGroupsTestServer(t)
	member := &polling.Target{ClientID: testClientID, Name: "core-a", IPAddress: "10.0.0.5"}
	outsider := &polling.Target{ClientID: testClientID, Name: "edge-a", IPAddress: "10.0.0.6"}
	for _, target := range []*polling.Target{member, outsider} {
		require.NoError(t, db.PollingTargets().Create(t.Context(), target))
	}
	require.NoError(t, db.PollingTargets().UpdateLastPoll(t.Context(), member.ID, "ok", ""))
	raise := func(source, severity string) {
		require.NoError(t, db.Alerts().Create(t.Context(), &alerts.Alert{
			Type: "connectivity", Severity: severity, Title: "t", Message: "m", Source: source,
		}))
	}
	raise("10.0.0.5:514", alerts.SeverityWarning) // the member's syslog
	raise(member.ID, alerts.SeverityCritical)     // the poller, about the member
	raise("10.0.0.6:514", alerts.SeverityCritical)
	raise(outsider.ID, alerts.SeverityCritical)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core", Members: []string{member.ID}})

	w := httptest.NewRecorder()
	routed(deviceGroupSummaryPath, s.handleDeviceGroupSummary)(w,
		groupRequest(t, http.MethodGet, deviceGroupsPath+"/"+g.ID+"/summary", nil))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var sum DeviceGroupSummaryResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &sum))
	require.Equal(t, g.ID, sum.ID)
	require.Equal(t, "core", sum.Name)
	require.Len(t, sum.Members, 1)
	require.Equal(t, member.ID, sum.Members[0].Target.ID)
	require.Equal(t, "ok", sum.Members[0].Target.LastStatus)
	require.NotEmpty(t, sum.Members[0].Target.LastSuccessAt)
	require.Equal(t, 2, sum.Members[0].OpenAlerts)
	require.Equal(t, DeviceGroupAlertCounts{Total: 2, Critical: 1, Warning: 1}, sum.OpenAlerts)
}

func TestDeviceGroupSummaryOfAnotherClientsGroupIsNotFound(t *testing.T) {
	s, db := newDeviceGroupsTestServer(t)
	seedOtherClient(t, db)
	theirs := &devicegroups.Group{ClientID: otherClientID, Name: "theirs"}
	require.NoError(t, db.DeviceGroups().Save(t.Context(), theirs))

	for _, id := range []string{theirs.ID, "grp-000000000000"} {
		w := httptest.NewRecorder()
		routed(deviceGroupSummaryPath, s.handleDeviceGroupSummary)(w,
			groupRequest(t, http.MethodGet, deviceGroupsPath+"/"+id+"/summary", nil))
		require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
	}
}
