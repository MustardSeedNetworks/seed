# Seed Auth Audit — 2026-05-19

Task: #77 — audit + modernize auth across all 3 repos.

Read-only audit of `internal/auth/` and `internal/api/` covering the
full login flow (hashing, session storage, CSRF, rate limit, lockout,
error messages, password policy, setup token, audit logging).

No severity ratings are assigned here. Humans should triage and
prioritize from the findings below.

---

## Summary

| Category                  | Result |
|---------------------------|--------|
| Password hashing          | WARN — bcrypt cost 12 (Argon2id is target per RFC 9106) |
| Session token storage     | PASS — httpOnly cookies, refresh rotation, blacklist on logout |
| CSRF protection           | PASS — SameSite=Strict, 32-byte tokens, constant-time compare |
| Rate limiting             | PASS — per-IP, sliding window, adaptive cleanup |
| Account lockout           | PASS — via UserStore (5 attempts / 15 min) |
| Error messages            | PASS — login always returns `errors.auth.invalidCredentials` |
| Password policy           | WARN — length+class only; no breach-corpus check |
| Setup wizard token        | PASS — 32 bytes, single-use, 15 min expiry, constant-time |
| Audit logging             | WARN — login events logged; password change events partial |

Totals: 6 PASS, 3 WARN, 0 FAIL.

---

## Checklist

### Password hashing algorithm
- **Result**: WARN
- **Where**: `internal/auth/auth.go:476-483`
- **Detail**: `HashPassword` uses bcrypt with cost factor 12. bcrypt is
  acceptable but RFC 9106 / NIST 800-63B prefer Argon2id for new code.
- **Remediation note**: Add Argon2id support behind a feature flag and
  re-hash on next successful login. Migration, not a fix.

### Session token storage
- **Result**: PASS
- **Where**:
  - JWT signing: `internal/auth/auth.go:335-375` (HS256, 32-byte secret)
  - Cookie store: `internal/auth/cookie.go:55-84`
    (`HttpOnly: true`, `SameSite: SameSiteStrictMode`, `Secure` toggled by
    HTTPS config)
  - Refresh rotation: `internal/auth/auth.go:447-471`
    (15-min access, 7-day refresh, 24-hr max session lifetime)
  - Revocation: `internal/auth/auth.go:856-879` blacklists fingerprint
    on logout; `tokenVersion` field invalidates all tokens on password
    change.

### CSRF protection
- **Result**: PASS
- **Where**: `internal/auth/csrf.go:80-167`
- **Detail**: 32-byte token, `subtle.ConstantTimeCompare`, 24-hr expiry,
  TOCTOU race fixed (`csrf.go:117-129`), `X-Csrf-Token` header required
  on all state-changing requests, session-scoped, expired-token cleanup
  goroutine.

### Rate limiting
- **Result**: PASS
- **Where**: `internal/api/ratelimit.go:79-114`
- **Detail**: 5 attempts per 15-min window per IP for login; 5/min for
  expensive endpoints; memory-protection cap of 10000 visitors with
  adaptive TTL at 80%/90% capacity; uses `RemoteAddr` not
  `X-Forwarded-For` to prevent spoofing
  (`ratelimit.go:298-313`).

### Account lockout
- **Result**: PASS
- **Where**: `internal/auth/auth.go:255-291` (UserStore path checks
  `IsLocked` then records failure/success).
- **Detail**: When backed by a database UserStore the user record gets
  per-user attempt counters + lock timestamp. The in-memory fallback
  relies on the IP rate limiter only.

### Error messages
- **Result**: PASS
- **Where**: `internal/auth/auth.go:267-281`,
  `internal/api/handlers_auth.go:65-92`
- **Detail**: `Authenticate` returns the same `ErrInvalidCredentials`
  whether the user is not found, the account is locked, or the password
  is wrong. The handler emits `errors.auth.invalidCredentials` only.
  No user-enumeration leak.

### Password policy
- **Result**: WARN
- **Where**: `internal/auth/auth.go:603-627`
- **Detail**: Min length 12 + uppercase + lowercase + digit + symbol.
  No breach-corpus (HIBP k-anonymity) check; NIST 800-63B currently
  recommends a corpus check over rigid complexity classes.
- **Remediation note**: Add HIBP k-anonymity check on first-time
  password set (filed as followup, not in scope here).

### Setup wizard token
- **Result**: PASS
- **Where**: `internal/api/setup_token.go:36-103`
- **Detail**: 32-byte base64url token, single-use (cleared after
  validation), 15-min TTL, `subtle.ConstantTimeCompare`, also blocked
  after initial setup completes via
  `handlers_auth.go:454-463`.

### Audit logging
- **Result**: WARN
- **Where**: `internal/api/handlers_auth.go:48-208` (login + logout +
  refresh events tagged `event=auth.*` with IP + outcome).
- **Detail**: Login success/failure, logout, refresh, account lock,
  setup events are all logged. UA is not consistently captured on
  login events, and password-change-by-admin
  (`UpdatePasswordHash` at `auth.go:784-804`) emits a log but not as a
  structured security event the same way logins are.
- **Remediation note**: Standardize on a `SecurityEvent` struct
  carrying IP+UA+outcome+request_id like stem does
  (`stem:internal/logging/audit.go`). Filed as followup.

---

## Small fixes shipped in this PR

None for seed. The existing flow already returns generic
`invalidCredentials`, uses `SameSite=Strict`, applies rate limiting,
and revokes tokens on logout. Tightening anything else
crosses the "no new deps, no schema changes, no behavior shifts" line.

This PR is audit-only for seed.

---

## Followup tickets (deferred work)

1. **TOTP second factor** — store per-user TOTP secret in
   `internal/database/`, add `/api/v1/auth/totp/{enroll,verify}` and a
   `RequireMFA` middleware. New dep:
   `github.com/pquerna/otp`. Proposed task: `feat(auth): TOTP 2FA enrollment + login challenge`.
2. **WebAuthn / Passkeys** — replace password-only login for the admin
   user. New dep: `github.com/go-webauthn/webauthn`.
   Proposed task: `feat(auth): WebAuthn passkey enrollment + login`.
3. **Argon2id migration** — add Argon2id alongside bcrypt, re-hash on
   successful login. Stdlib + `golang.org/x/crypto/argon2`. Proposed
   task: `refactor(auth): migrate password hashing to Argon2id (RFC 9106)`.
4. **Password strength meter + breach check** — surface zxcvbn-style
   score in the setup wizard, optional HIBP k-anonymity lookup.
   Proposed task: `feat(auth): zxcvbn password meter + HIBP breach check`.
5. **Magic-link recovery** — replace the file-based recovery token
   with an email magic link. Requires SMTP config + delivery
   integration tests. Proposed task: `feat(auth): magic-link account recovery via email`.
6. **Structured `SecurityEvent` for all auth events** — port stem's
   `SecurityEvent`+`AuditXxx` helpers; capture
   IP, UA, outcome, request_id consistently on every auth touchpoint.
   Proposed task: `refactor(auth): unify structured audit events across login/logout/refresh/password-change`.
