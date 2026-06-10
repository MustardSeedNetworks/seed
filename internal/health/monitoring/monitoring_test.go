package monitoring_test

import (
	"context"
	"errors"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/health"
	"github.com/MustardSeedNetworks/seed/internal/health/monitoring"
)

// --- fakes --------------------------------------------------------------------

type fakeResults struct {
	available bool
	query     []*database.HealthCheckResult
	latest    []*database.HealthCheckResult
	daily     []*database.HealthCheckDailyRollup
	hourly    []*database.HealthCheckHourlyRollup
	lastOpts  database.HealthCheckQueryOptions
	err       error
}

func (f *fakeResults) Available() bool { return f.available }
func (f *fakeResults) Query(
	_ context.Context, opts database.HealthCheckQueryOptions,
) ([]*database.HealthCheckResult, error) {
	f.lastOpts = opts
	return f.query, f.err
}

func (f *fakeResults) LatestForAllEndpoints(_ context.Context) ([]*database.HealthCheckResult, error) {
	return f.latest, f.err
}

func (f *fakeResults) DailyRollups(
	_ context.Context, _, _ string, _ database.TimeRange,
) ([]*database.HealthCheckDailyRollup, error) {
	return f.daily, f.err
}

func (f *fakeResults) HourlyRollups(
	_ context.Context, _, _ string, _ database.TimeRange,
) ([]*database.HealthCheckHourlyRollup, error) {
	return f.hourly, f.err
}

type fakeScorer struct {
	available bool
	scores    []*health.EndpointHealthScore
	err       error
}

func (f *fakeScorer) Available() bool { return f.available }
func (f *fakeScorer) AllScores(_ context.Context) ([]*health.EndpointHealthScore, error) {
	return f.scores, f.err
}

type fakeSLA struct {
	available  bool
	lastPeriod string
}

func (f *fakeSLA) Available() bool { return f.available }
func (f *fakeSLA) CurrentPeriodReport(_ context.Context, _ string) (*health.SLAReport, error) {
	return &health.SLAReport{}, nil
}

func (f *fakeSLA) Summary(_ context.Context, period string) (*health.SLASummary, error) {
	f.lastPeriod = period
	return &health.SLASummary{}, nil
}

type fakeAlerts struct {
	available bool
	active    []*alerts.HealthAlert
	ackOK     bool
	lastID    string
}

func (f *fakeAlerts) Available() bool                     { return f.available }
func (f *fakeAlerts) ActiveAlerts() []*alerts.HealthAlert { return f.active }
func (f *fakeAlerts) Stats() alerts.AlertStats            { return alerts.AlertStats{Active: len(f.active)} }
func (f *fakeAlerts) Acknowledge(alertID, _ string) bool {
	f.lastID = alertID
	return f.ackOK
}

type fakeAnomaly struct {
	available bool
	anomalies []*health.Anomaly
	stats     map[string]*health.EndpointStats
}

func (f *fakeAnomaly) Available() bool                            { return f.available }
func (f *fakeAnomaly) ActiveAnomalies() []*health.Anomaly         { return f.anomalies }
func (f *fakeAnomaly) AllStats() map[string]*health.EndpointStats { return f.stats }

func newService(
	res *fakeResults, sc *fakeScorer, sla *fakeSLA, al *fakeAlerts, an *fakeAnomaly,
) *monitoring.Service {
	return monitoring.NewService(res, sc, sla, al, an)
}

// --- tests --------------------------------------------------------------------

func TestResults(t *testing.T) {
	t.Parallel()
	t.Run("unavailable returns ErrUnavailable", func(t *testing.T) {
		t.Parallel()
		svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
		if _, err := svc.Results(context.Background(), "", ""); !errors.Is(err, monitoring.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})
	t.Run("no filter uses latest-for-all", func(t *testing.T) {
		t.Parallel()
		latest := []*database.HealthCheckResult{{}}
		res := &fakeResults{available: true, latest: latest}
		svc := newService(res, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
		got, err := svc.Results(context.Background(), "", "")
		if err != nil || len(got) != 1 {
			t.Fatalf("want 1 latest result, got %d (err %v)", len(got), err)
		}
	})
	t.Run("filter uses query with the filter options", func(t *testing.T) {
		t.Parallel()
		res := &fakeResults{available: true, query: []*database.HealthCheckResult{{}, {}}}
		svc := newService(res, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
		got, err := svc.Results(context.Background(), "host-a", "tls")
		if err != nil || len(got) != 2 {
			t.Fatalf("want 2 results, got %d (err %v)", len(got), err)
		}
		if res.lastOpts.EndpointName != "host-a" || res.lastOpts.CheckType != "tls" {
			t.Fatalf("query opts not forwarded: %+v", res.lastOpts)
		}
	})
}

func TestHistoryKindByPeriod(t *testing.T) {
	t.Parallel()
	tests := []struct {
		period string
		want   monitoring.HistoryKind
	}{
		{"7d", monitoring.HistoryDailyRollups},
		{"30d", monitoring.HistoryDailyRollups},
		{"6h", monitoring.HistoryHourlyRollups},
		{"24h", monitoring.HistoryHourlyRollups},
		{"1h", monitoring.HistoryRaw},
		{"", monitoring.HistoryRaw},
		{"bogus", monitoring.HistoryRaw},
	}
	for _, tc := range tests {
		t.Run(tc.period, func(t *testing.T) {
			t.Parallel()
			res := &fakeResults{available: true}
			svc := newService(res, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
			h, err := svc.History(context.Background(), "", "", tc.period)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if h.Kind != tc.want {
				t.Fatalf("period %q: want kind %q, got %q", tc.period, tc.want, h.Kind)
			}
		})
	}
}

func TestHistoryUnavailable(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
	if _, err := svc.History(context.Background(), "", "", "24h"); !errors.Is(err, monitoring.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}

func TestScoresTally(t *testing.T) {
	t.Parallel()
	sc := &fakeScorer{available: true, scores: []*health.EndpointHealthScore{
		{Status: "healthy"},
		{Status: "healthy"},
		{Status: "degraded"},
		{Status: "critical"},
		{Status: "mystery"},
	}}
	svc := newService(&fakeResults{}, sc, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
	got, err := svc.Scores(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := monitoring.ScoreSummary{TotalEndpoints: 5, Healthy: 2, Degraded: 1, Critical: 1, Unknown: 1}
	if got.Summary != want {
		t.Fatalf("tally mismatch: want %+v, got %+v", want, got.Summary)
	}
}

func TestSLASummaryDefaultsPeriod(t *testing.T) {
	t.Parallel()
	sla := &fakeSLA{available: true}
	svc := newService(&fakeResults{}, &fakeScorer{}, sla, &fakeAlerts{}, &fakeAnomaly{})
	if _, err := svc.SLASummary(context.Background(), ""); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if sla.lastPeriod != "daily" {
		t.Fatalf("want default period daily, got %q", sla.lastPeriod)
	}
}

func TestAcknowledgeAlert(t *testing.T) {
	t.Parallel()
	t.Run("unavailable", func(t *testing.T) {
		t.Parallel()
		svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
		if err := svc.AcknowledgeAlert("a1", "bob"); !errors.Is(err, monitoring.ErrUnavailable) {
			t.Fatalf("want ErrUnavailable, got %v", err)
		}
	})
	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{},
			&fakeAlerts{available: true, ackOK: false}, &fakeAnomaly{})
		if err := svc.AcknowledgeAlert("a1", "bob"); !errors.Is(err, monitoring.ErrAlertNotFound) {
			t.Fatalf("want ErrAlertNotFound, got %v", err)
		}
	})
	t.Run("success", func(t *testing.T) {
		t.Parallel()
		al := &fakeAlerts{available: true, ackOK: true}
		svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{}, al, &fakeAnomaly{})
		if err := svc.AcknowledgeAlert("a1", "bob"); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if al.lastID != "a1" {
			t.Fatalf("alert id not forwarded: %q", al.lastID)
		}
	})
}

func TestAnomaliesFilterAndCount(t *testing.T) {
	t.Parallel()
	an := &fakeAnomaly{
		available: true,
		anomalies: []*health.Anomaly{
			{EndpointName: "host-a"}, {EndpointName: "host-b"}, {EndpointName: "host-a"},
		},
		stats: map[string]*health.EndpointStats{
			"host-a": {EndpointName: "host-a"},
			"host-b": {EndpointName: "host-b"},
		},
	}
	svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, an)

	t.Run("filtered list, total active count, filtered stats", func(t *testing.T) {
		t.Parallel()
		got, err := svc.Anomalies("host-a", true)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got.Anomalies) != 2 {
			t.Fatalf("want 2 filtered anomalies, got %d", len(got.Anomalies))
		}
		if got.ActiveCount != 3 {
			t.Fatalf("ActiveCount must be total (3), got %d", got.ActiveCount)
		}
		if len(got.Stats) != 1 || got.Stats[0].EndpointName != "host-a" {
			t.Fatalf("want 1 filtered stat for host-a, got %+v", got.Stats)
		}
	})
	t.Run("no filter returns all, no stats unless requested", func(t *testing.T) {
		t.Parallel()
		got, err := svc.Anomalies("", false)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(got.Anomalies) != 3 || got.ActiveCount != 3 {
			t.Fatalf("want all 3, got %d (count %d)", len(got.Anomalies), got.ActiveCount)
		}
		if got.Stats != nil {
			t.Fatalf("want nil stats when not requested, got %+v", got.Stats)
		}
	})
}

func TestAnomaliesUnavailable(t *testing.T) {
	t.Parallel()
	svc := newService(&fakeResults{}, &fakeScorer{}, &fakeSLA{}, &fakeAlerts{}, &fakeAnomaly{})
	if _, err := svc.Anomalies("", false); !errors.Is(err, monitoring.ErrUnavailable) {
		t.Fatalf("want ErrUnavailable, got %v", err)
	}
}
