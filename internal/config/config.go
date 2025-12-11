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
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
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
