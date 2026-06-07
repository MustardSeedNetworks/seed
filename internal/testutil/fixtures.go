package testutil

import "github.com/MustardSeedNetworks/seed/internal/config"

// Test fixture constants.
const (
	// DefaultTestPort is the standard HTTP port used for test configurations.
	DefaultTestPort = 8080

	// FullScanConcurrency is the concurrency level for full discovery scans in tests.
	FullScanConcurrency = 50

	// PassiveOnlyConcurrency is the concurrency level for passive-only scans in tests.
	PassiveOnlyConcurrency = 10

	// StandardScanConcurrency is the concurrency level for standard scans in tests.
	StandardScanConcurrency = 25
)

// MinimalValidConfig returns a minimal valid configuration for testing.
// This is the most commonly used fixture for basic tests.
func MinimalValidConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(DefaultTestPort).
		WithInterface("lo").
		WithHTTPS(false).
		Build()
}

// InsecureConfig returns a configuration that triggers the setup wizard
// due to empty password hash. Used for testing setup flows.
func InsecureConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(DefaultTestPort).
		WithInterface("lo").
		WithHTTPS(false).
		WithAuth("admin", ""). // Empty password hash triggers setup wizard
		Build()
}

// FullScanConfig returns a configuration with full discovery profile
// and all features enabled. Used for integration tests.
func FullScanConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(DefaultTestPort).
		WithInterface("lo").
		WithHTTPS(false).
		WithDiscoveryConcurrency(FullScanConcurrency).
		WithDiscoveryMethods(true, true, true). // All methods enabled
		WithTCPPorts("22,80,443,445,8080").
		Build()
}

// PassiveOnlyConfig returns a configuration with passive scanning only.
func PassiveOnlyConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(DefaultTestPort).
		WithInterface("lo").
		WithHTTPS(false).
		WithDiscoveryConcurrency(PassiveOnlyConcurrency).
		WithDiscoveryMethods(false, false, false). // Passive only
		Build()
}

// StandardScanConfig returns a configuration with standard discovery settings.
func StandardScanConfig() *config.Config {
	return NewConfigBuilder().
		WithPort(DefaultTestPort).
		WithInterface("lo").
		WithHTTPS(false).
		WithDiscoveryConcurrency(StandardScanConcurrency).
		WithDiscoveryMethods(true, true, false). // ARP + ICMP
		Build()
}
