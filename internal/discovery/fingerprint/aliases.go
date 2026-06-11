// Package fingerprint is the discovery pipeline's active port-scan stage
// (ADR-0018, Phase 6): the TCP prober and the banner-grabbing port scanner the
// Engine drives via the discovery.PortScannerPort seam. It depends inward on
// the discovery kernel for the shared port-scan result types; the kernel never
// imports this package (depguard-enforced one-way direction).
//
// Scope note: only the PortScanner/TCPProber leaf cluster lives here. The
// SNMPCollector + DeviceProfiler cluster stays in the kernel because the kernel
// orchestrator discovery.Service co-owns the profiler lifecycle (Start/Stop/
// Clear/queue/get) and cannot move without an import cycle — see ADR-0018 §2.
package fingerprint

import "github.com/MustardSeedNetworks/seed/internal/discovery"

// Kernel type aliases. The scanner returns the kernel's port-scan result types
// (they sit in the PortScannerPort.QuickScan signature, so they must live in the
// kernel). Aliasing them here lets the stage code reference them unqualified
// without duplicating the definitions or inverting the dependency.
type (
	// PortState is the kernel TCP port-state enum.
	PortState = discovery.PortState
	// ServiceInfo is a single detected service (kernel result type).
	ServiceInfo = discovery.ServiceInfo
	// PortScanResult is a host's complete scan result (kernel result type).
	PortScanResult = discovery.PortScanResult
)

// Kernel port-state constant aliases, re-exported so the prober/scanner logic
// references them unqualified.
const (
	// PortOpen indicates a SYN/ACK was received.
	PortOpen = discovery.PortOpen
	// PortClosed indicates an RST was received.
	PortClosed = discovery.PortClosed
	// PortFiltered indicates no response (timeout).
	PortFiltered = discovery.PortFiltered
)
