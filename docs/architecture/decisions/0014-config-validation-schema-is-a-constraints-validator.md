# ADR-0014: internal/config/schema.json is a curated constraints validator, not a duplicate

**Status:** Accepted — 2026-06-05

## Context

Phase 7 S6.1 (#1510) added a code-first generated config schema —
`docs/schemas/api/config.schema.json`, reflected from `config.Config` by
`cmd/seed-schema` — as the drift-gated backend contract for the UI's generated
`Config` TypeScript type. That raised a follow-up (S6.3): retire the older
hand-maintained `internal/config/schema.json` (embedded by `internal/config/
schema.go` and used at runtime by `ValidateConfig` / `ValidateWithSchema`),
treating it as a duplicate now superseded by the generated schema.

Inspecting the two schemas shows they are **not** duplicates:

| Schema | Source | Captures | Constraint keywords |
|---|---|---|---|
| `docs/schemas/api/config.schema.json` | reflected from `config.Config` | **type/shape** | **0** |
| `internal/config/schema.json` (embedded) | hand-curated | **value constraints** | **82** (`minimum`/`maximum`/`enum`/`pattern`) |

The hand-maintained schema encodes runtime business rules a reflected
type-schema cannot express: server-port ranges, VLAN-ID bounds, IP-mode enums,
port-preset values, etc. The validation tests (`TestValidateConfig_Invalid*`)
depend on exactly these. A Go-struct → JSON-schema reflection yields field
types, not `minimum`/`maximum`/`enum` rules, so the generated schema has none.

## Decision

**Keep `internal/config/schema.json`. Do not retire it.** It is a curated
runtime **constraints validator**, a distinct artifact from the generated
type/shape contract — the same conclusion ADR-0009 reached for
`profile.ts`/`settings.ts` (a thing that looks like a duplicate but serves a
different purpose). The two are complementary:

- `docs/schemas/api/config.schema.json` — code-first, drift-gated, the *shape*
  contract the UI's generated `Config` type mirrors.
- `internal/config/schema.json` — hand-curated, embedded, the *value-constraint*
  validator `ValidateConfig` enforces at runtime.

Retiring the constraints schema in favour of the generated one would silently
delete all config value validation (port/VLAN/IP-mode/preset bounds) — a
regression, not cleanup. S6.3 is therefore closed as "keep + document".

## Consequences

- Config value validation is preserved; no regression.
- The remaining genuine concern is *coverage drift* — a new `config.Config`
  field could be added without a matching constraint entry, leaving it
  unvalidated. The generated schema already tracks the struct's *shape* (its
  drift gate fires on shape changes), which surfaces new fields; wiring a
  dedicated "every Config field has a constraints-schema entry" guard is a
  possible future enhancement, not required here.
- If a single source is ever wanted, the path is to annotate `config.Config`
  fields with constraint metadata and extend `cmd/seed-schema` to emit
  `minimum`/`maximum`/`enum` — then the generated schema could subsume the
  hand-maintained one. That is a deliberate, larger investment on a
  validation-critical path, explicitly out of scope for S6.3.
