#!/usr/bin/env bash
# check-json-casing.sh — JSON wire-casing discipline gate (ADR-0010, revised 2026-06-14).
#
# ADR-0010 (pure boundary mapping): every JSON `json:"..."` wire tag our API
# emits or accepts is camelCase. There are NO wire-level snake_case exceptions
# and NO key allow-list / baseline. snake_case lives only OFF the wire — config
# files (internal/config), SQL columns, and internal adapters that parse an
# external tool's output.
#
# External-tool adapters: a struct that `json.Unmarshal`s an external tool's
# output (e.g. macOS `system_profiler SPBluetoothDataType -json` in
# bluetooth_darwin.go) MUST match that tool's casing to deserialize it, and is
# mapped to a camelCase domain type before any API emission. Such a file opts
# out of this gate with the marker comment:
#
#     // json-wire:external-adapter — <which external tool, and where it maps>
#
# That is a STRUCTURAL boundary ("this file parses a foreign format"), not a
# per-key allow-list — there is nothing to grandfather. Do NOT add the marker
# to a file that emits our own wire; fix the casing instead.
#
# This gate scans `json:"..."` struct tags in internal/api + internal/discovery
# for snake_case keys and FAILS on any that are not inside a marked adapter.
#
# Run locally: scripts/check-json-casing.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SCAN_DIRS=("internal/api" "internal/discovery")
EXEMPT_MARKER="json-wire:external-adapter"

# Files that declare themselves external-tool adapters (parse a foreign format,
# map to camelCase before emission). Their snake tags are the tool's contract.
exempt_files() {
	grep -rlF "$EXEMPT_MARKER" "${SCAN_DIRS[@]}" --include='*.go' 2>/dev/null | sort -u || true
}

# violations prints "path\ttag" for every snake_case json tag in a non-exempt,
# non-test file. `|| true` tolerates the zero-match (healthy) case under pipefail.
violations() {
	local exempt
	exempt="$(exempt_files)"
	grep -rnoE 'json:"[a-z][a-z0-9]*_[a-z0-9_]+[^"]*"' "${SCAN_DIRS[@]}" --include='*.go' 2>/dev/null \
		| grep -v '_test\.go:' \
		| { [[ -n "$exempt" ]] && grep -vFf <(printf '%s\n' "$exempt") || cat; } \
		| sed -E 's/^([^:]+):[0-9]+:json:"([^",]+).*/\1\t\2/' \
		| sort -u || true
}

found="$(violations)"

if [[ -n "$found" ]]; then
	echo "::error::snake_case JSON wire tag(s) — use camelCase (ADR-0010, no exceptions):" >&2
	echo "$found" | sed 's/^/  /' >&2
	echo "" >&2
	echo "If this file parses an external tool's output (and maps to camelCase" >&2
	echo "before emission), mark it: // ${EXEMPT_MARKER} — <tool, where it maps>" >&2
	exit 1
fi

echo "JSON casing gate OK — wire is 100% camelCase (ADR-0010); external-tool adapters exempt by marker: $(exempt_files | tr '\n' ' ')"
