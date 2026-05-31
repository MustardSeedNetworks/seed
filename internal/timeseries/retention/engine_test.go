package retention_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/license"
	"github.com/krisarmstrong/seed/internal/timeseries/retention"
)

func silentLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

type fakeTierProvider struct{ tier license.Tier }

func (f fakeTierProvider) GetTier() license.Tier { return f.tier }

type fakeSource struct {
	mu sync.Mutex

	name          string
	hourlyBuckets int
	dailyBuckets  int
	rollupHourErr error
	rollupDayErr  error
	purgeRawN     int64
	purgeHourlyN  int64
	purgeDailyN   int64

	calls struct {
		rollupHour  int
		rollupDay   int
		purgeRaw    int
		purgeHourly int
		purgeDaily  int
	}
	lastRawCutoff    time.Time
	lastHourlyCutoff time.Time
	lastDailyCutoff  time.Time
}

func (f *fakeSource) Name() string { return f.name }

func (f *fakeSource) RollupHour(_ context.Context, _ time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls.rollupHour++
	return f.hourlyBuckets, f.rollupHourErr
}

func (f *fakeSource) RollupDay(_ context.Context, _ time.Time) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls.rollupDay++
	return f.dailyBuckets, f.rollupDayErr
}

func (f *fakeSource) PurgeRaw(_ context.Context, cutoff time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls.purgeRaw++
	f.lastRawCutoff = cutoff
	return f.purgeRawN, nil
}

func (f *fakeSource) PurgeHourly(_ context.Context, cutoff time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls.purgeHourly++
	f.lastHourlyCutoff = cutoff
	return f.purgeHourlyN, nil
}

func (f *fakeSource) PurgeDaily(_ context.Context, cutoff time.Time) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls.purgeDaily++
	f.lastDailyCutoff = cutoff
	return f.purgeDailyN, nil
}

func TestEngine_RunOnce_FreeTier_PurgesHourlyAndDaily(t *testing.T) {
	t.Parallel()
	src := &fakeSource{name: "test"}
	e := retention.New(fakeTierProvider{tier: license.TierFree}, silentLogger())
	e.Register(src)

	e.RunOnce(context.Background())

	if src.calls.rollupHour != 1 {
		t.Errorf("RollupHour called %d times, want 1", src.calls.rollupHour)
	}
	// Free tier has DailyDays=0 → daily rollup is SKIPPED.
	if src.calls.rollupDay != 0 {
		t.Errorf("RollupDay called %d times, want 0 for Free", src.calls.rollupDay)
	}
	// All three purges always fire (cutoffs differ by tier).
	if src.calls.purgeRaw != 1 || src.calls.purgeHourly != 1 || src.calls.purgeDaily != 1 {
		t.Errorf("purges = (%d, %d, %d), want all 1",
			src.calls.purgeRaw, src.calls.purgeHourly, src.calls.purgeDaily)
	}
}

func TestEngine_RunOnce_ProTier_RunsDailyRollup(t *testing.T) {
	t.Parallel()
	src := &fakeSource{name: "test"}
	e := retention.New(fakeTierProvider{tier: license.TierPro}, silentLogger())
	e.Register(src)

	e.RunOnce(context.Background())

	if src.calls.rollupDay != 1 {
		t.Errorf("RollupDay called %d times, want 1 for Pro", src.calls.rollupDay)
	}
}

func TestEngine_RunOnce_TierHorizonsApplied(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier            license.Tier
		wantHourlyDelta time.Duration
		wantDailyDelta  time.Duration
	}{
		{license.TierFree, 0, 0},                                     // immediate
		{license.TierStarter, 30 * 24 * time.Hour, 0},                // 30d
		{license.TierPro, 90 * 24 * time.Hour, 730 * 24 * time.Hour}, // 90d, 2y
	}
	for _, tc := range tests {
		t.Run(tc.tier.String(), func(t *testing.T) {
			t.Parallel()
			src := &fakeSource{name: "test"}
			e := retention.New(fakeTierProvider{tier: tc.tier}, silentLogger())
			e.Register(src)

			before := time.Now().UTC()
			e.RunOnce(context.Background())

			// Hourly cutoff = now - HourlyDays.
			hourlyAge := before.Sub(src.lastHourlyCutoff)
			diff := absDuration(hourlyAge - tc.wantHourlyDelta)
			if diff > time.Minute {
				t.Errorf("hourly cutoff age = %v, want ~%v", hourlyAge, tc.wantHourlyDelta)
			}
			dailyAge := before.Sub(src.lastDailyCutoff)
			diff = absDuration(dailyAge - tc.wantDailyDelta)
			if diff > time.Minute {
				t.Errorf("daily cutoff age = %v, want ~%v", dailyAge, tc.wantDailyDelta)
			}
		})
	}
}

func TestEngine_Register_DuplicateNameReplaces(t *testing.T) {
	t.Parallel()
	a := &fakeSource{name: "x", hourlyBuckets: 1}
	b := &fakeSource{name: "x", hourlyBuckets: 99}
	e := retention.New(fakeTierProvider{tier: license.TierPro}, silentLogger())
	e.Register(a)
	e.Register(b) // replaces a

	e.RunOnce(context.Background())
	if a.calls.rollupHour != 0 {
		t.Errorf("original source still called after replacement")
	}
	if b.calls.rollupHour != 1 {
		t.Errorf("replacement source not called")
	}
}

func TestEngine_RunOnce_ErrorsAreLoggedNotPropagated(t *testing.T) {
	t.Parallel()
	src := &fakeSource{
		name:          "broken",
		rollupHourErr: errors.New("simulated DB error"),
	}
	e := retention.New(fakeTierProvider{tier: license.TierPro}, silentLogger())
	e.Register(src)

	// Should not panic; remaining steps (daily, purges) should still run.
	e.RunOnce(context.Background())

	if src.calls.rollupDay != 1 {
		t.Errorf("RollupDay should still run after RollupHour failure")
	}
	if src.calls.purgeRaw != 1 {
		t.Errorf("PurgeRaw should still run after rollup failure")
	}
}

func TestEngine_Start_NoSources_ReturnsError(t *testing.T) {
	t.Parallel()
	e := retention.New(fakeTierProvider{tier: license.TierFree}, silentLogger())
	err := e.Start(context.Background())
	if err == nil {
		t.Error("Start with no sources should return error")
	}
}

func TestEngine_Stop_Idempotent(t *testing.T) {
	t.Parallel()
	src := &fakeSource{name: "test"}
	e := retention.New(fakeTierProvider{tier: license.TierFree}, silentLogger())
	e.Register(src)
	// Stop before Start should be a no-op (no error).
	if err := e.Stop(context.Background()); err != nil {
		t.Errorf("Stop before Start = %v, want nil", err)
	}
}

func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}
