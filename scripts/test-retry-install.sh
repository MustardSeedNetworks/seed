#!/usr/bin/env bash
# Both outcomes of the benchstat retry, because the failure path is the one
# that was wrong last time and the one CI never exercises.
set -uo pipefail

script="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/retry-install-benchstat.sh"
failures=0

check() {
  local name="$1" want="$2"
  shift 2
  "$@" >/dev/null 2>&1
  local got=$?
  if [ "$got" -ne "$want" ]; then
    echo "FAIL: $name exited $got, want $want" >&2
    failures=$((failures + 1))
  else
    echo "ok: $name exited $want"
  fi
}

# Every attempt fails: must exit non-zero rather than fall out of the loop.
check "all attempts fail" 1 env ATTEMPTS=3 INSTALL_CMD=false "$script"
# Succeeds immediately.
check "first attempt succeeds" 0 env ATTEMPTS=3 INSTALL_CMD=true "$script"
# A single attempt that fails must still report failure.
check "single attempt fails" 1 env ATTEMPTS=1 INSTALL_CMD=false "$script"

if [ "$failures" -ne 0 ]; then
  echo "$failures retry self-test failure(s)" >&2
  exit 1
fi
echo "retry self-test passed"
