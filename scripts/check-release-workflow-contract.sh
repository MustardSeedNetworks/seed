#!/usr/bin/env bash
# check-release-workflow-contract.sh — release-mode and supply-chain regression gate.

set -euo pipefail

workflow="${RELEASE_WORKFLOW_PATH:-.github/workflows/release.yml}"
# Local composite refs (`uses: ./.github/actions/...`) resolve against the repo
# root, not the workflow's directory. Overridable so the self-test can stage a
# mutated composite without touching the tree.
repo_root="${RELEASE_REPO_ROOT:-.}"

require() {
  local pattern="$1"
  if ! grep -Fq -- "$pattern" "$workflow"; then
    echo "release workflow contract missing: $pattern" >&2
    exit 1
  fi
}

# Every external action reachable from the release path must be SHA-pinned.
# Local composites are followed rather than skipped: the Node/npm pin lives in
# .github/actions/setup-node, so skipping them would leave the actions it calls
# unchecked on the path that produces signed, attested artifacts.
validate_action_pins() {
  local file="$1"
  local line
  local ref
  local composite

  while IFS= read -r line; do
    ref=$(awk '{ for (i = 1; i <= NF; i++) if ($i == "uses:") { print $(i + 1); exit } }' <<<"$line")
    if [[ "$ref" == ./* ]]; then
      composite="$repo_root/${ref#./}/action.yml"
      if [[ ! -f "$composite" ]]; then
        echo "release workflow contract references a missing composite: $ref" >&2
        exit 1
      fi
      validate_action_pins "$composite"
      continue
    fi
    if [[ ! "$ref" =~ ^[^@[:space:]]+@[0-9a-f]{40}$ ]]; then
      echo "release workflow contract has mutable action reference: $line" >&2
      exit 1
    fi
  done < <(grep -E '^[[:space:]]*(-[[:space:]]+)?uses:' "$file")
}

require "if: \${{ !inputs.provenance_only }}"
require "if: \${{ !cancelled() && ((inputs.provenance_only && needs.goreleaser-backfill-hashes.result == 'success') || (!inputs.provenance_only && !inputs.dry_run && needs.goreleaser.result == 'success')) }}"
require "if: \${{ !inputs.dry_run && !inputs.provenance_only }}"
require 'IPERF3_VERSION: "3.21"'
require 'IPERF3_SHA256: "656e4405ebd620121de7ceca3eaf43a88f79ea1b857d041a6a0b1314801acdd8"'
require 'image: goreleaser/goreleaser-cross:v1.27.0@sha256:3ce3506ee9179c4122ba0b5dc13ab564ff259fb65f45bfad005ddd5e4a3d326d'
require 'SYFT_VERSION: "1.51.0"'
require 'SYFT_SHA256: "2a2e837a2c8d59ec9af5472ee22d3b04ee463c4e44476ecf993fd1e5ab6ebc7f"'
require 'COSIGN_VERSION: "v3.1.3"'
require 'COSIGN_SHA256: "4629c757b7618056f8ddd7e2625ae9fdd94c0372a65049520bc7d9df9efc7f71"'
require "syft_dir=\$(mktemp -d)"
require "trap 'rm -rf \"\$syft_dir\"' EXIT"
require "cosign_dir=\$(mktemp -d)"
require "trap 'rm -rf \"\$cosign_dir\"' EXIT"

validate_action_pins "$workflow"

if grep -Fq '/releases/latest' "$workflow"; then
  echo "release workflow contract contains a mutable latest-release lookup" >&2
  exit 1
fi

if ! awk '
  /^permissions:$/ { top_permissions = 1; next }
  top_permissions && /^  contents: read$/ { contents_read = 1; next }
  top_permissions && /^  [a-z-]+:/ { unexpected = 1; next }
  top_permissions && /^[^ ]/ { exit }
  END { exit !(top_permissions && contents_read && !unexpected) }
' "$workflow"; then
  echo "release workflow contract missing read-only workflow permissions" >&2
  exit 1
fi

if ! awk '
  /- name: Download and build iperf3/ { in_step = 1; next }
  in_step && /^        shell: bash$/ { bash = 1 }
  in_step && /releases\/download\/\$\{IPERF3_VERSION\}\/iperf-\$\{IPERF3_VERSION\}\.tar\.gz/ { download = 1 }
  in_step && /echo "\$\{IPERF3_SHA256\}  iperf3\.tar\.gz" \| sha256sum -c -/ { checksum = 1 }
  in_step && /^      - name:/ { exit }
  END { exit !(bash && download && checksum) }
' "$workflow"; then
  echo "release workflow contract missing checksum-bound iperf3 installation under Bash" >&2
  exit 1
fi

if ! awk '
  /- name: Install Syft \(SBOM\) inside container/ { in_step = 1; next }
  in_step && /^        shell: bash$/ { bash = 1 }
  in_step && /echo "\$\{SYFT_SHA256\}  \$syft_dir\/syft\.tar\.gz" \| sha256sum -c -/ { checksum = 1 }
  in_step && /^      - name:/ { exit }
  END { exit !(bash && checksum) }
' "$workflow"; then
  echo "release workflow contract missing checksum-bound Syft installation under Bash" >&2
  exit 1
fi

if ! awk '
  /- name: Install Cosign inside container/ { in_step = 1; next }
  in_step && /^        shell: bash$/ { bash = 1 }
  in_step && /releases\/download\/\$\{COSIGN_VERSION\}\/cosign-linux-amd64/ { download = 1 }
  in_step && /echo "\$\{COSIGN_SHA256\}  \$cosign_dir\/cosign-linux-amd64" \| sha256sum -c -/ { checksum = 1 }
  in_step && /^      - name:/ { exit }
  END { exit !(bash && download && checksum) }
' "$workflow"; then
  echo "release workflow contract missing checksum-bound Cosign installation under Bash" >&2
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

echo "release workflow contract: modes, checksums, pins, and permissions verified."
