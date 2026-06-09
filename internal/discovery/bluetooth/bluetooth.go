// Package bluetooth holds the Bluetooth-discovery application (use-case) layer
// (ADR-0020). It owns the orchestration that previously lived in the api.Server
// Bluetooth handlers — triggering an active scan, returning the most recent
// devices, reading aggregate statistics, and reporting adapter status — behind a
// narrow consumer-defined port over the Bluetooth scanner. Handlers keep
// transport concerns: response mapping (the flat wire DTOs), time formatting,
// and error-to-status mapping. The adapter satisfying the port lives in the
// composition root (internal/app) and resolves the scanner lazily; a nil scanner
// degrades the scan/devices/stats methods to ErrUnavailable, while Status reports
// availability=false rather than erroring (matching the pre-strangle behavior).
package bluetooth

import (
	"context"
	"errors"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// ErrUnavailable signals the Bluetooth scanner is not wired (handlers map it to
// 503, the pre-strangle degraded behavior).
var ErrUnavailable = errors.New("bluetooth scanner not available")

// Scanner is the Bluetooth-scanner surface the use-case drives, defined at the
// consumer (ADR-0020) and satisfied by an adapter over *enumerate.BluetoothScanner
// in internal/app. Available reports whether a scanner is wired (resolved per
// call); the remaining methods are only invoked once availability is confirmed.
type Scanner interface {
	Available() bool
	Scan(ctx context.Context) (*discovery.BluetoothScanResult, error)
	LastScan() *discovery.BluetoothScanResult
	Stats() *discovery.BluetoothDiscoveryStats
}

// ScanResult pairs a scan's result with the post-scan aggregate statistics, the
// two pieces the scan response is assembled from.
type ScanResult struct {
	Result *discovery.BluetoothScanResult
	Stats  *discovery.BluetoothDiscoveryStats
}

// Status is the use-case read model for the adapter-status response.
type Status struct {
	Available bool
	LastScan  *discovery.BluetoothScanResult
}

// Service is the Bluetooth-discovery use-case.
type Service struct {
	scanner Scanner
}

// NewService builds the use-case over its narrow scanner dependency.
func NewService(scanner Scanner) *Service {
	return &Service{scanner: scanner}
}

// Scan triggers an active Bluetooth scan and returns the result plus the
// post-scan statistics.
func (s *Service) Scan(ctx context.Context) (ScanResult, error) {
	if !s.scanner.Available() {
		return ScanResult{}, ErrUnavailable
	}
	result, err := s.scanner.Scan(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	return ScanResult{Result: result, Stats: s.scanner.Stats()}, nil
}

// Devices returns the most recent scan (nil if no scan has run), letting the
// handler render an empty device list for the no-scan case.
func (s *Service) Devices() (*discovery.BluetoothScanResult, error) {
	if !s.scanner.Available() {
		return nil, ErrUnavailable
	}
	return s.scanner.LastScan(), nil
}

// Stats returns the aggregate Bluetooth discovery statistics.
func (s *Service) Stats() (*discovery.BluetoothDiscoveryStats, error) {
	if !s.scanner.Available() {
		return nil, ErrUnavailable
	}
	return s.scanner.Stats(), nil
}

// Status reports adapter availability and the most recent scan. Unlike the other
// methods it never errors: an absent scanner is a valid "unavailable" status.
func (s *Service) Status() Status {
	if !s.scanner.Available() {
		return Status{Available: false}
	}
	return Status{Available: true, LastScan: s.scanner.LastScan()}
}
