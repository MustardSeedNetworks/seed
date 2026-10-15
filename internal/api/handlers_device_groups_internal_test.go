package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/database"
)

const deviceGroupPath = deviceGroupsPath + "/{groupId}"

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
