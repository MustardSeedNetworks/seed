package probeanomaly

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/probe"
)

const (
	// defaultFlushInterval is how often the maintenance tick batches recurrence
	// writes and ages out recovered probes.
	defaultFlushInterval = 30 * time.Second
	// defaultResolveWindow is how long a probe must go without re-breaching before
	// its anomaly is considered resolved. It MUST exceed any probe's interval so a
	// still-failing probe (which re-breaches every interval, refreshing lastSeen)
	// stays active; a recovered probe resolves once silent this long (ADR-0025:
	// push-model resolution = TTL-on-silence).
	defaultResolveWindow = 15 * time.Minute
)

// Producer is the long-lived active-monitoring anomaly source (ADR-0025). It
// subscribes to the probe engine's ResultEvent channel, maps threshold breaches
// onto anomaly detections, and persists them through a Coordinator under
// source=probe (write-through on a material change, batched recurrence on a
// periodic Flush, resolve-on-Prune when a probe goes silent). It implements
// engine.Engine so the lifecycle registry drives Start/Stop alongside the probe
// engine itself.
type Producer struct {
	events <-chan probe.ResultEvent
	coord  *anomaly.Coordinator
	logger *slog.Logger

	flushInterval time.Duration
	resolveWindow time.Duration

	mu     sync.Mutex
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// Option configures a Producer.
type Option func(*config)

type config struct {
	flushInterval time.Duration
	resolveWindow time.Duration
	logger        *slog.Logger
}

// WithFlushInterval sets the maintenance cadence (Flush + Prune). Non-positive
// values are ignored.
func WithFlushInterval(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.flushInterval = d
		}
	}
}

// WithResolveWindow sets the silence window after which a non-re-breaching probe
// is resolved. Non-positive values are ignored.
func WithResolveWindow(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.resolveWindow = d
		}
	}
}

// WithLogger sets the logger for persistence diagnostics. Defaults to
// [slog.Default].
func WithLogger(l *slog.Logger) Option {
	return func(c *config) {
		if l != nil {
			c.logger = l
		}
	}
}

// New builds a probe anomaly producer over the probe engine's event channel
// (from Engine.Subscribe) and the unified anomaly store. It errors only if the
// probe anomaly catalog is malformed — a programming error surfaced at startup,
// never at runtime.
func New(events <-chan probe.ResultEvent, store anomaly.Store, opts ...Option) (*Producer, error) {
	cfg := config{flushInterval: defaultFlushInterval, resolveWindow: defaultResolveWindow}
	for _, o := range opts {
		o(&cfg)
	}
	cat, err := Catalog()
	if err != nil {
		return nil, err
	}
	logger := cfg.logger
	if logger == nil {
		logger = slog.Default()
	}
	engine := anomaly.NewEngine(cat)
	coord := anomaly.NewCoordinator(engine, store)
	return &Producer{
		events:        events,
		coord:         coord,
		logger:        logger,
		flushInterval: cfg.flushInterval,
		resolveWindow: cfg.resolveWindow,
	}, nil
}

// Name implements engine.Engine.
func (*Producer) Name() string { return "probe-anomaly" }

// Start loads active instances from the store (so a restart keeps coalescing onto
// persisted anomalies) and launches the consume + maintenance goroutines. It is
// idempotent: a second call while running is a no-op.
func (p *Producer) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.cancel != nil {
		p.mu.Unlock()
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.mu.Unlock()

	// Load-on-start (ADR-0021). Best-effort — a store error degrades to a
	// cold-start, not a failed Start.
	if n, err := p.coord.Load(ctx); err != nil {
		p.logger.WarnContext(ctx, "probe anomaly load-on-start failed", "error", err)
	} else if n > 0 {
		p.logger.InfoContext(ctx, "restored persisted probe anomalies", "count", n)
	}

	p.wg.Add(1)
	go p.consume(loopCtx)
	p.wg.Add(1)
	go p.maintain(loopCtx)
	return nil
}

// Stop cancels the goroutines, waits for them to drain, and performs a final
// Flush so the last recurrence batch is durable. Safe to call repeatedly and
// safe when never started.
func (p *Producer) Stop(ctx context.Context) error {
	p.mu.Lock()
	cancel := p.cancel
	p.cancel = nil
	p.mu.Unlock()
	if cancel == nil {
		return nil
	}
	cancel()
	p.wg.Wait()
	if err := p.coord.Flush(ctx); err != nil {
		p.logger.WarnContext(ctx, "probe anomaly final flush failed", "error", err)
	}
	return nil
}

// consume drains the probe event channel, folding each ResultEvent's breaches
// into the engine. It exits on context cancellation or a closed channel.
func (p *Producer) consume(ctx context.Context) {
	defer p.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case re, ok := <-p.events:
			if !ok {
				return
			}
			p.observe(ctx, re, time.Now().UTC())
		}
	}
}

// observe folds one event's detections into the coordinator at time at. Store
// errors are logged, not fatal — the in-memory engine stays authoritative and
// the next Flush re-persists. Separated from consume so it is unit-testable
// without the goroutine.
//
// A clean run (no breaches) is the recovery fast-path (ADR-0025 §3): the probe
// ran and nothing tripped, so any active anomalies for that probe are resolved
// immediately rather than aging out over the silence window. The maintenance
// Prune remains the backstop for a probe that goes fully silent (disabled or
// deleted) and never emits a clean result.
func (p *Producer) observe(ctx context.Context, re probe.ResultEvent, at time.Time) {
	dets := Detections(re)
	if len(dets) == 0 {
		p.resolveClean(ctx, re.Result.ProbeID, at)
		return
	}
	for _, d := range dets {
		// Stamp the producer source at the hand-off (ADR-0029 §2).
		d.Source = anomaly.SourceProbe
		if err := p.coord.Observe(ctx, d, at); err != nil {
			p.logger.WarnContext(ctx, "probe anomaly persist (observe) failed",
				"defKey", d.DefKey, "probeID", d.Subject.ID, "error", err)
		}
	}
}

// resolveClean resolves every active anomaly for a recovered probe. A blank
// probe ID (a malformed event) is skipped — there is no subject to key on.
func (p *Producer) resolveClean(ctx context.Context, probeID string, at time.Time) {
	if probeID == "" {
		return
	}
	subject := anomaly.SubjectRef{Kind: anomaly.SubjectProbe, ID: probeID}
	if _, err := p.coord.ResolveSubject(ctx, subject, at); err != nil {
		p.logger.WarnContext(ctx, "probe anomaly persist (resolve-on-clean) failed",
			"probeID", probeID, "error", err)
	}
}

// maintain batches recurrence writes and ages out recovered probes on a fixed
// tick. It exits on context cancellation.
func (p *Producer) maintain(ctx context.Context) {
	defer p.wg.Done()
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.flushAndPrune(ctx, time.Now().UTC())
		}
	}
}

// flushAndPrune persists pending recurrence updates and resolves probes that have
// not re-breached within the resolve window (cutoff = now − resolveWindow).
// Separated from maintain so it is unit-testable without the goroutine.
func (p *Producer) flushAndPrune(ctx context.Context, now time.Time) {
	if err := p.coord.Flush(ctx); err != nil {
		p.logger.WarnContext(ctx, "probe anomaly persist (flush) failed", "error", err)
	}
	if _, err := p.coord.Prune(ctx, anomaly.SourceProbe, now.Add(-p.resolveWindow)); err != nil {
		p.logger.WarnContext(ctx, "probe anomaly persist (prune) failed", "error", err)
	}
}

// Anomalies returns the current probe anomaly stream, most urgent first — the
// in-memory projection (the persisted view is read via the store).
func (p *Producer) Anomalies() []anomaly.Anomaly { return p.coord.Engine().Snapshot() }
