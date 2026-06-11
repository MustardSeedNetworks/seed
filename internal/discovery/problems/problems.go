// Package problems holds the network problem-detection application (use-case)
// layer (ADR-0020). It owns the orchestration that previously lived in the
// api.Server problem handlers — listing active problems, running a detection
// scan over the currently discovered devices, and reading/updating the
// detection thresholds — behind narrow consumer-defined ports over the problem
// detector and the device-discovery source. Handlers keep transport concerns:
// request decode, response mapping, and error-to-status mapping. The adapters
// satisfying the ports live in the composition root (internal/app) and resolve
// their collaborators lazily; a nil detector degrades every method to
// ErrUnavailable rather than panicking.
package problems

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// ErrUnavailable signals the problem detector is not wired (handlers map it to
// 503, the pre-strangle degraded behavior).
var ErrUnavailable = errors.New("problem detector not available")

// Detector is the problem-detector surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *discovery.ProblemDetector
// in internal/app. Available reports whether a detector is wired (resolved per
// call); the remaining methods are only invoked once availability is confirmed.
type Detector interface {
	Available() bool
	ActiveProblems() []discovery.NetworkProblem
	Summary() *discovery.ProblemSummary
	Scan(ctx context.Context, devices []*discovery.DiscoveredDevice) (*discovery.ProblemDetectionResult, error)
	Thresholds() discovery.ProblemThresholds
	SetThresholds(t discovery.ProblemThresholds)
}

// DeviceSource supplies the discovered devices a detection scan runs against. A
// nil source yields no devices (the scan still runs over an empty set), matching
// the handler's historic nil-guard.
type DeviceSource interface {
	Devices() []*discovery.DiscoveredDevice
}

// Active is the use-case read model for the current-problems response.
type Active struct {
	Problems []discovery.NetworkProblem
	Summary  *discovery.ProblemSummary
}

// Service is the problem-detection use-case.
type Service struct {
	detector Detector
	devices  DeviceSource
}

// NewService builds the use-case over the detector and the device source.
func NewService(detector Detector, devices DeviceSource) *Service {
	return &Service{detector: detector, devices: devices}
}

// Available reports whether the problem detector is wired. Handlers probe this
// once before branching by method so a nil detector returns 503 uniformly.
func (s *Service) Available() bool { return s.detector.Available() }

// Active returns the problems detected on the most recent scan plus the summary.
func (s *Service) Active() (Active, error) {
	if !s.detector.Available() {
		return Active{}, ErrUnavailable
	}
	return Active{
		Problems: s.detector.ActiveProblems(),
		Summary:  s.detector.Summary(),
	}, nil
}

// Scan runs problem detection over the currently discovered devices and returns
// the full detection result.
func (s *Service) Scan(ctx context.Context) (*discovery.ProblemDetectionResult, error) {
	if !s.detector.Available() {
		return nil, ErrUnavailable
	}
	return s.detector.Scan(ctx, s.devices.Devices())
}

// Thresholds returns the current detection thresholds.
func (s *Service) Thresholds() (discovery.ProblemThresholds, error) {
	if !s.detector.Available() {
		return discovery.ProblemThresholds{}, ErrUnavailable
	}
	return s.detector.Thresholds(), nil
}

// SetThresholds updates the detection thresholds.
func (s *Service) SetThresholds(t discovery.ProblemThresholds) error {
	if !s.detector.Available() {
		return ErrUnavailable
	}
	s.detector.SetThresholds(t)
	return nil
}
