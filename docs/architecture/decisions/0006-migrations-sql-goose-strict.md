# ADR-0006: Schema as embedded `.sql` files, run by goose, STRICT tables

**Status:** Accepted — 2026-05-31

## Context

The schema lives as raw SQL inside Go string literals (`internal/database/migrations.go`,
2,190 lines, ~40 tables). A homegrown up-only runner applies an ordered
`[]migrationDef{Description, Up}` slice (version = index+1, migrations immutable). This
loses all SQL tooling (highlighting, linting, formatting), is hard to diff/review (a
change is a hunk inside a 2,190-line Go blob), and accumulates fragile hand-written
table-rebuild migrations (`PRAGMA foreign_keys=OFF; CREATE …_new …`) to work around
SQLite's `ALTER` limitations.

The connection setup (`database.go`) is, by contrast, already sound: `foreign_keys=ON`,
`journal_mode=WAL`, `synchronous=NORMAL`, `busy_timeout`, `cache_size`,
`temp_store=MEMORY`, pool limits, WAL checkpoint on close, corruption auto-rebuild.

Greenfield (no production data) + the schema-collapse already decided in the blueprint
make this the right moment to change *how* the schema is expressed.

## Decision

- **Schema/migrations move to `.sql` files**, one per migration
  (`internal/adapters/store/migrations/NNNN_description.sql`), **embedded via `//go:embed`**
  so the binary stays self-contained (same mechanism as the UI embed).
- **Runner: `github.com/pressly/goose`** (pinned), the community-standard tool — `embed.FS`
  support, up + down migrations, a versioning table, `status` command. Replaces the
  homegrown index+1 scheme.
- **Single baseline:** collapse the 2,190-line history into one clean `0001_init.sql`
  (no data to preserve); the table-rebuild dances disappear into it.
- **STRICT tables** (SQLite 3.37+) throughout the baseline — enforce declared column types.
- **Explicit constraints** in the baseline (FKs, `CHECK`, `NOT NULL`, `UNIQUE`); one
  uniform timestamp representation.
- **Connection setup ported verbatim** from `database.go` into `adapters/store` (it's correct).
- Forward-only in production; down-migrations for dev/rollback; no business logic in
  migrations; a "migrate-from-empty → assert schema" test gates drift.

## Consequences

- SQL gets first-class tooling and reviewability; migrations are real, diffable artifacts.
- One pinned dependency (`goose`) — justified against the "minimise dependencies" rule by
  the up/down + status + community-standard payoff over the homegrown runner.
- The baseline is a one-time clean cut; the data-model review (blueprint §7) happens here.
- STRICT tables may surface latent type-sloppiness in repository code — caught at the
  Phase-5 cutover, by design.
- Reference data (OUI, MIB) is split to embedded read-only stores, not the mutable DB.
