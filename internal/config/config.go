package config

// config.go holds the top-level Config struct, its mutex helpers, and the
// Clone / CopyFieldsFrom routines that callers use to safely snapshot or
// update config under lock. The constants, types, defaults, load/save,
// validation, profile import/export, and active-interface helpers all live
// in sibling config_*.go files.

import (
	"errors"
	"sync"
)

// ErrInsecureCredentials is returned when default credentials are detected.
var ErrInsecureCredentials = errors.New("insecure default credentials detected")

// Config represents the application configuration.
// All API handlers must use Lock/Unlock for writes or RLock/RUnlock for reads
// to prevent concurrent config update race conditions.
type Config struct {
	mu               sync.RWMutex           `json:"-"`       // Protects access
	Version          int                    `json:"version"` // Schema version
	Server           ServerConfig           `json:"server"`
	Interface        InterfaceConfig        `json:"interface"`
	VLAN             VLANConfig             `json:"vlan"`
	IP               IPConfig               `json:"ip"`
	Discovery        DiscoveryConfig        `json:"discovery"`
	NetworkDiscovery NetworkDiscoveryConfig `json:"networkDiscovery"`
	DNS              DNSConfig              `json:"dns"`
	HealthChecks     HealthChecksConfig     `json:"healthChecks"`
	Speedtest        SpeedtestConfig        `json:"speedtest"`
	Iperf            IperfConfig            `json:"iperf"`
	Thresholds       ThresholdsConfig       `json:"thresholds"`
	Auth             AuthConfig             `json:"auth"`
	Security         SecurityConfig         `json:"security"`
	DHCP             DHCPConfig             `json:"dhcp"`
	SNMP             SNMPConfig             `json:"snmp"`
	FABOptions       FABOptionsConfig       `json:"fabOptions"`
	DisplayOptions   DisplayOptionsConfig   `json:"displayOptions"`
	Logging          LoggingConfig          `json:"logging"`
	Database         DatabaseConfig         `json:"database"`
	Pipeline         PipelineConfig         `json:"pipeline"`
	// Profile-specific settings (not in YAML config, only stored per-profile)
	Link      LinkConfig      `json:"link,omitzero"`
	CableTest CableTestConfig `json:"cableTest,omitzero"`
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
	c.Database = temp.Database
	c.Pipeline = temp.Pipeline
}
