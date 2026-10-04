// Package telemetry records seed's own link, gateway and DNS health as time
// series in the metrics table, so its hourly and daily rollups have data
// (#2623). The live cards measure the same things, but only while a browser
// is connected and without keeping anything.
//
// The engine owns the cadence. What a sample contains is the composition
// root's [Sampler]; where it goes is the [Store] port, which internal/database
// implements, so this package stays persistence-free.
package telemetry

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Interval is how often the engine samples. A minute gives the hourly
// rollup sixty points and costs one gateway ping burst and one DNS lookup.
const Interval = time.Minute

// TargetKind is the metrics target_kind of a telemetry point; its target_id
// is the interface name.
const TargetKind = "interface"

// Metric types. Each is a series per interface.
const (
	GatewayLatencyMs = "gateway_latency_ms"
	GatewayLossPct   = "gateway_loss_pct"
	GatewayReachable = "gateway_reachable"
	DNSQueryMs       = "dns_query_ms"
	DNSResolved      = "dns_resolved"
	LinkCarrier      = "link_carrier"
)

// Units of the metric types above.
const (
	UnitMs      = "ms"
	UnitPercent = "%"
	UnitBool    = "bool"
)

// Point is one value of one series.
type Point struct {
	Type  string
	Value float64
	Unit  string
}

// Sample is everything measured on one interface at one instant.
type Sample struct {
	Interface string
	At        time.Time
	Points    []Point
}

// Sampler measures the selected interface now.
type Sampler interface {
	Sample(ctx context.Context) Sample
}

// Store persists a sample.
type Store interface {
	RecordTelemetry(ctx context.Context, s Sample) error
}

// Engine samples on a fixed cadence and stores what it measures.
type Engine struct {
	sampler  Sampler
	store    Store
	interval time.Duration
	logger   *slog.Logger

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New returns an engine that samples every interval.
func New(sampler Sampler, store Store, interval time.Duration, logger *slog.Logger) *Engine {
	return &Engine{sampler: sampler, store: store, interval: interval, logger: logger}
}

// Name implements engine.Engine.
func (*Engine) Name() string { return "telemetry" }

// Start samples once immediately and then every interval. It is a no-op
// when already started.
func (e *Engine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	e.cancel = cancel
	e.started = true

	e.wg.Add(1)
	go e.loop(loopCtx)
	e.logger.InfoContext(ctx, "telemetry engine started", "interval", e.interval)
	return nil
}

// Stop ends the loop and waits, up to ctx's deadline, for an in-flight
// sample to finish.
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
	return nil
}

func (e *Engine) loop(ctx context.Context) {
	defer e.wg.Done()
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()
	for {
		e.record(ctx)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (e *Engine) record(ctx context.Context) {
	s := e.sampler.Sample(ctx)
	if len(s.Points) == 0 {
		return
	}
	if err := e.store.RecordTelemetry(ctx, s); err != nil && ctx.Err() == nil {
		e.logger.WarnContext(ctx, "telemetry: record failed", "interface", s.Interface, "error", err)
	}
}
