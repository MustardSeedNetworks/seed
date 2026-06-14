# ADR-0024: Identity decomposition — users / oauth / tokens use-cases over repository ports

**Status:** Accepted — 2026-06-10 · applies ADR-0020 to the identity surface (C4, the final `internal/api` strangle slice before the `ServiceContainer` deletion)

## Context

C4 is the last and heaviest slice of the ADR-0020 clean-hexagonal strangle. The
identity surface — `/api/v1/users`, the `/api/sso` OAuth flow, and
`/api/v1/tokens` — is the only place left where `internal/api` handlers reach raw
persistence (`s.services.Database.DB`, `s.services.Auth.APITokens`). Those reaches
are also the most security-sensitive in the codebase: they read and write the
`users` table and the `api_tokens` store that the whole authorization model rests
on.

Two properties make identity different from the earlier slices (network, alerts,
settings, profiles, wifi, discovery, health, update):

1. **The data access is entangled with the security edge, not just handlers.**
   `callerRole` / `callerIsAdmin` / `requireRole` / `requireAdmin` / `writeGated`
   are the fleet-wide authorization primitives (used by `route.go` at registration
   and by handlers everywhere); `apiTokenMiddleware` / `resolveAPIToken` are the
   PAT authentication seam (wired in `server_lifecycle.go`). These are **policy at
   the registry edge** per ADR-0002 and ADR-0020 §5 — they must not move into a
   use-case — yet `callerRole` and the PAT middleware each read a store directly.

2. **OAuth is mostly transport.** The SSO login/callback handlers are a cookie /
   CSRF-state / code-exchange / redirect dance. Their *only* domain data operation
   is `UpsertSSOUser`, which writes the **users** table — SSO is not a third
   independent store, it is a second writer of the user store. Session-token
   issuance in the callback belongs to the auth subsystem (`authManager()`), a
   separate workstream.

## Decision

Decompose identity into **three capability packages** under `internal/identity`,
each with consumer-defined ports, adapters in the composition root, and pure
transport in `internal/api` (ADR-0020 shape). Repository ports replace every raw
`Database.DB` / `APITokens` reach **in handler bodies** — that is the security win.

### Packages

- **`internal/identity/users`** — `users.Service` over a `Repository` port
  (`List` / `Create` / `Get` / `UpdatePassword` / `UpdateRole` / `Deactivate` /
  `Delete`). Drives the `/users` CRUD handlers. Thin CRUD stays thin: the port
  returns the existing `*database.User` rather than a new domain type — ADR-0020
  does not mandate a DTO rewrite, and inventing one here is churn without a
  security or clarity gain.

- **`internal/identity/tokens`** — `tokens.Service` over a `Store` port
  (`Insert` / `ListByOwner` / `Revoke`) plus a `LicenseGate` port
  (`AllowsMinting() bool`) for the Pro gate. Drives `/tokens` mint/list/revoke.
  The per-token scope cap is validated against the **owner's role resolved at the
  edge** (`callerRole`) and passed into the mint use-case as a value — the
  use-case never performs authorization, it only enforces `scope ≤ ownerRole` as
  an input invariant.

- **`internal/identity/oauth`** — `oauth.Service` over a `Repository` port whose
  single method `SyncUser(ctx, SSOUserInput) (*database.User, error)` wraps
  `UpsertSSOUser`. The handler keeps the OAuth transport dance (state cookies,
  code exchange, provider lookup, token issuance, redirects); it calls the
  use-case for the one data operation. The package is deliberately thin — SSO
  identity sync is a distinct capability (upsert-by-external-id, provider linkage)
  from interactive user CRUD, so it earns its own port without dragging transport
  into a use-case.

### The security edge stays at the edge, but sources data through the ports

`callerRole` / `callerIsAdmin` / `requireRole` / `requireAdmin` / `writeGated` and
`apiTokenMiddleware` / `resolveAPIToken` **remain `internal/api` constructs** — they
are policy, and moving them would both create an import cycle and violate ADR-0020
§5. Their *decision logic is unchanged* (role ranking, the "no user DB ⇒ admin"
dev tolerance, PAT scope clamping). What changes is only the **data fetch**:
`callerRole`'s user lookup routes through the `users` repository seam instead of
raw `s.services.Database.DB`, so that after C4 **no raw user-store handle is held
anywhere in `internal/api` except the authentication middleware wiring**. The PAT
authN middleware keeps its direct `*database.APITokenRepository` (it is the
authentication seam, constructed once in `server_lifecycle.go`); `FindActiveByHash`
/ `TouchLastUsed` are authentication reads, not handler data access, and stay
there. This is the "narrow, auditable seam for auth/session data access" the
strangle plan calls for: one place per store.

### Out of scope (deliberately)

- **`handleLicenseStatus`** (`GET /api/v1/license`) is a license-state projection,
  not identity-store access. It keeps reading `s.services.Auth.License` directly; a
  dedicated license slice can absorb it later. It is not a `Database.DB` reach, so
  it is not a security target of this slice.
- **Session-token issuance** in the OAuth callback stays on `authManager()` — the
  auth/oauth workstream owns it.

### Composition + wiring (ADR-0020 mechanics)

Adapters live in `internal/app/identity.go` behind `app.NewIdentityUsers` /
`NewIdentityOAuth` / `NewIdentityTokens`, each over a **lazy accessor** so a store
set or replaced after `NewServer` (the api test harness does this) is honored and a
nil store degrades to the use-case's `ErrUnavailable` (preserving the golden-pinned
503 paths). `internal/api` wires via those constructors in `s.initUseCases()` and
keeps no `*_usecases.go` file. The handler files are renamed to drop the monolith
prefix (`handlers_users.go` → `users.go`, `handlers_oauth.go` → `oauth.go`,
`handlers_api_tokens.go` → `tokens.go`) per the filename policy (`scripts/check-filename-policy.sh`).

## Consequences

- **Security:** every user-store and token-store read/write in handler bodies goes
  through a narrow, named, unit-tested port; the only raw store handles left in
  `internal/api` are the two authentication seams (authZ role lookup via the users
  port, PAT authN middleware), each in exactly one place.
- **Route policy is byte-identical.** No CSRF / rate-limit / role gate moves;
  `scripts/check-route-policy.sh` and `check-output-escaping.sh` pass unchanged.
- **`ServiceContainer` is now unreached from handlers.** After C4, `git grep
  's\.services\.' internal/api/handlers_*.go` (and the renamed files) is empty
  except the authentication-middleware wiring in `server_lifecycle.go` and lifecycle
  files — unblocking D1 (delete the container).
- **Cost:** three small port packages with table-driven fake-port tests; the
  OAuth package is thin by design (one method) — accepted as honest capability
  segregation, not over-packaging, because it backs an independent route tree with
  upsert-by-external-id semantics distinct from CRUD.
- **Relationship to WS-B (repository-port discipline, 2026-06-12).** WS-B relocates
  each domain package's row types into the package so it stops importing
  `internal/database`. **This ADR takes precedence for identity:** the ports here
  intentionally return `*database.User` / `database.APITokenRecord` /
  `database.SSOUserInput` (and pass the `ErrUser*`/`ErrLastAdmin` sentinels through),
  so `internal/identity/{users,oauth,tokens}` keep their `internal/database` import
  **by design** — the "thin CRUD stays thin" decision above stands. Identity is
  therefore the **one documented exception** to WS-B: it gets no
  `identity-no-persistence` depguard rule (the exemption is recorded as a comment in
  `.golangci.yml`, next to the other WS-B rules). Should a future change give the
  user/token store domain-meaningful behavior beyond thin CRUD, revisit this — the
  exception is scoped to the current thin-CRUD shape, not a permanent carve-out.

## Implementation phasing

C4 lands as one cohesive PR (the three packages share the edge helpers and the
`initUseCases` wiring; splitting would leave `callerRole` half-migrated between
PRs). Verified per the ADR-0020 per-PR gate: `go build ./...`; `golangci-lint run`
0 in changed packages; `go test ./internal/api/ -count=1` run ALONE; the three new
`internal/identity/*` packages green; route-policy + output-escaping +
filename-policy + schema-drift pass; goldens reviewed byte-identical. D1 follows.
