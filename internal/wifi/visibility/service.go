// Package visibility is the runtime orchestrator for Wi-Fi airspace visibility
// and anomaly detection. It owns the live cross-referenced airspace model
// (internal/wifi/airspace) and the detector that evaluates the Wi-Fi rules
// (internal/wifi/anomaly) against it. It is a *producer* into the shared,
// server-owned anomaly Coordinator (internal/anomaly, ADR-0029): it no longer
// owns an engine — detections are stamped source=wifi and folded into the one
// engine every producer shares, persisted through the unified store.
//
// A capture source (W3, libpcap/monitor-mode, built separately and CGO-tagged)
// feeds decoded 802.11 frames in via Ingest; this package is itself CGO-free and
// frame-source-agnostic, so it builds and tests everywhere. Without capture, an
// ordinary managed-mode scan (SetScanSource) feeds the BSSes it sees, so the
// rules that read beacon information elements run on any radio (#2351). Run
// drives the scan and evaluation loops (detector → Coordinator), and Tree/Status
// are the read model
// the API layer (W5b) serves Pro-gated; the anomaly list itself is read from the
// store (ADR-0029 §4), not from here.
package visibility

import (
	"context"
	"log/slog"
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
	// defaultScanInterval is how often the loop scans for neighbour networks
	// while no capture source is feeding frames. Each scan takes the radio
	// off-channel for a few seconds, so this stays well above the evaluation
	// cadence; the retention window still covers several missed scans.
	defaultScanInterval = time.Minute
)

// ScanSource performs one managed-mode scan and returns the BSSes it saw, with
// lowercase BSSIDs. A scan carries no station or frame data, so those view
// fields stay empty.
type ScanSource func() ([]airspace.BSSView, error)

// Status is the read-model summary of the visibility service: whether a capture
// source is feeding it, and the current entity/anomaly counts. Pure data with
// camelCase wire tags (ADR-0010) so the API layer can serialize it directly.
type Status struct {
	CaptureActive bool      `json:"captureActive"`
	Source        string    `json:"source,omitempty"`
	LastScan      time.Time `json:"lastScan,omitzero"`
	SSIDs         int       `json:"ssids"`
	APs           int       `json:"aps"`
	BSSes         int       `json:"bsses"`
	Stations      int       `json:"stations"`
	Anomalies     int       `json:"anomalies"`
	LastEvaluated time.Time `json:"lastEvaluated,omitzero"`
	// NeedsCapture lists the rules that cannot run while no capture source
	// feeds frames, so their absence from the anomaly list reads as
	// unavailable rather than clear.
	NeedsCapture []wifianomaly.Rule `json:"needsCapture,omitempty"`
}

// Service owns the live airspace, the Wi-Fi detector, and the evaluation loop. It
// is a producer into the shared anomaly Coordinator. All exported methods are
// safe for concurrent use.
type Service struct {
	air      *airspace.Airspace
	detector *wifianomaly.Detector
	// coordinator is the shared, server-owned anomaly Coordinator (ADR-0029).
	// Wi-Fi detections persist through it under source=wifi. Nil leaves the
	// service airspace-only (Ingest/Tree/Status still work; no anomaly
	// detection) — the cmd layer builds the Service before the server owns the
	// Coordinator, then injects it via SetCoordinator.
	coordinator *anomaly.Coordinator
	logger      *slog.Logger

	evalInterval time.Duration
	scanInterval time.Duration
	retention    time.Duration

	mu            sync.Mutex
	source        string
	lastEvaluated time.Time
	scanSource    ScanSource
	// scanned is the most recent scan's BSSes, merged under the captured
	// airspace by Tree and dropped once it is older than the retention window.
	scanned  []airspace.BSSView
	lastScan time.Time
}

// Option configures a Service.
type Option func(*config)

type config struct {
	evalInterval time.Duration
	scanInterval time.Duration
	retention    time.Duration
	detector     *wifianomaly.Detector
	coordinator  *anomaly.Coordinator
	logger       *slog.Logger
}

// WithEvalInterval sets the background evaluation cadence.
func WithEvalInterval(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.evalInterval = d
		}
	}
}

// WithScanInterval sets the cadence of the managed-mode scan feed.
func WithScanInterval(d time.Duration) Option {
	return func(c *config) {
		if d > 0 {
			c.scanInterval = d
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

// WithCoordinator injects the shared, server-owned anomaly Coordinator
// (ADR-0029) at construction. Production builds the Service in the cmd layer
// before the server owns the Coordinator and injects it later via
// SetCoordinator; this option is for tests that have the Coordinator up front.
func WithCoordinator(coord *anomaly.Coordinator) Option {
	return func(c *config) { c.coordinator = coord }
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

// New builds a visibility service. The Wi-Fi anomaly catalog now lives on the
// shared, server-owned engine (ADR-0029), so construction cannot fail; the
// Coordinator is injected via WithCoordinator (tests) or SetCoordinator (the
// server, after it owns the merged engine).
func New(opts ...Option) *Service {
	cfg := config{
		evalInterval: defaultEvalInterval,
		scanInterval: defaultScanInterval,
		retention:    defaultRetention,
	}
	for _, o := range opts {
		o(&cfg)
	}
	det := cfg.detector
	if det == nil {
		det = wifianomaly.NewDetector()
	}
	logger := cfg.logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{
		air:          airspace.New(),
		detector:     det,
		coordinator:  cfg.coordinator,
		logger:       logger,
		evalInterval: cfg.evalInterval,
		scanInterval: cfg.scanInterval,
		retention:    cfg.retention,
	}
}

// SetCoordinator injects the shared anomaly Coordinator after construction
// (ADR-0029): the cmd layer builds the Service before the server owns the merged
// engine, so the server wires the Coordinator in during init, before Start. Wi-Fi
// detections then persist through it under source=wifi.
func (s *Service) SetCoordinator(coord *anomaly.Coordinator) {
	s.mu.Lock()
	s.coordinator = coord
	s.mu.Unlock()
}

// SetScanSource injects the managed-mode scan feed. The server owns the Wi-Fi
// scanner, so it wires this during init, like SetCoordinator.
func (s *Service) SetScanSource(src ScanSource) {
	s.mu.Lock()
	s.scanSource = src
	s.mu.Unlock()
}

// IngestScan replaces the scan snapshot with the BSSes one scan saw at `at`.
// A BSS the scan no longer reports disappears with it.
func (s *Service) IngestScan(views []airspace.BSSView, at time.Time) {
	s.mu.Lock()
	s.scanned = views
	s.lastScan = at
	s.mu.Unlock()
}

// Ingest folds one decoded frame into the airspace at observation time `at`.
// The capture source (W3) calls this for every frame; a nil frame is a no-op.
func (s *Service) Ingest(f *dot11.Frame, at time.Time) {
	s.air.Observe(f, at)
}

// Evaluate ages out stale entities, runs the Wi-Fi rules over the current
// airspace, and folds the resulting detections into the engine (which coalesces,
// escalates on recurrence, and ages out resolved anomalies). When a store is
// configured it persists through the Coordinator: write-through on a material
// change, a single batched Flush for the tick's recurrences, and resolve-on-prune.
func (s *Service) Evaluate(ctx context.Context, at time.Time) {
	cutoff := at.Add(-s.retention)
	s.air.Prune(cutoff)
	s.mu.Lock()
	if s.lastScan.Before(cutoff) {
		s.scanned = nil
	}
	s.mu.Unlock()

	// Airspace-only when no Coordinator is wired (ADR-0029): Ingest/Tree/Status
	// stay live, but anomaly detection needs the shared engine.
	if coord := s.snapshotCoordinator(); coord != nil {
		s.evaluatePersistent(ctx, coord, at, cutoff)
	}

	s.mu.Lock()
	s.lastEvaluated = at
	s.mu.Unlock()
}

// evaluatePersistent runs the detection + persistence path through the shared
// Coordinator. Store errors are logged, not fatal — the in-memory engine stays
// authoritative and the next tick re-persists.
func (s *Service) evaluatePersistent(ctx context.Context, coord *anomaly.Coordinator, at, cutoff time.Time) {
	for _, d := range s.detector.Detect(s.Tree()) {
		// Stamp the producer source at the hand-off (ADR-0029 §2); the Wi-Fi
		// detector stays source-agnostic.
		d.Source = anomaly.SourceWiFi
		if err := coord.Observe(ctx, d, at); err != nil {
			s.logger.WarnContext(ctx, "anomaly persist (observe) failed",
				"defKey", d.DefKey, "error", err)
		}
	}
	if err := coord.Flush(ctx); err != nil {
		s.logger.WarnContext(ctx, "anomaly persist (flush) failed", "error", err)
	}
	// Source-scoped prune (ADR-0029 §3): resolve only Wi-Fi instances idle past
	// the 5 m retention window, never another producer's still-live instances.
	if _, err := coord.Prune(ctx, anomaly.SourceWiFi, cutoff); err != nil {
		s.logger.WarnContext(ctx, "anomaly persist (prune) failed", "error", err)
	}
}

// snapshotCoordinator reads the injected Coordinator under the lock so a
// concurrent SetCoordinator (server init) is race-free.
func (s *Service) snapshotCoordinator() *anomaly.Coordinator {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.coordinator
}

// Tree returns the current cross-referenced SSID → AP → BSSID → client view:
// the captured airspace plus every scanned BSS capture has not seen. A captured
// BSS wins because it carries the stations and frame counts a scan cannot.
func (s *Service) Tree() []airspace.SSIDGroup {
	captured := s.air.Tree()
	s.mu.Lock()
	scanned := s.scanned
	s.mu.Unlock()
	if len(scanned) == 0 {
		return captured
	}

	var views []airspace.BSSView
	seen := make(map[string]struct{})
	for _, g := range captured {
		for _, ap := range g.APs {
			for _, b := range ap.BSSes {
				views = append(views, b)
				seen[b.BSSID] = struct{}{}
			}
		}
	}
	for _, b := range scanned {
		if _, ok := seen[b.BSSID]; !ok {
			views = append(views, b)
		}
	}
	return airspace.TreeFromBSSViews(views)
}

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
	tree := s.Tree()
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
	src, last, scanned, coord := s.source, s.lastEvaluated, s.lastScan, s.coordinator
	s.mu.Unlock()

	// Source-scoped count (ADR-0029 §4): the shared engine holds every producer's
	// instances, so Wi-Fi reports only its own — engine.Len would over-count.
	anomalies := 0
	if coord != nil {
		anomalies = coord.Engine().LenBySource(anomaly.SourceWiFi)
	}

	st := Status{
		CaptureActive: src != "",
		Source:        src,
		LastScan:      scanned,
		SSIDs:         len(tree),
		APs:           len(apKeys),
		BSSes:         bsses,
		Stations:      stations,
		Anomalies:     anomalies,
		LastEvaluated: last,
	}
	if !st.CaptureActive {
		st.NeedsCapture = wifianomaly.CaptureOnlyRules()
	}
	return st
}

// Run scans on the scan interval and evaluates the airspace on the eval
// interval until ctx is cancelled. It blocks: the loop is a supervised worker
// (#2748), and the `go s.loop` it replaced ran outside the supervisor's recover.
func (s *Service) Run(ctx context.Context) error {
	// Load-on-start is server-owned and happens once on the shared Coordinator
	// before any producer observes (ADR-0029 §5), so the service does not load
	// here — it would re-load the merged engine a second time.
	ticker := time.NewTicker(s.evalInterval)
	defer ticker.Stop()
	scanTicker := time.NewTicker(s.scanInterval)
	defer scanTicker.Stop()
	s.scan(ctx)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			s.Evaluate(ctx, time.Now())
		case <-scanTicker.C:
			s.scan(ctx)
		}
	}
}

// scan runs one managed-mode scan into the snapshot. It stands down while a
// capture source feeds frames: the radio is in monitor mode then, and the
// captured airspace already holds everything a scan would report. A failed
// scan keeps the previous snapshot until the retention window expires it.
func (s *Service) scan(ctx context.Context) {
	s.mu.Lock()
	src, capturing := s.scanSource, s.source != ""
	s.mu.Unlock()
	if src == nil || capturing {
		return
	}
	views, err := src()
	if err != nil {
		// Debug, not Warn: a host with no wireless adapter fails every scan.
		s.logger.DebugContext(ctx, "wifi airspace scan failed", "error", err)
		return
	}
	s.IngestScan(views, time.Now())
}
