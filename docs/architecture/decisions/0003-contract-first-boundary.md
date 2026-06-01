# ADR-0003: Contract-first OpenAPI boundary

**Status:** Accepted — 2026-05-31

## Context

Only ~6 of ~120 routes flow through the JSON-Schema→TS generation pipeline. The rest
are hand-typed on both the Go and TS sides, and the two largest type files
(`profile.ts` ~601 LOC, `settings.ts` ~757 LOC) are hand-maintained in TS with no
enforced link to their Go counterparts. This is the single largest latent-bug surface:
the frontend/backend contract drifts silently.

## Decision

An **OpenAPI 3.1 spec in `contract/` is the source of truth.** Generate Go transport
DTOs (`oapi-codegen`, types mode) into `adapters/http`, and the TS client
(`openapi-typescript`). Greenfield (no consumers) → design the API we want;
reverse-generate only as a starting draft, then refine.

- Transport DTOs live in `adapters/http`; handlers map DTO↔domain so `modules/` never
  import contract types (purity preserved).
- Spectral lints the spec in CI (consistent error envelope, RFC3339, pagination).
- `check-contract-drift.sh` regenerates and fails on diff (extends the existing
  schema-drift gate to the whole boundary).
- A gated Redoc reference is auto-published from the spec.
- Roll module-by-module (Sap first). End state: delete hand-written `client.ts` and
  hand-maintained domain types.

## Consequences

- The FE/BE contract cannot drift — generation + drift gate enforce it.
- Clean API taxonomy, not inherited wartiness (greenfield).
- Adds a codegen step to the build; CI must run it and gate on drift.
- New endpoints must start from the spec, not the handler.
