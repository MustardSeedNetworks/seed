package api_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	fnd "github.com/MustardSeedNetworks/foundation/pkg/license"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/license"
	"github.com/MustardSeedNetworks/seed/internal/license/licensetest"
)

const schedulesPath = "/api/v1/reports/schedules"

// nightlyInventory is a valid create body: the inventory template produces CSV.
const nightlyInventory = `{"name":"Nightly inventory","template":"inventory","format":"csv",` +
	`"schedule":{"frequency":"daily","hour":2,"minute":30,"timezone":"UTC"},"enabled":true}`

func createSchedule(t *testing.T, s *api.Server, body string) api.ReportScheduleInfo {
	t.Helper()

	rec := reportsDo(t, s, http.MethodPost, schedulesPath, body)
	require.Equal(t, http.StatusCreated, rec.Code, "body: %s", rec.Body.String())

	var got api.ReportScheduleInfo
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	return got
}

// Scheduled reports are Pro. Free and Starter callers never reach the handler,
// and Starter is the case that matters: the rest of /reports is Starter.
func TestReportSchedules_ProOnly(t *testing.T) {
	fp, err := fnd.GenerateFingerprint()
	require.NoError(t, err)

	for _, tc := range []struct {
		name string
		mgr  func(t *testing.T) *license.Manager
	}{
		{"free", func(t *testing.T) *license.Manager {
			mgr, mgrErr := license.NewManagerWithDir(t.TempDir())
			require.NoError(t, mgrErr)
			return mgr
		}},
		{"starter", func(t *testing.T) *license.Manager {
			return licensetest.PaidManager(t, license.TierStarter, time.Now().Add(24*time.Hour), fp.Hash())
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := reportsTestServer(t)
			s.SetLicenseManagerForTest(tc.mgr(t))

			for _, req := range []struct{ method, path, body string }{
				{http.MethodGet, schedulesPath, ""},
				{http.MethodPost, schedulesPath, nightlyInventory},
				{http.MethodGet, schedulesPath + "/some-id", ""},
			} {
				rec := reportsDo(t, s, req.method, req.path, req.body)
				assert.Equal(
					t,
					http.StatusPaymentRequired,
					rec.Code,
					"%s %s: %s",
					req.method,
					req.path,
					rec.Body.String(),
				)
				assert.Contains(t, rec.Body.String(), "scheduled_reports")
			}
		})
	}
}

func TestReportSchedules_RoundTrip(t *testing.T) {
	s := reportsTestServer(t)

	listed := reportsDo(t, s, http.MethodGet, schedulesPath, "")
	require.Equal(t, http.StatusOK, listed.Code, "body: %s", listed.Body.String())
	assert.JSONEq(t, `{"schedules":[]}`, listed.Body.String(), "empty, not null")

	created := createSchedule(t, s, nightlyInventory)
	assert.NotEmpty(t, created.ID)
	assert.Equal(t, "Nightly inventory", created.Name)
	assert.True(t, created.Enabled)
	assert.Empty(t, created.LastRun)
	next, err := time.Parse(time.RFC3339, created.NextRun)
	require.NoError(t, err)
	assert.True(t, next.After(time.Now()))
	assert.Equal(t, 2, next.UTC().Hour())
	assert.Equal(t, 30, next.UTC().Minute())

	listed = reportsDo(t, s, http.MethodGet, schedulesPath, "")
	var list api.ReportSchedulesResponse
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &list))
	require.Len(t, list.Schedules, 1)
	assert.Equal(t, created.ID, list.Schedules[0].ID)

	byID := schedulesPath + "/" + created.ID
	updated := reportsDo(t, s, http.MethodPut, byID,
		`{"name":"Weekly vulnerabilities","template":"vulnerability","format":"pdf",`+
			`"schedule":{"frequency":"weekly","dayOfWeek":1,"hour":7,"minute":0,"timezone":"UTC"},"enabled":false}`)
	require.Equal(t, http.StatusOK, updated.Code, "body: %s", updated.Body.String())
	var put api.ReportScheduleInfo
	require.NoError(t, json.Unmarshal(updated.Body.Bytes(), &put))
	assert.Equal(t, created.ID, put.ID)
	assert.Equal(t, created.CreatedAt, put.CreatedAt, "a PUT keeps the creation time")
	assert.False(t, put.Enabled)
	next, err = time.Parse(time.RFC3339, put.NextRun)
	require.NoError(t, err)
	assert.Equal(t, time.Monday, next.UTC().Weekday())

	got := reportsDo(t, s, http.MethodGet, byID, "")
	require.Equal(t, http.StatusOK, got.Code)
	var one api.ReportScheduleInfo
	require.NoError(t, json.Unmarshal(got.Body.Bytes(), &one))
	assert.Equal(t, "Weekly vulnerabilities", one.Name)

	deleted := reportsDo(t, s, http.MethodDelete, byID, "")
	require.Equal(t, http.StatusNoContent, deleted.Code, "body: %s", deleted.Body.String())
	assert.Equal(t, http.StatusNotFound, reportsDo(t, s, http.MethodGet, byID, "").Code)
}

// A schedule created through the API is still there after a restart: a fresh
// reporting service over the same database loads it.
func TestReportSchedules_SurviveRestart(t *testing.T) {
	cfg, configPath := reportsTestConfig(t)
	db := reportsTestDB(t)
	s := reportsTestServerOn(t, cfg, configPath, db)

	created := createSchedule(t, s, nightlyInventory)

	restarted := app.NewReporting(cfg, db)
	require.NoError(t, restarted.Load(t.Context()))
	got, err := restarted.Scheduler().Get(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nightly inventory", got.Name)
	require.NotNil(t, got.NextRun)
	assert.Equal(t, created.NextRun, got.NextRun.UTC().Format(time.RFC3339))
}

func TestReportSchedules_RejectsInvalidBodies(t *testing.T) {
	s := reportsTestServer(t)

	for _, tc := range []struct {
		name, body, details string
	}{
		{
			name:    "format the template does not produce",
			body:    strings.Replace(nightlyInventory, `"csv"`, `"json"`, 1),
			details: "does not produce",
		},
		{
			name:    "unknown template",
			body:    strings.Replace(nightlyInventory, `"inventory"`, `"nope"`, 1),
			details: "unknown template",
		},
		{
			name:    "weekly without a day",
			body:    strings.Replace(nightlyInventory, `"daily"`, `"weekly"`, 1),
			details: "dayOfWeek",
		},
		{
			name:    "hour out of range",
			body:    strings.Replace(nightlyInventory, `"hour":2`, `"hour":24`, 1),
			details: "hour",
		},
		{
			name:    "unknown timezone",
			body:    strings.Replace(nightlyInventory, `"UTC"`, `"Mars/Olympus"`, 1),
			details: "timezone",
		},
		{name: "enabled left out", body: strings.Replace(nightlyInventory, `,"enabled":true`, "", 1)},
		{name: "unknown field", body: strings.Replace(nightlyInventory, `"enabled":true`, `"enabled":true,"recipients":[]`, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := reportsDo(t, s, http.MethodPost, schedulesPath, tc.body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, "body: %s", rec.Body.String())
			if tc.details != "" {
				assert.Contains(t, rec.Body.String(), tc.details)
			}
		})
	}

	listed := reportsDo(t, s, http.MethodGet, schedulesPath, "")
	assert.JSONEq(t, `{"schedules":[]}`, listed.Body.String(), "nothing invalid was stored")
}

func TestReportSchedules_ByIDPaths(t *testing.T) {
	s := reportsTestServer(t)
	unknown := schedulesPath + "/00000000-0000-4000-8000-000000000000"

	for _, tc := range []struct {
		name, method, path, body string
		want                     int
	}{
		{"malformed id", http.MethodGet, schedulesPath + "/not-a-uuid", "", http.StatusBadRequest},
		{"nested path", http.MethodGet, unknown + "/x", "", http.StatusBadRequest},
		{"unknown GET", http.MethodGet, unknown, "", http.StatusNotFound},
		{"unknown PUT", http.MethodPut, unknown, nightlyInventory, http.StatusNotFound},
		{"unknown DELETE", http.MethodDelete, unknown, "", http.StatusNotFound},
		{"POST on an id", http.MethodPost, unknown, nightlyInventory, http.StatusMethodNotAllowed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := reportsDo(t, s, tc.method, tc.path, tc.body)
			assert.Equal(t, tc.want, rec.Code, "body: %s", rec.Body.String())
		})
	}
}
