# Harmonization Notes - Stem ↔ Seed

## Shared Patterns (Keep in Sync)

### Authentication (`useAuth` hook)
Both projects should have:
- `isAuthenticated`, `connected` state
- `login()`, `logout()`, `expireSession()` methods
- `clearError()` for clearing login errors
- `pollingIntervalRef` / `statsIntervalRef` for cleanup
- Cookie-based auth with CSRF protection

### i18n Structure
- Shared locale structure: `internal/i18n/locales/{en,es}/`
- Namespaces: common, errors, settings, setup, recovery, modules, params
- Type definitions in `ui/src/i18n/types.ts`

### Theme Tokens
- Import from `ui/src/styles/theme.ts`
- Use `cn()` for className composition
- Tokens: `layout`, `spacing`, `icon`, `alert`, `input`, `radius`, `button`

### Real-time Communication
- Both use SSE (Server-Sent Events)
- WebSocket is deprecated/removed

## Key Differences

| Aspect | Stem | Seed |
|--------|------|------|
| Purpose | Network testing | Network discovery |
| Modules | Benchmark, ServiceTest, etc. | Discovery engine |
| Dataplane | C/DPDK | Pure Go |

## Last Harmonized
- SetupWizard: ✅ Both have i18n + theme tokens
- RecoveryForm: ✅ Both have i18n + theme tokens
- useAuth: ✅ Both have expireSession, clearError, connected
