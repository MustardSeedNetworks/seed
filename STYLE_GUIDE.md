# The Seed Style Guide

This document defines the official coding standards and naming conventions for the The Seed project.
All contributors should follow these guidelines to ensure consistency and maintainability.

## Table of Contents

- [File Naming](#file-naming)
- [Variable & Function Naming](#variable--function-naming)
- [Go Code Standards](#go-code-standards)
- [TypeScript/React Code Standards](#typescriptreact-code-standards)
- [API Design](#api-design)
- [Documentation Standards](#documentation-standards)
- [Git Commit Messages](#git-commit-messages)

## File Naming

### Go Files

Use `snake_case` for all Go files:

```
internal/api/handlers.go
internal/api/rate_limit.go
internal/discovery/arp_linux.go
internal/network/interfaces_darwin.go
```

Platform-specific files use `_linux.go` or `_darwin.go` suffixes.

### React Components

Use `PascalCase` for component files:

```
web/src/components/cards/LinkCard.tsx
web/src/components/ui/StatusBadge.tsx
web/src/components/settings/SettingsDrawer.tsx
```

### React Hooks

Use `camelCase` with `use` prefix:

```
web/src/hooks/useWebSocket.ts
web/src/hooks/useApi.ts
web/src/hooks/useTheme.ts
```

### Test Files

- **Go**: `*_test.go` in the same directory
- **React**: `*.test.ts` or `*.test.tsx` co-located with source

```
internal/api/handlers_test.go
web/src/hooks/useWebSocket.test.ts
web/src/components/ui/Card.test.tsx
```

### Story Files (Storybook)

Use `*.stories.tsx` co-located with components:

```
web/src/components/ui/Card.stories.tsx
web/src/components/ui/StatusBadge.stories.tsx
```

## Variable & Function Naming

### Go

| Scope    | Convention   | Example                           |
| -------- | ------------ | --------------------------------- |
| Private  | `camelCase`  | `func parseConfig()`, `var count` |
| Public   | `PascalCase` | `func NewServer()`, `type Config` |
| Constant | `PascalCase` | `const DefaultPort = 8443`        |
| Package  | `lowercase`  | `package discovery`               |

```go
// Good
type Server struct {
    port     int    // private
    Address  string // public
}

func NewServer(port int) *Server { ... }
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) { ... }

// Bad
type server struct { ... }  // Should be Server if used externally
func Handle_Request() { ... }  // No underscores
```

### TypeScript

| Element     | Convention   | Example                        |
| ----------- | ------------ | ------------------------------ |
| Variables   | `camelCase`  | `const deviceCount = 5`        |
| Functions   | `camelCase`  | `function fetchData() { ... }` |
| Components  | `PascalCase` | `function LinkCard() { ... }`  |
| Hooks       | `useCamel`   | `function useWebSocket() { }`  |
| Types       | `PascalCase` | `interface DeviceInfo { ... }` |
| Constants   | `UPPER_CASE` | `const API_BASE_URL = '...'`   |
| Enums       | `PascalCase` | `enum Status { Active, ... }`  |
| CSS classes | `kebab-case` | `className="card-header"`      |

```typescript
// Good
interface NetworkDevice {
  ipAddress: string;
  macAddress: string;
  isOnline: boolean;
}

function DeviceCard({ device }: { device: NetworkDevice }) {
  const [isExpanded, setIsExpanded] = useState(false);
  return <div className="device-card">...</div>;
}

// Bad
interface network_device { ... }  // Use PascalCase
function device_card() { ... }    // Components use PascalCase
const IsExpanded = true;          // Variables use camelCase
```

## Go Code Standards

### Error Handling

Always handle errors explicitly:

```go
// Good
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
result, _ := doSomething()  // Ignored error
```

### Documentation

Document all exported functions:

```go
// NewServer creates a new HTTP server with the given configuration.
// It returns an error if the port is invalid or already in use.
func NewServer(cfg *Config) (*Server, error) {
    // ...
}
```

### Struct Tags

Use consistent struct tag ordering: `json`, `yaml`, then others:

```go
type Config struct {
    Port     int    `json:"port" yaml:"port"`
    Hostname string `json:"hostname" yaml:"hostname" validate:"required"`
}
```

### Import Grouping

Group imports in this order: stdlib, external, internal:

```go
import (
    "context"
    "fmt"
    "net/http"

    "github.com/gorilla/websocket"
    "go.uber.org/zap"

    "github.com/krisarmstrong/seed/internal/config"
    "github.com/krisarmstrong/seed/internal/discovery"
)
```

## TypeScript/React Code Standards

### Component Structure

Organize components in this order:

```typescript
// 1. Imports
import { useState, useEffect } from 'react';
import { cn } from '../styles/theme';

// 2. Types/Interfaces
interface CardProps {
  title: string;
  status: 'success' | 'warning' | 'error';
  children: React.ReactNode;
}

// 3. Component
export function Card({ title, status, children }: CardProps) {
  // 3a. Hooks
  const [isExpanded, setIsExpanded] = useState(false);

  // 3b. Derived state / memos
  const statusColor = status === 'success' ? 'green' : 'red';

  // 3c. Effects
  useEffect(() => {
    // ...
  }, []);

  // 3d. Event handlers
  const handleClick = () => setIsExpanded(!isExpanded);

  // 3e. Render
  return (
    <div className="card">
      <h2>{title}</h2>
      {children}
    </div>
  );
}
```

### Design System Usage

Always use design system tokens instead of hardcoded values:

```typescript
// Good - uses design tokens
import { buttonClass, cn, icon } from '../styles/theme';

<button className={buttonClass('primary', 'md')}>Save</button>
<span className="text-text-primary">Label</span>
<Settings className={icon.size.sm} />

// Bad - hardcoded values
<button className="px-4 py-2 bg-blue-500 text-white">Save</button>
<span className="text-gray-900 dark:text-white">Label</span>
<Settings className="w-4 h-4" />
```

See [web/THEMING.md](web/THEMING.md) for complete design system documentation.

### Prop Types

Use TypeScript interfaces for props, not inline types:

```typescript
// Good
interface ButtonProps {
  variant: 'primary' | 'secondary';
  onClick: () => void;
  children: React.ReactNode;
}

function Button({ variant, onClick, children }: ButtonProps) { ... }

// Acceptable for simple cases
function Icon({ size = 'md' }: { size?: 'sm' | 'md' | 'lg' }) { ... }

// Bad - complex inline types
function Button({ variant, onClick, children }: {
  variant: 'primary' | 'secondary' | 'danger';
  onClick: () => void;
  disabled?: boolean;
  loading?: boolean;
  children: React.ReactNode;
}) { ... }
```

## API Design

### RESTful Endpoints

Use consistent naming for REST endpoints:

| Pattern                     | Method | Description              |
| --------------------------- | ------ | ------------------------ |
| `/api/devices`              | GET    | List all devices         |
| `/api/devices`              | POST   | Create a device          |
| `/api/devices/:id`          | GET    | Get a specific device    |
| `/api/devices/:id`          | PUT    | Update a device          |
| `/api/devices/:id`          | DELETE | Delete a device          |
| `/api/devices/:id/scan`     | POST   | Trigger action on device |
| `/api/devices/:id/settings` | GET    | Get sub-resource         |

### Naming Conventions

- Use plural nouns for collections: `/api/devices` not `/api/device`
- Use kebab-case for multi-word resources: `/api/public-ip` not `/api/publicIP`
- Nest related resources: `/api/devices/:id/ports`
- Use query params for filtering: `/api/devices?status=online`

### Response Format

Use consistent JSON response structure:

```json
// Success response
{
  "data": { ... },
  "meta": {
    "timestamp": "2024-01-15T10:30:00Z"
  }
}

// Error response
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid IP address format",
    "details": { "field": "ipAddress" }
  }
}

// List response
{
  "data": [ ... ],
  "meta": {
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

## Documentation Standards

### Go Documentation

```go
// Package discovery provides network device discovery functionality
// including ARP scanning, ICMP probing, and protocol detection (LLDP, CDP).
package discovery

// Scanner discovers devices on the local network.
// It supports multiple discovery methods and can be configured
// with custom timeouts and worker counts.
type Scanner struct {
    // ...
}

// Scan performs a network scan on the specified subnet.
// It returns discovered devices and any errors encountered.
//
// Parameters:
//   - ctx: Context for cancellation
//   - subnet: CIDR notation (e.g., "192.168.1.0/24")
//
// Returns:
//   - []Device: List of discovered devices
//   - error: nil on success, error on failure
func (s *Scanner) Scan(ctx context.Context, subnet string) ([]Device, error) {
    // ...
}
```

### TypeScript/JSDoc Documentation

```typescript
/**
 * Fetches network devices from the API.
 *
 * @param options - Fetch options
 * @param options.subnet - Subnet to scan (e.g., "192.168.1.0/24")
 * @param options.timeout - Request timeout in milliseconds
 * @returns Promise resolving to array of devices
 * @throws {ApiError} When the request fails
 *
 * @example
 * const devices = await fetchDevices({ subnet: "192.168.1.0/24" });
 */
async function fetchDevices(options: FetchOptions): Promise<Device[]> {
  // ...
}

/**
 * Network device discovered during scan.
 */
interface Device {
  /** IPv4 address of the device */
  ipAddress: string;
  /** MAC address in colon-separated format */
  macAddress: string;
  /** Device hostname if resolved via DNS */
  hostname?: string;
  /** Whether the device responded to ICMP */
  isOnline: boolean;
}
```

## Git Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/) format:

```
type(scope): description

[optional body]

[optional footer]
```

### Types

| Type       | Description                         |
| ---------- | ----------------------------------- |
| `feat`     | New feature                         |
| `fix`      | Bug fix                             |
| `docs`     | Documentation only                  |
| `style`    | Code style (formatting, whitespace) |
| `refactor` | Code change without feature/fix     |
| `perf`     | Performance improvement             |
| `test`     | Adding or updating tests            |
| `chore`    | Maintenance tasks                   |
| `ci`       | CI/CD changes                       |
| `build`    | Build system changes                |

### Scopes

Common scopes for this project:

- `api` - Backend REST/WebSocket handlers
- `ui` - Frontend components
- `discovery` - Network discovery
- `auth` - Authentication
- `config` - Configuration
- `deps` - Dependencies

### Examples

```
feat(discovery): add SNMP v3 support for device profiling

fix(ui): resolve dark mode flicker on page load (closes #123)

docs: update API endpoint documentation

chore(deps): upgrade gopacket to v1.2.0

refactor(api): extract rate limiting to middleware
```

### Footer

- Reference issues: `closes #123`, `fixes #456`
- Breaking changes: `BREAKING CHANGE: removed deprecated API`
- Co-authors: `Co-Authored-By: Name <email>`

## Additional Resources

- [web/THEMING.md](web/THEMING.md) - Design system tokens and patterns
- [CONTRIBUTING.md](CONTRIBUTING.md) - Contribution guidelines
- [Effective Go](https://go.dev/doc/effective_go) - Go best practices
- [React TypeScript Cheatsheet](https://react-typescript-cheatsheet.netlify.app/) - React + TS
  patterns
