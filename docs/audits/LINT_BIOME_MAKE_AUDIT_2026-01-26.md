# Lint, Biome, and Make Audit - Seed (2026-01-26)

## Go Lint (golangci-lint)
- Config: `.golangci.yml` based on maratori “golden” strict config.
- Strictness: **High** (large linter set, gofumpt enabled).
- Status: Strict and consistent with Stem/NiAC.

## Biome
- Root `biome.json` extends `ui/biome.json` and scopes to `ui/**`.
- UI config is **very strict** (recommended + correctness/style/suspicious/performance/complexity/security/a11y/nursery mostly on).
- Overrides allow tests/e2e/scripts to relax rules slightly.

## Make System
- Makefile uses modular `mk/*.mk` with lint/security/test/package targets.
- Lint targets: golangci-lint + Biome; markdownlint optional.
- Formatting: gofumpt used in `fmt`, but `fix-backend` uses `gofmt -w -s` (slightly less strict than gofumpt).

## Findings
- No major gaps. System is already strict.
- Minor consistency: consider using `gofumpt` in `fix-backend` to match `fmt` behavior.
