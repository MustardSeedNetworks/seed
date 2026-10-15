package sla_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp/collectors/ipsla"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
	"github.com/MustardSeedNetworks/seed/internal/timeseries/sla"
)

type fakeReader struct {
	ops      []sla.Operation
	runs     []sla.Run
	err      error
	from, to time.Time
}

func (f *fakeReader) Operations(context.Context) ([]sla.Operation, error) {
	return f.ops, f.err
}

func (f *fakeReader) Runs(_ context.Context, _ string, _ int, from, to time.Time) ([]sla.Run, error) {
	f.from, f.to = from, to
	return f.runs, f.err
}

func op(target string, index, sense int, ran bool) sla.Operation {
	o := sla.Operation{TargetID: target, TargetName: target, Index: index}
	if ran {
		o.Latest = &ipsla.Result{Sense: sense}
	}
	return o
}

func TestServiceList(t *testing.T) {
	t.Parallel()
	store := &fakeReader{ops: []sla.Operation{
		op("rtr-b", 1, ipsla.SenseOK, true),
		op("rtr-a", 2, 0, false),
		op("rtr-a", 3, ipsla.SenseOverThreshold, true),
		op("rtr-b", 4, ipsla.SenseTimeout, true),
		op("rtr-a", 1, ipsla.SenseOK, true),
	}}
	got, err := sla.NewService(store).List(context.Background())
	require.NoError(t, err)

	var order []string
	for _, o := range got {
		order = append(order, o.TargetID+"/"+strconv.Itoa(o.Index))
	}
	// Failed, over threshold, healthy by target then index, never run.
	require.Equal(t, []string{"rtr-b/4", "rtr-a/3", "rtr-a/1", "rtr-b/1", "rtr-a/2"}, order)
}

func TestServiceListError(t *testing.T) {
	t.Parallel()
	_, err := sla.NewService(&fakeReader{err: sla.ErrUnavailable}).List(context.Background())
	require.ErrorIs(t, err, sla.ErrUnavailable)
}

func TestServiceHistory(t *testing.T) {
	t.Parallel()
	to := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC)
	from := to.Add(-24 * time.Hour)
	at := from.Add

	tests := []struct {
		name string
		runs []sla.Run
		want []sla.Run
	}{
		{name: "no runs, no points", want: []sla.Run{}},
		{
			name: "a lone run per bucket is served as recorded",
			runs: []sla.Run{
				{At: at(time.Minute), CompletedPct: 100, RTTMs: new(7.0)},
				{At: at(7 * time.Minute), CompletedPct: 0},
			},
			want: []sla.Run{
				{At: at(time.Minute), Runs: 1, CompletedPct: 100, RTTMs: new(7.0)},
				{At: at(7 * time.Minute), Runs: 1, CompletedPct: 0},
			},
		},
		{
			name: "a bucket is the share completed and the mean of what was measured",
			runs: []sla.Run{
				{
					At: at(time.Minute), CompletedPct: 100,
					RTTMs: new(10.0), JitterMs: new(2.0), LossPct: new(0.0), MOS: new(4.4),
				},
				{At: at(2 * time.Minute), CompletedPct: 0},
				{
					At: at(3 * time.Minute), CompletedPct: 100,
					RTTMs: new(20.0), JitterMs: new(4.0), LossPct: new(10.0), MOS: new(4.0),
				},
				{At: at(4 * time.Minute), CompletedPct: 0},
			},
			want: []sla.Run{{
				At: at(4 * time.Minute), Runs: 4, CompletedPct: 50,
				RTTMs: new(15.0), JitterMs: new(3.0), LossPct: new(5.0), MOS: new(4.2),
			}},
		},
		{
			name: "a run at the window's end lands in the last bucket",
			runs: []sla.Run{{At: to, CompletedPct: 100}},
			want: []sla.Run{{At: to, Runs: 1, CompletedPct: 100}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeReader{runs: tt.runs}
			h, err := sla.NewService(store).History(context.Background(), "rtr-1", 10, ifstats.RangeDay, to)
			require.NoError(t, err)
			require.Equal(t, from, store.from)
			require.Equal(t, to, store.to)
			require.Equal(t, 24*time.Hour/ifstats.HistoryBuckets, h.Bucket)
			require.Len(t, h.Points, len(tt.want))
			for i, w := range tt.want {
				p := h.Points[i]
				require.Equal(t, w.At, p.At)
				require.Equal(t, w.Runs, p.Runs)
				require.InDelta(t, w.CompletedPct, p.CompletedPct, 1e-9)
				for _, pair := range [][2]*float64{{w.RTTMs, p.RTTMs}, {w.JitterMs, p.JitterMs}, {w.LossPct, p.LossPct}, {w.MOS, p.MOS}} {
					if pair[0] == nil {
						require.Nil(t, pair[1])
						continue
					}
					require.NotNil(t, pair[1])
					require.InDelta(t, *pair[0], *pair[1], 1e-9)
				}
			}
		})
	}
}

func TestServiceHistoryError(t *testing.T) {
	t.Parallel()
	boom := errors.New("boom")
	svc := sla.NewService(&fakeReader{err: boom})
	_, err := svc.History(context.Background(), "rtr-1", 10, ifstats.RangeHour, time.Now())
	require.ErrorIs(t, err, boom)
}
