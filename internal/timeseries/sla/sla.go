// Package sla turns the IP-SLA results the ip_sla collector reads into
// metric series, so the metrics rollups trend them like any other series.
//
// An operation keeps only its latest result, and it runs on its own schedule
// rather than seed's poll interval. Recording that result on every poll
// would weight a result by how many polls saw it, so a Tracker records a
// result once: when the run's timestamp (rttMonLatestRttOperTime) differs
// from the previous poll's. State lives in memory, so the first poll after a
// restart records the result it finds.
package sla

import (
	"context"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
)

// TargetKind is the metrics target_kind of an IP-SLA result. Its target_id
// is "<polling target ID>/<rttMonCtrlAdminIndex>".
const TargetKind = "ip_sla"

// Metric types. Completed is recorded for every run, so its average over a
// bucket is the share of runs that reached the target. The others are
// recorded only for a completed run, and the jitter metrics only for a
// jitter-type operation.
const (
	MetricCompleted = "ip_sla_completed"
	MetricRTT       = "ip_sla_rtt_ms"
	MetricJitter    = "ip_sla_jitter_ms"
	MetricLoss      = "ip_sla_loss_pct"
	MetricMOS       = "ip_sla_mos"
)

// Units of the metric types above.
const (
	UnitBool    = "bool"
	UnitMs      = "ms"
	UnitPercent = "percent"
	UnitMOS     = "mos"
)

const fullPercent = 100

// Point is one value of one series.
type Point struct {
	Type  string
	Value float64
	Unit  string
}

// Result is the points of one operation's new run.
type Result struct {
	ClientID string
	TargetID string
	Index    int
	At       time.Time
	Points   []Point
}

// Store persists results.
type Store interface {
	RecordSLAResults(ctx context.Context, results []Result) error
}

// Tracker remembers which run of each operation it last recorded. It is
// safe for concurrent use.
type Tracker struct {
	mu sync.Mutex
	// last maps a target to each operation's last recorded run uptime. A
	// poll replaces its target's map, so a deleted operation is forgotten.
	last map[string]map[int]uint32
}

// NewTracker returns a Tracker that has recorded nothing.
func NewTracker() *Tracker {
	return &Tracker{last: make(map[string]map[int]uint32)}
}

// Observe returns the points of every operation in obs whose latest run is
// new since the target's previous observation.
func (t *Tracker) Observe(obs ipsla.Observation) []Result {
	key := obs.ClientID + "\x00" + obs.TargetID
	runs := make(map[int]uint32, len(obs.Operations))

	t.mu.Lock()
	defer t.mu.Unlock()
	previous := t.last[key]

	var results []Result
	for _, op := range obs.Operations {
		if op.Latest == nil {
			continue
		}
		runs[op.Index] = op.Latest.RunUptime
		if last, seen := previous[op.Index]; seen && last == op.Latest.RunUptime {
			continue
		}
		results = append(results, Result{
			ClientID: obs.ClientID,
			TargetID: obs.TargetID,
			Index:    op.Index,
			At:       obs.ObservedAt,
			Points:   points(*op.Latest),
		})
	}
	t.last[key] = runs
	return results
}

func points(r ipsla.Result) []Point {
	if !r.Completed() {
		return []Point{{Type: MetricCompleted, Value: 0, Unit: UnitBool}}
	}
	out := []Point{
		{Type: MetricCompleted, Value: 1, Unit: UnitBool},
		{Type: MetricRTT, Value: float64(r.RTTMs), Unit: UnitMs},
	}
	if j := r.Jitter; j != nil {
		out = append(out, Point{Type: MetricJitter, Value: float64(j.AvgJitterMs), Unit: UnitMs})
		if sent := uint64(j.Received) + uint64(j.Lost); sent > 0 {
			out = append(out, Point{
				Type:  MetricLoss,
				Value: float64(j.Lost) / float64(sent) * fullPercent,
				Unit:  UnitPercent,
			})
		}
		if j.MOS > 0 {
			out = append(out, Point{Type: MetricMOS, Value: j.MOS, Unit: UnitMOS})
		}
	}
	return out
}
