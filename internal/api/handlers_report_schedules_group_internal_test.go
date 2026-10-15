package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

func scheduleBody(template, groupID string) ReportScheduleRequest {
	return ReportScheduleRequest{
		Name:     "weekly " + template,
		Template: template,
		Format:   "json",
		Schedule: reporting.Schedule{Frequency: reporting.FrequencyDaily, Hour: 6, Timezone: "UTC"},
		Enabled:  new(true),
		GroupID:  groupID,
	}
}

func postSchedule(t *testing.T, s *Server, body ReportScheduleRequest) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleReportSchedules(w, groupRequest(t, http.MethodPost, APIVersionPrefix+"/report-schedules", body))
	return w
}

// TestReportScheduleKeepsItsGroup: a capacity or summary schedule that names
// one of the caller's groups stores it, returns it, and keeps it on a PUT.
func TestReportScheduleKeepsItsGroup(t *testing.T) {
	s, _, target := newReportGroupTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core", Members: []string{target.ID}})

	for _, template := range []string{"capacity", "summary"} {
		t.Run(template, func(t *testing.T) {
			w := postSchedule(t, s, scheduleBody(template, g.ID))
			require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
			var created ReportScheduleInfo
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
			require.Equal(t, g.ID, created.GroupID)

			path := APIVersionPrefix + "/report-schedules/" + created.ID
			unscoped := scheduleBody(template, "")
			req := groupRequest(t, http.MethodPut, path, unscoped)
			req.SetPathValue("scheduleId", created.ID)
			w = httptest.NewRecorder()
			s.handleReportScheduleByID(w, req)
			require.Equal(t, http.StatusOK, w.Code, w.Body.String())
			var updated ReportScheduleInfo
			require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
			require.Empty(t, updated.GroupID, "a PUT replaces the whole schedule")
		})
	}
}

func TestReportScheduleRefusesABadGroup(t *testing.T) {
	s, db, _ := newReportGroupTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core"})
	seedOtherClient(t, db)
	other, err := s.deviceGroups.Create(t.Context(), devicegroups.Group{ClientID: otherClientID, Name: "theirs"})
	require.NoError(t, err)

	tests := []struct {
		name string
		body ReportScheduleRequest
	}{
		{"neither a capacity nor a summary template", scheduleBody("performance", g.ID)},
		{"unknown group", scheduleBody("capacity", "no-such-group")},
		{"another client's group", scheduleBody("capacity", other.ID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := postSchedule(t, s, tt.body)
			require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
		})
	}

	schedules, err := s.background.Reporting.Scheduler().List(t.Context())
	require.NoError(t, err)
	require.Empty(t, schedules, "a refused request must not create a schedule")
}

// TestDeviceGroupNamedByAScheduleCannotBeDeleted: deleting the group would
// turn a report on these devices into a report on every device.
func TestDeviceGroupNamedByAScheduleCannotBeDeleted(t *testing.T) {
	s, _, _ := newReportGroupTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core"})
	require.Equal(t, http.StatusCreated, postSchedule(t, s, scheduleBody("summary", g.ID)).Code)

	req := groupRequest(t, http.MethodDelete, deviceGroupsPath+"/"+g.ID, nil)
	req.SetPathValue("groupId", g.ID)
	w := httptest.NewRecorder()
	s.handleDeviceGroupByID(w, req)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}

// TestResolveReportGroupReadsMembersAtRunTime: a scheduled run reads the
// group as it stands then, so a member added after the schedule was saved is
// in the next report.
func TestResolveReportGroupReadsMembersAtRunTime(t *testing.T) {
	s, db, target := newReportGroupTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core"})
	require.Equal(t, http.StatusCreated, postSchedule(t, s, scheduleBody("summary", g.ID)).Code)

	scope, err := s.resolveReportGroup(t.Context(), g.ID)
	require.NoError(t, err)
	require.Equal(t, &reporting.GroupScope{ID: g.ID, Name: "core"}, scope)

	_, err = s.deviceGroups.Update(t.Context(), devicegroups.Group{
		ID: g.ID, ClientID: testClientID, Name: "core", Members: []string{target.ID},
	})
	require.NoError(t, err)
	scope, err = s.resolveReportGroup(t.Context(), g.ID)
	require.NoError(t, err)
	require.Equal(t, &reporting.GroupScope{
		ID: g.ID, Name: "core", Targets: []string{target.ID}, Addresses: []string{"10.0.0.1"},
	}, scope)

	seedOtherClient(t, db)
	other, err := s.deviceGroups.Create(t.Context(), devicegroups.Group{ClientID: otherClientID, Name: "theirs"})
	require.NoError(t, err)
	scope, err = s.resolveReportGroup(t.Context(), other.ID)
	require.NoError(t, err, "the scheduler reads any client's group; the routes checked it on save")
	require.Equal(t, other.ID, scope.ID)

	_, err = s.resolveReportGroup(t.Context(), "no-such-group")
	require.ErrorIs(t, err, devicegroups.ErrNotFound)
}
