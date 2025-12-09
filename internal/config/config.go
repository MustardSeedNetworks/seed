// Package config handles application configuration.
package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	Server           ServerConfig           `yaml:"server"`
	Interface        InterfaceConfig        `yaml:"interface"`
	VLAN             VLANConfig             `yaml:"vlan"`
	IP               IPConfig               `yaml:"ip"`
	Discovery        DiscoveryConfig        `yaml:"discovery"`
	NetworkDiscovery NetworkDiscoveryConfig `yaml:"network_discovery"`
	DNS              DNSConfig              `yaml:"dns"`
	Tests            TestsConfig            `yaml:"tests"`
	Speedtest        SpeedtestConfig        `yaml:"speedtest"`
	Thresholds       ThresholdsConfig       `yaml:"thresholds"`
	Auth             AuthConfig             `yaml:"auth"`
	Security         SecurityConfig         `yaml:"security"`
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port             int    `yaml:"port"`
	HTTPS            bool   `yaml:"https"`
	CertFile         string `yaml:"cert_file"`
	KeyFile          string `yaml:"key_file"`
	LogAccessToken   string `yaml:"log_access_token,omitempty"`   // Optional token required to read /api/logs
	LogAccessHeader  string `yaml:"log_access_header,omitempty"`  // Header name to supply the token (default: X-Log-Token)
	RequireLogAccess bool   `yaml:"require_log_access,omitempty"` // Force token check even if empty token (future-proof)
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
	Mode   string    `yaml:"mode"` // "dhcp" or "static"
	Static *StaticIP `yaml:"static,omitempty"`
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

// NetworkDiscoveryConfig contains network device discovery settings.
type NetworkDiscoveryConfig struct {
	Enabled           bool           `yaml:"enabled"`            // Enable network discovery
	ARPScanWorkers    int            `yaml:"arp_scan_workers"`   // Number of concurrent ping workers (default 50)
	PingTimeout       time.Duration  `yaml:"ping_timeout"`       // Timeout for each ping (default 500ms)
	ScanTimeout       time.Duration  `yaml:"scan_timeout"`       // Total scan timeout (default 30s)
	AutoScan          bool           `yaml:"auto_scan"`          // Auto-scan on startup/interface change
	ScanInterval      time.Duration  `yaml:"scan_interval"`      // Interval for auto-scan (0 = disabled)
	OUIFilePath       string         `yaml:"oui_file_path"`      // Path to IEEE OUI file (oui.txt)
	AdditionalSubnets []SubnetConfig `yaml:"additional_subnets"` // Additional subnets to scan
}

// SubnetConfig represents a configured subnet for network discovery.
type SubnetConfig struct {
	CIDR    string `yaml:"cidr"`    // CIDR notation (e.g., "10.0.0.0/24")
	Name    string `yaml:"name"`    // Friendly name (e.g., "Server VLAN")
	Enabled bool   `yaml:"enabled"` // Whether to scan this subnet
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
	PingTargets    []PingTarget   `yaml:"ping_targets"`
	TCPPorts       []TCPPortTest  `yaml:"tcp_ports"`
	UDPPorts       []UDPPortTest  `yaml:"udp_ports"`
	HTTPEndpoints  []HTTPEndpoint `yaml:"http_endpoints"`
	RunPerformance bool           `yaml:"run_performance"` // Master toggle for speedtest + iperf
	RunSpeedtest   bool           `yaml:"run_speedtest"`   // Toggle internet speed test
	RunIperf       bool           `yaml:"run_iperf"`       // Toggle LAN iperf test
	RunDiscovery   bool           `yaml:"run_discovery"`   // Toggle network discovery card
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
	ServerID      string `yaml:"server_id"`        // Specific server ID (empty = auto)
	AutoRunOnLink bool   `yaml:"auto_run_on_link"` // Run automatically when link comes up
}

// ThresholdsConfig contains all threshold settings.
type ThresholdsConfig struct {
	DHCP        DHCPThresholds   `yaml:"dhcp"`
	DNS         Threshold        `yaml:"dns"`
	Ping        Threshold        `yaml:"ping"`
	WiFi        WiFiThresholds   `yaml:"wifi"`
	CustomTests CustomThresholds `yaml:"custom_tests"`
}

// CustomThresholds contains thresholds for custom tests.
type CustomThresholds struct {
	Ping       Threshold           `yaml:"ping"`        // Custom ping targets
	TCP        Threshold           `yaml:"tcp"`         // TCP port tests
	UDP        Threshold           `yaml:"udp"`         // UDP port tests
	HTTP       Threshold           `yaml:"http"`        // HTTP endpoint tests
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

// SecurityConfig contains security settings for CORS and WebSocket origins.
type SecurityConfig struct {
	// AllowedOrigins specifies explicit origins allowed for CORS and WebSocket.
	// If empty, defaults to RFC 1918 private network ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x).
	// Use "*" to allow all origins (not recommended for production).
	// Examples: ["http://192.168.1.100:8080", "https://netscope.local"]
	AllowedOrigins []string `yaml:"allowed_origins"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:            8443,
			HTTPS:           true,
			LogAccessHeader: "X-Log-Token",
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
		NetworkDiscovery: NetworkDiscoveryConfig{
			Enabled:           true,
			ARPScanWorkers:    50,
			PingTimeout:       500 * time.Millisecond,
			ScanTimeout:       30 * time.Second,
			AutoScan:          true,             // Auto-scan on startup by default
			ScanInterval:      0,                // Disabled by default
			OUIFilePath:       "oui.txt",        // IEEE OUI file (download from https://standards-oui.ieee.org/oui/oui.txt)
			AdditionalSubnets: []SubnetConfig{}, // No additional subnets by default
		},
		DNS: DNSConfig{
			TestHostname: "google.com",
			Timeout:      5 * time.Second,
		},
		Tests: TestsConfig{
			PingTargets:    []PingTarget{},
			TCPPorts:       []TCPPortTest{},
			UDPPorts:       []UDPPortTest{},
			HTTPEndpoints:  []HTTPEndpoint{},
			RunPerformance: true,
			RunSpeedtest:   true,
			RunIperf:       true,
			RunDiscovery:   true,
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
			DefaultPasswordHash: "$2y$10$1w5ktZnNS0UxbOvHKH2.hu01jsPh2RjkszVsP.7jR5cOZYa4oAI52", // "netscope"
			SessionTimeout:      24 * time.Hour,
		},
		Security: SecurityConfig{
			AllowedOrigins: []string{}, // Empty = use RFC 1918 defaults
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
	return os.WriteFile(path, data, 0o600)
}
