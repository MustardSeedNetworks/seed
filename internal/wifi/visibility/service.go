// Package visibility is the runtime orchestrator for Wi-Fi airspace visibility
// and anomaly detection. It owns the live cross-referenced airspace model
// (internal/wifi/airspace), a persistent anomaly engine (internal/anomaly) seeded
// with the Wi-Fi rule catalog (internal/wifi/anomaly), and the detector that
// evaluates one against the other.
//
// A capture source (W3, libpcap/monitor-mode, built separately and CGO-tagged)
// feeds decoded 802.11 frames in via Ingest; this package is itself CGO-free and
// frame-source-agnostic, so it builds and tests everywhere. Start runs a periodic
// evaluation loop (detector → engine), and Tree/Anomalies/Status are the read
// model the API layer (W5b) serves Pro-gated.
package visibility

import (
	"context"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/airspace"
	wifianomaly "github.com/MustardSeedNetworks/seed/internal/wifi/anomaly"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
)

const (
	// defaultEvalInterval is how often the background loop re-evaluates the
	// airspace for anomalies.
	defaultEvalInterval = 5 * time.Second
	// defaultRetention is how long a BSS/station/anomaly survives without being
	// re-observed before it is pruned (the condition is considered resolved).
	defaultRetention = 5 * time.Minute
)

// Status is the read-model summary of the visibility service: whether a capture
// source is feeding it, and the current entity/anomaly counts. Pure data with
// camelCase wire tags (ADR-0010) so the API layer can serialize it directly.
type Status struct {
	CaptureActive bool      `json:"captureActive"`
	Source        string    `json:"source,omitempty"`
	SSIDs         int       `json:"ssids"`
	APs           int       `json:"aps"`
	BSSes         int       `json:"bsses"`
	Stations      int       `json:"stations"`
	Anomalies     int       `json:"anomalies"`
	LastEvaluated time.Time `json:"lastEvaluated,omitzero"`
}

// Service owns the live airspace, the anomaly engine, and the evaluation loop.
// All exported methods are safe for concurrent use.
type Service struct {
	air      *airspace.Airspace
	engine   *anomaly.Engine
	detector *wifianomaly.Detector

	evalInterval time.Duration
	retention    time.Duration

	mu            sync.Mutex
	source        string
	lastEvaluated time.Time
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// Option configures a Service.
type Option func(*config)

type config struct {
	evalInterval time.Duration
	retention    time.Duration
	detector     *wifianomaly.Detector
	capabilities []string
}

// WithEvalInterval sets the background evaluation cadence.
func WithEvalInterval(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.evalInterval = d
		}
	}
}

// WithRetention sets how long entities/anomalies persist without re-observation.
func WithRetention(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.retention = d
		}
	}
}

// WithDetector overrides the default Wi-Fi detector (e.g. tuned thresholds).
func WithDetector(d *wifianomaly.Detector) Option {
	return func(c *config) {
		if d != nil {
			c.detector = d
		}
	}
}

// WithCapabilities registers platform capabilities for the engine's auto
// follow-ups (e.g. wifianomaly.CapActiveTest when active testing is available).
func WithCapabilities(caps ...string) Option {
	return func(c *config) { c.capabilities = append(c.capabilities, caps...) }
}

// New builds a visibility service. It errors only if the Wi-Fi anomaly catalog
// is malformed — a programming error surfaced at startup, never at runtime.
func New(opts ...Option) (*Service, error) {
	cfg := config{evalInterval: defaultEvalInterval, retention: defaultRetention}
	for _, o := range opts {
		o(&cfg)
	}
	cat, err := wifianomaly.Catalog()
	if err != nil {
		return nil, err
	}
	det := cfg.detector
	if det == nil {
		det = wifianomaly.NewDetector()
	}
	return &Service{
		air:          airspace.New(),
		engine:       anomaly.NewEngine(cat, anomaly.WithCapabilities(cfg.capabilities...)),
		detector:     det,
		evalInterval: cfg.evalInterval,
		retention:    cfg.retention,
	}, nil
}

// Ingest folds one decoded frame into the airspace at observation time `at`.
// The capture source (W3) calls this for every frame; a nil frame is a no-op.
func (s *Service) Ingest(f *dot11.Frame, at time.Time) {
	s.air.Observe(f, at)
}

// Evaluate ages out stale entities, runs the Wi-Fi rules over the current
// airspace, and folds the resulting detections into the persistent engine
// (which coalesces, escalates on recurrence, and ages out resolved anomalies).
func (s *Service) Evaluate(at time.Time) {
	cutoff := at.Add(-s.retention)
	s.air.Prune(cutoff)
	for _, d := range s.detector.Detect(s.air.Tree()) {
		// Observe only errors on an uncatalogued def or invalid severity; the
		// detector emits neither, so the error is structurally impossible here.
		_ = s.engine.Observe(d, at)
	}
	s.engine.Prune(cutoff)

	s.mu.Lock()
	s.lastEvaluated = at
	s.mu.Unlock()
}

// Tree returns the current cross-referenced SSID → AP → BSSID → client view.
func (s *Service) Tree() []airspace.SSIDGroup { return s.air.Tree() }

// Anomalies returns the current anomaly stream, most urgent first.
func (s *Service) Anomalies() []anomaly.Anomaly { return s.engine.Snapshot() }

// SetSource records that a named capture source is actively feeding frames.
func (s *Service) SetSource(name string) {
	s.mu.Lock()
	s.source = name
	s.mu.Unlock()
}

// ClearSource records that capture has stopped (graceful degrade to OS scan).
func (s *Service) ClearSource() {
	s.mu.Lock()
	s.source = ""
	s.mu.Unlock()
}

// Status summarizes the live read model.
func (s *Service) Status() Status {
	tree := s.air.Tree()
	apKeys := make(map[string]struct{})
	bsses, stations := 0, 0
	for _, g := range tree {
		bsses += g.BSSCount
		stations += g.StationCount
		for _, ap := range g.APs {
			apKeys[ap.Key] = struct{}{}
		}
	}

	s.mu.Lock()
	src, last := s.source, s.lastEvaluated
	s.mu.Unlock()

	return Status{
		CaptureActive: src != "",
		Source:        src,
		SSIDs:         len(tree),
		APs:           len(apKeys),
		BSSes:         bsses,
		Stations:      stations,
		Anomalies:     s.engine.Len(),
		LastEvaluated: last,
	}
}

// Start launches the background evaluation loop. It is idempotent: a second
// call while running is a no-op.
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return nil
	}
	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	s.wg.Add(1)
	go s.loop(loopCtx)
	return nil
}

func (s *Service) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(s.evalInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Evaluate(time.Now())
		}
	}
}

// Stop halts the background loop and waits for it to exit. Idempotent.
func (s *Service) Stop() error {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	s.wg.Wait()
	return nil
}
