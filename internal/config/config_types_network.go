package config

// config_types_network.go contains the runtime network configuration types:
// HTTPS server, interface selection, VLAN, IP / static IP, switch
// discovery, network device discovery, fingerprinting, and subnet config.

import "time"

// ServerConfig contains HTTPS server settings.
type ServerConfig struct {
	Port         int    `json:"port"`
	PublicOrigin string `json:"public_origin"`
	CertFile     string `json:"cert_file"`
	KeyFile      string `json:"key_file"`
	// Security fix #301: Removed LogAccessToken/LogAccessHeader - JWT authentication is sufficient
}

// InterfaceConfig contains network interface settings.
//
// Multi-interface (Pro tier, seed#1192): the canonical list of monitored
// ethernet interfaces is Ethernet[]; the canonical list of monitored
// Wi-Fi interfaces is WiFiList[]. The legacy single-string fields
// (Default + WiFi) remain the "active/primary" indicators that 71+
// downstream callers still read. The AllEthernet / AllWiFi helpers
// reconcile both shapes so callers can choose either model:
//
//   - cfg.Interface.AllEthernet()  → full set (Ethernet[] ∪ {Default})
//   - cfg.Interface.AllWiFi()      → full set (WiFiList[] ∪ {WiFi})
//   - cfg.Interface.Default         → primary ethernet (single)
//   - cfg.Interface.WiFi            → primary Wi-Fi    (single)
//
// Free/Starter licenses cap at 1 ethernet + 1 Wi-Fi (the single fields
// only). Pro is unlimited via the slice fields. The gate fires in
// enforceMultiInterfaceGate when a saved profile would exceed the cap.
type InterfaceConfig struct {
	Default          string        `json:"default"`
	Fallbacks        []string      `json:"fallbacks"`
	WiFi             string        `json:"wifi,omitempty"`      // Separate WiFi interface (optional)
	Ethernet         []string      `json:"ethernet,omitempty"`  // Pro: additional ethernet interfaces to monitor (seed#1192)
	WiFiList         []string      `json:"wifi_list,omitempty"` // Pro: additional Wi-Fi interfaces to monitor (seed#1192)
	StartupRetries   int           `json:"startup_retries"`     // Number of retries when finding interface at startup (fixes #528)
	StartupRetryWait time.Duration `json:"startup_retry_wait"`  // Delay between startup retries (fixes #528)
}

// AllEthernet returns the de-duplicated list of ethernet interfaces the
// operator configured. Default is folded in as the first element so the
// legacy single-interface workflow remains the canonical "primary." The
// returned slice may be empty if neither field is populated.
func (c *InterfaceConfig) AllEthernet() []string {
	// Sized to len(Ethernet), not len(Ethernet)+1. The +1 anticipated Default
	// being folded in below, but it made the allocation size an arithmetic
	// expression over a value that comes straight from operator config, which
	// CodeQL's go/allocation-size-overflow taint query reports as high
	// (alerts 407-408). Overflowing would need a slice of MaxInt strings, so
	// the finding is not reachable — but the +1 was only a capacity hint, and
	// dropping it costs at most one re-allocation in the one case where
	// Default is set. Cheaper than carrying a dismissal that the next query
	// update would raise again.
	seen := make(map[string]struct{}, len(c.Ethernet))
	out := make([]string, 0, len(c.Ethernet))
	if c.Default != "" {
		seen[c.Default] = struct{}{}
		out = append(out, c.Default)
	}
	for _, name := range c.Ethernet {
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// ResolvedWiFi returns the Wi-Fi interface to scan and report on: the one the
// operator selected, else the default interface.
func (c *InterfaceConfig) ResolvedWiFi() string {
	if c.WiFi != "" {
		return c.WiFi
	}
	return c.Default
}

// AllWiFi returns the de-duplicated list of Wi-Fi interfaces the
// operator configured. WiFi is folded in as the first element so the
// legacy single-interface workflow remains the canonical "primary".
func (c *InterfaceConfig) AllWiFi() []string {
	// Sized without the +1 for the same reason as AllEthernet above
	// (CodeQL alerts 409-410).
	seen := make(map[string]struct{}, len(c.WiFiList))
	out := make([]string, 0, len(c.WiFiList))
	if c.WiFi != "" {
		seen[c.WiFi] = struct{}{}
		out = append(out, c.WiFi)
	}
	for _, name := range c.WiFiList {
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

// MultiInterfaceCounts returns (ethernetCount, wifiCount) — used by the
// license gate to decide whether the operator has exceeded the
// Free/Starter 1+1 cap.
func (c *InterfaceConfig) MultiInterfaceCounts() (int, int) {
	return len(c.AllEthernet()), len(c.AllWiFi())
}

// VLANConfig contains VLAN settings.
type VLANConfig struct {
	Enabled bool `json:"enabled"`
	ID      int  `json:"id"`
}

// IPConfig contains IP configuration settings.
type IPConfig struct {
	Mode   string    `json:"mode"` // "dhcp" or "static"
	Static *StaticIP `json:"static,omitempty"`
}

// StaticIP contains static IP configuration.
type StaticIP struct {
	Address string   `json:"address"`
	Netmask string   `json:"netmask"`
	Gateway string   `json:"gateway"`
	DNS     []string `json:"dns"`
}

// DiscoveryConfig contains switch discovery settings.
type DiscoveryConfig struct {
	Protocol string        `json:"protocol"` // "auto", "lldp", "cdp", "edp", "fdp"
	Timeout  time.Duration `json:"timeout"`
}

// PortPreset defines commonly used port scanning presets.
type PortPreset string

const (
	// PortPresetCommon scans common service ports for OS/app identification.
	// TCP: 21,22,23,25,53,80,110,111,135,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,5985,8080,8443.
	PortPresetCommon PortPreset = "common"

	// PortPresetSecure scans encrypted/authenticated service ports (good services).
	// TCP: 22,443,465,587,636,853,993,995,8443,9443.
	PortPresetSecure PortPreset = "secure"

	// PortPresetInsecure scans ports that should probably be disabled if found running.
	// TCP: 21,23,25,69,80,110,111,135,139,143,445,512,513,514,1099,2049,3389,5800,5900,6000-6009.
	PortPresetInsecure PortPreset = "insecure"

	// PortPresetCustom uses user-defined port lists.
	PortPresetCustom PortPreset = "custom"
)

// NetworkDiscoveryConfig contains network device discovery settings.
type NetworkDiscoveryConfig struct {
	// Options controls all discovery methods (no profile system).
	Options DiscoveryOptions `json:"options"`

	// Timing controls the "chattiness" of active scans.
	Timing DiscoveryTiming `json:"timing"`

	// TargetNetworks to scan in full_scan or custom mode.
	TargetNetworks []SubnetConfig `json:"target_networks"`

	Enabled     bool          `json:"enabled"`       // Enable network discovery
	ScanTimeout time.Duration `json:"scan_timeout"`  // Total scan timeout
	OUIFilePath string        `json:"oui_file_path"` // Path to IEEE OUI file
	// OUIMaxAge opts in to refreshing OUIFilePath from the IEEE registry once
	// it is older than this. Zero, the default, never calls out: the registry
	// embedded in the binary is used.
	OUIMaxAge time.Duration `json:"oui_max_age"`

	// Fingerprinting enables OS/service detection.
	Fingerprinting FingerprintingConfig `json:"fingerprinting,omitzero"`

	// IPv6Enabled enables IPv6 Neighbor Discovery Protocol (NDP) scanning.
	IPv6Enabled bool `json:"ipv6_enabled"`
}

// DiscoveryOptions provides control over all discovery methods.
type DiscoveryOptions struct {
	PassiveProtocols PassiveProtocolConfig `json:"passiveProtocols"` // Granular passive protocol control
	ARPScan          bool                  `json:"arpScan"`          // ARP-based host discovery
	ICMPScan         bool                  `json:"icmpScan"`         // ICMP ping sweep
	PortScan         PortScanConfig        `json:"portScan"`         // TCP port scanning
	Traceroute       bool                  `json:"traceroute"`       // Path discovery
	SNMPQuery        bool                  `json:"snmpQuery"`        // SNMP device interrogation
}

// PortScanConfig controls port scanning behavior. Enabling it widens the
// profiler's per-device TCP scan from the quick classification list to the
// preset's ports, or TCPPorts when the preset is custom.
type PortScanConfig struct {
	Enabled  bool       `json:"enabled"`
	Preset   PortPreset `json:"preset"`   // Port preset: common, secure, insecure, custom
	TCPPorts string     `json:"tcpPorts"` // Comma-separated ports or ranges (used when preset is "custom")
}

// EffectivePorts returns the TCP ports the preset, or the custom list, names.
func (c PortScanConfig) EffectivePorts() []int {
	switch c.Preset {
	case PortPresetSecure:
		return ParsePortList(PortsSecureTCP)
	case PortPresetInsecure:
		return ParsePortList(PortsInsecureTCP)
	case PortPresetCustom:
		return ParsePortList(c.TCPPorts)
	case PortPresetCommon:
	}
	// Common, and a config that predates presets and names none.
	return ParsePortList(PortsCommonTCP)
}

// Port preset definitions.
const (
	// PortsCommonTCP are common service ports for OS/app identification.
	PortsCommonTCP = "21,22,23,25,53,80,110,111,135,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,5985,8080,8443"

	// PortsSecureTCP are encrypted/authenticated service ports (good services).
	PortsSecureTCP = "22,443,465,587,636,853,993,995,8443,9443"

	// PortsInsecureTCP are ports that should probably be disabled if found running.
	PortsInsecureTCP = "21,23,25,69,80,110,111,135,139,143,445,512,513,514,1099,2049,3389,5800,5900,6000-6009"
)

// PassiveProtocolConfig provides granular control over passive discovery protocols.
type PassiveProtocolConfig struct {
	LLDP bool `json:"lldp"` // IEEE 802.1AB Link Layer Discovery Protocol
	CDP  bool `json:"cdp"`  // Cisco Discovery Protocol
	EDP  bool `json:"edp"`  // Extreme Discovery Protocol
	NDP  bool `json:"ndp"`  // IPv6 Neighbor Discovery Protocol
}

// TCPProbeConfig controls TCP connection probing behavior.
type TCPProbeConfig struct {
	Timeout time.Duration `json:"timeout"` // Connection timeout (default 2s)
	Workers int           `json:"workers"` // Concurrent probe workers (default 20)
}

// DeviceProfilerConfig controls automatic device profiling.
type DeviceProfilerConfig struct {
	Enabled       bool          `json:"enabled"`        // Enable automatic profiling
	Timeout       time.Duration `json:"timeout"`        // Profile operation timeout (default 2s)
	MaxConcurrent int           `json:"max_concurrent"` // Max concurrent profile operations (default 5)
	QuickPorts    []int         `json:"quick_ports"`    // Quick scan ports for profiling (default: 22,80,443,8080)
}

// DiscoveryTiming controls scan frequency.
type DiscoveryTiming struct {
	RescanInterval time.Duration `json:"rescan_interval"` // Time between full rescans (default 1m)
}

// FingerprintingConfig controls OS and service detection.
type FingerprintingConfig struct {
	Enabled       bool `json:"enabled"`        // Enable fingerprinting
	OSDetection   bool `json:"os_detection"`   // TCP stack analysis for OS detection
	ServiceProbes bool `json:"service_probes"` // Banner grabbing and service version detection
}

// SubnetConfig represents a configured subnet for network discovery.
type SubnetConfig struct {
	CIDR    string `json:"cidr"`    // CIDR notation (e.g., "10.0.0.0/24")
	Name    string `json:"name"`    // Friendly name (e.g., "Server VLAN")
	Enabled bool   `json:"enabled"` // Whether to scan this subnet

	// Learned marks a network Seed derived from a router's tables rather than
	// one an operator typed in (seed#2695). It ships disabled: learning names
	// a network, switching it on stays the operator's decision.
	Learned bool `json:"learned,omitempty"`

	// Decision records the operator's answer to a learned network (seed#3108).
	// Until there is one, the network is offered for review; a dismissed one
	// stays in the list so the learner never offers it again.
	Decision LearnedDecision `json:"decision,omitempty" jsonschema:"enum=added,enum=dismissed"`
}

// LearnedDecision is the operator's answer to a learned target network. The
// zero value means nobody has answered yet.
type LearnedDecision string

// The two answers an operator can give a learned network.
const (
	LearnedAdded     LearnedDecision = "added"
	LearnedDismissed LearnedDecision = "dismissed"
)

// AwaitingDecision reports whether the network is a learned one the operator
// has not yet been asked about. A learned network already switched on was
// answered by switching it on.
func (s SubnetConfig) AwaitingDecision() bool {
	return s.Learned && s.Decision == "" && !s.Enabled
}
