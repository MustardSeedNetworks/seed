package orchestrator

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

// TestRatingSinkTrendsIPSLAResults drives the production publisher with a
// real database through four polls of an echo operation that runs every
// other poll and fails once. Every poll lands in snmp_observations; the
// metrics store holds one point per run, and the hourly rollup trends them.
func TestRatingSinkTrendsIPSLAResults(t *testing.T) {
	t.Parallel()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	pub := ratingSink{
		Sink:    sink.New(db.SNMPObservations(), nil, nil),
		rater:   ifrate.NewRater(),
		tracker: sla.NewTracker(),
		metrics: db.Metrics(),
	}
	start := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	runs := []ipsla.Result{
		{Sense: ipsla.SenseOK, RTTMs: 10, RunUptime: 1000},
		{Sense: ipsla.SenseOK, RTTMs: 10, RunUptime: 1000},
		{Sense: ipsla.SenseTimeout, RunUptime: 13_000},
		{Sense: ipsla.SenseOK, RTTMs: 20, RunUptime: 25_000},
	}
	for i, run := range runs {
		obs := ipsla.Observation{
			ClientID:   "default",
			TargetID:   "rtr-1",
			ObservedAt: start.Add(time.Duration(i) * time.Minute),
			Operations: []ipsla.Operation{{Index: 5, Tag: "wan-gw", RttType: 1, Latest: &run}},
		}
		if pubErr := pub.PublishIPSLA(ctx, obs); pubErr != nil {
			t.Fatalf("poll %d: %v", i, pubErr)
		}
	}

	stored, err := db.SNMPObservations().List(ctx, observation.ListOptions{Kind: sink.KindIPSLA})
	if err != nil {
		t.Fatalf("list observations: %v", err)
	}
	if len(stored) != len(runs) {
		t.Errorf("ip_sla observations = %d, want %d", len(stored), len(runs))
	}

	for _, tc := range []struct {
		metric string
		want   []float64
	}{
		{sla.MetricCompleted, []float64{1, 0, 1}},
		{sla.MetricRTT, []float64{10, 20}},
	} {
		got, queryErr := db.Metrics().Query(ctx, database.MetricQueryOptions{
			InterfaceName: "rtr-1/5",
			MetricType:    tc.metric,
		})
		if queryErr != nil {
			t.Fatalf("query %s: %v", tc.metric, queryErr)
		}
		values := make([]float64, 0, len(got))
		for _, m := range got {
			values = append(values, m.Value)
		}
		slices.Sort(values)
		want := slices.Sorted(slices.Values(tc.want))
		if !slices.Equal(values, want) {
			t.Errorf("%s = %v, want %v (one point per run)", tc.metric, values, want)
		}
	}

	rolled, err := database.NewMetricsRollupSource(db).RollupHour(ctx, start)
	if err != nil {
		t.Fatalf("RollupHour: %v", err)
	}
	if rolled != 2 {
		t.Errorf("hourly rollup wrote %d series, want 2 (completed and rtt)", rolled)
	}
}
