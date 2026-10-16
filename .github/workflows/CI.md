# CI/CD Pipeline

The CI pipeline runs on every pull request and merge queue run. **All checks
must pass.** A push to `main` runs `ci-main-smoke.yml` instead of this suite:
the merge group already tested that commit, and release-please still needs a
successful workflow result on main.

`changes`, `backend`, `race`, `frontend`, `storybook`, and `e2e` call the
reusable workflows in `MustardSeedNetworks/.github`, pinned to
`e092081a97245d29978e229be7425179cc1dcde1`. macOS, Windows, and the
`GOOS=windows` lint run inside `backend`. The E2E daemon, the sharded browser
suite, and smoke run inside `e2e`. `CI Complete` stays a job in this file so
the required check keeps that exact name.

Every job except `changes` is gated on the `changes` job's path filters, so a
docs-only PR does not pay for a Go build. `ci-complete` is the single required
status check — it depends on every other job, so adding a job to `ci.yml`
without adding it to `ci-complete`'s `needs:` list makes that job advisory.

## GitHub Actions Workflows

### ci.yml - Main CI Pipeline

| Job             | Description              | Checks                                                                                                                                                   |
| --------------- | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `changes`       | Path filtering           | Shared `ci-changes`: decides which downstream jobs run                                                                                                   |
| `build-ui`      | Shared frontend build    | Builds `internal/api/ui/` once and uploads it as an artifact                                                                                             |
| `backend`       | Go checks                | Shared `ci-backend`: lint, vet, fmt, tests, macOS, Windows, coverage floor (55%)                                                                         |
| `race`          | Go race detector         | Shared `ci-race`: `go test -race`, split from `backend` so it fails distinctly                                                                           |
| `frontend`      | React/TS checks          | Shared `ci-frontend`: tsc typecheck, Biome, design-token gate, Vitest, lockfile integrity                                                                |
| `storybook`     | Component tests          | Shared `ci-storybook`: interactions and axe accessibility                                                                                                |
| `security`      | Security scans           | govulncheck (hard gate), gosec via golangci-lint (blocking gate; G103/G204/G304/G702 are SARIF-only via the standalone step), npm audit, gitleaks, Trivy |
| `semgrep`       | SAST                     | Semgrep rules                                                                                                                                            |
| `quality`       | Code quality gates       | banned vocabulary, file size ratchet, output escaping, sensitive files                                                                                   |
| `bridge-format` | Darwin bridge format     | clang-format on the cgo bridge headers (seed has no C)                                                                                                   |
| `workflow-lint` | Workflow static analysis | actionlint; zizmor (blocks on High)                                                                                                                      |
| `i18n`          | Internationalization     | Catalog completeness, no translated standard terms                                                                                                       |
| `docs`          | Documentation            | Markdown lint (blocking, scoped to changed files)                                                                                                        |
| `build`         | Build verification       | Multi-arch binaries with full ldflags, UIBuildHash verified                                                                                              |
| `e2e`           | Browser tests            | Shared `ci-e2e`: daemon, chromium + webkit, 2 shards, and the `@smoke` job                                                                               |
| `ci-complete`   | Aggregate gate           | The required status check, named exactly `CI Complete`                                                                                                   |

### Other Workflows

| Workflow                   | Purpose                                            |
| -------------------------- | -------------------------------------------------- |
| `ci-main-smoke.yml`        | Push-to-main signal for release-please             |
| `codeql.yml`               | CodeQL security analysis (Go, JS/TS)               |
| `dead-code.yml`            | Weekly dead code detection                         |
| `docs-link-check.yml`      | Weekly external link check (split out of `ci.yml`) |
| `label-sync.yml`           | Sync label definitions                             |
| `license-check.yml`        | Verify dependency licenses                         |
| `main-retry.yml`           | Retry a failed main smoke or CI dispatch once      |
| `merge-queue-ejection.yml` | Stop a PR the merge queue keeps ejecting           |
| `pr-hygiene.yml`           | Title lint, PR body template, auto-labels          |
| `release-please.yml`       | Automated version management and release PRs       |
| `release.yml`              | goreleaser release builds, signing, provenance     |
| `scorecard.yml`            | OpenSSF Scorecard                                  |
| `todo-tracker.yml`         | Weekly TODO tracking                               |
| `update-oui.yml`           | Refresh the OUI vendor database                    |

These jobs stay in `ci.yml` because the shared product workflows do not cover
them: `build-ui` (the production bundle and the first-load budget), `bench`,
`security`, `quality`, `release-notes`, `bridge-format`, `workflow-lint`,
`i18n`, `docs`, `build`, `iperf3`, and `phone-width`. `semgrep`,
`dependency-review`, `ci-conformance`, and `codeql-alert-gate` were already
calls into `MustardSeedNetworks/.github`. `ci-complete` stays local so branch
protection and `release.yml` keep matching the check name `CI Complete`.

## Shared UI artifact

`build-ui` runs the Vite build once and publishes `internal/api/ui/` as an
artifact. Jobs that need embedded assets — `backend`, `race`, `build`, and
`e2e` — download it rather than rebuilding.

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
- **zizmor** (pinned 1.30.1) — Actions security scanner. **Blocks on High
  findings.** The repo sits at zero High. One finding survived review and carries
  a `# zizmor: ignore[...]` comment with the reasoning inline (in
  `release-please.yml`); anything else that reaches High fails the build.
  Low/Informational are reported but not enforced.

Permissions follow least privilege: workflows declare `permissions: {}` (or
`contents: read`) at the top level and grant scopes per job. A new job that needs
a write scope declares it on the job, never workflow-wide. `release.yml`
deliberately runs without npm caching, because its output is published and
attested and a restored cache entry could land inside a signed artifact; it opts
out by passing `cache: ""` to the `setup-node` composite action. The shared
`ci-backend` workflow's golangci-lint steps opt out too, with `skip-cache:
true`: a restored analysis cache reported two `nolintlint` findings that were
not in the tree and survived a re-run (#2511), and its key carries neither
`GOOS` nor the linter config, so the `GOOS=windows` lint job shares one entry
with `Backend (Go)`.

## Disabled lint rule: `nursery/useAwaitThenable`

Off in `ui/biome.json` since Biome 2.5.8. That release made overload selection
type-aware, and Biome now picks React's `act(callback: () => VoidOrUndefinedOnly):
void` overload for `await act(async () => ...)`, reporting 16 awaits as awaiting a
non-thenable. TypeScript picks the `Promise<T>` overload instead, because
`VoidOrUndefinedOnly` is branded to reject Promise returns under `strictNullChecks`
(see the NOTES comment above `act` in `@types/react`). Verified with a type probe:
`const p: number = act(async () => {})` reports `Type 'Promise<void>' is not
assignable to type 'number'`. The awaits are correct, so the rule is off rather
than the tests rewritten. Re-enable when Biome models the brand. stem and niac-go
keep it at `error` — neither uses `await act(`.

## The Node.js pin lives in one file

`.nvmrc` is the single source of truth for the Node version. Jobs that stay in
this repo use `./.github/actions/setup-node`, and that composite reads
`.nvmrc` via `node-version-file` — it has **no `node-version` input**, so no
caller can override it and no second copy of the version can exist. The shared
backend, frontend, Storybook, and E2E workflows read the same `.nvmrc` and
install the composite's npm pin (`12.1.0`).

That input used to default to a literal, and it drifted: Renovate bumps the
manifests it can see and cannot see a default buried inside a composite, so CI
ran 26.7.0 against manifests demanding 26.8.1 and logged EBADENGINE on every
job for weeks. "Must stay in step" was the previous rule here, and a rule that
depends on someone remembering is not a mechanism.

The remaining pair that can disagree is `.nvmrc` and the `engines` field in
`package.json`. Making that a hard failure (`engine-strict=true`) is the obvious
next step and is deliberately **not** taken yet: Homebrew's newest `node` is
26.7.0, so 26.8.1 is not installable through the fleet's normal channel, and
turning the mismatch fatal would block local development in all four repos. See
the linked issue.

The npm version is declared in the composite and as the shared workflows'
`npm-version` default. `packageManager` in `package.json` is what `engines`
checks it against.

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
make lint-backend      # golangci-lint v2.14.0
make test-backend      # Go tests
make test-coverage     # Coverage report
make security-backend  # gosec (blocking via golangci-lint; G103/G204/G304/G702 SARIF-only) + govulncheck
```

### Frontend

```bash
make lint-frontend     # Biome + tsc
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
