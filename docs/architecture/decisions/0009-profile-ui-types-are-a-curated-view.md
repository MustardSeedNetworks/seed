# ADR-0009: The profile.ts / settings.ts UI types are a curated view, not a config.Config mirror

**Status:** Accepted — 2026-06-05

## Context

Phase 7 S6 set out to retire the hand-written `ui/src/types/profile.ts` (~596 lines,
46 types) and `ui/src/types/settings.ts` (~757 lines, 53 types) "twins" — long
flagged (DEFERRED #3) as a hand-maintained mirror at risk of drifting from the
backend. The per-profile config blob is applied via `Config.ApplyProfileJSON`, so
`config.Config` is its canonical backend shape.

S6.1 (ADR-0008, #1510) registered `config.Config` code-first and generated
`ui/src/types/generated/config.ts` (a `Config` type + all section sub-types),
intending to replace the twins with the generated type.

**A parity-mapping pass before migrating revealed the twins are not a mirror at
all — they are a deliberately distinct, curated UI view:**

- **Naming convention differs.** `config.Config` carries **snake_case** JSON tags
  (141 snake_case vs 9 camelCase) because it is the config *file* format; the
  generated `Config` type is therefore snake_case (`available_modes`,
  `custom_tests`, `ping_targets`). The entire rest of the seed API and the UI use
  **camelCase**. The hand types are camelCase.
- **Structure differs.** Of the 18 type names that overlap by coincidence of
  domain, none are structurally equal. Examples:
  - `ThresholdsConfig` — hand: `{latencyWarningMs, latencyCriticalMs,
    packetLossWarningPct, packetLossCriticalPct}`; generated: `{dhcp, dns, ping,
    wifi, link, custom_tests}`.
  - `HealthChecksConfig` — hand: 3 fields (`pingTargets, tcpChecks, httpChecks`);
    generated: 18 fields (every endpoint protocol + run-flags).
  - `DiscoveryConfig` — hand: `{additionalSubnets, scanIntervalSeconds}`;
    generated: `{protocol, timeout}` (entirely different fields).

So the UI types are a **simplified, camelCase, UI-shaped** projection — not a
1:1 reflection of `config.Config`. Replacing them with the generated `Config`
would push snake_case naming and the full config tree into ~64 consumer files,
degrading the UI's conventions for no functional gain. There are **zero exact
duplicate types** to delete.

## Decision

- **`profile.ts` / `settings.ts` are kept as a legitimate curated UI view.** They
  are not retired; the "twin" framing was inaccurate — they are not a stale mirror
  but a distinct projection.
- **DEFERRED #3 (retire the profile/settings twins) is resolved as "keep the
  view."** The original drift concern is addressed structurally: the *backend*
  contract now has a canonical, code-first, drift-gated representation
  (`config.Config` → `docs/schemas/api/config.schema.json` →
  `ui/src/types/generated/config.ts`, enforced by `check-types-drift.sh`). The
  hand UI types are a separate concern with no mirror obligation, so no
  parity-vs-`config.Config` guard is applicable.
- **S6.1's generated `Config` type stays** — it is the drift-gated backend config
  contract, available for future use (config-form generation, validation, a
  typed profile-config wire DTO) without forcing a UI remap now.
- **No `config.Config` JSON-tag normalization is undertaken.** Making the config
  file format camelCase to match the API would be a large change to the on-disk
  config format with backward-compatibility risk, and is not justified by the
  twins question alone.

## Consequences

- Phase 7 closes without a churny 64-file UI migration that would have traded the
  UI's camelCase conventions for `config.Config`'s snake_case file format.
- The generated `Config` type is landed and drift-gated, so a future, intentional
  effort (e.g. generating a profile-config editor, or a camelCase config-tag
  normalization) has a foundation to build on.
- **Open follow-up (not blocking):** the UI sends a profile config that
  `Config.ApplyProfileJSON` parses into the snake_case `config.Config`. Whether
  the UI's camelCase profile types serialize to the snake_case keys
  `ApplyProfileJSON` expects should be verified independently — a naming mismatch
  there would be a latent profile-apply bug, orthogonal to this decision.
- Supersedes the S6 portion of the blueprint that assumed the generated `Config`
  type would replace `profile.ts` / `settings.ts`.
