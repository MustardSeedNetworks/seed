package api

// jobs_vuln.go registers the vulnerability scan as a unified job kind
// (ADR-0005), the first discovery-coupled long-op on the runner. It is a thin
// additive wrapper over the EXISTING public scanner + device-registry methods
// (both already ctx-aware) behind an interface seam — no discovery-internal
// refactor. The legacy /security/vulnerabilities/scan + /status + /results
// endpoints are unchanged (retire at the Phase-7 frontend cutover).

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/vuln"
	"github.com/MustardSeedNetworks/seed/internal/logging"
	"github.com/MustardSeedNetworks/seed/internal/platform/jobs"
)

// vulnScanJobKind is the registered kind name for a vulnerability scan.
const vulnScanJobKind = "vuln-scan"

// VulnScanParams is the job params for a vulnerability scan. An empty IP scans
// every discovered device; a set IP scans just that one.
type VulnScanParams struct {
	IP string `json:"ip,omitempty"`
}

// VulnScanJobResult is the job Result for a vulnerability scan: the aggregated
// per-device vulnerabilities plus how many devices were scanned and how many
// failed (surfaced, not silently dropped, unlike the legacy log-and-continue).
type VulnScanJobResult struct {
	Results []*discovery.DeviceVulnerabilities `json:"results"`
	Count   int                                `json:"count"`
	Scanned int                                `json:"scanned"`
	Failed  int                                `json:"failed"`
}

// vulnScanService is the slice of behaviour the vuln-scan kind needs:
// enumerate target devices, scan one (ctx-aware), and read aggregated results.
// The Server adapts its scanner + device registry into this seam, which also
// keeps the kind unit-testable without the real discovery services.
type vulnScanService interface {
	Devices(targetIP string) []*discovery.DiscoveredDevice
	ScanDevice(ctx context.Context, device *discovery.DiscoveredDevice) (*discovery.DeviceVulnerabilities, error)
	Results() []*discovery.DeviceVulnerabilities
}

// newVulnScanHandler returns the job Handler for the "vuln-scan" kind. It scans
// each target device in turn, reports per-device progress onto the job, and
// returns the aggregated results. Per-device errors are counted (not fatal) so
// one offline device does not fail the whole scan; a cancelled context unwinds
// the scan between (or during) devices.
func newVulnScanHandler(newSvc func() vulnScanService) jobs.Handler {
	return func(ctx context.Context, params any, report func(float64)) (any, error) {
		p, err := decodeVulnScanParams(params)
		if err != nil {
			return nil, err
		}

		svc := newSvc()
		devices := svc.Devices(p.IP)
		total := len(devices)

		var result VulnScanJobResult
		for i, device := range devices {
			if cerr := ctx.Err(); cerr != nil {
				return nil, cerr
			}
			if _, scanErr := svc.ScanDevice(ctx, device); scanErr != nil {
				// A cancelled scan unwinds the job; any other per-device error is
				// counted and the scan continues.
				if cerr := ctx.Err(); cerr != nil {
					return nil, cerr
				}
				result.Failed++
			} else {
				result.Scanned++
			}
			report(float64(i+1) / float64(total))
		}

		result.Results = svc.Results()
		result.Count = len(result.Results)
		return result, nil
	}
}

// decodeVulnScanParams parses the optional job params; absent params scan all
// devices.
func decodeVulnScanParams(params any) (VulnScanParams, error) {
	raw, ok := params.(json.RawMessage)
	if !ok || len(raw) == 0 {
		return VulnScanParams{}, nil
	}
	var p VulnScanParams
	if err := json.Unmarshal(raw, &p); err != nil {
		return VulnScanParams{}, fmt.Errorf("invalid vuln-scan params: %w", err)
	}
	return p, nil
}

// serverVulnScanService adapts the Server's scanner + device registry to the
// vulnScanService seam.
type serverVulnScanService struct {
	scanner *vuln.VulnerabilityScanner
	devices *discovery.DeviceDiscovery
}

func (a serverVulnScanService) Devices(targetIP string) []*discovery.DiscoveredDevice {
	if targetIP != "" {
		if d := a.devices.GetDeviceByIP(targetIP); d != nil {
			return []*discovery.DiscoveredDevice{d}
		}
		return nil
	}
	return a.devices.GetDevices()
}

func (a serverVulnScanService) ScanDevice(
	ctx context.Context, device *discovery.DiscoveredDevice,
) (*discovery.DeviceVulnerabilities, error) {
	return a.scanner.ScanDevice(ctx, device)
}

func (a serverVulnScanService) Results() []*discovery.DeviceVulnerabilities {
	return a.scanner.GetAllVulnerabilities()
}

// registerVulnScanKind registers the vuln-scan kind with an injectable service
// factory (the seam that makes the wiring testable without the real scanner).
func (s *Server) registerVulnScanKind(newSvc func() vulnScanService) {
	if err := s.jobsRunner().Register(vulnScanJobKind, newVulnScanHandler(newSvc)); err != nil {
		logging.GetLogger().Error("failed to register vuln-scan job kind", "error", err)
	}
}
