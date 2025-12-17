# testutil - Centralized Test Utilities

The `testutil` package provides centralized test utilities for the Seed project, ensuring consistent test configuration across all test files.

## Features

- **TestDefaults**: Singleton test defaults derived from `config.DefaultConfig()`
- **ConfigBuilder**: Fluent builder for creating test configurations
- **Fixtures**: Pre-built configuration fixtures for common test scenarios
- **High Test Coverage**: 92.6% coverage ensuring reliability

## Quick Start

### Using Test Defaults

```go
import "github.com/krisarmstrong/seed/internal/testutil"

func TestMyFunction(t *testing.T) {
    defaults := testutil.GetTestDefaults()

    // Access pre-computed values
    username := defaults.Auth.Username           // "admin"
    password := defaults.Auth.Password           // "TestP@ssw0rd!Secure123"
    passwordHash := defaults.Auth.PasswordHash   // Pre-computed bcrypt hash
    jwtSecret := defaults.Auth.JWTSecret         // Test JWT secret
}
```

### Building Custom Configurations

```go
func TestWithCustomConfig(t *testing.T) {
    // Fluent builder with test-friendly defaults
    cfg := testutil.NewConfigBuilder().
        WithPort(8080).
        WithInterface("eth0").
        WithHTTPS(false).
        WithDiscoveryProfile(config.ProfileFullScan).
        WithDiscoveryConcurrency(100).
        Build()

    // Use config in tests
    // ...
}
```

### Using MustBuild for Validation

```go
func TestRequiresValidConfig(t *testing.T) {
    // MustBuild validates and fails the test if invalid
    cfg := testutil.NewConfigBuilder().
        WithPort(8080).
        MustBuild(t) // Calls t.Fatalf if validation fails

    // Config is guaranteed valid here
}
```

### Using Pre-built Fixtures

```go
func TestMinimalConfig(t *testing.T) {
    // Minimal valid configuration
    cfg := testutil.MinimalValidConfig()
}

func TestInsecureConfig(t *testing.T) {
    // Empty password hash (triggers setup wizard)
    cfg := testutil.InsecureConfig()
}

func TestFullScan(t *testing.T) {
    // Full discovery profile with all features
    cfg := testutil.FullScanConfig()
}

func TestStealthScan(t *testing.T) {
    // Passive discovery only
    cfg := testutil.StealthScanConfig()
}

func TestStandardScan(t *testing.T) {
    // Standard ARP + ICMP discovery
    cfg := testutil.StandardScanConfig()
}
```

## ConfigBuilder Methods

### Server Configuration

- `WithPort(port int)` - Set server port
- `WithHTTPS(enabled bool)` - Enable/disable HTTPS

### Authentication

- `WithAuth(username, passwordHash string)` - Set credentials
- `WithJWTSecret(secret string)` - Set JWT secret

### Network Interface

- `WithInterface(iface string)` - Set default interface

### Discovery

- `WithDiscoveryProfile(profile config.DiscoveryProfile)` - Set profile (Stealth, Standard, FullScan, Custom)
- `WithDiscoveryConcurrency(workers int)` - Set ARP scan workers
- `WithDiscoveryMethods(arp, icmp, portScan bool)` - Enable/disable methods
- `WithTCPPorts(ports string)` - Set TCP ports (e.g., "22,80,443")

### DNS

- `WithDNSTestHostname(hostname string)` - Set test hostname

### Building

- `Build()` - Return config without validation
- `MustBuild(t *testing.T)` - Return config with validation (fails test if invalid)
- `Validate()` - Validate config and return errors

## Test Defaults

The `TestDefaults` struct provides consistent values across all tests:

```go
type TestDefaults struct {
    Auth             AuthDefaults
    Server           ServerDefaults
    DNS              DNSDefaults
    Discovery        DiscoveryDefaults
    NetworkDiscovery NetworkDiscoveryDefaults
}
```

### AuthDefaults

- `Username`: "admin"
- `Password`: "TestP@ssw0rd!Secure123"
- `PasswordHash`: Pre-computed bcrypt hash
- `JWTSecret`: "test-jwt-secret-for-testing-only-32b"
- `Timeout`: Derived from DefaultConfig()

### ServerDefaults

- `Port`: 8443 (from DefaultConfig)
- `HTTPS`: false (easier for testing)

### DNSDefaults

- `TestHostname`: "google.com" (from DefaultConfig)
- `Timeout`: 5 seconds (from DefaultConfig)

### DiscoveryDefaults

- `Protocol`: "auto" (from DefaultConfig)
- `Timeout`: 30 seconds (from DefaultConfig)

### NetworkDiscoveryDefaults

- `Profile`: ProfileStandard (from DefaultConfig)
- `ARPScanWorkers`: 50 (from DefaultConfig)
- `PingTimeout`: 500ms (from DefaultConfig)
- `ScanTimeout`: 30s (from DefaultConfig)
- `AutoScan`: true (from DefaultConfig)

## Design Principles

1. **DRY (Don't Repeat Yourself)**: Centralized defaults prevent duplication
2. **Single Source of Truth**: Derived from `config.DefaultConfig()`
3. **Test-Friendly**: Sensible defaults for testing (HTTPS disabled, loopback interface)
4. **Lazy Initialization**: Expensive operations (bcrypt) computed once
5. **Type Safety**: Compile-time checking via typed structs
6. **Fluent API**: Chainable builder methods for readability

## Migration Guide

### Before (Duplicated Config Creation)

```go
func TestOldWay(t *testing.T) {
    cfg := &config.Config{
        Server: config.ServerConfig{
            Port: 8080,
            HTTPS: false,
        },
        Interface: config.InterfaceConfig{
            Default: "lo",
        },
        Auth: config.AuthConfig{
            DefaultUsername: "admin",
            DefaultPasswordHash: "$2a$10$...",
            JWTSecret: "test-secret",
        },
        // ... dozens more lines
    }
}
```

### After (Using testutil)

```go
func TestNewWay(t *testing.T) {
    cfg := testutil.MinimalValidConfig()
    // Or customize:
    cfg = testutil.NewConfigBuilder().
        WithPort(8080).
        Build()
}
```

## Best Practices

1. **Use Fixtures for Common Cases**: Prefer `MinimalValidConfig()` over custom builders
2. **Use MustBuild in Tests**: Catch config errors early with validation
3. **Keep Builders Readable**: Chain methods logically (server → auth → discovery)
4. **Don't Hardcode Values**: Use `GetTestDefaults()` for shared values
5. **Document Custom Configs**: Add comments explaining why custom config is needed

## Examples

### Testing Auth with Known Credentials

```go
func TestAuthWithDefaults(t *testing.T) {
    defaults := testutil.GetTestDefaults()

    // Use pre-computed hash
    cfg := testutil.NewConfigBuilder().
        WithAuth(defaults.Auth.Username, defaults.Auth.PasswordHash).
        MustBuild(t)

    // Test authentication
    // The password is defaults.Auth.Password
}
```

### Testing Discovery Profiles

```go
func TestDiscoveryProfiles(t *testing.T) {
    tests := []struct {
        name   string
        config *config.Config
    }{
        {"Stealth", testutil.StealthScanConfig()},
        {"Standard", testutil.StandardScanConfig()},
        {"Full", testutil.FullScanConfig()},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test each profile
        })
    }
}
```

### Testing Setup Wizard

```go
func TestSetupWizard(t *testing.T) {
    // Use insecure config (empty password hash)
    cfg := testutil.InsecureConfig()

    if cfg.Auth.DefaultPasswordHash != "" {
        t.Error("expected empty password to trigger setup")
    }
}
```

## Maintenance

- **Sync with DefaultConfig**: Update when `config.DefaultConfig()` changes
- **Add New Builders**: Add methods as new config options are added
- **Update Fixtures**: Keep fixtures relevant to common test scenarios
- **Maintain Coverage**: Keep test coverage above 80%

## Testing the testutil Package

```bash
# Run tests
go test ./internal/testutil/... -v

# Check coverage
go test ./internal/testutil/... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run with race detector
go test ./internal/testutil/... -race
```
