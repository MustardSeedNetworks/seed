# ADR-0003: Contract boundary (code-first, OpenAPI deferred)

**Status:** Amended — 2026-05-31 (supersedes the original OpenAPI-first decision below)

## Context

The frontend/backend contract is the single largest latent-bug surface: most
request/response types are hand-typed on *both* the Go and TS sides — the two
largest, `profile.ts` (~601 LOC) and `settings.ts` (~757 LOC), are hand-maintained
in TS with no enforced link to their Go counterparts, so the contract drifts silently.

**Correction (recorded deliberately).** This ADR originally recommended a
hand-authored **OpenAPI-first** boundary (author specs in `contract/`, add
`oapi-codegen`, replace the existing generator). That recommendation was made on an
**incompletely-verified view of the codebase** — it reached for the standard
"contract-first OpenAPI" best-practice label instead of reckoning with what already
exists. On actually reading `cmd/seed-schema`, there is already a working, CI-gated
**code-first** pipeline:

```
Go DTO ──seed-schema (invopop/jsonschema)──► docs/schemas/api/*.schema.json
                                                   │ json-schema-to-typescript (ui/scripts/gen-types.mjs)
                                                   ▼
                                            ui generated TS types
   gated by check-schema-drift.sh + check-types-drift.sh
```

The Go struct is the single source of truth; TS is generated; drift fails CI. The real
problem is **coverage** (6 of ~180 routes), not the absence of machinery.

## Decision

**Code-first. Extend the existing Go-first pipeline; do not hand-author specs.**

- **Go DTOs remain the single source of truth.** Widen `seed-schema`'s target list from
  6 DTOs to all request/response types, generating TS via the existing `gen-types`
  step, gated by `check-schema-drift.sh` + `check-types-drift.sh`.
- **Replace hand-maintained dual types** — once a DTO is generated, delete its
  hand-written TS twin (`profile.ts`, `settings.ts`, etc.) in favour of the generated
  type.
- **OpenAPI is a deferred, additive output, not a rewrite.** When there is a reader for
  it — published API docs (Redoc) or a third-party consumer — emit an OpenAPI 3.1
  document from the route registry manifest (`/__capabilities`, ADR-0002) + the
  seed-schema DTOs. This is a non-breaking bolt-on: the Go DTOs, schemas, registry and
  TS pipeline are unchanged; an emitter is added on top. This mirrors the existing
  org decision that API versioning is "deferred until 3rd-party needs arise."

Rejected: hand-authored OpenAPI-first (the original recommendation) — it discards a
working, CI-gated pipeline and adds a permanent spec-vs-implementation sync burden for
an API that currently serves only our own frontend with no external consumers.

## Consequences

- The FE/BE contract cannot drift for any covered DTO — generation + drift gate enforce it.
- Far less disruption than OpenAPI-first: extend coverage rather than replace tooling.
- The OpenAPI/Redoc capability is preserved as a future additive step, reusing the
  Phase-1 capability manifest — no rework, no lock-out.
- New endpoints add their DTO to the `seed-schema` target list; the schema + TS type
  generate from the Go struct.
