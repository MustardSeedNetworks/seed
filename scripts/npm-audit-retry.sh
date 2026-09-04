#!/usr/bin/env bash
#
# npm audit, retried, because the advisory endpoint is somebody else's uptime.
#
# `npm audit` posts the dependency set to registry.npmjs.org's advisory endpoint.
# When that endpoint is unavailable the command exits non-zero with an error that
# says nothing about our dependencies, and because Security Scanning is a
# required check, the whole fleet's merge queue stops. That happened on
# 2026-09-04 for about two hours (#2387):
#
#   npm warn audit 503 Service Unavailable - POST .../security/advisories/bulk
#   npm error audit endpoint returned an error
#
# Retrying fixes the common blip. It deliberately does NOT paper over a sustained
# outage: if every attempt fails, so does the build. A gate that passes when it
# could not run is not a gate — see the decision recorded on #2387.
#
# Usage: scripts/npm-audit-retry.sh [audit-level]
set -euo pipefail

LEVEL="${1:-high}"
ATTEMPTS="${NPM_AUDIT_ATTEMPTS:-3}"
DELAY="${NPM_AUDIT_RETRY_DELAY:-15}"

cd "$(dirname "$0")/../ui"

for attempt in $(seq 1 "$ATTEMPTS"); do
    set +e
    OUT="$(npm audit --audit-level="$LEVEL" 2>&1)"
    STATUS=$?
    set -e

    if [ "$STATUS" -eq 0 ]; then
        echo "$OUT" | grep -E "(found|vulnerabilities)" | head -3 ||
            printf "   No vulnerabilities found\n"
        exit 0
    fi

    # Distinguish "the endpoint is down" from "we have vulnerabilities". Only the
    # former is worth retrying; a real finding will not fix itself in 15 seconds.
    if ! grep -qE "audit endpoint returned an error|Service Unavailable|ECONNRESET|ETIMEDOUT|socket hang up|network" <<<"$OUT"; then
        echo "$OUT" | tail -40
        exit "$STATUS"
    fi

    printf "   npm's advisory endpoint did not answer (attempt %d/%d)\n" "$attempt" "$ATTEMPTS"
    if [ "$attempt" -lt "$ATTEMPTS" ]; then
        sleep "$((DELAY * attempt))"
    fi
done

echo "$OUT" | tail -40
printf "::error::npm audit could not reach the advisory endpoint after %d attempts. " "$ATTEMPTS"
printf "This is npm's availability, not a finding about our dependencies — but the "
printf "audit did not run, so the build fails rather than claiming a clean scan (#2387).\n"
exit 1
