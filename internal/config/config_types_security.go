package config

// config_types_security.go contains the authentication, SSO, CORS / origin,
// DHCP-rogue, vulnerability-scan, and SNMP configuration types.

import "time"

// AuthConfig contains authentication settings.
type AuthConfig struct {
	DefaultUsername     string        `json:"default_username"`
	DefaultPasswordHash string        `json:"default_password_hash"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	JWTSecret           string        `json:"jwt_secret,omitempty"`
	SSO                 SSOConfig     `json:"sso,omitzero"`
}

// SSOConfig contains settings for all SSO providers.
type SSOConfig struct {
	Providers []SSOProviderConfig `json:"providers"`
}

// SSOProviderConfig contains settings for a single SSO provider.
type SSOProviderConfig struct {
	Enabled      bool     `json:"enabled"`
	Name         string   `json:"name"`
	ClientID     string   `json:"client_id"`
	ClientSecret string   `json:"client_secret"`
	RedirectURL  string   `json:"redirect_url"`
	Scopes       []string `json:"scopes,omitempty"`    // Custom OAuth scopes (uses defaults if empty)
	TenantID     string   `json:"tenant_id,omitempty"` // Microsoft only: "common", "organizations", "consumers", or specific tenant
}

// SecurityConfig contains security settings for CORS and WebSocket origins.
type SecurityConfig struct {
	// AllowedOrigins specifies explicit origins allowed for CORS and WebSocket.
	// If empty, defaults to RFC 1918 private network ranges (192.168.x.x, 10.x.x.x, 172.16-31.x.x).
	// Use "*" to allow all origins (not recommended for production).
	// Examples: ["http://192.168.1.100:8080", "https://seed.local"]
	AllowedOrigins []string `json:"allowed_origins"`

	// VulnerabilityScanning configures CVE vulnerability scanning for discovered devices.
	VulnerabilityScanning VulnerabilityScanConfig `json:"vulnerability_scanning"`

	// GuestNetworkAudit configures the on-demand guest-network isolation test (#397).
	GuestNetworkAudit GuestNetworkAuditConfig `json:"guest_network_audit"`
}

// GuestNetworkAuditConfig stores the list of sensitive internal targets that
// MUST NOT be reachable from a guest network, plus optional audit-run settings.
// The audit is run on demand from the UI: the technician connects the appliance
// to the guest network and triggers the test, which probes the configured
// targets and raises an alert if any are reachable (#397).
type GuestNetworkAuditConfig struct {
	// Enabled gates whether the audit endpoint accepts run requests.
	Enabled bool `json:"enabled"`

	// Targets is the list of sensitive internal hosts (EMR, PACS, etc.).
	Targets []GuestAuditTarget `json:"targets,omitempty"`

	// Ports overrides the default port list probed against each target.
	// Empty means use the package default (HTTP/HTTPS/SSH/RDP/SMB/etc).
	Ports []int `json:"ports,omitempty"`
}

// GuestAuditTarget identifies a single sensitive internal host.
type GuestAuditTarget struct {
	IP    string `json:"ip"`              // IPv4 address; validated server-side
	Label string `json:"label,omitempty"` // Friendly name (e.g. "EMR primary")
}

// DHCPConfig contains DHCP monitoring and security settings.
type DHCPConfig struct {
	// RogueDetection configures rogue DHCP server detection.
	RogueDetection RogueDetectionConfig `json:"rogue_detection"`
}

// RogueDetectionConfig contains settings for rogue DHCP server detection.
type RogueDetectionConfig struct {
	Enabled          bool     `json:"enabled"`
	KnownServers     []string `json:"known_servers"`
	AlertOnDetection bool     `json:"alert_on_detection"`
}

// VulnerabilityScanConfig contains settings for CVE vulnerability scanning.
type VulnerabilityScanConfig struct {
	Enabled           bool   `json:"enabled"`
	CVEDatabase       string `json:"cve_database"`       // "nvd" or "local"
	NVDAPIKey         string `json:"nvd_api_key"`        // Optional NVD API key
	UpdateInterval    int    `json:"update_interval"`    // Seconds between updates
	SeverityThreshold string `json:"severity_threshold"` // "low", "medium", "high", "critical"
	MaxConcurrent     int    `json:"max_concurrent"`     // Max concurrent vulnerability checks
	AutoScan          bool   `json:"auto_scan"`          // Auto-scan after device discovery
}

// SNMPConfig contains SNMP settings for device interrogation.
type SNMPConfig struct {
	// Communities is a list of SNMP v1/v2c community strings to try (read-only).
	Communities []string `json:"communities"`

	// V3Credentials for SNMP v3 authentication.
	V3Credentials []SNMPv3Credential `json:"v3_credentials,omitempty"`

	// Timeout for SNMP queries.
	Timeout time.Duration `json:"timeout"`

	// Retries for failed SNMP queries.
	Retries int `json:"retries"`

	// Port for SNMP queries (default 161).
	Port int `json:"port"`

	// MaxRepetitions controls how many OID values are returned per GetBulk request.
	// Lower values reduce memory usage and network load on slow devices.
	// Default: 10. Range: 1-50.
	MaxRepetitions uint32 `json:"max_repetitions"`
}

// SNMPv3Credential contains SNMP v3 authentication credentials.
type SNMPv3Credential struct {
	// Friendly name for this credential set.
	Name string `json:"name"`
	// Security name (user).
	Username string `json:"username"`
	// AuthProtocol specifies the authentication protocol.
	// Supported values: "SHA", "SHA256", "SHA512", or "" for noAuth.
	// Note: The "MD5" value is cryptographically broken and will be removed in the next major version.
	// Use SHA256 or SHA512 instead for secure authentication.
	// "SHA", "SHA256", "SHA512", or "" for noAuth (MD5 is deprecated).
	AuthProtocol string `json:"auth_protocol"`
	// Authentication password.
	AuthPassword string `json:"auth_password"`
	// "DES", "AES", "AES192", "AES256", or "" for noPriv.
	PrivProtocol string `json:"priv_protocol"`
	// Privacy password.
	PrivPassword string `json:"priv_password"`
	// Optional SNMP context.
	ContextName string `json:"context_name"`
	// "noAuthNoPriv", "authNoPriv", "authPriv".
	SecurityLevel string `json:"security_level"`
}
