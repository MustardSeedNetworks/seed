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
