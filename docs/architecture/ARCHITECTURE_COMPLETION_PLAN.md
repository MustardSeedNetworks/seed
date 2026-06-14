# Seed — Architecture Completion Plan (no shortcuts)

> **Status:** APPROVED — 2026-06-12. Successor to `RE_ARCHITECTURE_BLUEPRINT.md`
> (Phases 0–7) and `PHASE3_RECONCILE_PROPOSAL.md`: the finish-line plan that takes
> the strangle from "complete for purpose" to **actually complete**.
> Owner directive 2026-06-12: **complete the ENTIRE re-architecture — no shortcuts,
> no "complete for purpose" stopping points, file/folder names correct.**
> Driven by the 3-agent conformance audit of the same date (findings below).
> Plan of record. Each item = its own golden-gated PR; Opus orchestrates + gates,
> Sonnet fans out mechanical execution; every PR green before merge.

## Audit verdict (2026-06-12) — what's actually wrong

The reorg's bones are sound: layering *direction* clean (0 domain→`internal/api`
imports), capability-first packages, legacy `internal/{modules,adapters,services}`
gone, botanical names in code = 0, security CI gates real. **But the strangle
stalled and was papered over:**

1. **ADR-0020 half-applied (~54%).** 17 of 37 non-auth handlers still carry inline
   orchestration + direct `s.db()` / `s.config.*` access. Worst: `devices` (41),
   `settings` (37), `security` (32), `health_checks_settings` (28), `vuln` (11),
   plus raw `s.db().Repo()` CRUD in `topology`, `polling_targets`, `alerts`.
2. **Repository-port discipline bypassed in 17 files / 7 packages.** `probe`,
   `topology`, `identity`, `listener`, `timeseries`, `polling`, `alerts` import
   `internal/database` directly with no consumer-side port and no depguard gate.
3. **depguard covers only `internal/reporting`.** 6+ domain packages have zero
   purity rules — nothing prevents regression.
4. **Domain-purity leaks:** `net/http` in `internal/diagnostics/iperf/installer.go`
   (clean violation) + discovery I/O + `probe/checkers` (protocol-inherent, to be
   documented/excepted).
5. **CI gate defanged:** `golangci-lint --new-from-rev=HEAD~1` only lints the last
   commit (full-tree backlog measured = **1** issue, so low debt — but the gate is
   blind). File-size gate `STRICT=0` (warn-only): 22 Go files >700 LOC, 21 UI
   >500. Coverage floor 42% vs 90% goal.
6. **Pre-v1 no-compat violations:** live `config.ToLegacyConfig()`; `revive`
   stutter `//nolint`s in `logging`/`update` "for backward compatibility";
   deprecated fields/functions kept (`LinkUp`, `WarnDeprecatedSNMPSettings`).
7. **UI complexity hidden:** 19 `biome-ignore-all noExcessiveCognitiveComplexity`
   file-level blankets on oversized components.
8. **3 dodge `t.Skip`s** (unimplemented / unwired / upstream-race).
9. **Deferred blueprint phases:** Phase 3.x IA & API-taxonomy redesign; Phase 7
   discovery-orchestrator convergence (ADR-0007 Amended); docs reconcile.

## Naming/folder rules (ADR-0010 + `check-filename-policy.sh`) — enforced throughout

- Go dirs/packages: short, **lowercase, no underscores** (`internal/discovery`).
- `cmd/` dirs: kebab allowed (binary name). Shell scripts: kebab. Generated
  JSON-schema + UI `.ts` types: kebab.
- **`handlers_*`/`jobs_*` prefixes are legal ONLY inside `internal/api`** (monolith
  grouping). When a concern is decomposed into a capability package, **drop the
  prefix** — name files by role within the package (`health.go`, `checks.go`,
  `handler.go`, `repository.go`). The filename-policy gate enforces this.
- Net effect: **strangling a handler out of `internal/api` IS the naming fix.** The
  thin transport that stays in `internal/api` keeps its `handlers_*` name; the
  extracted application-service/port/adapter files get role-based names in their
  package.

### Use-case = the concept, never the name (ADR-0020 §2)

"Use-case" is the **architectural concept** (clean-hexagonal application service);
it is **never a code identifier**. The application service + its consumer-defined
ports live in a **domain-meaningful package named for the capability**
(`internal/<domain>/<capability>`), with a **domain-meaningful type name** —
`Queries`, `Management`, `Discovery`, `Service` (cf. `troubleshooting.Queries`,
`monitoring.Service`). **Never** `…UseCase`, `internal/<domain>/usecase`, or
`internal/<domain>/app`. Verified 0 `usecase` identifiers in the tree — keep it 0.
Wherever this plan says "use-case", read it as "the capability's application
service, named domain-meaningfully." Adapters that satisfy the ports live in
`internal/app` (the composition root).

---

## Workstreams (the entire plan)

### WS-A — Complete the ADR-0020 handler strangle (100%, the core "away from handlers")
Every fat/mixed **non-auth** handler → thin transport over a use-case + repository
port. New files named per the rules above. One golden-gated PR each.

- **A1 `devices`** (41) → `diagnostics`/`discovery` settings use-case + device/discovery repo port.
- **A2 `settings`** (37) → extend `settingsapp` to own the READ assembly (not just save); kill inline `s.config.*` map-building + threshold conversion.
- **A3 `security`** (32) → security settings use-case (SNMP/DHCP-rogue/password-masking rule).
- **A4 `health_checks_settings`** (28) → probe-settings use-case (ties ADR-0027); single data source.
- **A5 `vuln`** (11) → diagnostics vuln settings use-case.
- **A6 `topology`** (4 db) → topology read use-case + repo port.
- **A7 `polling_targets`** (5 db) → polling-targets use-case + repo port (NOT "leave thin" — owner: no shortcuts).
- **A8 `alerts`** (2 db) → alerts read use-case + repo port.
- **A9 `config`** → move `BackupManager`/`config.Load` construction to composition root; thin handler.
- **A10 `status` export** → status-export use-case coordinating the 8 card sources.
- **A11 residuals** `discovery`, `dns`, `logs`, `path`, `tools`, `network` (9 funcs), `guest_audit`, `profiles`/`wifi` minor reaches.
- **Exit:** no non-auth handler reaches `s.db()`/`s.config.*`; every handler file thin; god handler files <~300 LOC.
- **Auth workstream (`mfa`/`auth`/`recovery`/`oauth`): coordinate, do not cross** — its own ADR-0024 track.

### WS-B — Repository-port discipline everywhere
The 7 bypass packages define consumer-side repo ports + receive concretes from
`internal/app`; remove direct `internal/database` imports.
- B1 `probe` (`lifecycle.go`) · B2 `topology` (4 reconcilers) · B3 `identity`
  (users/oauth/tokens) · B4 `listener/sink` · B5 `timeseries/retention` ·
  B6 `polling/snmp` · B7 `alerts/pipeline`.
- Each PR adds a **depguard rule** (RED-proven) banning `internal/database` in that
  package, so it can't regress.
- **Status (2026-06-14): B1–B2 + B4–B7 DONE** (probe #1672, topology #1673,
  listener/sink + alerts #1676, polling #1675, retention #1677, health #1678 — each
  with its RED-proven `*-no-persistence` depguard rule).
- **B3 `identity` — CLOSED as a documented exception, NOT a relocation.** It is the
  one intentional exception to this workstream. ADR-0024 (Accepted 2026-06-10)
  deliberately kept the identity repository ports typed on the existing
  `*database.User` / `database.APITokenRecord` / `database.SSOUserInput` rows
  (plus the `ErrUser*`/`ErrLastAdmin` sentinels): "thin CRUD stays thin — inventing
  a DTO here is churn without a security or clarity gain" on the codebase's most
  security-sensitive store. The repository-port win (no raw store handle in handler
  bodies) already landed in #1624; relocating the row types would reverse a reasoned
  ADR for no gain. So `internal/identity` keeps its `internal/database` import by
  design and gets **no** `identity-no-persistence` deny rule — the exception is
  recorded as a comment in `.golangci.yml` and in ADR-0024 §Consequences. This makes
  WS-B complete: every domain package has a purity rule **or** a documented exception.

### WS-C — Domain purity (net/http) + depguard coverage
- C1 Move `iperf/installer.go` net/http into an adapter (composition root or `internal/app`).
- C2 Discovery outbound HTTP (`profiler`, `resolve/oui`, `vuln/cve_*`) → confine behind I/O ports OR carve documented depguard exceptions (these are I/O adapters by nature).
- C3 `probe/checkers` net/http: document as protocol-inherent; scoped depguard exception with rationale.
- C4 Add **domain-purity depguard rules for every capability package** (`wifi`, `security`, `network`, `health`, `settings`, `profiles`, `anomaly`, `diagnostics`, `probe`, `topology`, `identity`) — replicate the `reporting-*` pattern. Close all 6+ gaps.

### WS-D — God-file decomposition (so the size gate can go strict)
Decompose every non-test Go file >700 LOC and UI file >500 LOC by role. Overlaps
WS-A (handlers shrink when strangled). Non-handler targets incl. `snmp/interface.go`
(1196), `repository_discovery.go` (1157), `dhcp.go` (1002), `wifi/survey/*`,
`mibdb/builtin_oids.go` (data — assess), `repository_topology.go`. UI: `helpDrawerContent`
(1282 — data), `useSurvey`, settings sections, survey components.

### WS-E — Kill legacy/compat shims (pre-v1 no-compat law)
- E1 Remove `config.ToLegacyConfig()` + migrate callers to `SystemConfig`.
- E2 Resolve `inMemorySuppressionStore` "legacy backend".
- E3 Rename the `revive`-stutter `//nolint`s away (`logging`, `update/types`) — real renames, delete suppressions.
- E4 Remove deprecated `netif.LinkUp` field, `WarnDeprecatedSNMPSettings`, assess `survey/migration.go`.
- E5 Fix the 4 borderline `//nolint`s (errcheck-without-log, gocyclo-instead-of-decompose).

### WS-F — Naming/folder conformance sweep
- F1 Verify ADR-0010 + filename-policy hold after every WS-A/B decomposition (gate stays green).
- F2 Fix stale botanical comments (`discovery/enumerate/wifi_bridge.go` "canopy").
- F3 Audit package/dir names tree-wide against ADR-0010; fix any underscore dirs or stutter files.

### WS-G — UI complexity + size
- G1 Decompose the 19 `biome-ignore-all noExcessiveCognitiveComplexity` files; remove the blankets (no file-level suppression survives).
- G2 UI god files >500 LOC decomposed (data files like `helpDrawerContent`/`builtin_oids` exempted with rationale).

### WS-H — Tests & coverage
- H1 Implement or delete the 3 dodge `t.Skip`s.
- H2 Raise coverage floor toward the 90% goal; ratchet the CI floor up each merge.

### WS-I — Remaining blueprint phases
- I1 **Phase 3.x** — ADR for the `/api/v1/*` resource taxonomy + sidebar IA redesign, then execute (registry-driven, regen goldens, UI fetch sites + nav).
- I2 **Phase 7** — finalize ADR-0007 discovery engine-vs-pipeline convergence + `/jobs` consumption.
- I3 Docs reconcile — CLAUDE.md + msn-docs architecture trees → capability-first layout + route prefixes.

### WS-Z — Gate keystone (LAST — locks "done" in place)
Only after the backlog above is cleared:
- Z1 `golangci-lint` → **full-tree** (drop `--new-from-rev=HEAD~1`); fix the 1 residual.
- Z2 File-size gate `STRICT=1` (blocking) with the agreed thresholds.
- Z3 Add a depguard regression gate verifying every domain package has a purity rule.
- Z4 Coverage floor raised to the new real number, ratchet enforced.

---

## Sequencing
WS-A and WS-B interleave (strangling a handler often creates the port that fixes a
bypass). WS-C/E/F ride alongside per package. WS-D overlaps WS-A. WS-G/H parallel
(UI vs Go). WS-I after the layer is uniform. **WS-Z is the final keystone** — it
must land last, or it reds the board and blocks the very PRs that fix the backlog.

## Execution model
Opus main loop owns design + review + gating. Sonnet subagents fan out the
mechanical per-handler strangles and per-package port extractions (each opens a PR,
Opus reviews/gates). Every PR: build + `-race` on touched pkgs + `internal/api`
alone if touched + `--new-from-rev` lint 0 + escaping/route-policy/filename-policy
+ goldens. Conventional commits, feature branch, auto-merge on green.

## Tracking
Workstream-level tasks in the session task list; this doc is the durable index.
