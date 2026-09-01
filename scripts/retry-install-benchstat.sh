#!/usr/bin/env bash
# Install benchstat, retrying a transient module-proxy failure.
#
# Kept as a script rather than inline YAML so the failure path can be tested:
# the previous hand-rolled retry in this repo reported success when every
# attempt failed (#1950), and nothing caught it because nothing ran it.
set -uo pipefail

BENCHSTAT_VERSION="${BENCHSTAT_VERSION:-v0.0.0-20260825160852-19be9d8e6c70}"
ATTEMPTS="${ATTEMPTS:-3}"
# Overridable so the self-test can drive both outcomes without a network.
INSTALL_CMD="${INSTALL_CMD:-go install golang.org/x/perf/cmd/benchstat@${BENCHSTAT_VERSION}}"

attempt=1
while [ "$attempt" -le "$ATTEMPTS" ]; do
  if $INSTALL_CMD; then
    exit 0
  fi
  echo "benchstat install attempt ${attempt}/${ATTEMPTS} failed" >&2
  if [ "$attempt" -lt "$ATTEMPTS" ]; then
    sleep "$((attempt * 5))"
  fi
  attempt=$((attempt + 1))
done

echo "::error::benchstat install failed after ${ATTEMPTS} attempts" >&2
exit 1
