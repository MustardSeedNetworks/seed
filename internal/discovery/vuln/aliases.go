// Package vuln is the discovery pipeline's assessment stage (ADR-0018, Phase 6):
// the vulnerability scanner, its CVE data providers (NVD / local / CISA KEV), and
// the Assessor adapter the Engine drives. It depends inward on the discovery
// kernel for the shared device + result types; the kernel never imports this
// package (depguard-enforced one-way direction).
package vuln

import "github.com/MustardSeedNetworks/seed/internal/discovery"

// Kernel type aliases. The scanner and providers operate on the device aggregate
// and the vulnerability result types, which live in the kernel (they are fields
// of DiscoveredDevice). Aliasing them here lets the stage code reference them
// unqualified without duplicating the definitions or inverting the dependency.
type (
	// DiscoveredDevice is the kernel device aggregate the scanner inspects.
	DiscoveredDevice = discovery.DiscoveredDevice
	// Vulnerability is a single CVE finding (kernel result type).
	Vulnerability = discovery.Vulnerability
	// DeviceVulnerabilities is a device's assessment result (kernel result type).
	DeviceVulnerabilities = discovery.DeviceVulnerabilities
)
