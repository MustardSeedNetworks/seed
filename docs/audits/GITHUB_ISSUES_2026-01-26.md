# GitHub Issues - Seed (2026-01-26)

## Security / Error Handling
- Title: Panic on crypto/rand failure in JWT secret generation
  Labels: security, backend
  File: internal/auth/auth.go:222
  Body: `GenerateJWTSecret` panics if crypto/rand fails after retries. Consider returning an error and failing startup with explicit messaging instead of runtime panic. Repro: see panic path in `GenerateJWTSecret`.

- Title: Panic on request ID generation failure
  Labels: security, backend
  File: internal/logging/middleware.go:62
  Body: `generateRequestID` panics on crypto/rand failure. Consider returning 503 or fallback ID to avoid crashing request handling.

- Title: MustMarshal panics on marshal errors
  Labels: backend
  File: internal/api/json_helpers.go:76
  Body: `MustMarshal` panics on JSON errors. Ensure only static data uses this helper or replace with error returns.

## Incomplete Implementations
- Title: Vulnerability status updates are unimplemented
  Labels: backend, incomplete
  File: internal/services/shell/services.go:283
  Body: `UpdateStatus` returns `ErrNotImplemented` and has a TODO. Implement persistence or disable endpoint until supported.

- Title: Rogue device acknowledgements are unimplemented
  Labels: backend, incomplete
  File: internal/services/shell/services.go:460
  Body: `AcknowledgeDevice` returns `ErrNotImplemented` with TODO. Implement storage or remove action.

- Title: SNMP GetInterfaces not implemented
  Labels: backend, incomplete
  File: internal/services/services.go:414
  Body: `GetInterfaces` returns `ErrNotImplemented`. Implement `snmp.GetInterfaces` integration or gate API.

- Title: SNMP GetMACTable not implemented
  Labels: backend, incomplete
  File: internal/services/services.go:420
  Body: `GetMACTable` returns `ErrNotImplemented`. Implement `snmp.GetMACTable` integration or gate API.

- Title: Telemetry aggregation loop not implemented
  Labels: backend, incomplete
  File: internal/services/services.go:627
  Body: `TelemetryService.Start` has TODO and returns nil. Implement aggregation loop or remove service wiring.

- Title: Telemetry snapshot/history retrieval not implemented
  Labels: backend, incomplete
  File: internal/services/services.go:641
  Body: `GetSnapshot` and `GetHistory` return `ErrNotImplemented`. Implement DB retrieval or remove endpoints.

## Integration / UX
- Title: Logs query total count is inaccurate
  Labels: backend, integration
  File: internal/api/handlers_logs.go:226
  Body: `TotalCount` returns `len(dbLogs)` with TODO. Return actual total count for pagination accuracy.

## Lint / Hygiene
- Title: Extract repeated "linux" string constant
  Labels: lint, backend
  File: cmd/seed/cmd_uninstall.go:230
  Body: `goconst` flags repeated "linux" string.

- Title: Fix gofumpt formatting in handlers types
  Labels: lint, backend
  File: internal/api/handlers_types.go:37
  Body: `gofumpt` reports file not properly formatted.

- Title: Replace magic number for generated password length
  Labels: lint, backend
  File: cmd/seed/cmd_serve.go:216
  Body: `mnd` flags `GenerateSecurePassword(16)`; extract a constant.

- Title: Remove unused cobra handler params
  Labels: lint, backend
  File: cmd/seed/cmd_auth.go:24
  Body: `revive` flags unused params in cobra handlers; rename to `_` or remove.
