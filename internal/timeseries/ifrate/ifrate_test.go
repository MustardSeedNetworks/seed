package ifrate_test

import (
	"math"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifrate"
)

func t0() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC) }

// snap builds a one-interface snapshot of target "t1" taken secs after t0.
func snap(secs int, upTime uint32, rd ifrate.Reading) ifrate.Snapshot {
	rd.IfIndex = 1
	return ifrate.Snapshot{
		ClientID:  "c1",
		TargetID:  "t1",
		At:        t0().Add(time.Duration(secs) * time.Second),
		SysUpTime: &upTime,
		Readings:  []ifrate.Reading{rd},
	}
}

func TestRaterObserve(t *testing.T) {
	t.Parallel()
	base := ifrate.Reading{
		InOctets: 1000, OutOctets: 2000,
		InErrors: 10, OutErrors: 20, InDiscards: 30, OutDiscards: 40,
	}
	advanced := ifrate.Reading{
		InOctets: 7000, OutOctets: 14000,
		InErrors: 16, OutErrors: 32, InDiscards: 36, OutDiscards: 46,
	}
	tests := []struct {
		name   string
		second ifrate.Snapshot
		want   *ifrate.Rate
	}{
		{
			name:   "rates over the interval",
			second: snap(60, 106000, advanced),
			want: &ifrate.Rate{
				InOctets: 100, OutOctets: 200,
				InErrors: 0.1, OutErrors: 0.2, InDiscards: 0.1, OutDiscards: 0.1,
			},
		},
		{
			name: "one 32-bit wrap",
			second: snap(10, 101000, ifrate.Reading{
				InOctets: 999, OutOctets: 2000,
				InErrors: 10, OutErrors: 20, InDiscards: 30, OutDiscards: 40,
			}),
			want: &ifrate.Rate{InOctets: float64(math.MaxUint32-1000+999+1) / 10},
		},
		{
			name:   "agent restart: sysUpTime went backwards",
			second: snap(60, 500, ifrate.Reading{InOctets: 5}),
		},
		{
			name: "counters cleared: ifCounterDiscontinuityTime moved",
			second: snap(60, 106000, ifrate.Reading{
				InOctets: 5, OutOctets: 5, Discontinuity: 105000,
			}),
		},
		{
			name: "octet width changed between polls",
			second: snap(60, 106000, ifrate.Reading{
				InOctets: 7000, OutOctets: 14000, WideOctets: true,
				InErrors: 16, OutErrors: 32, InDiscards: 36, OutDiscards: 46,
			}),
		},
		{
			name:   "no time elapsed",
			second: snap(0, 100000, advanced),
		},
		{
			name: "interface not in the previous snapshot",
			second: func() ifrate.Snapshot {
				s := snap(60, 106000, advanced)
				s.Readings[0].IfIndex = 2
				return s
			}(),
		},
		{
			name: "sysUpTime not served",
			second: func() ifrate.Snapshot {
				s := snap(60, 0, advanced)
				s.SysUpTime = nil
				return s
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := ifrate.NewRater()
			if got := r.Observe(snap(0, 100000, base)); got != nil {
				t.Fatalf("first snapshot rated %+v, want only a baseline", got)
			}
			got := r.Observe(tt.second)
			if tt.want == nil {
				if len(got) != 0 {
					t.Errorf("rates = %+v, want none", got)
				}
				return
			}
			want := *tt.want
			want.ClientID, want.TargetID, want.IfIndex, want.At = "c1", "t1", 1, tt.second.At
			if len(got) != 1 || got[0] != want {
				t.Errorf("rates = %+v, want [%+v]", got, want)
			}
		})
	}
}

func TestRater64BitOctets(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		in     uint64
		upTime uint32
		wantOK bool
	}{
		{"advances past 2^32", 1<<40 + 6000, 106000, true},
		{"smaller value is a reset, not a wrap", 1 << 39, 106000, false},
		// The kernel's counters outlive an SNMP agent restart, so the value
		// can still rise while sysUpTime starts over.
		{"agent restart with the counter still rising", 1<<40 + 6000, 500, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r := ifrate.NewRater()
			r.Observe(snap(0, 100000, ifrate.Reading{InOctets: 1 << 40, WideOctets: true}))
			got := r.Observe(snap(60, tt.upTime, ifrate.Reading{InOctets: tt.in, WideOctets: true}))
			if (len(got) == 1) != tt.wantOK {
				t.Fatalf("rates = %+v, want rated=%v", got, tt.wantOK)
			}
			if tt.wantOK && got[0].InOctets != 100 {
				t.Errorf("InOctets = %v, want 100", got[0].InOctets)
			}
		})
	}
}

// TestRaterResetSeries is the row acceptance in miniature: three steady
// samples, a counter reset, then steady again. No sample may be negative
// or spike, and the reset costs exactly one rate.
func TestRaterResetSeries(t *testing.T) {
	t.Parallel()
	series := []struct {
		secs   int
		upTime uint32
		octets uint64
	}{
		{0, 100000, 0},
		{60, 106000, 60_000},
		{120, 112000, 120_000},
		{180, 118000, 180_000},
		{240, 1000, 500}, // agent rebooted
		{300, 7000, 60_500},
	}
	r := ifrate.NewRater()
	var got []float64
	for _, s := range series {
		for _, rate := range r.Observe(snap(s.secs, s.upTime, ifrate.Reading{InOctets: s.octets})) {
			got = append(got, rate.InOctets)
		}
	}
	want := []float64{1000, 1000, 1000, 1000}
	if len(got) != len(want) {
		t.Fatalf("rates = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("rate[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestRaterKeepsTargetsApart(t *testing.T) {
	t.Parallel()
	r := ifrate.NewRater()
	a := snap(0, 100000, ifrate.Reading{InOctets: 0})
	b := snap(0, 100000, ifrate.Reading{InOctets: 0})
	b.TargetID = "t2"
	r.Observe(a)
	r.Observe(b)
	b2 := snap(60, 106000, ifrate.Reading{InOctets: 600})
	b2.TargetID = "t2"
	got := r.Observe(b2)
	if len(got) != 1 || got[0].TargetID != "t2" || got[0].InOctets != 10 {
		t.Errorf("rates = %+v, want t2 at 10 octets/s", got)
	}
}

func TestRateServesAllSixPoints(t *testing.T) {
	t.Parallel()
	pts := ifrate.Rate{InOctets: 1, OutOctets: 2, InErrors: 3, OutErrors: 4, InDiscards: 5, OutDiscards: 6}.Points()
	want := []ifrate.Point{
		{ifrate.MetricInOctets, ifrate.UnitOctets, 1},
		{ifrate.MetricOutOctets, ifrate.UnitOctets, 2},
		{ifrate.MetricInErrors, ifrate.UnitPackets, 3},
		{ifrate.MetricOutErrors, ifrate.UnitPackets, 4},
		{ifrate.MetricInDiscards, ifrate.UnitPackets, 5},
		{ifrate.MetricOutDiscards, ifrate.UnitPackets, 6},
	}
	if len(pts) != len(want) {
		t.Fatalf("points = %+v", pts)
	}
	for i := range want {
		if pts[i] != want[i] {
			t.Errorf("point[%d] = %+v, want %+v", i, pts[i], want[i])
		}
	}
}
