#!/usr/bin/env bash
# check-error-shape.sh — one error contract for the HTTP API (#2749).
#
# Every 4xx/5xx from internal/api carries the JSON ErrorResponse
# ({"error", "code", "details"}) written by writeError or
# sendErrorResponseWithDetails. net/http's plain-text writers reply
# with text/plain bodies that the UI's ApiError cannot parse, so a
# single call reintroduces a second error shape. This gate bans them
# in production code; tests may still use them to fake upstreams.
#
# Run locally: scripts/check-error-shape.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

hits="$(grep -rnE '\bhttp\.(Error|NotFound)\(' internal/api --include='*.go' |
	grep -v '_test\.go:' || true)"

if [[ -n "$hits" ]]; then
	echo "❌ plain-text error writers in internal/api (use writeError):"
	echo "$hits"
	exit 1
fi
echo "✓ every internal/api error uses the JSON ErrorResponse"
