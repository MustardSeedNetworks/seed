#!/usr/bin/env bash
# check-release-workflow-contract.sh — dispatch-mode regression gate for release.yml.
#
# The default workflow_dispatch value for dry_run is true. A provenance-only
# backfill must therefore bypass that default, while a normal dry-run must not
# attest or publish. Keep the mode predicates explicit and independently gated.

set -euo pipefail

workflow=".github/workflows/release.yml"

require() {
  local pattern="$1"
  if ! grep -Fq -- "$pattern" "$workflow"; then
    echo "release workflow contract missing: $pattern" >&2
    exit 1
  fi
}

require "if: \${{ !inputs.provenance_only }}"
require "if: \${{ !cancelled() && ((inputs.provenance_only && needs.goreleaser-backfill-hashes.result == 'success') || (!inputs.provenance_only && !inputs.dry_run && needs.goreleaser.result == 'success')) }}"
require "if: \${{ !inputs.dry_run && !inputs.provenance_only }}"

if ! awk '
  /- name: Install Syft \(SBOM\) inside container/ { in_syft_step = 1; next }
  in_syft_step && /^        shell: bash$/ { found_bash = 1; exit }
  in_syft_step && /^      - name:/ { exit }
  END { exit !found_bash }
' "$workflow"; then
  echo "release workflow contract missing Bash shell for Syft installation" >&2
  exit 1
fi

attests() {
  local provenance_only="$1"
  local dry_run="$2"
  local goreleaser="$3"
  local backfill="$4"

  [[ "$provenance_only" == true && "$backfill" == success ]] ||
    [[ "$provenance_only" != true && "$dry_run" != true && "$goreleaser" == success ]]
}

assert_mode() {
  local name="$1"
  local expected="$2"
  shift 2

  if attests "$@"; then
    actual=true
  else
    actual=false
  fi
  if [[ "$actual" != "$expected" ]]; then
    echo "release workflow contract failed for $name: got $actual, want $expected" >&2
    exit 1
  fi
}

assert_mode "dry-run" false false true success skipped
assert_mode "normal release" true false false success skipped
assert_mode "provenance-only default dry-run" true true true skipped success
assert_mode "failed provenance-only backfill" false true true skipped failure

echo "release workflow contract: dry-run, release, and provenance-only modes are disjoint."
