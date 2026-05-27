# i18n tooling

Shell tooling for verifying translation files against the cross-repo
conventions documented in `msn-docs-internal/05-Engineering/`:

- [`I18N_CONVENTIONS.md`](../../../msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md) — framework, file structure, CI rules
- [`I18N_GLOSSARY.md`](../../../msn-docs-internal/05-Engineering/I18N_GLOSSARY.md) — terms preserved verbatim in all secondary locales
- [`I18N_STYLE_GUIDE_ES.md`](../../../msn-docs-internal/05-Engineering/I18N_STYLE_GUIDE_ES.md) — Spanish style guide

Identical layout across **seed / stem / niac** so the same tooling
applies everywhere with no per-repo customization.

## Files

| File | Purpose |
|---|---|
| `validate.sh` | Main validation script. Runs in CI; can run locally. |
| `glossary.txt` | One term per line — must appear verbatim in es when present in en. Mirror of the Glossary doc. |
| `banned-vocab.txt` | One term per line — must NOT appear in any locale file. Mirror of CLAUDE.md banned list. |

## Usage

```bash
# Run all checks against the standard locale layout
./scripts/i18n/validate.sh

# Skip the slower hardcoded-JSX scan
./scripts/i18n/validate.sh --quick

# Run a single check
./scripts/i18n/validate.sh --check key-parity
```

Path overrides via env vars:

```bash
LOCALES_DIR=path/to/locales \
UI_SRC_DIR=other/src \
GLOSSARY_FILE=path/to/glossary.txt \
BANNED_FILE=path/to/banned.txt \
./scripts/i18n/validate.sh
```

## What it checks

| Check | Failure means |
|---|---|
| key-parity | Locale files have drifted; a key exists in en but not es, or vice versa |
| no-empty-values | At least one locale value is `""` |
| no-fallback-patterns | Source uses banned `t('key', 'English fallback')` shortcut |
| banned-vocab | A locale value contains a banned term (CLAUDE.md) |
| glossary-preservation | A glossary term appears in en but not verbatim in es for the same key |
| interpolation-parity | `{{var}}` tokens differ between en and es for the same key |
| plural-completeness | A `_one` key exists without `_other` or vice versa |
| locked-versions | `package.json` pins to a version other than the I18N_CONVENTIONS lockstep target |
| hardcoded-jsx | (warn-only) JSX heuristic flag — manual review needed |

## CI integration

Each repo's `.github/workflows/ci.yml` has an `i18n Validation` job
that calls this script. Job blocks merge on any failed check.

A lighter pre-commit hook covers the no-fallback-patterns and
hardcoded-jsx checks locally, before push.

## Updating the glossary / banned list

1. Update the canonical doc in `msn-docs-internal/05-Engineering/`.
2. Update this repo's `glossary.txt` / `banned-vocab.txt` to match.
3. Repeat for the other two repos (cross-repo lockstep — see
   `I18N_CONVENTIONS.md` "Adding a new key" section).

## Dependencies

- bash 4+
- jq
- grep (GNU or BSD)
- find

Available in the GitHub Actions ubuntu-latest runners by default; on
macOS dev machines `brew install jq` if missing.
