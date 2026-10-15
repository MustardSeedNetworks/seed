package reporting_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/devicegroups"
	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// A run that finishes after its schedule was edited or deleted must not write
// back the copy it started with: the upsert would revert the edit or bring a
// deleted row back.
func TestRunScheduledReport_EditedOrDeletedMidRun(t *testing.T) {
	db, cleanup := testDBHelper(t)
	defer cleanup()

	cfg := testConfigHelper()
	ts := reporting.NewTemplateService(cfg)
	require.NoError(t, ts.Load())

	as := reporting.NewAggregatorService(cfg, database.NewReportMetricsRepository(db))
	gs := newGenerator(cfg, db, ts, as)
	repo := database.NewReportScheduleRepository(db)
	ss := reporting.NewSchedulerService(cfg, repo, gs)
	ctx := context.Background()

	newSchedule := func(name string) *reporting.ScheduledReport {
		sr := &reporting.ScheduledReport{
			Name:     name,
			Template: "executive",
			Format:   reporting.FormatHTML,
			Schedule: reporting.Schedule{Frequency: reporting.FrequencyDaily, Hour: 9, Timezone: "UTC"},
			Enabled:  true,
		}
		require.NoError(t, ss.Create(ctx, sr))
		return sr
	}

	t.Run("edited", func(t *testing.T) {
		due := newSchedule("before edit")
		edit := *due
		edit.Name = "after edit"
		require.NoError(t, ss.Update(ctx, &edit))

		ss.ExportRunScheduledReport(ctx, due)

		rows, err := repo.ListSchedules(ctx)
		require.NoError(t, err)
		idx := slices.IndexFunc(rows, func(r reporting.ScheduledReport) bool { return r.ID == due.ID })
		require.GreaterOrEqual(t, idx, 0)
		assert.Equal(t, "after edit", rows[idx].Name)
		assert.NotNil(t, rows[idx].LastRun)
	})

	t.Run("deleted", func(t *testing.T) {
		due := newSchedule("deleted")
		require.NoError(t, ss.Delete(ctx, due.ID))

		ss.ExportRunScheduledReport(ctx, due)

		rows, err := repo.ListSchedules(ctx)
		require.NoError(t, err)
		assert.False(t, slices.ContainsFunc(rows, func(r reporting.ScheduledReport) bool { return r.ID == due.ID }))
	})
}

func TestSchedulerService_Validate(t *testing.T) {
	ss, cleanup := setupSchedulerService(t)
	defer cleanup()

	ctx := context.Background()
	valid := func() *reporting.ScheduledReport {
		return &reporting.ScheduledReport{
			Name:     "nightly",
			Template: "inventory",
			Format:   reporting.FormatCSV,
			Schedule: reporting.Schedule{Frequency: reporting.FrequencyDaily, Hour: 2, Timezone: "UTC"},
			Enabled:  true,
		}
	}

	tests := []struct {
		name   string
		mutate func(sr *reporting.ScheduledReport)
		want   string
	}{
		{"blank name", func(sr *reporting.ScheduledReport) { sr.Name = "  " }, "name"},
		{"unknown template", func(sr *reporting.ScheduledReport) { sr.Template = "nope" }, "template"},
		{
			"format the template lacks",
			func(sr *reporting.ScheduledReport) { sr.Format = reporting.FormatJSON },
			"does not produce",
		},
		{"unknown frequency", func(sr *reporting.ScheduledReport) { sr.Schedule.Frequency = "hourly" }, "frequency"},
		{"hour 24", func(sr *reporting.ScheduledReport) { sr.Schedule.Hour = 24 }, "hour"},
		{"minute -1", func(sr *reporting.ScheduledReport) { sr.Schedule.Minute = -1 }, "minute"},
		{"unknown timezone", func(sr *reporting.ScheduledReport) { sr.Schedule.Timezone = "Mars/Olympus" }, "timezone"},
		{
			"weekly without day",
			func(sr *reporting.ScheduledReport) { sr.Schedule.Frequency = reporting.FrequencyWeekly },
			"dayOfWeek",
		},
		{"daily with day of week", func(sr *reporting.ScheduledReport) { sr.Schedule.DayOfWeek = new(1) }, "dayOfWeek"},
		{"weekly day 7", func(sr *reporting.ScheduledReport) {
			sr.Schedule.Frequency = reporting.FrequencyWeekly
			sr.Schedule.DayOfWeek = new(7)
		}, "dayOfWeek"},
		{
			"monthly without day",
			func(sr *reporting.ScheduledReport) { sr.Schedule.Frequency = reporting.FrequencyMonthly },
			"dayOfMonth",
		},
		{"monthly day 29", func(sr *reporting.ScheduledReport) {
			sr.Schedule.Frequency = reporting.FrequencyMonthly
			sr.Schedule.DayOfMonth = new(29)
		}, "dayOfMonth"},
		{"group on an inventory", func(sr *reporting.ScheduledReport) { sr.GroupID = "grp-1" }, "groupId"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sr := valid()
			tt.mutate(sr)
			err := ss.Create(ctx, sr)
			require.ErrorIs(t, err, reporting.ErrInvalidSchedule)
			assert.Contains(t, err.Error(), tt.want)
		})
	}

	require.NoError(t, ss.Create(ctx, valid()), "the unmutated schedule is valid")
}

// A group-scoped schedule runs over the group as the resolver reads it then,
// and a run whose group cannot be read generates nothing: it must never fall
// back to a report on every device. The schedule still moves to its next run.
func TestRunScheduledReport_GroupScope(t *testing.T) {
	db, cleanup := testDBHelper(t)
	defer cleanup()

	cfg := testConfigHelper()
	ts := reporting.NewTemplateService(cfg)
	require.NoError(t, ts.Load())
	gs := newGenerator(cfg, db, ts, reporting.NewAggregatorService(cfg, database.NewReportMetricsRepository(db)))
	repo := database.NewReportScheduleRepository(db)
	ctx := context.Background()

	group := devicegroups.Group{ClientID: database.DefaultClientID, Name: "core"}
	require.NoError(t, db.DeviceGroups().Save(ctx, &group))
	scope := &reporting.GroupScope{
		ID: group.ID, Name: "core", Targets: []string{"t-1"}, Addresses: []string{"10.0.0.1"},
	}

	tests := []struct {
		name     string
		resolver reporting.GroupResolver
		want     *reporting.GroupScope
	}{
		{"resolved", func(context.Context, string) (*reporting.GroupScope, error) { return scope, nil }, scope},
		{"group unreadable", func(context.Context, string) (*reporting.GroupScope, error) {
			return nil, errors.New("database is locked")
		}, nil},
		{"no resolver wired", nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ss := reporting.NewSchedulerService(cfg, repo, gs)
			if tt.resolver != nil {
				ss.SetGroupResolver(tt.resolver)
			}
			sr := &reporting.ScheduledReport{
				Name:     tt.name,
				Template: "summary",
				Format:   reporting.FormatJSON,
				Schedule: reporting.Schedule{Frequency: reporting.FrequencyDaily, Hour: 6, Timezone: "UTC"},
				GroupID:  group.ID,
				Enabled:  true,
			}
			require.NoError(t, ss.Create(ctx, sr))
			before, err := gs.ListReports(ctx)
			require.NoError(t, err)

			ss.ExportRunScheduledReport(ctx, sr)

			after, err := gs.ListReports(ctx)
			require.NoError(t, err)
			if tt.want == nil {
				assert.Len(t, after, len(before), "an unresolved group must not produce a report")
			} else {
				require.Len(t, after, len(before)+1)
				idx := slices.IndexFunc(after, func(r reporting.Report) bool {
					return !slices.ContainsFunc(before, func(b reporting.Report) bool { return b.ID == r.ID })
				})
				require.GreaterOrEqual(t, idx, 0)
				assert.Equal(t, tt.want, after[idx].Parameters.Group)
			}

			stored, err := ss.Get(ctx, sr.ID)
			require.NoError(t, err)
			assert.NotNil(t, stored.LastRun, "a failed run still moves the schedule on")
			assert.Equal(t, group.ID, stored.GroupID)
		})
	}
}

// The group survives a restart, and a group that does not exist is refused by
// the store rather than saved as a schedule on every device.
func TestScheduleRepo_GroupID(t *testing.T) {
	db, cleanup := testDBHelper(t)
	defer cleanup()
	ctx := context.Background()
	repo := database.NewReportScheduleRepository(db)

	group := devicegroups.Group{ClientID: database.DefaultClientID, Name: "core"}
	require.NoError(t, db.DeviceGroups().Save(ctx, &group))

	sr := &reporting.ScheduledReport{
		ID: "s-1", Name: "scoped", Template: "capacity", Format: reporting.FormatJSON,
		Schedule: reporting.Schedule{Frequency: reporting.FrequencyDaily, Timezone: "UTC"},
		GroupID:  group.ID, Enabled: true,
	}
	require.NoError(t, repo.SaveSchedule(ctx, sr))
	rows, err := repo.ListSchedules(ctx)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, group.ID, rows[0].GroupID)

	require.ErrorIs(t, db.DeviceGroups().Delete(ctx, database.DefaultClientID, group.ID), devicegroups.ErrInUse)

	sr.ID, sr.GroupID = "s-2", "no-such-group"
	require.ErrorIs(t, repo.SaveSchedule(ctx, sr), reporting.ErrInvalidSchedule)
}
