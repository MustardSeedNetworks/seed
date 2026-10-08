package ifstats_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/ifstats"
)

func TestHistoryRangeDuration(t *testing.T) {
	t.Parallel()
	tests := map[ifstats.HistoryRange]time.Duration{
		ifstats.RangeHour: time.Hour,
		ifstats.RangeDay:  24 * time.Hour,
		ifstats.RangeWeek: 7 * 24 * time.Hour,
		"30d":             0,
		"":                0,
	}
	for rng, want := range tests {
		if got := rng.Duration(); got != want {
			t.Errorf("%q.Duration() = %v, want %v", rng, got, want)
		}
	}
}

func TestServiceHistory(t *testing.T) {
	t.Parallel()
	to := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	from := to.Add(-24 * time.Hour)
	bucket := 24 * time.Hour / ifstats.HistoryBuckets // 6 minutes
	at := from.Add

	tests := []struct {
		name  string
		polls []ifstats.Rates
		want  []ifstats.Rates
	}{
		{
			name: "no polls, no points",
			want: []ifstats.Rates{},
		},
		{
			name: "a lone poll per bucket is served as it was stored",
			polls: []ifstats.Rates{
				{At: at(time.Minute), InOctets: new(10.0), InErrors: 1},
				{At: at(7 * time.Minute), InOctets: new(20.0)},
			},
			want: []ifstats.Rates{
				{At: at(time.Minute), InOctets: new(10.0), InErrors: 1},
				{At: at(7 * time.Minute), InOctets: new(20.0)},
			},
		},
		{
			name: "traffic is the bucket's mean, errors its peak, At its last poll",
			polls: []ifstats.Rates{
				{At: at(time.Minute), InOctets: new(10.0), InUtilization: new(1.0), InErrors: 4, OutDiscards: 1},
				{At: at(2 * time.Minute), InOctets: new(30.0), InUtilization: new(3.0), OutErrors: 2},
				{At: at(5 * time.Minute), InOctets: new(20.0), InUtilization: new(2.0), InErrors: 1},
			},
			want: []ifstats.Rates{{
				At: at(5 * time.Minute), InOctets: new(20.0), InUtilization: new(2.0),
				InErrors: 4, OutErrors: 2, OutDiscards: 1,
			}},
		},
		{
			name: "an unrated octet counter does not pull the mean to zero",
			polls: []ifstats.Rates{
				{At: at(time.Minute), OutOctets: new(40.0)},
				{At: at(2 * time.Minute)},
			},
			want: []ifstats.Rates{{At: at(2 * time.Minute), OutOctets: new(40.0)}},
		},
		{
			name: "a poll at the window's close falls in the last bucket",
			polls: []ifstats.Rates{
				{At: to.Add(-bucket / 2), InErrors: 1},
				{At: to, InErrors: 2},
			},
			want: []ifstats.Rates{{At: to, InErrors: 2}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			svc := ifstats.NewService(fakeStore{polls: tc.polls})
			got, err := svc.History(context.Background(), "t1", 1, ifstats.RangeDay, to)
			require.NoError(t, err)
			require.Equal(t, from, got.From)
			require.Equal(t, to, got.To)
			require.Equal(t, bucket, got.Bucket)
			require.Equal(t, tc.want, got.Points)
		})
	}
}

func TestServiceHistoryPassesStoreError(t *testing.T) {
	t.Parallel()
	_, err := ifstats.NewService(fakeStore{err: ifstats.ErrUnavailable}).
		History(context.Background(), "t1", 1, ifstats.RangeHour, time.Now())
	if !errors.Is(err, ifstats.ErrUnavailable) {
		t.Errorf("err = %v, want ErrUnavailable", err)
	}
}
