package retention

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/license"
)

// hourlyPassInterval is how often the engine attempts hourly rollups
// and tier-aware purges. The actual hour-bucket selection rounds
// down to the previous completed hour.
const hourlyPassInterval = time.Hour

// dailyPassInterval is how often the engine attempts daily rollups.
// Runs at the same cadence as hourly; the source's RollupDay
// implementation is responsible for skipping incomplete days.
const dailyPassInterval = time.Hour

// initialPassDelay is the delay before the first rollup pass after
// Start. Stagger lets the application warm up; 60s is long enough
// to avoid contention with first-run migrations on slow disks.
const initialPassDelay = 60 * time.Second

// rawRetentionDays is the universal raw-table retention floor; the
// architecture locks 7 days across all tiers.
const rawRetentionDays = 7

// tierHorizons returns the (raw, hourly, daily) day-count tuple for
// a given license tier. Lock the architecture's locked-decision:
// Free retains raw-7d only; Starter adds hourly-30d; Pro adds
// daily-2y.
func tierHorizons(t license.Tier) TierHorizons {
	switch t {
	case license.TierPro:
		const proHourlyDays = 90
		const proDailyDays = 730
		return TierHorizons{RawDays: rawRetentionDays, HourlyDays: proHourlyDays, DailyDays: proDailyDays}
	case license.TierStarter:
		const starterHourlyDays = 30
		return TierHorizons{RawDays: rawRetentionDays, HourlyDays: starterHourlyDays, DailyDays: 0}
	case license.TierFree, license.TierInvalid:
		fallthrough
	default:
		return TierHorizons{RawDays: rawRetentionDays, HourlyDays: 0, DailyDays: 0}
	}
}

// TierProvider is the narrowed surface the engine needs from the
// license subsystem. Implementations of *license.Manager satisfy
// this; tests inject a fake.
type TierProvider interface {
	GetTier() license.Tier
}

// constTierProvider returns a fixed tier — used in fallback cases
// where no license manager was wired.
type constTierProvider struct{ tier license.Tier }

func (c constTierProvider) GetTier() license.Tier { return c.tier }

// Engine periodically rolls up and purges every registered
// RollupSource. Tier-aware horizons are re-read on each pass.
type Engine struct {
	logger *slog.Logger
	tier   TierProvider

	mu      sync.Mutex
	sources []RollupSource
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup

	// now is overridable in tests; production uses time.Now.
	now func() time.Time
}

// New returns an unstarted Engine. Pass a nil tier provider to
// default to Free (the safest tier). Pass nil logger to use the
// default slog logger.
func New(tier TierProvider, logger *slog.Logger) *Engine {
	if tier == nil {
		tier = constTierProvider{tier: license.TierFree}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Engine{
		logger: logger,
		tier:   tier,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Register adds a RollupSource. Must be called before Start.
// Re-registering the same Name replaces the previous source.
func (e *Engine) Register(src RollupSource) {
	if src == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, existing := range e.sources {
		if existing.Name() == src.Name() {
			e.sources[i] = src
			return
		}
	}
	e.sources = append(e.sources, src)
}

// Name returns "retention". Implements [engine.Engine] so the
// retention engine registers in the lifecycle registry alongside
// the probe + snmp engines.
func (*Engine) Name() string { return "retention" }

// Start begins the periodic rollup + purge loop. Returns nil if
// already started.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return nil
	}
	if len(e.sources) == 0 {
		return errors.New("retention engine: no sources registered")
	}

	loopCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.started = true

	e.wg.Add(1)
	go e.loop(loopCtx)

	e.logger.InfoContext(ctx, "retention engine started",
		"sources", len(e.sources),
	)
	return nil
}

// Stop signals the loop to exit and waits up to ctx deadline for
// in-flight passes to complete.
func (e *Engine) Stop(ctx context.Context) error {
	e.mu.Lock()
	if !e.started {
		e.mu.Unlock()
		return nil
	}
	cancel := e.cancel
	e.mu.Unlock()
	cancel()

	done := make(chan struct{})
	go func() {
		e.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	e.mu.Lock()
	e.started = false
	e.mu.Unlock()
	e.logger.InfoContext(ctx, "retention engine stopped")
	return nil
}

// RunOnce performs one full pass synchronously — rollup + purge
// across all sources. Used by tests and for one-off operator
// triggers. Safe to call concurrently with the loop because the
// underlying SQL operations are idempotent.
func (e *Engine) RunOnce(ctx context.Context) {
	e.mu.Lock()
	sources := make([]RollupSource, len(e.sources))
	copy(sources, e.sources)
	tier := e.tier.GetTier()
	now := e.now()
	e.mu.Unlock()

	horizons := tierHorizons(tier)
	hourStart := now.Add(-hourlyPassInterval).Truncate(time.Hour)
	dayStart := now.Add(-dailyPassInterval).Truncate(hoursPerDay * time.Hour)

	rawCutoff := now.AddDate(0, 0, -horizons.RawDays)
	hourlyCutoff := now.AddDate(0, 0, -horizons.HourlyDays)
	dailyCutoff := now.AddDate(0, 0, -horizons.DailyDays)

	for _, src := range sources {
		e.runOneSource(ctx, src, hourStart, dayStart,
			rawCutoff, hourlyCutoff, dailyCutoff, horizons)
	}
}

// runOneSource processes one source through the full rollup + purge
// pipeline. Errors are logged and continued past — partial progress
// is better than an entire pass aborting on one source's hiccup.
func (e *Engine) runOneSource(
	ctx context.Context,
	src RollupSource,
	hourStart, dayStart, rawCutoff, hourlyCutoff, dailyCutoff time.Time,
	horizons TierHorizons,
) {
	if n, err := src.RollupHour(ctx, hourStart); err != nil {
		e.logger.WarnContext(ctx, "retention: hourly rollup failed",
			"source", src.Name(), "error", err)
	} else if n > 0 {
		e.logger.DebugContext(ctx, "retention: hourly rollup",
			"source", src.Name(), "buckets", n)
	}

	if horizons.DailyDays > 0 {
		if n, err := src.RollupDay(ctx, dayStart); err != nil {
			e.logger.WarnContext(ctx, "retention: daily rollup failed",
				"source", src.Name(), "error", err)
		} else if n > 0 {
			e.logger.DebugContext(ctx, "retention: daily rollup",
				"source", src.Name(), "buckets", n)
		}
	}

	if n, err := src.PurgeRaw(ctx, rawCutoff); err != nil {
		e.logger.WarnContext(ctx, "retention: purge raw failed",
			"source", src.Name(), "error", err)
	} else if n > 0 {
		e.logger.DebugContext(ctx, "retention: purged raw rows",
			"source", src.Name(), "rows", n)
	}

	// Hourly purge: if HourlyDays=0 the cutoff is "now" — purges
	// everything older than now (i.e. all hourly rollups).
	if n, err := src.PurgeHourly(ctx, hourlyCutoff); err != nil {
		e.logger.WarnContext(ctx, "retention: purge hourly failed",
			"source", src.Name(), "error", err)
	} else if n > 0 {
		e.logger.DebugContext(ctx, "retention: purged hourly rollups",
			"source", src.Name(), "rows", n)
	}

	if n, err := src.PurgeDaily(ctx, dailyCutoff); err != nil {
		e.logger.WarnContext(ctx, "retention: purge daily failed",
			"source", src.Name(), "error", err)
	} else if n > 0 {
		e.logger.DebugContext(ctx, "retention: purged daily rollups",
			"source", src.Name(), "rows", n)
	}
}

// loop runs the periodic pass. Each tick triggers one RunOnce.
func (e *Engine) loop(ctx context.Context) {
	defer e.wg.Done()

	select {
	case <-ctx.Done():
		return
	case <-time.After(initialPassDelay):
	}

	ticker := time.NewTicker(hourlyPassInterval)
	defer ticker.Stop()

	for {
		// Run a pass on every tick (and immediately after the
		// initial delay).
		e.RunOnce(ctx)

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// hoursPerDay is the conversion factor used in day truncation.
const hoursPerDay = 24
