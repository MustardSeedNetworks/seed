package config

// config_types_network.go contains the runtime network configuration types:
// HTTP server, ACME, interface selection, VLAN, IP / static IP, switch
// discovery, network device discovery, fingerprinting, and subnet config.

import "time"

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port     int    `json:"port"`
	HTTPS    bool   `json:"https"`
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`
	// Security fix #301: Removed LogAccessToken/LogAccessHeader - JWT authentication is sufficient

	// ACME/Let's Encrypt automatic certificate management
	ACME ACMEConfig `json:"acme,omitzero"`
}

// ACMEConfig contains ACME/Let's Encrypt certificate settings.
type ACMEConfig struct {
	Enabled  bool   `json:"enabled"`             // Enable automatic certificate management
	Domain   string `json:"domain"`              // Domain name for the certificate (e.g., "seed.example.com")
	Email    string `json:"email"`               // Contact email for Let's Encrypt notifications
	CacheDir string `json:"cache_dir,omitempty"` // Directory to cache certificates (default: "certs/acme")
	Staging  bool   `json:"staging,omitempty"`   // Use Let's Encrypt staging server (for testing)
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
	seen := make(map[string]struct{}, len(c.Ethernet)+1)
	out := make([]string, 0, len(c.Ethernet)+1)
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

// AllWiFi returns the de-duplicated list of Wi-Fi interfaces the
// operator configured. WiFi is folded in as the first element so the
// legacy single-interface workflow remains the canonical "primary".
func (c *InterfaceConfig) AllWiFi() []string {
	seen := make(map[string]struct{}, len(c.WiFiList)+1)
	out := make([]string, 0, len(c.WiFiList)+1)
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
	// UDP: 53,67,68,69,123,137,138,161,162,500,514,1900.
	PortPresetCommon PortPreset = "common"

	// PortPresetSecure scans encrypted/authenticated service ports (good services).
	// TCP: 22,443,465,587,636,853,993,995,8443,9443.
	// UDP: 443,500,4500,853.
	PortPresetSecure PortPreset = "secure"

	// PortPresetInsecure scans ports that should probably be disabled if found running.
	// TCP: 21,23,25,69,80,110,111,135,139,143,445,512,513,514,1099,2049,3389,5800,5900,6000-6009.
	// UDP: 67,68,69,111,137,138,161,162,514,1900,2049.
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

	// AdditionalSubnets to scan in full_scan or custom mode.
	AdditionalSubnets []SubnetConfig `json:"additional_subnets"`

	// Legacy fields (kept for backward compatibility, will be deprecated)
	Enabled        bool          `json:"enabled"`          // Enable network discovery
	ARPScanWorkers int           `json:"arp_scan_workers"` // Number of concurrent workers
	PingTimeout    time.Duration `json:"ping_timeout"`     // Timeout for each ping
	ScanTimeout    time.Duration `json:"scan_timeout"`     // Total scan timeout
	AutoScan       bool          `json:"auto_scan"`        // Auto-scan on startup
	ScanInterval   time.Duration `json:"scan_interval"`    // Interval for auto-scan
	OUIFilePath    string        `json:"oui_file_path"`    // Path to IEEE OUI file
	OUIMaxAge      time.Duration `json:"oui_max_age"`      // Max age before auto-download (0 = never auto-update)

	// Fingerprinting enables OS/service detection.
	Fingerprinting FingerprintingConfig `json:"fingerprinting,omitzero"`

	// Profiler controls automatic device profiling.
	Profiler DeviceProfilerConfig `json:"profiler,omitzero"`

	// IPv6Enabled enables IPv6 Neighbor Discovery Protocol (NDP) scanning.
	IPv6Enabled bool `json:"ipv6_enabled"`
}

// DiscoveryOptions provides control over all discovery methods.
type DiscoveryOptions struct {
	PassiveProtocols PassiveProtocolConfig `json:"passiveProtocols"` // Granular passive protocol control
	ARPScan          bool                  `json:"arpScan"`          // ARP-based host discovery
	ICMPScan         bool                  `json:"icmpScan"`         // ICMP ping sweep
	PortScan         PortScanConfig        `json:"portScan"`         // TCP/UDP port scanning
	TCPProbe         TCPProbeConfig        `json:"tcpProbe"`         // TCP probe settings
	Traceroute       bool                  `json:"traceroute"`       // Path discovery
	SNMPQuery        bool                  `json:"snmpQuery"`        // SNMP device interrogation
}

// PortScanConfig controls port scanning behavior.
type PortScanConfig struct {
	Enabled       bool          `json:"enabled"`
	Preset        PortPreset    `json:"preset"`        // Port preset: common, secure, insecure, custom
	TCPPorts      string        `json:"tcpPorts"`      // Comma-separated ports or ranges (used when preset is "custom")
	UDPPorts      string        `json:"udpPorts"`      // Comma-separated ports or ranges (used when preset is "custom")
	BannerTimeout time.Duration `json:"bannerTimeout"` // Timeout for banner grabbing (default 2s)
}

// GetEffectivePorts returns the TCP and UDP ports based on the preset or custom settings.
func (c *PortScanConfig) GetEffectivePorts() (string, string) {
	switch c.Preset {
	case PortPresetCommon:
		return PortsCommonTCP, PortsCommonUDP
	case PortPresetSecure:
		return PortsSecureTCP, PortsSecureUDP
	case PortPresetInsecure:
		return PortsInsecureTCP, PortsInsecureUDP
	case PortPresetCustom:
		return c.TCPPorts, c.UDPPorts
	default:
		return PortsCommonTCP, PortsCommonUDP
	}
}

// Port preset definitions.
const (
	// PortsCommonTCP are common service ports for OS/app identification.
	PortsCommonTCP = "21,22,23,25,53,80,110,111,135,139,143,443,445,993,995,1433,1521,3306,3389,5432,5900,5985,8080,8443"
	// PortsCommonUDP are common UDP service ports.
	PortsCommonUDP = "53,67,68,69,123,137,138,161,162,500,514,1900"

	// PortsSecureTCP are encrypted/authenticated service ports (good services).
	PortsSecureTCP = "22,443,465,587,636,853,993,995,8443,9443"
	// PortsSecureUDP are encrypted UDP service ports.
	PortsSecureUDP = "443,500,4500,853"

	// PortsInsecureTCP are ports that should probably be disabled if found running.
	PortsInsecureTCP = "21,23,25,69,80,110,111,135,139,143,445,512,513,514,1099,2049,3389,5800,5900,6000-6009"
	// PortsInsecureUDP are insecure UDP service ports.
	PortsInsecureUDP = "67,68,69,111,137,138,161,162,514,1900,2049"
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

// DiscoveryTiming controls scan frequency and probe intervals.
type DiscoveryTiming struct {
	ProbeInterval  time.Duration `json:"probe_interval"`  // Time between sending probes (default 75ms)
	RescanInterval time.Duration `json:"rescan_interval"` // Time between full rescans (default 10m)
	Workers        int           `json:"workers"`         // Concurrent scan workers (default 50)
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
}
