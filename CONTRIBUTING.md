# Contributing to The Seed

Thank you for your interest in contributing to The Seed! This document provides guidelines and instructions for
contributing.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions.

## Getting Started

### Prerequisites

- Go 1.22+
- Node.js 24+
- libpcap-dev (Linux) or libpcap (macOS)
- Git

### Development Setup

````bash
# Clone the repository
git clone https://github.com/MustardSeedNetworks/seed.git
cd seed

# Install Go dependencies
go mod download

# Install frontend dependencies
cd ui && npm install && cd ..

# Run tests
make test

# Run linting
make lint
```bash

### Running as a service (Linux)

There is no hand-rolled systemd installer in this repository. On Linux a service
install is the released package, which is the only layout the product is built
and tested against: `/usr/bin/seed`, `/etc/seed`, `/var/lib/seed`, `/var/log/seed`, a
system `seed` user, and the unit at `/usr/lib/systemd/system/seed.service`.

```bash
# Ubuntu / Debian
sudo apt-get install -y ./seed_<version>_<arch>.deb

# Fedora / RHEL
sudo dnf install ./seed-<version>-1.<arch>.rpm

# Service management
sudo systemctl status seed
journalctl -u seed -f
```

Packages for every supported architecture are attached to each
[release](https://github.com/MustardSeedNetworks/seed/releases). After
installing, confirm the deployment reports the version and commit you installed,
and that its UI was embedded:

```bash
./scripts/deploy-validate.sh <expected-version> <expected-commit> [host] [port]
```

For day-to-day development, run the binary in the foreground instead. An
unprivileged run that is not a systemd service resolves to the XDG user
directories rather than `/etc/seed` and `/var/lib/seed`
(`internal/paths/paths.go`, `detectActualMode`), so it needs no root and leaves
a packaged install alone:

```bash
make build   # builds the UI, embeds it, and stamps version/commit/uiBuildHash
./seed serve
```

`make build` is the only build that embeds the frontend and injects the build
metadata `/__version` reports; a bare `go build` produces a binary whose
`uiBuildHash` and `version` both read `unknown`
(`internal/version/version.go`, `GetUIBuildHash`).

ICMP and Wi-Fi features need raw-socket privileges. Either run with `sudo`, or
grant the capabilities the package grants:

```bash
sudo setcap cap_net_raw,cap_net_admin=+ep ./seed
```

#### First-boot setup

There are no default credentials: the first visit to the web UI runs the setup
wizard, which sets the admin password. To check whether a deployment still
needs that wizard — useful on a headless host — run:

```bash
seed credentials          # status banner
seed credentials --json   # machine-readable
```

#### Uninstall

```bash
sudo apt-get remove seed     # or: sudo apt-get purge seed
sudo dnf remove seed
```

## Development Workflow

### Branch Naming

Use descriptive branch names with prefixes:

- `feat/` - New features (e.g., `feat/wifi-card`)
- `fix/` - Bug fixes (e.g., `fix/dhcp-timeout`)
- `docs/` - Documentation (e.g., `docs/api-reference`)
- `chore/` - Maintenance (e.g., `chore/update-deps`)
- `refactor/` - Code refactoring (e.g., `refactor/websocket-handler`)

### Commit Messages

We use [Conventional Commits](https://www.conventionalcommits.org/). All commits must follow this format:

```text
type(scope): description

[optional body]

[optional footer]
```text

#### Types

- `feat` - New feature
- `fix` - Bug fix
- `docs` - Documentation changes
- `style` - Code style changes (formatting)
- `refactor` - Code refactoring
- `perf` - Performance improvements
- `test` - Adding or updating tests
- `chore` - Maintenance tasks
- `ci` - CI/CD changes
- `build` - Build system changes

#### Examples

```text
feat(dhcp): add phase timing breakdown
fix(websocket): resolve connection drop on idle
docs: update installation instructions
chore(deps): upgrade gopacket to v1.2.0
```python

### Pull Request Process

1. **Create an issue first** - Discuss the change before implementing
2. **Fork and branch** - Create a feature branch from `main`
3. **Write tests** - Ensure adequate test coverage
4. **Update docs** - Update relevant documentation
5. **Run checks** - Ensure all tests and linting pass
6. **Submit PR** - Reference the related issue

### PR Title Format

Use the same format as commit messages:

```text
feat(dhcp): add phase timing breakdown (#123)
```bash

### Test Report Templates

Use `docs/templates/WIFI_TEST_REPORT.md` for Wi-Fi validation and
`docs/templates/ETHERNET_TEST_REPORT.md` for Ethernet validation.

## Code Standards

### Go

- Follow [Effective Go](https://go.dev/doc/effective_go)
- Use `gofmt` for formatting
- Run `golangci-lint` before committing
- Write table-driven tests
- Document exported functions

### TypeScript/React

- Follow the existing code style
- Use TypeScript strict mode
- Write unit tests with Vitest
- Use functional components with hooks
- Develop components in isolation with Storybook

#### Storybook

We use [Storybook](https://storybook.js.org/) for developing and documenting UI components in isolation.

```bash
# Start Storybook development server
cd ui && npm run storybook

# Build Storybook static site
cd ui && npm run build-storybook
```bash

When creating new UI components:

1. Create a `.stories.tsx` file alongside your component
2. Document all component variants and states
3. Test accessibility with the built-in a11y addon
4. Verify dark/light mode appearance

### General

- Keep functions small and focused
- Write self-documenting code
- Add comments for complex logic
- Handle errors explicitly
- No hardcoded secrets or credentials

## Testing

### Running Tests

```bash
# All tests
make test

# Go tests with coverage
go test -coverprofile=coverage.out ./...

# Go tests with race detection (recommended for concurrent code)
go test -race ./...

# Frontend tests
cd ui && npm test

# Frontend tests with coverage
cd ui && npm test -- --coverage

# E2E tests (requires running server)
make test-e2e
```

### SNMP integration suite

The unit tests drive the SNMP collectors through a fake client. That is the
right shape for behaviour, but a fake returns whatever Go value the test author
picked, while a real agent returns whatever BER it encodes — so wire-format
divergence (Counter64 vs Counter32, an OID vs an OCTET STRING, an empty table vs
`noSuchObject`) is invisible to it. The `integration` build tag covers that gap
against a live net-snmp agent.

Start a seeded agent on loopback, then point the suite at it:

```bash
snmpd -C -c internal/polling/snmp/snmpclient/testdata/snmpd.conf \
      -Lf /tmp/snmpd.log -p /tmp/snmpd.pid

SEED_SNMP_ADDR=127.0.0.1:1161 \
  go test -tags integration ./internal/polling/snmp/...

kill "$(cat /tmp/snmpd.pid)"
```

Without `SEED_SNMP_ADDR` the suite skips, which is right locally and wrong in
CI — a job that skips everything reports green. The workflow therefore also
sets `SEED_SNMP_REQUIRED=1`, which turns a missing address into a failure.

Nightly and on-demand via `.github/workflows/snmp-integration.yml`; not on every
PR, since it needs an agent and adds no signal to a typical change.

**One caveat on macOS.** `/usr/sbin/snmpd` there is net-snmp 5.6.2.1, which
accepts any community string regardless of `rocommunity`. A local pass therefore
says nothing about community handling — it exercises the wire format, which is
the point. CI runs a current net-snmp from apt.

### First-run discovery over a NIAC link

`make test-e2e-niac-link` checks the out-of-the-box discovery defaults against
a simulated segment: it puts NIAC's `home-network` scenario in a network
namespace at the far end of a veth pair, starts seed on the near end with only
its interface, port and paths configured, and requires every scenario device in
the daemon's inventory and on the Network and Security pages within 90 s of
start. Linux only; it needs `niac` on `PATH` and sudo for the namespace and the
two daemons. Another scenario and address can be passed straight to
`scripts/e2e-niac-link.sh TEMPLATE SEED_ADDRESS/PREFIX`.

### Target networks behind a routed NIAC pack

`make test-e2e-niac-routed` checks target-network learning (#2695) behind a
router. It has NIAC generate its `hospital` pack, runs it behind a veth pair
bound to the pack's transit network, and gives seed one transit address plus
the edge router's own summary route to the site. The spec saves the SNMP
community, requires the summary as a learned (switched-off) target, enters one
site network, and requires every other site network to be learned from a site
device's SNMP tables and its devices found once switched on. Same requirements
as above, plus PyYAML. Another pack can be passed to
`scripts/e2e-niac-routed.sh PACK`.

### SNMP collectors against NIAC's packs

`make test-snmp-niac-packs` is the SNMP acceptance: seed's ten collectors,
over the wire, against every SNMP agent of NIAC's six presentation packs
(hospital, warehouse, campus, retail, manufacturing, service-provider). Each
pack is generated by NIAC with its manifest, run in one network namespace, and
polled from a second namespace routed through the pack's edge router. The run
fails on any collector that errors against an agent and on any device or row
count that disagrees with the manifest's `expectedObservations`; the test log
lists every agent's rows per collector, so a finding can be traced to the
devices behind it. Needs `niac`, PyYAML, Go and sudo; no seed daemon or UI.
Named packs can be passed to `scripts/snmp-acceptance-niac.sh PACK...`.

### Test Requirements

- Unit tests for business logic
- Integration tests for API endpoints
- Aim for >80% coverage on new code

---

## Testing Guidelines

### Go Backend Testing

#### Table-Driven Tests

Use table-driven tests to cover multiple test cases efficiently:

```go
func TestValidateURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid http", "http://example.com", false},
        {"valid https", "https://example.com", false},
        {"missing scheme", "example.com", true},
        {"empty string", "", true},
        {"private IP blocked", "http://192.168.1.1", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateURL(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateURL(%q) error = %v, wantErr %v",
                    tt.input, err, tt.wantErr)
            }
        })
    }
}
```go

#### Test Isolation with Temporary Resources

Use temporary directories and files for tests that involve disk operations:

```go
func TestConfigLoad(t *testing.T) {
    // Create temp directory
    tmpDir := t.TempDir() // Auto-cleaned after test

    // Create test config file
    configPath := filepath.Join(tmpDir, "test-config.json")
    testConfig := []byte(`{"interface":{"default":"eth0"},"server":{"port":8080}}`)
    if err := os.WriteFile(configPath, testConfig, 0644); err != nil {
        t.Fatal(err)
    }

    // Test config loading
    cfg, err := LoadConfig(configPath)
    if err != nil {
        t.Fatalf("LoadConfig() error = %v", err)
    }
    // assertions...
}
```go

#### Mocking External Dependencies

Use interfaces for dependencies that need mocking:

```go
// Define interface for the dependency
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// Production code uses the interface
type Service struct {
    client HTTPClient
}

// Test with mock
type mockHTTPClient struct {
    response *http.Response
    err      error
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    return m.response, m.err
}

func TestServiceFetch(t *testing.T) {
    mock := &mockHTTPClient{
        response: &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("OK"))},
    }
    svc := &Service{client: mock}
    // test svc.Fetch()...
}
```tsx

#### Race Condition Checks

Always run tests with `-race` for concurrent code:

```bash
go test -race ./internal/websocket/...
```tsx

### React Frontend Testing

#### Test File Organization

Tests are co-located with source files:

- `Component.tsx` → `Component.test.tsx`
- `useHook.ts` → `useHook.test.ts`

#### Using the Test Setup

Import mocks from the setup file:

```typescript
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  mockFetch,
  mockLocalStorage,
  MockWebSocket,
  createMockResponse,
  createMockAuthToken,
} from "../test/setup";

describe("MyComponent", () => {
  it("fetches data on mount", async () => {
    mockFetch.mockResolvedValueOnce(
      createMockResponse({ data: "test" })
    );

    render(<MyComponent />);

    await screen.findByText("test");
    expect(mockFetch).toHaveBeenCalledWith("/api/data", expect.any(Object));
  });
});
```text

#### Mocking Strategies

##### Mocking fetch

```typescript
mockFetch.mockImplementation((url: string) => {
  if (url.includes("/api/auth")) {
    return createMockResponse({ token: "test" });
  }
  return createMockResponse({});
});
```typescript

#### Mocking WebSocket

```typescript
beforeEach(() => {
  global.WebSocket = MockWebSocket as unknown as typeof WebSocket;
});

it("handles WebSocket messages", () => {
  render(<Component />);
  const ws = MockWebSocket.instances[0];
  ws.simulateOpen();
  ws.simulateMessage({ type: "update", data: {} });
  // assertions...
});
```text

#### Mocking localStorage

```typescript
it("persists auth token", () => {
  mockLocalStorage.setItem("token", "test-token");
  render(<AuthComponent />);
  expect(mockLocalStorage.setItem).toHaveBeenCalledWith("token", "test-token");
});
```tsx

#### Testing Async Code

Use `waitFor` for async operations:

```typescript
import { waitFor } from "@testing-library/react";

it("loads data asynchronously", async () => {
  mockFetch.mockResolvedValueOnce(createMockResponse({ items: [] }));

  render(<DataList />);

  await waitFor(() => {
    expect(screen.getByRole("list")).toBeInTheDocument();
  });
});
```tsx

#### Testing Context Providers

Wrap components with providers:

```typescript
import { SettingsProvider } from "../contexts/SettingsContext";

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <SettingsProvider>{ui}</SettingsProvider>
  );
}

it("uses settings context", () => {
  renderWithProviders(<SettingsConsumer />);
  // assertions...
});
```python

### Test Data Factories

Use factories for consistent test data:

```typescript
// From setup.ts
import { createMockAuthToken, createMockThresholds } from "../test/setup";

it("handles expired token", () => {
  const expiredToken = createMockAuthToken(-3600); // Expired 1 hour ago
  mockLocalStorage.setItem("token", expiredToken.token);
  // ...
});
```python

### Common Patterns

1. **Arrange-Act-Assert** - Structure tests clearly
2. **One assertion per test** (when practical)
3. **Descriptive test names** - `it("shows error when login fails")`
4. **Avoid testing implementation details** - Test behavior, not internals
5. **Clean up after tests** - The setup file handles this automatically

## Issue Guidelines

### Issue Templates

We use GitHub Issue Forms for consistent, well-structured issues. When creating an issue, select the appropriate
template:

| Template        | Prefix       | Use Case                                   |
| --------------- | ------------ | ------------------------------------------ |
| Bug Report      | `[BUG]`      | Report bugs and unexpected behavior        |
| Feature Request | `[FEATURE]`  | Suggest new features or enhancements       |
| Task            | `[CHORE]`    | Internal development tasks and maintenance |
| Hardware Report | `[Hardware]` | Report hardware compatibility results      |

### Label Taxonomy

Labels are automatically applied based on issue form inputs and PR file paths.

**Type Labels** (applied by template):

- `type: defect` - Product defects/regressions
- `type: feature` - New feature requests
- `type: chore` - Maintenance and refactoring
- `type: docs` - Documentation
- `type: security` - Security vulnerabilities/hardening
- `type: epic` - Epic/umbrella tracking

**Priority Labels** (one per issue):

- `priority: critical`, `priority: high`, `priority: medium`, `priority: low`

**Area Labels** (applied by form selection or PR paths; use one or two):

- `area: auth/setup`, `area: survey/floorplan`, `area: discovery/network`, `area: logging`, `area: UI/UX`, `area: backend`, `area: frontend`, `area: infra`, `area: i18n`

**Status Labels**:

- `status: needs-triage`, `status: investigating`, `status: blocked`, `status: regression`

**Meta Labels**:

- `help wanted`, `good first issue`, `question`, `duplicate`, `invalid`, `wontfix`
- `area: security` - Security-related

**Priority Labels** (applied by form selection):

- `priority: critical` - Blocking or severe
- `priority: high` - Important for next release
- `priority: medium` - Should be done soon
- `priority: low` - Nice to have

### Bug Reports

Include:

- The Seed version
- OS and architecture
- Steps to reproduce
- Expected vs actual behavior
- Relevant logs

### Feature Requests

Include:

- Clear description of the feature
- Use case / problem it solves
- Proposed implementation (if any)

## Hardware Testing Contributions

The Seed's advanced features (WiFi scanning, cable diagnostics, etc.) depend on hardware support. We welcome hardware
compatibility reports from the community!

### Reporting Hardware Compatibility

Use the **Hardware Report** issue template to document your experience with specific network adapters. Include:

- **Hardware details**: Make, model, chipset
- **Platform**: OS, kernel version, driver
- **Features tested**: WiFi scanning, TDR, packet capture, etc.
- **Test results**: What works, what doesn't, performance notes
- **Configuration**: Any special setup required

### Testing Guidelines

1. **Reference HARDWARE.md** - Check our hardware compatibility guide first
2. **Test systematically** - Document each feature (WiFi scan, speed test, cable diagnostics)
3. **Include diagnostics** - Provide relevant command outputs (`lsusb`, `lspci`, `ethtool`)
4. **Report issues** - Both successes and failures help improve compatibility
5. **Update documentation** - Propose updates to HARDWARE.md via PR

### What to Test

#### WiFi Adapters

- Passive scanning (`internal/wifi/scanner_*.go`)
- Channel availability
- Signal strength accuracy
- Packet injection (if applicable)

#### Ethernet NICs

- Cable diagnostics/TDR (`internal/cable/cable_*.go`)
- Link speed detection
- Autonegotiation
- Statistics accuracy

#### Platform-Specific

- macOS WiFi scanning via CoreWLAN
- Linux nl80211 support
- Driver-specific quirks

### Hardware Testing Label

Issues labeled `hardware-testing` track compatibility testing efforts. Feel free to claim tests from the backlog or
propose new hardware to test.

### Example Hardware Report

```markdown
**Hardware**: Intel AX200 WiFi 6 Adapter **OS**: Ubuntu 22.04 LTS (kernel 5.15.0) **Driver**: iwlwifi **Chipset**: Intel
AX200

**Features Tested**:

- ✅ Passive scanning: Works perfectly, all 2.4/5 GHz channels
- ✅ Signal strength: Accurate RSSI readings
- ✅ Interface switching: Seamless
- ❌ Packet injection: Not tested

**Notes**: Required `sudo` for scanning. Performance excellent. **Recommendation**: Highly recommended for WiFi survey
features.
```text

## Quality Gates

Commits + PRs are gated by automation, all configured in-repo:

- **Commit message format** — enforced by `commitlint.config.js`
  (conventional commits + per-project scope list). Husky runs it on
  every commit via `.husky/commit-msg`.
- **Pre-commit checks** — `.pre-commit-config.yaml` runs on
  `git commit`: secret detection (gitleaks), formatting (biome,
  clang-format), shellcheck on `.sh` scripts, large-file checks,
  schema validation.
- **CI required checks** — `CI Complete`, `License Compliance Check`,
  and CodeQL (`Analyze (go)` + `Analyze (javascript-typescript)`)
  must pass before merge.
- **Coverage floor** — `ui/vitest.config.ts` enforces a per-project
  anti-regression threshold; the Go test job in CI enforces its own.

If pre-commit blocks a commit, fix the issue locally — **do not** use
`--no-verify` (forbidden by the development policy).

## Reporting Security Vulnerabilities

**Do not open a public issue for a security vulnerability.** See
[SECURITY.md](SECURITY.md) for the private disclosure channels
(GitHub Security Advisories or email).

## Questions?

- Check existing issues and documentation
- Open a discussion for general questions
- File an issue for bugs or features

Thank you for contributing!
````
