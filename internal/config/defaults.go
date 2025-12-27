// Package config provides default settings as the single source of truth.
// The default profile settings are exposed via API so the frontend
// doesn't need to maintain duplicate default values.
package config

import "time"

// DefaultSettings contains all user-facing default settings.
// This is the single source of truth for defaults across the entire application.
// Frontend should load these from /api/settings/defaults instead of hardcoding.
type DefaultSettings struct {
	// CardSettings controls per-card visibility and auto-run behavior
	CardSettings CardSettingsDefaults `json:"cardSettings"`

	// DisplayOptions controls UI display preferences
	DisplayOptions DisplayOptionsDefaults `json:"displayOptions"`

	// Thresholds contains all threshold default values
	Thresholds ThresholdDefaults `json:"thresholds"`

	// Iperf contains default iPerf settings
	Iperf IperfDefaults `json:"iperf"`

	// Tests contains default health check test configurations
	Tests TestsDefaults `json:"tests"`

	// NetworkDiscovery contains default discovery settings
	NetworkDiscovery NetworkDiscoveryDefaults `json:"networkDiscovery"`

	// SNMP contains default SNMP settings
	SNMP SNMPDefaults `json:"snmp"`

	// Link contains default link settings
	Link LinkDefaults `json:"link"`

	// CableTest contains default cable test settings
	CableTest CableTestDefaults `json:"cableTest"`

	// Vulnerability contains default vulnerability scan settings
	Vulnerability VulnerabilityDefaults `json:"vulnerability"`
}

// CardOptionDefaults defines defaults for a single card's behavior.
type CardOptionDefaults struct {
	Enabled       bool `json:"enabled"`
	AutoRunOnLink bool `json:"autoRunOnLink"`
}

// PerformanceCardDefaults includes sub-options for speedtest and iperf.
type PerformanceCardDefaults struct {
	Enabled       bool               `json:"enabled"`
	AutoRunOnLink bool               `json:"autoRunOnLink"`
	Speedtest     CardOptionDefaults `json:"speedtest"`
	Iperf         CardOptionDefaults `json:"iperf"`
}

// CardSettingsDefaults contains defaults for all card behaviors.
type CardSettingsDefaults struct {
	Link             CardOptionDefaults      `json:"link"`
	Switch           CardOptionDefaults      `json:"switch"`
	VLAN             CardOptionDefaults      `json:"vlan"`
	Network          CardOptionDefaults      `json:"network"`
	Gateway          CardOptionDefaults      `json:"gateway"`
	DNS              CardOptionDefaults      `json:"dns"`
	HealthChecks     CardOptionDefaults      `json:"healthChecks"`
	NetworkDiscovery CardOptionDefaults      `json:"networkDiscovery"`
	Performance      PerformanceCardDefaults `json:"performance"`
}

// DisplayOptionsDefaults contains default display/UI preferences.
type DisplayOptionsDefaults struct {
	ShowPublicIP bool   `json:"showPublicIP"`
	UnitSystem   string `json:"unitSystem"`
}

// ThresholdPairDefaults contains good/warning threshold values.
type ThresholdPairDefaults struct {
	Good    int64 `json:"good"`
	Warning int64 `json:"warning"`
}

// ThresholdPairIntDefaults contains good/warning threshold values as integers.
type ThresholdPairIntDefaults struct {
	Good    int `json:"good"`
	Warning int `json:"warning"`
}

// HTTPTimingThresholdDefaults contains per-phase HTTP timing thresholds.
type HTTPTimingThresholdDefaults struct {
	DNS  ThresholdPairDefaults `json:"dns"`
	TCP  ThresholdPairDefaults `json:"tcp"`
	TLS  ThresholdPairDefaults `json:"tls"`
	TTFB ThresholdPairDefaults `json:"ttfb"`
}

// ThresholdDefaults contains all threshold default values.
type ThresholdDefaults struct {
	DNS         ThresholdPairDefaults       `json:"dns"`
	Gateway     ThresholdPairDefaults       `json:"gateway"`
	WiFi        ThresholdPairIntDefaults    `json:"wifi"`
	CustomPing  ThresholdPairDefaults       `json:"customPing"`
	CustomTCP   ThresholdPairDefaults       `json:"customTcp"`
	CustomHTTP  ThresholdPairDefaults       `json:"customHttp"`
	HTTPTimings HTTPTimingThresholdDefaults `json:"httpTimings"`
}

// IperfDefaults contains default iPerf configuration.
type IperfDefaults struct {
	Server        string `json:"server"`
	Port          int    `json:"port"`
	Protocol      string `json:"protocol"`
	Direction     string `json:"direction"`
	Duration      int    `json:"duration"`
	ServerPort    int    `json:"serverPort"`
	EnableServer  bool   `json:"enableServer"`
	AutoRunOnLink bool   `json:"autoRunOnLink"`
}

// DefaultPingTarget represents a default ping target configuration.
type DefaultPingTarget struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
	Count   int    `json:"count,omitempty"`
}

// DefaultHTTPEndpoint represents a default HTTP endpoint configuration.
type DefaultHTTPEndpoint struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expectedStatus"`
	Enabled        bool   `json:"enabled"`
}

// SpeedtestDefaults contains default speedtest configuration.
type SpeedtestDefaults struct {
	ServerID      string `json:"serverId"`
	AutoRunOnLink bool   `json:"autoRunOnLink"`
}

// TestsDefaults contains default health check test configurations.
type TestsDefaults struct {
	DNSHostname    string                `json:"dnsHostname"`
	PingTargets    []DefaultPingTarget   `json:"pingTargets"`
	HTTPEndpoints  []DefaultHTTPEndpoint `json:"httpEndpoints"`
	RunPerformance bool                  `json:"runPerformance"`
	RunSpeedtest   bool                  `json:"runSpeedtest"`
	RunIperf       bool                  `json:"runIperf"`
	RunDiscovery   bool                  `json:"runDiscovery"`
	Speedtest      SpeedtestDefaults     `json:"speedtest"`
}

// PassiveProtocolDefaults contains default passive protocol settings.
type PassiveProtocolDefaults struct {
	LLDP bool `json:"lldp"`
	CDP  bool `json:"cdp"`
	EDP  bool `json:"edp"`
	NDP  bool `json:"ndp"`
}

// PortScanDefaults contains default port scan settings.
type PortScanDefaults struct {
	Enabled         bool   `json:"enabled"`
	Preset          string `json:"preset"`
	TCPPorts        string `json:"tcpPorts"`
	UDPPorts        string `json:"udpPorts"`
	BannerTimeoutMs int64  `json:"bannerTimeoutMs"`
}

// TCPProbeDefaults contains default TCP probe settings.
type TCPProbeDefaults struct {
	TimeoutMs int64 `json:"timeoutMs"`
	Workers   int   `json:"workers"`
}

// DiscoveryOptionsDefaults contains default discovery option settings.
type DiscoveryOptionsDefaults struct {
	PassiveProtocols PassiveProtocolDefaults `json:"passiveProtocols"`
	ARPScan          bool                    `json:"arpScan"`
	ICMPScan         bool                    `json:"icmpScan"`
	PortScan         PortScanDefaults        `json:"portScan"`
	TCPProbe         TCPProbeDefaults        `json:"tcpProbe"`
	Traceroute       bool                    `json:"traceroute"`
	SNMPQuery        bool                    `json:"snmpQuery"`
}

// DiscoveryTimingDefaults contains default timing settings.
type DiscoveryTimingDefaults struct {
	ProbeIntervalMs  int64 `json:"probeIntervalMs"`
	RescanIntervalMs int64 `json:"rescanIntervalMs"`
	Workers          int   `json:"workers"`
}

// DeviceProfilerDefaults contains default device profiler settings.
type DeviceProfilerDefaults struct {
	Enabled       bool  `json:"enabled"`
	TimeoutMs     int64 `json:"timeoutMs"`
	MaxConcurrent int   `json:"maxConcurrent"`
	QuickPorts    []int `json:"quickPorts"`
}

// FingerprintingDefaults contains default fingerprinting settings.
type FingerprintingDefaults struct {
	Enabled       bool `json:"enabled"`
	OSDetection   bool `json:"osDetection"`
	ServiceProbes bool `json:"serviceProbes"`
}

// NetworkDiscoveryDefaults contains all network discovery defaults.
// Note: OUI database is baked into binary at build time - no runtime path needed.
type NetworkDiscoveryDefaults struct {
	Enabled        bool                     `json:"enabled"`
	ARPScanWorkers int                      `json:"arpScanWorkers"`
	PingTimeoutMs  int64                    `json:"pingTimeoutMs"`
	ScanTimeoutMs  int64                    `json:"scanTimeoutMs"`
	AutoScan       bool                     `json:"autoScan"`
	ScanIntervalMs int64                    `json:"scanIntervalMs"`
	IPv6Enabled    bool                     `json:"ipv6Enabled"`
	Options        DiscoveryOptionsDefaults `json:"options"`
	Timing         DiscoveryTimingDefaults  `json:"timing"`
	Profiler       DeviceProfilerDefaults   `json:"profiler"`
	Fingerprinting FingerprintingDefaults   `json:"fingerprinting"`
}

// SNMPDefaults contains default SNMP settings.
type SNMPDefaults struct {
	Communities []string `json:"communities"`
	TimeoutMs   int64    `json:"timeoutMs"`
	Retries     int      `json:"retries"`
	Port        int      `json:"port"`
}

// LinkDefaults contains default link settings.
// Uses combined mode format (e.g., "10/half", "100/full", "1000/full") matching ethtool output.
type LinkDefaults struct {
	// Mode is the combined speed/duplex (e.g., "100/full", "1000/full") or "auto" for auto-negotiation
	Mode string `json:"mode"`
	// AvailableModes lists available modes from the interface
	AvailableModes []string `json:"availableModes"`
}

// CableTestDefaults contains default cable test settings.
// Note: Cable test auto-runs automatically when link is down AND PHY supports TDR.
// No user toggle needed - it's either possible or not based on hardware capability.
type CableTestDefaults struct {
	// Enabled controls whether cable testing is available (requires PHY TDR support)
	Enabled bool `json:"enabled"`
}

// VulnerabilityDefaults contains default vulnerability scan settings.
type VulnerabilityDefaults struct {
	Enabled           bool   `json:"enabled"`
	CVEDatabase       string `json:"cveDatabase"`
	NVDAPIKey         string `json:"nvdApiKey"`
	UpdateInterval    int    `json:"updateInterval"`
	SeverityThreshold string `json:"severityThreshold"`
	MaxConcurrent     int    `json:"maxConcurrent"`
	AutoScan          bool   `json:"autoScan"`
}

// GetDefaultSettings returns all default settings as the single source of truth.
// This eliminates the need for duplicated DEFAULT_* constants in the frontend.
//
//nolint:dupl // Similar structure to ProfileSettings.FromConfig but different types and semantics (Good/Warning vs Warning/Critical)
func GetDefaultSettings() *DefaultSettings {
	cfg := DefaultConfig()

	return &DefaultSettings{
		CardSettings: CardSettingsDefaults{
			Link:             CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunLink},
			Switch:           CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunSwitch},
			VLAN:             CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunVLAN},
			Network:          CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunIPConfig},
			Gateway:          CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunGateway},
			DNS:              CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunDNS},
			HealthChecks:     CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunHealthChecks},
			NetworkDiscovery: CardOptionDefaults{Enabled: true, AutoRunOnLink: cfg.FABOptions.RunNetworkDiscovery},
			Performance: PerformanceCardDefaults{
				Enabled:       true,
				AutoRunOnLink: cfg.FABOptions.RunPerformance,
				Speedtest:     CardOptionDefaults{Enabled: cfg.HealthChecks.RunSpeedtest, AutoRunOnLink: cfg.Speedtest.AutoRunOnLink},
				Iperf:         CardOptionDefaults{Enabled: cfg.HealthChecks.RunIperf, AutoRunOnLink: cfg.Iperf.AutoRunOnLink},
			},
		},
		DisplayOptions: DisplayOptionsDefaults{
			ShowPublicIP: cfg.DisplayOptions.ShowPublicIP,
			UnitSystem:   cfg.DisplayOptions.UnitSystem,
		},
		Thresholds: ThresholdDefaults{
			DNS: ThresholdPairDefaults{
				Good:    cfg.Thresholds.DNS.Warning.Milliseconds(),
				Warning: cfg.Thresholds.DNS.Critical.Milliseconds(),
			},
			Gateway: ThresholdPairDefaults{
				Good:    cfg.Thresholds.Ping.Warning.Milliseconds(),
				Warning: cfg.Thresholds.Ping.Critical.Milliseconds(),
			},
			WiFi: ThresholdPairIntDefaults{
				Good:    cfg.Thresholds.WiFi.Signal.Warning,
				Warning: cfg.Thresholds.WiFi.Signal.Critical,
			},
			CustomPing: ThresholdPairDefaults{
				Good:    cfg.Thresholds.CustomTests.Ping.Warning.Milliseconds(),
				Warning: cfg.Thresholds.CustomTests.Ping.Critical.Milliseconds(),
			},
			CustomTCP: ThresholdPairDefaults{
				Good:    cfg.Thresholds.CustomTests.TCP.Warning.Milliseconds(),
				Warning: cfg.Thresholds.CustomTests.TCP.Critical.Milliseconds(),
			},
			CustomHTTP: ThresholdPairDefaults{
				Good:    cfg.Thresholds.CustomTests.HTTP.Warning.Milliseconds(),
				Warning: cfg.Thresholds.CustomTests.HTTP.Critical.Milliseconds(),
			},
			HTTPTimings: HTTPTimingThresholdDefaults{
				DNS: ThresholdPairDefaults{
					Good:    cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
					Warning: cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
				},
				TCP: ThresholdPairDefaults{
					Good:    cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
					Warning: cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
				},
				TLS: ThresholdPairDefaults{
					Good:    cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
					Warning: cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
				},
				TTFB: ThresholdPairDefaults{
					Good:    cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
					Warning: cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
				},
			},
		},
		Iperf: IperfDefaults{
			Server:        cfg.Iperf.Server,
			Port:          cfg.Iperf.Port,
			Protocol:      cfg.Iperf.Protocol,
			Direction:     cfg.Iperf.Direction,
			Duration:      cfg.Iperf.Duration,
			ServerPort:    cfg.Iperf.ServerPort,
			EnableServer:  cfg.Iperf.EnableServer,
			AutoRunOnLink: cfg.Iperf.AutoRunOnLink,
		},
		Tests: buildTestsDefaults(cfg),
		NetworkDiscovery: NetworkDiscoveryDefaults{
			Enabled:        cfg.NetworkDiscovery.Enabled,
			ARPScanWorkers: cfg.NetworkDiscovery.ARPScanWorkers,
			PingTimeoutMs:  cfg.NetworkDiscovery.PingTimeout.Milliseconds(),
			ScanTimeoutMs:  cfg.NetworkDiscovery.ScanTimeout.Milliseconds(),
			AutoScan:       cfg.NetworkDiscovery.AutoScan,
			ScanIntervalMs: cfg.NetworkDiscovery.ScanInterval.Milliseconds(),
			IPv6Enabled:    cfg.NetworkDiscovery.IPv6Enabled,
			Options: DiscoveryOptionsDefaults{
				PassiveProtocols: PassiveProtocolDefaults{
					LLDP: cfg.NetworkDiscovery.Options.PassiveProtocols.LLDP,
					CDP:  cfg.NetworkDiscovery.Options.PassiveProtocols.CDP,
					EDP:  cfg.NetworkDiscovery.Options.PassiveProtocols.EDP,
					NDP:  cfg.NetworkDiscovery.Options.PassiveProtocols.NDP,
				},
				ARPScan:  cfg.NetworkDiscovery.Options.ARPScan,
				ICMPScan: cfg.NetworkDiscovery.Options.ICMPScan,
				PortScan: PortScanDefaults{
					Enabled:         cfg.NetworkDiscovery.Options.PortScan.Enabled,
					Preset:          string(cfg.NetworkDiscovery.Options.PortScan.Preset),
					TCPPorts:        cfg.NetworkDiscovery.Options.PortScan.TCPPorts,
					UDPPorts:        cfg.NetworkDiscovery.Options.PortScan.UDPPorts,
					BannerTimeoutMs: cfg.NetworkDiscovery.Options.PortScan.BannerTimeout.Milliseconds(),
				},
				TCPProbe: TCPProbeDefaults{
					TimeoutMs: cfg.NetworkDiscovery.Options.TCPProbe.Timeout.Milliseconds(),
					Workers:   cfg.NetworkDiscovery.Options.TCPProbe.Workers,
				},
				Traceroute: cfg.NetworkDiscovery.Options.Traceroute,
				SNMPQuery:  cfg.NetworkDiscovery.Options.SNMPQuery,
			},
			Timing: DiscoveryTimingDefaults{
				ProbeIntervalMs:  cfg.NetworkDiscovery.Timing.ProbeInterval.Milliseconds(),
				RescanIntervalMs: cfg.NetworkDiscovery.Timing.RescanInterval.Milliseconds(),
				Workers:          cfg.NetworkDiscovery.Timing.Workers,
			},
			Profiler: DeviceProfilerDefaults{
				Enabled:       cfg.NetworkDiscovery.Profiler.Enabled,
				TimeoutMs:     cfg.NetworkDiscovery.Profiler.Timeout.Milliseconds(),
				MaxConcurrent: cfg.NetworkDiscovery.Profiler.MaxConcurrent,
				QuickPorts:    cfg.NetworkDiscovery.Profiler.QuickPorts,
			},
			Fingerprinting: FingerprintingDefaults{
				Enabled:       cfg.NetworkDiscovery.Fingerprinting.Enabled,
				OSDetection:   cfg.NetworkDiscovery.Fingerprinting.OSDetection,
				ServiceProbes: cfg.NetworkDiscovery.Fingerprinting.ServiceProbes,
			},
		},
		SNMP: SNMPDefaults{
			Communities: cfg.SNMP.Communities,
			TimeoutMs:   cfg.SNMP.Timeout.Milliseconds(),
			Retries:     cfg.SNMP.Retries,
			Port:        cfg.SNMP.Port,
		},
		Link: LinkDefaults{
			Mode:           "auto",
			AvailableModes: []string{},
		},
		CableTest: CableTestDefaults{
			Enabled: true,
		},
		Vulnerability: VulnerabilityDefaults{
			Enabled:           cfg.Security.VulnerabilityScanning.Enabled,
			CVEDatabase:       cfg.Security.VulnerabilityScanning.CVEDatabase,
			NVDAPIKey:         "", // Never expose API key in defaults
			UpdateInterval:    cfg.Security.VulnerabilityScanning.UpdateInterval,
			SeverityThreshold: cfg.Security.VulnerabilityScanning.SeverityThreshold,
			MaxConcurrent:     cfg.Security.VulnerabilityScanning.MaxConcurrent,
			AutoScan:          cfg.Security.VulnerabilityScanning.AutoScan,
		},
	}
}

// buildTestsDefaults constructs test defaults from config.
func buildTestsDefaults(cfg *Config) TestsDefaults {
	pingTargets := make([]DefaultPingTarget, 0, len(cfg.HealthChecks.PingTargets))
	for i, pt := range cfg.HealthChecks.PingTargets {
		pingTargets = append(pingTargets, DefaultPingTarget{
			ID:      generateDefaultID("ping", i),
			Name:    pt.Name,
			Host:    pt.Host,
			Enabled: pt.Enabled,
			Count:   3, // Default ping count
		})
	}

	httpEndpoints := make([]DefaultHTTPEndpoint, 0, len(cfg.HealthChecks.HTTPEndpoints))
	for i, he := range cfg.HealthChecks.HTTPEndpoints {
		httpEndpoints = append(httpEndpoints, DefaultHTTPEndpoint{
			ID:             generateDefaultID("http", i),
			Name:           he.Name,
			URL:            he.URL,
			ExpectedStatus: he.ExpectedStatus,
			Enabled:        he.Enabled,
		})
	}

	return TestsDefaults{
		DNSHostname:    cfg.DNS.TestHostname,
		PingTargets:    pingTargets,
		HTTPEndpoints:  httpEndpoints,
		RunPerformance: cfg.HealthChecks.RunPerformance,
		RunSpeedtest:   cfg.HealthChecks.RunSpeedtest,
		RunIperf:       cfg.HealthChecks.RunIperf,
		RunDiscovery:   cfg.HealthChecks.RunDiscovery,
		Speedtest: SpeedtestDefaults{
			ServerID:      cfg.Speedtest.ServerID,
			AutoRunOnLink: cfg.Speedtest.AutoRunOnLink,
		},
	}
}

// generateDefaultID creates a stable ID for default items.
func generateDefaultID(prefix string, index int) string {
	return prefix + "-default-" + time.Now().Format("2006") + "-" + string(rune('0'+index))
}
