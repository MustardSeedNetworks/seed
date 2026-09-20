package reporting_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/reporting"
)

// runUntilCancelled runs a blocking Run on its own goroutine and returns the
// stop function the supervisor stands in for: cancel, then wait for Run to
// return. Every reporting loop is a supervised worker since #2748, so a Run
// that ignores its context hangs shutdown — this helper fails such a Run
// instead of leaking it.
func runUntilCancelled(t *testing.T, run func(context.Context) error) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	var once sync.Once
	stop := func() {
		once.Do(func() {
			cancel()
			select {
			case err := <-done:
				// A cancellation caught mid-load surfaces as the context
				// error; the supervisor reads that as its own stop, not a
				// worker fault, so the helper accepts it too.
				if err != nil && !errors.Is(err, context.Canceled) {
					t.Errorf("Run returned %v, want nil on cancellation", err)
				}
			case <-time.After(10 * time.Second):
				t.Error("Run did not return after its context was cancelled")
			}
		})
	}
	t.Cleanup(stop)
	return stop
}

// startErr runs Run just long enough to surface a startup failure (template
// load, schedule load) without waiting out its loop.
func startErr(t *testing.T, run func(context.Context) error) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	return run(ctx)
}

// eventually waits for a condition a running worker brings about. Run blocks,
// so anything it does happens on the worker's goroutine and a read straight
// after starting it races that work.
func eventually(t *testing.T, done func() bool) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !done() {
		if time.Now().After(deadline) {
			t.Fatal("the running worker did not reach the expected state in time")
		}
		time.Sleep(time.Millisecond)
	}
}

// panicOnTypeReportRepo saves every report except the one whose type matches
// panicOn, for which it panics. It is the forced fault that proves a report
// generation cannot take the daemon — or its sibling reports — down (#2748).
type panicOnTypeReportRepo struct {
	panicOn reporting.ReportType

	mu    sync.Mutex
	saved []reporting.ReportType
}

func (r *panicOnTypeReportRepo) SaveReport(_ context.Context, rep *reporting.Report) error {
	if rep.Type == r.panicOn {
		panic("forced report-generation fault")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, rep.Type)
	return nil
}

func (r *panicOnTypeReportRepo) savedTypes() []reporting.ReportType {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.saved)
}

func (r *panicOnTypeReportRepo) GetReport(context.Context, string) (*reporting.Report, error) {
	return nil, errNotUsed
}

func (r *panicOnTypeReportRepo) ListReports(context.Context) ([]reporting.Report, error) {
	return nil, nil
}

// errNotUsed marks a port method this test never exercises.
var errNotUsed = errors.New("not used by this test")

func (r *panicOnTypeReportRepo) UpdateReport(context.Context, *reporting.Report) (bool, error) {
	return true, nil
}
func (r *panicOnTypeReportRepo) DeleteReport(context.Context, string) error { return nil }

// memScheduleRepo serves a fixed set of schedules and records what was saved.
type memScheduleRepo struct {
	rows []reporting.ScheduledReport
}

func (r *memScheduleRepo) ListSchedules(context.Context) ([]reporting.ScheduledReport, error) {
	return slices.Clone(r.rows), nil
}

func (r *memScheduleRepo) SaveSchedule(context.Context, *reporting.ScheduledReport) error {
	return nil
}
func (r *memScheduleRepo) DeleteSchedule(context.Context, string) error { return nil }

// TestSchedulerContainsAPanickingReport: two schedules come due in the same
// tick and one of them panics deep in generation. The panic must not kill the
// process, must not cancel the sibling report, and must not stop the tick loop
// — which is why each firing gets its own supervisor rather than a bare `go`
// or one group for the whole cycle.
func TestSchedulerContainsAPanickingReport(t *testing.T) {
	cfg := testConfig()
	templates := reporting.NewTemplateService(cfg)
	require.NoError(t, templates.Load())

	reports := &panicOnTypeReportRepo{panicOn: reporting.ReportTypeExecutive}
	gen := reporting.NewGeneratorService(cfg, reports, &nopExportRepo{}, templates,
		reporting.NewAggregatorService(cfg, &nopMetricsRepo{}))

	past := time.Now().Add(-time.Hour)
	schedules := &memScheduleRepo{rows: []reporting.ScheduledReport{
		{ID: "boom", Template: "executive", Format: reporting.FormatHTML, Enabled: true, NextRun: &past},
		{ID: "ok", Template: "inventory", Format: reporting.FormatHTML, Enabled: true, NextRun: &past},
	}}

	ss := reporting.NewSchedulerService(cfg, schedules, gen,
		reporting.WithTickInterval(5*time.Millisecond))
	require.NoError(t, ss.Load(context.Background()))
	stop := runUntilCancelled(t, ss.Run)

	// The sibling due in the same tick still generates.
	eventually(t, func() bool {
		return slices.Contains(reports.savedTypes(), reporting.ReportTypeInventory)
	})
	stop()
}

// nopExportRepo and nopMetricsRepo are empty data sources: the test asserts
// containment of a panic, not report content.
type nopExportRepo struct{}

func (nopExportRepo) ExportDevices(context.Context) ([]map[string]any, error) { return nil, nil }
func (nopExportRepo) ExportVulnerabilities(context.Context) ([]map[string]any, error) {
	return nil, nil
}

type nopMetricsRepo struct{}

func (nopMetricsRepo) CountDevices(context.Context) (int, error) { return 0, nil }
func (nopMetricsRepo) VulnerabilitySeverityCounts(
	context.Context, time.Time,
) (map[string]int, error) {
	return map[string]int{}, nil
}

func (nopMetricsRepo) PerformanceMetrics(context.Context, time.Time) (reporting.PerformanceMetrics, error) {
	return reporting.PerformanceMetrics{}, nil
}

func (nopMetricsRepo) TopIssues(context.Context) ([]reporting.IssueSummary, error) { return nil, nil }

func (nopMetricsRepo) Trends(context.Context, string, string) ([]reporting.DataPoint, error) {
	return nil, nil
}
