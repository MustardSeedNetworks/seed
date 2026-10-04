package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/database"
	"github.com/MustardSeedNetworks/seed/internal/database/dbtest"
	"github.com/MustardSeedNetworks/seed/internal/polling/observation"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/iftable"
	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/sink"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

// TestRatingSinkPersistsRatesAcrossPolls drives the production publisher
// with a real database through three steady polls, an agent restart and a
// recovery poll. Every if_table observation lands in snmp_observations; the
// metrics store holds one rate per interval except the one spanning the
// restart, and none of them is negative.
func TestRatingSinkPersistsRatesAcrossPolls(t *testing.T) {
	t.Parallel()
	db, err := database.Open(dbtest.Path(t))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()

	pub := ratingSink{
		Sink:  sink.New(db.SNMPObservations(), nil, nil),
		rater: ifrate.NewRater(),
		rates: db.Metrics(),
	}
	start := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	polls := []struct {
		upTime   uint32
		inOctets uint64
		inErrors uint64
	}{
		{100_000, 1_000, 0},
		{106_000, 61_000, 60},
		{112_000, 121_000, 120},
		{118_000, 181_000, 180},
		{500, 10, 0}, // agent restarted: counters back near zero
		{6_500, 60_010, 60},
	}
	for i, p := range polls {
		upTime := p.upTime
		obs := iftable.Observation{
			ClientID:   "default",
			TargetID:   "sw-1",
			ObservedAt: start.Add(time.Duration(i) * time.Minute),
			SysUpTime:  &upTime,
			Rows: []iftable.Row{{
				IfIndex:  3,
				IfName:   "Gi0/3",
				Counters: iftable.Counters{InOctets: p.inOctets, InErrors: p.inErrors},
			}},
		}
		if pubErr := pub.PublishIfTable(ctx, obs); pubErr != nil {
			t.Fatalf("poll %d: %v", i, pubErr)
		}
	}

	stored, err := db.SNMPObservations().List(ctx, observation.ListOptions{Kind: sink.KindIfTable})
	if err != nil {
		t.Fatalf("list observations: %v", err)
	}
	if len(stored) != len(polls) {
		t.Errorf("if_table observations = %d, want %d", len(stored), len(polls))
	}

	for _, tc := range []struct {
		metric string
		want   float64
	}{
		{ifrate.MetricInOctets, 1000},
		{ifrate.MetricInErrors, 1},
		{ifrate.MetricOutOctets, 0},
	} {
		got, queryErr := db.Metrics().Query(ctx, database.MetricQueryOptions{
			InterfaceName: "sw-1/3",
			MetricType:    tc.metric,
		})
		if queryErr != nil {
			t.Fatalf("query %s: %v", tc.metric, queryErr)
		}
		if len(got) != 4 {
			t.Fatalf("%s samples = %d, want 4 (five intervals, one across the restart)", tc.metric, len(got))
		}
		for _, m := range got {
			if m.Value != tc.want {
				t.Errorf("%s at %s = %v, want %v", tc.metric, m.Timestamp, m.Value, tc.want)
			}
		}
	}
}
