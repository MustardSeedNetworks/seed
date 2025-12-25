// Package config provides profile-specific settings support.
package config

import (
	"encoding/json"
	"time"
)

// ProfileSettings contains settings that are stored per-profile.
// These settings are applied when a profile is activated.
// Global settings (server, auth, security, etc.) are NOT included.
type ProfileSettings struct {
	// Version for future migrations
	Version int `json:"version"`

	// Interfaces contains per-interface configurations (v2+).
	// Each profile can select one ethernet and one wifi interface.
	Interfaces ProfileInterfaceConfigs `json:"interfaces,omitempty"`

	// Thresholds for network tests (profile-level defaults, can be overridden per-interface)
	Thresholds ProfileThresholds `json:"thresholds,omitempty"`

	// HealthChecks contains custom test configurations
	HealthChecks ProfileHealthChecks `json:"health_checks,omitempty"`

	// Speedtest configuration
	Speedtest ProfileSpeedtest `json:"speedtest,omitempty"`

	// Iperf configuration
	Iperf ProfileIperf `json:"iperf,omitempty"`

	// FABOptions controls what tests run on FAB press
	FABOptions ProfileFABOptions `json:"fab_options,omitempty"`

	// DisplayOptions contains UI preferences
	DisplayOptions ProfileDisplayOptions `json:"display_options,omitempty"`

	// DNS test configuration
	DNS ProfileDNS `json:"dns,omitempty"`

	// SNMP credentials for device interrogation
	SNMP ProfileSNMP `json:"snmp,omitempty"`

	// NetworkDiscovery settings
	NetworkDiscovery ProfileNetworkDiscovery `json:"network_discovery,omitempty"`

	// Link settings for interface speed/duplex configuration
	Link ProfileLinkSettings `json:"link,omitempty"`

	// CableTest settings for TDR cable diagnostics
	CableTest ProfileCableTestSettings `json:"cable_test,omitempty"`

	// Notes field for user documentation
	Notes string `json:"notes,omitempty"`
}

// ProfileThresholds contains threshold settings per profile.
type ProfileThresholds struct {
	DNS         ThresholdPair               `json:"dns,omitempty"`
	Gateway     ThresholdPair               `json:"gateway,omitempty"`
	WiFi        WiFiThresholdPair           `json:"wifi,omitempty"`
	CustomPing  ThresholdPair               `json:"custom_ping,omitempty"`
	CustomTCP   ThresholdPair               `json:"custom_tcp,omitempty"`
	CustomHTTP  ThresholdPair               `json:"custom_http,omitempty"`
	HTTPTimings ProfileHTTPTimingThresholds `json:"http_timings,omitempty"`
}

// ThresholdPair stores warning/critical thresholds in milliseconds.
type ThresholdPair struct {
	Warning  int64 `json:"warning"`
	Critical int64 `json:"critical"`
}

// WiFiThresholdPair stores WiFi signal thresholds in dBm.
type WiFiThresholdPair struct {
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

// ProfileHTTPTimingThresholds contains per-phase HTTP thresholds.
type ProfileHTTPTimingThresholds struct {
	DNS  ThresholdPair `json:"dns,omitempty"`
	TCP  ThresholdPair `json:"tcp,omitempty"`
	TLS  ThresholdPair `json:"tls,omitempty"`
	TTFB ThresholdPair `json:"ttfb,omitempty"`
}

// ProfileHealthChecks contains health check test configurations.
type ProfileHealthChecks struct {
	PingTargets    []ProfilePingTarget   `json:"ping_targets,omitempty"`
	TCPPorts       []ProfileTCPPort      `json:"tcp_ports,omitempty"`
	UDPPorts       []ProfileUDPPort      `json:"udp_ports,omitempty"`
	HTTPEndpoints  []ProfileHTTPEndpoint `json:"http_endpoints,omitempty"`
	RunPerformance bool                  `json:"run_performance"`
	RunSpeedtest   bool                  `json:"run_speedtest"`
	RunIperf       bool                  `json:"run_iperf"`
	RunDiscovery   bool                  `json:"run_discovery"`
}

// ProfilePingTarget represents a ping target.
type ProfilePingTarget struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

// ProfileTCPPort represents a TCP port test.
type ProfileTCPPort struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// ProfileUDPPort represents a UDP port test.
type ProfileUDPPort struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// ProfileHTTPEndpoint represents an HTTP endpoint test.
type ProfileHTTPEndpoint struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	Enabled        bool   `json:"enabled"`
}

// ProfileSpeedtest contains speedtest settings.
type ProfileSpeedtest struct {
	ServerID      string `json:"server_id,omitempty"`
	AutoRunOnLink bool   `json:"auto_run_on_link"`
}

// ProfileIperf contains iperf settings.
type ProfileIperf struct {
	AutoRunOnLink bool   `json:"auto_run_on_link"`
	Server        string `json:"server,omitempty"`
	Port          int    `json:"port,omitempty"`
	Protocol      string `json:"protocol,omitempty"`
	Direction     string `json:"direction,omitempty"`
	Duration      int    `json:"duration,omitempty"`
	ServerPort    int    `json:"server_port,omitempty"`
	EnableServer  bool   `json:"enable_server"`
}

// ProfileFABOptions controls FAB behavior.
type ProfileFABOptions struct {
	RunLink             bool `json:"run_link"`
	RunSwitch           bool `json:"run_switch"`
	RunVLAN             bool `json:"run_vlan"`
	RunIPConfig         bool `json:"run_ip_config"`
	RunGateway          bool `json:"run_gateway"`
	RunDNS              bool `json:"run_dns"`
	RunHealthChecks     bool `json:"run_health_checks"`
	RunNetworkDiscovery bool `json:"run_network_discovery"`
	RunSpeedtest        bool `json:"run_speedtest"`
	RunIperf            bool `json:"run_iperf"`
	RunPerformance      bool `json:"run_performance"`
	AutoScanOnLink      bool `json:"auto_scan_on_link"`
}

// ProfileDisplayOptions contains display preferences.
type ProfileDisplayOptions struct {
	ShowPublicIP bool   `json:"show_public_ip"`
	UnitSystem   string `json:"unit_system,omitempty"`
}

// ProfileDNS contains DNS test configuration.
type ProfileDNS struct {
	TestHostname string             `json:"test_hostname,omitempty"`
	Timeout      int64              `json:"timeout_ms,omitempty"`
	Servers      []ProfileDNSServer `json:"servers,omitempty"`
}

// ProfileDNSServer represents a DNS server to test.
type ProfileDNSServer struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

// ProfileSNMP contains SNMP configuration.
type ProfileSNMP struct {
	Communities   []string                  `json:"communities,omitempty"`
	V3Credentials []ProfileSNMPv3Credential `json:"v3_credentials,omitempty"`
	Timeout       int64                     `json:"timeout_ms,omitempty"`
	Retries       int                       `json:"retries,omitempty"`
	Port          int                       `json:"port,omitempty"`
}

// ProfileSNMPv3Credential contains SNMPv3 credentials.
type ProfileSNMPv3Credential struct {
	Name          string `json:"name"`
	Username      string `json:"username"`
	AuthProtocol  string `json:"auth_protocol,omitempty"`
	AuthPassword  string `json:"auth_password,omitempty"`
	PrivProtocol  string `json:"priv_protocol,omitempty"`
	PrivPassword  string `json:"priv_password,omitempty"`
	ContextName   string `json:"context_name,omitempty"`
	SecurityLevel string `json:"security_level,omitempty"`
}

// ProfileNetworkDiscovery contains network discovery settings.
type ProfileNetworkDiscovery struct {
	Enabled           bool                        `json:"enabled"`
	AutoScan          bool                        `json:"auto_scan"`
	ScanIntervalSecs  int64                       `json:"scan_interval_secs,omitempty"`
	AdditionalSubnets []ProfileSubnet             `json:"additional_subnets,omitempty"`
	Fingerprinting    ProfileFingerprintingConfig `json:"fingerprinting,omitempty"`
	IPv6Enabled       bool                        `json:"ipv6_enabled"`
}

// ProfileSubnet represents an additional subnet to scan.
type ProfileSubnet struct {
	CIDR    string `json:"cidr"`
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled"`
}

// ProfileFingerprintingConfig controls fingerprinting options.
type ProfileFingerprintingConfig struct {
	Enabled       bool `json:"enabled"`
	OSDetection   bool `json:"os_detection"`
	ServiceProbes bool `json:"service_probes"`
}

// ProfileLinkSettings contains interface speed/duplex configuration.
// These settings control how the network interface negotiates link parameters.
type ProfileLinkSettings struct {
	// AutoNegotiation enables automatic speed/duplex negotiation (default: true)
	AutoNegotiation bool `json:"auto_negotiation"`
	// Speed is the fixed link speed in Mbps when auto-negotiation is disabled
	// Valid values: "auto", "10", "100", "1000", "2500", "5000", "10000"
	Speed string `json:"speed,omitempty"`
	// Duplex is the duplex mode when auto-negotiation is disabled
	// Valid values: "auto", "full", "half"
	Duplex string `json:"duplex,omitempty"`
	// AvailableModes lists the speed/duplex combinations supported by the interface
	AvailableModes []string `json:"available_modes,omitempty"`
}

// ProfileCableTestSettings contains TDR cable diagnostic settings.
type ProfileCableTestSettings struct {
	// Enabled controls whether the cable test card is shown
	Enabled bool `json:"enabled"`
	// AutoRunOnLinkDown triggers cable test automatically when link goes down
	AutoRunOnLinkDown bool `json:"auto_run_on_link_down"`
}

// ProfileInterfaceConfigs stores the selected interfaces for a profile.
// Each profile can have multiple ethernet and wifi interfaces, each with independent settings.
// Version 3 changed from single interface per type to arrays supporting multiple interfaces.
type ProfileInterfaceConfigs struct {
	// Ethernet contains all configured ethernet interfaces for this profile.
	// Each interface can have its own thresholds and health check configurations.
	Ethernet []ProfileInterfaceSelection `json:"ethernet,omitempty" yaml:"ethernet,omitempty"`

	// WiFi contains all configured WiFi interfaces for this profile.
	// Each interface can have its own thresholds and health check configurations.
	WiFi []ProfileInterfaceSelection `json:"wifi,omitempty" yaml:"wifi,omitempty"`

	// ActiveEthernet is the name of the currently active ethernet interface.
	// Used to track which interface is being monitored when multiple are configured.
	ActiveEthernet string `json:"active_ethernet,omitempty" yaml:"active_ethernet,omitempty"`

	// ActiveWiFi is the name of the currently active WiFi interface.
	// Used to track which interface is being monitored when multiple are configured.
	ActiveWiFi string `json:"active_wifi,omitempty" yaml:"active_wifi,omitempty"`
}

// ProfileInterfaceSelection stores configuration for a selected interface within a profile.
// Per-interface settings (thresholds, health checks, etc.) will be added
// as the multi-interface implementation progresses.
type ProfileInterfaceSelection struct {
	// Name is the interface name (e.g., "eth0", "wlan0").
	Name string `json:"name" yaml:"name"`

	// Enabled indicates if this interface is active for testing.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Thresholds for this specific interface (optional override).
	Thresholds *ProfileThresholds `json:"thresholds,omitempty" yaml:"thresholds,omitempty"`

	// HealthChecks for this specific interface (optional override).
	HealthChecks *ProfileHealthChecks `json:"health_checks,omitempty" yaml:"health_checks,omitempty"`
}

// ProfileSettingsVersion is the current profile settings schema version.
// Supports multiple interfaces per type with per-interface settings.
const ProfileSettingsVersion = 1

// NewProfileSettings creates a new ProfileSettings with defaults from config.
func NewProfileSettings() *ProfileSettings {
	return &ProfileSettings{
		Version: ProfileSettingsVersion,
	}
}

// FromConfig extracts profile-specific settings from a Config.
func (ps *ProfileSettings) FromConfig(cfg *Config) {
	ps.Version = ProfileSettingsVersion

	// Thresholds
	ps.Thresholds = ProfileThresholds{
		DNS: ThresholdPair{
			Warning:  cfg.Thresholds.DNS.Warning.Milliseconds(),
			Critical: cfg.Thresholds.DNS.Critical.Milliseconds(),
		},
		Gateway: ThresholdPair{
			Warning:  cfg.Thresholds.Ping.Warning.Milliseconds(),
			Critical: cfg.Thresholds.Ping.Critical.Milliseconds(),
		},
		WiFi: WiFiThresholdPair{
			Warning:  cfg.Thresholds.WiFi.Signal.Warning,
			Critical: cfg.Thresholds.WiFi.Signal.Critical,
		},
		CustomPing: ThresholdPair{
			Warning:  cfg.Thresholds.CustomTests.Ping.Warning.Milliseconds(),
			Critical: cfg.Thresholds.CustomTests.Ping.Critical.Milliseconds(),
		},
		CustomTCP: ThresholdPair{
			Warning:  cfg.Thresholds.CustomTests.TCP.Warning.Milliseconds(),
			Critical: cfg.Thresholds.CustomTests.TCP.Critical.Milliseconds(),
		},
		CustomHTTP: ThresholdPair{
			Warning:  cfg.Thresholds.CustomTests.HTTP.Warning.Milliseconds(),
			Critical: cfg.Thresholds.CustomTests.HTTP.Critical.Milliseconds(),
		},
		HTTPTimings: ProfileHTTPTimingThresholds{
			DNS: ThresholdPair{
				Warning:  cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning.Milliseconds(),
				Critical: cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical.Milliseconds(),
			},
			TCP: ThresholdPair{
				Warning:  cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning.Milliseconds(),
				Critical: cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical.Milliseconds(),
			},
			TLS: ThresholdPair{
				Warning:  cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning.Milliseconds(),
				Critical: cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical.Milliseconds(),
			},
			TTFB: ThresholdPair{
				Warning:  cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning.Milliseconds(),
				Critical: cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical.Milliseconds(),
			},
		},
	}

	// Health Checks
	ps.HealthChecks = ProfileHealthChecks{
		RunPerformance: cfg.HealthChecks.RunPerformance,
		RunSpeedtest:   cfg.HealthChecks.RunSpeedtest,
		RunIperf:       cfg.HealthChecks.RunIperf,
		RunDiscovery:   cfg.HealthChecks.RunDiscovery,
	}
	for _, pt := range cfg.HealthChecks.PingTargets {
		ps.HealthChecks.PingTargets = append(ps.HealthChecks.PingTargets, ProfilePingTarget(pt))
	}
	for _, tp := range cfg.HealthChecks.TCPPorts {
		ps.HealthChecks.TCPPorts = append(ps.HealthChecks.TCPPorts, ProfileTCPPort(tp))
	}
	for _, up := range cfg.HealthChecks.UDPPorts {
		ps.HealthChecks.UDPPorts = append(ps.HealthChecks.UDPPorts, ProfileUDPPort(up))
	}
	for _, he := range cfg.HealthChecks.HTTPEndpoints {
		ps.HealthChecks.HTTPEndpoints = append(ps.HealthChecks.HTTPEndpoints, ProfileHTTPEndpoint(he))
	}

	// Speedtest
	ps.Speedtest = ProfileSpeedtest{
		ServerID:      cfg.Speedtest.ServerID,
		AutoRunOnLink: cfg.Speedtest.AutoRunOnLink,
	}

	// Iperf
	ps.Iperf = ProfileIperf{
		AutoRunOnLink: cfg.Iperf.AutoRunOnLink,
		Server:        cfg.Iperf.Server,
		Port:          cfg.Iperf.Port,
		Protocol:      cfg.Iperf.Protocol,
		Direction:     cfg.Iperf.Direction,
		Duration:      cfg.Iperf.Duration,
		ServerPort:    cfg.Iperf.ServerPort,
		EnableServer:  cfg.Iperf.EnableServer,
	}

	// FAB Options
	ps.FABOptions = ProfileFABOptions{
		RunLink:             cfg.FABOptions.RunLink,
		RunSwitch:           cfg.FABOptions.RunSwitch,
		RunVLAN:             cfg.FABOptions.RunVLAN,
		RunIPConfig:         cfg.FABOptions.RunIPConfig,
		RunGateway:          cfg.FABOptions.RunGateway,
		RunDNS:              cfg.FABOptions.RunDNS,
		RunHealthChecks:     cfg.FABOptions.RunHealthChecks,
		RunNetworkDiscovery: cfg.FABOptions.RunNetworkDiscovery,
		RunSpeedtest:        cfg.FABOptions.RunSpeedtest,
		RunIperf:            cfg.FABOptions.RunIperf,
		RunPerformance:      cfg.FABOptions.RunPerformance,
		AutoScanOnLink:      cfg.FABOptions.AutoScanOnLink,
	}

	// Display Options
	ps.DisplayOptions = ProfileDisplayOptions{
		ShowPublicIP: cfg.DisplayOptions.ShowPublicIP,
		UnitSystem:   cfg.DisplayOptions.UnitSystem,
	}

	// DNS
	ps.DNS = ProfileDNS{
		TestHostname: cfg.DNS.TestHostname,
		Timeout:      cfg.DNS.Timeout.Milliseconds(),
	}
	for _, ds := range cfg.DNS.Servers {
		ps.DNS.Servers = append(ps.DNS.Servers, ProfileDNSServer(ds))
	}

	// SNMP
	ps.SNMP = ProfileSNMP{
		Communities: cfg.SNMP.Communities,
		Timeout:     cfg.SNMP.Timeout.Milliseconds(),
		Retries:     cfg.SNMP.Retries,
		Port:        cfg.SNMP.Port,
	}
	for i := range cfg.SNMP.V3Credentials {
		v3 := &cfg.SNMP.V3Credentials[i]
		ps.SNMP.V3Credentials = append(ps.SNMP.V3Credentials, ProfileSNMPv3Credential{
			Name:          v3.Name,
			Username:      v3.Username,
			AuthProtocol:  v3.AuthProtocol,
			AuthPassword:  v3.AuthPassword,
			PrivProtocol:  v3.PrivProtocol,
			PrivPassword:  v3.PrivPassword,
			ContextName:   v3.ContextName,
			SecurityLevel: v3.SecurityLevel,
		})
	}

	// Network Discovery
	ps.NetworkDiscovery = ProfileNetworkDiscovery{
		Enabled:          cfg.NetworkDiscovery.Enabled,
		AutoScan:         cfg.NetworkDiscovery.AutoScan,
		ScanIntervalSecs: int64(cfg.NetworkDiscovery.ScanInterval.Seconds()),
		IPv6Enabled:      cfg.NetworkDiscovery.IPv6Enabled,
		Fingerprinting: ProfileFingerprintingConfig{
			Enabled:       cfg.NetworkDiscovery.Fingerprinting.Enabled,
			OSDetection:   cfg.NetworkDiscovery.Fingerprinting.OSDetection,
			ServiceProbes: cfg.NetworkDiscovery.Fingerprinting.ServiceProbes,
		},
	}
	for _, sn := range cfg.NetworkDiscovery.AdditionalSubnets {
		ps.NetworkDiscovery.AdditionalSubnets = append(ps.NetworkDiscovery.AdditionalSubnets, ProfileSubnet(sn))
	}
}

// ApplyTo applies profile settings to a Config.
// This modifies the Config in place with the profile's settings.
//
//nolint:gocyclo // Complexity is inherent to applying many settings sections.
func (ps *ProfileSettings) ApplyTo(cfg *Config) {
	// Thresholds
	cfg.Thresholds.DNS.Warning = time.Duration(ps.Thresholds.DNS.Warning) * time.Millisecond
	cfg.Thresholds.DNS.Critical = time.Duration(ps.Thresholds.DNS.Critical) * time.Millisecond
	cfg.Thresholds.Ping.Warning = time.Duration(ps.Thresholds.Gateway.Warning) * time.Millisecond
	cfg.Thresholds.Ping.Critical = time.Duration(ps.Thresholds.Gateway.Critical) * time.Millisecond
	cfg.Thresholds.WiFi.Signal.Warning = ps.Thresholds.WiFi.Warning
	cfg.Thresholds.WiFi.Signal.Critical = ps.Thresholds.WiFi.Critical
	cfg.Thresholds.CustomTests.Ping.Warning = time.Duration(ps.Thresholds.CustomPing.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.Ping.Critical = time.Duration(ps.Thresholds.CustomPing.Critical) * time.Millisecond
	cfg.Thresholds.CustomTests.TCP.Warning = time.Duration(ps.Thresholds.CustomTCP.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.TCP.Critical = time.Duration(ps.Thresholds.CustomTCP.Critical) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTP.Warning = time.Duration(ps.Thresholds.CustomHTTP.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTP.Critical = time.Duration(ps.Thresholds.CustomHTTP.Critical) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.DNS.Warning = time.Duration(ps.Thresholds.HTTPTimings.DNS.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.DNS.Critical = time.Duration(ps.Thresholds.HTTPTimings.DNS.Critical) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.TCP.Warning = time.Duration(ps.Thresholds.HTTPTimings.TCP.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.TCP.Critical = time.Duration(ps.Thresholds.HTTPTimings.TCP.Critical) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.TLS.Warning = time.Duration(ps.Thresholds.HTTPTimings.TLS.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.TLS.Critical = time.Duration(ps.Thresholds.HTTPTimings.TLS.Critical) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Warning = time.Duration(ps.Thresholds.HTTPTimings.TTFB.Warning) * time.Millisecond
	cfg.Thresholds.CustomTests.HTTPTimings.TTFB.Critical = time.Duration(ps.Thresholds.HTTPTimings.TTFB.Critical) * time.Millisecond

	// Health Checks
	cfg.HealthChecks.RunPerformance = ps.HealthChecks.RunPerformance
	cfg.HealthChecks.RunSpeedtest = ps.HealthChecks.RunSpeedtest
	cfg.HealthChecks.RunIperf = ps.HealthChecks.RunIperf
	cfg.HealthChecks.RunDiscovery = ps.HealthChecks.RunDiscovery

	cfg.HealthChecks.PingTargets = make([]PingTarget, 0, len(ps.HealthChecks.PingTargets))
	for _, pt := range ps.HealthChecks.PingTargets {
		cfg.HealthChecks.PingTargets = append(cfg.HealthChecks.PingTargets, PingTarget(pt))
	}

	cfg.HealthChecks.TCPPorts = make([]TCPPortTest, 0, len(ps.HealthChecks.TCPPorts))
	for _, tp := range ps.HealthChecks.TCPPorts {
		cfg.HealthChecks.TCPPorts = append(cfg.HealthChecks.TCPPorts, TCPPortTest(tp))
	}

	cfg.HealthChecks.UDPPorts = make([]UDPPortTest, 0, len(ps.HealthChecks.UDPPorts))
	for _, up := range ps.HealthChecks.UDPPorts {
		cfg.HealthChecks.UDPPorts = append(cfg.HealthChecks.UDPPorts, UDPPortTest(up))
	}

	cfg.HealthChecks.HTTPEndpoints = make([]HTTPEndpoint, 0, len(ps.HealthChecks.HTTPEndpoints))
	for _, he := range ps.HealthChecks.HTTPEndpoints {
		cfg.HealthChecks.HTTPEndpoints = append(cfg.HealthChecks.HTTPEndpoints, HTTPEndpoint(he))
	}

	// Speedtest
	cfg.Speedtest.ServerID = ps.Speedtest.ServerID
	cfg.Speedtest.AutoRunOnLink = ps.Speedtest.AutoRunOnLink

	// Iperf
	cfg.Iperf.AutoRunOnLink = ps.Iperf.AutoRunOnLink
	cfg.Iperf.Server = ps.Iperf.Server
	if ps.Iperf.Port > 0 {
		cfg.Iperf.Port = ps.Iperf.Port
	}
	if ps.Iperf.Protocol != "" {
		cfg.Iperf.Protocol = ps.Iperf.Protocol
	}
	if ps.Iperf.Direction != "" {
		cfg.Iperf.Direction = ps.Iperf.Direction
	}
	if ps.Iperf.Duration > 0 {
		cfg.Iperf.Duration = ps.Iperf.Duration
	}
	if ps.Iperf.ServerPort > 0 {
		cfg.Iperf.ServerPort = ps.Iperf.ServerPort
	}
	cfg.Iperf.EnableServer = ps.Iperf.EnableServer

	// FAB Options
	cfg.FABOptions.RunLink = ps.FABOptions.RunLink
	cfg.FABOptions.RunSwitch = ps.FABOptions.RunSwitch
	cfg.FABOptions.RunVLAN = ps.FABOptions.RunVLAN
	cfg.FABOptions.RunIPConfig = ps.FABOptions.RunIPConfig
	cfg.FABOptions.RunGateway = ps.FABOptions.RunGateway
	cfg.FABOptions.RunDNS = ps.FABOptions.RunDNS
	cfg.FABOptions.RunHealthChecks = ps.FABOptions.RunHealthChecks
	cfg.FABOptions.RunNetworkDiscovery = ps.FABOptions.RunNetworkDiscovery
	cfg.FABOptions.RunSpeedtest = ps.FABOptions.RunSpeedtest
	cfg.FABOptions.RunIperf = ps.FABOptions.RunIperf
	cfg.FABOptions.RunPerformance = ps.FABOptions.RunPerformance
	cfg.FABOptions.AutoScanOnLink = ps.FABOptions.AutoScanOnLink

	// Display Options
	cfg.DisplayOptions.ShowPublicIP = ps.DisplayOptions.ShowPublicIP
	if ps.DisplayOptions.UnitSystem != "" {
		cfg.DisplayOptions.UnitSystem = ps.DisplayOptions.UnitSystem
	}

	// DNS
	if ps.DNS.TestHostname != "" {
		cfg.DNS.TestHostname = ps.DNS.TestHostname
	}
	if ps.DNS.Timeout > 0 {
		cfg.DNS.Timeout = time.Duration(ps.DNS.Timeout) * time.Millisecond
	}
	cfg.DNS.Servers = make([]DNSServer, 0, len(ps.DNS.Servers))
	for _, ds := range ps.DNS.Servers {
		cfg.DNS.Servers = append(cfg.DNS.Servers, DNSServer(ds))
	}

	// SNMP
	if len(ps.SNMP.Communities) > 0 {
		cfg.SNMP.Communities = ps.SNMP.Communities
	}
	if ps.SNMP.Timeout > 0 {
		cfg.SNMP.Timeout = time.Duration(ps.SNMP.Timeout) * time.Millisecond
	}
	if ps.SNMP.Retries > 0 {
		cfg.SNMP.Retries = ps.SNMP.Retries
	}
	if ps.SNMP.Port > 0 {
		cfg.SNMP.Port = ps.SNMP.Port
	}
	cfg.SNMP.V3Credentials = make([]SNMPv3Credential, 0, len(ps.SNMP.V3Credentials))
	for i := range ps.SNMP.V3Credentials {
		v3 := &ps.SNMP.V3Credentials[i]
		cfg.SNMP.V3Credentials = append(cfg.SNMP.V3Credentials, SNMPv3Credential{
			Name:          v3.Name,
			Username:      v3.Username,
			AuthProtocol:  v3.AuthProtocol,
			AuthPassword:  v3.AuthPassword,
			PrivProtocol:  v3.PrivProtocol,
			PrivPassword:  v3.PrivPassword,
			ContextName:   v3.ContextName,
			SecurityLevel: v3.SecurityLevel,
		})
	}

	// Network Discovery
	cfg.NetworkDiscovery.Enabled = ps.NetworkDiscovery.Enabled
	cfg.NetworkDiscovery.AutoScan = ps.NetworkDiscovery.AutoScan
	if ps.NetworkDiscovery.ScanIntervalSecs > 0 {
		cfg.NetworkDiscovery.ScanInterval = time.Duration(ps.NetworkDiscovery.ScanIntervalSecs) * time.Second
	}
	cfg.NetworkDiscovery.IPv6Enabled = ps.NetworkDiscovery.IPv6Enabled
	cfg.NetworkDiscovery.Fingerprinting.Enabled = ps.NetworkDiscovery.Fingerprinting.Enabled
	cfg.NetworkDiscovery.Fingerprinting.OSDetection = ps.NetworkDiscovery.Fingerprinting.OSDetection
	cfg.NetworkDiscovery.Fingerprinting.ServiceProbes = ps.NetworkDiscovery.Fingerprinting.ServiceProbes

	cfg.NetworkDiscovery.AdditionalSubnets = make([]SubnetConfig, 0, len(ps.NetworkDiscovery.AdditionalSubnets))
	for _, sn := range ps.NetworkDiscovery.AdditionalSubnets {
		cfg.NetworkDiscovery.AdditionalSubnets = append(cfg.NetworkDiscovery.AdditionalSubnets, SubnetConfig(sn))
	}
}

// ToJSON serializes ProfileSettings to JSON.
func (ps *ProfileSettings) ToJSON() (string, error) {
	data, err := json.Marshal(ps)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSON deserializes ProfileSettings from JSON.
func (ps *ProfileSettings) FromJSON(data string) error {
	if data == "" {
		return nil
	}
	return json.Unmarshal([]byte(data), ps)
}

// ParseProfileSettings parses a JSON string into ProfileSettings.
func ParseProfileSettings(jsonStr string) (*ProfileSettings, error) {
	ps := NewProfileSettings()
	if err := ps.FromJSON(jsonStr); err != nil {
		return nil, err
	}
	// Migrate older versions.
	ps.Migrate()
	return ps, nil
}

// Migrate updates profile settings to the current version if needed.
// Currently a no-op as this is the initial version with multi-interface support.
func (ps *ProfileSettings) Migrate() {
	ps.Version = ProfileSettingsVersion
}

// AddEthernetInterface adds or updates an ethernet interface in this profile.
// If an interface with the same name exists, it is updated; otherwise, a new one is added.
func (ps *ProfileSettings) AddEthernetInterface(name string, enabled bool) {
	for i, iface := range ps.Interfaces.Ethernet {
		if iface.Name == name {
			ps.Interfaces.Ethernet[i].Enabled = enabled
			return
		}
	}
	ps.Interfaces.Ethernet = append(ps.Interfaces.Ethernet, ProfileInterfaceSelection{
		Name:    name,
		Enabled: enabled,
	})
}

// AddWiFiInterface adds or updates a WiFi interface in this profile.
// If an interface with the same name exists, it is updated; otherwise, a new one is added.
func (ps *ProfileSettings) AddWiFiInterface(name string, enabled bool) {
	for i, iface := range ps.Interfaces.WiFi {
		if iface.Name == name {
			ps.Interfaces.WiFi[i].Enabled = enabled
			return
		}
	}
	ps.Interfaces.WiFi = append(ps.Interfaces.WiFi, ProfileInterfaceSelection{
		Name:    name,
		Enabled: enabled,
	})
}

// RemoveEthernetInterface removes an ethernet interface from this profile.
func (ps *ProfileSettings) RemoveEthernetInterface(name string) {
	for i, iface := range ps.Interfaces.Ethernet {
		if iface.Name == name {
			ps.Interfaces.Ethernet = append(ps.Interfaces.Ethernet[:i], ps.Interfaces.Ethernet[i+1:]...)
			// Clear active if we removed the active interface
			if ps.Interfaces.ActiveEthernet == name {
				ps.Interfaces.ActiveEthernet = ""
			}
			return
		}
	}
}

// RemoveWiFiInterface removes a WiFi interface from this profile.
func (ps *ProfileSettings) RemoveWiFiInterface(name string) {
	for i, iface := range ps.Interfaces.WiFi {
		if iface.Name == name {
			ps.Interfaces.WiFi = append(ps.Interfaces.WiFi[:i], ps.Interfaces.WiFi[i+1:]...)
			// Clear active if we removed the active interface
			if ps.Interfaces.ActiveWiFi == name {
				ps.Interfaces.ActiveWiFi = ""
			}
			return
		}
	}
}

// SetActiveEthernetInterface sets the active ethernet interface.
// The interface must already be in the Ethernet list.
func (ps *ProfileSettings) SetActiveEthernetInterface(name string) bool {
	for _, iface := range ps.Interfaces.Ethernet {
		if iface.Name == name {
			ps.Interfaces.ActiveEthernet = name
			return true
		}
	}
	return false
}

// SetActiveWiFiInterface sets the active WiFi interface.
// The interface must already be in the WiFi list.
func (ps *ProfileSettings) SetActiveWiFiInterface(name string) bool {
	for _, iface := range ps.Interfaces.WiFi {
		if iface.Name == name {
			ps.Interfaces.ActiveWiFi = name
			return true
		}
	}
	return false
}

// GetActiveEthernetInterface returns the active ethernet interface configuration.
// Returns nil if no active interface is set or the active interface is not in the list.
func (ps *ProfileSettings) GetActiveEthernetInterface() *ProfileInterfaceSelection {
	if ps.Interfaces.ActiveEthernet == "" {
		return nil
	}
	for i, iface := range ps.Interfaces.Ethernet {
		if iface.Name == ps.Interfaces.ActiveEthernet {
			return &ps.Interfaces.Ethernet[i]
		}
	}
	return nil
}

// GetActiveWiFiInterface returns the active WiFi interface configuration.
// Returns nil if no active interface is set or the active interface is not in the list.
func (ps *ProfileSettings) GetActiveWiFiInterface() *ProfileInterfaceSelection {
	if ps.Interfaces.ActiveWiFi == "" {
		return nil
	}
	for i, iface := range ps.Interfaces.WiFi {
		if iface.Name == ps.Interfaces.ActiveWiFi {
			return &ps.Interfaces.WiFi[i]
		}
	}
	return nil
}

// GetEthernetInterfaceName returns the active ethernet interface name, or empty string.
// This is a convenience method for backwards compatibility.
func (ps *ProfileSettings) GetEthernetInterfaceName() string {
	return ps.Interfaces.ActiveEthernet
}

// GetWiFiInterfaceName returns the active WiFi interface name, or empty string.
// This is a convenience method for backwards compatibility.
func (ps *ProfileSettings) GetWiFiInterfaceName() string {
	return ps.Interfaces.ActiveWiFi
}

// GetEthernetInterface returns the configuration for a specific ethernet interface.
// Returns nil if the interface is not configured in this profile.
func (ps *ProfileSettings) GetEthernetInterface(name string) *ProfileInterfaceSelection {
	for i, iface := range ps.Interfaces.Ethernet {
		if iface.Name == name {
			return &ps.Interfaces.Ethernet[i]
		}
	}
	return nil
}

// GetWiFiInterface returns the configuration for a specific WiFi interface.
// Returns nil if the interface is not configured in this profile.
func (ps *ProfileSettings) GetWiFiInterface(name string) *ProfileInterfaceSelection {
	for i, iface := range ps.Interfaces.WiFi {
		if iface.Name == name {
			return &ps.Interfaces.WiFi[i]
		}
	}
	return nil
}

// SetEthernetInterface adds/updates an ethernet interface and sets it as active.
// This is a convenience method for backwards compatibility with single-interface usage.
func (ps *ProfileSettings) SetEthernetInterface(name string, enabled bool) {
	ps.AddEthernetInterface(name, enabled)
	ps.Interfaces.ActiveEthernet = name
}

// SetWiFiInterface adds/updates a WiFi interface and sets it as active.
// This is a convenience method for backwards compatibility with single-interface usage.
func (ps *ProfileSettings) SetWiFiInterface(name string, enabled bool) {
	ps.AddWiFiInterface(name, enabled)
	ps.Interfaces.ActiveWiFi = name
}
