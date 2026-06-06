# ADR-0002: Capability registry for route policy

**Status:** Accepted — 2026-05-31

## Context

Auth and CSRF are already enforced globally (`server_lifecycle.go` middleware
chain). But **role-gating** (`writeGated`), **license-gating** (`requireFeature`),
and **rate-limiting** are applied per-route by manually wrapping each handler at
registration — in inconsistent nesting order (Roots does
`rateLimit→requireFeature`; Canopy AirMapper does `rateLimit→requireFeature→writeGated`;
Harvest does bare `requireFeature` with no rate-limit). The project's own
conventions call adding a mutating route without the wrapper "a regression."
Enforcement is by convention, not construction.

## Decision

Routes become declarative `Route` values (method, path, auth, min-role, feature,
rate-limit class, handler, module). A single `register()` composes middleware in
**one canonical order** (`rateLimit → requireFeature → requireRole → handler`) for
every route and appends to an audit manifest.

- `GET /__capabilities` (or a build-time JSON artifact) emits the manifest.
- CI gate `check-route-policy.sh` fails if any handler is registered outside
  `register()`, or if a `POST/PUT/DELETE/PATCH` route has neither a `MinRole` nor an
  explicit `Auth: Public`.

## Consequences

- "Forgot the wrapper" becomes structurally impossible (CI-enforced).
- The nesting-order divergences are normalized (greenfield → fixed, not preserved).
- One machine-readable policy manifest for fleet security audits.
- Auth + CSRF stay global and untouched; only the per-route trio moves into the registry.
- Pure routing refactor — no business logic changes; guarded by golden tests.

## Amendment (2026-06-05) — method + body-limit are authoritative in the registry

The initial implementation captured only role/feature/rate-limit in `route{}`;
HTTP **method** validation and **request-body-size** limits still lived in
handlers (each handler's own `switch r.Method`/`MaxBytesReader`) and in a
separate path-switching `bodyLimitMiddleware` — a second, drift-prone source of
truth (arch-review finding #6; it had already drifted: `/wifi/survey/floor/floorplan`
wanted 10 MB but the path-switch capped it at 256 KB first).

`route{}` now also carries `methods []string` and `maxBodyBytes int64`, both
enforced by `register()`:

- **Method**: `register()` installs a `methodGate` that returns `405` + an
  `Allow` header for any method outside the declared set (using the project's
  JSON error envelope), composed *before* feature/role checks. Routes that use a
  Go 1.22 method-prefixed path (`"GET /api/v1/..."`) are enforced natively by
  ServeMux and leave `methods` empty.
- **Body limit**: `register()` wraps every route in a `MaxBytesReader` at
  `maxBodyBytes` (default `MaxBodySizeJSON`; explicit for tighter auth/config
  limits and the 10 MB/50 MB upload routes). The global `bodyLimitMiddleware` is
  reduced to a backstop for **non-API** paths only — `/api/v1` body limits are
  the registry's, eliminating the duplication and its drift.

Canonical composition order is now
`rateLimit → methodGate → requireFeature → requireRole → bodyLimit → handler`.

Enforced by `TestEveryAPIRouteDeclaresMethodAndBodyLimit`, which fails if any
`/api/v1` route is registered without a declared method or body limit (the
"no route bypasses the policy" guard), and `TestMethodGateRejectsUndeclaredMethod`.
The `/__capabilities` manifest exposes `methods`/`maxBodyBytes` for fleet audit.
