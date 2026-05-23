package config

// config_types_misc.go contains the remaining configuration types that don't
// belong to a larger domain: profile-scoped Link/CableTest, database, DNS,
// logging, FAB / display options, and setup-result.

import "time"

// LinkConfig contains interface speed/duplex settings (profile-specific).
type LinkConfig struct {
	// Mode is the combined speed/duplex (e.g., "100/full", "1000/full") or "auto".
	Mode string `json:"mode,omitempty"`
	// AvailableModes lists the speed/duplex combinations supported by the interface.
	AvailableModes []string `json:"available_modes,omitempty"`
}

// CableTestConfig contains TDR cable diagnostic settings (profile-specific).
type CableTestConfig struct {
	// Enabled controls whether the cable test card is shown.
	Enabled bool `json:"enabled"`
}

// DatabaseConfig contains SQLite database configuration.
type DatabaseConfig struct {
	// Path to the SQLite database file. Default: data/seed.db
	Path string `json:"path"`
	// RetentionDays sets how many days of historical data to keep.
	// 0 means keep forever.
	RetentionDays int `json:"retention_days"`
	// EnableWAL enables Write-Ahead Logging for better concurrency.
	// Default: true.
	EnableWAL bool `json:"enable_wal"`
	// MaxConnections sets the maximum number of database connections. Default: 10
	MaxConnections int `json:"max_connections"`
}

// DNSConfig contains DNS testing settings.
type DNSConfig struct {
	TestHostname string        `json:"test_hostname"`
	Timeout      time.Duration `json:"timeout"`
	Servers      []DNSServer   `json:"servers,omitempty"` // Additional DNS servers to test
}

// DNSServer represents a DNS server configuration.
type DNSServer struct {
	Address string `json:"address"`
	Enabled bool   `json:"enabled"`
}

// FABOptionsConfig contains FAB (Floating Action Button) settings.
type FABOptionsConfig struct {
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

// DisplayOptionsConfig contains display/UI settings.
type DisplayOptionsConfig struct {
	ShowPublicIP bool   `json:"show_public_ip"`
	UnitSystem   string `json:"unit_system"` // "sae" (feet) or "metric" (meters)
}

// LoggingConfig contains structured logging settings.
type LoggingConfig struct {
	Level      string `json:"level"`       // DEBUG, INFO, WARN, ERROR (default: INFO)
	Format     string `json:"format"`      // text or json (default: text)
	AddSource  bool   `json:"add_source"`  // Include file:line in logs
	File       string `json:"file"`        // Log file path (empty = stdout only)
	MaxSize    int    `json:"max_size"`    // Max MB per log file before rotation
	MaxBackups int    `json:"max_backups"` // Number of old files to keep
	MaxAge     int    `json:"max_age"`     // Days to keep old files
	Compress   bool   `json:"compress"`    // Compress rotated files
}

// SetupResult holds information about first-boot credential setup.
type SetupResult struct {
	IsFirstBoot     bool
	GeneratedCreds  bool
	Username        string
	Password        string // Only set if credentials were generated (display once!)
	JWTSecretStored bool
}
