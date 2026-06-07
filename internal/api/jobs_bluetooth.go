package api

// jobs_bluetooth.go registers the Bluetooth scan as a unified job kind
// (ADR-0005). Thin additive wrapper over the existing ctx-aware scanner behind
// an interface seam — no discovery-internal refactor. The scan is synchronous
// and exposes no progress fraction, so the kind takes no progress callback. The
// legacy /security/bluetooth/scan endpoint is unchanged (retire at Phase-7).

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// bluetoothScanJobKind is the registered kind name for a Bluetooth scan.
const bluetoothScanJobKind = "bluetooth-scan"

// bluetoothScannerService is the slice of *discovery.BluetoothScanner behaviour
// the kind needs.
type bluetoothScannerService interface {
	Scan(ctx context.Context) (*discovery.BluetoothScanResult, error)
	GetStats() *discovery.BluetoothDiscoveryStats
}

// newBluetoothScanHandler returns the job Handler for the "bluetooth-scan" kind.
// It runs one scan (cancellable via the job context) and returns the same
// BluetoothScanResponse the legacy endpoint produces.
func newBluetoothScanHandler(newScanner func() bluetoothScannerService) jobs.Handler {
	return func(ctx context.Context, _ any, _ func(float64)) (any, error) {
		scanner := newScanner()
		result, err := scanner.Scan(ctx)
		if err != nil {
			return nil, err
		}
		return toBluetoothScanResponse(result, scanner.GetStats()), nil
	}
}

// registerBluetoothScanKind registers the bluetooth-scan kind with an injectable
// scanner factory (the seam that makes the wiring testable without hardware).
func (s *Server) registerBluetoothScanKind(newScanner func() bluetoothScannerService) {
	if err := s.jobsRunner().Register(bluetoothScanJobKind, newBluetoothScanHandler(newScanner)); err != nil {
		logging.GetLogger().Error("failed to register bluetooth-scan job kind", "error", err)
	}
}
