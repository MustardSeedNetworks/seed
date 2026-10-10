#!/usr/bin/env bash
# check-openapi-drift.sh — fail if docs/openapi.yaml drifts from what the
# route registry generates. The API description is generated from the routes
# the daemon serves, so a new or changed route brings its documentation along
# or fails here.
#
# Run locally with: ./scripts/check-openapi-drift.sh (refresh: make openapi)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cd "$ROOT"

go run ./cmd/seed-openapi -o "$TMP/openapi.yaml"

if ! diff -u docs/openapi.yaml "$TMP/openapi.yaml"; then
  echo "::error::docs/openapi.yaml is stale. Run 'make openapi' and commit the result." >&2
  exit 1
fi

echo "OpenAPI description is up to date."
