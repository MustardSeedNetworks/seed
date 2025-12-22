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

	// Thresholds for network tests
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
	Profile           string                      `json:"profile,omitempty"`
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

// ProfileSettingsVersion is the current profile settings schema version.
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
		Profile:          string(cfg.NetworkDiscovery.Profile),
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
	if ps.NetworkDiscovery.Profile != "" {
		cfg.NetworkDiscovery.Profile = DiscoveryProfile(ps.NetworkDiscovery.Profile)
	}
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
	return ps, nil
}
