#!/usr/bin/env bash
# check-json-casing.sh — JSON wire-casing discipline gate (ADR-0010).
#
# ADR-0010: JSON API wire tags are camelCase. The config-file format
# (internal/config) and SQL columns are snake_case by design and are NOT
# scanned here. Protocol-mandated keys (OAuth client_id, etc.) keep their spec
# casing and are grandfathered via the baseline.
#
# This gate scans `json:"..."` struct tags in internal/api + internal/discovery
# for snake_case keys and compares them to a committed baseline
# (scripts/json-casing-baseline.txt). It is a RATCHET:
#   - it FAILS if a NEW snake_case tag appears that is not in the baseline
#     (i.e. new drift), and
#   - it passes when violations only shrink.
# Phase 8 normalizes the baselined tags to camelCase, regenerating the baseline
# (smaller) each step until it is empty.
#
# Regenerate the baseline after cleaning (or to grandfather a legitimate
# protocol key):  scripts/check-json-casing.sh --update
#
# Run locally: scripts/check-json-casing.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

BASELINE="scripts/json-casing-baseline.txt"
SCAN_DIRS=("internal/api" "internal/discovery")

# current_violations prints sorted "path\ttag" lines for every snake_case JSON
# tag in the scanned dirs (excluding tests). A snake_case tag is a json tag
# whose key contains an underscore between lowercase/alnum segments.
current_violations() {
	# -r recurse, only .go, skip _test.go; match json:"...snake...".
	grep -rnoE 'json:"[a-z][a-z0-9]*_[a-z0-9_]+[^"]*"' "${SCAN_DIRS[@]}" --include='*.go' 2>/dev/null \
		| grep -v '_test\.go:' \
		| sed -E 's/^([^:]+):[0-9]+:json:"([^",]+).*/\1\t\2/' \
		| sort -u
}

if [[ "${1:-}" == "--update" ]]; then
	current_violations >"$BASELINE"
	echo "Wrote $(wc -l <"$BASELINE" | tr -d ' ') baselined snake_case JSON tag(s) to $BASELINE"
	exit 0
fi

if [[ ! -f "$BASELINE" ]]; then
	echo "::error::$BASELINE missing — run scripts/check-json-casing.sh --update" >&2
	exit 1
fi

# New violations = current entries not present in the baseline.
new=$(comm -23 <(current_violations) <(sort -u "$BASELINE") || true)

if [[ -n "$new" ]]; then
	echo "::error::New snake_case JSON wire tag(s) introduced — use camelCase (ADR-0010):" >&2
	echo "$new" | sed 's/^/  /' >&2
	echo "" >&2
	echo "If this is a protocol-mandated key (e.g. OAuth client_id), grandfather it:" >&2
	echo "  scripts/check-json-casing.sh --update   # then commit the baseline" >&2
	exit 1
fi

# Informational: how much of the baseline remains to normalize.
remaining=$(current_violations | comm -12 - <(sort -u "$BASELINE") | wc -l | tr -d ' ')
echo "JSON casing gate OK — no new snake_case wire tags. ${remaining} baselined tag(s) remain to normalize (ADR-0010 / Phase 8)."
