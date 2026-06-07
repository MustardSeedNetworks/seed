#!/usr/bin/env bash
# check-schema-drift.sh — fail if docs/schemas/api/*.json drifts from
# the generator's current output. Wired into ci.yml so changes to
# internal/api Go DTOs without a matching `make schema` commit get
# caught at PR time instead of after merge.
#
# Mirrors the gate documented in MustardSeedNetworks/niac-go ADR 0001
# (docs/adr/0001-schema-generation-from-go-structs.md).
#
# Run locally with: ./scripts/check-schema-drift.sh
# CI uses the same invocation.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

cd "$ROOT"

go run ./cmd/seed-schema -o "$TMP/api"

if ! diff -ru docs/schemas/api "$TMP/api" >/dev/null; then
  echo "::error::docs/schemas/api/*.json is stale. Run 'make schema' and commit the result." >&2
  echo "" >&2
  echo "Diff:" >&2
  diff -ru docs/schemas/api "$TMP/api" || true
  exit 1
fi

echo "API schemas are up to date."
