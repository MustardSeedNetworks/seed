# ADR-0018: Discovery pipeline stage split — capabilities as staged ports

**Status:** Accepted — 2026-06-07 · Phase 6 (the discovery split)

## Context

`internal/discovery` is the largest package in the codebase (~20.8K lines of
non-test code across ~54 files). It is a flat package: the orchestrator (`Engine`),
the device store (`DeviceRegistry`), the event bus, and every capability — ARP,
ICMP, NDP, mDNS, NetBIOS, WoL, Bluetooth, LLDP/CDP/EDP, OUI, port scan, TCP probe,
SNMP, profiler, fingerprint, traceroute, vulnerability/CVE/KEV, problem detection
— all sit side by side with no enforced internal direction. Any file can call any
other; the only seams are four ports (`capture.Opener`, `CVEProvider`,
`DBDeviceWriter`, `ProblemListener`) and the registry/event bus.

The blueprint (§16) calls for splitting this monolith into discrete pipeline
**stages**: **enumerate → resolve → fingerprint → vuln**. That mirrors what the
`Engine` already does at runtime. `Engine.Scan` (`engine.go`) runs a fixed
five-phase sequence in `runScanPhases`:

1. **Discovery** (`runDiscoveryPhase`) — wired/Wi-Fi/Bluetooth collectors enumerate
   hosts and links, writing devices to the registry. → **enumerate**
2. **Correlation** (`correlateDevices`) — currently a no-op; the registry already
   de-dupes by MAC on `AddOrUpdate`. → folds into enumerate
3. **Name resolution** (`ResolveNetBIOSNames` / `ResolveMDNSNames`) — attach
   hostnames. → **resolve**
4. **Enrichment** (`runEnrichmentPhase`) — SNMP collect, port scan, profile. →
   **fingerprint**
5. **Assessment** (`runAssessmentPhase`) — CVE/KEV vulnerability scan. → **vuln**

So the stage boundaries are not speculative; they are the existing phase boundaries,
just not expressed as package or interface boundaries.

### Why now — ADR-0007's deferral is unblocked

ADR-0007 deferred the discovery **orchestrator convergence** (Engine vs Pipeline)
to Phase 7, gated on two facts that are **no longer true**:

- *"the shipping UX rides Pipeline, the preferred Engine has no users."* The
  `/api/v1/security/pipeline/*` endpoints have since been **retired**; only
  `/api/v1/discovery/engine/*` remains, and **Engine is the canonical
  orchestrator**. The Pipeline→Engine fold already happened.
- *"discovery is a hot shared workstream needing coordination."* The owner now
  holds **sole ownership** of discovery; there is no parallel session to coordinate
  with.

This ADR is about a **different, complementary axis** than ADR-0007: not *which
orchestrator wins* (settled — Engine), but *how the capability code is packaged*
(by stage, behind ports). With both of ADR-0007's gates gone, the Phase-6 stage
split proceeds.

## Decision

Split the capability code into four **stage subpackages** behind **stage ports**,
with the `Engine` orchestrating the ports and `depguard` enforcing a one-way
dependency direction. Strangle one stage at a time, leaf-first, behind the golden
HTTP harness — **byte-identical wire output at every step**.

### Target structure

```
internal/discovery/
  model/        — leaf: DiscoveredDevice + all inter-stage data types, enums, ports
  enumerate/    — ARP/ICMP/NDP/mDNS-listen/NetBIOS/WoL/Bluetooth/LLDP/CDP/EDP/Wi-Fi/L2/manager/devices
  resolve/      — OUI + active DNS/NetBIOS/mDNS name resolution
  fingerprint/  — SNMP collect, port scan, TCP probe, profiler, fingerprint, traceroute
  vuln/         — vulnerability scan, CVE (NVD/local), KEV, problem detection
  (root)        — Engine, DeviceRegistry, EventBus, metrics, retry: the orchestrator
                  + the FACADE that re-exports the external surface (see below)
```

**Dependency direction (depguard-enforced):**
`root (orchestrator) → {enumerate, resolve, fingerprint, vuln} → model`.
A stage may import `model`; it may **not** import the root or a sibling stage. The
root imports the stages. `model` imports nothing from `discovery`.

### Stage ports

Each stage exposes a single small interface the Engine drives; the registry remains
the single source of truth, written between stages by the orchestrator:

```go
// in model (consumed by the Engine; implemented by each stage package)
type Enumerator interface {
    Enumerate(ctx context.Context, opts EnumerateOptions) ([]*DiscoveredDevice, error)
}
type Resolver interface {
    Resolve(ctx context.Context, devices []*DiscoveredDevice) error // attaches names
}
type Fingerprinter interface {
    Fingerprint(ctx context.Context, devices []*DiscoveredDevice, opts FingerprintOptions) error
}
type Assessor interface {
    Assess(ctx context.Context, devices []*DiscoveredDevice) error // attaches vulnerabilities
}
```

The interfaces stay ≤ a few methods (interfacebloat-safe). Stage *constructors*
take the existing capability dependencies (e.g. `Assessor` takes a `CVEProvider`).

### The facade — preserve the external surface

`internal/api` references ~50 `discovery.*` symbols (`DiscoveredDevice`, `Engine`,
`NewVulnerabilityScanner`, `PortScanIntensity`, `ScanOptions`, …). To keep the
strangle **golden-safe and API-churn-free**, the root `discovery` package stays a
**facade**: it re-exports moved types via aliases (`type DiscoveredDevice =
model.DiscoveredDevice`) and moved constructors via thin wrappers
(`func NewVulnerabilityScanner(...) = vuln.NewScanner(...)`). Aliases preserve the
reflected type name, so the code-first schema (`cmd/seed-schema`) and the
`/api/v1/discovery/engine/*` response shapes are unchanged. Callers in
`internal/api` are not touched in the stage PRs; a later optional cleanup can point
them at the subpackages directly once the dust settles.

### Strangle order (leaf-first, lowest risk first)

1. **Foundation — `model`.** Move `DiscoveredDevice`, the protocol-info structs,
   enums, the inter-stage DTOs (`DeviceProfile`, SNMP data, `DeviceVulnerabilities`,
   `WiFiPresence`, `BluetoothPresence`, `ScanOptions`/`ScanResult`/`EngineStats`),
   and the stage port interfaces into `internal/discovery/model`. Add aliases in the
   root. Golden + schema unchanged. *(No depguard rule yet — nothing to enforce.)*
2. **vuln.** Move `vulnerability.go`, `cve_*.go`, `problem*.go` → `discovery/vuln`
   (it already has the `CVEProvider` port; pure leaf). Engine calls the `Assessor`
   port. depguard: `vuln → model` only.
3. **fingerprint.** Move SNMP/portscan/tcpprobe/profiler/fingerprint/traceroute →
   `discovery/fingerprint`. Engine calls `Fingerprinter`. depguard added.
4. **resolve.** Move OUI + active name resolution → `discovery/resolve`. Engine
   calls `Resolver`. depguard added.
5. **enumerate.** Move the host/link discovery capabilities (the largest set,
   including the platform-specific `*_linux/_windows/_darwin.go` and the
   `capture.Opener` consumers) → `discovery/enumerate`. Engine calls `Enumerator`.
   depguard added.
6. **Direction lock + cleanup.** Final depguard rule set, remove now-dead facade
   indirection where safe, document in §16.

### Invariants held at every step

- **Golden HTTP harness byte-identical** (`TestGoldenHTTP*`); regenerate with
  `UPDATE_GOLDEN` only when a change is *intended* and reviewed.
- **Schema drift gate green** — regenerate `docs/schemas/api` + TS types if a moved
  type's reflection changes; aliases are chosen specifically to avoid that.
- **`make test-fast` (CGO=0) locally**; `-race` + real capture run in CI.
- **depguard rule lands with the stage it governs** (RED-proven, as in Phase 3).
- **Registry stays the SSoT write seam; the event bus stays the read seam.** Stages
  do not call each other; the orchestrator threads devices between them.
- **No behavior change, no endpoint change** — this is packaging, not a redesign.
  Any intended behavior change is called out explicitly in its PR.

## Consequences

- The hottest package in the tree gains an enforced internal architecture: four
  stages with a one-way dependency graph, each testable in isolation behind a port.
- The strangle is ~6 focused PRs, each green and golden-stable; risk is bounded per
  PR and the facade keeps `internal/api` untouched.
- Short-term cost: the root package carries facade aliases/wrappers during and after
  the migration. That indirection is the price of a churn-free, golden-safe move; a
  later cleanup may retire it.
- Complements ADR-0007 (orchestrator convergence, settled on Engine) and ADR-0008
  (pure-data discovery types in the schema): the `model` package is the natural home
  for those pure-data types.
- Risk: a moved capability that secretly depended on a sibling becomes a compile
  error when the depguard rule lands — which is exactly the coupling this surfaces
  and fixes, not a regression.

## Implementation note (2026-06-07): ports-first, root-as-kernel

During implementation a lower-risk refinement of the foundation step emerged. The
hub type `DiscoveredDevice` aggregates a field of every stage's result type
(`*DeviceProfile`, `*SNMPFullData`, `*DeviceVulnerabilities`, the protocol-info
structs). Moving those result types into a `model` leaf is therefore impossible
without either moving the whole cluster or creating an import cycle. So instead of
a `model` leaf + facade aliases, the **root `discovery` package is the kernel**:
it keeps `DiscoveredDevice` + all result types + the stage **port interfaces**;
each stage *subpackage* will import the kernel for those types and implement its
port; the kernel does **not** import the stages (the orchestrator holds the ports
as interfaces, and the composition root injects the concrete stages). `depguard`
enforces `stages → kernel`, never the reverse, and no sibling-to-sibling imports.
This keeps shared types put (zero `internal/api` churn, golden/schema unchanged)
and avoids the large, risky type move the `model`-leaf plan implied.

The first increment ("define stage seams as ports") was therefore done
**in-package**: the four ports (`Enumerator`, `Resolver`, `Enricher`, `Assessor`
— `Enricher` rather than `Fingerprinter`, which is an existing capability type)
and their stage structs live in `stages.go`, and `Engine.runScanPhases` now
orchestrates the ports. Per-stage subpackage relocation + depguard follow.

## Implementation status

- [x] 0. stage ports + stage types in-package; Engine orchestrates ports
      (`stages.go`); behaviour-preserving, golden byte-identical
- [x] 1. `vuln` stage → `discovery/vuln` subpackage + depguard. `VulnerabilityScanner`
      + CVE providers (NVD/local/KEV) + the assess `Stage` moved; result types
      (`Vulnerability`/`DeviceVulnerabilities`) stay in the kernel (DiscoveredDevice
      fields) and are aliased in the stage. Engine holds the injected `Assessor`
      port; `internal/api` wires `vuln.NewStage(scanner, engine.Registry(),
      engine.EventBus())`. depguard rule `discovery-stage-direction` bans the kernel
      from importing the stage. Golden byte-identical, schema unchanged.
- [x] 2. `fingerprint` stage → `discovery/fingerprint` + depguard. **DONE — scoped to
      the port-scan leaf cluster (PortScanner + TCPProber), NOT the whole enrich cluster.**

      **Why the original "atomic ~9-file move" plan was wrong (discovered during
      implementation, 2026-06-08):** the plan rested on *"`DeviceProfiler` embeds
      `*SNMPCollector` + `*PortScanner`, so the three enrich components are mutually
      coupled."* The first half is false in the code: `DeviceProfiler` embeds only
      `*SNMPCollector` and does its own `net.Dial` port scanning — it holds **no**
      `PortScanner`. So the enrich code is two *independent* clusters:
      (A) `SNMPCollector` ← `DeviceProfiler`, and (B) `TCPProber` ← `PortScanner`.

      **Why cluster A (profiler/SNMP) STAYS kernel-resident, by design:** the kernel
      orchestrator `discovery.Service` co-owns the `DeviceProfiler` lifecycle — it
      constructs it and drives ~11 methods (`Start/Stop/QueueProfile/GetProfile/
      IsProfiling/GetSNMPData/GetResolvedNames/GetProfilingStatus/Clear{Profiles,
      SNMPData,ResolvedNames}`), and the Engine adds `ScanConfigSnapshot/UpdateScanConfig`.
      `Service` cannot move to a stage subpackage (it reaches into `DeviceDiscovery`'s
      unexported `protoManager`). Moving `DeviceProfiler` out while `Service` stays
      would force a ~13-method union "ProfilerPort" that trips `interfacebloat` and is a
      1:1 mirror of the struct — a header-interface anti-pattern that hides nothing and
      adds only indirection. The profiler is genuinely orchestrator-coupled, not a leaf;
      forcing it behind a port fights that reality. It stays in the kernel until/unless
      a deliberate redesign gives `Service` a narrow profiler contract.

      **What moved (cluster B — a clean leaf with a narrow seam):** `portscan.go`,
      `tcpprobe.go`, `tcpprobe_windows.go` (+ their tests) → `internal/discovery/fingerprint`.
      The kernel gains one narrow port `PortScannerPort{QuickScan}` (stages.go); the
      `enrichStage`/`Engine` `portScanner` field + `SetPortScanner` take that interface;
      the composition root injects the concrete `fingerprint.PortScanner`.
      `Engine.GetCapabilities` keeps its nil-check on the interface field → identical wire.
      **Result types STAY in the kernel (`portscan_types.go`), aliased in fingerprint:**
      `PortState` (+ `PortOpen/PortClosed/PortFiltered`), `ServiceInfo`, `PortScanResult`
      (they sit in the `QuickScan` signature). `TCPProbeResult` moved to fingerprint (no
      kernel port uses it). Two unexported strings (`serviceUnknown`, `errNoIPv4ForTarget`)
      stay in the kernel for staying callers (the classifier / traceroute) with private
      copies in fingerprint. api re-points `New{PortScanner,TCPProber}` → `fingerprint.*`
      and keeps `discovery.PortScanResult` (kernel result type). depguard
      `discovery-stage-direction` extended to ban the kernel from importing fingerprint.
      Golden byte-identical, schema unchanged. `Fingerprinter` + `Tracer` were never in
      the enrich path and remain kernel-resident.
- [ ] 3. `resolve` stage → `discovery/resolve` + depguard
- [ ] 4. `enumerate` stage → `discovery/enumerate` + depguard
- [ ] 5. direction-lock depguard + cleanup + §16 doc
