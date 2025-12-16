# The Seed - Copilot Instructions

## Project Identity

**Product**: "The Seed" by Mustard Seed Networks  
**Module**: `github.com/krisarmstrong/seed`  
**Binary**: `seed`  
**What**: Professional network diagnostic appliance - portable hardware device plugged into network
jacks for real-time diagnostics via web UI.

⚠️ **INCOMPLETE RENAME**: The codebase is currently mid-rename from "NetScope/LuminetIQ" to "The
Seed". You'll encounter legacy references throughout:

- Comments, docs, and UI text still reference "NetScope", "LuminetIQ"
- Some file paths and configs use old names
- Package/test data may use old identifiers

**When writing NEW code, always use**:

- **Product name**: "The Seed" (UI/user-facing)
- **Binary/CLI**: `seed`
- **Import paths**: `github.com/krisarmstrong/seed`
- **Config paths**: `configs/seed.yaml`, `~/.config/seed/`
- **Storage keys**: `seed-token`, `seed-theme` (not `netscope-*`)
- **File payloads**: `[]byte("seed")` (not `[]byte("netscope")`)

**When updating EXISTING code with old names**: Replace references consistently. See RENAME_PLAN.md
in the repo root for comprehensive list of what needs updating.

## Architecture & Tech Stack

**Backend**: Go 1.25.5+ monolith with modular internal packages  
**Frontend**: React 19 + Vite + Tailwind CSS 4.x (WiFi Vigilante theme)  
**Real-time**: WebSocket broadcast loop (5s interval) pushes card updates to clients  
**Platform**: Primary target is Raspberry Pi (ARM64), also supports x86_64 Linux

### Critical Dependencies

- **gopacket/libpcap**: Raw packet capture for LLDP/CDP/EDP/FDP switch discovery, VLAN detection,
  DHCP monitoring
- **vishvananda/netlink**: Linux netlink for interface management, routing tables
- **mdlayher/wifi**: nl80211 interface for WiFi diagnostics
- **gorilla/websocket**: Real-time dashboard updates
- **JWT + bcrypt**: Authentication (default admin/seed - must be changed)

## Key Design Patterns

### 1. Card-Based Dashboard Architecture

**Backend** (`internal/api/broadcast.go`): Collectors run every 5s, push to WebSocket hub  
**Frontend** (`web/src/App.tsx`): Card components subscribe to WebSocket, update independently

Cards: Link, Cable, Switch (LLDP/CDP), VLAN, WiFi, DHCP, DNS, Gateway, Public IP, Network Discovery

### 2. WebSocket Broadcast System

```go
// Broadcast loop skips work when no clients connected (optimization)
if s.wsHub.ClientCount() == 0 { continue }
s.broadcastAllCards()  // Collects & pushes all card data
```

Frontend connects once, receives typed `CardUpdate` messages, routes to appropriate card state.

### 3. Handler Organization (fixes #544)

API handlers split by domain into `handlers_*.go` files:

- `handlers_types.go` - Shared utilities
- `handlers_status.go` - System status, logs, health
- `handlers_network.go` - Interface, link, IP, VLAN, WiFi, cable
- `handlers_settings.go` - Application settings
- `handlers_tools.go` - TCP probe, traceroute, port scan
- `handlers_security.go` - Rogue DHCP detection, SNMP
- `handlers_tests.go` - DNS tests, speedtest, iperf
- `handlers_discovery.go` - Device discovery, LLDP/CDP, WiFi survey

### 4. Configuration Management

**Thread-safe** `Config` struct uses `sync.RWMutex`:

```go
cfg.RLock()
defer cfg.RUnlock()
// Read operations
```

Config persists to `configs/seed.yaml` on updates. Default file: `configs/seed.yaml`.

### 5. ICMP Capabilities

Ping requires `CAP_NET_RAW`. Check with `discovery.CheckICMPPrivilegesWithMessage()`.  
Gracefully degrades if unavailable - logs warning, disables ping-dependent features.

## Development Workflow

### Build & Run

```bash
# Build everything (frontend embedded in Go binary)
make build

# Development (hot reload frontend, backend watches Go files)
make dev

# Run tests
make test           # All tests
make test-backend   # Go tests only
make test-frontend  # Vitest + Playwright

# Linting
make lint           # All linters
make fix            # Auto-fix issues
```

### Key Commands

- `sudo ./seed` - Run with ICMP capabilities (required for ping)
- `seed credentials` - Generate secure credentials (for headless deployments)
- `make deploy` - Build + deploy to remote Ubuntu server (see Makefile for DEPLOY\_\* vars)

### Reproducible Builds

Always use `-trimpath -buildvcs=false` flags (enforced in Makefile, CI). See `go.mod` for exact Go
version.

## Code Standards

### Naming Conventions

| Element           | Style        | Example                    |
| ----------------- | ------------ | -------------------------- |
| Go files          | `snake_case` | `handlers_network.go`      |
| React files       | `PascalCase` | `LinkCard.tsx`             |
| Hooks             | `camelCase`  | `useWebSocket.ts`          |
| Tests             | `*_test.go`  | `handlers_network_test.go` |
| Platform-specific | `_os.go`     | `gateway_linux.go`         |

See `STYLE_GUIDE.md` for comprehensive standards.

### Commit Messages

Use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(wifi): add WPA3 support
fix(dhcp): correct phase timing calculation
docs(api): update WebSocket message format
chore(deps): update gopacket to v1.1.20
```

### Frontend Theming

**CSS-first** Tailwind 4.x with `@theme` directive. Use semantic classes from `web/THEMING.md`:

```tsx
<h1 className="heading-1">Page Title</h1>
<p className="body-small">Description</p>
<button className="text-status-success">Connected</button>
```

Color tokens in `web/src/index.css` as CSS variables, mapped in `web/src/styles/theme.ts`.

## Critical Context for AI Agents

### Interface Auto-Detection Issues

**Open Issue**: Hardcoded interface names (`eth0`, `en0`) break portability across Linux distros and
macOS. See Issue #572.

**Pattern to avoid**:

```go
iface := "eth0"  // ❌ Hardcoded
```

**Correct approach**:

```go
iface, err := network.GetDefaultInterface()  // ✅ Dynamic detection
```

Dynamic detection in `internal/network/interfaces.go`. Always use config-driven or auto-detected
interfaces.

### Security Considerations

1. **Input Validation**: All user input via `internal/validation` package
2. **Authentication**: JWT required for all endpoints except `/api/auth/login` and setup endpoints
3. **Credentials**: Encrypt sensitive data (SNMP community strings, WiFi passwords) before storage
4. **CORS**: Restrict origins via config `security.allowed_origins`
5. **Rate Limiting**: Applied to resource-intensive endpoints (speedtest, iperf, port scan)

### Testing Strategy

- **Go**: Table-driven tests, test helpers in `*_test.go` files
- **React**: Vitest for components, Playwright for E2E
- **Coverage**: CI enforces 40% minimum (incrementally increasing to 90%)
- **Integration**: See `internal/api/endpoints_integration_test.go` for HTTP/WebSocket test patterns

### Multi-Platform Support

Use build tags for platform-specific code:

```go
//go:build linux
// +build linux

package gateway
```

Common split: `gateway_linux.go` vs `gateway_darwin.go` (macOS stubs for development).

## Common Operations

### Adding a New Card

1. Backend: Create collector in `internal/api/broadcast_*.go`, add to `broadcastAllCards()`
2. Frontend: Create `web/src/components/cards/YourCard.tsx`
3. Update `CardUpdate` type in `web/src/hooks/useWebSocket.ts`
4. Add card state to `CardState` interface in `web/src/App.tsx`

### Adding API Endpoint

1. Create handler in appropriate `internal/api/handlers_*.go` file
2. Register route in `internal/api/server.go`
3. Add authentication middleware if needed
4. Update API validation in `internal/validation/api.go`

### Internationalization

I18n via `react-i18next`. Translation files in `locales/{lang}/common.json`:

```tsx
const { t } = useTranslation("common");
<p>{t("link.status")}</p>;
```

## Known Issues & Workarounds

- **Issue #565**: Staticcheck warnings (SA1019, SA4006, SA9003) - in progress
- **Issue #572**: Hardcoded interface names - use `network.GetDefaultInterface()`
- **ICMP Privileges**: Graceful degradation - check `icmpAvailable` flag before ping operations

## Deployment

**Systemd service** (Ubuntu/Linux):

```bash
sudo ./deploy/systemd/install.sh     # Install service
sudo systemctl status seed           # Check status
journalctl -u seed -f                # View logs
sudo ./deploy/systemd/uninstall.sh   # Uninstall
```

Logs: `logs/seed.log` (rotated via lumberjack)  
Config: `configs/seed.yaml`  
Default HTTPS port: `8443`

## Useful Files for Context

- `PROJECT_PLAN.md` - Complete project vision and roadmap
- `IMPLEMENTATION_PLAN.md` - Phased implementation plan for open issues
- `CONTRIBUTING.md` - Contributor workflow and setup
- `HARDWARE.md` - Hardware compatibility (Wi-Fi adapters, TDR support)
- `Makefile` - All build targets extensively documented

## Quick Reference

```bash
# First time setup
make build && sudo ./seed

# Access UI (default credentials: admin/seed)
open https://localhost:8443

# Development workflow
make dev              # Hot reload both frontend/backend
make test-coverage    # Run tests with coverage report
make lint-fix         # Auto-fix linting issues

# Deployment
make deploy DEPLOY_HOST=192.168.1.100  # Deploy to remote server
```
