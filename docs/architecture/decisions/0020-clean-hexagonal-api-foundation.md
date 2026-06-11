# ADR-0020: Clean-hexagonal `internal/api` foundation — domain-meaningful use-case packages, adapters in the composition root

**Status:** Accepted — 2026-06-09

**Refines:** ADR-0016 (strangle `internal/api` into use-cases). ADR-0016 set the
thin-handler → use-case → encode direction; ADR-0020 locks the package layout,
naming, and adapter placement that the remaining strangle PRs follow. Where the
two differ, ADR-0020 wins.

## Context

ADR-0016 introduced per-domain use-case packages but left two things loose, and
the first implementations drifted as a result:

1. **Package naming.** ADR-0016 §Naming mandated the `<domain>app` suffix
   (`networkapp`, `wifiapp`, …) purely to dodge an import-alias clash with the
   `internal/app` composition root. The suffix names the *layer*, not what the
   package *provides* — the opposite of idiomatic Go (`net/http`, not
   `net/httplayer`). It was a workaround, not a convention worth keeping.

2. **Adapter placement.** ADR-0016 §Decision rule 4 says the composition root
   wires the use-cases. In practice the network/settings/profiles/alerts
   adapters were defined *inside* `internal/api` in `<domain>_usecases.go` files
   (`networkHardware`, `networkConfigStore`, `initNetworkUseCase`, …). That keeps
   concrete knowledge of `netif`, the config store, and the database in the
   transport layer — exactly the coupling the strangle is meant to remove. The
   Wi-Fi capture/visibility/reporting components, by contrast, are correctly
   built in `internal/app` and injected. The api-layer wiring was the outlier.

We are pre-alpha (no backwards-compatibility burden, see master CLAUDE.md), so we
fix the convention now rather than grandfather the drift, and retrofit every
already-migrated domain to the corrected shape before adding new slices.

## Decision

A migrated domain is laid out in three rings with a single composition seam.

### 1. Feature core — `internal/<domain>/...` (unchanged)

The persistence-free domain logic. No `net/http`, no SQL. Unchanged by the
strangle.

### 2. Application service + ports — `internal/<domain>/<capability>`

The use-case service and the **consumer-defined ports** it drives live in a
**domain-meaningful package named for the capability it provides, not the layer**:

- `internal/network/ipconfig` (package `ipconfig`) — IP-configuration + MTU.
- `internal/discovery/scanning` (package `scanning`) — device discovery.
- …never `internal/<domain>/usecase` or `internal/<domain>/app`.

The service type de-stutters against its package: `ipconfig.Service`,
`ipconfig.NewService` — not `ipconfig.IPService`. The ports (`Hardware`,
`ConfigStore`, …) are declared **here, at the consumer** (interface segregation,
ADR-0016 rule 3): the service names the small interface it needs; the adapter
satisfies it. No `net/http`, no SQL, no concrete infra types in this ring.

Because the package is named for its capability, the `<domain>app` alias hack is
gone: `internal/app` imports `ipconfig`, `scanning`, … with no alias collision.

### 3. Adapters — the composition root `internal/app` (package `app`)

Infra adapters that satisfy the consumer-defined ports live in the composition
root **by default**. `internal/app` is the one layer that legitimately knows
every concrete (`netif.Manager`, the config store, the database, the engines), so
it is where port-to-concrete glue belongs. The root exposes one constructor per
use-case that builds the adapters and assembles the service:

```go
// internal/app/network.go
func NewNetworkIP(mgr func() *netif.Manager, cfg *config.Config, path string) *ipconfig.Service {
    return ipconfig.NewService(networkHardware{mgr: mgr}, networkConfigStore{cfg: cfg, path: path})
}
```

Promote an adapter to its own `internal/<domain>/<infra>` package **only** when it
is substantial or reused (e.g. `internal/reporting/store`). There is no generic
`adapter` package and no per-domain adapter package for thin glue — that was
considered (Option 1) and rejected as over-packaging.

### 4. Transport — `internal/api` is pure transport

Handlers decode the request → call **one** use-case method → encode the result,
mapping domain sentinels to HTTP status. `internal/api` holds **no adapter
definitions and no `<domain>_usecases.go` files.** It imports `internal/app` and
invokes the root's constructors, passing its own lazy accessors, then stores the
returned service on `Server`:

```go
s.networkIP = app.NewNetworkIP(s.netManager, s.config, s.configPath)
```

`s.netManager` is a method value that reads `s.services.Network.Manager`, so a
manager set or replaced after construction (the test harness does this) is
honored — the lazy seam that previously justified building in-package is
preserved without keeping the adapters in `internal/api`. `internal/app` never
imports `internal/api`; the dependency edge is one-way (`api → app`), matching the
existing `cmd/seed → app` wiring.

### 5. Security policy stays at the registry edge (unchanged, load-bearing)

Authorization, CSRF, and rate-limiting stay where ADR-0002 puts them —
`methodGate` / `writeGated` / `requireRole` / `requireFeature` at route
registration. The strangle **must not** move policy into use-cases or weaken any
Wave-4/5 invariant. Each PR verifies the route-policy manifest is byte-identical
(`scripts/check-route-policy.sh`) and output escaping is intact
(`scripts/check-output-escaping.sh`).

### 6. Goldens are a reviewed change-gate, not a freeze

`TestGoldenHTTP*` stays byte-identical across a structural move. Because we carry
no backwards-compatibility burden pre-v1.0.0, a genuinely wrong status code or
response shape *is* fixed when found — but every golden diff is reviewed and
intended, never blindly re-baselined to lock in cruft.

## Consequences

- **Positive:** package names read as capabilities; the transport layer is free
  of infra knowledge; adapters live in the one place allowed to know concretes;
  the `<domain>app` alias hack disappears; the layout is uniform across domains.
- **Cost:** every already-migrated domain (network, alerts, settings, profiles,
  wifi) is retrofitted to the corrected shape — mechanical, golden-pinned PRs.
- **Edge:** `internal/api → internal/app` is a new, intentional import. It is
  acyclic (the root never imports transport) and is the minimal seam that
  preserves the lazy-manager resolution the test harness relies on.

## Implementation phasing

- **A1 (this ADR's PR) — exemplar: `network`.** Rename `internal/network/app`
  (`networkapp`) → `internal/network/ipconfig` (`ipconfig`), `IPService` →
  `Service`; move `networkHardware` + `networkConfigStore` into
  `internal/app/network.go`; delete `internal/api/network_usecases.go`; api wires
  via `app.NewNetworkIP`. This is the reference implementation and the
  directory/naming exemplar.
- **B1–B4 — retrofit** alerts, settings, profiles, wifi to the same shape.
- **C1–C4 — new slices** (discovery, health, update, identity) land directly in
  this shape; identity replaces raw `Database.DB` handler access with repository
  ports.
- **D1 — terminal:** when no handler reaches `ServiceContainer`, delete it and the
  god accessors; mark ADR-0016 Completed and this ADR remains Accepted.
