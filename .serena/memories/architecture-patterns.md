# Architecture Patterns - The Seed

## Core Subsystems

| Subsystem | Path | Purpose |
|-----------|------|---------|
| discovery | `internal/discovery/` | Network device discovery |
| httpapi | `internal/httpapi/` | REST API handlers |
| database | `internal/database/` | SQLite storage |

## Frontend Patterns

### Authentication
- Cookie-based auth with httpOnly cookies
- CSRF protection via `X-CSRF-Token` header
- Use `useAuth()` hook for auth state
- API calls use `credentials: 'include'`

### Real-time Updates
- **Use SSE** (`useSse` hook) - NOT WebSocket
- SSE endpoint: `/api/events`
- Automatic browser reconnection

### State Management
- Zustand stores in `ui/src/stores/`
- Profile store: `profile-store.ts`
- Use `useDefaults()` hook for settings defaults (backend is source of truth)

### i18n
- Use `useTranslation('namespace')` 
- Namespaces: common, errors, settings, etc.
- Locale files in `internal/i18n/locales/`

### Theme
- Import tokens from `ui/src/styles/theme.ts`
- Use `cn()` for className merging

## API Conventions

- Base path: `/api/` (some legacy) and `/api/v1/`
- Auth: `/api/auth/login`, `/api/auth/logout`, `/api/auth/csrf`
- Status: `/api/status`
