# Seed Project - Issues & Implementation Plans

This document contains identified issues in the Seed project along with actionable implementation plans to address them.

---

## Part 1: Issue Analysis

## 1. Network Settings Icon

**Status:** CONFIRMED - Missing **Priority:** Low **Effort:** 15 minutes

Network Settings does not have a proper icon.

**Context:** In [SettingsDrawer.tsx:1351](web/src/components/settings/SettingsDrawer.tsx#L1351), the Network section
uses a `CollapsibleSection` without an icon, unlike other sections:

- CableTestSettings uses `<Cable />` icon
  ([CableTestSettings.tsx:91](web/src/components/settings/sections/CableTestSettings.tsx#L91))
- LinkSettings uses `<PlugZap />` icon
  ([LinkSettings.tsx:158](web/src/components/settings/sections/LinkSettings.tsx#L158))

**Fix:** Import and use the `Network` icon from [Icons.tsx:40](web/src/components/ui/Icons.tsx#L40) (already exported
from lucide-react).

---


## 2. Metric & SAE Settings Duplication

**Status:** CONFIRMED - Architecture Issue **Priority:** Medium **Effort:** 2 hours

Both Cable Test & Network have a Metric & SAE Settings - this is really a global setting for Cable Test & Survey /
Planner.

**Context:** Unit system IS implemented in [settings.ts:81-87](web/src/types/settings.ts#L81-L87):

```typescript
/** Unit system for measurements - SAE (feet) or Metric (meters) */
export type UnitSystem = "sae" | "metric";

/** Unit system for distances and measurements (default: SAE/feet) */
unitSystem: UnitSystem;
```

Also in Cable Test settings ([settings.ts:526-527](web/src/types/settings.ts#L526-L527)):

```typescript
/** Unit for cable length display */
lengthUnit: "feet" | "meters";
```

**Issue:** Unit settings exist in multiple places (global `unitSystem` AND cable-specific `lengthUnit`). These should be
consolidated into a single global setting that applies to:

- Cable Test measurements
- Survey / Planner distances
- Network discovery reporting

**Action Needed:** Consolidate to single global `unitSystem` setting and remove duplicate `lengthUnit` from Cable Test
settings.

---

## 3. Profile Multi-Interface Support

**Status:** PARTIALLY IMPLEMENTED **Priority:** High **Effort:** 1-2 days

### Current Implementation

- Profile creation exists in [ProfileManagement.tsx:89-93](web/src/components/profiles/ProfileManagement.tsx#L89-L93)
- [ProfileEditor.tsx](web/src/components/profiles/ProfileEditor.tsx) handles: Name, Description, Notes, Default checkbox

### The Goal: Multiple Interfaces Per Profile with Per-Interface Settings

A profile should support **multiple interfaces of each type** (e.g., 4 Ethernet + 3 WLAN in one profile), and **each
interface can have its own independent settings**.

**Example Configuration:**

```text
Profile: "Office Setup"
├── Ethernet Interfaces:
│   ├── eth0 (Main uplink)
│   │   ├── Thresholds: latency=20ms, packetLoss=1%
│   │   └── Health Checks: gateway=enabled, dns=enabled
│   ├── eth1 (Secondary)
│   │   ├── Thresholds: latency=50ms, packetLoss=5%
│   │   └── Health Checks: gateway=enabled, dns=disabled
│   └── eth2 (IoT network)
│       ├── Thresholds: latency=100ms, packetLoss=10%
│       └── Health Checks: gateway=disabled, dns=disabled
├── WiFi Interfaces:
│   ├── wlan0 (Corporate WiFi)
│   │   ├── Thresholds: signal=-65dBm, latency=30ms
│   │   └── Health Checks: all enabled
│   └── wlan1 (Guest WiFi)
│       ├── Thresholds: signal=-75dBm, latency=100ms
│       └── Health Checks: gateway only
```

### Current Limitation

**Backend** in [profile_settings.go:224-229](internal/config/profile_settings.go#L224-L229):

```go
// Each profile can have one ethernet and one wifi interface selected.
type ProfileInterfaceConfigs struct {
    Ethernet *ProfileInterfaceSelection  // Only ONE
    WiFi     *ProfileInterfaceSelection  // Only ONE
}
```

**Per-interface settings DO exist** in [profile_settings.go:231-246](internal/config/profile_settings.go#L231-L246):

```go
type ProfileInterfaceSelection struct {
    Name         string                 `json:"name"`
    Enabled      bool                   `json:"enabled"`
    Thresholds   *ProfileThresholds     `json:"thresholds,omitempty"`
    HealthChecks *ProfileHealthChecks   `json:"health_checks,omitempty"`
}
```

### Required Changes

#### Backend Changes

```go
// CHANGE FROM:
type ProfileInterfaceConfigs struct {
    Ethernet *ProfileInterfaceSelection  // Only ONE
    WiFi     *ProfileInterfaceSelection  // Only ONE
}

// CHANGE TO:
type ProfileInterfaceConfigs struct {
    Ethernet []ProfileInterfaceSelection  // Multiple interfaces
    WiFi     []ProfileInterfaceSelection  // Multiple interfaces
}
```

#### Frontend Changes

1. Update [ProfileEditor.tsx](web/src/components/profiles/ProfileEditor.tsx) to support:
   - Interface selection wizard (list available interfaces)
   - Per-interface threshold configuration
   - Per-interface health check toggles
   - "Use Profile Defaults" vs "Custom Settings" toggle per interface

2. Update TypeScript types in [settings.ts](web/src/types/settings.ts) to match backend changes

### Missing Features

1. **Multi-interface support per profile** - See above
2. **Interface configuration UI** - Wizard to configure each interface
3. **Export with all settings** - Verify interface configs are included
4. **Multi-profile export** - UI to select specific profiles for export

---

## 4. Default Profile - Single Source of Truth

**Status:** CRITICAL ARCHITECTURE ISSUE **Priority:** Critical **Effort:** 3-5 days

### The Problem

Since the app moved to profiles, the **default profile should be the single source of truth** for all settings. Instead,
defaults are scattered across 20+ locations:

**Frontend (TypeScript)** - [settings.ts:333-540](web/src/types/settings.ts#L333-L540):

- `DEFAULT_CARD_SETTINGS` (line 333)
- `DEFAULT_DISPLAY_OPTIONS` (line 350)
- `DEFAULT_THRESHOLDS` (line 355)
- `DEFAULT_IPERF_SETTINGS` (line 370)
- `DEFAULT_TESTS_SETTINGS` (line 381)
- `DEFAULT_NETWORK_DISCOVERY_SETTINGS` (line 424)
- `DEFAULT_SNMP_SETTINGS` (line 478)
- `DEFAULT_LINK_SETTINGS` (line 530)
- `DEFAULT_CABLE_TEST_SETTINGS` (line 537)

**Backend (Go)** - Various files:

- `DefaultConfig()` - [config.go:618](internal/config/config.go#L618)
- `DefaultConfig()` - [database.go:77](internal/database/database.go#L77)
- `DefaultRetentionPolicy()` - [retention.go:39](internal/database/retention.go#L39)
- `DefaultRateLimitConfig()` - [ratelimit.go:43](internal/api/ratelimit.go#L43)
- `DefaultThresholds()` - [gateway.go:62](internal/gateway/gateway.go#L62)
- `DefaultThresholds()` - [dns.go:85](internal/dns/dns.go#L85)
- `DefaultProfilerConfig()` - [profiler.go:89](internal/discovery/profiler.go#L89)
- `DefaultLoggingConfig()` - [logger.go:33](internal/logging/logger.go#L33)
- `DefaultHeatmapConfig()` - [heatmap.go:53](internal/survey/heatmap.go#L53)
- `DefaultCookieConfig()` - [cookie.go:45](internal/auth/cookie.go#L45)

### Current Architecture (Wrong)

```text
Hardcoded Go defaults
        ↓
    seed.yaml (partial)
        ↓
Hardcoded TypeScript defaults
        ↓
    User profile (overrides only)
```

### Target Architecture (Correct)

```text
seed.yaml (system config only: port, paths, auth)
        ↓
Default Profile (ALL user-facing settings)
        ↓
User Profiles (full copies, not overrides)
```

### What seed.yaml Should Contain

Only system-level, non-user config:

- Server port, bind address
- Database path, retention
- Auth settings (JWT secret, etc.)
- Logging configuration
- File paths

### What Default Profile Should Contain

All user-facing settings:

- Thresholds (latency, packet loss, signal, etc.)
- Card visibility and display options
- Test settings (iperf, cable, link)
- Network discovery settings
- SNMP settings
- Unit system (Metric/SAE)
- Per-interface configurations (with per-interface thresholds and health checks)

### Actions Needed

1. **Create "System Default" profile on first startup** with all settings
2. **Move all `DEFAULT_*` constants** from TypeScript into the default profile JSON
3. **Move user-facing Go defaults** into the default profile
4. **Keep seed.yaml** for system-only config (port, paths, auth, logging)
5. **User profiles should be full copies**, not sparse overrides
6. **Frontend loads settings from active profile**, not from hardcoded constants
7. **Remove hardcoded fallbacks** - if no profile exists, create one from embedded defaults

---

## 5. App.tsx Monolith

**Status:** CRITICAL - Code Structure Issue **Priority:** High **Effort:** 1-2 days

### Issue Description

[App.tsx](web/src/App.tsx) is **2082 lines** and violates Single Responsibility Principle by handling:

1. **Authentication management** (login, logout, session expiration)
2. **WebSocket connection** (real-time data handling)
3. **Network interface monitoring** (dual interface state: Ethernet/WiFi)
4. **12+ fetch functions** (fetchLinkData, fetchIPConfig, fetchInterfaces, fetchVersion, fetchDiscoveryData,
   fetchDNSData, fetchVLANData, fetchGatewayData, fetchWiFiData, fetchCableData, fetchPublicIP, fetchNetworkDiscovery)
5. **Card state management** (CardState interface with 9 card types)
6. **Profile management** (interface selection per profile)
7. **FAB event handling** (runAllTests coordination)
8. **Setup wizard flow** (first-time configuration)
9. **LoginForm component** (embedded at lines 1765-2079, ~315 lines)
10. **Main dashboard layout** (4 sections: Connectivity, Network, Testing, System)
11. **Footer rendering**

### Current Structure

```text
App.tsx (2082 lines)
├── Imports (66 lines)
├── Types & Constants (24 lines)
├── App function (1415 lines)
│   ├── 20+ useState hooks
│   ├── 15+ useCallback hooks (fetch functions)
│   ├── 10+ useEffect hooks
│   └── JSX return (350 lines)
├── LoginForm component (315 lines)
└── Helper functions (15 lines)
```

### Target Structure

```text
src/
├── App.tsx                       # Thin orchestrator (~50-100 lines)
├── app/
│   ├── AppProviders.tsx          # Context providers wrapper
│   ├── AppContent.tsx            # Main dashboard layout
│   ├── AppFooter.tsx             # Footer component
│   ├── LoginForm.tsx             # Login UI (extracted)
│   ├── SetupCheck.tsx            # Setup wizard flow
│   └── index.ts
├── hooks/
│   ├── useCardState.ts           # Card state management
│   ├── useInterfaceState.ts      # Dual interface switching logic
│   ├── useNetworkFetchers.ts     # All 12 fetch functions
│   ├── useAutoTests.ts           # FAB event coordination
│   └── index.ts
```

### Specific Extractions

| Lines     | Component/Hook          | Description                   |
| --------- | ----------------------- | ----------------------------- |
| 1765-2079 | `LoginForm.tsx`         | Login form with SSO support   |
| 456-856   | `useNetworkFetchers.ts` | 12 fetch functions            |
| 205-280   | `useCardState.ts`       | CardState and interface state |
| 1186-1292 | `useAutoTests.ts`       | FAB runAllTests coordination  |
| 1507-1626 | `DashboardSections.tsx` | Four dashboard sections       |
| 1627-1724 | `AppFooter.tsx`         | Footer with contact/legal     |

---

## 6. Deprecated WebSocket Auth Fallback

**Status:** NEEDS REMOVAL **Priority:** Medium **Effort:** 30 minutes

**File:** [websocket.go:548](internal/api/websocket.go#L548)

The code contains a deprecated authentication fallback for WebSocket connections using `Sec-WebSocket-Protocol`. Comment
states: `// TODO: Remove this fallback after clients are updated (deprecated)`

**Issue:** Legacy authentication method still in production code. Should be removed once all clients migrate to
cookie-based auth.

**Action Needed:** Track client migration and remove deprecated fallback.

---

## 7. Silent Error Handling in Uninstall

**Status:** HIGH PRIORITY - Debugging Issue **Priority:** High **Effort:** 1 hour

**File:** [cmd_uninstall.go](cmd/seed/cmd_uninstall.go)

Multiple "best effort" operations silently ignore errors (lines 108, 109, 111, 112, 134, 136, 164):

```go
_ = exec.Command("systemctl", "stop", "seed").Run() //nolint:errcheck // Best effort
```

**Issue:** Silent failures during uninstall could leave system in inconsistent state. Users won't know why uninstall
partially failed.

**Action Needed:** Log errors even if not fatal, to help troubleshooting.

---

## 8. PoE Detection Not Implemented

**Status:** INCOMPLETE **Priority:** Low **Effort:** Requires hardware

**File:** [phy_linux.go:41](internal/phy/phy_linux.go#L41)

```go
// TODO: Add sysfs PoE detection when we have hardware to test
```

**Issue:** Power over Ethernet detection is not implemented on Linux.

**Action Needed:** Implement when PoE hardware is available for testing.

---

## 9. Panic Recovery Hides Errors

**Status:** MEDIUM PRIORITY **Priority:** Medium **Effort:** 30 minutes

**File:** [linkmon.go:168-176](internal/network/linkmon.go#L168-L176)

Link monitor callbacks silently recover from panics:

```go
recover() //nolint:errcheck // Intentionally ignoring panic value
```

**Issue:** Panics in callbacks are silently lost, making debugging difficult.

**Action Needed:** Log panic details before recovering.

---

## 10. Code Structure Inconsistency

**Status:** MEDIUM PRIORITY - Technical Debt **Priority:** Medium **Effort:** 3-5 days (incremental)

The codebase has inconsistent patterns for splitting large files. Some areas have been modularized while others remain
monolithic.

### Backend API Handlers (Go)

Split was started but not completed consistently:

| File                                                                | Lines | Status                 |
| ------------------------------------------------------------------- | ----- | ---------------------- |
| [handlers_health_checks.go](internal/api/handlers_health_checks.go) | 1548  | Massive - needs split  |
| [handlers_network.go](internal/api/handlers_network.go)             | 1391  | Massive - needs split  |
| [handlers_survey.go](internal/api/handlers_survey.go)               | 1136  | Large - consider split |
| [handlers_devices.go](internal/api/handlers_devices.go)             | 645   | Borderline             |
| handlers_discovery.go                                               | 200   | Good ✓                 |
| handlers_auth.go                                                    | 375   | Good ✓                 |

**Note:** [HANDLERS_SPLIT_STATUS.md](internal/api/HANDLERS_SPLIT_STATUS.md) documents incomplete split work from issue
#544.

### Frontend Settings (TypeScript)

Discovery was split properly, others were not:

| Component                                                                                 | Lines | Structure                                          |
| ----------------------------------------------------------------------------------------- | ----- | -------------------------------------------------- |
| [DiscoverySettings.tsx](web/src/components/settings/sections/DiscoverySettings.tsx)       | 186   | ✓ Split into `discovery/` subfolder (7 components) |
| [ThresholdsSettings.tsx](web/src/components/settings/sections/ThresholdsSettings.tsx)     | 881   | ❌ Monolithic                                      |
| [SNMPSettings.tsx](web/src/components/settings/sections/SNMPSettings.tsx)                 | 692   | ❌ Monolithic                                      |
| [PerformanceSettings.tsx](web/src/components/settings/sections/PerformanceSettings.tsx)   | 679   | ❌ Monolithic                                      |
| [HealthChecksSettings.tsx](web/src/components/settings/sections/HealthChecksSettings.tsx) | 628   | ❌ Monolithic                                      |

**SettingsDrawer.tsx is 60KB / ~1800 lines** - should be split like DiscoverySettings.

### Frontend Cards (TypeScript)

| Component                                                                     | Lines | Status                |
| ----------------------------------------------------------------------------- | ----- | --------------------- |
| [NetworkDiscoveryCard.tsx](web/src/components/cards/NetworkDiscoveryCard.tsx) | 1630  | Massive - needs split |
| [PerformanceCard.tsx](web/src/components/cards/PerformanceCard.tsx)           | 845   | Large                 |
| [LogViewerCard.tsx](web/src/components/cards/LogViewerCard.tsx)               | 783   | Large                 |
| [HealthCheckCard.tsx](web/src/components/cards/HealthCheckCard.tsx)           | 624   | Large                 |

### Recommended Pattern

Follow the DiscoverySettings pattern:

```text
settings/sections/
├── DiscoverySettings.tsx      # Main component (thin wrapper)
└── discovery/                  # Subfolder with focused components
    ├── DiscoveryToggles.tsx
    ├── DiscoveryTimingSettings.tsx
    ├── DiscoveryCustomOptions.tsx
    ├── SubnetManager.tsx
    └── index.ts
```

---

## Part 2: Implementation Plans

## Plan A: Quick Wins (1-2 days total)

Low-effort fixes that can be done immediately.

### A1. Add Network Settings Icon (15 min) ✅ COMPLETED

**Status:** Fixed in prior commit

**File:** [SettingsDrawer.tsx](web/src/components/settings/SettingsDrawer.tsx)

Network icon was already added to the Network settings section.

### A2. Log Errors in Uninstall (1 hour) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**File:** [cmd_uninstall.go](cmd/seed/cmd_uninstall.go)

Replaced all silent error handling (`_ = exec.Command(...)`) with proper `slog.Warn()` logging for
all best-effort operations during uninstall.

### A3. Log Panic Details in Link Monitor (30 min) ✅ COMPLETED

**Status:** Already implemented in prior commit

**File:** [linkmon.go:168-176](internal/network/linkmon.go#L168-L176)

Panic recovery now logs panic details with stack trace using `slog.Error`.

### A4. Remove Deprecated WebSocket Auth (30 min)

**File:** [websocket.go:548](internal/api/websocket.go#L548)

1. Remove the `Sec-WebSocket-Protocol` authentication fallback code
2. Update any documentation referencing this method
3. Ensure all clients use cookie-based auth

### A5. Consolidate Unit System Settings (2 hours)

**Files:**

- [settings.ts](web/src/types/settings.ts)
- Cable test components

Steps:

1. Remove `lengthUnit` from Cable Test settings
2. Update Cable Test UI to use global `unitSystem`
3. Update all unit conversions to reference single source

---

## Plan B: Extract LoginForm from App.tsx (2-4 hours)

First step in refactoring the App.tsx monolith.

### Steps

1. **Create new file:** `web/src/app/LoginForm.tsx`

2. **Move code (lines 1765-2079):**
   - `LoginFormProps` interface
   - `getAndClearSsoError()` function
   - `SSOProvider` interface
   - `LoginForm` component

3. **Update imports in LoginForm.tsx:**
   - Add necessary imports (useState, useEffect, useTranslation)
   - Import theme utilities

4. **Update App.tsx:**
   - Add import: `import { LoginForm } from "./app/LoginForm"`
   - Remove extracted code

5. **Test:**
   - Login functionality
   - SSO providers display
   - Error handling

---

## Plan C: Create Network Fetcher Hooks (4-6 hours)

Extract all fetch functions from App.tsx into dedicated hooks.

### Step 1: Create useNetworkFetchers.ts

**File:** `web/src/hooks/useNetworkFetchers.ts`

```typescript
export function useNetworkFetchers(currentInterfaceRef: React.RefObject<string>) {
  const fetchLinkData = useCallback(async () => { ... }, []);
  const fetchIPConfig = useCallback(async () => { ... }, []);
  const fetchInterfaces = useCallback(async () => { ... }, []);
  // ... all 12 fetch functions

  return {
    fetchLinkData,
    fetchIPConfig,
    fetchInterfaces,
    fetchVersion,
    fetchDiscoveryData,
    fetchDNSData,
    fetchVLANData,
    fetchGatewayData,
    fetchWiFiData,
    fetchCableData,
    fetchPublicIP,
    fetchNetworkDiscovery,
    triggerDeviceScan,
    changeInterface,
  };
}
```

### Step 2: Create useCardState.ts

**File:** `web/src/hooks/useCardState.ts`

```typescript
export function useCardState() {
  const [cards, setCards] = useState<CardState>({ ... });
  const handleCardUpdate = useCallback((update: CardUpdate) => { ... }, []);
  // Link-up detection logic

  return { cards, setCards, handleCardUpdate };
}
```

### Step 3: Create useInterfaceState.ts

**File:** `web/src/hooks/useInterfaceState.ts`

```typescript
export function useInterfaceState() {
  const [ethernetInterface, setEthernetInterface] = useState("");
  const [wifiInterface, setWifiInterface] = useState("");
  const [activeMode, setActiveMode] = useState<"ethernet" | "wifi">("ethernet");
  // Computed currentInterface, setCurrentInterface, setIsWifi

  return {
    ethernetInterface,
    wifiInterface,
    activeMode,
    currentInterface,
    isWifi,
    setCurrentInterface,
    setIsWifi,
    switchToInterfaceType,
  };
}
```

### Step 4: Update App.tsx

Replace inline hooks with:

```typescript
const { cards, setCards, handleCardUpdate } = useCardState();
const { currentInterface, isWifi, ... } = useInterfaceState();
const fetchers = useNetworkFetchers(currentInterfaceRef);
```

---

## Plan D: Multi-Interface Profile Support (1-2 days)

Enable profiles to contain multiple interfaces with per-interface settings.

### Phase 1: Backend Changes

**File:** [profile_settings.go](internal/config/profile_settings.go)

```go
// Update ProfileInterfaceConfigs
type ProfileInterfaceConfigs struct {
    Ethernet []ProfileInterfaceSelection `json:"ethernet,omitempty"`
    WiFi     []ProfileInterfaceSelection `json:"wifi,omitempty"`
}
```

Update all handlers that read/write interface configs.

### Phase 2: Frontend Types

**File:** [settings.ts](web/src/types/settings.ts)

```typescript
interface ProfileInterfaceConfigs {
  ethernet: ProfileInterfaceSelection[];
  wifi: ProfileInterfaceSelection[];
}
```

### Phase 3: Profile Editor UI

**File:** [ProfileEditor.tsx](web/src/components/profiles/ProfileEditor.tsx)

Add new sections:

1. Interface Selection - Multi-select for available interfaces
2. Per-Interface Settings - Expandable section for each selected interface
3. "Use Profile Defaults" toggle per interface

### Phase 4: App.tsx Integration

Update interface loading from profile to handle arrays instead of single objects.

---

## Plan E: Default Profile as Single Source of Truth (3-5 days)

The most significant architectural change.

### Phase 1: Define Embedded Default Profile (Day 1)

**Create:** `internal/config/default_profile.json`

```json
{
  "name": "System Default",
  "description": "Factory default settings",
  "config": {
    "thresholds": { ... },
    "cardSettings": { ... },
    "displayOptions": { ... },
    "networkDiscovery": { ... },
    "snmp": { ... },
    "unitSystem": "sae"
  }
}
```

### Phase 2: Backend Profile Initialization (Day 1-2)

**File:** `internal/database/profiles.go`

```go
func (db *Database) EnsureDefaultProfile() error {
    // Check if default profile exists
    // If not, create from embedded default_profile.json
    // Mark as isDefault=true, isSystem=true
}
```

Call during startup before any other profile operations.

### Phase 3: Remove Frontend Hardcoded Defaults (Day 2-3)

**File:** [settings.ts](web/src/types/settings.ts)

1. Remove all `DEFAULT_*` constants
2. Update `SettingsContext` to require profile data
3. Add loading state when profile not yet loaded
4. Handle error state if no profile available

### Phase 4: Update Settings Loading (Day 3-4)

**File:** [SettingsContext.tsx](web/src/contexts/SettingsContext.tsx)

```typescript
// Remove: const settings = { ...DEFAULT_SETTINGS, ...profileSettings }
// Use: const settings = activeProfile.config  // Full profile, not merged
```

### Phase 5: Remove Backend Hardcoded Defaults (Day 4-5)

For each `Default*()` function in Go:

1. Keep for system-level configs (logging, auth, rate limits)
2. Remove for user-facing configs (thresholds, discovery settings)
3. User-facing configs come from active profile only

### Phase 6: Migration Support

Create migration to convert existing sparse profiles to full profiles:

```go
func MigrateProfilesToFullCopy() error {
    // For each user profile without full config:
    //   1. Load system default profile
    //   2. Merge user overrides
    //   3. Save as full profile
}
```

---

## Plan F: Code Structure Consistency (3-5 days, incremental)

Split large files following the DiscoverySettings pattern.

### Priority Order

| #   | Target                    | Lines | New Structure                 |
| --- | ------------------------- | ----- | ----------------------------- |
| 1   | App.tsx                   | 2082  | app/ folder (Plans B, C)      |
| 2   | NetworkDiscoveryCard.tsx  | 1630  | cards/network-discovery/      |
| 3   | handlers_health_checks.go | 1548  | api/health_checks/            |
| 4   | handlers_network.go       | 1391  | api/network/                  |
| 5   | SettingsDrawer.tsx        | 1800  | settings/drawer/              |
| 6   | ThresholdsSettings.tsx    | 881   | settings/sections/thresholds/ |

### Pattern for Each Split

**Frontend (TypeScript):**

```text
Component.tsx (large)
    ↓
component/
├── Component.tsx          # Thin wrapper, imports sub-components
├── ComponentSection1.tsx  # Focused sub-component
├── ComponentSection2.tsx  # Focused sub-component
├── hooks/                 # Component-specific hooks
│   └── useComponentLogic.ts
├── types.ts               # Component-specific types
└── index.ts               # Exports
```

**Backend (Go):**

```text
handlers_feature.go (large)
    ↓
feature/
├── handlers.go            # HTTP handlers
├── types.go               # Request/response types
├── service.go             # Business logic
└── README.md              # Package documentation
```

### Add Tooling Enforcement

**ESLint (.eslintrc.js):**

```javascript
rules: {
  'max-lines': ['warn', { max: 400, skipBlankLines: true, skipComments: true }]
}
```

**golangci-lint (.golangci.yml):**

```yaml
linters-settings:
  funlen:
    lines: 100
  lll:
    line-length: 120
```

---

## Part 3: Summary

## Issue Priority Matrix

| Issue                           | Status                | Priority | Effort   | Plan |
| ------------------------------- | --------------------- | -------- | -------- | ---- |
| Default Profile Single Source   | Architecture Overhaul | Critical | 3-5 days | E    |
| App.tsx Monolith                | Code Structure        | High     | 1-2 days | B, C |
| Profile Multi-Interface Support | Limited to 1 each     | High     | 1-2 days | D    |
| Silent Errors in Uninstall      | Needs Logging         | High     | 1 hour   | A2   |
| Code Structure Inconsistency    | Technical Debt        | Medium   | 3-5 days | F    |
| Metric/SAE Settings Duplication | Architecture Issue    | Medium   | 2 hours  | A5   |
| Deprecated WebSocket Auth       | Needs Removal         | Medium   | 30 min   | A4   |
| Panic Recovery Hides Errors     | Needs Logging         | Medium   | 30 min   | A3   |
| Network Settings Icon           | Missing               | Low      | 15 min   | A1   |
| PoE Detection                   | Not Implemented       | Low      | Needs HW | -    |

## Recommended Execution Order

1. **Week 1 - Quick Wins:** Complete Plan A (all items)
2. **Week 1-2 - App.tsx Refactor:** Complete Plans B and C
3. **Week 2 - Multi-Interface:** Complete Plan D
4. **Week 3-4 - Single Source of Truth:** Complete Plan E
5. **Ongoing - Code Structure:** Complete Plan F incrementally

## Dependencies

```text
Plan A (Quick Wins) → No dependencies, do first
Plan B (LoginForm) → No dependencies
Plan C (Fetcher Hooks) → Depends on B being stable
Plan D (Multi-Interface) → Can parallel with B/C
Plan E (Single Source) → Depends on D for full profile structure
Plan F (Code Split) → Can start after B/C, parallel with D/E
```

---

## Part 4: Additional Findings - Security, Defects & Incomplete Plumbing

### CRITICAL: Settings Save Failures Return 200 OK

**File:** [handlers_settings.go:255-257](internal/api/handlers_settings.go#L255-L257)

```go
if err := s.db.Profiles().Update(ctx, profile); err != nil {
    logger.Warn("Failed to save settings to profile", "error", err, "profile_id", profile.ID)
    return  // No HTTP error response!
}
```

**Issue:** When profile save fails, the handler returns without sending an error response. Client receives implicit 200
OK and believes settings were saved.

**Impact:** Users modify settings, see success, but changes are lost on restart.

**Fix:** Return HTTP 500 error:

```go
if err := s.db.Profiles().Update(ctx, profile); err != nil {
    logger.Error("Failed to save settings to profile", "error", err, "profile_id", profile.ID)
    sendJSONResponse(w, logger, http.StatusInternalServerError, map[string]string{
        "error": "Failed to save settings",
    })
    return
}
```

---

### CRITICAL: Lock/Unlock Without Defer (Deadlock Risk)

**File:** [handlers_auth.go:351-353](internal/api/handlers_auth.go#L351-L353)

```go
s.config.Lock()
s.config.Auth.DefaultPasswordHash = hash
s.config.Unlock()  // Not protected by defer!
```

**Issue:** If panic occurs between Lock and Unlock, deadlock occurs and service becomes unresponsive to all requests.

**Also found in:** [handlers_security.go:189-201](internal/api/handlers_security.go#L189-L201) (rogue DHCP config
update)

**Fix:** Always use defer:

```go
s.config.Lock()
defer s.config.Unlock()
s.config.Auth.DefaultPasswordHash = hash
```

---

### HIGH: Type Assertion Failures Silently Ignored

**File:** [handlers_settings.go:263-388](internal/api/handlers_settings.go#L263-L388)

```go
thresholds, ok := updates["thresholds"].(map[string]interface{})
if !ok {
    return  // Silent failure - no error to client
}
```

**Issue:** Throughout the settings update logic, type assertion failures cause silent returns. Malformed JSON is
silently ignored without client feedback.

**Impact:** Attackers or buggy clients can send invalid data; no indication settings weren't applied.

**Affected Functions:**

- `applyThresholdUpdates`
- `applyDNSThresholds`
- `applyGatewayThresholds`
- `applyWiFiThresholds`
- `applyCustomTestThresholds`
- `applyHTTPTimingThresholds`

**Fix:** Add validation layer with proper error responses before type assertions.

---

### HIGH: Settings Not Plumbed Through to Backend

**File:** [SettingsDrawer.tsx:1250](web/src/components/settings/SettingsDrawer.tsx#L1250)

```typescript
// TODO: Implement saveLinkSettings when backend API is ready
```

**File:** [SettingsDrawer.tsx:1265](web/src/components/settings/SettingsDrawer.tsx#L1265)

```typescript
// TODO: Implement saveCableTestSettings when backend API is ready
```

**Issue:** UI allows users to modify Link Settings and Cable Test Settings, but these only update local state. Changes
are NOT persisted to backend.

**Impact:** Users modify settings, changes appear to work, but revert on page reload.

**Related Issues:** #734, #740

**Fix:** Implement backend API endpoints `/api/settings/link` and `/api/settings/cable` and wire to frontend.

---

### MEDIUM: Cleanup Goroutine Without Shutdown Coordination

**File:** [csrf.go:57](internal/auth/csrf.go#L57)

```go
go manager.cleanupExpiredTokens()
```

**Issue:** Background cleanup goroutine is started but has no mechanism to stop on shutdown. Could attempt cleanup after
manager is destroyed.

**Fix:** Add context cancellation:

```go
func NewCSRFManager(ctx context.Context) *CSRFManager {
    m := &CSRFManager{...}
    go m.cleanupExpiredTokens(ctx)
    return m
}
```

---

### MEDIUM: X-Forwarded-For Trusted in Logging But Not Rate Limiter

**File:** [ratelimit.go:216](internal/api/ratelimit.go#L216) - GOOD:

```go
// SECURITY: Always uses RemoteAddr to prevent X-Forwarded-For spoofing
```

**File:** [redact.go:175-193](internal/logging/redact.go#L175-L193) - INCONSISTENT:

- Rate limiter correctly ignores X-Forwarded-For (secure)
- Logging code still logs X-Forwarded-For directly (can be spoofed)

**Issue:** Inconsistent security posture. In reverse proxy scenarios, logs could show spoofed IPs leading to incorrect
forensics.

**Fix:** Clearly document in logs when X-Forwarded-For is untrusted, or configure trusted proxies.

---

### MEDIUM: Panic on Crypto Failure

**File:** [auth.go:86](internal/auth/auth.go#L86)

```go
panic("crypto/rand failed: " + err.Error() + " - system is insecure, cannot continue")
```

**Issue:** While technically correct (crypto failure is unrecoverable), this crashes the entire application with no
restart mechanism.

**Impact:** Could be leveraged for denial-of-service if /dev/urandom becomes unavailable.

**Fix:** Consider graceful degradation with service restart, or systemd restart policy.

---

### MEDIUM: Inconsistent Request Body Size Limits

**File:** [handlers_auth.go:57](internal/api/handlers_auth.go#L57)

```go
http.MaxBytesReader(w, r.Body, 1024)  // Hardcoded
```

**Other handlers:** Use `MaxBodySizeJSON` constant

**Issue:** Inconsistent limits across endpoints. Some handlers use named constants, others use magic numbers.

**Fix:** Define and use consistent constants across all handlers:

```go
const (
    MaxBodySizeSmall = 1024      // Login, simple requests
    MaxBodySizeJSON  = 65536     // Settings, complex data
    MaxBodySizeLarge = 10485760  // File uploads
)
```

---

### MEDIUM: Frontend-Backend Type Mismatch

**Backend** [handlers_settings.go:52-93](internal/api/handlers_settings.go#L52-L93):

- Returns thresholds as `map[string]int64` (milliseconds)

**Frontend** [settings.ts:18-35](web/src/types/settings.ts#L18-L35):

- Defines `ThresholdPair { good: number; warning: number }`
- No explicit time unit specification

**Issue:** No shared type definitions. Backend sends milliseconds as int64, frontend expects generic numbers. Changes
require manual coordination.

**Fix:** Consider code generation from OpenAPI spec or shared schema definition.

---

### LOW: Database Tests Ignore Errors

**File:** [database_test.go](internal/database/database_test.go) - Lines 201, 222, 344, 434, 512, 523, 659

```go
_, _ = repo.Get(ctx, ...)  // Error ignored
```

**Issue:** Test setup ignores errors which could mask real issues. If test data fails to load, tests continue with nil
values.

**Fix:** Use `require.NoError(t, err)` in test setup.

---

### LOW: Empty Interface Usage Throughout

**File:** [handlers_status.go](internal/api/handlers_status.go) - Lines 32, 96, 119, etc.

Extensive use of `map[string]interface{}` creates type safety issues.

**Fix:** Define proper struct types for API responses.

---

## Updated Summary Table

| Issue                                | Status      | Priority | Effort    | Plan |
| ------------------------------------ | ----------- | -------- | --------- | ---- |
| Settings save returns 200 on failure | ✅ COMPLETE | Critical | 30 min    | G1   |
| Lock/Unlock without defer            | ✅ COMPLETE | Critical | 30 min    | G2   |
| Default Profile Single Source        | Documented  | Critical | 3-5 days  | E    |
| Type assertions silently fail        | ✅ COMPLETE | High     | 2-3 hours | G3   |
| Settings not plumbed to backend      | NEW         | High     | 1 day     | G4   |
| App.tsx Monolith                     | Documented  | High     | 1-2 days  | B, C |
| Profile Multi-Interface Support      | Documented  | High     | 1-2 days  | D    |
| Silent Errors in Uninstall           | ✅ COMPLETE | High     | 1 hour    | A2   |
| Cleanup goroutine no shutdown        | ✅ COMPLETE | Medium   | 1 hour    | G5   |
| X-Forwarded-For inconsistent         | ✅ COMPLETE | Medium   | 1 hour    | G6   |
| Crypto panic no recovery             | ✅ COMPLETE | Medium   | 2 hours   | G7   |
| Request body size inconsistent       | ✅ COMPLETE | Medium   | 1 hour    | G8   |
| Frontend-backend type mismatch       | NEW         | Medium   | 2-3 hours | G9   |
| Code Structure Inconsistency         | Documented  | Medium   | 3-5 days  | F    |
| Metric/SAE Settings Duplication      | ✅ COMPLETE | Medium   | 2 hours   | A5   |
| Deprecated WebSocket Auth            | ✅ COMPLETE | Medium   | 30 min    | A4   |
| Panic Recovery Hides Errors          | ✅ COMPLETE | Medium   | 30 min    | A3   |
| Database tests ignore errors         | ✅ COMPLETE | Low      | 1 hour    | G10  |
| Empty interface usage                | NEW         | Low      | Ongoing   | -    |
| Network Settings Icon                | ✅ COMPLETE | Low      | 15 min    | A1   |
| PoE Detection                        | Documented  | Low      | Needs HW  | -    |

---

## Plan G: New Critical Fixes

### G1. Fix Settings Save Error Response (30 min) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**File:** [handlers_settings.go](internal/api/handlers_settings.go)

Changed `saveSettingsToActiveProfile` to return `error` and the caller now returns HTTP 500 on
failure instead of implicit 200.

### G2. Add Defer to Lock/Unlock (30 min) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**Files:**

- [handlers_auth.go:351](internal/api/handlers_auth.go#L351)
- [handlers_security.go:189](internal/api/handlers_security.go#L189)

Added `defer s.config.Unlock()` after each `s.config.Lock()` to prevent deadlocks.

### G3. Add Validation Before Type Assertions (2-3 hours)

**File:** [handlers_settings.go](internal/api/handlers_settings.go)

Create validation middleware that checks JSON structure before handlers process it. Return 400 Bad Request with details
on validation failure.

### G4. Implement Link/Cable Settings Backend (1 day)

**Files:**

- Create `/api/settings/link` endpoint
- Create `/api/settings/cable` endpoint
- Wire to frontend auto-save handlers

### G5. Add Shutdown Coordination to CSRF Cleanup (1 hour)

**File:** [csrf.go](internal/auth/csrf.go)

Pass context to cleanup goroutine, cancel on shutdown.

### G6. Document X-Forwarded-For Trust Model (1 hour)

**File:** [redact.go](internal/logging/redact.go)

Add comments/logs indicating X-Forwarded-For is untrusted. Consider adding configurable trusted proxy list.

### G7. Add Crypto Failure Recovery (2 hours)

**File:** [auth.go](internal/auth/auth.go)

Replace panic with structured error that triggers graceful service restart.

### G8. Standardize Request Body Limits (1 hour)

**File:** Create `internal/api/limits.go`

Define constants and apply consistently across all handlers.

### G9. Document Type Contracts (2-3 hours)

Create shared documentation or OpenAPI spec defining exact types between frontend and backend. Consider future code
generation.

### G10. Fix Test Error Handling (1 hour) ✅ COMPLETED

**Status:** Fixed in commit f11979b

**File:** [database_test.go](internal/database/database_test.go)

Replaced all 10 instances of `_, _ =` with proper error assertions using `require.NoError()`. Tests now
fail immediately with clear error messages if setup operations fail, preventing silent failures and
improving test reliability.

---

## Updated Execution Order

1. **Immediate (This Week):** Plan G1, G2 (Critical fixes - 1 hour total)
2. **Week 1:** Plan A (Quick Wins) + G3, G5-G10 (Medium fixes)
3. **Week 1-2:** Plans B and C (App.tsx Refactor) + G4 (Settings plumbing)
4. **Week 2:** Plan D (Multi-Interface)
5. **Week 3-4:** Plan E (Single Source of Truth)
6. **Ongoing:** Plan F (Code Structure)

---

## Part 5: Deployment Architecture & Additional Security Findings

### Deployment: Nginx Reverse Proxy Consideration

**Status:** ARCHITECTURE DECISION **Priority:** Medium **Effort:** 4-6 hours (if implementing)

#### Current State

Go's `net/http` server handles everything directly:

- TLS termination
- Rate limiting (correctly uses `RemoteAddr`, ignores `X-Forwarded-For`)
- Authentication (JWT + cookies)
- CORS handling
- Static file serving

#### When to Use Nginx

| Deployment Scenario                   | Nginx Recommended? |
| ------------------------------------- | ------------------ |
| Enterprise/cloud deployment           | ✅ Yes             |
| Multiple instances (load balancing)   | ✅ Yes             |
| Let's Encrypt auto-renewal needed     | ✅ Yes             |
| Public internet facing                | ✅ Yes             |
| Embedded device/appliance             | ❌ Overkill        |
| Internal network only                 | ❌ Not necessary   |
| Single instance, resource constrained | ❌ No              |

#### If Adding Nginx - Required Changes

**1. Add trusted proxy configuration:**

```go
// internal/api/proxy.go
var trustedProxies = []string{"127.0.0.1", "::1"}

func getRealIP(r *http.Request) string {
    if isTrustedProxy(r.RemoteAddr) {
        if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
            return strings.Split(xff, ",")[0]
        }
    }
    return r.RemoteAddr
}
```

**2. Add CLI flag:**

```bash
# Direct (default - suitable for appliances)
seed serve --port 443 --tls

# Behind nginx (enterprise deployments)
seed serve --port 8080 --trusted-proxies 127.0.0.1
```

**3. Example nginx config:**

```nginx
location / {
    proxy_pass http://127.0.0.1:8080;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header Host $host;

    # WebSocket support
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
}
```

---

### MEDIUM: Session ID Extraction Trusts X-Username Header

**File:** [csrf.go:170-174](internal/auth/csrf.go#L170-L174)

```go
sessionID := r.Header.Get("X-Username")
if sessionID == "" {
    sessionID = getSessionIDFromRequest(r)
}
```

**Issue:** CSRF session extraction trusts client-supplied `X-Username` header. If an attacker controls this header, they
could potentially fixate a session.

**Fix:** Extract session ID only from verified JWT or secure httpOnly cookies, never from client-supplied headers.

---

### MEDIUM: No WebSocket Message Rate Limiting

**File:** [websocket.go](internal/api/websocket.go)

**Issue:** While login and endpoint rate limiting exists, there's no per-connection message rate limiting on WebSocket
connections.

**Risk:** Could allow resource exhaustion via rapid message flooding from a single connection.

**Fix:** Add per-connection rate limiter:

```go
type wsConn struct {
    conn      *websocket.Conn
    limiter   *rate.Limiter  // e.g., 100 messages/second
}

func (c *wsConn) ReadMessage() (int, []byte, error) {
    if !c.limiter.Allow() {
        return 0, nil, ErrRateLimited
    }
    return c.conn.ReadMessage()
}
```

---

### MEDIUM: SNMP Allows MD5 Authentication

**File:** [config.go:578](internal/config/config.go#L578)

```go
AuthProtocol  string `yaml:"auth_protocol"`  // "MD5", "SHA", "SHA256", "SHA512"
```

**Also:** [snmp.go:311-312](internal/snmp/snmp.go#L311-L312)

**Issue:** SNMP v3 still allows MD5 as auth protocol. MD5 is cryptographically broken.

**Fix:** Deprecate MD5/SHA support, require SHA256+ for SNMP v3. Add migration warning for existing configs using MD5.

---

### LOW: Report Renderer - Fragile XSS Prevention

**File:** [reportRenderer.ts:331-334](web/src/utils/reportRenderer.ts#L331-L334)

```typescript
function escapeHtml(text: string): string {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}
```

**Issue:** While currently safe, this pattern is unusual and fragile. If someone modifies to use `.innerHTML` instead of
`.textContent`, it becomes an XSS vector.

**Fix:** Use explicit entity encoding:

```typescript
function escapeHtml(text: string): string {
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}
```

---

### LOW: Crypto Randomness Fallback in Request IDs

**File:** [middleware.go:56-58](internal/logging/middleware.go#L56-L58)

```go
if _, err := rand.Read(b); err != nil {
    // Fallback to timestamp-based ID if crypto/rand fails
    return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000000")))[:16]
}
```

**Issue:** If `crypto/rand` fails, falls back to predictable timestamp-based IDs. Crypto failure usually indicates
serious system issues.

**Fix:** Either panic (consistent with auth.go:86) or log critical warning. Don't silently degrade.

---

### LOW: Error Details Leaked to Clients

**File:** [handlers_devices.go:472](internal/api/handlers_devices.go#L472)

```go
sendErrorResponseWithDetails(w, logger, http.StatusBadRequest, ErrCodeBadRequest, "Invalid CIDR format", err.Error())
```

**Issue:** Some error handlers pass raw `err.Error()` to clients, potentially leaking internal implementation details.

**Fix:** Create sanitized error messages for all user-facing errors. Log full error server-side, return generic message
to client.

---

### LOW: Unused CSRF Cookie Function

**File:** [csrf.go:230-245](internal/auth/csrf.go#L230-L245)

```go
// Note: This function is currently unused.
func SetCSRFCookie(w http.ResponseWriter, token string, secure bool) {
```

**Issue:** Dead code that could cause confusion or be accidentally enabled.

**Fix:** Remove unused function or document why it's retained.

---

### Missing Security Features (Future Consideration)

| Feature               | Priority | Notes                                                   |
| --------------------- | -------- | ------------------------------------------------------- |
| Audit logging         | Medium   | Log password changes, privilege changes, login attempts |
| API key rotation      | Low      | NVD API key has no rotation mechanism                   |
| CSP nonce support     | Low      | For report renderer inline scripts                      |
| Subresource Integrity | Low      | If using CDN for any assets                             |

---

## Final Summary Table

| Issue                                | Status      | Priority | Effort    | Plan |
| ------------------------------------ | ----------- | -------- | --------- | ---- |
| Settings save returns 200 on failure | ✅ COMPLETE | Critical | 30 min    | G1   |
| Lock/Unlock without defer            | ✅ COMPLETE | Critical | 30 min    | G2   |
| Default Profile Single Source        | Documented  | Critical | 3-5 days  | E    |
| Type assertions silently fail        | ✅ COMPLETE | High     | 2-3 hours | G3   |
| Settings not plumbed to backend      | NEW         | High     | 1 day     | G4   |
| App.tsx Monolith                     | Documented  | High     | 1-2 days  | B, C |
| Profile Multi-Interface Support      | Documented  | High     | 1-2 days  | D    |
| Silent Errors in Uninstall           | ✅ COMPLETE | High     | 1 hour    | A2   |
| Session ID trusts X-Username         | ✅ COMPLETE | Medium   | 1 hour    | H1   |
| No WebSocket rate limiting           | ✅ COMPLETE | Medium   | 2 hours   | H2   |
| SNMP allows MD5                      | ✅ COMPLETE | Medium   | 1 hour    | H3   |
| Cleanup goroutine no shutdown        | ✅ COMPLETE | Medium   | 1 hour    | G5   |
| X-Forwarded-For inconsistent         | ✅ COMPLETE | Medium   | 1 hour    | G6   |
| Crypto panic no recovery             | ✅ COMPLETE | Medium   | 2 hours   | G7   |
| Request body size inconsistent       | ✅ COMPLETE | Medium   | 1 hour    | G8   |
| Frontend-backend type mismatch       | NEW         | Medium   | 2-3 hours | G9   |
| Trusted proxy support (nginx)        | NEW         | Medium   | 4-6 hours | H4   |
| Code Structure Inconsistency         | Documented  | Medium   | 3-5 days  | F    |
| Metric/SAE Settings Duplication      | ✅ COMPLETE | Medium   | 2 hours   | A5   |
| Deprecated WebSocket Auth            | ✅ COMPLETE | Medium   | 30 min    | A4   |
| Panic Recovery Hides Errors          | ✅ COMPLETE | Medium   | 30 min    | A3   |
| Report renderer XSS pattern          | ✅ COMPLETE | Low      | 30 min    | H5   |
| Crypto fallback in request IDs       | ✅ COMPLETE | Low      | 30 min    | H6   |
| Error details leaked to clients      | ✅ COMPLETE | Low      | 1 hour    | H7   |
| Unused CSRF cookie function          | ✅ COMPLETE | Low      | 15 min    | H8   |
| Database tests ignore errors         | ✅ COMPLETE | Low      | 1 hour    | G10  |
| Empty interface usage                | NEW         | Low      | Ongoing   | -    |
| Network Settings Icon                | ✅ COMPLETE | Low      | 15 min    | A1   |
| PoE Detection                        | Documented  | Low      | Needs HW  | -    |

---

## Plan H: Security Hardening

### H1. Fix Session ID Extraction (1 hour)

**File:** [csrf.go](internal/auth/csrf.go)

Remove trust of `X-Username` header. Extract session ID only from verified JWT claims.

### H2. Add WebSocket Rate Limiting (2 hours)

**File:** [websocket.go](internal/api/websocket.go)

Add per-connection message rate limiter (e.g., 100 msg/sec with burst of 20).

### H3. Deprecate MD5 for SNMP (1 hour)

**Files:** [config.go](internal/config/config.go), [snmp.go](internal/snmp/snmp.go)

Log deprecation warning if MD5 configured. Plan removal in next major version.

### H4. Add Trusted Proxy Support (4-6 hours)

**Files:** Create `internal/api/proxy.go`, update CLI flags

Enable proper X-Forwarded-For handling when behind nginx/load balancer.

### H5. Fix Report Renderer XSS Pattern (30 min) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**File:** [reportRenderer.ts](web/src/utils/reportRenderer.ts)

Replaced DOM-based `escapeHtml` with explicit entity encoding using string replacements for
`&`, `<`, `>`, `"`, and `'`.

### H6. Remove Crypto Fallback (30 min) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**File:** [middleware.go](internal/logging/middleware.go)

Changed to panic on crypto/rand failure with `slog.Error` logging instead of silently falling
back to predictable timestamp-based IDs.

### H7. Sanitize Error Messages (1 hour)

**File:** [handlers_devices.go](internal/api/handlers_devices.go) and others

Audit all `sendErrorResponseWithDetails` calls, ensure no raw errors exposed.

### H8. Remove Dead CSRF Code (15 min) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**File:** [csrf.go](internal/auth/csrf.go)

Removed unused `SetCSRFCookie` function (16 lines of dead code).

---

## Part 7: Discovery Card Issues

### BUG: ICMP-Discovered Devices Incorrectly Marked as Extended

**Status:** BUG **Priority:** High **Effort:** 1 hour

**File:** [arp.go:280](internal/discovery/arp.go#L280)

```go
// Current code - WRONG
IsLocal: false, // Remote subnet - not local
```

**Issue:** The code marks ALL ICMP-discovered devices as `IsLocal = false`, but this is incorrect. A device on the local
subnet might respond to ICMP ping but not be in the ARP cache because:

- ARP entry expired
- Device firewall blocks ARP but allows ICMP
- Device was just powered on and hasn't been ARP-resolved yet

**Impact:** Local devices incorrectly appear in "Extended Networks" section instead of "Local Network" section.

**Fix:** Check if the device's IP is actually within the local subnet before setting `IsLocal`:

```go
// Check if IP is in local subnet
isInLocalSubnet := isIPInSubnet(ip, localSubnet)
entries = append(entries, &ARPEntry{
    IP:       ip,
    MAC:      "", // No MAC for ICMP-only hosts
    State:    "PING_ONLY",
    LastSeen: time.Now(),
    IsLocal:  isInLocalSubnet, // Correct: based on actual subnet membership
})
```

---

### UI: Discovery Card Header Shows Too Much Info

**Status:** UI/UX Issue **Priority:** Medium **Effort:** 30 min

**File:** [NetworkDiscoveryCard.tsx:614-641](web/src/components/cards/NetworkDiscoveryCard.tsx#L614-L641)

**Current display:**

- Subnet (e.g., 192.168.64.0/24)
- Devices found count
- Local address (e.g., 192.168.64.7)
- Discovery interface (e.g., enp0s1)

**Problem:** Too much technical detail. Users primarily care about what was found, not implementation details.

**Requested display:**

- Subnets scanned (count or list if ≤5)
- Devices found count

**Fix:** Simplify `DiscoverySummary` component:

```tsx
{
  /* Simplified network info row */
}
<div className="flex items-center justify-between caption text-text-muted">
  <span>
    {subnetCount === 1 ? calculateNetworkAddress(status.subnet) : t("discovery.subnetsScanned", { count: subnetCount })}
  </span>
  <span>
    {deviceCount === 1
      ? t("discovery.deviceFound", { count: deviceCount })
      : t("discovery.devicesFound", { count: deviceCount })}
  </span>
</div>;
```

Remove or collapse the "Local address" and "Discovery interface" rows - they're implementation details, not user-facing
info.

---

### UI: Subnet List Should Rollup When >5 Subnets

**Status:** UI/UX Enhancement **Priority:** Low **Effort:** 1 hour

**File:** [NetworkDiscoveryCard.tsx](web/src/components/cards/NetworkDiscoveryCard.tsx)

**Issue:** When scanning multiple subnets (via Discovery.AdditionalSubnets config), the UI currently doesn't handle long
subnet lists gracefully.

**Requested behavior:**

- If ≤5 subnets: Show all subnet CIDRs
- If >5 subnets: Collapse to "X subnets scanned" with expandable dropdown to see all

**Fix:** Add subnet rollup logic:

```tsx
function SubnetList({ subnets }: { subnets: string[] }) {
  const [expanded, setExpanded] = useState(false);

  if (subnets.length <= 5) {
    return <span className="font-mono">{subnets.join(", ")}</span>;
  }

  return (
    <div>
      <button onClick={() => setExpanded(!expanded)}>
        {t("discovery.subnetsScanned", { count: subnets.length })}
        <ChevronDown className={expanded ? "rotate-180" : ""} />
      </button>
      {expanded && (
        <div className="stack-xs mt-2">
          {subnets.map((subnet) => (
            <span key={subnet} className="font-mono caption">
              {subnet}
            </span>
          ))}
        </div>
      )}
    </div>
  );
}
```

**Note:** This requires backend changes to return array of scanned subnets in `DiscoveryStatus`, not just single
`subnet` field.

---

### BUG: System Health Card 401 Flakiness

**Status:** BUG **Priority:** Medium **Effort:** 1-2 hours

**File:** [SystemHealthCard.tsx:167-174](web/src/components/cards/SystemHealthCard.tsx#L167-L174)

```tsx
if (!response.ok) {
  throw new Error(`HTTP ${response.status}`);
}
```

**Issue:** The System Health card goes into error state on any non-2xx response, including 401 (session expired). Unlike
other cards that may handle 401 gracefully by triggering re-auth, this card just shows a generic error.

**Symptoms:**

- Card shows error state when session token expires
- User sees "HTTP 401" error instead of being prompted to re-login
- Card doesn't recover automatically after session refresh

**Fix:** Handle 401 specifically to trigger session refresh:

```tsx
if (response.status === 401) {
  // Trigger session refresh or redirect to login
  window.dispatchEvent(new CustomEvent("session-expired"));
  return;
}
if (!response.ok) {
  throw new Error(`HTTP ${response.status}`);
}
```

---

### ENHANCEMENT: Show What's Causing High Resource Usage

**Status:** Enhancement **Priority:** Medium **Effort:** 4-6 hours

**Files:**

- Backend: [system.go](internal/system/system.go)
- Frontend: [SystemHealthCard.tsx](web/src/components/cards/SystemHealthCard.tsx)

**Issue:** When CPU, memory, or disk usage is high, users only see a percentage. They have no visibility into:

- What processes are consuming resources
- What directories are using disk space
- What actions they can take

**Requested behavior:**

When a resource exceeds warning threshold (75%), show:

1. **Top consumers** - List top 3-5 processes/directories using that resource
2. **Remediation suggestions** - Actionable tips based on what's consuming resources

**Example UI when memory is high:**

```text
Memory: 87% ⚠️
├── Top consumers:
│   - seed (245 MB) - This application
│   - postgres (890 MB) - Database
│   - node (312 MB) - Frontend dev server
│
└── Suggestions:
    - Consider increasing system memory
    - Restart postgres to clear connection pool
```

**Backend changes needed:**

```go
// Add to Health struct
TopCPUProcesses    []ProcessInfo `json:"topCpuProcesses,omitempty"`
TopMemoryProcesses []ProcessInfo `json:"topMemoryProcesses,omitempty"`
TopDiskDirectories []DirUsage    `json:"topDiskDirectories,omitempty"`

type ProcessInfo struct {
  Name       string  `json:"name"`
  PID        int     `json:"pid"`
  CPUPercent float64 `json:"cpuPercent"`
  MemoryMB   float64 `json:"memoryMb"`
}
```

**Note:** Use `gopsutil/process` for process info. Only populate when thresholds exceeded to avoid overhead.

---

## Plan I: Discovery UI/UX Fixes

### I1. Fix ICMP IsLocal Bug (1 hour) ✅ COMPLETED

**Status:** Fixed in commit 41d9a3c

**File:** [arp.go](internal/discovery/arp.go)

Fixed `IsLocal` logic to check subnet membership rather than relying on ping result. Also added
graceful fallback when ICMP ping fails due to missing CAP_NET_RAW capability.

### I2. Simplify Discovery Card Header (30 min)

**File:** [NetworkDiscoveryCard.tsx](web/src/components/cards/NetworkDiscoveryCard.tsx)

Remove local address and interface from summary. Show only subnets and device count.

### I3. Add Subnet Rollup (1 hour)

**Files:**

- Backend: Add `subnets []string` to DiscoveryStatus response
- Frontend: Add SubnetList component with collapsible behavior

---

## Plan J: System Health Improvements

### J1. Fix 401 Handling in SystemHealthCard (1-2 hours) ✅ COMPLETED

**Status:** Fixed in commit b3c3c22

**File:** [SystemHealthCard.tsx](web/src/components/cards/SystemHealthCard.tsx)

Added 401 handling that dispatches `session-expired` custom event instead of showing error state.

### J2. Add Top Resource Consumers (4-6 hours)

**Files:**

- Backend: [system.go](internal/system/system.go) - Add process enumeration via gopsutil
- Frontend: [SystemHealthCard.tsx](web/src/components/cards/SystemHealthCard.tsx) - Display top consumers when threshold
  exceeded

Show top 3-5 processes by CPU/memory when usage exceeds 75%.

### J3. Add Remediation Suggestions (2 hours)

**File:** [SystemHealthCard.tsx](web/src/components/cards/SystemHealthCard.tsx)

Add contextual suggestions based on what's consuming resources (e.g., "Restart postgres to clear connection pool").

---

## Updated Summary Table (Including Discovery & System Health Issues)

| Issue                           | Status      | Priority | Effort    | Plan |
| ------------------------------- | ----------- | -------- | --------- | ---- |
| ICMP devices marked as extended | ✅ COMPLETE | High     | 1 hour    | I1   |
| System Health 401 flakiness     | ✅ COMPLETE | Medium   | 1-2 hours | J1   |
| Discovery header too verbose    | ✅ COMPLETE | Medium   | 30 min    | I2   |
| Show top resource consumers     | ✅ COMPLETE | Medium   | 4-6 hours | J2   |
| Remediation suggestions         | ✅ COMPLETE | Medium   | 2 hours   | J3   |
| Subnet list needs rollup        | ✅ COMPLETE | Low      | 1 hour    | I3   |

---

## Final Execution Order

1. **Immediate:** G1, G2 (Critical - 1 hour)
2. **Week 1:** Plan A + G3-G10 + H1, H5-H8 (Quick wins + security hardening)
3. **Week 1-2:** Plans B, C + G4 (App.tsx + settings plumbing)
4. **Week 2:** Plan D + H2, H3 (Multi-interface + WebSocket/SNMP security)
5. **Week 3-4:** Plan E + H4 (Single source + proxy support)
6. **Ongoing:** Plan F (Code structure)

---

## Part 6: Agent Coordination Guide

This section enables multiple agents to work in parallel without conflicts.

### File Ownership Matrix

Each plan has exclusive ownership of specific files. Agents must not modify files outside their ownership without
coordination.

| Plan | Owned Files (Backend)                                                         | Owned Files (Frontend)                                    |
| ---- | ----------------------------------------------------------------------------- | --------------------------------------------------------- |
| A1   | -                                                                             | `SettingsDrawer.tsx` (icon only)                          |
| A2   | `cmd/seed/cmd_uninstall.go`                                                   | -                                                         |
| A3   | `internal/network/linkmon.go`                                                 | -                                                         |
| A4   | `internal/api/websocket.go` (auth section)                                    | -                                                         |
| A5   | -                                                                             | `settings.ts` (lengthUnit removal), Cable components      |
| B    | -                                                                             | `web/src/app/LoginForm.tsx` (new), `App.tsx` (extract)    |
| C    | -                                                                             | `web/src/hooks/useNetworkFetchers.ts` (new), `App.tsx`    |
| D    | `internal/config/profile_settings.go`                                         | `ProfileEditor.tsx`, `settings.ts` (profile types)        |
| E    | `internal/config/default_profile.json` (new), `internal/database/profiles.go` | `settings.ts` (remove DEFAULT\_\*), `SettingsContext.tsx` |
| F    | `internal/api/handlers_*.go` (splits)                                         | `components/settings/sections/*`, `components/cards/*`    |
| G1   | `internal/api/handlers_settings.go:255-257`                                   | -                                                         |
| G2   | `internal/api/handlers_auth.go:351`, `handlers_security.go:189`               | -                                                         |
| G3   | `internal/api/handlers_settings.go` (validation)                              | -                                                         |
| G4   | `internal/api/handlers_settings.go` (new endpoints)                           | `SettingsDrawer.tsx` (save handlers)                      |
| G5   | `internal/auth/csrf.go`                                                       | -                                                         |
| G6   | `internal/logging/redact.go`                                                  | -                                                         |
| G7   | `internal/auth/auth.go`                                                       | -                                                         |
| G8   | `internal/api/limits.go` (new), all handlers                                  | -                                                         |
| G9   | -                                                                             | Documentation/OpenAPI spec                                |
| G10  | `internal/database/database_test.go`                                          | -                                                         |
| H1   | `internal/auth/csrf.go`                                                       | -                                                         |
| H2   | `internal/api/websocket.go` (rate limiting)                                   | -                                                         |
| H3   | `internal/config/config.go`, `internal/snmp/snmp.go`                          | -                                                         |
| H4   | `internal/api/proxy.go` (new), `cmd/seed/cmd_serve.go`                        | -                                                         |
| H5   | -                                                                             | `web/src/utils/reportRenderer.ts`                         |
| H6   | `internal/logging/middleware.go`                                              | -                                                         |
| H7   | `internal/api/handlers_devices.go`, others                                    | -                                                         |
| H8   | `internal/auth/csrf.go`                                                       | -                                                         |
| I1   | `internal/discovery/arp.go`                                                   | -                                                         |
| I2   | -                                                                             | `NetworkDiscoveryCard.tsx`                                |
| I3   | `internal/api/handlers_discovery.go`                                          | `NetworkDiscoveryCard.tsx`                                |
| J1   | -                                                                             | `SystemHealthCard.tsx`                                    |
| J2   | `internal/system/system.go`                                                   | `SystemHealthCard.tsx`                                    |
| J3   | -                                                                             | `SystemHealthCard.tsx`                                    |

### Dependency Graph

```text
                    ┌─────────────────────────────────────────────┐
                    │           NO DEPENDENCIES                    │
                    │  (Can run in parallel immediately)           │
                    └─────────────────────────────────────────────┘
                                        │
        ┌───────────────────────────────┼───────────────────────────────┐
        ▼                               ▼                               ▼
   ┌─────────┐                    ┌─────────┐                    ┌─────────┐
   │ Plan A  │                    │ Plan B  │                    │ Plan G  │
   │ (Quick  │                    │(Login   │                    │(Critical│
   │  Wins)  │                    │ Form)   │                    │  Fixes) │
   └────┬────┘                    └────┬────┘                    └────┬────┘
        │                              │                              │
        │                              ▼                              │
        │                        ┌─────────┐                          │
        │                        │ Plan C  │◄─────────────────────────┘
        │                        │(Fetcher │
        │                        │ Hooks)  │
        │                        └────┬────┘
        │                              │
        ▼                              ▼
   ┌─────────┐                   ┌─────────┐
   │ Plan H  │                   │ Plan D  │
   │(Security│                   │(Multi-  │
   │Hardening│                   │Interface│
   └────┬────┘                   └────┬────┘
        │                              │
        │                              ▼
        │                        ┌─────────┐
        └───────────────────────►│ Plan E  │
                                 │(Single  │
                                 │ Source) │
                                 └────┬────┘
                                      │
                                      ▼
                                 ┌─────────┐
                                 │ Plan F  │
                                 │(Code    │
                                 │Structure│
                                 └─────────┘
```

### Parallel Execution Groups

**Group 1 - No Dependencies (Start Immediately):**

| Agent   | Plan       | Files                                                              |
| ------- | ---------- | ------------------------------------------------------------------ |
| Agent-1 | G1, G2     | `handlers_settings.go`, `handlers_auth.go`, `handlers_security.go` |
| Agent-2 | A1, A3     | `SettingsDrawer.tsx` (icon), `linkmon.go`                          |
| Agent-3 | A2         | `cmd_uninstall.go`                                                 |
| Agent-4 | H5, H6, H8 | `reportRenderer.ts`, `middleware.go`, `csrf.go`                    |
| Agent-5 | I1         | `arp.go` (fix ICMP IsLocal bug)                                    |
| Agent-6 | J1         | `SystemHealthCard.tsx` (fix 401 handling)                          |

**Group 2 - After Group 1:**

| Agent    | Plan    | Files                                               | Waits For |
| -------- | ------- | --------------------------------------------------- | --------- |
| Agent-7  | B       | `LoginForm.tsx`, `App.tsx`                          | None      |
| Agent-8  | G3, G10 | `handlers_settings.go`, `database_test.go`          | G1        |
| Agent-9  | A4, A5  | `websocket.go`, `settings.ts`                       | None      |
| Agent-10 | H1, H7  | `csrf.go`, `handlers_devices.go`                    | H8        |
| Agent-11 | I2, I3  | `NetworkDiscoveryCard.tsx`, `handlers_discovery.go` | I1        |
| Agent-12 | J2      | `system.go`, `SystemHealthCard.tsx`                 | J1        |

**Group 3 - After Group 2:**

| Agent    | Plan   | Files                                  | Waits For |
| -------- | ------ | -------------------------------------- | --------- |
| Agent-13 | C      | `useNetworkFetchers.ts`, `App.tsx`     | B         |
| Agent-14 | G4, G5 | `handlers_settings.go`, `csrf.go`      | G3        |
| Agent-15 | G6, G8 | `redact.go`, `limits.go`               | None      |
| Agent-16 | H2, H3 | `websocket.go`, `config.go`, `snmp.go` | A4        |
| Agent-17 | J3     | `SystemHealthCard.tsx` (remediation)   | J2        |

**Group 4 - After Group 3:**

| Agent    | Plan | Files                                                     | Waits For |
| -------- | ---- | --------------------------------------------------------- | --------- |
| Agent-18 | D    | `profile_settings.go`, `ProfileEditor.tsx`, `settings.ts` | C         |
| Agent-19 | H4   | `proxy.go`, `cmd_serve.go`                                | G6        |
| Agent-20 | G9   | OpenAPI documentation                                     | G3, G4    |

**Group 5 - After Group 4:**

| Agent    | Plan | Files                                                                       | Waits For |
| -------- | ---- | --------------------------------------------------------------------------- | --------- |
| Agent-21 | E    | `default_profile.json`, `profiles.go`, `settings.ts`, `SettingsContext.tsx` | D         |

**Group 6 - Ongoing (After E):**

| Agent     | Plan | Files                            | Waits For |
| --------- | ---- | -------------------------------- | --------- |
| Agent-22+ | F    | Handler splits, component splits | E         |

### Verification Checklist Per Plan

Each agent must verify these before marking complete:

#### Plan A (Quick Wins)

- [ ] A1: Network icon visible in settings drawer
- [ ] A2: Uninstall logs errors (check with `seed uninstall --dry-run`)
- [ ] A3: Panic recovery logs stack trace (unit test)
- [ ] A4: WebSocket rejects `Sec-WebSocket-Protocol` auth (integration test)
- [ ] A5: Only `unitSystem` exists, `lengthUnit` removed, Cable Test uses global setting

#### Plan B (LoginForm)

- [ ] `LoginForm.tsx` exists in `web/src/app/`
- [ ] Login works (manual test)
- [ ] SSO providers display correctly
- [ ] Error messages display correctly
- [ ] `App.tsx` reduced by ~315 lines

#### Plan C (Fetcher Hooks)

- [ ] `useNetworkFetchers.ts` exists with all 12 fetch functions
- [ ] `useCardState.ts` exists with card state management
- [ ] `useInterfaceState.ts` exists with interface switching
- [ ] All cards still load data correctly (integration test)
- [ ] `App.tsx` reduced by ~400 lines

#### Plan D (Multi-Interface)

- [ ] Backend accepts array of interfaces per type
- [ ] Existing single-interface profiles migrate correctly
- [ ] UI shows interface multi-select
- [ ] Per-interface thresholds save and load
- [ ] Profile export includes all interfaces

#### Plan E (Single Source of Truth)

- [ ] `default_profile.json` embedded in binary
- [ ] System creates default profile on first startup
- [ ] No `DEFAULT_*` constants in TypeScript
- [ ] Settings load from profile, not hardcoded values
- [ ] Existing user profiles migrated to full copies

#### Plan F (Code Structure)

- [ ] Target file under 400 lines
- [ ] All imports updated
- [ ] Tests pass
- [ ] No circular dependencies
- [ ] Index files export all components

#### Plan G (Critical Fixes)

- [ ] G1: Settings save returns 500 on DB error (unit test)
- [ ] G2: All `Lock()` have matching `defer Unlock()` (grep verify)
- [ ] G3: Invalid JSON returns 400 Bad Request (integration test)
- [ ] G4: Link/Cable settings persist after reload (manual test)
- [ ] G5: CSRF cleanup stops on shutdown (graceful shutdown test)
- [ ] G6: Logs indicate X-Forwarded-For is untrusted
- [ ] G7: Crypto failure logs critical, triggers restart policy
- [ ] G8: All handlers use `limits.go` constants (grep verify)
- [ ] G9: OpenAPI spec matches actual API
- [ ] G10: All test assertions use `require.NoError`

#### Plan H (Security Hardening)

- [ ] H1: Session ID only from JWT, not X-Username header
- [ ] H2: WebSocket disconnects on rate limit exceeded
- [ ] H3: MD5 logs deprecation warning
- [ ] H4: `--trusted-proxies` flag works
- [ ] H5: `escapeHtml` uses string replacement, not DOM
- [ ] H6: Crypto failure panics or logs critical
- [ ] H7: No `err.Error()` in client responses
- [ ] H8: `SetCSRFCookie` removed

#### Plan I (Discovery UI/UX)

- [ ] I1: ICMP-discovered devices on local subnet show in "Local Network" section
- [ ] I2: Discovery header shows only subnets and device count (no local IP, no interface)
- [ ] I3: Subnet list collapses when >5 subnets with expandable dropdown

#### Plan J (System Health)

- [ ] J1: SystemHealthCard triggers session-expired event on 401 instead of showing error
- [ ] J2: Top 3-5 processes shown when CPU/memory exceeds 75%
- [ ] J3: Remediation suggestions appear for high resource usage

### Communication Protocol

When agents need to coordinate:

1. **File Conflict**: If two plans need the same file, the earlier group completes first
2. **Type Changes**: Plan D changes profile types → Plan E must wait
3. **API Changes**: Plan G4 adds endpoints → Plan G9 documents them
4. **Test Failures**: Agent stops, reports failure, waits for dependency fix

### Build & Test Commands

After each plan, run:

```bash
# Backend
cd ~/developer/projects/seed
go build ./...
go test ./...

# Frontend
cd web
npm run build
npm run test
npm run lint
```

### Rollback Procedure

If a plan breaks the build:

1. `git stash` changes
2. Notify dependent agents to pause
3. Fix issue or revert
4. `git stash pop` and retry
5. Signal dependent agents to continue

### Agent Prompt Template

When spawning an agent for a specific plan:

```text
You are implementing Plan [X] from the Seed Issues document.

**Your Task:** [Description]

**Files You Own:** [List from File Ownership Matrix]

**DO NOT MODIFY:** Any files not in your ownership list

**Verification:** Complete ALL items in the verification checklist for Plan [X]

**Dependencies:** [List any plans that must complete first]

**When Done:**
1. Run build and tests
2. Report: "Plan [X] complete. Verification: [checklist status]"
```
