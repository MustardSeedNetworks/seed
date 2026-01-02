// Package testutil_test tests the testutil package.
package testutil_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/testutil"
)

func TestGetTestDefaults(t *testing.T) {
	t.Run("returns consistent values", func(t *testing.T) {
		defaults1 := testutil.GetTestDefaults()
		defaults2 := testutil.GetTestDefaults()

		if defaults1 != defaults2 {
			t.Error("GetTestDefaults should return same instance")
		}

		if defaults1.Auth.Username != "admin" {
			t.Errorf("expected username 'admin', got %q", defaults1.Auth.Username)
		}

		if defaults1.Auth.Password != "TestP@ssw0rd!Secure123" {
			t.Errorf("expected test password, got %q", defaults1.Auth.Password)
		}

		if defaults1.Auth.PasswordHash != "$2a$10$bl5aXjxJJUKfo7K1x2MdFuBIU2peRMPiW8L0sPkccLl2JUKLs/xb." {
			t.Errorf("expected bcrypt hash, got %q", defaults1.Auth.PasswordHash)
		}

		if defaults1.Auth.JWTSecret != "test-jwt-secret-for-testing-only-32b" {
			t.Errorf("expected JWT secret, got %q", defaults1.Auth.JWTSecret)
		}

		if defaults1.Server.HTTPS {
			t.Error("expected HTTPS to be false for testing")
		}

		if defaults1.Server.Port == 0 {
			t.Error("expected non-zero server port")
		}
	})

	t.Run("derives from DefaultConfig", func(t *testing.T) {
		defaults := testutil.GetTestDefaults()
		cfg := config.DefaultConfig()

		// Should match production defaults for non-test-specific values
		if defaults.Server.Port != cfg.Server.Port {
			t.Errorf("port mismatch: test=%d, default=%d", defaults.Server.Port, cfg.Server.Port)
		}

		if defaults.DNS.TestHostname != cfg.DNS.TestHostname {
			t.Errorf("DNS hostname mismatch: test=%s, default=%s", defaults.DNS.TestHostname, cfg.DNS.TestHostname)
		}

		if defaults.NetworkDiscovery.ARPScanWorkers != cfg.NetworkDiscovery.ARPScanWorkers {
			t.Errorf("concurrency mismatch: test=%d, default=%d",
				defaults.NetworkDiscovery.ARPScanWorkers, cfg.NetworkDiscovery.ARPScanWorkers)
		}
	})
}

func TestConfigBuilder(t *testing.T) {
	t.Run("creates valid config with defaults", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().Build()

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		if cfg.Server.HTTPS {
			t.Error("expected HTTPS disabled for testing")
		}

		if cfg.Interface.Default != "lo" {
			t.Errorf("expected loopback interface, got %q", cfg.Interface.Default)
		}

		defaults := testutil.GetTestDefaults()
		if cfg.Auth.DefaultPasswordHash != defaults.Auth.PasswordHash {
			t.Error("expected test password hash")
		}
	})

	t.Run("fluent API modifies config", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().
			WithPort(9090).
			WithInterface("eth0").
			WithHTTPS(true).
			WithDiscoveryMethods(true, true, true). // All methods enabled
			WithDiscoveryConcurrency(100).
			Build()

		if cfg.Server.Port != 9090 {
			t.Errorf("expected port 9090, got %d", cfg.Server.Port)
		}

		if cfg.Interface.Default != "eth0" {
			t.Errorf("expected eth0, got %q", cfg.Interface.Default)
		}

		if !cfg.Server.HTTPS {
			t.Error("expected HTTPS enabled")
		}

		// WithDiscoveryMethods(true, true, true) enables all methods
		if !cfg.NetworkDiscovery.Options.ARPScan {
			t.Error("expected ARP scan enabled")
		}

		if cfg.NetworkDiscovery.ARPScanWorkers != 100 {
			t.Errorf("expected concurrency 100, got %d", cfg.NetworkDiscovery.ARPScanWorkers)
		}
	})

	t.Run("WithAuth sets credentials", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().
			WithAuth("testuser", "testhash").
			Build()

		if cfg.Auth.DefaultUsername != "testuser" {
			t.Errorf("expected testuser, got %q", cfg.Auth.DefaultUsername)
		}

		if cfg.Auth.DefaultPasswordHash != "testhash" {
			t.Errorf("expected testhash, got %q", cfg.Auth.DefaultPasswordHash)
		}
	})

	t.Run("WithDiscoveryMethods configures methods", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().
			WithDiscoveryMethods(true, false, true).
			Build()

		if !cfg.NetworkDiscovery.Options.ARPScan {
			t.Error("expected ARP enabled")
		}

		if cfg.NetworkDiscovery.Options.ICMPScan {
			t.Error("expected ICMP disabled")
		}

		if !cfg.NetworkDiscovery.Options.PortScan.Enabled {
			t.Error("expected PortScan enabled")
		}
	})

	t.Run("WithTCPPorts sets ports", func(t *testing.T) {
		ports := "22,80,443"
		cfg := testutil.NewConfigBuilder().
			WithTCPPorts(ports).
			Build()

		if cfg.NetworkDiscovery.Options.PortScan.TCPPorts != ports {
			t.Errorf("expected ports %q, got %q", ports, cfg.NetworkDiscovery.Options.PortScan.TCPPorts)
		}
	})
}

func TestMustBuild(t *testing.T) {
	t.Run("succeeds with valid config", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().MustBuild(t)

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}
	})

	t.Run("validates config", func(t *testing.T) {
		// This would fail validation due to invalid port
		builder := testutil.NewConfigBuilder().WithPort(999999)

		// We can't directly test MustBuild failing because it calls t.Fatalf
		// Instead, test the Validate method
		if err := builder.Validate(); err == nil {
			t.Error("expected validation error for invalid port")
		}
	})

	t.Run("rejects empty JWT secret", func(t *testing.T) {
		builder := testutil.NewConfigBuilder().WithJWTSecret("")

		if err := builder.Validate(); err == nil {
			t.Error("expected validation error for empty JWT secret")
		}
	})

	t.Run("rejects invalid concurrency", func(t *testing.T) {
		builder := testutil.NewConfigBuilder().WithDiscoveryConcurrency(0)

		if err := builder.Validate(); err == nil {
			t.Error("expected validation error for zero concurrency")
		}
	})

	t.Run("rejects empty interface", func(t *testing.T) {
		builder := testutil.NewConfigBuilder().WithInterface("")

		if err := builder.Validate(); err == nil {
			t.Error("expected validation error for empty interface")
		}
	})
}

func TestFixtures(t *testing.T) {
	t.Run("MinimalConfig is valid", func(t *testing.T) {
		cfg := testutil.Fixtures.MinimalConfig()

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		if err := cfg.Validate(); err != nil {
			t.Errorf("MinimalConfig should be valid: %v", err)
		}

		if cfg.Server.Port != 8080 {
			t.Errorf("expected port 8080, got %d", cfg.Server.Port)
		}

		if cfg.Interface.Default != "lo" {
			t.Errorf("expected loopback, got %q", cfg.Interface.Default)
		}
	})

	t.Run("InsecureConfig has empty password", func(t *testing.T) {
		cfg := testutil.Fixtures.InsecureConfig()

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		if cfg.Auth.DefaultPasswordHash != "" {
			t.Error("expected empty password hash to trigger setup")
		}
	})

	t.Run("FullConfig has all features", func(t *testing.T) {
		cfg := testutil.Fixtures.FullConfig()

		if cfg == nil {
			t.Fatal("expected non-nil config")
		}

		if !cfg.NetworkDiscovery.Options.ARPScan || !cfg.NetworkDiscovery.Options.ICMPScan ||
			!cfg.NetworkDiscovery.Options.PortScan.Enabled {
			t.Error("expected all discovery methods enabled")
		}

		if cfg.NetworkDiscovery.Options.PortScan.TCPPorts == "" {
			t.Error("expected TCP ports configured")
		}
	})

	t.Run("helper functions return same as fixtures", func(t *testing.T) {
		if testutil.MinimalValidConfig() == nil {
			t.Error("MinimalValidConfig should not be nil")
		}

		if testutil.InsecureConfig() == nil {
			t.Error("InsecureConfig should not be nil")
		}

		if testutil.FullScanConfig() == nil {
			t.Error("FullScanConfig should not be nil")
		}
	})

	t.Run("PassiveOnlyConfig has passive only", func(t *testing.T) {
		cfg := testutil.PassiveOnlyConfig()

		if cfg.NetworkDiscovery.Options.ARPScan || cfg.NetworkDiscovery.Options.ICMPScan ||
			cfg.NetworkDiscovery.Options.PortScan.Enabled {
			t.Error("expected passive only for passive scan")
		}
	})

	t.Run("StandardScanConfig has ARP and ICMP", func(t *testing.T) {
		cfg := testutil.StandardScanConfig()

		if !cfg.NetworkDiscovery.Options.ARPScan || !cfg.NetworkDiscovery.Options.ICMPScan ||
			cfg.NetworkDiscovery.Options.PortScan.Enabled {
			t.Error("expected ARP + ICMP for standard scan")
		}
	})
}
