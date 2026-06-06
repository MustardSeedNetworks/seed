# ADR-0016: Strangle `internal/api` into per-domain use-case services

**Status:** Accepted — 2026-06-06

## Context

The independent architecture review (2026-06-05, findings #2 and #3, both
verified true) flagged `internal/api` as the last structural debt after the
Phase 0–7 re-architecture:

- **#2 — god layer.** `internal/api` is ~100 non-test files and ~30k LOC.
  Individual handlers run 600–1100 lines and interleave four concerns in one
  function: request decoding, authorization/validation, business logic, and
  response encoding. The capability registry (ADR-0002) already made *routing*
  policy declarative, but the *handlers* behind the routes are still fat.
- **#3 — bag-of-services DI.** `ServiceContainer` (`internal/api/services.go`)
  exposes db / discovery / diagnostics / auth / jobs / health / update / wifi to
  every handler at once. Handlers also reach into `Server.background` directly
  (e.g. `s.background.WiFiVisibility`). A handler can touch anything, so nothing
  is isolated and the blast radius of a change is the whole package.

The Phase 0–7 work established the target for the *domain* layer:
capability-first `internal/<feature>` packages that are persistence-free, with
ports at the I/O seam and a composition root in `internal/app`. `internal/api`
never got the same treatment.

This is explicitly **not a rewrite**. The package works, its security
invariants are CI-gated, and the golden HTTP harness pins its behavior. We
strangle it incrementally: each phase moves one cohesive slice behind the new
structure while the goldens prove behavior is unchanged.

## Decision

Adopt the **thin-handler → use-case → encode** shape, backed by per-domain
**application (use-case) packages** and **narrow, consumer-defined ports**.

### Target structure

```
internal/api/handlers_*.go      thin: decode request → call a use-case → encode response
        │  depends on (narrow interface, defined HERE at the consumer)
        ▼
internal/<domain>/app           use-case services: orchestrate the domain for one
   (e.g. internal/wifi/app)     request shape; no net/http, no encoding, testable
        │
        ▼
internal/<domain>/...           the persistence-free feature core (unchanged)
```

Rules:

1. **Handlers do I/O translation only** — decode the HTTP request, call exactly
   one use-case method, encode the result (and map domain errors to status
   codes). No business logic, no multi-service orchestration in the handler.
2. **Use-cases live in `internal/<domain>/app`** (package `<domain>app`, e.g.
   `wifiapp`, to avoid clashing with the `internal/app` composition root). A
   use-case takes the narrow dependencies it needs as constructor arguments.
3. **Ports are defined at the consumer** (interface-segregation): the use-case
   declares the small interface it needs; the composition root supplies the
   concrete implementation. Handlers depend on the use-case (or a small
   interface over it), **not** on `ServiceContainer` or `Server.background`.
4. **The composition root wires it** — `internal/app` / `cmd/seed` build the
   use-cases from the feature components and hand them to the API layer, the
   same way `BackgroundComponents` are built today.
5. **Behavior is pinned by the goldens** — every phase keeps
   `TestGoldenHTTP*` byte-identical (or updates them as a reviewed diff only
   when behavior is *intended* to change).

### Phasing

Each phase is one PR, one domain slice, goldens green:

- **Phase 1 (this ADR's PR) — exemplar: Wi-Fi visibility reads.** Introduce
  `internal/wifi/app` (`wifiapp`) with a `Queries` use-case over a narrow
  `VisibilitySource` port, and convert the `/wifi/airspace` + `/wifi/anomalies`
  handlers from reaching into `s.background.WiFiVisibility` (the #3 anti-pattern)
  to calling the injected use-case. Small, self-contained, recently authored —
  the lowest-risk place to establish the pattern and the `internal/wifi/app`
  ring the later phases follow.
- **Phase 2+ — fat handlers, by domain.** Migrate the heavier handlers
  (wifi settings/connect, discovery, diagnostics, security, reporting) one
  cohesive slice at a time into their `internal/<domain>/app` use-cases, pulling
  each off `ServiceContainer` as it goes. Order by blast radius (smallest
  first); each lands only when its goldens are green.
- **Terminal — retire the bag.** When no handler reaches `ServiceContainer`
  fields directly, replace it with the set of injected use-cases and delete the
  god accessor.

### Naming

`internal/<domain>/app` packages are named `<domain>app` (`wifiapp`,
`discoveryapp`, …) so the composition root (`internal/app`, package `app`) can
import them without an alias collision — the same disambiguation already used by
`wifianomaly` / `wificapture`.

## Consequences

- **Positive:** handlers become readable and unit-testable without HTTP;
  business logic moves to packages that can be tested directly; the blast radius
  of a change shrinks to one domain; `ServiceContainer` reach-through stops
  spreading; new endpoints get a clear home.
- **Cost:** one indirection layer per domain; a migration that spans many PRs.
  Mitigated by doing it incrementally behind the golden harness — never a big-bang.
- **Non-goal:** changing behavior, routes, or the capability registry. This is
  a structural move only; ADR-0002 (routing policy) and the security invariants
  are untouched.

## Coordination

Independent of the auth/oauth workstream and of ADR-0015 (credential DEK). The
strangle touches handler wiring, not the `Auth.JWTSecret` boundary.
