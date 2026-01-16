package testutil

import (
	"sync"
	"time"

	"github.com/krisarmstrong/seed/internal/config"
)

// TestDefaults provides centralized test defaults.
type TestDefaults struct {
	Auth             AuthDefaults
	Server           ServerDefaults
	DNS              DNSDefaults
	Discovery        DiscoveryDefaults
	NetworkDiscovery NetworkDiscoveryDefaults
}

// AuthDefaults contains authentication-related test defaults.
type AuthDefaults struct {
	Username     string        // Test username
	Password     string        // Plaintext test password
	PasswordHash string        // Pre-computed bcrypt hash
	JWTSecret    string        // Test JWT secret
	Timeout      time.Duration // Auth timeout
}

// ServerDefaults contains server-related test defaults.
type ServerDefaults struct {
	Port  int  // Server port
	HTTPS bool // HTTPS enabled
}

// DNSDefaults contains DNS-related test defaults.
type DNSDefaults struct {
	TestHostname string        // Test hostname
	Timeout      time.Duration // DNS timeout
}

// DiscoveryDefaults contains discovery-related test defaults.
type DiscoveryDefaults struct {
	Protocol string        // Discovery protocol
	Timeout  time.Duration // Discovery timeout
}

// NetworkDiscoveryDefaults contains network discovery-related test defaults.
type NetworkDiscoveryDefaults struct {
	ARPScanWorkers int           // Concurrent workers
	PingTimeout    time.Duration // Ping timeout
	ScanTimeout    time.Duration // Scan timeout
	AutoScan       bool          // Auto-scan on startup
}

// Test defaults accessor functions use closure-encapsulated state for thread-safe singleton access.
// getTestDefaults returns the cached test defaults instance.
// setTestDefaults sets the cached test defaults instance.
// getTestDefaultsOnce returns the [sync.Once] for lazy initialization.
//
//nolint:gochecknoglobals // Intentional thread-safe singleton using closure pattern
var (
	getTestDefaults, setTestDefaults, getTestDefaultsOnce = func() (
		func() *TestDefaults,
		func(*TestDefaults),
		func() *sync.Once,
	) {
		var (
			mu       sync.RWMutex
			defaults *TestDefaults
			once     sync.Once
		)

		return func() *TestDefaults {
				mu.RLock()
				defer mu.RUnlock()
				return defaults
			}, func(d *TestDefaults) {
				mu.Lock()
				defer mu.Unlock()
				defaults = d
			}, func() *sync.Once {
				return &once
			}
	}()
)

// GetTestDefaults returns singleton test defaults derived from config.DefaultConfig().
// This function uses lazy initialization to compute expensive values only once.
func GetTestDefaults() *TestDefaults {
	getTestDefaultsOnce().Do(func() {
		cfg := config.DefaultConfig()

		setTestDefaults(&TestDefaults{
			Auth: AuthDefaults{
				Username:     "admin",
				Password:     "TestP@ssw0rd!Secure123",
				PasswordHash: "$2a$10$bl5aXjxJJUKfo7K1x2MdFuBIU2peRMPiW8L0sPkccLl2JUKLs/xb.",
				JWTSecret:    "test-jwt-secret-for-testing-only-32b",
				Timeout:      cfg.Auth.SessionTimeout,
			},
			Server: ServerDefaults{
				Port:  cfg.Server.Port,
				HTTPS: false, // Easier for testing
			},
			DNS: DNSDefaults{
				TestHostname: cfg.DNS.TestHostname,
				Timeout:      cfg.DNS.Timeout,
			},
			Discovery: DiscoveryDefaults{
				Protocol: cfg.Discovery.Protocol,
				Timeout:  cfg.Discovery.Timeout,
			},
			NetworkDiscovery: NetworkDiscoveryDefaults{
				ARPScanWorkers: cfg.NetworkDiscovery.ARPScanWorkers,
				PingTimeout:    cfg.NetworkDiscovery.PingTimeout,
				ScanTimeout:    cfg.NetworkDiscovery.ScanTimeout,
				AutoScan:       cfg.NetworkDiscovery.AutoScan,
			},
		})
	})

	return getTestDefaults()
}
