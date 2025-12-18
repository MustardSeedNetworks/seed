# CI/CD Pipeline

The CI pipeline runs on every push and PR. **All checks must pass.**

## GitHub Actions Workflows

### ci.yml - Main CI Pipeline

| Job         | Description        | Checks                                         |
| ----------- | ------------------ | ---------------------------------------------- |
| `backend`   | Go checks          | lint, vet, staticcheck, fmt, tests, coverage   |
| `frontend`  | React/TS checks    | TypeScript, ESLint, Prettier, build            |
| `security`  | Security scans     | govulncheck, gosec, npm audit, gitleaks, trivy |
| `quality`   | Code quality       | AI refs, sensitive files, unsafe logging       |
| `docs`      | Documentation      | Markdown lint, link check                      |
| `build`     | Build verification | Multi-arch binaries                            |
| `e2e`       | Browser tests      | Playwright E2E tests in Chrome                 |
| `storybook` | Component docs     | Storybook build                                |

### Other Workflows

| Workflow             | Purpose                         |
| -------------------- | ------------------------------- |
| `codeql.yml`         | GitHub CodeQL security analysis |
| `dead-code.yml`      | Weekly dead code detection      |
| `docker-publish.yml` | Publish Docker images           |
| `labeler.yml`        | Auto-label PRs                  |
| `license-check.yml`  | Verify dependency licenses      |
| `release-please.yml` | Automated releases              |
| `release.yml`        | Release builds                  |
| `title-lint.yml`     | Lint PR titles                  |
| `todo-tracker.yml`   | Weekly TODO tracking            |

## CI Must Pass Before Merge

PRs cannot be merged if any CI job fails. Fix all issues locally first:

```bash
make all      # Run full verification locally
make test-e2e # Run E2E browser tests
```

## Running CI Checks Locally

### Backend Checks

```bash
make lint-backend      # golangci-lint
make test-backend      # Go tests
make security-backend  # gosec + govulncheck
```

### Frontend Checks

```bash
make lint-frontend     # ESLint + TypeScript
make test-frontend     # Vitest
make build-frontend    # Vite build
```

### Security Checks

```bash
make security          # All security scans
make security-secrets  # gitleaks
```

### E2E Tests

```bash
make test-e2e          # Playwright headless
make test-e2e-ui       # Playwright with UI
```

## Debugging CI Failures

1. **Read the error** - CI logs show exactly what failed
2. **Run locally** - `make all` should catch the same errors
3. **Check specific job** - Run the specific failing command
4. **Don't push to fix** - Fix locally first, then push

## CI Environment

- **Go**: Version specified in `go.mod`
- **Node**: Version specified in `.nvmrc` or `package.json`
- **OS**: Ubuntu latest (CI), macOS (local dev)

## Secrets and Environment

CI uses GitHub secrets for:

- Docker registry credentials
- Release signing keys

Local development uses:

- `.env` files (not committed)
- Local config files
