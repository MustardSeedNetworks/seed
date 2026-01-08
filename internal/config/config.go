package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/krisarmstrong/seed/internal/logging"
)

// IP configuration mode constants.
const (
	ipModeDHCP   = "dhcp"
	ipModeStatic = "static"
)

// Default configuration value constants.
const (
	// defaultHTTPSPort is the default port for HTTPS server (standard HTTPS alternate port).
	defaultHTTPSPort = 8443

	// defaultStartupRetries is the number of retries when finding interface at startup.
	defaultStartupRetries = 3

	// defaultStartupRetryWaitSec is the delay in seconds between startup retries.
	defaultStartupRetryWaitSec = 5

	// defaultDiscoveryTimeoutSec is the default timeout in seconds for switch discovery.
	defaultDiscoveryTimeoutSec = 30

	// defaultSNMPPort is the standard SNMP protocol port.
	defaultSNMPPort = 161

	// defaultSNMPTimeoutSec is the default timeout in seconds for SNMP queries.
	defaultSNMPTimeoutSec = 5

	// defaultSNMPRetries is the default number of retries for failed SNMP queries.
	defaultSNMPRetries = 2

	// defaultSNMPMaxRepetitions is the default number of OID values per GetBulk request.
	defaultSNMPMaxRepetitions = 10

	// defaultDNSTimeoutSec is the default timeout in seconds for DNS queries.
	defaultDNSTimeoutSec = 5

	// defaultIperfPort is the standard iperf3 server port.
	defaultIperfPort = 5201

	// defaultIperfDurationSec is the default iperf test duration in seconds.
	defaultIperfDurationSec = 10
)

// Logging default constants.
const (
	// defaultLogMaxSizeMB is the maximum size in megabytes before log rotation.
	defaultLogMaxSizeMB = 100

	// defaultLogMaxBackups is the number of old log files to keep.
	defaultLogMaxBackups = 5

	// defaultLogMaxAgeDays is the maximum number of days to retain old log files.
	defaultLogMaxAgeDays = 30
)

// API and rate limiting constants.
const (
	// defaultRateLimitPerMinute is the default API rate limit per client per minute.
	defaultRateLimitPerMinute = 60
)

// Database default constants.
const (
	// defaultDBRetentionDays is the default number of days to retain historical data.
	defaultDBRetentionDays = 90

	// defaultDBMaxConnections is the default maximum number of database connections.
	defaultDBMaxConnections = 10

	// defaultBackupMaxCount is the default maximum number of config backups to retain.
	defaultBackupMaxCount = 10
)

// Network port constants for common services.
const (
	// portHTTPS is the standard HTTPS port.
	portHTTPS = 443

	// portDICOM is the standard DICOM medical imaging protocol port.
	portDICOM = 104

	// portFTP is the standard FTP control port.
	portFTP = 21

	// portSMB is the standard SMB/CIFS file sharing port.
	portSMB = 445

	// portRTSP is the standard RTSP streaming protocol port.
	portRTSP = 554

	// portPostgreSQL is the standard PostgreSQL database port.
	portPostgreSQL = 5432

	// portSSH is the standard SSH/SFTP port.
	portSSH = 22

	// portDNS is the standard DNS port.
	portDNS = 53

	// portNTP is the standard NTP time synchronization port.
	portNTP = 123

	// portHTTP is the standard HTTP port for default health check endpoints.
	portHTTP = 80

	// portHTTPAlt is the alternate HTTP port commonly used for web servers.
	portHTTPAlt = 8080
)

// HTTP response status codes.
const (
	// httpStatusOK is the HTTP 200 OK status code.
	httpStatusOK = 200
)

// Timeout and timing constants.
const (
	// defaultBannerTimeoutSec is the timeout in seconds for service banner grabbing.
	defaultBannerTimeoutSec = 2

	// defaultTracerouteTimeoutSec is the timeout in seconds for traceroute TCP probes.
	defaultTracerouteTimeoutSec = 2

	// defaultTracerouteWorkers is the number of concurrent traceroute workers.
	defaultTracerouteWorkers = 20

	// defaultMDNSTimeoutSec is the timeout in seconds for mDNS/device profiling operations.
	defaultMDNSTimeoutSec = 2

	// defaultMDNSMaxConcurrent is the maximum concurrent mDNS/profiler operations.
	defaultMDNSMaxConcurrent = 5

	// defaultProbeIntervalMs is the interval in milliseconds between network probes.
	defaultProbeIntervalMs = 75

	// defaultRescanIntervalMin is the interval in minutes between full network rescans.
	defaultRescanIntervalMin = 10

	// defaultARPWorkers is the number of concurrent ARP scan workers.
	defaultARPWorkers = 50

	// defaultPingTimeoutMs is the timeout in milliseconds for ICMP ping operations.
	defaultPingTimeoutMs = 500

	// defaultScanTimeoutSec is the total timeout in seconds for network scans.
	defaultScanTimeoutSec = 30

	// defaultOUIMaxAgeDays is the maximum age in days for OUI database before refresh.
	defaultOUIMaxAgeDays = 30

	// defaultSessionTimeoutHours is the default session timeout in hours.
	defaultSessionTimeoutHours = 24

	// defaultVulnUpdateIntervalSec is the interval in seconds for vulnerability database updates (24 hours).
	defaultVulnUpdateIntervalSec = 86400
)

// Pipeline timing constants.
const (
	// defaultPipelineProbeDelayMs is the delay in milliseconds between pipeline probes.
	defaultPipelineProbeDelayMs = 50

	// defaultPipelineHostDelayMs is the delay in milliseconds between scanning hosts.
	defaultPipelineHostDelayMs = 20

	// defaultPipelineMaxConcurrentHosts is the maximum concurrent hosts in pipeline scanning.
	defaultPipelineMaxConcurrentHosts = 20

	// defaultPipelinePhaseTimeoutMin is the timeout in minutes for each pipeline phase.
	defaultPipelinePhaseTimeoutMin = 10

	// defaultSNMPWalkTimeoutSec is the timeout in seconds for SNMP walk operations.
	defaultSNMPWalkTimeoutSec = 30

	// defaultSNMPMaxOIDsPerRequest is the maximum OIDs per SNMP bulk request.
	defaultSNMPMaxOIDsPerRequest = 10

	// defaultStalenessThresholdHours is hours before a device is considered stale.
	defaultStalenessThresholdHours = 24

	// defaultPurgeAfterDays is days before inactive devices are purged.
	defaultPurgeAfterDays = 30
)

// Threshold timing constants (in milliseconds).
const (
	// thresholdDHCPTotalWarningMs is the warning threshold for total DHCP time.
	thresholdDHCPTotalWarningMs = 500

	// thresholdDHCPPhaseWarningMs is the warning threshold for DHCP phase time.
	thresholdDHCPPhaseWarningMs = 200

	// thresholdDNSWarningMs is the warning threshold for DNS resolution.
	thresholdDNSWarningMs = 100

	// thresholdPingWarningMs is the warning threshold for ping latency.
	thresholdPingWarningMs = 50

	// thresholdPingCriticalMs is the critical threshold for ping latency.
	thresholdPingCriticalMs = 200

	// thresholdCustomPingCriticalMs is the critical threshold for custom ping tests.
	thresholdCustomPingCriticalMs = 100

	// thresholdTCPWarningMs is the warning threshold for TCP connections.
	thresholdTCPWarningMs = 100

	// thresholdHTTPWarningMs is the warning threshold for HTTP requests.
	thresholdHTTPWarningMs = 500

	// thresholdTLSWarningMs is the warning threshold for TLS handshakes.
	thresholdTLSWarningMs = 150
)

// Threshold integer constants.
const (
	// thresholdWiFiSignalWarningDBm is the warning threshold for WiFi signal strength in dBm.
	thresholdWiFiSignalWarningDBm = -70

	// thresholdWiFiSignalCriticalDBm is the critical threshold for WiFi signal strength in dBm.
	thresholdWiFiSignalCriticalDBm = -80

	// thresholdLinkFlapWarning is the warning threshold for link flaps in 24 hours.
	thresholdLinkFlapWarning = 3

	// thresholdLinkFlapCritical is the critical threshold for link flaps in 24 hours.
	thresholdLinkFlapCritical = 5

	// thresholdCertExpiryWarningDays is days until certificate expiry warning.
	thresholdCertExpiryWarningDays = 30

	// thresholdCertExpiryCriticalDays is days until certificate expiry critical alert.
	thresholdCertExpiryCriticalDays = 7
)

// ErrInsecureCredentials is returned when default credentials are detected.
var ErrInsecureCredentials = errors.New("insecure default credentials detected")

// Config represents the application configuration.
// All API handlers must use Lock/Unlock for writes or RLock/RUnlock for reads
// to prevent concurrent config update race conditions.
type Config struct {
	mu               sync.RWMutex           `yaml:"-"                 json:"-"`       // Protects access
	Version          int                    `yaml:"version"           json:"version"` // Schema version
	Server           ServerConfig           `yaml:"server"            json:"server"`
	Interface        InterfaceConfig        `yaml:"interface"         json:"interface"`
	VLAN             VLANConfig             `yaml:"vlan"              json:"vlan"`
	IP               IPConfig               `yaml:"ip"                json:"ip"`
	Discovery        DiscoveryConfig        `yaml:"discovery"         json:"discovery"`
	NetworkDiscovery NetworkDiscoveryConfig `yaml:"network_discovery" json:"networkDiscovery"`
	DNS              DNSConfig              `yaml:"dns"               json:"dns"`
	HealthChecks     HealthChecksConfig     `yaml:"health_checks"     json:"healthChecks"`
	Speedtest        SpeedtestConfig        `yaml:"speedtest"         json:"speedtest"`
	Iperf            IperfConfig            `yaml:"iperf"             json:"iperf"`
	Thresholds       ThresholdsConfig       `yaml:"thresholds"        json:"thresholds"`
	Auth             AuthConfig             `yaml:"auth"              json:"auth"`
	Security         SecurityConfig         `yaml:"security"          json:"security"`
	DHCP             DHCPConfig             `yaml:"dhcp"              json:"dhcp"`
	SNMP             SNMPConfig             `yaml:"snmp"              json:"snmp"`
	FABOptions       FABOptionsConfig       `yaml:"fab_options"       json:"fabOptions"`
	DisplayOptions   DisplayOptionsConfig   `yaml:"display_options"   json:"displayOptions"`
	Logging          LoggingConfig          `yaml:"logging"           json:"logging"`
	MCP              MCPConfig              `yaml:"mcp"               json:"mcp"`
	Database         DatabaseConfig         `yaml:"database"          json:"database"`
	Pipeline         PipelineConfig         `yaml:"pipeline"          json:"pipeline"`
}

// PipelineConfig controls the sequential discovery pipeline.
type PipelineConfig struct {
	// Phases controls which pipeline phases are enabled.
	Phases PipelinePhaseConfig `yaml:"phases" json:"phases"`

	// Timing controls rate limiting and delays.
	Timing PipelineTimingConfig `yaml:"timing" json:"timing"`

	// PortScan controls port scanning behavior and intensity.
	PortScan PipelinePortScanConfig `yaml:"port_scan" json:"port_scan"`

	// SNMPCollection controls extended SNMP MIB collection.
	SNMPCollection PipelineSNMPConfig `yaml:"snmp_collection" json:"snmp_collection"`

	// Persistence controls how results are stored.
	Persistence PipelinePersistenceConfig `yaml:"persistence" json:"persistence"`
}

// PipelinePhaseConfig controls which phases are executed.
type PipelinePhaseConfig struct {
	Enumeration      bool `yaml:"enumeration"       json:"enumeration"`       // Always true - core functionality
	NameResolution   bool `yaml:"name_resolution"   json:"name_resolution"`   // Default: true
	ServiceDiscovery bool `yaml:"service_discovery" json:"service_discovery"` // Default: false (passive only)
	VulnAssessment   bool `yaml:"vuln_assessment"   json:"vuln_assessment"`   // Default: false
}

// PipelineTimingConfig controls scan rate limiting.
type PipelineTimingConfig struct {
	// ProbeDelay is the minimum time between probes to a single host.
	ProbeDelay time.Duration `yaml:"probe_delay" json:"probe_delay"`

	// HostDelay is the minimum time between starting scans of different hosts.
	HostDelay time.Duration `yaml:"host_delay" json:"host_delay"`

	// MaxConcurrentHosts limits parallel host scanning.
	MaxConcurrentHosts int `yaml:"max_concurrent_hosts" json:"max_concurrent_hosts"`

	// PhaseTimeout is the max duration for any single phase.
	PhaseTimeout time.Duration `yaml:"phase_timeout" json:"phase_timeout"`

	// Profile selects a pre-defined timing profile: polite, normal, aggressive.
	Profile string `yaml:"profile" json:"profile"`
}

// PipelinePortScanConfig controls port scanning intensity.
type PipelinePortScanConfig struct {
	// Intensity controls which ports are scanned: off, quick, standard, comprehensive, custom.
	Intensity string `yaml:"intensity" json:"intensity"`

	// CustomPorts for Intensity="custom".
	CustomPorts []int `yaml:"custom_ports,omitempty" json:"custom_ports,omitempty"`

	// BannerGrab enables service banner reading.
	BannerGrab bool `yaml:"banner_grab" json:"banner_grab"`

	// ConnectTimeout for port connections.
	ConnectTimeout time.Duration `yaml:"connect_timeout" json:"connect_timeout"`
}

// PipelineSNMPConfig controls extended SNMP data collection.
type PipelineSNMPConfig struct {
	// Enabled turns on extended SNMP collection in Phase 3.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// MIBs specifies which MIB groups to collect.
	MIBs PipelineSNMPMIBs `yaml:"mibs" json:"mibs"`

	// WalkTimeout per MIB walk operation.
	WalkTimeout time.Duration `yaml:"walk_timeout" json:"walk_timeout"`

	// MaxOIDsPerRequest for bulk requests.
	MaxOIDsPerRequest int `yaml:"max_oids_per_request" json:"max_oids_per_request"`
}

// PipelineSNMPMIBs controls which MIBs are collected.
type PipelineSNMPMIBs struct {
	System      bool `yaml:"system"       json:"system"`       // SNMPv2-MIB::system (always on)
	Interfaces  bool `yaml:"interfaces"   json:"interfaces"`   // IF-MIB (ifTable, ifXTable)
	IPAddresses bool `yaml:"ip_addresses" json:"ip_addresses"` // IP-MIB (ipAddrTable)
	Routing     bool `yaml:"routing"      json:"routing"`      // IP-FORWARD-MIB
	Bridge      bool `yaml:"bridge"       json:"bridge"`       // BRIDGE-MIB (MAC table)
	Entity      bool `yaml:"entity"       json:"entity"`       // ENTITY-MIB (physical inventory)
	LLDP        bool `yaml:"lldp"         json:"lldp"`         // LLDP-MIB
	VLAN        bool `yaml:"vlan"         json:"vlan"`         // Q-BRIDGE-MIB
}

// PipelinePersistenceConfig controls database storage.
type PipelinePersistenceConfig struct {
	// StoreHistory keeps historical device state.
	StoreHistory bool `yaml:"store_history" json:"store_history"`

	// StalenessThreshold marks devices inactive after this duration.
	StalenessThreshold time.Duration `yaml:"staleness_threshold" json:"staleness_threshold"`

	// PurgeAfter removes inactive devices after this duration.
	PurgeAfter time.Duration `yaml:"purge_after" json:"purge_after"`
}

// GetPhases implements discovery.ConfigPipelineAdapter.
func (c *PipelineConfig) GetPhases() (bool, bool, bool, bool) {
	return c.Phases.Enumeration, c.Phases.NameResolution, c.Phases.ServiceDiscovery, c.Phases.VulnAssessment
}

// GetTiming implements discovery.ConfigPipelineAdapter.
func (c *PipelineConfig) GetTiming() (time.Duration, time.Duration, time.Duration, int, string) {
	return c.Timing.ProbeDelay, c.Timing.HostDelay, c.Timing.PhaseTimeout, c.Timing.MaxConcurrentHosts, c.Timing.Profile
}

// GetPortScan implements discovery.ConfigPipelineAdapter.
func (c *PipelineConfig) GetPortScan() (string, []int, bool, time.Duration) {
	// Fixes #959: Deep copy CustomPorts to prevent caller mutation
	var portsCopy []int
	if len(c.PortScan.CustomPorts) > 0 {
		portsCopy = make([]int, len(c.PortScan.CustomPorts))
		copy(portsCopy, c.PortScan.CustomPorts)
	}
	return c.PortScan.Intensity, portsCopy, c.PortScan.BannerGrab, c.PortScan.ConnectTimeout
}

// GetSNMP implements discovery.ConfigPipelineAdapter.
//

func (c *PipelineConfig) GetSNMP() (bool, bool, bool, bool, bool, bool, bool, bool, bool, time.Duration, int) {
	return c.SNMPCollection.Enabled,
		c.SNMPCollection.MIBs.System,
		c.SNMPCollection.MIBs.Interfaces,
		c.SNMPCollection.MIBs.IPAddresses,
		c.SNMPCollection.MIBs.Routing,
		c.SNMPCollection.MIBs.Bridge,
		c.SNMPCollection.MIBs.Entity,
		c.SNMPCollection.MIBs.LLDP,
		c.SNMPCollection.MIBs.VLAN,
		c.SNMPCollection.WalkTimeout,
		c.SNMPCollection.MaxOIDsPerRequest
}

// GetPersistence implements discovery.ConfigPipelineAdapter.
func (c *PipelineConfig) GetPersistence() (bool, time.Duration, time.Duration) {
	return c.Persistence.StoreHistory, c.Persistence.StalenessThreshold, c.Persistence.PurgeAfter
}

// DatabaseConfig contains SQLite database configuration.
type DatabaseConfig struct {
	// Path to the SQLite database file. Default: data/seed.db
	Path string `yaml:"path"            json:"path"`
	// RetentionDays sets how many days of historical data to keep (0 = forever)
	RetentionDays int `yaml:"retention_days"  json:"retention_days"`
	// EnableWAL enables Write-Ahead Logging for better concurrency. Default: true
	EnableWAL bool `yaml:"enable_wal"      json:"enable_wal"`
	// MaxConnections sets the maximum number of database connections. Default: 10
	MaxConnections int `yaml:"max_connections" json:"max_connections"`
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
// Fixes #958: Deep copy slices to prevent shared references.
func (c *Config) cloneFields() *Config {
	clone := &Config{
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
		Pipeline:         c.Pipeline,
	}

	// Deep copy slices to prevent shared references (fixes #958)
	if len(c.Security.AllowedOrigins) > 0 {
		clone.Security.AllowedOrigins = make([]string, len(c.Security.AllowedOrigins))
		copy(clone.Security.AllowedOrigins, c.Security.AllowedOrigins)
	}
	if len(c.SNMP.Communities) > 0 {
		clone.SNMP.Communities = make([]string, len(c.SNMP.Communities))
		copy(clone.SNMP.Communities, c.SNMP.Communities)
	}
	if len(c.SNMP.V3Credentials) > 0 {
		clone.SNMP.V3Credentials = make([]SNMPv3Credential, len(c.SNMP.V3Credentials))
		copy(clone.SNMP.V3Credentials, c.SNMP.V3Credentials)
	}

	return clone
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
	c.Pipeline = temp.Pipeline
}

// ServerConfig contains HTTP server settings.
type ServerConfig struct {
	Port             int    `yaml:"port"                         json:"port"`
	HTTPS            bool   `yaml:"https"                        json:"https"`
	HTTPRedirectPort int    `yaml:"http_redirect_port,omitempty" json:"http_redirect_port,omitempty"` // Port for HTTP→HTTPS redirect (0 = disabled, typically 80)
	CertFile         string `yaml:"cert_file"                    json:"cert_file"`
	KeyFile          string `yaml:"key_file"                     json:"key_file"`
	// Security fix #301: Removed LogAccessToken/LogAccessHeader - JWT authentication is sufficient

	// ACME/Let's Encrypt automatic certificate management
	ACME ACMEConfig `yaml:"acme,omitempty" json:"acme,omitzero"`
}

// ACMEConfig contains ACME/Let's Encrypt certificate settings.
type ACMEConfig struct {
	Enabled  bool   `yaml:"enabled"             json:"enabled"`             // Enable automatic certificate management
	Domain   string `yaml:"domain"              json:"domain"`              // Domain name for the certificate (e.g., "seed.example.com")
	Email    string `yaml:"email"               json:"email"`               // Contact email for Let's Encrypt notifications
	CacheDir string `yaml:"cache_dir,omitempty" json:"cache_dir,omitempty"` // Directory to cache certificates (default: "certs/acme")
	Staging  bool   `yaml:"staging,omitempty"   json:"staging,omitempty"`   // Use Let's Encrypt staging server (for testing)
}

// InterfaceConfig contains network interface settings.
type InterfaceConfig struct {
	Default          string        `yaml:"default"            json:"default"`
	Fallbacks        []string      `yaml:"fallbacks"          json:"fallbacks"`
	WiFi             string        `yaml:"wifi,omitempty"     json:"wifi,omitempty"`     // Separate WiFi interface (optional)
	StartupRetries   int           `yaml:"startup_retries"    json:"startup_retries"`    // Number of retries when finding interface at startup (fixes #528)
	StartupRetryWait time.Duration `yaml:"startup_retry_wait" json:"startup_retry_wait"` // Delay between startup retries (fixes #528)
}

// VLANConfig contains VLAN settings.
type VLANConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	ID      int  `yaml:"id"      json:"id"`
}

// IPConfig contains IP configuration settings.
type IPConfig struct {
	Mode   string    `yaml:"mode"             json:"mode"` // "dhcp" or "static"
	Static *StaticIP `yaml:"static,omitempty" json:"static,omitempty"`
}

// StaticIP contains static IP configuration.
type StaticIP struct {
	Address string   `yaml:"address" json:"address"`
	Netmask string   `yaml:"netmask" json:"netmask"`
	Gateway string   `yaml:"gateway" json:"gateway"`
	DNS     []string `yaml:"dns"     json:"dns"`
}

// DiscoveryConfig contains switch discovery settings.
type DiscoveryConfig struct {
	Protocol string        `yaml:"protocol" json:"protocol"` // "auto", "lldp", "cdp", "edp", "fdp"
	Timeout  time.Duration `yaml:"timeout"  json:"timeout"`
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
	Options DiscoveryOptions `yaml:"options" json:"options"`

	// Timing controls the "chattiness" of active scans.
	Timing DiscoveryTiming `yaml:"timing" json:"timing"`

	// AdditionalSubnets to scan in full_scan or custom mode.
	AdditionalSubnets []SubnetConfig `yaml:"additional_subnets" json:"additional_subnets"`

	// Legacy fields (kept for backward compatibility, will be deprecated)
	Enabled        bool          `yaml:"enabled"          json:"enabled"`          // Enable network discovery
	ARPScanWorkers int           `yaml:"arp_scan_workers" json:"arp_scan_workers"` // Number of concurrent workers
	PingTimeout    time.Duration `yaml:"ping_timeout"     json:"ping_timeout"`     // Timeout for each ping
	ScanTimeout    time.Duration `yaml:"scan_timeout"     json:"scan_timeout"`     // Total scan timeout
	AutoScan       bool          `yaml:"auto_scan"        json:"auto_scan"`        // Auto-scan on startup
	ScanInterval   time.Duration `yaml:"scan_interval"    json:"scan_interval"`    // Interval for auto-scan
	OUIFilePath    string        `yaml:"oui_file_path"    json:"oui_file_path"`    // Path to IEEE OUI file
	OUIMaxAge      time.Duration `yaml:"oui_max_age"      json:"oui_max_age"`      // Max age before auto-download (0 = never auto-update)

	// Fingerprinting enables OS/service detection.
	Fingerprinting FingerprintingConfig `yaml:"fingerprinting,omitempty" json:"fingerprinting,omitzero"`

	// Profiler controls automatic device profiling.
	Profiler DeviceProfilerConfig `yaml:"profiler,omitempty" json:"profiler,omitzero"`

	// IPv6Enabled enables IPv6 Neighbor Discovery Protocol (NDP) scanning.
	IPv6Enabled bool `yaml:"ipv6_enabled" json:"ipv6_enabled"`
}

// DiscoveryOptions provides control over all discovery methods.
type DiscoveryOptions struct {
	PassiveProtocols PassiveProtocolConfig `yaml:"passive_protocols" json:"passiveProtocols"` // Granular passive protocol control
	ARPScan          bool                  `yaml:"arp_scan"          json:"arpScan"`          // ARP-based host discovery
	ICMPScan         bool                  `yaml:"icmp_scan"         json:"icmpScan"`         // ICMP ping sweep
	PortScan         PortScanConfig        `yaml:"port_scan"         json:"portScan"`         // TCP/UDP port scanning
	TCPProbe         TCPProbeConfig        `yaml:"tcp_probe"         json:"tcpProbe"`         // TCP probe settings
	Traceroute       bool                  `yaml:"traceroute"        json:"traceroute"`       // Path discovery
	SNMPQuery        bool                  `yaml:"snmp_query"        json:"snmpQuery"`        // SNMP device interrogation
}

// PortScanConfig controls port scanning behavior.
type PortScanConfig struct {
	Enabled       bool          `yaml:"enabled"        json:"enabled"`
	Preset        PortPreset    `yaml:"preset"         json:"preset"`        // Port preset: common, secure, insecure, custom
	TCPPorts      string        `yaml:"tcp_ports"      json:"tcpPorts"`      // Comma-separated ports or ranges (used when preset is "custom")
	UDPPorts      string        `yaml:"udp_ports"      json:"udpPorts"`      // Comma-separated ports or ranges (used when preset is "custom")
	BannerTimeout time.Duration `yaml:"banner_timeout" json:"bannerTimeout"` // Timeout for banner grabbing (default 2s)
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
	LLDP bool `yaml:"lldp" json:"lldp"` // IEEE 802.1AB Link Layer Discovery Protocol
	CDP  bool `yaml:"cdp"  json:"cdp"`  // Cisco Discovery Protocol
	EDP  bool `yaml:"edp"  json:"edp"`  // Extreme Discovery Protocol
	NDP  bool `yaml:"ndp"  json:"ndp"`  // IPv6 Neighbor Discovery Protocol
}

// TCPProbeConfig controls TCP connection probing behavior.
type TCPProbeConfig struct {
	Timeout time.Duration `yaml:"timeout" json:"timeout"` // Connection timeout (default 2s)
	Workers int           `yaml:"workers" json:"workers"` // Concurrent probe workers (default 20)
}

// DeviceProfilerConfig controls automatic device profiling.
type DeviceProfilerConfig struct {
	Enabled       bool          `yaml:"enabled"        json:"enabled"`        // Enable automatic profiling
	Timeout       time.Duration `yaml:"timeout"        json:"timeout"`        // Profile operation timeout (default 2s)
	MaxConcurrent int           `yaml:"max_concurrent" json:"max_concurrent"` // Max concurrent profile operations (default 5)
	QuickPorts    []int         `yaml:"quick_ports"    json:"quick_ports"`    // Quick scan ports for profiling (default: 22,80,443,8080)
}

// DiscoveryTiming controls scan frequency and probe intervals.
type DiscoveryTiming struct {
	ProbeInterval  time.Duration `yaml:"probe_interval"  json:"probe_interval"`  // Time between sending probes (default 75ms)
	RescanInterval time.Duration `yaml:"rescan_interval" json:"rescan_interval"` // Time between full rescans (default 10m)
	Workers        int           `yaml:"workers"         json:"workers"`         // Concurrent scan workers (default 50)
}

// FingerprintingConfig controls OS and service detection.
type FingerprintingConfig struct {
	Enabled       bool `yaml:"enabled"        json:"enabled"`        // Enable fingerprinting
	OSDetection   bool `yaml:"os_detection"   json:"os_detection"`   // TCP stack analysis for OS detection
	ServiceProbes bool `yaml:"service_probes" json:"service_probes"` // Banner grabbing and service version detection
}

// SubnetConfig represents a configured subnet for network discovery.
type SubnetConfig struct {
	CIDR    string `yaml:"cidr"    json:"cidr"`    // CIDR notation (e.g., "10.0.0.0/24")
	Name    string `yaml:"name"    json:"name"`    // Friendly name (e.g., "Server VLAN")
	Enabled bool   `yaml:"enabled" json:"enabled"` // Whether to scan this subnet
}

// DNSConfig contains DNS testing settings.
type DNSConfig struct {
	TestHostname string        `yaml:"test_hostname"     json:"test_hostname"`
	Timeout      time.Duration `yaml:"timeout"           json:"timeout"`
	Servers      []DNSServer   `yaml:"servers,omitempty" json:"servers,omitempty"` // Additional DNS servers to test
}

// DNSServer represents a DNS server configuration.
type DNSServer struct {
	Address string `yaml:"address" json:"address"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// HealthChecksConfig contains custom health check configurations.
// This section corresponds to the "Health Checks" card in the UI.
type HealthChecksConfig struct {
	PingTargets    []PingTarget   `yaml:"ping_targets"    json:"ping_targets"`
	TCPPorts       []TCPPortTest  `yaml:"tcp_ports"       json:"tcp_ports"`
	UDPPorts       []UDPPortTest  `yaml:"udp_ports"       json:"udp_ports"`
	HTTPEndpoints  []HTTPEndpoint `yaml:"http_endpoints"  json:"http_endpoints"`
	RunPerformance bool           `yaml:"run_performance" json:"run_performance"` // Master toggle for speedtest + iperf
	RunSpeedtest   bool           `yaml:"run_speedtest"   json:"run_speedtest"`   // Toggle internet speed test
	RunIperf       bool           `yaml:"run_iperf"       json:"run_iperf"`       // Toggle LAN iperf test
	RunDiscovery   bool           `yaml:"run_discovery"   json:"run_discovery"`   // Toggle network discovery card
}

// PingTarget represents a custom ping target.
type PingTarget struct {
	Name    string `yaml:"name"    json:"name"`
	Host    string `yaml:"host"    json:"host"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// TCPPortTest represents a custom TCP port test.
type TCPPortTest struct {
	Name    string `yaml:"name"    json:"name"`
	Host    string `yaml:"host"    json:"host"`
	Port    int    `yaml:"port"    json:"port"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// UDPPortTest represents a custom UDP port test.
type UDPPortTest struct {
	Name    string `yaml:"name"    json:"name"`
	Host    string `yaml:"host"    json:"host"`
	Port    int    `yaml:"port"    json:"port"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// HTTPEndpoint represents a custom HTTP endpoint test.
type HTTPEndpoint struct {
	Name           string `yaml:"name"            json:"name"`
	URL            string `yaml:"url"             json:"url"`
	ExpectedStatus int    `yaml:"expected_status" json:"expected_status"`
	Enabled        bool   `yaml:"enabled"         json:"enabled"`
}

// SpeedtestConfig contains speedtest settings.
type SpeedtestConfig struct {
	ServerID      string `yaml:"server_id"        json:"server_id"`        // Specific server ID (empty = auto)
	AutoRunOnLink bool   `yaml:"auto_run_on_link" json:"auto_run_on_link"` // Run automatically when link comes up
}

// IperfConfig contains iperf3 settings.
type IperfConfig struct {
	AutoRunOnLink bool   `yaml:"auto_run_on_link" json:"auto_run_on_link"` // Run automatically when link comes up
	Server        string `yaml:"server"           json:"server"`           // iperf3 server address
	Port          int    `yaml:"port"             json:"port"`             // iperf3 server port (default 5201)
	Protocol      string `yaml:"protocol"         json:"protocol"`         // "tcp" or "udp"
	Direction     string `yaml:"direction"        json:"direction"`        // "upload", "download", or "bidirectional"
	Duration      int    `yaml:"duration"         json:"duration"`         // Test duration in seconds
	ServerPort    int    `yaml:"server_port"      json:"server_port"`      // Port for local iperf server mode
	EnableServer  bool   `yaml:"enable_server"    json:"enable_server"`    // Enable local iperf server mode
}

// FABOptionsConfig contains FAB (Floating Action Button) settings.
type FABOptionsConfig struct {
	RunLink             bool `yaml:"run_link"              json:"run_link"`
	RunSwitch           bool `yaml:"run_switch"            json:"run_switch"`
	RunVLAN             bool `yaml:"run_vlan"              json:"run_vlan"`
	RunIPConfig         bool `yaml:"run_ip_config"         json:"run_ip_config"`
	RunGateway          bool `yaml:"run_gateway"           json:"run_gateway"`
	RunDNS              bool `yaml:"run_dns"               json:"run_dns"`
	RunHealthChecks     bool `yaml:"run_health_checks"     json:"run_health_checks"`
	RunNetworkDiscovery bool `yaml:"run_network_discovery" json:"run_network_discovery"`
	RunSpeedtest        bool `yaml:"run_speedtest"         json:"run_speedtest"`
	RunIperf            bool `yaml:"run_iperf"             json:"run_iperf"`
	RunPerformance      bool `yaml:"run_performance"       json:"run_performance"`
	AutoScanOnLink      bool `yaml:"auto_scan_on_link"     json:"auto_scan_on_link"`
}

// DisplayOptionsConfig contains display/UI settings.
type DisplayOptionsConfig struct {
	ShowPublicIP bool   `yaml:"show_public_ip" json:"show_public_ip"`
	UnitSystem   string `yaml:"unit_system"    json:"unit_system"` // "sae" (feet) or "metric" (meters)
}

// ThresholdsConfig contains all threshold settings.
type ThresholdsConfig struct {
	DHCP        DHCPThresholds   `yaml:"dhcp"         json:"dhcp"`
	DNS         Threshold        `yaml:"dns"          json:"dns"`
	Ping        Threshold        `yaml:"ping"         json:"ping"`
	WiFi        WiFiThresholds   `yaml:"wifi"         json:"wifi"`
	Link        LinkThresholds   `yaml:"link"         json:"link"`
	CustomTests CustomThresholds `yaml:"custom_tests" json:"custom_tests"`
}

// LinkThresholds contains thresholds for link stability.
type LinkThresholds struct {
	FlapCount24h IntThreshold `yaml:"flap_count_24h" json:"flap_count_24h"` // Number of link flaps in 24h
}

// IntThreshold contains warning and critical thresholds for integer values.
type IntThreshold struct {
	Warning  int `yaml:"warning"  json:"warning"`
	Critical int `yaml:"critical" json:"critical"`
}

// CustomThresholds contains thresholds for custom tests.
type CustomThresholds struct {
	Ping        Threshold            `yaml:"ping"         json:"ping"`         // Custom ping targets
	TCP         Threshold            `yaml:"tcp"          json:"tcp"`          // TCP port tests
	UDP         Threshold            `yaml:"udp"          json:"udp"`          // UDP port tests
	HTTP        Threshold            `yaml:"http"         json:"http"`         // HTTP endpoint tests (total time)
	HTTPTimings HTTPTimingThresholds `yaml:"http_timings" json:"http_timings"` // Per-phase HTTP timing thresholds
	CertExpiry  CertExpiryThreshold  `yaml:"cert_expiry"  json:"cert_expiry"`  // Certificate expiry (days)
}

// HTTPTimingThresholds contains per-phase thresholds for HTTP requests.
type HTTPTimingThresholds struct {
	DNS  Threshold `yaml:"dns"  json:"dns"`  // DNS resolution time
	TCP  Threshold `yaml:"tcp"  json:"tcp"`  // TCP connection time
	TLS  Threshold `yaml:"tls"  json:"tls"`  // TLS handshake time
	TTFB Threshold `yaml:"ttfb" json:"ttfb"` // Time to first byte (server response)
}

// CertExpiryThreshold contains certificate expiry thresholds in days.
type CertExpiryThreshold struct {
	Warning  int `yaml:"warning"  json:"warning"`  // Days until warning (e.g., 30)
	Critical int `yaml:"critical" json:"critical"` // Days until critical (e.g., 7)
}

// DHCPThresholds contains DHCP-specific thresholds.
type DHCPThresholds struct {
	Total    Threshold `yaml:"total"     json:"total"`
	PerPhase Threshold `yaml:"per_phase" json:"per_phase"`
}

// Threshold contains warning and critical values.
type Threshold struct {
	Warning  time.Duration `yaml:"warning"  json:"warning"`
	Critical time.Duration `yaml:"critical" json:"critical"`
}

// WiFiThresholds contains WiFi signal thresholds.
type WiFiThresholds struct {
	Signal SignalThreshold `yaml:"signal" json:"signal"`
}

// SignalThreshold contains signal strength thresholds in dBm.
type SignalThreshold struct {
	Warning  int `yaml:"warning"  json:"warning"`
	Critical int `yaml:"critical" json:"critical"`
}

// AuthConfig contains authentication settings.
type AuthConfig struct {
	DefaultUsername     string        `yaml:"default_username"      json:"default_username"`
	DefaultPasswordHash string        `yaml:"default_password_hash" json:"default_password_hash"`
	SessionTimeout      time.Duration `yaml:"session_timeout"       json:"session_timeout"`
	JWTSecret           string        `yaml:"jwt_secret,omitempty"  json:"jwt_secret,omitempty"`
	SSO                 SSOConfig     `yaml:"sso,omitempty"         json:"sso,omitzero"`
}

// SSOConfig contains settings for all SSO providers.
type SSOConfig struct {
	Providers []SSOProviderConfig `yaml:"providers" json:"providers"`
}

// SSOProviderConfig contains settings for a single SSO provider.
type SSOProviderConfig struct {
	Enabled      bool     `yaml:"enabled"             json:"enabled"`
	Name         string   `yaml:"name"                json:"name"`
	ClientID     string   `yaml:"client_id"           json:"client_id"`
	ClientSecret string   `yaml:"client_secret"       json:"client_secret"`
	RedirectURL  string   `yaml:"redirect_url"        json:"redirect_url"`
	Scopes       []string `yaml:"scopes,omitempty"    json:"scopes,omitempty"`    // Custom OAuth scopes (uses defaults if empty)
	TenantID     string   `yaml:"tenant_id,omitempty" json:"tenant_id,omitempty"` // Microsoft only: "common", "organizations", "consumers", or specific tenant
}

// SecurityConfig contains security settings for CORS and WebSocket origins.
type SecurityConfig struct {
	// AllowedOrigins specifies explicit origins allowed for CORS and WebSocket.
	// If empty, defaults to RFC 1918 private network ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x).
	// Use "*" to allow all origins (not recommended for production).
	// Examples: ["http://192.168.1.100:8080", "https://seed.local"]
	AllowedOrigins []string `yaml:"allowed_origins" json:"allowed_origins"`

	// VulnerabilityScanning configures CVE vulnerability scanning for discovered devices.
	VulnerabilityScanning VulnerabilityScanConfig `yaml:"vulnerability_scanning" json:"vulnerability_scanning"`
}

// DHCPConfig contains DHCP monitoring and security settings.
type DHCPConfig struct {
	// RogueDetection configures rogue DHCP server detection.
	RogueDetection RogueDetectionConfig `yaml:"rogue_detection" json:"rogue_detection"`
}

// RogueDetectionConfig contains settings for rogue DHCP server detection.
type RogueDetectionConfig struct {
	Enabled          bool     `yaml:"enabled"            json:"enabled"`
	KnownServers     []string `yaml:"known_servers"      json:"known_servers"`
	AlertOnDetection bool     `yaml:"alert_on_detection" json:"alert_on_detection"`
}

// VulnerabilityScanConfig contains settings for CVE vulnerability scanning.
type VulnerabilityScanConfig struct {
	Enabled           bool   `yaml:"enabled"            json:"enabled"`
	CVEDatabase       string `yaml:"cve_database"       json:"cve_database"`       // "nvd" or "local"
	NVDAPIKey         string `yaml:"nvd_api_key"        json:"nvd_api_key"`        // Optional NVD API key
	UpdateInterval    int    `yaml:"update_interval"    json:"update_interval"`    // Seconds between updates
	SeverityThreshold string `yaml:"severity_threshold" json:"severity_threshold"` // "low", "medium", "high", "critical"
	MaxConcurrent     int    `yaml:"max_concurrent"     json:"max_concurrent"`     // Max concurrent vulnerability checks
	AutoScan          bool   `yaml:"auto_scan"          json:"auto_scan"`          // Auto-scan after device discovery
}

// SNMPConfig contains SNMP settings for device interrogation.
type SNMPConfig struct {
	// Communities is a list of SNMP v1/v2c community strings to try (read-only).
	Communities []string `yaml:"communities" json:"communities"`

	// V3Credentials for SNMP v3 authentication.
	V3Credentials []SNMPv3Credential `yaml:"v3_credentials,omitempty" json:"v3_credentials,omitempty"`

	// Timeout for SNMP queries.
	Timeout time.Duration `yaml:"timeout" json:"timeout"`

	// Retries for failed SNMP queries.
	Retries int `yaml:"retries" json:"retries"`

	// Port for SNMP queries (default 161).
	Port int `yaml:"port" json:"port"`

	// MaxRepetitions controls how many OID values are returned per GetBulk request.
	// Lower values reduce memory usage and network load on slow devices.
	// Default: 10. Range: 1-50.
	MaxRepetitions uint32 `yaml:"max_repetitions" json:"max_repetitions"`
}

// SNMPv3Credential contains SNMP v3 authentication credentials.
type SNMPv3Credential struct {
	Name     string `yaml:"name"           json:"name"`     // Friendly name for this credential set
	Username string `yaml:"username"       json:"username"` // Security name (user)
	// AuthProtocol specifies the authentication protocol.
	// Supported values: "SHA", "SHA256", "SHA512", or "" for noAuth.
	// Note: The "MD5" value is cryptographically broken and will be removed in the next major version.
	// Use SHA256 or SHA512 instead for secure authentication.
	AuthProtocol  string `yaml:"auth_protocol"  json:"auth_protocol"`  // "SHA", "SHA256", "SHA512", or "" for noAuth (MD5 is deprecated)
	AuthPassword  string `yaml:"auth_password"  json:"auth_password"`  // Authentication password
	PrivProtocol  string `yaml:"priv_protocol"  json:"priv_protocol"`  // "DES", "AES", "AES192", "AES256", or "" for noPriv
	PrivPassword  string `yaml:"priv_password"  json:"priv_password"`  // Privacy password
	ContextName   string `yaml:"context_name"   json:"context_name"`   // Optional SNMP context
	SecurityLevel string `yaml:"security_level" json:"security_level"` // "noAuthNoPriv", "authNoPriv", "authPriv"
}

// LoggingConfig contains structured logging settings.
type LoggingConfig struct {
	Level      string `yaml:"level"       json:"level"`       // DEBUG, INFO, WARN, ERROR (default: INFO)
	Format     string `yaml:"format"      json:"format"`      // text or json (default: text)
	AddSource  bool   `yaml:"add_source"  json:"add_source"`  // Include file:line in logs
	File       string `yaml:"file"        json:"file"`        // Log file path (empty = stdout only)
	MaxSize    int    `yaml:"max_size"    json:"max_size"`    // Max MB per log file before rotation
	MaxBackups int    `yaml:"max_backups" json:"max_backups"` // Number of old files to keep
	MaxAge     int    `yaml:"max_age"     json:"max_age"`     // Days to keep old files
	Compress   bool   `yaml:"compress"    json:"compress"`    // Compress rotated files
}

// MCPConfig contains MCP (Model Context Protocol) server settings.
// MCP enables AI assistants like Claude to interact with the network diagnostics tools.
type MCPConfig struct {
	// Enabled enables the MCP server endpoint.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// RequireAuth requires JWT authentication for MCP connections.
	// When true, MCP requests must include a valid Bearer token.
	RequireAuth bool `yaml:"require_auth" json:"require_auth"`

	// RateLimitPerMinute limits requests per minute per client.
	// Set to 0 for unlimited (not recommended).
	RateLimitPerMinute int `yaml:"rate_limit_per_minute" json:"rate_limit_per_minute"`

	// AllowedTools lists specific tools to expose via MCP.
	// Empty list means all tools are available.
	AllowedTools []string `yaml:"allowed_tools,omitempty" json:"allowed_tools,omitempty"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Version: ConfigVersion,
		Server:  ServerConfig{Port: defaultHTTPSPort, HTTPS: true},
		Interface: InterfaceConfig{
			Default:          "",
			Fallbacks:        []string{},
			StartupRetries:   defaultStartupRetries,
			StartupRetryWait: defaultStartupRetryWaitSec * time.Second,
		},
		VLAN:             VLANConfig{Enabled: false, ID: 0},
		IP:               IPConfig{Mode: ipModeDHCP},
		Discovery:        DiscoveryConfig{Protocol: "auto", Timeout: defaultDiscoveryTimeoutSec * time.Second},
		NetworkDiscovery: defaultNetworkDiscoveryConfig(),
		SNMP: SNMPConfig{
			Communities:    []string{"public"},
			Timeout:        defaultSNMPTimeoutSec * time.Second,
			Retries:        defaultSNMPRetries,
			Port:           defaultSNMPPort,
			MaxRepetitions: defaultSNMPMaxRepetitions,
		},
		DNS:          DNSConfig{TestHostname: "google.com", Timeout: defaultDNSTimeoutSec * time.Second},
		HealthChecks: defaultHealthChecksConfig(),
		Speedtest:    SpeedtestConfig{ServerID: "", AutoRunOnLink: true},
		Thresholds:   defaultThresholdsConfig(),
		Auth:         defaultAuthConfig(),
		Security:     defaultSecurityConfig(),
		Iperf: IperfConfig{
			AutoRunOnLink: false,
			Server:        "",
			Port:          defaultIperfPort,
			Protocol:      "tcp",
			Direction:     "download",
			Duration:      defaultIperfDurationSec,
			ServerPort:    defaultIperfPort,
			EnableServer:  true,
		},
		FABOptions: FABOptionsConfig{
			RunLink:             true,
			RunSwitch:           true,
			RunVLAN:             true,
			RunIPConfig:         true,
			RunGateway:          true,
			RunDNS:              true,
			RunHealthChecks:     true,
			RunNetworkDiscovery: true,
			RunSpeedtest:        true,
			RunIperf:            false,
			RunPerformance:      true,
			AutoScanOnLink:      true,
		},
		DisplayOptions: DisplayOptionsConfig{ShowPublicIP: true, UnitSystem: "sae"},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "text",
			AddSource:  false,
			File:       "",
			MaxSize:    defaultLogMaxSizeMB,
			MaxBackups: defaultLogMaxBackups,
			MaxAge:     defaultLogMaxAgeDays,
			Compress:   true,
		},
		MCP: MCPConfig{
			Enabled:            false,
			RequireAuth:        true,
			RateLimitPerMinute: defaultRateLimitPerMinute,
			AllowedTools:       nil,
		},
		Database: DatabaseConfig{
			Path:           "data/seed.db",
			RetentionDays:  defaultDBRetentionDays,
			EnableWAL:      true,
			MaxConnections: defaultDBMaxConnections,
		},
		Pipeline: defaultPipelineConfig(),
	}
}

// defaultNetworkDiscoveryConfig returns the default network discovery configuration.
func defaultNetworkDiscoveryConfig() NetworkDiscoveryConfig {
	return NetworkDiscoveryConfig{
		Options: DiscoveryOptions{
			PassiveProtocols: PassiveProtocolConfig{LLDP: true, CDP: true, EDP: true, NDP: true},
			ARPScan:          true, ICMPScan: true,
			PortScan: PortScanConfig{
				Enabled:       false,
				Preset:        PortPresetCommon,
				TCPPorts:      "",
				UDPPorts:      "",
				BannerTimeout: defaultBannerTimeoutSec * time.Second,
			},
			TCPProbe: TCPProbeConfig{
				Timeout: defaultTracerouteTimeoutSec * time.Second,
				Workers: defaultTracerouteWorkers,
			}, Traceroute: false, SNMPQuery: false,
		},
		Profiler: DeviceProfilerConfig{
			Enabled:       true,
			Timeout:       defaultMDNSTimeoutSec * time.Second,
			MaxConcurrent: defaultMDNSMaxConcurrent,
			QuickPorts:    []int{portSSH, portHTTP, portHTTPS, portHTTPAlt},
		},
		Timing: DiscoveryTiming{
			ProbeInterval:  defaultProbeIntervalMs * time.Millisecond,
			RescanInterval: defaultRescanIntervalMin * time.Minute,
			Workers:        defaultARPWorkers,
		},
		Fingerprinting: FingerprintingConfig{
			Enabled:       false,
			OSDetection:   false,
			ServiceProbes: false,
		},
		IPv6Enabled: true, Enabled: true, ARPScanWorkers: defaultARPWorkers, PingTimeout: defaultPingTimeoutMs * time.Millisecond, ScanTimeout: defaultScanTimeoutSec * time.Second,
		AutoScan: true, ScanInterval: 0, OUIFilePath: "data/oui.txt", OUIMaxAge: defaultOUIMaxAgeDays * 24 * time.Hour, AdditionalSubnets: []SubnetConfig{},
	}
}

// defaultHealthChecksConfig returns the default health checks configuration.
func defaultHealthChecksConfig() HealthChecksConfig {
	return HealthChecksConfig{
		PingTargets: []PingTarget{
			{Name: "Google DNS", Host: "8.8.8.8", Enabled: true},
			{Name: "Cloudflare", Host: "1.1.1.1", Enabled: true},
		},
		TCPPorts: []TCPPortTest{
			{
				Name:    "HTTPS",
				Host:    "www.google.com",
				Port:    portHTTPS,
				Enabled: true,
			}, {Name: "DICOM", Host: "dicomserver.co.uk", Port: portDICOM, Enabled: true},
			{
				Name:    "FTP",
				Host:    "ftp.debian.org",
				Port:    portFTP,
				Enabled: true,
			}, {Name: "SMB", Host: "files.example.com", Port: portSMB, Enabled: false},
			{
				Name:    "RTSP",
				Host:    "wowzaec2demo.streamlock.net",
				Port:    portRTSP,
				Enabled: true,
			}, {Name: "PostgreSQL", Host: "db.example.com", Port: portPostgreSQL, Enabled: false},
			{Name: "SFTP", Host: "sftp.example.com", Port: portSSH, Enabled: false},
		},
		UDPPorts: []UDPPortTest{
			{Name: "DNS", Host: "8.8.8.8", Port: portDNS, Enabled: true},
			{Name: "NTP", Host: "time.google.com", Port: portNTP, Enabled: true},
		},
		HTTPEndpoints: []HTTPEndpoint{
			{
				Name:           "Google HTTPS",
				URL:            "https://www.google.com",
				ExpectedStatus: httpStatusOK,
				Enabled:        true,
			},
			{
				Name:           "Cloudflare",
				URL:            "https://www.cloudflare.com",
				ExpectedStatus: httpStatusOK,
				Enabled:        true,
			},
			{Name: "Example HTTP", URL: "http://example.com", ExpectedStatus: httpStatusOK, Enabled: true},
		},
		RunPerformance: true, RunSpeedtest: true, RunIperf: true, RunDiscovery: true,
	}
}

// defaultThresholdsConfig returns the default thresholds configuration.
func defaultThresholdsConfig() ThresholdsConfig {
	return ThresholdsConfig{
		DHCP: DHCPThresholds{
			Total: Threshold{
				Warning:  thresholdDHCPTotalWarningMs * time.Millisecond,
				Critical: defaultBannerTimeoutSec * time.Second,
			},
			PerPhase: Threshold{Warning: thresholdDHCPPhaseWarningMs * time.Millisecond, Critical: 1 * time.Second},
		},
		DNS: Threshold{
			Warning:  thresholdDNSWarningMs * time.Millisecond,
			Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
		},
		Ping: Threshold{
			Warning:  thresholdPingWarningMs * time.Millisecond,
			Critical: thresholdPingCriticalMs * time.Millisecond,
		},
		WiFi: WiFiThresholds{
			Signal: SignalThreshold{Warning: thresholdWiFiSignalWarningDBm, Critical: thresholdWiFiSignalCriticalDBm},
		},
		Link: LinkThresholds{
			FlapCount24h: IntThreshold{Warning: thresholdLinkFlapWarning, Critical: thresholdLinkFlapCritical},
		},
		CustomTests: CustomThresholds{
			Ping: Threshold{
				Warning:  thresholdPingWarningMs * time.Millisecond,
				Critical: thresholdCustomPingCriticalMs * time.Millisecond,
			},
			TCP: Threshold{
				Warning:  thresholdTCPWarningMs * time.Millisecond,
				Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
			},
			UDP: Threshold{
				Warning:  thresholdTCPWarningMs * time.Millisecond,
				Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
			},
			HTTP: Threshold{
				Warning:  thresholdHTTPWarningMs * time.Millisecond,
				Critical: defaultBannerTimeoutSec * time.Second,
			},
			HTTPTimings: HTTPTimingThresholds{
				DNS: Threshold{
					Warning:  thresholdDNSWarningMs * time.Millisecond,
					Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
				},
				TCP: Threshold{
					Warning:  thresholdTCPWarningMs * time.Millisecond,
					Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
				},
				TLS: Threshold{
					Warning:  thresholdTLSWarningMs * time.Millisecond,
					Critical: thresholdDHCPTotalWarningMs * time.Millisecond,
				},
				TTFB: Threshold{
					Warning:  thresholdHTTPWarningMs * time.Millisecond,
					Critical: defaultBannerTimeoutSec * time.Second,
				},
			},
			CertExpiry: CertExpiryThreshold{
				Warning:  thresholdCertExpiryWarningDays,
				Critical: thresholdCertExpiryCriticalDays,
			},
		},
	}
}

// defaultAuthConfig returns the default authentication configuration.
func defaultAuthConfig() AuthConfig {
	return AuthConfig{
		DefaultUsername: "admin", DefaultPasswordHash: "", SessionTimeout: defaultSessionTimeoutHours * time.Hour, JWTSecret: "",
		SSO: SSOConfig{
			Providers: []SSOProviderConfig{
				{Name: "google", Enabled: false},
				{Name: "microsoft", Enabled: false},
				{Name: "github", Enabled: false},
			},
		},
	}
}

// defaultSecurityConfig returns the default security configuration.
func defaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		AllowedOrigins: []string{},
		VulnerabilityScanning: VulnerabilityScanConfig{
			Enabled:           true,
			CVEDatabase:       "nvd",
			NVDAPIKey:         "",
			UpdateInterval:    defaultVulnUpdateIntervalSec,
			SeverityThreshold: "medium",
			MaxConcurrent:     defaultMDNSMaxConcurrent,
			AutoScan:          true,
		},
	}
}

// defaultPipelineConfig returns the default pipeline configuration.
func defaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Phases: PipelinePhaseConfig{
			Enumeration:      true,
			NameResolution:   true,
			ServiceDiscovery: true,
			VulnAssessment:   false,
		},
		Timing: PipelineTimingConfig{
			ProbeDelay:         defaultPipelineProbeDelayMs * time.Millisecond,
			HostDelay:          defaultPipelineHostDelayMs * time.Millisecond,
			MaxConcurrentHosts: defaultPipelineMaxConcurrentHosts,
			PhaseTimeout:       defaultPipelinePhaseTimeoutMin * time.Minute,
			Profile:            "normal",
		},
		PortScan: PipelinePortScanConfig{
			Intensity:      "off",
			BannerGrab:     true,
			ConnectTimeout: defaultBannerTimeoutSec * time.Second,
		},
		SNMPCollection: PipelineSNMPConfig{
			Enabled: true,
			MIBs: PipelineSNMPMIBs{
				System:      true,
				Interfaces:  true,
				IPAddresses: true,
				Routing:     false,
				Bridge:      false,
				Entity:      false,
				LLDP:        true,
				VLAN:        false,
			},
			WalkTimeout: defaultSNMPWalkTimeoutSec * time.Second, MaxOIDsPerRequest: defaultSNMPMaxOIDsPerRequest,
		},
		Persistence: PipelinePersistenceConfig{
			StoreHistory:       true,
			StalenessThreshold: defaultStalenessThresholdHours * time.Hour,
			PurgeAfter:         defaultPurgeAfterDays * 24 * time.Hour,
		},
	}
}

// Load reads configuration from a YAML file.
// If the config has no version or an older version, it will be updated.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if file doesn't exist
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if unmarshalErr := yaml.Unmarshal(data, cfg); unmarshalErr != nil {
		return nil, fmt.Errorf("parse config yaml: %w", unmarshalErr)
	}

	// Handle unversioned configs (version 0 means unversioned)
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		logging.GetLogger().
			Info("Upgraded unversioned config to current version", "version", ConfigVersion)
	}

	return cfg, nil
}

// LoadWithMigration reads configuration from a YAML file and applies any necessary migrations.
// It creates a backup before applying migrations.
func LoadWithMigration(path string, migrator *MigrationManager) (*Config, bool, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, false, nil // Use defaults if file doesn't exist
		}
		return nil, false, fmt.Errorf("read config file: %w", err)
	}

	// Check current version in file
	var partial struct {
		Version int `yaml:"version" json:"version"`
	}
	if partialErr := yaml.Unmarshal(data, &partial); partialErr != nil {
		return nil, false, fmt.Errorf("failed to parse config version: %w", partialErr)
	}

	migrated := false
	if partial.Version < ConfigVersion && migrator != nil {
		// Create backup before migration
		backupMgr := NewBackupManager(path, "", defaultBackupMaxCount)
		if _, backupErr := backupMgr.CreateBackup(); backupErr != nil {
			logging.GetLogger().Warn("Failed to create backup before migration", "error", backupErr)
		}

		// Apply migrations
		migratedData, migrateErr := migrator.Migrate(data, partial.Version, ConfigVersion)
		if migrateErr != nil {
			return nil, false, fmt.Errorf("failed to migrate config from v%d to v%d: %w",
				partial.Version, ConfigVersion, migrateErr)
		}
		data = migratedData
		migrated = true
		logging.GetLogger().
			Info("Migrated config", "from_version", partial.Version, "to_version", ConfigVersion)
	}

	if unmarshalErr := yaml.Unmarshal(data, cfg); unmarshalErr != nil {
		return nil, false, fmt.Errorf("parse config yaml: %w", unmarshalErr)
	}

	// Ensure version is set
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
		migrated = true
	}

	// Save migrated config
	if migrated {
		if saveErr := cfg.Save(path); saveErr != nil {
			logging.GetLogger().Warn("Failed to save migrated config", "error", saveErr)
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
		errs = append(
			errs,
			fmt.Sprintf("server.port must be between 1-65535, got %d", c.Server.Port),
		)
	}
	if c.Server.HTTPRedirectPort < 0 || c.Server.HTTPRedirectPort > 65535 {
		errs = append(
			errs,
			fmt.Sprintf(
				"server.http_redirect_port must be between 0-65535, got %d",
				c.Server.HTTPRedirectPort,
			),
		)
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
		errs = append(
			errs,
			fmt.Sprintf(
				"interface.startup_retries must be >= 0, got %d",
				c.Interface.StartupRetries,
			),
		)
	}
	if c.Interface.StartupRetryWait < 0 {
		errs = append(
			errs,
			fmt.Sprintf(
				"interface.startup_retry_wait must be >= 0, got %s",
				c.Interface.StartupRetryWait,
			),
		)
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
	if c.IP.Mode != ipModeStatic {
		return errs
	}
	// Fixes #896: Check for nil Static config before accessing fields
	if c.IP.Static == nil {
		return append(errs, "ip.static is required when ip.mode is 'static'")
	}
	errs = append(errs, c.validateStaticIPFields()...)
	return errs
}

// validateStaticIPFields validates individual static IP configuration fields.
func (c *Config) validateStaticIPFields() []string {
	var errs []string
	if c.IP.Static.Address == "" {
		errs = append(errs, "ip.static.address is required when ip.mode is 'static'")
	}
	if c.IP.Static.Netmask == "" {
		errs = append(errs, "ip.static.netmask is required when ip.mode is 'static'")
	}
	if c.IP.Static.Gateway == "" {
		errs = append(errs, "ip.static.gateway is required when ip.mode is 'static'")
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
		errs = append(
			errs,
			fmt.Sprintf(
				"network_discovery.arp_scan_workers must be between 1-500, got %d",
				c.NetworkDiscovery.ARPScanWorkers,
			),
		)
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
		errs = append(
			errs,
			fmt.Sprintf("snmp.retries must be between 0-10, got %d", c.SNMP.Retries),
		)
	}
	if c.SNMP.Timeout <= 0 {
		errs = append(errs, "snmp.timeout must be positive")
	}
	return errs
}

// WarnDeprecatedSNMPSettings logs warnings for deprecated SNMP configurations.
// This function should be called after logging is initialized.
func (c *Config) WarnDeprecatedSNMPSettings() {
	c.RLock()
	defer c.RUnlock()

	// Check for MD5 authentication protocol in SNMPv3 credentials
	// MD5 is cryptographically broken and will be removed in the next major version
	for i := range c.SNMP.V3Credentials {
		cred := &c.SNMP.V3Credentials[i]
		if cred.AuthProtocol == "MD5" {
			logging.GetLogger().Warn(
				"SNMP MD5 authentication is deprecated and will be removed in the next major version",
				"credential_name",
				cred.Name,
				"username",
				cred.Username,
				"recommendation",
				"Use SHA256 or SHA512 for secure authentication",
			)
		}
	}
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
		errs = append(
			errs,
			fmt.Sprintf(
				"logging.level must be one of debug, info, warn, error; got %q",
				c.Logging.Level,
			),
		)
	}

	// Validate format
	format := strings.ToLower(c.Logging.Format)
	if format != "" && format != "text" && format != "json" {
		errs = append(
			errs,
			fmt.Sprintf("logging.format must be 'text' or 'json'; got %q", c.Logging.Format),
		)
	}

	// Validate rotation settings
	if c.Logging.MaxSize < 0 {
		errs = append(errs, fmt.Sprintf("logging.max_size must be >= 0, got %d", c.Logging.MaxSize))
	}
	if c.Logging.MaxBackups < 0 {
		errs = append(
			errs,
			fmt.Sprintf("logging.max_backups must be >= 0, got %d", c.Logging.MaxBackups),
		)
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
		return fmt.Errorf("marshal config yaml: %w", err)
	}
	if writeErr := os.WriteFile(path, data, 0o600); writeErr != nil {
		return fmt.Errorf("write config file: %w", writeErr)
	}
	return nil
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
func EnsureConfig(
	path string,
	checkDefaultPassword func(hash string) bool,
) (*Config, *SetupResult, error) {
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
		if saveErr := cfg.Save(path); saveErr != nil {
			return nil, nil, fmt.Errorf("failed to save config: %w", saveErr)
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
		logging.GetLogger().Warn(
			"Configured interface has no IPv4 address or doesn't exist",
			"interface",
			c.Interface.Default,
		)
	}

	// Try fallback interfaces
	for _, iface := range c.Interface.Fallbacks {
		if hasIPv4Address(iface) {
			logging.GetLogger().Info("Using fallback interface", "interface", iface)
			return iface, true
		}
	}

	// Auto-detect: scan all interfaces for one with an IPv4 address
	detected := detectActiveInterface()
	if detected != "" {
		logging.GetLogger().Info("Auto-detected active interface", "interface", detected)
		return detected, true
	}

	// Last resort: return the configured default even if it might not work
	if c.Interface.Default != "" {
		logging.GetLogger().Warn(
			"No active interface found, using configured default",
			"interface",
			c.Interface.Default,
		)
		return c.Interface.Default, true
	}

	// No hardcoded fallback - return empty to signal no interface found (#572)
	logging.GetLogger().Error("No active network interface found")
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
