// Package config handles application configuration.
package config

import (
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	// ErrInsecureCredentials is returned when default credentials are detected.
	ErrInsecureCredentials = errors.New("insecure default credentials detected")
)

// Config represents the application configuration.
// All API handlers must use Lock/Unlock for writes or RLock/RUnlock for reads
// to prevent concurrent config update race conditions.
type Config struct {
	mu               sync.RWMutex           `yaml:"-"` // Protects concurrent access
	Server           ServerConfig           `yaml:"server"`
	Interface        InterfaceConfig        `yaml:"interface"`
	VLAN             VLANConfig             `yaml:"vlan"`
	IP               IPConfig               `yaml:"ip"`
	Discovery        DiscoveryConfig        `yaml:"discovery"`
	NetworkDiscovery NetworkDiscoveryConfig `yaml:"network_discovery"`
	DNS              DNSConfig              `yaml:"dns"`
	Tests            TestsConfig            `yaml:"tests"`
	Speedtest        SpeedtestConfig        `yaml:"speedtest"`
	Iperf            IperfConfig            `yaml:"iperf"`
	Thresholds       ThresholdsConfig       `yaml:"thresholds"`
	Auth             AuthConfig             `yaml:"auth"`
	Security         SecurityConfig         `yaml:"security"`
	DHCP             DHCPConfig             `yaml:"dhcp"`
	SNMP             SNMPConfig             `yaml:"snmp"`
	FABOptions       FABOptionsConfig       `yaml:"fab_options"`
	DisplayOptions   DisplayOptionsConfig   `yaml:"display_options"`
}

// Lock acquires a write lock on the config.
// Must be called before modifying config and followed by Unlock.
func (c *Config) Lock() {
	c.mu.Lock()
}

// Unlock releases the write lock on the config.
func (c *Config) Unlock() {
	c.mu.Unlock()
}

// RLock acquires a read lock on the config.
// Must be called before reading config and followed by RUnlock.
func (c *Config) RLock() {
	c.mu.RLock()
}

// RUnlock releases the read lock on the config.
func (c *Config) RUnlock() {
	c.mu.RUnlock()
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port             int    `yaml:"port"`
	HTTPS            bool   `yaml:"https"`
	HTTPRedirectPort int    `yaml:"http_redirect_port,omitempty"` // Port for HTTP→HTTPS redirect (0 = disabled, typically 80)
	CertFile         string `yaml:"cert_file"`
	KeyFile          string `yaml:"key_file"`
	LogAccessToken   string `yaml:"log_access_token,omitempty"`   // Optional token required to read /api/logs
	LogAccessHeader  string `yaml:"log_access_header,omitempty"`  // Header name to supply the token (default: X-Log-Token)
	RequireLogAccess bool   `yaml:"require_log_access,omitempty"` // Force token check even if empty token (future-proof)

	// ACME/Let's Encrypt automatic certificate management
	ACME ACMEConfig `yaml:"acme,omitempty"`
}

// ACMEConfig contains ACME/Let's Encrypt certificate settings.
type ACMEConfig struct {
	Enabled  bool   `yaml:"enabled"`            // Enable automatic certificate management
	Domain   string `yaml:"domain"`             // Domain name for the certificate (e.g., "netscope.example.com")
	Email    string `yaml:"email"`              // Contact email for Let's Encrypt notifications
	CacheDir string `yaml:"cache_dir,omitempty"` // Directory to cache certificates (default: "certs/acme")
	Staging  bool   `yaml:"staging,omitempty"`   // Use Let's Encrypt staging server (for testing)
}

// InterfaceConfig contains network interface settings.
type InterfaceConfig struct {
	Default          string        `yaml:"default"`
	Fallbacks        []string      `yaml:"fallbacks"`
	WiFi             string        `yaml:"wifi,omitempty"`         // Separate WiFi interface (optional)
	StartupRetries   int           `yaml:"startup_retries"`        // Number of retries when finding interface at startup (fixes #528)
	StartupRetryWait time.Duration `yaml:"startup_retry_wait"`     // Delay between startup retries (fixes #528)
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

// DiscoveryProfile defines preset discovery modes for ease of use.
type DiscoveryProfile string

const (
	// ProfileStealth performs no active scanning - only passive listening (LLDP, CDP, DHCP).
	// Safest for sensitive networks, generates zero noise.
	ProfileStealth DiscoveryProfile = "stealth"

	// ProfileStandard performs safe active discovery using ARP/ICMP on local subnet only.
	// Recommended for most networks, low noise.
	ProfileStandard DiscoveryProfile = "standard"

	// ProfileFullScan performs aggressive discovery including port scans and additional subnets.
	// High noise, may trigger IDS/IPS.
	ProfileFullScan DiscoveryProfile = "full_scan"

	// ProfileCustom allows fine-grained control over all discovery methods.
	ProfileCustom DiscoveryProfile = "custom"
)

// NetworkDiscoveryConfig contains network device discovery settings.
type NetworkDiscoveryConfig struct {
	// Profile selects a preset discovery configuration.
	// Options: "stealth", "standard", "full_scan", "custom"
	Profile DiscoveryProfile `yaml:"profile"`

	// CustomOptions are used only when Profile is "custom".
	CustomOptions DiscoveryCustomOptions `yaml:"custom_options,omitempty"`

	// Timing controls the "chattiness" of active scans.
	Timing DiscoveryTiming `yaml:"timing"`

	// AdditionalSubnets to scan in full_scan or custom mode.
	AdditionalSubnets []SubnetConfig `yaml:"additional_subnets"`

	// Legacy fields (kept for backward compatibility, will be deprecated)
	Enabled        bool          `yaml:"enabled"`          // Enable network discovery
	ARPScanWorkers int           `yaml:"arp_scan_workers"` // Number of concurrent workers
	PingTimeout    time.Duration `yaml:"ping_timeout"`     // Timeout for each ping
	ScanTimeout    time.Duration `yaml:"scan_timeout"`     // Total scan timeout
	AutoScan       bool          `yaml:"auto_scan"`        // Auto-scan on startup
	ScanInterval   time.Duration `yaml:"scan_interval"`    // Interval for auto-scan
	OUIFilePath    string        `yaml:"oui_file_path"`    // Path to IEEE OUI file
	OUIMaxAge      time.Duration `yaml:"oui_max_age"`      // Max age before auto-download (0 = never auto-update)

	// Fingerprinting enables OS/service detection.
	Fingerprinting FingerprintingConfig `yaml:"fingerprinting,omitempty"`

	// IPv6Enabled enables IPv6 Neighbor Discovery Protocol (NDP) scanning.
	IPv6Enabled bool `yaml:"ipv6_enabled"`
}

// DiscoveryCustomOptions provides fine-grained control when Profile is "custom".
type DiscoveryCustomOptions struct {
	PassiveListen bool            `yaml:"passive_listen"` // LLDP, CDP, EDP, DHCP listening
	ARPScan       bool            `yaml:"arp_scan"`       // ARP-based host discovery
	ICMPScan      bool            `yaml:"icmp_scan"`      // ICMP ping sweep
	PortScan      PortScanConfig  `yaml:"port_scan"`      // TCP/UDP port scanning
	Traceroute    bool            `yaml:"traceroute"`     // Path discovery
	SNMPQuery     bool            `yaml:"snmp_query"`     // SNMP device interrogation
}

// PortScanConfig controls port scanning behavior.
type PortScanConfig struct {
	Enabled  bool   `yaml:"enabled"`
	TCPPorts string `yaml:"tcp_ports"` // Comma-separated ports or ranges (e.g., "22,80,443,8000-8100")
	UDPPorts string `yaml:"udp_ports"` // Comma-separated ports or ranges
}

// DiscoveryTiming controls scan frequency and probe intervals.
type DiscoveryTiming struct {
	ProbeInterval  time.Duration `yaml:"probe_interval"`  // Time between sending probes (default 75ms)
	RescanInterval time.Duration `yaml:"rescan_interval"` // Time between full rescans (default 10m)
	Workers        int           `yaml:"workers"`         // Concurrent scan workers (default 50)
}

// FingerprintingConfig controls OS and service detection.
type FingerprintingConfig struct {
	Enabled       bool `yaml:"enabled"`        // Enable fingerprinting
	OSDetection   bool `yaml:"os_detection"`   // TCP stack analysis for OS detection
	ServiceProbes bool `yaml:"service_probes"` // Banner grabbing and service version detection
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

// IperfConfig contains iperf3 settings.
type IperfConfig struct {
	AutoRunOnLink bool   `yaml:"auto_run_on_link"` // Run automatically when link comes up
	Server        string `yaml:"server"`           // iperf3 server address
	Port          int    `yaml:"port"`             // iperf3 server port (default 5201)
	Protocol      string `yaml:"protocol"`         // "tcp" or "udp"
	Direction     string `yaml:"direction"`        // "upload", "download", or "bidirectional"
	Duration      int    `yaml:"duration"`         // Test duration in seconds
	ServerPort    int    `yaml:"server_port"`      // Port for local iperf server mode
	EnableServer  bool   `yaml:"enable_server"`    // Enable local iperf server mode
}

// FABOptionsConfig contains FAB (Floating Action Button) settings.
type FABOptionsConfig struct {
	RunLink             bool `yaml:"run_link"`
	RunSwitch           bool `yaml:"run_switch"`
	RunVLAN             bool `yaml:"run_vlan"`
	RunIPConfig         bool `yaml:"run_ip_config"`
	RunGateway          bool `yaml:"run_gateway"`
	RunDNS              bool `yaml:"run_dns"`
	RunHealthChecks     bool `yaml:"run_health_checks"`
	RunNetworkDiscovery bool `yaml:"run_network_discovery"`
	RunSpeedtest        bool `yaml:"run_speedtest"`
	RunIperf            bool `yaml:"run_iperf"`
	RunPerformance      bool `yaml:"run_performance"`
	AutoScanOnLink      bool `yaml:"auto_scan_on_link"`
}

// DisplayOptionsConfig contains display/UI settings.
type DisplayOptionsConfig struct {
	ShowPublicIP bool `yaml:"show_public_ip"`
}

// ThresholdsConfig contains all threshold settings.
type ThresholdsConfig struct {
	DHCP        DHCPThresholds   `yaml:"dhcp"`
	DNS         Threshold        `yaml:"dns"`
	Ping        Threshold        `yaml:"ping"`
	WiFi        WiFiThresholds   `yaml:"wifi"`
	Link        LinkThresholds   `yaml:"link"`
	CustomTests CustomThresholds `yaml:"custom_tests"`
}

// LinkThresholds contains thresholds for link stability.
type LinkThresholds struct {
	FlapCount24h IntThreshold `yaml:"flap_count_24h"` // Number of link flaps in 24h
}

// IntThreshold contains warning and critical thresholds for integer values.
type IntThreshold struct {
	Warning  int `yaml:"warning"`
	Critical int `yaml:"critical"`
}

// CustomThresholds contains thresholds for custom tests.
type CustomThresholds struct {
	Ping        Threshold            `yaml:"ping"`         // Custom ping targets
	TCP         Threshold            `yaml:"tcp"`          // TCP port tests
	UDP         Threshold            `yaml:"udp"`          // UDP port tests
	HTTP        Threshold            `yaml:"http"`         // HTTP endpoint tests (total time)
	HTTPTimings HTTPTimingThresholds `yaml:"http_timings"` // Per-phase HTTP timing thresholds
	CertExpiry  CertExpiryThreshold  `yaml:"cert_expiry"`  // Certificate expiry (days)
}

// HTTPTimingThresholds contains per-phase thresholds for HTTP requests.
type HTTPTimingThresholds struct {
	DNS  Threshold `yaml:"dns"`  // DNS resolution time
	TCP  Threshold `yaml:"tcp"`  // TCP connection time
	TLS  Threshold `yaml:"tls"`  // TLS handshake time
	TTFB Threshold `yaml:"ttfb"` // Time to first byte (server response)
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

	// VulnerabilityScanning configures CVE vulnerability scanning for discovered devices.
	VulnerabilityScanning VulnerabilityScanConfig `yaml:"vulnerability_scanning"`
}

// DHCPConfig contains DHCP monitoring and security settings.
type DHCPConfig struct {
	// RogueDetection configures rogue DHCP server detection.
	RogueDetection RogueDetectionConfig `yaml:"rogue_detection"`
}

// RogueDetectionConfig contains settings for rogue DHCP server detection.
type RogueDetectionConfig struct {
	Enabled         bool     `yaml:"enabled"`
	KnownServers    []string `yaml:"known_servers"`
	AlertOnDetection bool     `yaml:"alert_on_detection"`
}

// VulnerabilityScanConfig contains settings for CVE vulnerability scanning.
type VulnerabilityScanConfig struct {
	Enabled           bool   `yaml:"enabled"`
	CVEDatabase       string `yaml:"cve_database"`       // "nvd" or "local"
	NVDAPIKey         string `yaml:"nvd_api_key"`        // Optional NVD API key
	UpdateInterval    int    `yaml:"update_interval"`    // Seconds between updates
	SeverityThreshold string `yaml:"severity_threshold"` // "low", "medium", "high", "critical"
	MaxConcurrent     int    `yaml:"max_concurrent"`     // Max concurrent vulnerability checks
	AutoScan          bool   `yaml:"auto_scan"`          // Auto-scan after device discovery
}

// SNMPConfig contains SNMP settings for device interrogation.
type SNMPConfig struct {
	// Communities is a list of SNMP v1/v2c community strings to try (read-only).
	Communities []string `yaml:"communities"`

	// V3Credentials for SNMP v3 authentication.
	V3Credentials []SNMPv3Credential `yaml:"v3_credentials,omitempty"`

	// Timeout for SNMP queries.
	Timeout time.Duration `yaml:"timeout"`

	// Retries for failed SNMP queries.
	Retries int `yaml:"retries"`

	// Port for SNMP queries (default 161).
	Port int `yaml:"port"`
}

// SNMPv3Credential contains SNMP v3 authentication credentials.
type SNMPv3Credential struct {
	Name           string `yaml:"name"`            // Friendly name for this credential set
	Username       string `yaml:"username"`        // Security name (user)
	AuthProtocol   string `yaml:"auth_protocol"`   // "MD5", "SHA", "SHA256", "SHA512", or "" for noAuth
	AuthPassword   string `yaml:"auth_password"`   // Authentication password
	PrivProtocol   string `yaml:"priv_protocol"`   // "DES", "AES", "AES192", "AES256", or "" for noPriv
	PrivPassword   string `yaml:"priv_password"`   // Privacy password
	ContextName    string `yaml:"context_name"`    // Optional SNMP context
	SecurityLevel  string `yaml:"security_level"`  // "noAuthNoPriv", "authNoPriv", "authPriv"
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
			Default:          "eth0",
			Fallbacks:        []string{"enp0s3", "wlan0"},
			StartupRetries:   3,               // Retry 3 times when finding interface at startup (fixes #528)
			StartupRetryWait: 5 * time.Second, // Wait 5 seconds between retries (fixes #528)
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
			// New profile-based configuration (recommended)
			Profile: ProfileStandard, // Safe default: ARP/ICMP on local subnet only
			CustomOptions: DiscoveryCustomOptions{
				PassiveListen: true,  // Always listen for LLDP/CDP
				ARPScan:       true,  // ARP scan local subnet
				ICMPScan:      true,  // ICMP ping sweep
				PortScan:      PortScanConfig{Enabled: false},
				Traceroute:    false,
				SNMPQuery:     false, // Requires SNMP config
			},
			Timing: DiscoveryTiming{
				ProbeInterval:  75 * time.Millisecond, // Time between probes
				RescanInterval: 10 * time.Minute,      // Full rescan interval
				Workers:        50,                    // Concurrent workers
			},
			Fingerprinting: FingerprintingConfig{
				Enabled:       false, // Disabled by default
				OSDetection:   false,
				ServiceProbes: false,
			},
			IPv6Enabled: true, // Enable IPv6 NDP scanning by default
			// Legacy fields (for backward compatibility)
			Enabled:           true,
			ARPScanWorkers:    50,
			PingTimeout:       500 * time.Millisecond,
			ScanTimeout:       30 * time.Second,
			AutoScan:          true,             // Auto-scan on startup by default
			ScanInterval:      0,                // Disabled by default
			OUIFilePath:       "oui.txt",            // IEEE OUI file (download from https://standards-oui.ieee.org/oui/oui.txt)
			OUIMaxAge:         30 * 24 * time.Hour, // Auto-update OUI database monthly
			AdditionalSubnets: []SubnetConfig{},   // No additional subnets by default
		},
		SNMP: SNMPConfig{
			Communities: []string{"public"}, // Default read-only community
			Timeout:     5 * time.Second,
			Retries:     2,
			Port:        161,
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
			Link: LinkThresholds{
				FlapCount24h: IntThreshold{Warning: 3, Critical: 5}, // 3+ flaps = warning, 5+ = critical
			},
			CustomTests: CustomThresholds{
				Ping: Threshold{Warning: 50 * time.Millisecond, Critical: 100 * time.Millisecond},
				TCP:  Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
				UDP:  Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
				HTTP: Threshold{Warning: 500 * time.Millisecond, Critical: 2 * time.Second},
				HTTPTimings: HTTPTimingThresholds{
					DNS:  Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
					TCP:  Threshold{Warning: 100 * time.Millisecond, Critical: 500 * time.Millisecond},
					TLS:  Threshold{Warning: 150 * time.Millisecond, Critical: 500 * time.Millisecond},
					TTFB: Threshold{Warning: 500 * time.Millisecond, Critical: 2 * time.Second},
				},
				CertExpiry: CertExpiryThreshold{Warning: 30, Critical: 7}, // Days
			},
		},
		Auth: AuthConfig{
			DefaultUsername:     "admin",
			DefaultPasswordHash: "", // Empty = requires first-boot setup
			SessionTimeout:      24 * time.Hour,
			JWTSecret:           "", // Empty = will be generated and persisted
		},
		Security: SecurityConfig{
			AllowedOrigins: []string{}, // Empty = use RFC 1918 defaults
			VulnerabilityScanning: VulnerabilityScanConfig{
				Enabled:           false,
				CVEDatabase:       "nvd",
				NVDAPIKey:         "",
				UpdateInterval:    86400,  // 24 hours
				SeverityThreshold: "medium",
				MaxConcurrent:     5,
				AutoScan:          false,
			},
		},
		Iperf: IperfConfig{
			AutoRunOnLink: false,
			Server:        "",
			Port:          5201,
			Protocol:      "tcp",
			Direction:     "download",
			Duration:      10,
			ServerPort:    5201,
			EnableServer:  false,
		},
		FABOptions: FABOptionsConfig{
			RunLink:             true,
			RunSwitch:           true,
			RunVLAN:             true,
			RunIPConfig:         true,
			RunGateway:          true,
			RunDNS:              true,
			RunHealthChecks:     true,
			RunNetworkDiscovery: false,
			RunSpeedtest:        false,
			RunIperf:            false,
			RunPerformance:      false,
			AutoScanOnLink:      true,
		},
		DisplayOptions: DisplayOptionsConfig{
			ShowPublicIP: true,
		},
	}
}

// Load reads configuration from a YAML file.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	//nolint:gosec // G304: path is user-provided configuration file path, validated by caller
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
// This method acquires a read lock to prevent data races during marshaling.
// Validate checks if the configuration values are valid (fixes #542).
// This prevents the server from starting with invalid configuration.
func (c *Config) Validate() error {
	var errors []string

	// Server configuration
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errors = append(errors, fmt.Sprintf("server.port must be between 1-65535, got %d", c.Server.Port))
	}
	if c.Server.HTTPRedirectPort < 0 || c.Server.HTTPRedirectPort > 65535 {
		errors = append(errors, fmt.Sprintf("server.http_redirect_port must be between 0-65535, got %d", c.Server.HTTPRedirectPort))
	}
	if c.Server.HTTPRedirectPort > 0 && c.Server.Port == c.Server.HTTPRedirectPort {
		errors = append(errors, "server.port and server.http_redirect_port cannot be the same")
	}

	// Interface configuration
	if c.Interface.Default == "" {
		errors = append(errors, "interface.default is required")
	}
	if c.Interface.StartupRetries < 0 {
		errors = append(errors, fmt.Sprintf("interface.startup_retries must be >= 0, got %d", c.Interface.StartupRetries))
	}
	if c.Interface.StartupRetryWait < 0 {
		errors = append(errors, fmt.Sprintf("interface.startup_retry_wait must be >= 0, got %s", c.Interface.StartupRetryWait))
	}

	// VLAN configuration
	if c.VLAN.Enabled && (c.VLAN.ID < 1 || c.VLAN.ID > 4094) {
		errors = append(errors, fmt.Sprintf("vlan.id must be between 1-4094, got %d", c.VLAN.ID))
	}

	// IP configuration
	if c.IP.Mode != "dhcp" && c.IP.Mode != "static" {
		errors = append(errors, fmt.Sprintf("ip.mode must be 'dhcp' or 'static', got '%s'", c.IP.Mode))
	}
	if c.IP.Mode == "static" {
		if c.IP.Static.Address == "" {
			errors = append(errors, "ip.static.address is required when ip.mode is 'static'")
		}
		if c.IP.Static.Netmask == "" {
			errors = append(errors, "ip.static.netmask is required when ip.mode is 'static'")
		}
		if c.IP.Static.Gateway == "" {
			errors = append(errors, "ip.static.gateway is required when ip.mode is 'static'")
		}
	}

	// Timeout validations
	if c.Discovery.Timeout <= 0 {
		errors = append(errors, "discovery.timeout must be positive")
	}
	if c.NetworkDiscovery.PingTimeout <= 0 {
		errors = append(errors, "network_discovery.ping_timeout must be positive")
	}
	if c.NetworkDiscovery.ScanTimeout <= 0 {
		errors = append(errors, "network_discovery.scan_timeout must be positive")
	}
	if c.DNS.Timeout <= 0 {
		errors = append(errors, "dns.timeout must be positive")
	}

	// Worker/concurrency limits
	if c.NetworkDiscovery.ARPScanWorkers < 1 || c.NetworkDiscovery.ARPScanWorkers > 500 {
		errors = append(errors, fmt.Sprintf("network_discovery.arp_scan_workers must be between 1-500, got %d", c.NetworkDiscovery.ARPScanWorkers))
	}

	// Auth configuration
	if c.Auth.SessionTimeout <= 0 {
		errors = append(errors, "auth.session_timeout must be positive")
	}
	if c.Auth.DefaultUsername == "" {
		errors = append(errors, "auth.default_username is required")
	}
	if c.Auth.DefaultPasswordHash == "" {
		errors = append(errors, "auth.default_password_hash is required")
	}

	// SNMP configuration
	if c.SNMP.Port < 1 || c.SNMP.Port > 65535 {
		errors = append(errors, fmt.Sprintf("snmp.port must be between 1-65535, got %d", c.SNMP.Port))
	}
	if c.SNMP.Retries < 0 || c.SNMP.Retries > 10 {
		errors = append(errors, fmt.Sprintf("snmp.retries must be between 0-10, got %d", c.SNMP.Retries))
	}
	if c.SNMP.Timeout <= 0 {
		errors = append(errors, "snmp.timeout must be positive")
	}

	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errors, "\n  - "))
	}

	return nil
}

func (c *Config) Save(path string) error {
	c.mu.RLock()
	data, err := yaml.Marshal(c)
	c.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SetupResult holds information about first-boot credential setup.
type SetupResult struct {
	IsFirstBoot     bool
	GeneratedCreds  bool
	Username        string
	Password        string // Only set if credentials were generated (display once!)
	JWTSecretStored bool
}

// EnsureConfig handles first-boot setup and credential security.
// It checks for insecure default credentials and generates secure ones if needed.
// Returns SetupResult with credentials to display if they were generated.
//
// The function will:
// 1. Create config directory if it doesn't exist
// 2. Load existing config or create default
// 3. Check if using insecure default credentials (admin/netscope)
// 4. Generate and persist secure credentials if needed
// 5. Ensure JWT secret is persisted
func EnsureConfig(path string, checkDefaultPassword func(hash string) bool) (*Config, *SetupResult, error) {
	result := &SetupResult{}

	// Ensure config directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return nil, nil, fmt.Errorf("failed to create config directory: %w", err)
		}
	}

	// Check if config file exists
	_, err := os.Stat(path)
	isFirstBoot := os.IsNotExist(err)
	result.IsFirstBoot = isFirstBoot

	// Load or create config
	cfg, err := Load(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load config: %w", err)
	}

	needsSave := false

	// Check for insecure or missing credentials
	// Empty password hash = first boot, needs credential generation
	// Default password hash = insecure, needs credential generation
	if cfg.Auth.DefaultPasswordHash == "" ||
		(checkDefaultPassword != nil && checkDefaultPassword(cfg.Auth.DefaultPasswordHash)) {
		// Generate new secure credentials
		result.GeneratedCreds = true
		result.Username = cfg.Auth.DefaultUsername

		// Return error to signal caller needs to generate credentials
		return cfg, result, ErrInsecureCredentials
	}

	// Ensure JWT secret is set and persisted
	if cfg.Auth.JWTSecret == "" {
		needsSave = true
		result.JWTSecretStored = true
	}

	if needsSave && !isFirstBoot {
		if err := cfg.Save(path); err != nil {
			return nil, nil, fmt.Errorf("failed to save config: %w", err)
		}
	}

	return cfg, result, nil
}

// UpdateCredentials updates the authentication credentials in the config.
func (c *Config) UpdateCredentials(username, passwordHash, jwtSecret string) {
	c.Auth.DefaultUsername = username
	c.Auth.DefaultPasswordHash = passwordHash
	if jwtSecret != "" {
		c.Auth.JWTSecret = jwtSecret
	}
}

// UpdateJWTSecret updates only the JWT secret in the config.
func (c *Config) UpdateJWTSecret(secret string) {
	c.Auth.JWTSecret = secret
}

// GetActiveInterface returns an active network interface with an IPv4 address.
// It first tries the configured default, then fallbacks, then auto-detects.
// Returns the interface name and whether fallback was used.
func (c *Config) GetActiveInterface() (string, bool) {
	// Try the configured default interface first
	if c.Interface.Default != "" {
		if hasIPv4Address(c.Interface.Default) {
			return c.Interface.Default, false
		}
		log.Printf("Warning: configured interface %q has no IPv4 address or doesn't exist", c.Interface.Default)
	}

	// Try fallback interfaces
	for _, iface := range c.Interface.Fallbacks {
		if hasIPv4Address(iface) {
			log.Printf("Using fallback interface: %s", iface)
			return iface, true
		}
	}

	// Auto-detect: scan all interfaces for one with an IPv4 address
	detected := detectActiveInterface()
	if detected != "" {
		log.Printf("Auto-detected active interface: %s", detected)
		return detected, true
	}

	// Last resort: return the configured default even if it might not work
	if c.Interface.Default != "" {
		log.Printf("Warning: no active interface found, using configured default: %s", c.Interface.Default)
		return c.Interface.Default, true
	}

	return "eth0", true // Ultimate fallback
}

// hasIPv4Address checks if an interface exists and has at least one IPv4 address.
func hasIPv4Address(ifaceName string) bool {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false
	}

	// Check if interface is up
	if iface.Flags&net.FlagUp == 0 {
		return false
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return false
	}

	for _, addr := range addrs {
		// Check for IPv4 address (not loopback)
		if ipNet, ok := addr.(*net.IPNet); ok {
			if ipv4 := ipNet.IP.To4(); ipv4 != nil && !ipv4.IsLoopback() {
				return true
			}
		}
	}

	return false
}

// detectActiveInterface scans all network interfaces and returns the first
// non-loopback interface with an IPv4 address that is up.
func detectActiveInterface() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	// Priority order: prefer ethernet over wifi, physical over virtual
	// Pre-allocate slice with expected capacity
	candidates := make([]string, 0, len(interfaces))

	for _, iface := range interfaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		// Skip virtual/bridge interfaces (common prefixes)
		name := iface.Name
		if strings.HasPrefix(name, "docker") ||
			strings.HasPrefix(name, "br-") ||
			strings.HasPrefix(name, "veth") ||
			strings.HasPrefix(name, "virbr") ||
			strings.HasPrefix(name, "vbox") {
			continue
		}

		// Check if it has an IPv4 address
		if !hasIPv4Address(name) {
			continue
		}

		candidates = append(candidates, name)
	}

	if len(candidates) == 0 {
		return ""
	}

	// Sort candidates by preference (ethernet before wifi)
	// Common ethernet: eth*, enp*, eno*, ens*
	// Common wifi: wlan*, wlp*
	for _, c := range candidates {
		if strings.HasPrefix(c, "eth") || strings.HasPrefix(c, "en") {
			return c
		}
	}

	// Return first candidate if no ethernet found
	return candidates[0]
}
