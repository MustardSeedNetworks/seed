# Testing Requirements

## Before Committing (MANDATORY)

**ALL CI checks MUST pass before committing. No exceptions.**

```bash
make all   # Runs: lint → test → security → build → docker-test
```

## What Must Pass

| Check                  | Command                             | Requirement                  |
| ---------------------- | ----------------------------------- | ---------------------------- |
| Go linting             | `make lint-backend`                 | Zero errors                  |
| Frontend linting       | `make lint-frontend`                | Zero errors                  |
| Go unit tests          | `make test-backend`                 | All pass                     |
| Frontend unit tests    | `make test-frontend`                | All pass                     |
| Security scan (gosec)  | `make security-backend`             | Zero findings                |
| Secret scan (gitleaks) | `make security-secrets`             | No secrets detected          |
| npm audit              | `make security-frontend`            | No high/critical vulns       |
| Build                  | `make build`                        | Succeeds                     |
| Playwright E2E tests   | `make test-e2e`                     | All pass                     |
| Storybook build        | `cd web && npm run build-storybook` | Succeeds                     |
| Docker smoke test      | `make docker-test`                  | Passes (if Docker available) |

## Do NOT Commit If

- Any linting errors exist
- Any tests fail (unit OR E2E)
- Any security warnings from gosec
- Any secrets detected by gitleaks
- Any high/critical npm vulnerabilities
- Build fails

**CI must be green. If it would fail in GitHub Actions, don't commit.**

## Pre-commit Hooks

Pre-commit hooks automatically run checks before each commit:

```bash
# Install pre-commit hooks (one-time setup)
pip install pre-commit
make pre-commit-install

# Run manually
make pre-commit
```

Pre-commit runs: gitleaks, go-fmt, go-vet, golangci-lint, ESLint, TypeScript check, tests.

## E2E Browser Tests (Playwright)

Playwright end-to-end tests verify critical user flows in real browsers:

```bash
make test-e2e           # Run E2E tests headless
make test-e2e-ui        # Run with UI for debugging
make test-e2e-install   # Install Playwright browsers (first time)
```

### E2E Test Coverage

- Authentication flow (login/logout)
- Dashboard cards rendering
- Settings drawer functionality
- Network discovery
- WiFi survey
- Speed/performance testing
- Vulnerability scanning
- WebSocket connectivity
- Setup wizard

## Storybook Component Testing

Storybook provides visual component documentation and testing:

```bash
cd web && npm run storybook        # Run Storybook locally
cd web && npm run build-storybook  # Build static Storybook
```

## Test Coverage Requirements

**Minimum 80% code coverage is REQUIRED for all new code.**

When building new features, ALL testing artifacts must be updated together:

| New Code       | Required Tests                          |
| -------------- | --------------------------------------- |
| UI component   | Storybook story + unit test             |
| Card component | Storybook story + Playwright E2E test   |
| User flow      | Playwright E2E test                     |
| API endpoint   | Go unit tests + E2E test if user-facing |

### Coverage Commands

**Go Backend:**

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

**React Frontend:**

```bash
cd web && npm run test:coverage
```

## Coverage Gates

| Metric            | Requirement                     |
| ----------------- | ------------------------------- |
| Backend overall   | 40% minimum (increasing to 90%) |
| New backend code  | 80% minimum                     |
| Frontend overall  | Maintained or improved          |
| New frontend code | 80% minimum                     |
| New UI components | Must have Storybook stories     |
| New user flows    | Must have E2E tests             |

## Test File Conventions

| Type                | Pattern                   | Location                    |
| ------------------- | ------------------------- | --------------------------- |
| Go unit tests       | `*_test.go`               | Same package as code        |
| Frontend unit tests | `*.test.ts`, `*.test.tsx` | Same directory as component |
| E2E tests           | `*.spec.ts`               | `web/e2e/`                  |
| Storybook stories   | `*.stories.tsx`           | Same directory as component |

## Security Tools

| Tool        | Purpose                   | Command                   |
| ----------- | ------------------------- | ------------------------- |
| gosec       | Go security scanner       | `make security-backend`   |
| govulncheck | Go vulnerability check    | `make security-backend`   |
| npm audit   | Node.js dependency audit  | `make security-frontend`  |
| gitleaks    | Secret detection          | `make security-secrets`   |
| trivy       | Container/filesystem scan | CI only (or `trivy fs .`) |

Run all security checks:

```bash
make security   # Runs all security scans
```

## Security Principles

- **gosec**: All security warnings must be resolved
- **gitleaks**: No secrets in code or git history
- **npm audit**: No high/critical vulnerabilities in dependencies
- **Never commit secrets**: No API keys, passwords, or tokens in code
- **Validate inputs**: All user/external input must be validated
- **Secure defaults**: HTTPS enabled, strong passwords required
