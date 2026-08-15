# CI/CD Pipeline

The CI pipeline runs on every push and PR. **All checks must pass.**

Every job except `changes` is gated on the `changes` job's path filters, so a
docs-only PR does not pay for a Go build. `ci-complete` is the single required
status check — it depends on every other job, so adding a job to `ci.yml`
without adding it to `ci-complete`'s `needs:` list makes that job advisory.

## GitHub Actions Workflows

### ci.yml - Main CI Pipeline

| Job             | Description              | Checks                                                                 |
| --------------- | ------------------------ | ---------------------------------------------------------------------- |
| `changes`       | Path filtering           | Decides which downstream jobs run                                      |
| `build-ui`      | Shared frontend build    | Builds `internal/api/ui/` once and uploads it as an artifact           |
| `backend`       | Go checks                | lint, vet, staticcheck, fmt, tests, coverage floor (50%)               |
| `race`          | Go race detector         | `go test -race`, split from `backend` so it fails distinctly           |
| `frontend`      | React/TS checks          | tsgo typecheck, Biome, design-token gate, Vitest, lockfile integrity   |
| `storybook`     | Component tests          | Storybook interactions and axe accessibility                           |
| `security`      | Security scans           | govulncheck (hard gate), gosec, npm audit, gitleaks, Trivy             |
| `semgrep`       | SAST                     | Semgrep rules                                                          |
| `quality`       | Code quality gates       | banned vocabulary, file size ratchet, output escaping, sensitive files |
| `c-lint`        | C lint (C23)             | clang-format, clang-tidy                                               |
| `workflow-lint` | Workflow static analysis | actionlint; zizmor (blocks on High)                                    |
| `i18n`          | Internationalization     | Catalog completeness, no translated standard terms                     |
| `docs`          | Documentation            | Markdown lint (blocking, scoped to changed files)                      |
| `build`         | Build verification       | Multi-arch binaries with full ldflags, UIBuildHash verified            |
| `e2e`           | Browser tests            | Playwright, chromium + webkit, 2 shards each                           |
| `e2e-smoke`     | Smoke tests              | Fast critical-path Playwright run                                      |
| `lighthouse`    | Frontend performance     | Lighthouse budgets                                                     |
| `ci-complete`   | Aggregate gate           | The required status check                                              |

### Other Workflows

| Workflow              | Purpose                                            |
| --------------------- | -------------------------------------------------- |
| `codeql.yml`          | CodeQL security analysis (Go, JS/TS)               |
| `dead-code.yml`       | Weekly dead code detection                         |
| `docs-link-check.yml` | Weekly external link check (split out of `ci.yml`) |
| `label-sync.yml`      | Sync label definitions                             |
| `labeler.yml`         | Auto-label PRs and issues                          |
| `license-check.yml`   | Verify dependency licenses                         |
| `pr-body-lint.yml`    | Enforce the PR body template                       |
| `release-please.yml`  | Automated version management and release PRs       |
| `release.yml`         | goreleaser release builds, signing, provenance     |
| `scorecard.yml`       | OpenSSF Scorecard                                  |
| `title-lint.yml`      | Lint PR and issue titles                           |
| `todo-tracker.yml`    | Weekly TODO tracking                               |
| `update-oui.yml`      | Refresh the OUI vendor database                    |

## Shared UI artifact

`build-ui` runs the Vite build once and publishes `internal/api/ui/` as an
artifact. Jobs that need embedded assets — `backend`, `race`, `build`, `e2e`,
`e2e-smoke` and `lighthouse` — download it rather than rebuilding.

`frontend` does not run the Vite build at all; it typechecks, lints and unit
tests. It still declares `needs: build-ui` so a broken build fails the pipeline
early rather than surfacing several jobs later. `storybook` builds its own
static Storybook, which is a different artifact.

## Build contract

`build` verifies the Universal Build Contract, not just that compilation
succeeds: every binary embeds `Version`, `Commit`, `BuildTime` and
`UIBuildHash`, and the job asserts `UIBuildHash` is non-empty. A raw `go build`
in CI produces a binary whose `/__version` reports `"unknown"` — the silent
failure that check exists to catch.

## Workflow security

`workflow-lint` runs two scanners over `.github/workflows/` itself:

- **actionlint** — syntax, expression and shell errors inside `run:` blocks. It
  catches things a plain YAML parse does not; it found a duplicated `with:` key
  in `ci.yml` that `yaml.safe_load` accepts silently by keeping the last one.
  `SC2129` is ignored as a pure style preference; every correctness rule stays on.
- **zizmor** (pinned 1.29.0) — Actions security scanner. **Blocks on High
  findings.** The repo sits at zero High. One finding survived review and carries
  a `# zizmor: ignore[...]` comment with the reasoning inline (in
  `release-please.yml`); anything else that reaches High fails the build.
  Low/Informational are reported but not enforced.

Permissions follow least privilege: workflows declare `permissions: {}` (or
`contents: read`) at the top level and grant scopes per job. A new job that needs
a write scope declares it on the job, never workflow-wide. `release.yml`
deliberately runs without npm caching, because its output is published and
attested and a restored cache entry could land inside a signed artifact; it opts
out by passing `cache: ""` to the `setup-node` composite action.

## The Node.js pin lives in one file

Every workflow that needs Node uses `./.github/actions/setup-node`; none pin a
`node-version:` literal. The composite is the single place the Node and npm
versions are declared, and it must stay in step with `.nvmrc` and the `engines` /
`packageManager` fields in `package.json`. Release and CI therefore build on the
same Node version by construction rather than by convention.

## CI Must Pass Before Merge

`main` is protected. Push a feature branch, open a PR, and let CI gate it:

```bash
gh pr create --fill
gh pr merge --auto --squash --delete-branch
```

Fix issues locally first:

```bash
make all       # Full local verification
make verify    # lint, test, security, build
make test-e2e  # Playwright E2E
```

## Running CI Checks Locally

### Backend

```bash
make lint-backend      # golangci-lint v2.12.2
make test-backend      # Go tests
make test-coverage     # Coverage report
make security-backend  # gosec + govulncheck
```

### Frontend

```bash
make lint-frontend     # Biome + tsgo
make test-frontend     # Vitest
make build-frontend    # Vite build into internal/api/ui/
cd ui && npm run test:storybook
```

### Security

```bash
make security          # All security scans
make security-secrets  # gitleaks
make security-trivy    # Trivy
```

### E2E

```bash
make test-e2e          # Playwright headless
make test-e2e-ui       # Playwright with UI
```

### Workflows

```bash
actionlint -ignore 'SC2129'
zizmor --min-severity high .github/workflows/
```

## Debugging CI Failures

1. **Read the error** — CI logs show exactly what failed
2. **Run locally** — `make all` should catch the same errors
3. **Pull the trace** — for Playwright failures, download the trace artifact and
   read `error-context.md` rather than guessing at a fix
4. **Fix locally first** — don't push speculative fixes to see if CI goes green

## CI Environment

- **Go**: version from `go.mod`
- **Node**: pinned in `.github/actions/setup-node` and `.nvmrc`
- **OS**: Ubuntu latest (CI), macOS (local dev)
