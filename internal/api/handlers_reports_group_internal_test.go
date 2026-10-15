package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/polling"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// newReportGroupTestServer is the device-groups test server with reporting
// wired over the same database.
func newReportGroupTestServer(t *testing.T) (*Server, *database.DB, *polling.Target) {
	t.Helper()
	s, db := newDeviceGroupsTestServer(t)
	rep := app.NewReporting(&config.Config{}, s.persist)
	require.NoError(t, rep.Load(t.Context()))
	s.background = &BackgroundComponents{Reporting: rep}
	return s, db, seedTarget(t, db, "core-a")
}

func generateReport(t *testing.T, s *Server, body GenerateReportRequest) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	s.handleReportGenerate(w, groupRequest(t, http.MethodPost, APIVersionPrefix+"/reports/generate", body))
	return w
}

// TestReportGenerateScopesCapacityToTheCallersGroup: a capacity report that
// names one of the caller's groups carries the group's members, resolved at
// request time, into the generation.
func TestReportGenerateScopesCapacityToTheCallersGroup(t *testing.T) {
	s, _, target := newReportGroupTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core", Members: []string{target.ID}})

	w := generateReport(t, s, GenerateReportRequest{Type: "capacity", Format: "json", GroupID: g.ID})
	require.Equal(t, http.StatusAccepted, w.Code, w.Body.String())
	var created ReportInfo
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	report, err := s.background.Reporting.Generator().GetReport(t.Context(), created.ID)
	require.NoError(t, err)
	require.Equal(t, &reporting.GroupScope{ID: g.ID, Name: "core", Targets: []string{target.ID}},
		report.Parameters.Group)
}

func TestReportGenerateRefusesABadGroupFilter(t *testing.T) {
	s, _, _ := newReportGroupTestServer(t)
	g := createGroup(t, s, DeviceGroupRequest{Name: "core"})

	tests := []struct {
		name string
		body GenerateReportRequest
		want int
	}{
		{
			"not a capacity report",
			GenerateReportRequest{Type: "summary", Format: "json", GroupID: g.ID},
			http.StatusBadRequest,
		},
		{
			"unknown group",
			GenerateReportRequest{Type: "capacity", Format: "json", GroupID: "no-such-group"},
			http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := generateReport(t, s, tt.body)
			require.Equal(t, tt.want, w.Code, w.Body.String())
		})
	}

	reports, err := s.background.Reporting.Generator().ListReports(t.Context())
	require.NoError(t, err)
	require.Empty(t, reports, "a refused request must not create a report")
}

// TestReportGenerateDoesNotReachAnotherClientsGroup: a group is resolved
// within the caller's client, so another client's group is not found.
func TestReportGenerateDoesNotReachAnotherClientsGroup(t *testing.T) {
	s, db, _ := newReportGroupTestServer(t)
	seedOtherClient(t, db)
	other, err := s.deviceGroups.Create(t.Context(), devicegroups.Group{ClientID: otherClientID, Name: "theirs"})
	require.NoError(t, err)

	w := generateReport(t, s, GenerateReportRequest{Type: "capacity", Format: "json", GroupID: other.ID})
	require.Equal(t, http.StatusNotFound, w.Code, w.Body.String())
}
