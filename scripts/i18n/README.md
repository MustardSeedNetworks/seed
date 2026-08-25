# i18n data

The i18n **checks** live once in
[`MustardSeedNetworks/.github`](https://github.com/MustardSeedNetworks/.github)
under `scripts/i18n/`. This directory holds only the data they read for this
repo, plus a shim so the gate still runs from a checkout.

They were maintained as near-copies in seed, stem and niac until 2026-08-24.
Every fix had to be applied three times and they drifted anyway — including one
that was fixed in stem and never reached here, leaving this repo's gate
reporting three unreferenced keys on every run while the job passed (#2085).

Conventions live in `msn-docs-internal/05-Engineering/`:

- [`I18N_CONVENTIONS.md`](../../../msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md)
  — framework, file structure, CI rules
- [`I18N_GLOSSARY.md`](../../../msn-docs-internal/05-Engineering/I18N_GLOSSARY.md)
  — terms preserved verbatim in all secondary locales
- [`I18N_STYLE_GUIDE_ES.md`](../../../msn-docs-internal/05-Engineering/I18N_STYLE_GUIDE_ES.md)
  — Spanish style guide

## Files

| File | Purpose |
| --- | --- |
| `validate.sh` | Shim. Reads the pinned shared-gate SHA from this repo's `ci.yml` and execs the canonical script. |
| `glossary.txt` | One term per line — must appear verbatim in es when present in en. |
| `banned-vocab.txt` | One term per line — must NOT appear in any locale file. |
| `glossary-exceptions.txt` | Per-key allow-list for glossary false positives. |
| `dynamic-prefixes.txt` | Key prefixes reached by data-driven lookup. Each entry needs a one-line WHY. |

## Usage

```bash
./scripts/i18n/validate.sh
```

There is exactly one pinned SHA per repo — the `uses:` line CI runs — so a
local run and CI cannot disagree. Renovate bumps it; the shim follows. The
first run for a given SHA fetches and caches under `~/.cache/msn-shared/`;
after that it never touches the network.

To change a **check**, edit `MustardSeedNetworks/.github` and bump the pin.
To change **this repo's data**, edit the files above.
