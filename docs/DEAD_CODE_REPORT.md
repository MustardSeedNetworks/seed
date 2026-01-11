# Dead/Deprecated Code Report

Generated: 2025-01-11

## Summary

This report identifies dead code, deprecated patterns, and cleanup opportunities in The Seed codebase.

---

## 1. Deprecated Frontend Code

### `ui/src/types/settings.ts` - Deprecated Constants

Multiple settings constants are marked as deprecated (lines 493-738):

```typescript
// DEPRECATED: These constants are now fallbacks only.
/** @deprecated Use useDefaults() hook instead - backend is single source of truth */
```

**Affected:**
- `DEFAULT_DNS_SETTINGS` (line 501)
- `DEFAULT_GATEWAY_SETTINGS` (line 519)
- `DEFAULT_DHCP_SETTINGS` (line 525)
- `DEFAULT_CABLE_SETTINGS` (line 541)
- `DEFAULT_SPEEDTEST_SETTINGS` (line 554)
- `DEFAULT_LINK_SETTINGS` (line 599)
- `DEFAULT_IP_SETTINGS` (line 652)
- `DEFAULT_IPERF_SETTINGS` (line 697)
- `DEFAULT_INTERFACE_SELECTOR_SETTINGS` (line 703)
- `DEFAULT_FAB_SETTINGS` (line 738)

**Action Required:** Remove these constants once all code uses `useDefaults()` hook.

---

### `ui/src/hooks/useSurvey.ts` - Legacy Fields (lines 234, 239)

```typescript
floorPlan?: FloorPlan; // Legacy: floor plan (deprecated, use floors)
samples?: SamplePoint[]; // Legacy: samples (deprecated, use floors)
```

**Action Required:** Remove after confirming no surveys use legacy format.

---

## 2. WebSocket vs SSE

The architecture now uses **Server-Sent Events (SSE)** for real-time updates. WebSocket support is deprecated but retained for backward compatibility.

### Backend Files
| File | Status | Notes |
|------|--------|-------|
| `internal/httpapi/handlers_websocket.go` | **Deprecated** | 39KB, fully functional but prefer SSE |
| `internal/httpapi/handlers_websocket_test.go` | Deprecated | Tests for deprecated WebSocket |
| `internal/httpapi/handlers_websocket_origin_test.go` | Deprecated | Origin validation tests |
| `internal/httpapi/handlers_sse.go` | **Active** | 11KB, primary real-time mechanism |

### Frontend Files
| File | Status | Notes |
|------|--------|-------|
| `ui/src/hooks/useSse.ts` | **Active** | Primary real-time hook |
| `ui/src/app.test.tsx` | **Needs Update** | Still mocks WebSocket, should mock SSE |

### E2E Tests
| File | Status | Notes |
|------|--------|-------|
| `ui/e2e/websocket.spec.ts` | **Outdated** | Tests WebSocket connectivity |
| `ui/e2e/websocket-realtime.spec.ts` | **Outdated** | Tests WebSocket real-time updates |

**Action Required:**
1. Update `app.test.tsx` to mock SSE instead of WebSocket
2. Update or rename E2E tests to test SSE
3. Consider removing WebSocket handlers if no clients use them

---

## 3. Fix #669 Comment Artifacts

Multiple files contain comments about "Fix #669" (removed `getAuthHeaders`):

```
ui/src/app.tsx:63
ui/src/components/survey/air-mapper-import.tsx:26
ui/src/components/cards/health-check-card.tsx:33
ui/src/components/survey/survey-view.tsx:37
ui/src/components/survey/report-dialog.tsx:30
ui/src/components/cards/system-health-card.tsx:38
ui/src/components/settings/settings-drawer.tsx:32
ui/src/components/settings/sections/config-backups-section.tsx:27
```

**Action Required:** These comments can be removed as they document a completed migration.

---

## 4. Legacy State in Settings Drawer

`ui/src/components/settings/settings-drawer.tsx:382`:
```typescript
// Legacy state (keep for IP settings which still needs manual apply)
```

**Action Required:** Investigate if IP settings can use auto-save like other settings.

---

## 5. Unused Data Table Feature

`ui/src/components/ui/data-table.tsx:113`:
```typescript
// Note: _expandedContent is available for future row expansion feature (prefixed to suppress unused warning)
```

**Action Required:** Either implement row expansion or remove the placeholder.

---

## 6. Recommendations

### High Priority
1. **Update tests** - `app.test.tsx` and WebSocket E2E tests need updating for SSE
2. **Remove Fix #669 comments** - Clean up completed migration artifacts

### Medium Priority
3. **Deprecate WebSocket handlers** - Add deprecation notices to Go WebSocket code
4. **Remove legacy survey fields** - After data migration verification

### Low Priority
5. **Clean up settings defaults** - Once `useDefaults()` is fully adopted
6. **Implement or remove data-table expansion** - Decide on feature

---

## 7. Files Safe to Remove (After Verification)

These files could potentially be removed if WebSocket is fully deprecated:

| File | Size | Reason |
|------|------|--------|
| `internal/httpapi/handlers_websocket.go` | 39KB | Deprecated in favor of SSE |
| `internal/httpapi/handlers_websocket_test.go` | 1.4KB | Tests deprecated code |
| `internal/httpapi/handlers_websocket_origin_test.go` | 13.8KB | Tests deprecated code |
| `ui/e2e/websocket.spec.ts` | 3.7KB | Tests deprecated feature |
| `ui/e2e/websocket-realtime.spec.ts` | 30.7KB | Tests deprecated feature |

**Total potential removal:** ~88KB

**Note:** Before removing, verify:
1. No external clients depend on WebSocket endpoint
2. SSE provides all necessary real-time features
3. Update ARCHITECTURE.md to remove WebSocket references
