# Phase 3 — Reconcile Proposal: right-sized modular monolith + descriptive names

**Status:** PROPOSED — 2026-06-01 — *awaiting owner decision, no code moves yet*
**Supersedes (if approved):** the "extract every feature into a Module" framing of
`PHASE3_EXTRACTION_PLAN.md` (the harvest pilot under it is retained — see §6).

---

## 1. Executive summary

The Phase 3 blueprint assumed a hexagon where **thin HTTP handlers call module
domain cores** (`internal/{harvest,canopy,…}.Module`). The codebase does **not**
work that way. Investigation (2026-06-01) found:

- **No HTTP handler calls any module-facade accessor.** Not harvest, canopy,
  roots, sap, or shell. Verified by grep (`m.<Module>.<Accessor>()` → 0 hits).
- The **real** wiring is fat handlers + the api's own `internal/api/services.go`
  groupings (`SapServices`, `CanopyServices`, …) built **directly** from the
  underlying packages — and these are a **separate, duplicate** construction from
  the modules. Example: `services.Sap.DNS = dns.NewTester(...)` (served over HTTP)
  vs `sap.Module.dns` = a different `DNSService` instance (started in the
  background, read by nobody).
- So there are **two composition roots** that don't agree; the request path uses
  one and the module facades float beside it, partly dead, partly running
  background work on disconnected instances.

**This is a "two sources of truth" anti-pattern, not a naming nit** — though the
names make it worse (§3).

**Recommendation:** collapse to **one** composition root, delete the redundant
module facades, keep feature-grouped packages, model a **component only where
there is real owned state/lifecycle**, put **ports at the genuine I/O seams**,
and give everything **descriptive names**. This is a modular monolith sized to
the tool — neither "a Module for everything" (over-engineering) nor "no
structure" (under-engineering).

---

## 2. Audit — facade-by-facade

| Area (botanical) | Facade | `Start()` does | HTTP api consumes the facade? | Real impl the api actually uses | Verdict |
|---|---|---|---|---|---|
| **roots** | `internal/modules/roots` | (was no-op) | No | `handleTraceroute`/`handlePath` → `discovery` directly | **Dead — DELETED (#1439)** |
| **sap** | `internal/services` (`Module`) | starts link/gateway/telemetry **monitors on its own instances** | No | `SapServices`: `dns.Tester`, `dhcp.Monitor`, `gateway.Tester`, `vlan.Manager`, `speedtest`, `iperf`, `cable`, `publicip` — all `New`'d directly in `server.go` | **Redundant parallel build**; background monitors disconnected from served data (possible wasted goroutines) |
| **shell** | `internal/services/shell` | no-op TODO | No | security/discovery handlers directly | **Dead facade** (like roots) |
| **canopy** | `internal/canopy` | `wifi.Init()` on its own instance | No | `CanopyServices`: `wifi.NewManager`, `wifi.NewScanner`, `survey.NewManager` — `New`'d directly | **~Dead facade**; possible duplicate wifi init |
| **harvest** | `internal/modules/harvest` | `templates.Load()` + **scheduler.Start()** (real background) | No (HTTP `/harvest/export` builds its own `ExportData`) | scheduler→generator→`ReportRepo` (the pilot's ports) run in background; HTTP bypasses | **Real component (scheduler path)** but disconnected from HTTP |

**Pattern:** every facade is bypassed by the request path. `sap`/`harvest` do real
background work but on instances the api doesn't serve from; `shell`/`roots`/
`canopy` do little or nothing. The api's `services.go` groupings are the de-facto
real "modules."

---

## 3. Naming — botanical names are a concrete code problem

Two objective issues (independent of taste):

1. **Non-descriptive:** `internal/canopy` requires memorizing canopy=Wi-Fi,
   sap=telemetry, harvest=reporting, roots=path-analysis, shell=security. A
   permanent translation tax on every reader.
2. **Collide with technical terms:** `shell` = command shell (and a *second*
   `internal/services/shell` exists), `sap` = SAP ERP, `roots` = fs/math roots,
   `harvest` = data scraping. The names mislead, not just under-inform.

**Proposed code names (by function):**

| Botanical | → Code | Concern |
|---|---|---|
| roots | `pathanalysis` (or `netpath`) | traceroute / topology / IP enrichment |
| canopy | `wifi` | Wi-Fi visibility & troubleshooting |
| shell | `security` (or `posture`) | security posture & vulnerability |
| sap | `telemetry` | live link/gateway/dns/dhcp telemetry & monitoring |
| harvest | `reporting` | reports & export |

The api's own groupings already mix descriptive (`Auth`/`Network`/`Discovery`/
`Database`/`Health`/`Probe`/`RealTime`) with botanical (`Sap`/`Canopy`/`Roots`) —
so the rename lands there too.

**Marketing names** are a *separate, deliberate* decision (they're baked into
`LICENSE_STRATEGY.md`, the CLAUDE.md tables, brand, UI colors). Engineering
recommendation: **decouple** — code is descriptive regardless; keep the botanical
theme as decoration (product name "The Seed", per-area colors) if brand wants it,
but for a network-engineer buyer, descriptive feature labels (*Wi-Fi · Path
Analysis · Security · Telemetry · Reports*) read better than a metaphor. **Owner's
call** — flagged, not assumed.

---

## 4. Target architecture — modular monolith, sized right

1. **Feature-grouped packages** (`wifi`, `survey`, `dns`, `dhcp`, `gateway`,
   `discovery`, `database`, the report generator, …) — **keep**. Good structure.
2. **One composition root.** `internal/app` (created in the harvest pilot) becomes
   *the* place that constructs and lifecycle-manages everything. The api's
   `services.go` groupings are folded into / fed by it. No second root.
3. **A component (with `Start/Stop`) only where there is owned state/lifecycle** —
   the telemetry monitors, the report scheduler, the discovery engine, polling,
   alerts. Started from the one root, **on the same instances the api serves**.
   Stateless request/response logic (traceroute, export-snapshot) stays as plain
   handler-called functions — no facade.
4. **Ports at the genuine I/O seams** — DB → repo interfaces, `discovery` → a
   tracer/path interface — applied to the **live** code paths for testability and
   CGO isolation (the `discovery`/CGO seam coordinated with Phase 6).
5. **Delete the redundant module facades.**

```
                internal/app  (THE composition root: build + Start/Stop)
                      │
        ┌─────────────┼───────────────────────────┐
   api handlers   long-lived components        adapters (ports impl)
   (thin, call    (telemetry monitors,         store: *Repo over sqlite
    functions/    report scheduler,            net:   tracer/capture over discovery
    components)   discovery engine)            ── consumed by handlers/components ──
        └─────────────┴───────────────┬───────────┘
                          feature packages (wifi, dns, dhcp, gateway, survey, …)
```

---

## 5. Migration plan (phased, each PR green + golden-gated)

- **R1 — delete provably-dead facades.** `shell` (no-op) and `canopy` (if the
  `wifi.Init` lifecycle has another owner) — same surgical removal as roots
  (#1439). Drop the stale `RootsServices`.
- **R2 — reconcile sap (the big one).** Make the background monitors and the HTTP
  api use the **same** instances: fold `sap.Module`'s real lifecycle (link/
  gateway/telemetry monitors) into the one root started on the `SapServices`
  instances; delete the duplicate `internal/services.Module` wrapper layer. This
  removes the double-construction and any duplicate goroutines.
- **R3 — harvest.** Keep the ports/adapters/`app` wiring (good, on a live path).
  Decide: wire `/harvest/export` to the generator (close the HTTP gap) **or**
  keep it scheduler-only and document that export is a separate snapshot concern.
  Collapse the facade into the root.
- **R4 — descriptive rename** (mechanical, cross-cutting: package dirs, `Color()`,
  UI theme keys, CLAUDE.md tables, docs). Done **with** the restructure, not as a
  second pass.
- **R5 — rewrite the blueprint** to this model; take the **marketing-name**
  decision explicitly (owner), with locked-doc impact listed.

Order rationale: cold/dead first (R1), then the highest-duplication live area
(R2), then harvest (R3), rename once the shape is final (R4). Avoid the HOT zones
(polling/alerts/discovery internals/auth) — wrap, don't modify.

---

## 6. What we keep from work already done

- **harvest ports + `internal/adapters/store` ring** — good practice, sits on the
  live scheduler path; the `*Repo` pattern is the template for R2/R3 DB seams.
- **`internal/app` composition root** — becomes *the* root in this model.
- **depguard discipline, CI Go-cache fix (#1433), golden HTTP harness, the
  capability registry** — all still load-bearing.
- The roots relocate+cleanup+delete (#1435/#1438/#1439) — the autopsy that
  surfaced the dual-composition problem.

---

## 7. Decision needed

1. **Approve the Reconcile direction?** (collapse to one root, delete facades,
   descriptive code names, ports at real seams.)
2. **Marketing names:** keep botanical as decoration, or go descriptive
   customer-facing too? (separate brand call)

No code moves until (1) is approved.
