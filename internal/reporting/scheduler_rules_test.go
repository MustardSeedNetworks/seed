package reporting_test

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
	"github.com/MustardSeedNetworks/seed/internal/reporting/store"
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

	as := reporting.NewAggregatorService(cfg, store.NewMetricsRepo(db))
	gs := reporting.NewGeneratorService(cfg, store.NewReportRepo(db), store.NewExportRepo(db), ts, as)
	repo := store.NewScheduleRepo(db)
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
