// Package ifrate turns successive interface counter readings into
// per-second rates. SNMP counters are cumulative: a single reading says
// nothing about current traffic, and the difference between two readings
// is only a rate when nothing reset the counters in between.
//
// A Rater remembers the previous reading per target and interface. A pair
// of readings yields no rate when the agent restarted (sysUpTime went
// backwards), when the interface's counters were cleared
// (ifCounterDiscontinuityTime changed), or when the agent switched between
// 32-bit and 64-bit octet counters. A smaller 32-bit value with none of
// those is one wrap through 2^32; a smaller 64-bit value is a reset, since a
// Counter64 does not wrap in practice. Every skipped pair re-baselines, so
// the next reading rates normally.
package ifrate

import (
	"context"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/protocols/snmp"
)

// TargetKind is the metrics target_kind of an interface rate. Its
// target_id is "<polling target ID>/<ifIndex>".
const TargetKind = "snmp_interface"

// Metric types and units of the six rates a Rate carries.
const (
	MetricInOctets    = "if_in_octets"
	MetricOutOctets   = "if_out_octets"
	MetricInErrors    = "if_in_errors"
	MetricOutErrors   = "if_out_errors"
	MetricInDiscards  = "if_in_discards"
	MetricOutDiscards = "if_out_discards"

	UnitOctets  = "octets/s"
	UnitPackets = "packets/s"
)

// Reading is one interface's cumulative counters at one poll. Octet
// counters are 64-bit when WideOctets is set and 32-bit otherwise; errors
// and discards are always Counter32. Discontinuity is the interface's
// ifCounterDiscontinuityTime.
type Reading struct {
	IfIndex       uint32
	InOctets      uint64
	OutOctets     uint64
	WideOctets    bool
	InErrors      uint64
	OutErrors     uint64
	InDiscards    uint64
	OutDiscards   uint64
	Discontinuity uint32
}

// Snapshot is every interface reading of one target at one poll.
// SysUpTime is nil when the agent did not report it; such a snapshot
// cannot be told apart from a restart, so it yields no rates and clears
// what the Rater remembered for the target.
type Snapshot struct {
	ClientID  string
	TargetID  string
	At        time.Time
	SysUpTime *uint32
	Readings  []Reading
}

// Rate is one interface's per-second counter rates over the interval
// ending at At.
type Rate struct {
	ClientID    string
	TargetID    string
	IfIndex     uint32
	At          time.Time
	InOctets    float64
	OutOctets   float64
	InErrors    float64
	OutErrors   float64
	InDiscards  float64
	OutDiscards float64
}

// Point is one metric value of a Rate.
type Point struct {
	Type  string
	Unit  string
	Value float64
}

// Points lists the six rates as metric points, in a fixed order.
func (r Rate) Points() []Point {
	return []Point{
		{MetricInOctets, UnitOctets, r.InOctets},
		{MetricOutOctets, UnitOctets, r.OutOctets},
		{MetricInErrors, UnitPackets, r.InErrors},
		{MetricOutErrors, UnitPackets, r.OutErrors},
		{MetricInDiscards, UnitPackets, r.InDiscards},
		{MetricOutDiscards, UnitPackets, r.OutDiscards},
	}
}

// Store persists rates.
type Store interface {
	RecordInterfaceRates(ctx context.Context, rates []Rate) error
}

// Rater computes rates between each target's successive snapshots. It is
// safe for concurrent use. State lives in memory, so the first snapshot of
// a target after a restart only sets the baseline.
type Rater struct {
	mu   sync.Mutex
	last map[string]baseline
}

type baseline struct {
	at        time.Time
	sysUpTime uint32
	readings  map[uint32]Reading
}

// NewRater returns a Rater with no remembered readings.
func NewRater() *Rater {
	return &Rater{last: make(map[string]baseline)}
}

// Observe records snap as the target's baseline and returns the rates of
// every interface that also appeared in the previous snapshot, minus any
// whose counters were discontinuous between the two.
func (r *Rater) Observe(snap Snapshot) []Rate {
	r.mu.Lock()
	defer r.mu.Unlock()

	if snap.SysUpTime == nil {
		delete(r.last, snap.TargetID)
		return nil
	}
	prev, hadPrev := r.last[snap.TargetID]
	cur := baseline{at: snap.At, sysUpTime: *snap.SysUpTime, readings: make(map[uint32]Reading, len(snap.Readings))}
	for _, rd := range snap.Readings {
		cur.readings[rd.IfIndex] = rd
	}
	r.last[snap.TargetID] = cur

	seconds := snap.At.Sub(prev.at).Seconds()
	if !hadPrev || seconds <= 0 {
		return nil
	}
	var rates []Rate
	for _, rd := range snap.Readings {
		before, ok := prev.readings[rd.IfIndex]
		if !ok {
			continue
		}
		deltas, ok := counterDeltas(before, rd, prev.sysUpTime, cur.sysUpTime)
		if !ok {
			continue
		}
		rates = append(rates, Rate{
			ClientID:    snap.ClientID,
			TargetID:    snap.TargetID,
			IfIndex:     rd.IfIndex,
			At:          snap.At,
			InOctets:    float64(deltas[0]) / seconds,
			OutOctets:   float64(deltas[1]) / seconds,
			InErrors:    float64(deltas[2]) / seconds,
			OutErrors:   float64(deltas[3]) / seconds,
			InDiscards:  float64(deltas[4]) / seconds,
			OutDiscards: float64(deltas[5]) / seconds,
		})
	}
	return rates
}

// counterDeltas returns how far each counter advanced, in Rate field
// order, or false when any of them was discontinuous: one reset counter
// means the whole interface's counters restarted.
func counterDeltas(prev, cur Reading, prevUp, curUp uint32) ([6]uint64, bool) {
	if curUp < prevUp || prev.Discontinuity != cur.Discontinuity || prev.WideOctets != cur.WideOctets {
		return [6]uint64{}, false
	}
	// A Counter64 moving at 100 Gb/s takes decades to wrap, so a smaller
	// 64-bit value is a reset rather than a wrap.
	delta := func(p, c uint64, wide bool) (uint64, bool) {
		if wide {
			return c - p, c >= p
		}
		return snmp.Counter32Delta(p, c, prevUp, curUp)
	}
	var out [6]uint64
	var ok [6]bool
	out[0], ok[0] = delta(prev.InOctets, cur.InOctets, cur.WideOctets)
	out[1], ok[1] = delta(prev.OutOctets, cur.OutOctets, cur.WideOctets)
	out[2], ok[2] = delta(prev.InErrors, cur.InErrors, false)
	out[3], ok[3] = delta(prev.OutErrors, cur.OutErrors, false)
	out[4], ok[4] = delta(prev.InDiscards, cur.InDiscards, false)
	out[5], ok[5] = delta(prev.OutDiscards, cur.OutDiscards, false)
	return out, ok == [6]bool{true, true, true, true, true, true}
}
