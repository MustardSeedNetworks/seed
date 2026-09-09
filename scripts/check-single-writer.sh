#!/usr/bin/env bash
# check-single-writer.sh — SQLite single-writer gate (#2453).
#
# SQLite admits one writer at a time. When several pooled connections write
# concurrently the loser is rejected with SQLITE_BUSY once the busy timeout is
# spent and the caller loses the row — that is how ten of 74 SNMP polling
# targets lost observations on CT313. internal/database therefore keeps two
# handles: readConn (pooled) and writeConn (exactly one connection), so
# competing writers queue in Go instead of racing at the SQLite level.
#
# The invariant is one line of code away from being lost again: a new
# repository method that writes through readConn reintroduces the contention
# and no test outside a timing race would notice. This gate rejects that.
#
# Run locally: scripts/check-single-writer.sh

set -uo pipefail

FAIL=0

# Writes through the read pool.
WRITES_ON_READ=$(grep -rEn 'readConn\.(Exec|ExecContext|BeginTx)\(' internal/database 2>/dev/null || true)
if [ -n "$WRITES_ON_READ" ]; then
  echo "============================================================"
  echo "[single-writer] write executed on the read pool:"
  echo "$WRITES_ON_READ"
  echo "Writes go through writeConn (or db.Exec / db.WithTx, which route there)."
  echo ""
  FAIL=1
fi

# A write dressed as a read: UPDATE ... RETURNING through QueryRow lands on the
# read pool with no Exec in sight. DB.QueryRowWrite is the write-side spelling.
DML_ON_READ=$(awk '
  /\.(Query|QueryRow)\(ctx, `/ { pending = 3; file = FILENAME; line = FNR; call = $0; next }
  pending > 0 {
    if ($0 ~ /^[[:space:]]*(INSERT|UPDATE|DELETE|REPLACE)[[:space:]]/) {
      printf "%s:%d:%s\n", file, line, call
      pending = 0
      next
    }
    pending--
  }
' $(find internal -name '*.go' ! -name '*_test.go') || true)
if [ -n "$DML_ON_READ" ]; then
  echo "============================================================"
  echo "[single-writer] write statement issued through the read path:"
  echo "$DML_ON_READ"
  echo "Use db.Exec, db.WithTx, or db.QueryRowWrite for UPDATE ... RETURNING."
  echo ""
  FAIL=1
fi

# The pool size is the guarantee; anything but one connection is not a writer.
if ! grep -q 'openHandle(dsn, 1, 1, cfg.ConnMaxLifetime, pragmas)' internal/database/database.go; then
  echo "============================================================"
  echo "[single-writer] the write handle is no longer opened with one connection"
  echo "(expected openHandle(dsn, 1, 1, ...) in internal/database/database.go)."
  echo ""
  FAIL=1
fi

# Nothing outside internal/database may hold a raw handle and write on it. The
# one sanctioned consumer is mibdb, which takes the write handle by name.
RAW_HANDLE=$(grep -rEn '\.WriteConn\(\)' --include='*.go' internal cmd 2>/dev/null \
  | grep -v '^internal/database/' \
  | grep -v 'internal/api/server_init.go' || true)
if [ -n "$RAW_HANDLE" ]; then
  echo "============================================================"
  echo "[single-writer] raw write handle taken outside the sanctioned sites:"
  echo "$RAW_HANDLE"
  echo "Use the repositories; they already serialise on the write connection."
  echo ""
  FAIL=1
fi

if [ "$FAIL" -ne 0 ]; then
  echo "FAIL: single-writer gate (#2453)."
  exit 1
fi

echo "OK: every write is serialised on the single write connection."
