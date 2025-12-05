// Package config handles application configuration.
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Server     ServerConfig     `yaml:"server"`
	Interface  InterfaceConfig  `yaml:"interface"`
	VLAN       VLANConfig       `yaml:"vlan"`
	IP         IPConfig         `yaml:"ip"`
	Discovery  DiscoveryConfig  `yaml:"discovery"`
	DNS        DNSConfig        `yaml:"dns"`
	Tests      TestsConfig      `yaml:"tests"`
	Speedtest  SpeedtestConfig  `yaml:"speedtest"`
	Thresholds ThresholdsConfig `yaml:"thresholds"`
	Auth       AuthConfig       `yaml:"auth"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port     int    `yaml:"port"`
	HTTPS    bool   `yaml:"https"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// InterfaceConfig contains network interface settings.
type InterfaceConfig struct {
	Default   string   `yaml:"default"`
	Fallbacks []string `yaml:"fallbacks"`
	WiFi      string   `yaml:"wifi,omitempty"` // Separate WiFi interface (optional)
}

// VLANConfig contains VLAN settings.
type VLANConfig struct {
	Enabled bool `yaml:"enabled"`
	ID      int  `yaml:"id"`
}

// IPConfig contains IP configuration settings.
type IPConfig struct {
	Mode   string       `yaml:"mode"` // "dhcp" or "static"
	Static *StaticIP    `yaml:"static,omitempty"`
}

// StaticIP contains static IP configuration.
type StaticIP struct {
	Address string   `yaml:"address"`
	Netmask string   `yaml:"netmask"`
	Gateway string   `yaml:"gateway"`
	DNS     []string `yaml:"dns"`
}

// DiscoveryConfig contains switch discovery settings.
type DiscoveryConfig struct {
	Protocol string        `yaml:"protocol"` // "auto", "lldp", "cdp", "edp", "fdp"
	Timeout  time.Duration `yaml:"timeout"`
}

// DNSConfig contains DNS testing settings.
type DNSConfig struct {
	TestHostname string        `yaml:"test_hostname"`
	Timeout      time.Duration `yaml:"timeout"`
	Servers      []DNSServer   `yaml:"servers,omitempty"` // Additional DNS servers to test
}

// DNSServer represents a DNS server configuration.
type DNSServer struct {
	Address string `yaml:"address"`
	Enabled bool   `yaml:"enabled"`
}

// TestsConfig contains custom test configurations.
type TestsConfig struct {
	PingTargets   []PingTarget   `yaml:"ping_targets"`
	TCPPorts      []TCPPortTest  `yaml:"tcp_ports"`
	UDPPorts      []UDPPortTest  `yaml:"udp_ports"`
	HTTPEndpoints []HTTPEndpoint `yaml:"http_endpoints"`
}

// PingTarget represents a custom ping target.
type PingTarget struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	Enabled bool   `yaml:"enabled"`
}

// TCPPortTest represents a custom TCP port test.
type TCPPortTest struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Enabled bool   `yaml:"enabled"`
}

// UDPPortTest represents a custom UDP port test.
type UDPPortTest struct {
	Name    string `yaml:"name"`
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Enabled bool   `yaml:"enabled"`
}

// HTTPEndpoint represents a custom HTTP endpoint test.
type HTTPEndpoint struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	ExpectedStatus int    `yaml:"expected_status"`
	Enabled        bool   `yaml:"enabled"`
}

// SpeedtestConfig contains speedtest settings.
type SpeedtestConfig struct {
	ServerID      string `yaml:"server_id"`       // Specific server ID (empty = auto)
	AutoRunOnLink bool   `yaml:"auto_run_on_link"` // Run automatically when link comes up
}

// ThresholdsConfig contains all threshold settings.
type ThresholdsConfig struct {
	DHCP        DHCPThresholds    `yaml:"dhcp"`
	DNS         Threshold         `yaml:"dns"`
	Ping        Threshold         `yaml:"ping"`
	WiFi        WiFiThresholds    `yaml:"wifi"`
	CustomTests CustomThresholds  `yaml:"custom_tests"`
}

// CustomThresholds contains thresholds for custom tests.
type CustomThresholds struct {
	Ping       Threshold        `yaml:"ping"`        // Custom ping targets
	TCP        Threshold        `yaml:"tcp"`         // TCP port tests
	UDP        Threshold        `yaml:"udp"`         // UDP port tests
	HTTP       Threshold        `yaml:"http"`        // HTTP endpoint tests
	CertExpiry CertExpiryThreshold `yaml:"cert_expiry"` // Certificate expiry (days)
}

// CertExpiryThreshold contains certificate expiry thresholds in days.
type CertExpiryThreshold struct {
	Warning  int `yaml:"warning"`  // Days until warning (e.g., 30)
	Critical int `yaml:"critical"` // Days until critical (e.g., 7)
}

// DHCPThresholds contains DHCP-specific thresholds.
type DHCPThresholds struct {
	Total    Threshold `yaml:"total"`
	PerPhase Threshold `yaml:"per_phase"`
}

// Threshold contains warning and critical values.
type Threshold struct {
	Warning  time.Duration `yaml:"warning"`
	Critical time.Duration `yaml:"critical"`
}

// WiFiThresholds contains WiFi signal thresholds.
type WiFiThresholds struct {
	Signal SignalThreshold `yaml:"signal"`
}

// SignalThreshold contains signal strength thresholds in dBm.
type SignalThreshold struct {
	Warning  int `yaml:"warning"`
	Critical int `yaml:"critical"`
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	DefaultUsername     string        `yaml:"default_username"`
	DefaultPasswordHash string        `yaml:"default_password_hash"`
	SessionTimeout      time.Duration `yaml:"session_timeout"`
	JWTSecret           string        `yaml:"jwt_secret,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:  8443,
			HTTPS: true,
		},
		Interface: InterfaceConfig{
			Default:   "eth0",
			Fallbacks: []string{"enp0s3", "wlan0"},
		},
		VLAN: VLANConfig{
			Enabled: false,
			ID:      0,
		},
		IP: IPConfig{
			Mode: "dhcp",
		},
		Discovery: DiscoveryConfig{
			Protocol: "auto",
			Timeout:  30 * time.Second,
		},
		DNS: DNSConfig{
			TestHostname: "google.com",
			Timeout:      5 * time.Second,
		},
		Tests: TestsConfig{
			PingTargets:   []PingTarget{},
			TCPPorts:      []TCPPortTest{},
			UDPPorts:      []UDPPortTest{},
			HTTPEndpoints: []HTTPEndpoint{},
		},
		Speedtest: SpeedtestConfig{
			ServerID:      "",    // Auto-select closest
			AutoRunOnLink: false, // Disabled by default
		},
		Thresholds: ThresholdsConfig{
			DHCP: DHCPThresholds{
				Total:    Threshold{Warning: 500 * time.Millisecond, Critical: 2 * time.Second},
				PerPhase: Threshold{Warning: 200 * time.Millisecond, Critical: 1 * time.Second},
			},
			DNS:  Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
			Ping: Threshold{Warning: 50 * time.Millisecond, Critical: 200 * time.Millisecond},
			WiFi: WiFiThresholds{
				Signal: SignalThreshold{Warning: -70, Critical: -80},
			},
			CustomTests: CustomThresholds{
				Ping:       Threshold{Warning: 50 * time.Millisecond, Critical: 100 * time.Millisecond},
				TCP:        Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
				UDP:        Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
				HTTP:       Threshold{Warning: 500 * time.Millisecond, Critical: 2 * time.Second},
				CertExpiry: CertExpiryThreshold{Warning: 30, Critical: 7}, // Days
			},
		},
		Auth: AuthConfig{
			DefaultUsername:     "admin",
			DefaultPasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrqPqMqYjP8tTO.C5p3QBrC4qZlFIG", // "netscope"
			SessionTimeout:      24 * time.Hour,
		},
	}
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes configuration to a YAML file.
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
