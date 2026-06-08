package vuln

import (
	"context"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
)

// Stage is the assessment stage adapter (ADR-0018): it satisfies
// discovery.Assessor, scanning every registry device for vulnerabilities and
// emitting a discovery event per finding. It holds the kernel collaborators
// (registry + event bus) the Engine exposes, so the orchestrator depends only on
// the Assessor port while the concrete scanner lives here.
type Stage struct {
	registry *discovery.DeviceRegistry
	eventBus *discovery.EventBus
	scanner  *VulnerabilityScanner
}

// NewStage builds the assessment stage over scanner, writing results into reg
// and publishing findings on bus. Returned as the discovery.Assessor port the
// Engine consumes.
func NewStage(
	scanner *VulnerabilityScanner,
	reg *discovery.DeviceRegistry,
	bus *discovery.EventBus,
) discovery.Assessor {
	return &Stage{registry: reg, eventBus: bus, scanner: scanner}
}

// Assess scans each registry device and records any vulnerabilities found.
func (s *Stage) Assess(ctx context.Context, stats *discovery.ScanStats) {
	for _, device := range s.registry.GetDevices() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		vulns := s.assess(ctx, device)
		if vulns != nil && len(vulns.Vulnerabilities) > 0 {
			device.Vulnerabilities = vulns
			stats.VulnerableDevices++
			s.registry.AddOrUpdate(device)

			for _, v := range vulns.Vulnerabilities {
				s.eventBus.Publish(discovery.NewVulnDiscoveredEvent(device, v.CVEID, v.Severity))
			}
		}
	}
}

func (s *Stage) assess(ctx context.Context, device *DiscoveredDevice) *DeviceVulnerabilities {
	if s.scanner == nil {
		return nil
	}
	vulns, err := s.scanner.ScanDevice(ctx, device)
	if err != nil {
		return nil
	}
	return vulns
}
