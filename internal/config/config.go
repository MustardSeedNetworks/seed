// Package config handles application configuration.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// IP configuration mode constants.
const (
	ipModeDHCP   = "dhcp"
	ipModeStatic = "static"
)

// ErrInsecureCredentials is returned when default credentials are detected.
var ErrInsecureCredentials = errors.New("insecure default credentials detected")

// Config represents the application configuration.
// All API handlers must use Lock/Unlock for writes or RLock/RUnlock for reads
// to prevent concurrent config update race conditions.
type Config struct {
	mu               sync.RWMutex           `yaml:"-"`       // Protects concurrent access
	Version          int                    `yaml:"version"` // Schema version for migrations
	Server           ServerConfig           `yaml:"server"`
	Interface        InterfaceConfig        `yaml:"interface"`
	VLAN             VLANConfig             `yaml:"vlan"`
	IP               IPConfig               `yaml:"ip"`
	Discovery        DiscoveryConfig        `yaml:"discovery"`
	NetworkDiscovery NetworkDiscoveryConfig `yaml:"network_discovery"`
	DNS              DNSConfig              `yaml:"dns"`
	HealthChecks     HealthChecksConfig     `yaml:"health_checks"`
	Speedtest        SpeedtestConfig        `yaml:"speedtest"`
	Iperf            IperfConfig            `yaml:"iperf"`
	Thresholds       ThresholdsConfig       `yaml:"thresholds"`
	Auth             AuthConfig             `yaml:"auth"`
	Security         SecurityConfig         `yaml:"security"`
	DHCP             DHCPConfig             `yaml:"dhcp"`
	SNMP             SNMPConfig             `yaml:"snmp"`
	FABOptions       FABOptionsConfig       `yaml:"fab_options"`
	DisplayOptions   DisplayOptionsConfig   `yaml:"display_options"`
	Logging          LoggingConfig          `yaml:"logging"`
	MCP              MCPConfig              `yaml:"mcp"`
	Database         DatabaseConfig         `yaml:"database"`
}

// DatabaseConfig contains SQLite database configuration.
type DatabaseConfig struct {
	// Path to the SQLite database file. Default: data/seed.db
	Path string `yaml:"path"`
	// RetentionDays sets how many days of historical data to keep (0 = forever)
	RetentionDays int `yaml:"retention_days"`
	// EnableWAL enables Write-Ahead Logging for better concurrency. Default: true
	EnableWAL bool `yaml:"enable_wal"`
	// MaxConnections sets the maximum number of database connections. Default: 10
	MaxConnections int `yaml:"max_connections"`
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

// Clone creates a deep copy of the config.
// This is used to safely copy config values when holding a lock,
// preventing race conditions where new fields might be added but not copied.
// The mutex is NOT copied - the clone gets a fresh mutex.
func (c *Config) Clone() *Config {
	return c.cloneFields()
}

// cloneFields creates a Config with all fields copied from the receiver (fixes #691).
// This uses a struct literal, ensuring compile-time checking that no fields are missed.
// The mutex is NOT copied - returns a new Config with a fresh mutex.
func (c *Config) cloneFields() *Config {
	return &Config{
		Version:          c.Version,
		Server:           c.Server,
		Interface:        c.Interface,
		VLAN:             c.VLAN,
		IP:               c.IP,
		Discovery:        c.Discovery,
		NetworkDiscovery: c.NetworkDiscovery,
		DNS:              c.DNS,
		HealthChecks:     c.HealthChecks,
		Speedtest:        c.Speedtest,
		Iperf:            c.Iperf,
		Thresholds:       c.Thresholds,
		Auth:             c.Auth,
		Security:         c.Security,
		DHCP:             c.DHCP,
		SNMP:             c.SNMP,
		FABOptions:       c.FABOptions,
		DisplayOptions:   c.DisplayOptions,
		Logging:          c.Logging,
		MCP:              c.MCP,
		Database:         c.Database,
	}
}

// CopyFieldsFrom copies all fields from src to the receiver (fixes #691).
// This uses cloneFields internally, ensuring compile-time checking that no fields are missed.
// The mutex is NOT copied. The receiver must be locked before calling this method.
func (c *Config) CopyFieldsFrom(src *Config) {
	temp := src.cloneFields()
	// Copy each field individually to avoid copying the mutex
	c.Version = temp.Version
	c.Server = temp.Server
	c.Interface = temp.Interface
	c.VLAN = temp.VLAN
	c.IP = temp.IP
	c.Discovery = temp.Discovery
	c.NetworkDiscovery = temp.NetworkDiscovery
	c.DNS = temp.DNS
	c.HealthChecks = temp.HealthChecks
	c.Speedtest = temp.Speedtest
	c.Iperf = temp.Iperf
	c.Thresholds = temp.Thresholds
	c.Auth = temp.Auth
	c.Security = temp.Security
	c.DHCP = temp.DHCP
	c.SNMP = temp.SNMP
	c.FABOptions = temp.FABOptions
	c.DisplayOptions = temp.DisplayOptions
	c.Logging = temp.Logging
	c.MCP = temp.MCP
	c.Database = temp.Database
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port             int    `yaml:"port"`
	HTTPS            bool   `yaml:"https"`
	HTTPRedirectPort int    `yaml:"http_redirect_port,omitempty"` // Port for HTTP→HTTPS redirect (0 = disabled, typically 80)
	CertFile         string `yaml:"cert_file"`
	KeyFile          string `yaml:"key_file"`
	// Security fix #301: Removed LogAccessToken/LogAccessHeader - JWT authentication is sufficient

	// ACME/Let's Encrypt automatic certificate management
	ACME ACMEConfig `yaml:"acme,omitempty"`
}

// ACMEConfig contains ACME/Let's Encrypt certificate settings.
type ACMEConfig struct {
	Enabled  bool   `yaml:"enabled"`             // Enable automatic certificate management
	Domain   string `yaml:"domain"`              // Domain name for the certificate (e.g., "seed.example.com")
	Email    string `yaml:"email"`               // Contact email for Let's Encrypt notifications
	CacheDir string `yaml:"cache_dir,omitempty"` // Directory to cache certificates (default: "certs/acme")
	Staging  bool   `yaml:"staging,omitempty"`   // Use Let's Encrypt staging server (for testing)
}

// InterfaceConfig contains network interface settings.
type InterfaceConfig struct {
	Default          string        `yaml:"default"`
	Fallbacks        []string      `yaml:"fallbacks"`
	WiFi             string        `yaml:"wifi,omitempty"`     // Separate WiFi interface (optional)
	StartupRetries   int           `yaml:"startup_retries"`    // Number of retries when finding interface at startup (fixes #528)
	StartupRetryWait time.Duration `yaml:"startup_retry_wait"` // Delay between startup retries (fixes #528)
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
	PassiveListen bool           `yaml:"passive_listen"` // LLDP, CDP, EDP, DHCP listening
	ARPScan       bool           `yaml:"arp_scan"`       // ARP-based host discovery
	ICMPScan      bool           `yaml:"icmp_scan"`      // ICMP ping sweep
	PortScan      PortScanConfig `yaml:"port_scan"`      // TCP/UDP port scanning
	Traceroute    bool           `yaml:"traceroute"`     // Path discovery
	SNMPQuery     bool           `yaml:"snmp_query"`     // SNMP device interrogation
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

// HealthChecksConfig contains custom health check configurations.
// This section corresponds to the "Health Checks" card in the UI.
type HealthChecksConfig struct {
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
	ShowPublicIP bool   `yaml:"show_public_ip"`
	UnitSystem   string `yaml:"unit_system"` // "sae" (feet) or "metric" (meters)
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
	SSO                 SSOConfig     `yaml:"sso,omitempty"`
}

// SSOConfig contains settings for all SSO providers.
type SSOConfig struct {
	Providers []SSOProviderConfig `yaml:"providers"`
}

// SSOProviderConfig contains settings for a single SSO provider.
type SSOProviderConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Name         string   `yaml:"name"`
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"`
	Scopes       []string `yaml:"scopes,omitempty"`    // Custom OAuth scopes (uses defaults if empty)
	TenantID     string   `yaml:"tenant_id,omitempty"` // Microsoft only: "common", "organizations", "consumers", or specific tenant
}

// SecurityConfig contains security settings for CORS and WebSocket origins.
type SecurityConfig struct {
	// AllowedOrigins specifies explicit origins allowed for CORS and WebSocket.
	// If empty, defaults to RFC 1918 private network ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x).
	// Use "*" to allow all origins (not recommended for production).
	// Examples: ["http://192.168.1.100:8080", "https://seed.local"]
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
	Enabled          bool     `yaml:"enabled"`
	KnownServers     []string `yaml:"known_servers"`
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
	Name          string `yaml:"name"`           // Friendly name for this credential set
	Username      string `yaml:"username"`       // Security name (user)
	AuthProtocol  string `yaml:"auth_protocol"`  // "MD5", "SHA", "SHA256", "SHA512", or "" for noAuth
	AuthPassword  string `yaml:"auth_password"`  // Authentication password
	PrivProtocol  string `yaml:"priv_protocol"`  // "DES", "AES", "AES192", "AES256", or "" for noPriv
	PrivPassword  string `yaml:"priv_password"`  // Privacy password
	ContextName   string `yaml:"context_name"`   // Optional SNMP context
	SecurityLevel string `yaml:"security_level"` // "noAuthNoPriv", "authNoPriv", "authPriv"
}

// LoggingConfig contains structured logging settings.
type LoggingConfig struct {
	Level      string `yaml:"level"`       // DEBUG, INFO, WARN, ERROR (default: INFO)
	Format     string `yaml:"format"`      // text or json (default: text)
	AddSource  bool   `yaml:"add_source"`  // Include file:line in logs
	File       string `yaml:"file"`        // Log file path (empty = stdout only)
	MaxSize    int    `yaml:"max_size"`    // Max MB per log file before rotation
	MaxBackups int    `yaml:"max_backups"` // Number of old files to keep
	MaxAge     int    `yaml:"max_age"`     // Days to keep old files
	Compress   bool   `yaml:"compress"`    // Compress rotated files
}

// MCPConfig contains MCP (Model Context Protocol) server settings.
// MCP enables AI assistants like Claude to interact with the network diagnostics tools.
type MCPConfig struct {
	// Enabled enables the MCP server endpoint.
	Enabled bool `yaml:"enabled"`

	// RequireAuth requires JWT authentication for MCP connections.
	// When true, MCP requests must include a valid Bearer token.
	RequireAuth bool `yaml:"require_auth"`

	// RateLimitPerMinute limits requests per minute per client.
	// Set to 0 for unlimited (not recommended).
	RateLimitPerMinute int `yaml:"rate_limit_per_minute"`

	// AllowedTools lists specific tools to expose via MCP.
	// Empty list means all tools are available.
	AllowedTools []string `yaml:"allowed_tools,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		Server: ServerConfig{
			Port:  8443,
			HTTPS: true,
		},
		Interface: InterfaceConfig{
			Default:          "",              // Auto-detect by default (#572)
			Fallbacks:        []string{},      // No hardcoded fallbacks - use auto-detection
			StartupRetries:   3,               // Retry 3 times when finding interface at startup (fixes #528)
			StartupRetryWait: 5 * time.Second, // Wait 5 seconds between retries (fixes #528)
		},
		VLAN: VLANConfig{
			Enabled: false,
			ID:      0,
		},
		IP: IPConfig{
			Mode: ipModeDHCP,
		},
		Discovery: DiscoveryConfig{
			Protocol: "auto",
			Timeout:  30 * time.Second,
		},
		NetworkDiscovery: NetworkDiscoveryConfig{
			// New profile-based configuration (recommended)
			Profile: ProfileStandard, // Safe default: ARP/ICMP on local subnet only
			CustomOptions: DiscoveryCustomOptions{
				PassiveListen: true, // Always listen for LLDP/CDP
				ARPScan:       true, // ARP scan local subnet
				ICMPScan:      true, // ICMP ping sweep
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
			AutoScan:          true,                // Auto-scan on startup by default
			ScanInterval:      0,                   // Disabled by default
			OUIFilePath:       "oui.txt",           // IEEE OUI file (download from https://standards-oui.ieee.org/oui/oui.txt)
			OUIMaxAge:         30 * 24 * time.Hour, // Auto-update OUI database monthly
			AdditionalSubnets: []SubnetConfig{},    // No additional subnets by default
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
		HealthChecks: HealthChecksConfig{
			// Fixes #730: Add default health check tests so the card and timing graph have data on first boot
			PingTargets: []PingTarget{
				{Name: "Google DNS", Host: "8.8.8.8", Enabled: true},
				{Name: "Cloudflare", Host: "1.1.1.1", Enabled: true},
			},
			TCPPorts: []TCPPortTest{
				{Name: "HTTPS", Host: "www.google.com", Port: 443, Enabled: true},
				// Disabled example to keep prior DICOM test visible without causing failures
				{Name: "DICOM", Host: "example-dicom.local", Port: 104, Enabled: false},
				// FTP smoke test (disabled by default to avoid false positives on firewalled sites)
				{Name: "FTP", Host: "ftp.debian.org", Port: 21, Enabled: false},
				// SMB/Windows file sharing reachability
				{Name: "SMB", Host: "files.example.com", Port: 445, Enabled: false},
				// RTSP stream control port
				{Name: "RTSP", Host: "rtsp.example.com", Port: 554, Enabled: false},
				// Database (PostgreSQL) connectivity/latency
				{Name: "PostgreSQL", Host: "db.example.com", Port: 5432, Enabled: false},
				// SFTP/SSH file transfer
				{Name: "SFTP", Host: "sftp.example.com", Port: 22, Enabled: false},
			},
			UDPPorts: []UDPPortTest{
				{Name: "DNS", Host: "8.8.8.8", Port: 53, Enabled: true},
				{Name: "NTP", Host: "time.google.com", Port: 123, Enabled: true},
			},
			HTTPEndpoints: []HTTPEndpoint{
				{Name: "Google HTTPS", URL: "https://www.google.com", ExpectedStatus: 200, Enabled: true},
				{Name: "Cloudflare", URL: "https://www.cloudflare.com", ExpectedStatus: 200, Enabled: true},
				{Name: "Example HTTP", URL: "http://example.com", ExpectedStatus: 200, Enabled: true},
			},
			RunPerformance: true,
			RunSpeedtest:   true,
			RunIperf:       true,
			RunDiscovery:   true,
		},
		Speedtest: SpeedtestConfig{
			ServerID:      "",   // Auto-select closest
			AutoRunOnLink: true, // Fixes #728: Enable by default
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
			SSO: SSOConfig{
				Providers: []SSOProviderConfig{
					{
						Name:    "google",
						Enabled: false,
					},
					{
						Name:    "microsoft",
						Enabled: false,
					},
					{
						Name:    "github",
						Enabled: false,
					},
				},
			},
		},
		Security: SecurityConfig{
			AllowedOrigins: []string{}, // Empty = use RFC 1918 defaults
			VulnerabilityScanning: VulnerabilityScanConfig{
				Enabled:           false,
				CVEDatabase:       "nvd",
				NVDAPIKey:         "",
				UpdateInterval:    86400, // 24 hours
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
			UnitSystem:   "sae", // Default to SAE (feet) for US users
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			AddSource:  false,
			File:       "",
			MaxSize:    100,
			MaxBackups: 5,
			MaxAge:     30,
			Compress:   true,
		},
		MCP: MCPConfig{
			Enabled:            false, // Disabled by default, enable explicitly
			RequireAuth:        true,  // Require authentication by default for security
			RateLimitPerMinute: 60,    // 60 requests per minute default
			AllowedTools:       nil,   // All tools available when empty
		},
		Database: DatabaseConfig{
			Path:           "data/seed.db",
			RetentionDays:  90,   // 3 months of historical data
			EnableWAL:      true, // Better concurrency
			MaxConnections: 10,
		},
	}
}

// Load reads configuration from a YAML file.
// If the config has no version or an older version, it will be updated.
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

	// Handle unversioned configs (version 0 means unversioned)
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		slog.Info("Upgraded unversioned config to current version", "version", ConfigVersion)
	}

	return cfg, nil
}

// LoadWithMigration reads configuration from a YAML file and applies any necessary migrations.
// It creates a backup before applying migrations.
func LoadWithMigration(path string, migrator *MigrationManager) (*Config, bool, error) {
	cfg := DefaultConfig()

	//nolint:gosec // G304: path is user-provided configuration file path, validated by caller
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil // Use defaults if file doesn't exist
		}
		return nil, false, err
	}

	// Check current version in file
	var partial struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &partial); err != nil {
		return nil, false, fmt.Errorf("failed to parse config version: %w", err)
	}

	migrated := false
	if partial.Version < ConfigVersion && migrator != nil {
		// Create backup before migration
		backupMgr := NewBackupManager(path, "", 10)
		if _, backupErr := backupMgr.CreateBackup(); backupErr != nil {
			slog.Warn("Failed to create backup before migration", "error", backupErr)
		}

		// Apply migrations
		migratedData, err := migrator.Migrate(data, partial.Version, ConfigVersion)
		if err != nil {
			return nil, false, fmt.Errorf("failed to migrate config from v%d to v%d: %w",
				partial.Version, ConfigVersion, err)
		}
		data = migratedData
		migrated = true
		slog.Info("Migrated config", "from_version", partial.Version, "to_version", ConfigVersion)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, false, err
	}

	// Ensure version is set
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		migrated = true
	}

	// Save migrated config
	if migrated {
		if err := cfg.Save(path); err != nil {
			slog.Warn("Failed to save migrated config", "error", err)
		}
	}

	return cfg, migrated, nil
}

// Validate checks if the configuration values are valid.
// This prevents the server from starting with invalid configuration.
func (c *Config) Validate() error {
	var errs []string
	errs = append(errs, c.validateServerConfig()...)
	errs = append(errs, c.validateInterfaceConfig()...)
	errs = append(errs, c.validateVLANConfig()...)
	errs = append(errs, c.validateIPConfig()...)
	errs = append(errs, c.validateTimeouts()...)
	errs = append(errs, c.validateConcurrency()...)
	errs = append(errs, c.validateAuthConfig()...)
	errs = append(errs, c.validateSNMPConfig()...)
	errs = append(errs, c.validateLoggingConfig()...)

	if len(errs) > 0 {
		return fmt.Errorf("configuration validation failed:\n  - %s", strings.Join(errs, "\n  - "))
	}
	return nil
}

// validateServerConfig checks server port configuration.
func (c *Config) validateServerConfig() []string {
	var errs []string
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("server.port must be between 1-65535, got %d", c.Server.Port))
	}
	if c.Server.HTTPRedirectPort < 0 || c.Server.HTTPRedirectPort > 65535 {
		errs = append(errs, fmt.Sprintf("server.http_redirect_port must be between 0-65535, got %d", c.Server.HTTPRedirectPort))
	}
	if c.Server.HTTPRedirectPort > 0 && c.Server.Port == c.Server.HTTPRedirectPort {
		errs = append(errs, "server.port and server.http_redirect_port cannot be the same")
	}
	return errs
}

// validateInterfaceConfig checks interface startup configuration.
func (c *Config) validateInterfaceConfig() []string {
	var errs []string
	if c.Interface.StartupRetries < 0 {
		errs = append(errs, fmt.Sprintf("interface.startup_retries must be >= 0, got %d", c.Interface.StartupRetries))
	}
	if c.Interface.StartupRetryWait < 0 {
		errs = append(errs, fmt.Sprintf("interface.startup_retry_wait must be >= 0, got %s", c.Interface.StartupRetryWait))
	}
	return errs
}

// validateVLANConfig checks VLAN ID configuration.
func (c *Config) validateVLANConfig() []string {
	var errs []string
	if c.VLAN.Enabled && (c.VLAN.ID < 1 || c.VLAN.ID > 4094) {
		errs = append(errs, fmt.Sprintf("vlan.id must be between 1-4094, got %d", c.VLAN.ID))
	}
	return errs
}

// validateIPConfig checks IP mode and static IP configuration.
func (c *Config) validateIPConfig() []string {
	var errs []string
	if c.IP.Mode != ipModeDHCP && c.IP.Mode != ipModeStatic {
		errs = append(errs, fmt.Sprintf("ip.mode must be 'dhcp' or 'static', got '%s'", c.IP.Mode))
	}
	if c.IP.Mode == ipModeStatic {
		if c.IP.Static.Address == "" {
			errs = append(errs, "ip.static.address is required when ip.mode is 'static'")
		}
		if c.IP.Static.Netmask == "" {
			errs = append(errs, "ip.static.netmask is required when ip.mode is 'static'")
		}
		if c.IP.Static.Gateway == "" {
			errs = append(errs, "ip.static.gateway is required when ip.mode is 'static'")
		}
	}
	return errs
}

// validateTimeouts checks all timeout configurations are positive.
func (c *Config) validateTimeouts() []string {
	var errs []string
	if c.Discovery.Timeout <= 0 {
		errs = append(errs, "discovery.timeout must be positive")
	}
	if c.NetworkDiscovery.PingTimeout <= 0 {
		errs = append(errs, "network_discovery.ping_timeout must be positive")
	}
	if c.NetworkDiscovery.ScanTimeout <= 0 {
		errs = append(errs, "network_discovery.scan_timeout must be positive")
	}
	if c.DNS.Timeout <= 0 {
		errs = append(errs, "dns.timeout must be positive")
	}
	return errs
}

// validateConcurrency checks worker/concurrency limits.
func (c *Config) validateConcurrency() []string {
	var errs []string
	if c.NetworkDiscovery.ARPScanWorkers < 1 || c.NetworkDiscovery.ARPScanWorkers > 500 {
		errs = append(errs, fmt.Sprintf("network_discovery.arp_scan_workers must be between 1-500, got %d", c.NetworkDiscovery.ARPScanWorkers))
	}
	return errs
}

// validateAuthConfig checks authentication configuration.
func (c *Config) validateAuthConfig() []string {
	var errs []string
	if c.Auth.SessionTimeout <= 0 {
		errs = append(errs, "auth.session_timeout must be positive")
	}
	if c.Auth.DefaultUsername == "" {
		errs = append(errs, "auth.default_username is required")
	}
	if c.Auth.DefaultPasswordHash == "" {
		errs = append(errs, "auth.default_password_hash is required")
	}
	return errs
}

// validateSNMPConfig checks SNMP configuration.
func (c *Config) validateSNMPConfig() []string {
	var errs []string
	if c.SNMP.Port < 1 || c.SNMP.Port > 65535 {
		errs = append(errs, fmt.Sprintf("snmp.port must be between 1-65535, got %d", c.SNMP.Port))
	}
	if c.SNMP.Retries < 0 || c.SNMP.Retries > 10 {
		errs = append(errs, fmt.Sprintf("snmp.retries must be between 0-10, got %d", c.SNMP.Retries))
	}
	if c.SNMP.Timeout <= 0 {
		errs = append(errs, "snmp.timeout must be positive")
	}
	return errs
}

// validateLoggingConfig checks logging configuration.
func (c *Config) validateLoggingConfig() []string {
	var errs []string

	// Validate log level
	validLevels := map[string]bool{
		"debug": true, "info": true, "warn": true, "warning": true, "error": true,
	}
	level := strings.ToLower(c.Logging.Level)
	if level != "" && !validLevels[level] {
		errs = append(errs, fmt.Sprintf("logging.level must be one of debug, info, warn, error; got %q", c.Logging.Level))
	}

	// Validate format
	format := strings.ToLower(c.Logging.Format)
	if format != "" && format != "text" && format != "json" {
		errs = append(errs, fmt.Sprintf("logging.format must be 'text' or 'json'; got %q", c.Logging.Format))
	}

	// Validate rotation settings
	if c.Logging.MaxSize < 0 {
		errs = append(errs, fmt.Sprintf("logging.max_size must be >= 0, got %d", c.Logging.MaxSize))
	}
	if c.Logging.MaxBackups < 0 {
		errs = append(errs, fmt.Sprintf("logging.max_backups must be >= 0, got %d", c.Logging.MaxBackups))
	}
	if c.Logging.MaxAge < 0 {
		errs = append(errs, fmt.Sprintf("logging.max_age must be >= 0, got %d", c.Logging.MaxAge))
	}

	return errs
}

// Save writes the configuration to a YAML file at the specified path.
// This method acquires a read lock to prevent data races during marshaling.
func (c *Config) Save(path string) error {
	c.mu.RLock()
	data, err := yaml.Marshal(c)
	c.mu.RUnlock()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// SaveWithBackup writes the configuration to a YAML file, creating a backup first.
// This method acquires a read lock to prevent data races during marshaling.
// Returns the backup info if a backup was created, or nil if the file didn't exist.
func (c *Config) SaveWithBackup(path, backupDir string, maxBackups int) (*BackupInfo, error) {
	// Create backup if file exists
	var backup *BackupInfo
	if _, err := os.Stat(path); err == nil {
		backupMgr := NewBackupManager(path, backupDir, maxBackups)
		backup, err = backupMgr.CreateBackup()
		if err != nil {
			return nil, fmt.Errorf("failed to create backup: %w", err)
		}
	}

	// Save the config
	if err := c.Save(path); err != nil {
		return backup, err
	}

	return backup, nil
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
// 1. Create config directory if it doesn't exist.
// 2. Load existing config or create default.
// 3. Check if using insecure default credentials (admin/seed).
// 4. Generate and persist secure credentials if needed.
// 5. Ensure JWT secret is persisted.
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
		slog.Warn("Configured interface has no IPv4 address or doesn't exist", "interface", c.Interface.Default)
	}

	// Try fallback interfaces
	for _, iface := range c.Interface.Fallbacks {
		if hasIPv4Address(iface) {
			slog.Info("Using fallback interface", "interface", iface)
			return iface, true
		}
	}

	// Auto-detect: scan all interfaces for one with an IPv4 address
	detected := detectActiveInterface()
	if detected != "" {
		slog.Info("Auto-detected active interface", "interface", detected)
		return detected, true
	}

	// Last resort: return the configured default even if it might not work
	if c.Interface.Default != "" {
		slog.Warn("No active interface found, using configured default", "interface", c.Interface.Default)
		return c.Interface.Default, true
	}

	// No hardcoded fallback - return empty to signal no interface found (#572)
	slog.Error("No active network interface found")
	return "", false
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
