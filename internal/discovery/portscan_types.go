package discovery

// portscan_types.go holds the port-scan RESULT vocabulary that stays in the
// discovery kernel after the fingerprint stage relocation (ADR-0018, Phase 6):
// PortState + its constants, ServiceInfo, and PortScanResult. These are kept
// kernel-side because they appear in the PortScannerPort.QuickScan signature
// (stages.go) — the Engine and enrich stage consume them without importing the
// fingerprint subpackage, and the fingerprint scanner aliases them. PortState
// also previously lived (duplicated) in the build-tagged tcpprobe files; it is
// platform-independent, so consolidating it here removes that duplication.
//
// Two unexported strings also stay here because staying kernel code references
// them (they originated in the relocated portscan.go): serviceUnknown (used by
// the device-type classifier in profiler_infer.go) and errNoIPv4ForTarget (used
// by traceroute.go). The fingerprint scanner keeps its own private copies.

import "time"

// Shared port-scan string constants retained in the kernel for staying callers.
const (
	// errNoIPv4ForTarget is reported when a hostname resolves to no IPv4 address.
	// Referenced by traceroute.go.
	errNoIPv4ForTarget = "no IPv4 address found for target"
	// serviceUnknown is the placeholder service name. Referenced by the device
	// classifier in profiler_infer.go.
	serviceUnknown = "unknown"
)

// PortState represents the state of a TCP port.
type PortState string

// TCP port state constants indicating probe results.
const (
	PortOpen     PortState = "open"     // SYN/ACK received
	PortClosed   PortState = "closed"   // RST received
	PortFiltered PortState = "filtered" // No response (timeout)
)

// ServiceInfo contains information about a detected service.
type ServiceInfo struct {
	Port     int       `json:"port"`
	State    PortState `json:"state"`
	Service  string    `json:"service"`            // Service name (http, ssh, etc.)
	Banner   string    `json:"banner,omitempty"`   // Raw banner text
	Version  string    `json:"version,omitempty"`  // Parsed version if available
	Protocol string    `json:"protocol,omitempty"` // tcp or udp
}

// PortScanResult contains the complete result of a port scan.
type PortScanResult struct {
	IP       string        `json:"ip"`
	Hostname string        `json:"hostname,omitempty"`
	Services []ServiceInfo `json:"services"`
	ScanTime time.Duration `json:"scanTime"`
	Error    string        `json:"error,omitempty"`
}
