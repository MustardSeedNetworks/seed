#!/usr/bin/env bash
# check-types-drift.sh — fail if ui/src/types/generated/*.ts drifts from
# what gen-types.mjs would produce against the committed schemas.
#
# Wired into ci.yml so a stale TS bundle (Go DTO changed → schema
# regenerated → TS forgotten) gets caught at PR time. Paired with the
# Go-side schema-drift gate in scripts/check-schema-drift.sh.

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SNAPSHOT="$(mktemp -d)"
trap 'rm -rf "$SNAPSHOT"' EXIT

if [[ -d ui/src/types/generated ]]; then
  cp -R ui/src/types/generated "$SNAPSHOT/before"
else
  mkdir -p "$SNAPSHOT/before"
fi

npm --prefix ui run gen-types >/dev/null

if ! diff -ru "$SNAPSHOT/before" ui/src/types/generated >/dev/null; then
  echo "::error::ui/src/types/generated/*.ts is stale. Run 'npm --prefix ui run gen-types' and commit." >&2
  echo "" >&2
  echo "Diff:" >&2
  diff -ru "$SNAPSHOT/before" ui/src/types/generated || true
  exit 1
fi

echo "Generated TypeScript types are up to date."
