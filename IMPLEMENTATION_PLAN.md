# LuminetIQ Implementation Plan

## Overview

This document outlines the implementation plan for addressing all open GitHub issues, organized into
phases by priority and dependency order.

---

## Phase 1: Critical Bug Fixes & High-Priority Items (Week 1-2)

### 1.1 Staticcheck Issues - Potential Bugs (#565) ⚠️ HIGH PRIORITY

**Why First**: These are potential runtime bugs that could cause crashes or incorrect behavior.

**Issues to Fix** (6 total):

- SA1019: Deprecated API usage
- SA4006: Value assigned but never used
- SA9003: Empty branch statements
- Potential nil pointer dereferences

**Files to Check**:

```bash
golangci-lint run --enable staticcheck 2>&1 | grep -E "^internal/"
```

**Estimated Effort**: 2-4 hours

---

### 1.2 Remove Hardcoded Interface Names (#572) ⚠️ HIGH PRIORITY

**Why**: Hardcoded interfaces break portability across Linux distributions and macOS.

**Search Pattern**:

```bash
grep -rn "eth0\|en0\|enp\|wlan0\|wlp" internal/
```

**Replace With**: Dynamic interface detection via `internal/network/interfaces.go`

**Estimated Effort**: 4-6 hours

---

### 1.3 Add .nvmrc File (#569) ⚠️ HIGH PRIORITY

**Why**: Ensures consistent Node.js version across all environments.

**Implementation**:

```bash
echo "22" > web/.nvmrc
```

**Update** `web/package.json`:

```json
{
  "engines": {
    "node": ">=22.0.0"
  }
}
```

**Estimated Effort**: 30 minutes

---

## Phase 2: Interface Auto-Detection System (Week 2-3)

### 2.1 Intelligent Interface Auto-Detection (#571)

**New Package**: `internal/network/detection/`

**Files to Create**:

```
internal/network/detection/
├── detection.go      # Main detection logic
├── scoring.go        # Interface scoring algorithm
├── chipsets.go       # Chipset database and identification
├── capabilities.go   # TDR/DOM capability detection
└── detection_test.go # Comprehensive tests
```

**Core Algorithm**:

```go
type InterfaceScore struct {
    Name           string
    FriendlyName   string
    Score          int
    LinkStatus     bool
    Speed          int64    // bits per second
    ChipsetQuality int      // 1-100
    HasTDR         bool
    HasDOM         bool
    Type           string   // "ethernet", "wifi", "fiber"
}

func ScoreInterface(iface net.Interface) InterfaceScore {
    score := 0

    // Link status (heavily weighted)
    if isLinked(iface) {
        score += 1000
    }

    // TDR capability for cable testing
    if hasTDRCapability(iface) {
        score += 1000
    }

    // Speed bonus
    speed := getSpeed(iface)
    score += speedScore(speed) // 100G=500, 10G=400, 5G=350, 2.5G=300, 1G=200

    // Chipset quality
    chipset := identifyChipset(iface)
    score += chipsetScore(chipset) // Intel=100, Broadcom=80, Realtek=50

    return InterfaceScore{Score: score, ...}
}
```

**Chipset Database** (from issue #571):

- 1G: Intel I210/I211/I350, Broadcom BCM5720, Realtek RTL8111
- 2.5G: Intel I225/I226, Realtek RTL8125, Aquantia AQC107
- 5G: Aquantia AQC107/108, Marvell AQC113, Realtek RTL8126
- 10G: Intel X540/X550/X710, Mellanox ConnectX-3/4/5
- 25G/40G/100G: Intel XXV710/XL710/E810, Mellanox ConnectX-5/6
- WiFi: Intel AX200/210/211/BE200, Qualcomm, MediaTek

**Estimated Effort**: 16-24 hours

---

### 2.2 Friendly Interface Names in UI (#574)

**Backend Changes**:

```go
type InterfaceInfo struct {
    Name         string `json:"name"`          // "enp3s0"
    FriendlyName string `json:"friendlyName"`  // "Intel I225-V 2.5GbE"
    Description  string `json:"description"`   // "2.5 Gigabit Ethernet"
}
```

**Frontend Changes**:

- Display `friendlyName` in dropdowns and cards
- Show `name` in tooltips or advanced view
- Update `InterfaceSelector` component

**Estimated Effort**: 4-6 hours

---

### 2.3 WiFi Multi-Adapter Support (#573)

**Changes to Survey System**:

```go
type SurveyConfig struct {
    Adapters []string `json:"adapters"` // Multiple adapters
    Primary  string   `json:"primary"`  // Primary adapter
}
```

**UI Changes**:

- Multi-select for WiFi adapters in survey setup
- Show per-adapter results in survey view

**Estimated Effort**: 8-12 hours

---

## Phase 3: Code Quality - Linting Issues (Week 3-4)

### 3.1 Cyclomatic Complexity (#560) - 29 issues

**Strategy**: Extract helper functions, use early returns, simplify conditionals.

**High Complexity Files** (likely candidates):

- `internal/discovery/scanner.go`
- `internal/api/handlers.go`
- `internal/wifi/scanner.go`

**Refactoring Pattern**:

```go
// Before: Complex function with many branches
func processDevice(d Device) error {
    if d.Type == "router" {
        // 50 lines of router logic
    } else if d.Type == "switch" {
        // 50 lines of switch logic
    }
    // ...more branches
}

// After: Extracted handlers
func processDevice(d Device) error {
    handlers := map[string]func(Device) error{
        "router": processRouter,
        "switch": processSwitch,
    }
    if handler, ok := handlers[d.Type]; ok {
        return handler(d)
    }
    return ErrUnknownDeviceType
}
```

**Estimated Effort**: 8-12 hours

---

### 3.2 Revive Style Issues (#561) - 50 issues

**Common Fixes**:

- `exported`: Add documentation to exported functions
- `unused-parameter`: Remove or use `_` prefix
- `error-return`: Return error as last value
- `context-as-argument`: Context should be first parameter
- `var-naming`: Use camelCase, not snake_case

**Estimated Effort**: 4-6 hours

---

### 3.3 Code Duplication (#563) - 13 issues

**Strategy**: Extract common patterns into shared functions.

**Common Duplication Patterns**:

- HTTP handler boilerplate
- Error response formatting
- Configuration loading

**Estimated Effort**: 6-8 hours

---

### 3.4 Exhaustive Switch Statements (#564) - 7 issues

**Fix Pattern**:

```go
// Before: Missing cases
switch status {
case StatusOK:
    return "ok"
case StatusError:
    return "error"
}

// After: All cases handled
switch status {
case StatusOK:
    return "ok"
case StatusError:
    return "error"
case StatusWarning:
    return "warning"
case StatusUnknown:
    return "unknown"
}
```

**Estimated Effort**: 2-3 hours

---

### 3.5 Repeated Strings to Constants (#562) - 23 issues

**Create constants file**: `internal/constants/strings.go`

```go
package constants

const (
    // HTTP Headers
    HeaderContentType   = "Content-Type"
    HeaderAuthorization = "Authorization"

    // Content Types
    ContentTypeJSON = "application/json"

    // Status Messages
    MsgSuccess = "success"
    MsgError   = "error"
)
```

**Estimated Effort**: 2-3 hours

---

### 3.6 Minor Lint Issues (#566) - 10 issues

**Issues**:

- `godot`: Add periods to comments
- `intrange`: Use integer range in for loops
- `unparam`: Remove unused parameters
- `nilnil`: Don't return (nil, nil)
- `nolintlint`: Fix malformed nolint directives

**Estimated Effort**: 1-2 hours

---

## Phase 4: Node.js 22+ Enhancements (Week 4)

### 4.1 Native Node.js Features (#568)

**Replace Dependencies**:

| Current                   | Node.js 22+ Native        |
| ------------------------- | ------------------------- |
| `node-fetch`              | `fetch()`                 |
| `ws` (if used for client) | Consider native WebSocket |
| `dotenv`                  | `--env-file` flag         |

**Update Scripts** in `package.json`:

```json
{
  "scripts": {
    "dev": "node --env-file=.env vite",
    "test:native": "node --test"
  }
}
```

**Estimated Effort**: 4-6 hours

---

### 4.2 Evaluate Native Test Runner (#570)

**Comparison Matrix**:

| Feature               | Vitest      | Node.js Test Runner                |
| --------------------- | ----------- | ---------------------------------- |
| Speed                 | Fast (Vite) | Fast (native)                      |
| Watch mode            | Yes         | Yes (--watch)                      |
| Coverage              | Yes         | Yes (--experimental-test-coverage) |
| Mocking               | Built-in    | Built-in                           |
| React Testing Library | Supported   | Manual setup                       |
| Snapshot Testing      | Yes         | Basic                              |

**Recommendation**: Keep Vitest for React component tests, consider native for pure utility tests.

**Estimated Effort**: 2-4 hours (evaluation only)

---

## Phase 5: Epic Completion & Verification (Week 5)

### 5.1 Final Verification

**Run Full Lint Suite**:

```bash
golangci-lint run ./...
```

**Expected Result**: 0 issues

**Run All Tests**:

```bash
make test
cd web && npm test
```

### 5.2 Close Epic Issue (#567)

Once all child issues are resolved, close the epic with summary.

---

## Implementation Order Summary

```
Priority Order (by issue number):
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
1. #565 - staticcheck (BUGS)        ← Start here
2. #572 - hardcoded interfaces (BUG)
3. #569 - .nvmrc (HIGH)
4. #571 - interface auto-detection
5. #574 - friendly names UI
6. #573 - WiFi multi-adapter
7. #560 - gocyclo (29 issues)
8. #563 - dupl (13 issues)
9. #564 - exhaustive (7 issues)
10. #561 - revive (50 issues)
11. #562 - goconst (23 issues)
12. #566 - minor lint (10 issues)
13. #568 - Node.js features
14. #570 - test runner eval
15. #567 - close EPIC
```

---

## Git Workflow for Each Issue

```bash
# 1. Create branch
git checkout -b fix/issue-565-staticcheck

# 2. Make changes
# ... code changes ...

# 3. Run verification
make lint && make test

# 4. Commit with issue reference
git commit -m "fix(lint): resolve staticcheck issues

- Fix SA1019 deprecated API usage
- Fix SA4006 unused value assignments
- Fix SA9003 empty branches

Closes #565"

# 5. Push and create PR
git push -u origin fix/issue-565-staticcheck
gh pr create --title "fix(lint): resolve staticcheck issues" \
  --body "Closes #565" --base main

# 6. After merge, tag if milestone reached
git tag -a v0.103.0 -m "v0.103.0: Code quality improvements"
git push origin --tags
```

---

## Metrics & Success Criteria

| Metric                | Before   | Target       |
| --------------------- | -------- | ------------ |
| golangci-lint issues  | 138+     | 0            |
| Hardcoded interfaces  | Multiple | 0            |
| Test coverage (Go)    | ~60%     | 80%+         |
| Test coverage (React) | ~70%     | 85%+         |
| Node.js version       | Any      | 22+ enforced |

---

## Risk Mitigation

1. **Breaking Changes**: Run full test suite after each refactor
2. **Regression**: E2E tests catch UI regressions
3. **Interface Detection**: Extensive testing on multiple platforms (macOS, Ubuntu)
4. **Chipset Detection**: Fallback to generic detection if specific chipset unknown

---

## Notes

- All work follows `CLAUDE.md` guidelines
- Commits use conventional commit format
- Issues closed via commit messages (`Closes #XXX`)
- Each phase ends with a version tag
