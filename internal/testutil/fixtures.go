package testutil

import "github.com/krisarmstrong/seed/internal/config"

// Fixtures provides pre-built configuration fixtures for common test scenarios.
var Fixtures = struct {
	MinimalConfig  func() *config.Config
	InsecureConfig func() *config.Config
	FullConfig     func() *config.Config
}{
	MinimalConfig: func() *config.Config {
		return NewConfigBuilder().
			WithPort(8080).
			WithInterface("lo").
			WithHTTPS(false).
			Build()
	},

	InsecureConfig: func() *config.Config {
		cfg := NewConfigBuilder().
			WithPort(8080).
			WithInterface("lo").
			WithHTTPS(false).
			WithAuth("admin", ""). // Empty password hash triggers setup wizard
			Build()
		return cfg
	},

	FullConfig: func() *config.Config {
		return NewConfigBuilder().
			WithPort(8080).
			WithInterface("lo").
			WithHTTPS(false).
			WithDiscoveryConcurrency(50).
			WithDiscoveryMethods(true, true, true). // All methods enabled
			WithTCPPorts("22,80,443,445,8080").
			Build()
	},
}

// MinimalValidConfig returns a minimal valid configuration for testing.
// This is the most commonly used fixture for basic tests.
func MinimalValidConfig() *config.Config {
	return Fixtures.MinimalConfig()
}

// InsecureConfig returns a configuration that triggers the setup wizard
// due to empty password hash. Used for testing setup flows.
func InsecureConfig() *config.Config {
	return Fixtures.InsecureConfig()
}

// FullScanConfig returns a configuration with full discovery profile
// and all features enabled. Used for integration tests.
func FullScanConfig() *config.Config {
	return Fixtures.FullConfig()
}

// PassiveOnlyConfig returns a configuration with passive scanning only.
func PassiveOnlyConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(8080).
		WithInterface("lo").
		WithHTTPS(false).
		WithDiscoveryConcurrency(10).
		WithDiscoveryMethods(false, false, false). // Passive only
		Build()
}

// StandardScanConfig returns a configuration with standard discovery settings.
func StandardScanConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(8080).
		WithInterface("lo").
		WithHTTPS(false).
		WithDiscoveryConcurrency(25).
		WithDiscoveryMethods(true, true, false). // ARP + ICMP
		Build()
}
