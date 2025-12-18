# Development Guidelines

## Low-Level System Access

**IMPORTANT: Avoid raw syscalls whenever possible.** Prefer Go packages that provide clean abstractions:

| Operation              | Preferred Approach               | Avoid                           |
| ---------------------- | -------------------------------- | ------------------------------- |
| Network interface info | `github.com/safchain/ethtool`    | Raw ioctl syscalls              |
| Routing tables         | `golang.org/x/net/route`         | `syscall.RouteRIB` (deprecated) |
| Network links          | `github.com/vishvananda/netlink` | Manual netlink messages         |
| Process/system info    | `github.com/shirou/gopsutil`     | `/proc` parsing                 |
| MAC addresses          | `net.Interface.HardwareAddr`     | Raw ARP parsing                 |

### When a Go Package Doesn't Exist

1. Check for a pure Go implementation first
2. Consider C/C++ bindings via cgo if necessary
3. For macOS-specific functionality, Swift bindings are acceptable
4. Raw syscalls are **last resort only** when no alternative exists

The goal is maximum portability, safety, and maintainability.

## Documentation Requirements

**All code must be properly documented.**

### Go Files

```go
// Package discovery provides network device discovery via ICMP, ARP, and TCP probes.
//
// It supports multiple discovery methods and device fingerprinting for identifying
// device types, manufacturers, and operating systems.
package discovery

// DeviceScanner discovers devices on the local network using configurable probe methods.
// It maintains a cache of discovered devices and supports concurrent scanning.
type DeviceScanner struct {
    // ...
}

// Scan performs a network scan using the configured probe methods.
// It returns discovered devices or an error if the scan fails.
//
// The timeout parameter controls how long to wait for responses.
// Use 0 for the default timeout of 5 seconds.
func (s *DeviceScanner) Scan(ctx context.Context, timeout time.Duration) ([]Device, error) {
    // Implementation...
}
```

### TypeScript/React Files

```typescript
/**
 * NetworkDiscoveryCard displays discovered network devices.
 *
 * Features:
 * - Real-time device list updates via WebSocket
 * - Sorting by IP, hostname, or last seen
 * - Device type icons and status indicators
 * - Click to view device details
 *
 * @example
 * <NetworkDiscoveryCard onDeviceSelect={(d) => setSelected(d)} />
 */
export function NetworkDiscoveryCard({ onDeviceSelect }: Props) {
  // Implementation...
}
```

### Required Documentation

| Element            | Requirement                              |
| ------------------ | ---------------------------------------- |
| Package/file       | Header comment explaining purpose        |
| Exported functions | Doc comment with params, return, example |
| Exported types     | Doc comment explaining purpose           |
| Complex logic      | Inline comments explaining why           |
| Config options     | Comments explaining each option          |

## TODO Management

**Every TODO comment MUST have a corresponding GitHub issue.**

```go
// TODO(#123): Add retry logic for transient network failures.
// See: https://github.com/krisarmstrong/seed/issues/123
```

The TODO Tracker workflow runs weekly and creates issues for orphaned TODOs.

## Dead Code Prevention

**No unused code should be committed.**

Linting catches:

- Unused variables and parameters (`unparam`)
- Unused functions and types (`deadcode`, `unused`)
- Unused imports (`goimports`)
- Unused struct fields (`structcheck`)

The Dead Code Detection workflow runs weekly to find orphaned files.

## Make Targets Reference

| Target                    | Description                                             | Docker Required |
| ------------------------- | ------------------------------------------------------- | --------------- |
| `make all`                | Full verification (lint, test, security, build, docker) | Optional        |
| `make verify`             | Same as all - full verification pipeline                | Optional        |
| `make build`              | Build frontend + backend                                | No              |
| `make test`               | Run all tests                                           | No              |
| `make test-e2e`           | Run Playwright E2E browser tests                        | No              |
| `make lint`               | Run linters                                             | No              |
| `make security`           | Run all security scans                                  | No              |
| `make security-backend`   | Run gosec + govulncheck                                 | No              |
| `make security-frontend`  | Run npm audit                                           | No              |
| `make security-secrets`   | Run gitleaks secret scan                                | No              |
| `make docker-build`       | Build Docker image                                      | Yes             |
| `make docker-test`        | Build and test Docker image                             | Yes             |
| `make test-integration`   | Full systemd test on Ubuntu                             | Yes             |
| `make deploy`             | Deploy to Ubuntu server                                 | Yes             |
| `make release-check`      | Pre-release validation                                  | Optional        |
| `make pre-commit`         | Run pre-commit hooks manually                           | No              |
| `make pre-commit-install` | Install pre-commit hooks                                | No              |
| `make fmt`                | Format Go code                                          | No              |
| `make fmt-frontend`       | Format frontend code                                    | No              |
| `make clean`              | Remove build artifacts                                  | No              |

## Version & Release

Version is automatically extracted from git tags:

```bash
# Create release
git tag v1.0.0
git push --tags

# Version format
vMAJOR.MINOR.PATCH (e.g., v1.2.3)
```

Pre-release checklist: `make release-check`
