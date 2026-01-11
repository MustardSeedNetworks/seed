# Code Duplication Report

Generated: 2025-01-11

## Summary

This report identifies code duplication patterns across the codebase with recommendations for refactoring.

**Total Estimated Duplicate Lines:** 400+ lines

---

## 1. Frontend (TypeScript) Duplication

### 1.1 IP Validation Logic (High Priority)

**Location:** `ui/src/hooks/useVulnerabilities.ts:56-105`

**Issue:** Custom IP validation that duplicates backend logic.

```typescript
// Duplicated in frontend
function isValidIpv4(ip: string): boolean { /* 8 lines */ }
function isValidIpv6(ip: string): boolean { /* 35 lines */ }
function isValidIp(ip: string): boolean { /* 3 lines */ }
```

**Backend Alternative:** `internal/validation/validation.go:51-60` already has:
- `IsValidIP()`
- `IsValidIPv4()`

**Recommendation:**
1. Create shared `ui/src/lib/validation.ts` utility
2. Or validate on backend only, return 400 for invalid IPs

---

### 1.2 API Error Handling Pattern (High Priority)

**Locations:** Multiple hooks (85+ API calls)

**Pattern:**
```typescript
try {
  return await api.get<Type>(endpoint);
} catch (err) {
  const message = err instanceof Error ? err.message : "Failed to...";
  logger.error(LogComponents.XXX, "Error message", err, { endpoint });
  return null;
}
```

**Files Affected:**
- `useDevices.ts` (7 instances)
- `useVulnerabilities.ts` (5 instances)
- `useHealthChecks.ts` (3 instances)
- `useNetworkData.ts` (4 instances)

**Recommendation:** Use existing `useFetch.ts` hook instead of manual error handling.

---

### 1.3 Settings Fetch/Update Boilerplate (High Priority)

**Locations:**
- `useDevices.ts:261-289`
- `useVulnerabilities.ts:220-245`
- `useHealthChecks.ts:250-275`

**Repeated Pattern (~15-20 lines each):**
```typescript
const fetchSettings = useCallback(async () => {
  try {
    return await api.get<SettingsType>(endpoint);
  } catch (err) {
    logger.error(...);
    return null;
  }
}, []);

const updateSettings = useCallback(async (settings: Partial<SettingsType>) => {
  try {
    await api.put(endpoint, settings);
    return true;
  } catch (err) {
    logger.error(...);
    return false;
  }
}, []);
```

**Recommendation:** Create generic `useSettings<T>(endpoint)` hook.

---

### 1.4 Network Data Fetching (Medium Priority)

**Location:** `ui/src/hooks/useNetworkFetchers.ts:96-595`

**Issue:** 12 separate fetch functions with identical patterns (~40 lines each):
- `fetchLinkData`
- `fetchIpConfig`
- `fetchInterfaces`
- `fetchVersion`
- `fetchDiscoveryData`
- `fetchDnsData`
- `fetchVlanData`
- `fetchGatewayData`
- `fetchWifiData`
- `fetchCableData`
- `fetchPublicIp`
- `fetchNetworkDiscovery`

**Estimated Duplication:** ~480 lines that could be ~120 lines

**Recommendation:** Use `useFetch` hook with card-specific config objects.

---

### 1.5 Scan Trigger Pattern (Medium Priority)

**Locations:**
- `useDevices.ts:224-238`
- `useVulnerabilities.ts:131-159`

**Pattern:**
```typescript
const triggerScan = useCallback(async () => {
  setIsScanning(true);
  setScanError(null);
  try {
    const data = await api.post<ScanResponse>(endpoint);
    return data.status === "scan started" || data.status === "scan already in progress";
  } catch (error) {
    setScanError(message);
    return false;
  } finally {
    setIsScanning(false);
  }
}, []);
```

**Recommendation:** Create `useScan(endpoint)` hook.

---

## 2. Backend (Go) Duplication

### 2.1 Method Validation Boilerplate (Medium Priority)

**Locations:** 99 instances across handler files

**Pattern:**
```go
if r.Method != http.MethodGET {
  sendErrorResponseWithDetails(
    w, logger,
    http.StatusMethodNotAllowed,
    ErrCodeMethodNotAllowed,
    "Method not allowed",
    "",
  )
  return
}
```

**Files Affected:**
- `handlers_devices.go:32-42, 70-80`
- `handlers_vuln.go:36-46, 130-140`
- `handlers_profiles.go:67-100`
- Most other handlers

**Recommendation:** Use router-based method filtering or middleware:
```go
// Instead of checking in each handler
mux.HandleFunc("GET /api/v1/devices", s.handleGetDevices)
```

---

### 2.2 Service Availability Checks (Medium Priority)

**Locations:** 5+ instances per service type

**Pattern:**
```go
if s.deviceDiscovery() == nil {
  sendErrorResponseWithDetails(
    w, logger,
    http.StatusServiceUnavailable,
    ErrCodeServiceUnavail,
    "Device discovery not available",
    "",
  )
  return
}
```

**Files Affected:**
- `handlers_devices.go:44-54, 82-92`
- `handlers_vuln.go:48-58, 142-147`
- `handlers_profiles.go:72-76`

**Recommendation:** Create `requireService(w, r, service)` helper:
```go
func (s *Server) requireService(w http.ResponseWriter, r *http.Request, svc any, name string) bool {
  if svc == nil {
    sendServiceUnavailable(w, r, name)
    return false
  }
  return true
}
```

---

### 2.3 Background Scan Execution (Medium Priority)

**Locations:**
- `handlers_devices.go:104-135`
- `handlers_vuln.go:85-117`

**Pattern (~25 lines each):**
```go
go func(reqCtx context.Context) {
  bgLogger := logging.FromContext(reqCtx)
  ctx, cancel := context.WithTimeout(context.Background(), timeout)
  defer cancel()

  bgLogger.Info("Starting background scan")
  start := time.Now()
  defer func() {
    bgLogger.Info("Scan finished", "duration_ms", time.Since(start).Milliseconds())
  }()

  if err := scanner.Scan(ctx); err != nil {
    bgLogger.Error("Scan error", "error", err)
  }

  s.wsHub().Broadcast(Message{...})
}(r.Context())
```

**Recommendation:** Create `runBackgroundScan(ctx, scanner, messageType)` helper.

---

### 2.4 Logger + Localizer Initialization (Low Priority)

**Locations:** 20+ handlers

**Pattern:**
```go
logger := logging.FromContext(r.Context())
localizer := i18n.FromRequest(r)
```

**Recommendation:** Consider a handler wrapper that provides these:
```go
func (s *Server) withContext(fn func(w http.ResponseWriter, r *http.Request, logger *slog.Logger, loc *i18n.Localizer)) http.HandlerFunc
```

---

## 3. Cross-Layer Duplication

### 3.1 IP Validation (Frontend ↔ Backend)

**Frontend:** `useVulnerabilities.ts:56-105` (custom implementation)
**Backend:** `validation.go:51-60` (canonical implementation)

**Gap:** No shared validation means:
1. Duplicate logic maintenance
2. Potential behavior differences
3. Double validation (client and server)

**Recommendation:**
- Frontend should do basic format checking only
- Backend is source of truth for validation
- Remove frontend IPv6 validation complexity

---

## 4. Duplication Metrics

| Pattern | Locations | Est. Lines | Priority |
|---------|-----------|-----------|----------|
| API error handling | 7+ hooks | 85+ calls | HIGH |
| Settings fetch/update | 3+ hooks | 45+ | HIGH |
| IP validation | Frontend + Backend | 50+ | HIGH |
| Network data fetchers | 1 file | 360+ → 120 | MEDIUM |
| Method validation | 34+ handlers | 200+ | MEDIUM |
| Service availability | 5+ instances | 40+ | MEDIUM |
| Background scans | 2+ handlers | 50+ | MEDIUM |
| Logger/localizer init | 20+ handlers | 20+ | LOW |

---

## 5. Recommended Refactoring Order

### Phase 1: Frontend Hooks (Highest Impact)
1. Create `useSettings<T>(endpoint)` generic hook
2. Create `useScan(endpoint)` generic hook
3. Consolidate IP validation to `ui/src/lib/validation.ts`
4. Refactor `useNetworkFetchers.ts` to use config-based approach

### Phase 2: Backend Helpers (Medium Impact)
1. Create `requireService()` helper
2. Create `runBackgroundScan()` helper
3. Use Go 1.22+ router method filtering

### Phase 3: Cross-Cutting (Lower Priority)
1. Handler context wrapper
2. Remove frontend IPv6 validation complexity

---

## 6. Example Refactored Code

### Generic Settings Hook
```typescript
// ui/src/hooks/useSettings.ts
export function useSettings<T>(endpoint: string) {
  const [settings, setSettings] = useState<T | null>(null);
  const [error, setError] = useState<string | null>(null);

  const fetch = useCallback(async () => {
    try {
      const data = await api.get<T>(endpoint);
      setSettings(data);
      return data;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to fetch settings");
      return null;
    }
  }, [endpoint]);

  const update = useCallback(async (updates: Partial<T>) => {
    try {
      await api.put(endpoint, updates);
      await fetch(); // Refresh
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update settings");
      return false;
    }
  }, [endpoint, fetch]);

  return { settings, error, fetch, update };
}

// Usage:
const { settings, update } = useSettings<DeviceSettings>("/api/v1/shell/devices/settings");
```

### Backend Service Check Helper
```go
// internal/httpapi/handlers_helpers.go
func (s *Server) requireService(
  w http.ResponseWriter,
  r *http.Request,
  svc any,
  name string,
) bool {
  if svc == nil {
    logger := logging.FromContext(r.Context())
    sendErrorResponseWithDetails(
      w, logger,
      http.StatusServiceUnavailable,
      ErrCodeServiceUnavail,
      fmt.Sprintf("%s service not available", name),
      "",
    )
    return false
  }
  return true
}

// Usage:
if !s.requireService(w, r, s.deviceDiscovery(), "Device discovery") {
  return
}
```
