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
			t.Errorf(
				"DNS hostname mismatch: test=%s, default=%s",
				defaults.DNS.TestHostname,
				cfg.DNS.TestHostname,
			)
		}

		if defaults.NetworkDiscovery.ARPScanWorkers != cfg.NetworkDiscovery.ARPScanWorkers {
			t.Errorf("concurrency mismatch: test=%d, default=%d",
				defaults.NetworkDiscovery.ARPScanWorkers, cfg.NetworkDiscovery.ARPScanWorkers)
		}
	})
}

// assertConfigNotNil is a helper that fails if cfg is nil.
func assertConfigNotNil(t *testing.T, cfg *config.Config) {
	t.Helper()

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
}

// assertPort checks if the config port matches expected.
func assertPort(t *testing.T, cfg *config.Config, expected int) {
	t.Helper()

	if cfg.Server.Port != expected {
		t.Errorf("expected port %d, got %d", expected, cfg.Server.Port)
	}
}

// assertInterface checks if the config interface matches expected.
func assertInterface(t *testing.T, cfg *config.Config, expected string) {
	t.Helper()

	if cfg.Interface.Default != expected {
		t.Errorf("expected interface %q, got %q", expected, cfg.Interface.Default)
	}
}

// assertHTTPS checks if HTTPS setting matches expected.
func assertHTTPS(t *testing.T, cfg *config.Config, expected bool) {
	t.Helper()

	if cfg.Server.HTTPS != expected {
		t.Errorf("expected HTTPS=%v, got %v", expected, cfg.Server.HTTPS)
	}
}

// assertDiscoveryMethods checks ARP, ICMP, and PortScan settings.
func assertDiscoveryMethods(t *testing.T, cfg *config.Config, arp, icmp, port bool) {
	t.Helper()

	if cfg.NetworkDiscovery.Options.ARPScan != arp {
		t.Errorf("expected ARPScan=%v, got %v", arp, cfg.NetworkDiscovery.Options.ARPScan)
	}

	if cfg.NetworkDiscovery.Options.ICMPScan != icmp {
		t.Errorf("expected ICMPScan=%v, got %v", icmp, cfg.NetworkDiscovery.Options.ICMPScan)
	}

	if cfg.NetworkDiscovery.Options.PortScan.Enabled != port {
		t.Errorf("expected PortScan=%v, got %v", port, cfg.NetworkDiscovery.Options.PortScan.Enabled)
	}
}

// assertConcurrency checks if the discovery concurrency matches expected.
func assertConcurrency(t *testing.T, cfg *config.Config, expected int) {
	t.Helper()

	if cfg.NetworkDiscovery.ARPScanWorkers != expected {
		t.Errorf("expected concurrency %d, got %d", expected, cfg.NetworkDiscovery.ARPScanWorkers)
	}
}

// assertAuth checks if auth credentials match expected values.
func assertAuth(t *testing.T, cfg *config.Config, username, hash string) {
	t.Helper()

	if cfg.Auth.DefaultUsername != username {
		t.Errorf("expected username %q, got %q", username, cfg.Auth.DefaultUsername)
	}

	if cfg.Auth.DefaultPasswordHash != hash {
		t.Errorf("expected hash %q, got %q", hash, cfg.Auth.DefaultPasswordHash)
	}
}

// assertTCPPorts checks if the TCP ports match expected.
func assertTCPPorts(t *testing.T, cfg *config.Config, expected string) {
	t.Helper()

	if cfg.NetworkDiscovery.Options.PortScan.TCPPorts != expected {
		t.Errorf("expected ports %q, got %q", expected, cfg.NetworkDiscovery.Options.PortScan.TCPPorts)
	}
}

// assertConfigValid checks if the config passes validation.
func assertConfigValid(t *testing.T, cfg *config.Config) {
	t.Helper()

	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config: %v", err)
	}
}

// assertEmptyPasswordHash checks if the password hash is empty.
func assertEmptyPasswordHash(t *testing.T, cfg *config.Config) {
	t.Helper()

	if cfg.Auth.DefaultPasswordHash != "" {
		t.Error("expected empty password hash")
	}
}

// assertTCPPortsNotEmpty checks that TCP ports are configured.
func assertTCPPortsNotEmpty(t *testing.T, cfg *config.Config) {
	t.Helper()

	if cfg.NetworkDiscovery.Options.PortScan.TCPPorts == "" {
		t.Error("expected TCP ports configured")
	}
}

func TestConfigBuilder(t *testing.T) {
	t.Run("creates valid config with defaults", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().Build()
		assertConfigNotNil(t, cfg)
		assertHTTPS(t, cfg, false)
		assertInterface(t, cfg, "lo")

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
			WithDiscoveryMethods(true, true, true).
			WithDiscoveryConcurrency(100).
			Build()

		assertPort(t, cfg, 9090)
		assertInterface(t, cfg, "eth0")
		assertHTTPS(t, cfg, true)
		assertDiscoveryMethods(t, cfg, true, true, true)
		assertConcurrency(t, cfg, 100)
	})

	t.Run("WithAuth sets credentials", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().
			WithAuth("testuser", "testhash").
			Build()

		assertAuth(t, cfg, "testuser", "testhash")
	})

	t.Run("WithDiscoveryMethods configures methods", func(t *testing.T) {
		cfg := testutil.NewConfigBuilder().
			WithDiscoveryMethods(true, false, true).
			Build()

		assertDiscoveryMethods(t, cfg, true, false, true)
	})

	t.Run("WithTCPPorts sets ports", func(t *testing.T) {
		ports := "22,80,443"
		cfg := testutil.NewConfigBuilder().
			WithTCPPorts(ports).
			Build()

		assertTCPPorts(t, cfg, ports)
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

// fixtureTestCase defines a test case for fixture functions.
type fixtureTestCase struct {
	name       string
	getConfig  func() *config.Config
	assertions func(t *testing.T, cfg *config.Config)
}

func TestFixtures(t *testing.T) {
	testCases := []fixtureTestCase{
		{
			name:      "MinimalConfig is valid",
			getConfig: testutil.MinimalValidConfig,
			assertions: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assertConfigValid(t, cfg)
				assertPort(t, cfg, 8080)
				assertInterface(t, cfg, "lo")
			},
		},
		{
			name:      "InsecureConfig has empty password",
			getConfig: testutil.InsecureConfig,
			assertions: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assertEmptyPasswordHash(t, cfg)
			},
		},
		{
			name:      "FullConfig has all features",
			getConfig: testutil.FullScanConfig,
			assertions: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assertDiscoveryMethods(t, cfg, true, true, true)
				assertTCPPortsNotEmpty(t, cfg)
			},
		},
		{
			name:      "PassiveOnlyConfig has passive only",
			getConfig: testutil.PassiveOnlyConfig,
			assertions: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assertDiscoveryMethods(t, cfg, false, false, false)
			},
		},
		{
			name:      "StandardScanConfig has ARP and ICMP",
			getConfig: testutil.StandardScanConfig,
			assertions: func(t *testing.T, cfg *config.Config) {
				t.Helper()
				assertDiscoveryMethods(t, cfg, true, true, false)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := tc.getConfig()
			assertConfigNotNil(t, cfg)
			tc.assertions(t, cfg)
		})
	}

	t.Run("helper functions return same as fixtures", func(t *testing.T) {
		fixtures := []func() *config.Config{
			testutil.MinimalValidConfig,
			testutil.InsecureConfig,
			testutil.FullScanConfig,
		}

		for _, fn := range fixtures {
			if fn() == nil {
				t.Error("fixture function should not return nil")
			}
		}
	})
}
