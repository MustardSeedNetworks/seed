package config

// config_types_health.go contains the HealthChecksConfig type tree (custom
// ping, TCP/UDP/HTTP/RTSP/DICOM/HL7/FHIR/SQL/FileShare/LDAP/LTI/OPCUA/Modbus
// endpoints), the Speedtest/Iperf perf-test types, and the shared threshold
// types used by health-check evaluation.

import "time"

// HealthChecksConfig contains custom health check configurations.
// This section corresponds to the "Health Checks" card in the UI.
type HealthChecksConfig struct {
	PingTargets        []PingTarget        `json:"ping_targets"`
	TCPPorts           []TCPPortTest       `json:"tcp_ports"`
	UDPPorts           []UDPPortTest       `json:"udp_ports"`
	HTTPEndpoints      []HTTPEndpoint      `json:"http_endpoints"`
	RTSPEndpoints      []RTSPEndpoint      `json:"rtsp_endpoints"`      // Issue #778
	DICOMEndpoints     []DICOMEndpoint     `json:"dicom_endpoints"`     // Issue #777
	HL7Endpoints       []HL7Endpoint       `json:"hl7_endpoints"`       // Health Checks 100x - Medical
	FHIREndpoints      []FHIREndpoint      `json:"fhir_endpoints"`      // Health Checks 100x - Medical
	SQLEndpoints       []SQLEndpoint       `json:"sql_endpoints"`       // Health Checks 100x - Enterprise
	FileShareEndpoints []FileShareEndpoint `json:"fileshare_endpoints"` // Health Checks 100x - Enterprise
	LDAPEndpoints      []LDAPEndpoint      `json:"ldap_endpoints"`      // Health Checks 100x - Enterprise
	LTIEndpoints       []LTIEndpoint       `json:"lti_endpoints"`       // Health Checks 100x - Education
	OPCUAEndpoints     []OPCUAEndpoint     `json:"opcua_endpoints"`     // Health Checks 100x - Manufacturing
	ModbusEndpoints    []ModbusEndpoint    `json:"modbus_endpoints"`    // Health Checks 100x - Manufacturing
	RunPerformance     bool                `json:"run_performance"`     // Master toggle for speedtest + iperf
	RunSpeedtest       bool                `json:"run_speedtest"`       // Toggle internet speed test
	RunIperf           bool                `json:"run_iperf"`           // Toggle LAN iperf test
	RunDiscovery       bool                `json:"run_discovery"`       // Toggle network discovery card
}

// PingTarget represents a custom ping target.
type PingTarget struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Enabled bool   `json:"enabled"`
}

// TCPPortTest represents a custom TCP port test.
type TCPPortTest struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// UDPPortTest represents a custom UDP port test.
type UDPPortTest struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
	Enabled bool   `json:"enabled"`
}

// HTTPEndpoint represents a custom HTTP endpoint test.
type HTTPEndpoint struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
	Enabled        bool   `json:"enabled"`
	// HTTP enhancements (Health Checks 100x)
	BodyMatch            string `json:"body_match,omitempty"`             // Regex/substring to match in response body
	BodyMatchIsRegex     bool   `json:"body_match_is_regex,omitempty"`    // Whether body_match is a regex
	CheckSecurityHeaders bool   `json:"check_security_headers,omitempty"` // Check HSTS, CSP, etc.
	FollowRedirects      bool   `json:"follow_redirects,omitempty"`       // Track redirect chain
	MaxRedirects         int    `json:"max_redirects,omitempty"`          // Max redirect hops (default 10)
}

// RTSPEndpoint represents a custom RTSP stream test (Issue #778).
type RTSPEndpoint struct {
	Name    string `json:"name"`
	URL     string `json:"url"` // rtsp://host:port/path
	Enabled bool   `json:"enabled"`
}

// DICOMEndpoint represents a custom DICOM server test (Issue #777).
type DICOMEndpoint struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`       // Default 104
	CalledAE  string `json:"called_ae"`  // Called Application Entity title
	CallingAE string `json:"calling_ae"` // Calling Application Entity title
	Enabled   bool   `json:"enabled"`
}

// HL7Endpoint represents a custom HL7 MLLP endpoint test (Health Checks 100x).
type HL7Endpoint struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`               // Default: 2575
	SendingApp   string `json:"sending_app"`        // Sending application name
	SendingFac   string `json:"sending_facility"`   // Sending facility name
	ReceivingApp string `json:"receiving_app"`      // Receiving application name
	ReceivingFac string `json:"receiving_facility"` // Receiving facility name
	Enabled      bool   `json:"enabled"`
}

// FHIREndpoint represents a custom FHIR R4 endpoint test (Health Checks 100x).
type FHIREndpoint struct {
	Name         string `json:"name"`
	BaseURL      string `json:"base_url"`  // FHIR server base URL
	AuthType     string `json:"auth_type"` // "none", "basic", "bearer", "oauth2"
	Username     string `json:"username,omitempty"`
	Password     string `json:"password,omitempty"`
	BearerToken  string `json:"bearer_token,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	TokenURL     string `json:"token_url,omitempty"` // OAuth2 token endpoint
	Enabled      bool   `json:"enabled"`
}

// SQLEndpoint represents a custom SQL database test (Health Checks 100x).
type SQLEndpoint struct {
	Name      string `json:"name"`
	Driver    string `json:"driver"` // "mysql", "postgres", "sqlserver", "oracle", "sqlite"
	Host      string `json:"host"`
	Port      int    `json:"port"` // Default varies by driver
	Database  string `json:"database"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	SSLMode   string `json:"ssl_mode,omitempty"`   // "disable", "require", "verify-ca", "verify-full"
	TestQuery string `json:"test_query,omitempty"` // Custom query to execute (default: SELECT 1)
	Enabled   bool   `json:"enabled"`
}

// FileShareEndpoint represents a file share test (SMB/CIFS or NFS) with performance testing.
type FileShareEndpoint struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"` // "smb", "nfs"
	Host     string `json:"host"`
	Share    string `json:"share"`              // Share name (SMB) or export path (NFS)
	Path     string `json:"path,omitempty"`     // Optional subdirectory path
	Username string `json:"username,omitempty"` // SMB authentication
	Password string `json:"password,omitempty"`
	Domain   string `json:"domain,omitempty"` // SMB domain
	// Performance testing options
	TestReadPerformance  bool `json:"test_read_performance,omitempty"`  // Read test file to measure speed
	TestWritePerformance bool `json:"test_write_performance,omitempty"` // Write test file to measure speed
	TestFileSizeMB       int  `json:"test_file_size_mb,omitempty"`      // Size of test file in MB (default: 10)
	Enabled              bool `json:"enabled"`
}

// LDAPEndpoint represents an LDAP/Active Directory test (Health Checks 100x).
type LDAPEndpoint struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`              // Default: 389 (LDAP), 636 (LDAPS)
	UseTLS       bool   `json:"use_tls"`           // Use LDAPS
	StartTLS     bool   `json:"start_tls"`         // Use StartTLS
	BaseDN       string `json:"base_dn"`           // e.g., "dc=example,dc=com"
	BindDN       string `json:"bind_dn,omitempty"` // Bind user DN
	BindPassword string `json:"bind_password,omitempty"`
	SearchFilter string `json:"search_filter,omitempty"` // Test search filter
	Enabled      bool   `json:"enabled"`
}

// LTIEndpoint represents an LTI/LMS endpoint test (Health Checks 100x - Education).
type LTIEndpoint struct {
	Name           string `json:"name"`
	LaunchURL      string `json:"launch_url"`                // LTI tool launch URL
	ConsumerKey    string `json:"consumer_key,omitempty"`    // OAuth consumer key
	ConsumerSecret string `json:"consumer_secret,omitempty"` // OAuth consumer secret (for LTI 1.x)
	LTIVersion     string `json:"lti_version,omitempty"`     // "1.1", "1.3", or "advantage"
	ClientID       string `json:"client_id,omitempty"`       // LTI 1.3 client ID
	DeploymentID   string `json:"deployment_id,omitempty"`   // LTI 1.3 deployment ID
	PlatformURL    string `json:"platform_url,omitempty"`    // LTI 1.3 platform issuer URL
	Enabled        bool   `json:"enabled"`
}

// OPCUAEndpoint represents an OPC-UA endpoint test (Health Checks 100x - Manufacturing).
type OPCUAEndpoint struct {
	Name           string `json:"name"`
	EndpointURL    string `json:"endpoint_url"`              // opc.tcp://host:4840/path
	SecurityMode   string `json:"security_mode,omitempty"`   // "None", "Sign", "SignAndEncrypt"
	SecurityPolicy string `json:"security_policy,omitempty"` // "None", "Basic128Rsa15", "Basic256", "Basic256Sha256"
	Username       string `json:"username,omitempty"`
	Password       string `json:"password,omitempty"`
	CertPath       string `json:"cert_path,omitempty"` // Client certificate path
	KeyPath        string `json:"key_path,omitempty"`  // Client private key path
	Enabled        bool   `json:"enabled"`
}

// ModbusEndpoint represents a Modbus TCP endpoint test (Health Checks 100x - Manufacturing).
type ModbusEndpoint struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`                    // Default: 502
	UnitID       int    `json:"unit_id"`                 // Modbus unit/slave ID (1-247)
	TestRegister int    `json:"test_register"`           // Register address to read for testing
	RegisterType string `json:"register_type,omitempty"` // "holding", "input", "coil", "discrete"
	Enabled      bool   `json:"enabled"`
}

// SpeedtestConfig contains speedtest settings.
type SpeedtestConfig struct {
	ServerID      string `json:"server_id"`        // Specific server ID (empty = auto)
	AutoRunOnLink bool   `json:"auto_run_on_link"` // Run automatically when link comes up
}

// IperfConfig contains iperf3 settings.
type IperfConfig struct {
	AutoRunOnLink bool   `json:"auto_run_on_link"` // Run automatically when link comes up
	Server        string `json:"server"`           // iperf3 server address
	Port          int    `json:"port"`             // iperf3 server port (default 5201)
	Protocol      string `json:"protocol"`         // "tcp" or "udp"
	Direction     string `json:"direction"`        // "upload", "download", or "bidirectional"
	Duration      int    `json:"duration"`         // Test duration in seconds
	ServerPort    int    `json:"server_port"`      // Port for local iperf server mode
	EnableServer  bool   `json:"enable_server"`    // Enable local iperf server mode
}

// ThresholdsConfig contains all threshold settings.
type ThresholdsConfig struct {
	DHCP        DHCPThresholds   `json:"dhcp"`
	DNS         Threshold        `json:"dns"`
	Ping        Threshold        `json:"ping"`
	WiFi        WiFiThresholds   `json:"wifi"`
	Link        LinkThresholds   `json:"link"`
	CustomTests CustomThresholds `json:"custom_tests"`
}

// LinkThresholds contains thresholds for link stability.
type LinkThresholds struct {
	FlapCount24h IntThreshold `json:"flap_count_24h"` // Number of link flaps in 24h
}

// IntThreshold contains warning and critical thresholds for integer values.
type IntThreshold struct {
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}

// CustomThresholds contains thresholds for custom tests.
type CustomThresholds struct {
	Ping        Threshold            `json:"ping"`         // Custom ping targets
	TCP         Threshold            `json:"tcp"`          // TCP port tests
	UDP         Threshold            `json:"udp"`          // UDP port tests
	HTTP        Threshold            `json:"http"`         // HTTP endpoint tests (total time)
	HTTPTimings HTTPTimingThresholds `json:"http_timings"` // Per-phase HTTP timing thresholds
	CertExpiry  CertExpiryThreshold  `json:"cert_expiry"`  // Certificate expiry (days)
}

// HTTPTimingThresholds contains per-phase thresholds for HTTP requests.
type HTTPTimingThresholds struct {
	DNS  Threshold `json:"dns"`  // DNS resolution time
	TCP  Threshold `json:"tcp"`  // TCP connection time
	TLS  Threshold `json:"tls"`  // TLS handshake time
	TTFB Threshold `json:"ttfb"` // Time to first byte (server response)
}

// CertExpiryThreshold contains certificate expiry thresholds in days.
type CertExpiryThreshold struct {
	Warning  int `json:"warning"`  // Days until warning (e.g., 30)
	Critical int `json:"critical"` // Days until critical (e.g., 7)
}

// DHCPThresholds contains DHCP-specific thresholds.
type DHCPThresholds struct {
	Total    Threshold `json:"total"`
	PerPhase Threshold `json:"per_phase"`
}

// Threshold contains warning and critical values.
type Threshold struct {
	Warning  time.Duration `json:"warning"`
	Critical time.Duration `json:"critical"`
}

// WiFiThresholds contains WiFi signal thresholds.
type WiFiThresholds struct {
	Signal SignalThreshold `json:"signal"`
}

// SignalThreshold contains signal strength thresholds in dBm.
type SignalThreshold struct {
	Warning  int `json:"warning"`
	Critical int `json:"critical"`
}
