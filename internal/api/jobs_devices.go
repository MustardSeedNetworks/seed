package api

// jobs_devices.go registers the network device scan as a unified job kind
// (ADR-0005). Thin additive wrapper over the existing ctx-aware
// DeviceDiscovery.Scan behind an interface seam — no discovery-internal
// refactor. Unlike the legacy /security/devices/scan endpoint, the kind does NOT
// auto-trigger a vulnerability scan afterwards: in the jobs model that is a
// separate vuln-scan submission (one job = one operation). The scan is
// synchronous with no progress fraction, so the kind takes no progress callback.

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// deviceScanJobKind is the registered kind name for a network device scan.
const deviceScanJobKind = "device-scan"

// DeviceScanJobResult is the job Result for a device scan: the discovered
// devices and their count after the scan completes.
type DeviceScanJobResult struct {
	Devices []*discovery.DiscoveredDevice `json:"devices"`
	Count   int                           `json:"count"`
}

// deviceScanService is the slice of *discovery.DeviceDiscovery behaviour the
// kind needs.
type deviceScanService interface {
	Scan(ctx context.Context) error
	GetDevices() []*discovery.DiscoveredDevice
	Count() int
}

// newDeviceScanHandler returns the job Handler for the "device-scan" kind. It
// runs one scan (cancellable via the job context) and returns the discovered
// devices.
func newDeviceScanHandler(newSvc func() deviceScanService) jobs.Handler {
	return func(ctx context.Context, _ any, _ func(float64)) (any, error) {
		svc := newSvc()
		if err := svc.Scan(ctx); err != nil {
			return nil, err
		}
		return DeviceScanJobResult{Devices: svc.GetDevices(), Count: svc.Count()}, nil
	}
}

// registerDeviceScanKind registers the device-scan kind with an injectable
// service factory (the seam that makes the wiring testable without the network).
func (s *Server) registerDeviceScanKind(newSvc func() deviceScanService) {
	if err := s.jobsRunner().Register(deviceScanJobKind, newDeviceScanHandler(newSvc)); err != nil {
		logging.GetLogger().Error("failed to register device-scan job kind", "error", err)
	}
}
