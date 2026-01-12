# Lint Issues Summary

**Total Issues: 229**

Generated: 2025-01-11

## Issue Categories

| Category | Count | Description |
|----------|-------|-------------|
| mnd (magic numbers) | 50 | Hardcoded numeric values that should be constants |
| revive | 18 | Code style and naming issues |
| musttag | 13 | Missing struct tags for serialization |
| noctx | 12 | HTTP requests missing context |
| golines | 11 | Lines exceeding max length |
| gocognit | 10 | Functions with high cognitive complexity |
| govet | 10 | Go vet warnings |
| modernize | 9 | Opportunities to use newer Go features |
| tparallel | 9 | Test parallelization issues |
| funcorder | 7 | Constructor/method ordering |
| goimports | 7 | Import formatting |
| godoclint | 7 | Documentation formatting |
| usetesting | 7 | Using os.MkdirTemp instead of t.TempDir |
| goconst | 6 | Repeated strings that should be constants |
| unused | 6 | Unused code |
| nestif | 5 | Deeply nested if statements |
| funlen | 4 | Functions exceeding max statements |
| gosec | 4 | Security issues |
| rowserrcheck | 4 | Unchecked rows.Err() |
| sqlclosecheck | 4 | SQL Close() not using defer |
| unconvert | 3 | Unnecessary type conversions |
| gochecknoglobals | 3 | Global variables |
| nilnil | 3 | Functions returning nil, nil |
| testpackage | 3 | Tests not in _test package |
| dupl | 2 | Duplicate code blocks |
| gocritic | 2 | Code improvement suggestions |
| godot | 2 | Comments not ending with period |
| intrange | 2 | For loops that could use range |
| staticcheck | 2 | Static analysis issues |
| errcheck | 1 | Unchecked error |
| exhaustive | 1 | Missing switch cases |
| perfsprint | 1 | Inefficient string formatting |
| unparam | 1 | Unused function parameters |

## Critical Issues (Should Fix)

### Security (gosec) - 4 issues
- Potential security vulnerabilities flagged by gosec

### Error Handling (errcheck, rowserrcheck) - 5 issues
- `internal/discovery/problem_detector.go:191` - Unchecked `fmt.Sscanf` error
- 4 unchecked `rows.Err()` in `repository_discovery.go`

### SQL Safety (sqlclosecheck) - 4 issues
- `repository_discovery.go` - `rows.Close()` not using defer

### Deprecated APIs (staticcheck) - 2 issues
- `internal/discovery/problem_detector.go:342,343` - Using deprecated `strings.Title`

### Unused Code (unused) - 6 issues
- `handlers_sse.go:60` - unused const `sseWriteTimeout`
- `handlers_sse.go:411,416` - unused type and method `ssePipelineBroadcastAdapter`
- `handlers_websocket.go:793,798` - unused type and method `logBroadcastAdapter`
- `roots/services_test.go:161` - unused field `wantScore`

## Code Duplication (dupl) - 2 issues
- `handlers_health_checks.go:1337-1370` duplicates `handlers_health_checks.go:1517-1542`
  - Both create identical `httptrace.ClientTrace` configurations

## High Complexity Functions (gocognit > 20)

| File | Function | Complexity |
|------|----------|------------|
| `discovery/registry.go:157` | `mergeDevice` | 62 |
| `handlers_health_checks.go:1147` | `runSingleHTTPTest` | 47 |
| `discovery/engine.go:435` | `runDiscoveryPhase` | 42 |
| `handlers_health_checks.go:1393` | `runHTTPTestEnhanced` | 40 |
| `handlers_network.go:300` | `handleInterface` | 38 |
| `discovery/bluetooth_darwin.go:93` | `parseSystemProfilerOutput` | 31 |
| `discovery/bluetooth_darwin.go:165` | `parseSystemProfilerText` | 28 |
| `discovery/events.go:171` | `Matches` | 25 |
| `database/repository_health_checks.go:276` | `GetLatencyStats` | 22 |
| `discovery/problem_detector.go:114` | `ScanWiFi` | 22 |

## Long Functions (funlen > 50/100 statements)

- `handlers_health_checks.go:308` - `getHealthChecksSettings` (188 lines)
- `discovery/engine.go:313` - `Scan` (59 statements)
- `discovery/bluetooth.go:394` - `BLEAppearanceToClass` (56 statements)
- `handlers_health_api.go:71` - `handleHealthCheckHistory` (52 statements)

## Recommendations

### Priority 1 - Fix Now
1. Fix deprecated `strings.Title` usage
2. Add proper error checking for `rows.Err()`
3. Use defer for `rows.Close()`
4. Remove unused code (adapters in handlers_sse.go, handlers_websocket.go)

### Priority 2 - Technical Debt
1. Extract duplicate `httptrace.ClientTrace` into shared function
2. Break down high-complexity functions
3. Add constants for magic numbers (mnd issues)

### Priority 3 - Code Quality
1. Fix import formatting (goimports)
2. Add missing struct tags (musttag)
3. Fix test parallelization (tparallel)
