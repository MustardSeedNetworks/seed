package sla_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

func start() time.Time { return time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC) }

func observation(minute int, ops ...ipsla.Operation) ipsla.Observation {
	return ipsla.Observation{
		ClientID:   "default",
		TargetID:   "rtr-1",
		ObservedAt: start().Add(time.Duration(minute) * time.Minute),
		Operations: ops,
	}
}

func echoRun(index int, uptime, rtt uint32) ipsla.Operation {
	return ipsla.Operation{
		Index:  index,
		Latest: &ipsla.Result{Sense: ipsla.SenseOK, RTTMs: rtt, RunUptime: uptime},
	}
}

func indexes(results []sla.Result) []int {
	out := []int{}
	for _, r := range results {
		out = append(out, r.Index)
	}
	return out
}

// An operation that runs every two minutes, polled every minute, is
// recorded once per run, not once per poll.
func TestTracker_RecordsEachRunOnce(t *testing.T) {
	t.Parallel()
	tracker := sla.NewTracker()
	polls := []struct {
		obs  ipsla.Observation
		want []int
	}{
		{observation(0, echoRun(10, 1000, 12), ipsla.Operation{Index: 40}), []int{10}},
		{observation(1, echoRun(10, 1000, 12)), []int{}},
		{observation(2, echoRun(10, 13_000, 14)), []int{10}},
		{observation(3, echoRun(10, 13_000, 14), echoRun(20, 13_500, 30)), []int{20}},
		// The agent rebooted: a new run with a smaller uptime is still new.
		{observation(4, echoRun(10, 200, 11), echoRun(20, 13_500, 30)), []int{10}},
	}
	for i, p := range polls {
		if got := indexes(tracker.Observe(p.obs)); !reflect.DeepEqual(got, p.want) {
			t.Errorf("poll %d recorded %v, want %v", i, got, p.want)
		}
	}
}

// A deleted operation is forgotten, so one re-created at the same index is
// recorded even when its first run lands on the old uptime.
func TestTracker_ForgetsDeletedOperations(t *testing.T) {
	t.Parallel()
	tracker := sla.NewTracker()
	tracker.Observe(observation(0, echoRun(10, 1000, 12)))
	tracker.Observe(observation(1))
	if got := indexes(tracker.Observe(observation(2, echoRun(10, 1000, 12)))); !reflect.DeepEqual(got, []int{10}) {
		t.Errorf("recorded %v, want [10]", got)
	}
}

func TestTracker_TargetsAreIndependent(t *testing.T) {
	t.Parallel()
	tracker := sla.NewTracker()
	first := observation(0, echoRun(10, 1000, 12))
	second := first
	second.TargetID = "rtr-2"
	tracker.Observe(first)
	if got := indexes(tracker.Observe(second)); !reflect.DeepEqual(got, []int{10}) {
		t.Errorf("second target recorded %v, want [10]", got)
	}
}

func TestTracker_Points(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		result ipsla.Result
		want   []sla.Point
	}{
		{
			"echo ok",
			ipsla.Result{Sense: ipsla.SenseOK, RTTMs: 12},
			[]sla.Point{
				{Type: sla.MetricCompleted, Value: 1, Unit: sla.UnitBool},
				{Type: sla.MetricRTT, Value: 12, Unit: sla.UnitMs},
			},
		},
		{
			"timeout records only that the run failed",
			ipsla.Result{Sense: ipsla.SenseTimeout, RTTMs: 5000},
			[]sla.Point{{Type: sla.MetricCompleted, Value: 0, Unit: sla.UnitBool}},
		},
		{
			"jitter over threshold",
			ipsla.Result{
				Sense: ipsla.SenseOverThreshold, RTTMs: 310,
				Jitter: &ipsla.Jitter{AvgJitterMs: 7, Received: 95, Lost: 5, MOS: 4.34},
			},
			[]sla.Point{
				{Type: sla.MetricCompleted, Value: 1, Unit: sla.UnitBool},
				{Type: sla.MetricRTT, Value: 310, Unit: sla.UnitMs},
				{Type: sla.MetricJitter, Value: 7, Unit: sla.UnitMs},
				{Type: sla.MetricLoss, Value: 5, Unit: sla.UnitPercent},
				{Type: sla.MetricMOS, Value: 4.34, Unit: sla.UnitMOS},
			},
		},
		{
			"jitter without codec or packets",
			ipsla.Result{Sense: ipsla.SenseOK, RTTMs: 20, Jitter: &ipsla.Jitter{AvgJitterMs: 2}},
			[]sla.Point{
				{Type: sla.MetricCompleted, Value: 1, Unit: sla.UnitBool},
				{Type: sla.MetricRTT, Value: 20, Unit: sla.UnitMs},
				{Type: sla.MetricJitter, Value: 2, Unit: sla.UnitMs},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.result
			results := sla.NewTracker().Observe(observation(0, ipsla.Operation{Index: 7, Latest: &result}))
			want := []sla.Result{{ClientID: "default", TargetID: "rtr-1", Index: 7, At: start(), Points: tt.want}}
			if !reflect.DeepEqual(results, want) {
				t.Errorf("results =\n%+v\nwant\n%+v", results, want)
			}
		})
	}
}
