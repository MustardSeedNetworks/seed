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
	Profile        config.DiscoveryProfile // Discovery profile
	ARPScanWorkers int                     // Concurrent workers
	PingTimeout    time.Duration           // Ping timeout
	ScanTimeout    time.Duration           // Scan timeout
	AutoScan       bool                    // Auto-scan on startup
}

var (
	testDefaults     *TestDefaults
	testDefaultsOnce sync.Once
)

// GetTestDefaults returns singleton test defaults derived from config.DefaultConfig().
// This function uses lazy initialization to compute expensive values only once.
func GetTestDefaults() *TestDefaults {
	testDefaultsOnce.Do(func() {
		cfg := config.DefaultConfig()

		testDefaults = &TestDefaults{
			Auth: AuthDefaults{
				Username:     "admin",
				Password:     "TestP@ssw0rd!Secure123",
				PasswordHash: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZRGdjGj/n3.SRpKh4xPIHxwFxdlHK",
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
				Profile:        cfg.NetworkDiscovery.Profile,
				ARPScanWorkers: cfg.NetworkDiscovery.ARPScanWorkers,
				PingTimeout:    cfg.NetworkDiscovery.PingTimeout,
				ScanTimeout:    cfg.NetworkDiscovery.ScanTimeout,
				AutoScan:       cfg.NetworkDiscovery.AutoScan,
			},
		}
	})

	return testDefaults
}
