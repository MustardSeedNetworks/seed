package history_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/timeseries/history"
)

// fakeStore answers each read with a marker naming the method, so a test sees
// which table the Service chose and what it was asked for.
type fakeStore struct {
	calls []string
	daily bool
}

func (f *fakeStore) ProbeTrendRaw(
	_ context.Context,
	_, _ string,
	_, _ time.Time,
	daily bool,
) ([]history.ProbeTrendPoint, error) {
	f.calls, f.daily = append(f.calls, "probe-raw"), daily
	return nil, nil
}

func (f *fakeStore) ProbeTrendRollup(
	_ context.Context,
	_, _ string,
	_, _ time.Time,
	daily bool,
) ([]history.ProbeTrendPoint, error) {
	f.calls, f.daily = append(f.calls, "probe-rollup"), daily
	return nil, nil
}

func (f *fakeStore) AnomalyCountsByDayLive(context.Context, time.Time, time.Time) ([]history.AnomalyDayCount, error) {
	f.calls = append(f.calls, "anomaly-live")
	return nil, nil
}

func (f *fakeStore) AnomalyCountsByDayRollup(context.Context, time.Time, time.Time) ([]history.AnomalyDayCount, error) {
	f.calls = append(f.calls, "anomaly-rollup")
	return nil, nil
}

func TestProbeTrendReadsTheSourceTheWindowNames(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name       string
		raw, daily bool
		want       string
	}{
		{"raw hourly", true, false, "probe-raw"},
		{"raw daily", true, true, "probe-raw"},
		{"rollup hourly", false, false, "probe-rollup"},
		{"rollup daily", false, true, "probe-rollup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			store := &fakeStore{}
			_, err := history.NewService(store).ProbeTrend(t.Context(), history.ProbeTrendQuery{
				ClientID: "c", ProbeID: "p", Raw: tc.raw, Daily: tc.daily,
			})
			require.NoError(t, err)
			require.Equal(t, []string{tc.want}, store.calls)
			require.Equal(t, tc.daily, store.daily)
		})
	}
}

func TestAnomalyCountsReadsTheSourceTheWindowNames(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		raw  bool
		want string
	}{
		{true, "anomaly-live"},
		{false, "anomaly-rollup"},
	} {
		store := &fakeStore{}
		_, err := history.NewService(store).AnomalyCounts(t.Context(), time.Time{}, time.Time{}, tc.raw)
		require.NoError(t, err)
		require.Equal(t, []string{tc.want}, store.calls)
	}
}
