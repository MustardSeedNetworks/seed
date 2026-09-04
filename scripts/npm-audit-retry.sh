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
# The first cut of this script (#2396) told the two cases apart by grepping
# npm's human-readable text for outage-sounding prose, ending in the bare word
# "network". A genuine High finding whose package name or advisory title
# happened to contain "network" matched that pattern too, so it was retried
# three times and reported as an unreachable endpoint instead of a
# vulnerability (#2416) — the opposite of #2387's fix, and worse: the gate
# still failed, but the message pointed away from the real finding.
#
# `npm audit --json` gives a deterministic signal instead: a transport failure
# returns a top-level "error" key, a completed audit returns
# "metadata.vulnerabilities". Decide on that, never on prose. Converges with
# stem#1004 / trellis#315, which implement the same distinction.
#
# Usage: scripts/npm-audit-retry.sh [audit-level]
set -uo pipefail

LEVEL="${1:-high}"
ATTEMPTS="${NPM_AUDIT_ATTEMPTS:-3}"
DELAY="${NPM_AUDIT_RETRY_DELAY:-15}"

cd "$(dirname "$0")/../ui" || exit 1

# verdict decides pass/fail from a completed `npm audit --json` payload (no
# top-level "error" key) on stdin — never from npm's prose. Exposed as a
# function, not inlined into main, so tests can pipe synthetic JSON straight
# at the decision without shelling out to npm or a registry. By the time
# verdict runs, main has already proven the audit completed, so a finding
# whose package or advisory title contains a transport-sounding word (the
# bare "network" that misfired in #2416) cannot be misread as an outage.
verdict() {
    local out vulns high critical total
    out="$(cat)"
    vulns="$(jq -e '.metadata.vulnerabilities' <<<"$out" 2>/dev/null)" || {
        printf '::error::npm audit returned no metadata.vulnerabilities -- unexpected output, failing closed\n' >&2
        printf '%s\n' "$out" >&2
        return 1
    }
    high="$(jq -r '.high // 0' <<<"$vulns")"
    critical="$(jq -r '.critical // 0' <<<"$vulns")"
    total="$(jq -r '.total // 0' <<<"$vulns")"
    printf 'npm audit: %s vulnerabilities found (%s high, %s critical)\n' "$total" "$high" "$critical"
    if [ "$((high + critical))" -gt 0 ]; then
        jq -r '(.vulnerabilities // {}) | to_entries[] | select(.value.severity == "high" or .value.severity == "critical") | "  \(.value.severity): \(.key) (\(.value.range // "unknown range"))"' <<<"$out"
        return 1
    fi
    return 0
}

main() {
    local out endpoint attempt
    for attempt in $(seq 1 "$ATTEMPTS"); do
        out="$(npm audit --json --audit-level="$LEVEL" || true)"

        if ! jq -e 'has("error")' >/dev/null 2>&1 <<<"$out"; then
            verdict <<<"$out"
            return $?
        fi

        endpoint="$(jq -r '(.error.summary | select(. != "")) // .message // "registry.npmjs.org advisory endpoint"' <<<"$out" 2>/dev/null)"
        endpoint="${endpoint:-registry.npmjs.org advisory endpoint}"

        if [ "$attempt" -lt "$ATTEMPTS" ]; then
            printf '   npm audit could not reach the advisory endpoint (attempt %d/%d): %s\n' \
                "$attempt" "$ATTEMPTS" "$endpoint"
            sleep "$((DELAY * attempt))"
            continue
        fi

        printf '::error::npm audit could not reach the advisory endpoint (registry.npmjs.org) after %d attempts: %s -- this is not a dependency finding, but the audit did not run so the gate fails rather than claiming a clean scan (#2387)\n' \
            "$ATTEMPTS" "$endpoint"
        return 1
    done
}

# Sourcing the script (as the test suite does, to call `verdict` directly on
# synthetic JSON) must not also run `main` against the real registry.
if [ "${BASH_SOURCE[0]}" = "${0}" ]; then
    main
    exit $?
fi
