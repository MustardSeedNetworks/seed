# 0031 — One SQLite write connection, a pool for reads

Status: Accepted
Date: 2026-09-09

## Context

SQLite admits exactly one writer at a time. `database/sql` does not know that:
it hands every caller a connection from the pool, so ten goroutines that write
concurrently become ten connections competing for one lock. The loser waits for
the busy timeout and is then rejected with `SQLITE_BUSY`, and the caller has to
decide what to do with a row it has already built.

That decision was never made, so the row was lost. On CT313 during the S4-4
hospital-pack run (#2453), ten of 74 SNMP polling targets ended a cycle in
`error`:

```text
snmp poller: collector failed target_id=… collector=arp error="arp: publish: snmp sink: insert arp: insert snmp_observation: executing query: database is locked (5) (SQLITE_BUSY)"
snmp poller: update last_poll failed target_id=… error="update polling_target last poll: … (SQLITE_BUSY)"
```

The topology for those devices was partial for that cycle, and the operator saw
a failed poll rather than a slow one. 74 targets is a small site; the niac packs
go to 159.

Raising the busy timeout only moves the cliff, and retrying inside the
repository would make every write path carry a retry loop for a condition the
process can prevent outright. WAL is already on, which is why readers were never
part of the problem: in WAL a reader never blocks the writer.

## Decision

`internal/database` holds two handles over the same file:

- **`readConn`** — the pooled handle (`MaxOpenConns` from config, 10 by default).
  `Query` and `QueryRow` use it.
- **`writeConn`** — one connection, always. `Exec`, `BeginTx`, `WithTx`,
  `QueryRowWrite`, goose migrations, `VACUUM`/`ANALYZE` and every direct
  `ExecContext` in the package use it.

With a single write connection the queue moves from SQLite's busy handler into
`database/sql`, which waits on the caller's context. A competing writer is
delayed by exactly as long as the writer ahead of it holds the connection, and
is cancelled only by its own deadline — never rejected.

`UPDATE … RETURNING` is a write even though it returns a row, so it gets its own
spelling (`DB.QueryRowWrite`) rather than riding `QueryRow` onto the read pool.

`scripts/check-single-writer.sh` is the CI gate: it rejects a write executed on
`readConn`, a DML statement issued through the read path, a write handle opened
with more than one connection, and a raw write handle taken outside the
sanctioned call sites.

## Consequences

- A burst of collectors cannot lose an observation to lock contention. The
  regression is `internal/database/single_writer_test.go`, which holds a write
  transaction past the busy timeout and asserts the competing insert commits.
- Writes are strictly serialised process-wide. A slow write delays other writes;
  it does not fail them. Long write transactions are now a latency problem
  rather than a correctness one, which is the trade we want.
- Cross-process contention is unchanged and still handled by the busy timeout —
  a CLI subcommand run against a live daemon's database can still see
  `SQLITE_BUSY`. Seed is one process per database in every supported
  deployment.
- `DB.Conn()` is gone. `DB.WriteConn()` replaces it and says what it hands out;
  its only consumer is `mibdb`, which fills its OID table at startup.
- Two handles mean two connection pools over one file. Pragmas are
  connection-scoped, so `openHandle` applies them to each.
