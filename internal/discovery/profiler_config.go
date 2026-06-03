package discovery

// profiler_config.go contains ProfilerConfig (the device-profiler's tunable
// surface), its defaults / pipeline-derived constructor, the port-list
// resolver, and the TLS-config helper plus the runtime config update path
// used by Pipeline.

import (
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/krisarmstrong/seed/internal/logging"
)

// ProfilerConfig holds configuration for the device profiler.
type ProfilerConfig struct {
	Enabled       bool
	Timeout       time.Duration
	MaxConcurrent int
	QuickPorts    []int // Ports to check during quick profile

	// Enhanced settings for pipeline integration
	PortScanIntensity PortScanIntensity // Intensity level for port scanning
	TimingProfile     ScanTimingProfile // Timing profile for rate limiting
	CustomPorts       []int             // Custom port list when intensity is PortScanCustom
	BannerGrab        bool              // Whether to attempt banner grabbing
	ProbeDelay        time.Duration     // Delay between probes to same host
	HostDelay         time.Duration     // Delay between starting different hosts
	ConnectTimeout    time.Duration     // Timeout for TCP connections

	// TLS configuration for HTTPS probing
	// SkipTLSVerify allows connecting to devices with self-signed certificates.
	// This is common for network devices, printers, and other internal infrastructure.
	// Default: true (required for profiling internal network devices)
	SkipTLSVerify bool

	// EnableSNMPCollection enables automatic full SNMP MIB collection during profiling.
	// When true and SNMP credentials are configured, the profiler will collect full
	// interface MIB data (IF-MIB, IP-MIB, BRIDGE-MIB, etc.) from each device.
	EnableSNMPCollection bool

	// SNMPMIBs specifies which MIBs to collect. If nil, defaults to interface MIBs.
	SNMPMIBs *SNMPMIBSelection

	// EnableNameResolution enables automatic DNS, NetBIOS, and mDNS name resolution.
	EnableNameResolution bool

	// ResolveDNS enables reverse DNS PTR lookups.
	ResolveDNS bool

	// ResolveNetBIOS enables NetBIOS name queries for Windows devices.
	ResolveNetBIOS bool

	// ResolveMDNS enables mDNS queries for Apple/Linux devices.
	ResolveMDNS bool

	// NameResolutionTimeout is the timeout for each name resolution query.
	NameResolutionTimeout time.Duration
}

// DefaultProfilerConfig returns sensible defaults.
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		Enabled:       true,
		Timeout:       profilerTimeoutS * time.Second,
		MaxConcurrent: profilerMaxConcurrent,
		QuickPorts: []int{
			22,   // SSH
			23,   // Telnet
			80,   // HTTP
			443,  // HTTPS
			8080, // HTTP Alt
			8443, // HTTPS Alt
			// Note: SNMP (port 161) is UDP, probed separately via probeSNMP()
		},
		PortScanIntensity: PortScanOff, // Default: OFF for security
		TimingProfile:     ScanProfileNormal,
		BannerGrab:        true,
		ProbeDelay:        profilerProbeDelayMs * time.Millisecond,
		HostDelay:         profilerHostDelayMs * time.Millisecond,
		ConnectTimeout:    profilerTimeoutS * time.Second,
		SkipTLSVerify:     false, // Set to true for internal network devices with self-signed certs
		// Enable automatic SNMP collection when credentials are configured
		EnableSNMPCollection: true,
		// Default to interface MIBs for network device discovery
		SNMPMIBs: &SNMPMIBSelection{
			System:      true,  // sysDescr, sysName, sysLocation, etc.
			Interfaces:  true,  // IF-MIB (interface speeds, MACs, status)
			IPAddresses: true,  // IP-MIB (device IPs)
			Bridge:      true,  // BRIDGE-MIB (MAC table for switches)
			VLAN:        true,  // Q-BRIDGE-MIB (VLAN info)
			LLDP:        true,  // LLDP-MIB (neighbor discovery)
			Routing:     false, // IP-FORWARD-MIB (disable by default - can be large)
			Entity:      false, // ENTITY-MIB (disable by default - not always useful)
		},
		// Enable automatic name resolution
		EnableNameResolution:  true,
		ResolveDNS:            true, // DNS PTR lookups for all IPs
		ResolveNetBIOS:        true, // NetBIOS for Windows devices
		ResolveMDNS:           true, // mDNS for Apple/Linux devices
		NameResolutionTimeout: profilerNameResolveTimeMs * time.Millisecond,
	}
}

// NewProfilerConfigFromPipeline creates a ProfilerConfig from pipeline settings.
func NewProfilerConfigFromPipeline(pipelineConfig *PipelineConfig) *ProfilerConfig {
	cfg := DefaultProfilerConfig()
	cfg.PortScanIntensity = pipelineConfig.PortScan.Intensity
	cfg.CustomPorts = pipelineConfig.PortScan.CustomPorts
	cfg.BannerGrab = pipelineConfig.PortScan.BannerGrab
	cfg.ConnectTimeout = pipelineConfig.PortScan.ConnectTimeout
	cfg.TimingProfile = pipelineConfig.Timing.Profile
	cfg.ProbeDelay = pipelineConfig.Timing.ProbeDelay
	cfg.HostDelay = pipelineConfig.Timing.HostDelay
	cfg.MaxConcurrent = pipelineConfig.Timing.MaxConcurrentHosts
	cfg.Timeout = pipelineConfig.Timing.PhaseTimeout
	return cfg
}

// GetPortsForIntensity returns the appropriate port list based on intensity level.
func (c *ProfilerConfig) GetPortsForIntensity() []int {
	switch c.PortScanIntensity {
	case PortScanOff:
		return nil
	case PortScanQuick:
		return GetQuickPorts()
	case PortScanStandard:
		return GetStandardPorts()
	case PortScanComprehensive:
		return GetComprehensivePorts()
	case PortScanCustom:
		return c.CustomPorts
	default:
		return nil
	}
}

// UpdateScanConfig updates the port scanning configuration.
// This allows Pipeline to set the scan intensity without recreating the profiler.
// Thread-safe: can be called while profiler is running.
func (p *DeviceProfiler) UpdateScanConfig(
	intensity PortScanIntensity,
	customPorts []int,
	timing ScanTimingProfile,
) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.PortScanIntensity = intensity
	p.config.CustomPorts = customPorts
	p.config.TimingProfile = timing
	logging.GetLogger().
		Info("Updated profiler scan config", "intensity", intensity, "timing", timing)
}

// newProfilerTLSConfig creates a TLS config for the device profiler.
// When insecure is true, the config uses a custom verification function
// that accepts all certificates (required for internal network devices
// with self-signed certificates).
func newProfilerTLSConfig(insecure bool) *tls.Config {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	if insecure {
		// Use custom verification that accepts all certificates.
		// This is required for profiling internal network devices with self-signed certs.
		cfg.VerifyPeerCertificate = func(_ [][]byte, _ [][]*x509.Certificate) error {
			return nil // Accept all certificates
		}
		cfg.VerifyConnection = func(_ tls.ConnectionState) error {
			return nil // Accept all connections
		}
	}
	return cfg
}
