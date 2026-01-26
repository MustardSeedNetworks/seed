# Audit Suite - Seed (2026-01-26)

## Commands Run
- `golangci-lint run ./...`
- `govulncheck ./...`
- `npm audit --production` (root and `ui/`)
- `rg -n "TODO|FIXME|HACK|XXX"`
- `rg -n "panic\(|log\.Fatal"`
- `rg -n "(?i)(password|passwd|secret|api[_-]?key|token|private key|AKIA[0-9A-Z]{16})"`

## 01 - Initial Audit (Security/Quality)
[SEVERITY: MEDIUM]
[CATEGORY: Incomplete]
[FILE: internal/services/shell/services.go:283]
[ISSUE: Vulnerability status updates are not implemented]
[EVIDENCE: `UpdateStatus` returns `ErrNotImplemented` with TODO]
[RECOMMENDATION: Implement persistence for vulnerability status or remove the endpoint if not supported]

[SEVERITY: MEDIUM]
[CATEGORY: Incomplete]
[FILE: internal/services/shell/services.go:460]
[ISSUE: Rogue device acknowledgements are not persisted]
[EVIDENCE: `AcknowledgeDevice` returns `ErrNotImplemented` with TODO]
[RECOMMENDATION: Implement acknowledgement storage or remove the action until supported]

[SEVERITY: MEDIUM]
[CATEGORY: Incomplete]
[FILE: internal/services/services.go:414]
[ISSUE: SNMP interfaces retrieval not implemented]
[EVIDENCE: `GetInterfaces` returns `ErrNotImplemented` with TODO]
[RECOMMENDATION: Implement `snmp.GetInterfaces` integration or gate the API]

[SEVERITY: MEDIUM]
[CATEGORY: Incomplete]
[FILE: internal/services/services.go:420]
[ISSUE: SNMP MAC table retrieval not implemented]
[EVIDENCE: `GetMACTable` returns `ErrNotImplemented` with TODO]
[RECOMMENDATION: Implement `snmp.GetMACTable` or disable endpoint]

[SEVERITY: MEDIUM]
[CATEGORY: Incomplete]
[FILE: internal/services/services.go:627]
[ISSUE: Telemetry collection loop is unimplemented]
[EVIDENCE: `Start` contains TODO and returns nil]
[RECOMMENDATION: Implement aggregation loop or remove the service from runtime wiring]

[SEVERITY: MEDIUM]
[CATEGORY: Incomplete]
[FILE: internal/services/services.go:641]
[ISSUE: Telemetry snapshot/history retrievals are unimplemented]
[EVIDENCE: `GetSnapshot` and `GetHistory` return `ErrNotImplemented`]
[RECOMMENDATION: Implement DB-backed retrieval or remove API paths]

[SEVERITY: LOW]
[CATEGORY: Integration]
[FILE: internal/api/handlers_logs.go:226]
[ISSUE: Log query response total count is inaccurate]
[EVIDENCE: `TotalCount: len(dbLogs) // TODO: get actual count from DB`]
[RECOMMENDATION: Return total count from DB for correct pagination]

## 02 - Lint Remediation
[SEVERITY: LOW]
[CATEGORY: Backend]
[FILE: cmd/seed/cmd_uninstall.go:230]
[ISSUE: goconst flagged repeated "linux" string]
[EVIDENCE: `if runtime.GOOS != "linux" {`]
[RECOMMENDATION: Extract constant or ignore if not worth refactor]

[SEVERITY: LOW]
[CATEGORY: Backend]
[FILE: internal/api/handlers_types.go:37]
[ISSUE: gofumpt formatting error]
[EVIDENCE: Lint output reports file not properly formatted]
[RECOMMENDATION: Run `gofumpt` on file]

[SEVERITY: LOW]
[CATEGORY: Backend]
[FILE: cmd/seed/cmd_serve.go:216]
[ISSUE: Magic number flagged by mnd]
[EVIDENCE: `GenerateSecurePassword(16)`]
[RECOMMENDATION: Extract to constant]

[SEVERITY: LOW]
[CATEGORY: Backend]
[FILE: cmd/seed/cmd_auth.go:24]
[ISSUE: Unused parameter flagged by revive]
[EVIDENCE: Unused `args` in cobra handler]
[RECOMMENDATION: Rename to `_` or remove]

## 03 - Error Handling
[SEVERITY: MEDIUM]
[CATEGORY: Backend]
[FILE: internal/auth/auth.go:222]
[ISSUE: Panic on crypto/rand failure]
[EVIDENCE: `panic("crypto/rand failed after retries...")`]
[RECOMMENDATION: Prefer returning error and failing fast at startup with a clear message rather than runtime panic]

[SEVERITY: MEDIUM]
[CATEGORY: Backend]
[FILE: internal/logging/middleware.go:62]
[ISSUE: Panic on request ID generation failure]
[EVIDENCE: `panic("crypto/rand failed: ...")`]
[RECOMMENDATION: Fallback to deterministic ID or return 503 with explicit error path]

[SEVERITY: LOW]
[CATEGORY: Backend]
[FILE: internal/api/json_helpers.go:76]
[ISSUE: MustMarshal panics on marshal error]
[EVIDENCE: `panic(fmt.Sprintf("MustMarshal: failed..."))`]
[RECOMMENDATION: Ensure only static data uses this helper or replace with error returns]

## 04 - Input Validation
No concrete defects found via automated scan. Manual validation of all request payloads still required (form schemas, size limits, normalization).

## 05 - Auth Hardening
No concrete defects found via automated scan. Manual review needed for password reset, token revocation, CSRF flows.

## 06 - Database Security
No concrete defects found via automated scan. Manual review needed for query construction and transaction boundaries.

## 07 - API Contracts
Not fully assessed. Requires route-by-route tracing of API handlers against UI calls.

## 08 - Dependency Audit
- `govulncheck ./...`: No vulnerabilities found.
- `npm audit --production`: No vulnerabilities found (root and `ui/`).

## 09 - Secrets Audit
No hardcoded secrets detected by pattern scan. Verified configs use empty/default placeholders (e.g., `configs/seed.yaml` uses empty `jwt_secret` with auto-generation).

## 10 - Frontend Robustness
Not fully assessed. Requires UI route inspection and runtime testing.

## 11 - Test Coverage
Not fully assessed. Requires coverage data and critical-path mapping.

## 12 - Concurrency Audit
Not fully assessed. Requires race detector and review of goroutine lifecycles.

## 13 - Logging & Observability
Not fully assessed. Requires log coverage, metrics, and alerting review.

## 14 - Performance Audit
Not fully assessed. Requires profiling and resource leak review.

## 15 - Documentation & Maintainability
Not fully assessed.

## 16 - Configuration Management
Not fully assessed. Requires environment variable and config drift review.

## 17 - API Design
Not fully assessed.

## 18 - CI/CD Audit
Not fully assessed.

## 19 - Final Sweep
Pending after fixes.

## 20 - Rate Limiting
Not fully assessed.

## 21 - Internationalization
Not fully assessed.

## 22 - Accessibility
Not fully assessed.

## 23 - Responsive Design
Not fully assessed.

## 24 - SSE Security
Not fully assessed.

## 25 - File Upload Security
Not fully assessed.

## 26 - GraphQL Security
Not applicable unless GraphQL is introduced.

## 27 - Architecture Audit
Not fully assessed. See existing architecture docs for baseline.

## 28 - Dead Code Audit
Partial: multiple TODOs indicate stubs; full dead-code sweep pending.

## 29 - Code Duplication Audit
Not fully assessed.
