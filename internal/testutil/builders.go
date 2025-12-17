package testutil

import (
	"fmt"
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
)

// ConfigBuilder creates test configurations with sensible defaults.
type ConfigBuilder struct {
	cfg *config.Config
}

// NewConfigBuilder creates a new ConfigBuilder with test-friendly defaults.
func NewConfigBuilder() *ConfigBuilder {
	defaults := GetTestDefaults()
	cfg := config.DefaultConfig()

	// Apply test-friendly overrides
	cfg.Server.HTTPS = false // Easier for testing
	cfg.Server.Port = defaults.Server.Port
	cfg.Interface.Default = "lo" // Loopback for tests
	cfg.Auth.DefaultPasswordHash = defaults.Auth.PasswordHash
	cfg.Auth.JWTSecret = defaults.Auth.JWTSecret

	return &ConfigBuilder{cfg: cfg}
}

// WithPort sets the server port.
func (b *ConfigBuilder) WithPort(port int) *ConfigBuilder {
	b.cfg.Server.Port = port
	return b
}

// WithAuth sets the authentication credentials.
func (b *ConfigBuilder) WithAuth(username, passwordHash string) *ConfigBuilder {
	b.cfg.Auth.DefaultUsername = username
	b.cfg.Auth.DefaultPasswordHash = passwordHash
	return b
}

// WithInterface sets the default network interface.
func (b *ConfigBuilder) WithInterface(iface string) *ConfigBuilder {
	b.cfg.Interface.Default = iface
	return b
}

// WithDiscoveryProfile sets the discovery profile.
func (b *ConfigBuilder) WithDiscoveryProfile(profile config.DiscoveryProfile) *ConfigBuilder {
	b.cfg.NetworkDiscovery.Profile = profile
	return b
}

// WithHTTPS enables or disables HTTPS.
func (b *ConfigBuilder) WithHTTPS(enabled bool) *ConfigBuilder {
	b.cfg.Server.HTTPS = enabled
	return b
}

// WithJWTSecret sets the JWT secret.
func (b *ConfigBuilder) WithJWTSecret(secret string) *ConfigBuilder {
	b.cfg.Auth.JWTSecret = secret
	return b
}

// WithDNSTestHostname sets the DNS test hostname.
func (b *ConfigBuilder) WithDNSTestHostname(hostname string) *ConfigBuilder {
	b.cfg.DNS.TestHostname = hostname
	return b
}

// WithDiscoveryConcurrency sets the network discovery concurrency (ARP scan workers).
func (b *ConfigBuilder) WithDiscoveryConcurrency(concurrency int) *ConfigBuilder {
	b.cfg.NetworkDiscovery.ARPScanWorkers = concurrency
	return b
}

// WithDiscoveryMethods configures which discovery methods are enabled.
func (b *ConfigBuilder) WithDiscoveryMethods(arp, icmp, portScan bool) *ConfigBuilder {
	b.cfg.NetworkDiscovery.CustomOptions.ARPScan = arp
	b.cfg.NetworkDiscovery.CustomOptions.ICMPScan = icmp
	b.cfg.NetworkDiscovery.CustomOptions.PortScan.Enabled = portScan
	return b
}

// WithTCPPorts sets the TCP ports to probe during discovery.
func (b *ConfigBuilder) WithTCPPorts(ports string) *ConfigBuilder {
	b.cfg.NetworkDiscovery.CustomOptions.PortScan.TCPPorts = ports
	return b
}

// Build returns the configured Config without validation.
func (b *ConfigBuilder) Build() *config.Config {
	return b.cfg
}

// MustBuild validates the configuration and returns it, failing the test if invalid.
func (b *ConfigBuilder) MustBuild(t *testing.T) *config.Config {
	t.Helper()

	if err := b.cfg.Validate(); err != nil {
		t.Fatalf("ConfigBuilder.MustBuild: invalid configuration: %v", err)
	}

	return b.cfg
}

// Validate validates the configuration and returns any errors.
func (b *ConfigBuilder) Validate() error {
	if b.cfg.Server.Port < 1 || b.cfg.Server.Port > 65535 {
		return fmt.Errorf("invalid port: %d", b.cfg.Server.Port)
	}

	if b.cfg.Auth.JWTSecret == "" {
		return fmt.Errorf("JWT secret cannot be empty")
	}

	if b.cfg.NetworkDiscovery.ARPScanWorkers < 1 {
		return fmt.Errorf("discovery concurrency must be at least 1")
	}

	if b.cfg.Interface.Default == "" {
		return fmt.Errorf("default interface cannot be empty")
	}

	return b.cfg.Validate()
}
