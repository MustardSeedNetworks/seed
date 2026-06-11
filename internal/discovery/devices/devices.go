// Package devices holds the unified-discovery application (use-case) layer
// (ADR-0020). It owns the device-inventory orchestration that previously lived
// in the api.Server discovery-engine handlers — querying the discovered-device
// inventory, triggering quick/full/custom scans, looking up a device by MAC,
// and subscribing to the live discovery event stream — behind a narrow
// consumer-defined port over the discovery engine. Handlers keep transport
// concerns: request decode, scan-option validation, MAC extraction, SSE framing,
// and error-to-status mapping. The adapter satisfying the port lives in the
// composition root (internal/app) and resolves the engine lazily so a later-set
// engine (the api test harness) is honored; a nil engine degrades every method
// to ErrUnavailable rather than panicking.
package devices

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// Sentinel errors, mapped by handlers to the pre-strangle HTTP responses.
var (
	// ErrUnavailable signals the discovery engine is not wired (handlers map
	// it to 503, the golden-pinned degraded behavior).
	ErrUnavailable = errors.New("discovery engine not available")
	// ErrScanInProgress signals a scan is already running (handlers map it to 409).
	ErrScanInProgress = errors.New("a scan is already in progress")
	// ErrDeviceNotFound signals the requested MAC is unknown (handlers map it to 404).
	ErrDeviceNotFound = errors.New("device not found")
)

// Engine is the discovery-engine surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *discovery.Engine in
// internal/app. Available reports whether an engine is wired (resolved per call
// so the use-case can degrade gracefully); the remaining methods are only
// invoked once availability is confirmed.
type Engine interface {
	Available() bool
	Scanning() bool
	Devices() []*discovery.DiscoveredDevice
	Device(mac string) *discovery.DiscoveredDevice
	Stats() *discovery.EngineStats
	LastScan() *discovery.ScanResult
	Capabilities() map[string]bool
	Scan(ctx context.Context, opts *discovery.ScanOptions) (*discovery.ScanResult, error)
	SubscribeAll(handler func(*discovery.Event)) string
	Unsubscribe(id string)
}

// Snapshot is the use-case read model for the discovery-engine responses: the
// current device inventory, engine statistics, the most recent scan result, and
// the engine's capability set.
type Snapshot struct {
	Devices      []*discovery.DiscoveredDevice
	Stats        *discovery.EngineStats
	ScanResult   *discovery.ScanResult
	Capabilities map[string]bool
}

// Service is the unified-discovery use-case.
type Service struct {
	engine Engine
}

// NewService builds the use-case over its narrow engine dependency.
func NewService(engine Engine) *Service {
	return &Service{engine: engine}
}

// Available reports whether the discovery engine is wired. Handlers probe this
// before parsing a request body or path so the engine-nil 503 precedes any
// validation error (the order pinned by the engine handler tests).
func (s *Service) Available() bool { return s.engine.Available() }

// Scanning reports whether a scan is currently running. Handlers probe this to
// surface a 409 before parsing a scan body.
func (s *Service) Scanning() bool { return s.engine.Available() && s.engine.Scanning() }

// Snapshot returns the current inventory snapshot (no last-scan trigger).
func (s *Service) Snapshot() (Snapshot, error) {
	if !s.engine.Available() {
		return Snapshot{}, ErrUnavailable
	}
	return Snapshot{
		Devices:      s.engine.Devices(),
		Stats:        s.engine.Stats(),
		ScanResult:   s.engine.LastScan(),
		Capabilities: s.engine.Capabilities(),
	}, nil
}

// Scan applies the shared guards (availability, in-progress) then runs a scan
// with explicit options, returning the post-scan snapshot. The scan result is
// the one just produced (not the cached last-scan); a scan failure is returned
// verbatim so the handler can surface its message, while the guards return
// sentinels.
func (s *Service) Scan(ctx context.Context, opts *discovery.ScanOptions) (Snapshot, error) {
	if !s.engine.Available() {
		return Snapshot{}, ErrUnavailable
	}
	if s.engine.Scanning() {
		return Snapshot{}, ErrScanInProgress
	}
	result, err := s.engine.Scan(ctx, opts)
	if err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Devices:      s.engine.Devices(),
		Stats:        s.engine.Stats(),
		ScanResult:   result,
		Capabilities: s.engine.Capabilities(),
	}, nil
}

// QuickScan correlates existing data without fresh discovery (the engine's
// quick-scan defaults).
func (s *Service) QuickScan(ctx context.Context) (Snapshot, error) {
	return s.Scan(ctx, discovery.DefaultQuickScanOpts())
}

// FullScan runs fresh discovery, enrichment, and assessment (the engine's
// full-scan defaults).
func (s *Service) FullScan(ctx context.Context) (Snapshot, error) {
	return s.Scan(ctx, discovery.DefaultFullScanOpts())
}

// Stats returns the engine statistics.
func (s *Service) Stats() (*discovery.EngineStats, error) {
	if !s.engine.Available() {
		return nil, ErrUnavailable
	}
	return s.engine.Stats(), nil
}

// Capabilities returns the engine's capability set.
func (s *Service) Capabilities() (map[string]bool, error) {
	if !s.engine.Available() {
		return nil, ErrUnavailable
	}
	return s.engine.Capabilities(), nil
}

// Device returns a single device by MAC, in any common MAC format.
func (s *Service) Device(mac string) (*discovery.DiscoveredDevice, error) {
	if !s.engine.Available() {
		return nil, ErrUnavailable
	}
	device := s.engine.Device(mac)
	if device == nil {
		return nil, ErrDeviceNotFound
	}
	return device, nil
}

// Subscribe registers an event handler on the engine's stream and returns the
// subscription id, which the caller passes to Unsubscribe on teardown.
func (s *Service) Subscribe(handler func(*discovery.Event)) (string, error) {
	if !s.engine.Available() {
		return "", ErrUnavailable
	}
	return s.engine.SubscribeAll(handler), nil
}

// Unsubscribe removes a previously registered subscription.
func (s *Service) Unsubscribe(id string) { s.engine.Unsubscribe(id) }
