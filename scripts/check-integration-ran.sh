#!/usr/bin/env bash
#
# check-integration-ran.sh - fail an integration suite that tested nothing.
#
# `go test` exits 0 when a package has no tests matching the build tag, and 0
# when every test skips. Both report green while proving nothing -- the same
# failure mode as --passWithNoTests, with a nightly feedback loop.
#
# Counting tests is not enough on its own: the SNMP package also holds unit
# tests, so a build-tag mistake that excludes every integration test still
# leaves a healthy-looking count. The guard therefore requires named tests to
# have passed, which only the integration file can satisfy.
#
# Usage:
#   go test -json -tags integration ./path/... | \
#     scripts/check-integration-ran.sh TestOne TestTwo ...
#
# Exits non-zero if any required test did not pass, if any test failed, or if
# any test skipped.

set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "usage: $0 <required-test-name>..." >&2
  exit 2
fi

INPUT="$(cat)"

if [ -z "$INPUT" ]; then
  echo "::error::go test produced no JSON output at all — the suite did not run" >&2
  exit 1
fi

# Top-level test actions only: subtests carry a '/' in their name, and
# package-level lines have no "Test" field at all.
actions_for() {
  printf '%s\n' "$INPUT" \
    | grep -o "\"Action\":\"$1\",\"Package\":\"[^\"]*\",\"Test\":\"[^\"/]*\"" \
    | sed 's/.*"Test":"\([^"]*\)"/\1/' \
    || true
}

passed="$(actions_for pass)"
failed="$(actions_for fail)"
skipped="$(actions_for skip)"

count() {
  if [ -z "$1" ]; then
    echo 0
  else
    printf '%s\n' "$1" | wc -l | tr -d ' '
  fi
}

# Prints one indented bullet per newline-separated name.
bullets() { printf '%s\n' "$1" | sed 's/^/  - /'; }

echo "test results: $(count "$passed") passed, $(count "$failed") failed, $(count "$skipped") skipped"

status=0

if [ -n "$failed" ]; then
  echo "::error::test(s) failed:" >&2
  bullets "$failed" >&2
  status=1
fi

if [ -n "$skipped" ]; then
  echo "::error::test(s) skipped — a skipped test proves nothing here:" >&2
  bullets "$skipped" >&2
  status=1
fi

for required in "$@"; do
  if ! printf '%s\n' "$passed" | grep -qx "$required"; then
    echo "::error::required integration test '$required' did not run and pass." >&2
    echo "A suite that runs nothing is not a passing suite — check the build tag and the agent." >&2
    status=1
  fi
done

if [ "$status" -eq 0 ]; then
  echo "OK: all $# required integration test(s) ran and passed."
fi

exit "$status"
