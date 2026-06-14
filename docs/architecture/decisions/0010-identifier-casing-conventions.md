# ADR-0010: Identifier casing conventions (camelCase JSON wire, snake_case files/SQL)

**Status:** Accepted — 2026-06-05 · **Revised 2026-06-14** (see "Revision" at end — wire is now 100% camelCase; the config-blob and external-key wire exceptions are removed in favour of boundary mapping)

## Context

A casing audit (prompted while resolving the profile/settings types in S6) found the
codebase's **JSON wire casing is inconsistent**, while Go identifiers, SQL columns, and
TypeScript identifiers are consistent and idiomatic:

| Layer | Convention in use | State |
|---|---|---|
| Go identifiers | PascalCase exported / camelCase unexported | consistent (gofumpt/revive) |
| TypeScript identifiers | camelCase | consistent (Biome) |
| SQL columns | snake_case | consistent (DB norm) |
| **JSON `json:"..."` wire tags** | **mixed** | **inconsistent** |

Measured JSON-tag casing (non-test):

- `internal/api` — camelCase **235** / snake_case **102**
- `internal/discovery` — camelCase **192** / snake_case **190** (≈50/50)
- `internal/config` — camelCase 75 / snake_case **169** (config-file format)
- `internal/database` — camelCase 106 / snake_case 0

The UI also leaks snake_case object keys (`last_seen`, `client_id`, `is_default`, …) where
it echoes backend snake fields. The acute pain point surfaced in S6: the per-profile config
blob is *both* a config-file format (snake) *and* an API payload, so the two conventions
collide.

## Decision

The canonical casing conventions for seed (and, going forward, stem and niac):

1. **JSON API wire tags → camelCase.** Every `json:"..."` tag on a type that crosses the
   HTTP API boundary is camelCase. The UI is TypeScript (camelCase), and camelCase is the
   JS/JSON-API norm. This is the convention the codebase converges on.
2. **Config file format (`internal/config`, on-disk YAML/JSON) → snake_case.** A principled
   exception: snake_case is the conventional config-file style. The per-profile config blob
   delivered over the API is config-file content and therefore also snake_case — the one
   API payload that is snake by design (the UI echoes it rather than rebuilding it).
3. **SQL columns → snake_case.** Unchanged (DB norm).
4. **Go identifiers → Go standard (Pascal/camel); TypeScript identifiers → camelCase.**
   Unchanged; already enforced.
5. **Protocol-mandated keys keep their spec casing.** OAuth (`client_id`, `client_secret`,
   `redirect_uri`), SNMP, and other external-contract fields stay as the external spec
   dictates, even if snake_case — they are allow-listed exceptions, not drift.

### File and directory naming (audited 2026-06-05 — already consistent)

A full naming audit confirmed these are already followed everywhere; recorded here so they
stay that way:

| Artifact | Convention | Example |
|---|---|---|
| Go source files | snake_case | `config_types_network.go` |
| Go packages / directories | short, lowercase, no underscores | `internal/discovery` |
| Go command directories | kebab-case allowed (binary name) | `cmd/seed-schema` |
| SQL migration files | goose `NNNNN_snake.sql` | `00003_job_idempotency.sql` |
| SQL tables / columns | snake_case | `polling_targets.credentials_id` |
| Shell scripts | kebab-case | `check-json-casing.sh` |
| Config files (on disk) | on-disk *format* is a per-product best-practice choice; seed = JSON (`.json`), snake_case keys | `configs/seed.json`, `"jwt_secret"` (amended 2026-06-05 — see below) |
| Generated JSON schema files | kebab-case | `engine-discovery-response.schema.json` |
| UI React components (`.tsx`) | PascalCase | `NetworkDiscoveryCard.tsx` |
| UI hooks (`.ts`) | `useXxx` camelCase | `useEngineScan.ts` |
| UI utilities / modules (`.ts`) | camelCase | `jobsClient.ts` |
| UI generated types (`.ts`) | kebab-case (mirror their schema file) | `job-response.ts` |

The only naming debt in the codebase is the JSON wire-tag casing (item 1) — everything above
already conforms.

## Consequences

- **Phase 8** normalizes the snake_case JSON wire tags in `internal/api` (102) and
  `internal/discovery` (190) to camelCase, with a `scripts/check-json-casing.sh` CI gate to
  prevent re-drift (allow-listing `internal/config`, DB models, and protocol-mandated keys).
  Each change is a wire-contract change: edit tag → regenerate schemas + TS → fix consumers
  (tsc + grep) → golden regen → verify. Sequenced in `SEED_PHASE8_CASING_PLAN.md`.
- This ADR is the standard new code is held to; the gate makes it enforceable rather than
  aspirational (the lesson from the design-token gate).
- stem and niac adopt the same *casing* convention + gate during their re-architectures (the
  seed template is mirrored, per the no-master, harmonized-by-convention rule). On-disk config
  *format* (JSON vs YAML) is decided per product on its own merits — see the amendment below.

## Amendment (2026-06-05) — config on-disk format is JSON for seed, decided per product

The original "Config files (on disk)" row claimed `.yaml`, and the naming audit recorded it as
"already followed everywhere." That was wrong on the substantive point: seed's config **loader
has only ever been `encoding/json`** (`internal/config/config_load.go` — `json.Unmarshal` /
`json.MarshalIndent`). The `.yaml` filenames (`configs/seed.yaml`, `internal/paths` defaults,
deploy scripts, docs) were aspirational drift that was never wired — the shipped `configs/
seed.yaml` was real YAML the JSON loader could not parse, and `internal/paths` resolved
`config.yaml` while `seed install` wrote `seed.json`, so the installed file was never loaded
(arch-review finding #1).

**Correction:** for **seed**, the on-disk config format is **JSON** (`seed.json`), aligning
docs/paths/deploy/sample to what the engine actually does (JSON Schema validation and the
casing gate are JSON-native too). The casing rule is **unchanged and universal**: config-file
keys stay **snake_case** (`"jwt_secret"`), distinct from the camelCase JSON *wire* convention.

On-disk format is **not** a harmonized-across-products decision. Each product chooses on its
own merits: seed is machine-managed (setup wizard + API write the file) and env-var-dominated,
so JSON fits; a product with hand-authored, comment-heavy config (e.g. NIAC simulation
scenarios) may legitimately choose YAML. Only the *casing* convention is mirrored fleet-wide.

## Revision (2026-06-14) — pure boundary mapping; the wire is 100% camelCase, no exceptions

The original Decision carried two snake-case exceptions **on the wire**: the config-blob "snake
by design" payload (§2, last sentence) and the protocol/external-key allow-list (§5). On review
they are **pragmatic shortcuts, not best practice**, and they conflict with our own
**ADR-0020 (clean hexagonal API foundation)**: a ports-and-adapters boundary presents *one*
uniform contract and maps everything foreign at the edge. We are pre-`v1.0.0` (no wire-compat
burden), so we fix this now rather than grandfather it.

**Revised rule — supersedes §2's wire clause and §5 entirely:**

> **Every field our API emits or accepts is camelCase. There are no wire-level snake_case
> exceptions and no wire allow-list / baseline.** snake_case exists only *off* the wire:
> 1. **Config files on disk** (snake keys, e.g. `"jwt_secret"`) — unchanged.
> 2. **SQL columns** — unchanged.
> 3. **Internal adapters that talk to an external system** — the structs that *parse* an
>    external tool's output (iperf3 `-json`, macOS `system_profiler`) or *call* an external
>    spec (OAuth `client_id`) match that system's casing **inside the adapter package only**,
>    and are **mapped to camelCase before crossing our API boundary**. They are never
>    re-emitted verbatim and never on a type the API serializes.
>
> Therefore the config blob delivered over the API is **camelCase** (mapped from the snake
> config file by the loader), and `scripts/check-json-casing.sh` runs with an **empty
> baseline** — there is nothing legitimate to allow-list, because nothing foreign reaches the
> wire.

**What stays the same:** §1 (camelCase wire), §3 (SQL snake), §4 (Go/TS identifiers), the
on-disk config-file snake rule, and the **file/directory naming table** — in particular
**Go source files remain snake_case** (`config_types_network.go`); that is idiomatic Go and is
*not* affected by this revision. The only naming work is killing monolith *stutter prefixes*
during decomposition (ADR-0016), which `check-filename-policy.sh` already enforces.

**Consequences of the revision (tracked, per repo):**
- **seed** — map the ~13 macOS `system_profiler` keys (`internal/discovery`) and any config-blob
  fields to camelCase at the API edge; the snake parsing structs stay inside the adapter. Then
  **empty `scripts/json-casing-baseline.txt`** and drop the allow-list language from the gate.
- **stem / niac** — adopt this revised ADR verbatim (their own ADRs reference it); build to
  100% camelCase wire with an empty baseline from the start. niac's config-heavy API carries
  the most boundary-mapping work (its YAML serialization is already decoupled —
  `serializeDeviceToYAML` uses literal snake keys — so camelCasing the wire does not touch the
  config files).
