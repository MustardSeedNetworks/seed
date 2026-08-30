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

# require_step_condition pins a condition to the step that must carry it.
# A bare `require` cannot: the publish predicate appears on more than one step,
# so dropping it from one of them would still match elsewhere and pass.
require_step_condition() {
  local step="$1"
  local condition="$2"

  if ! awk -v step="- name: $step" -v cond="$condition" '
    index($0, step) { in_step = 1; next }
    in_step && index($0, cond) { found = 1 }
    in_step && /^      - name:/ { exit }
    END { exit !found }
  ' "$workflow"; then
    echo "release workflow contract: step \"$step\" is missing its condition: $condition" >&2
    exit 1
  fi
}

# require_job_condition pins a condition to the job that must carry it, the way
# require_step_condition does for steps. A bare `require` cannot distinguish a
# job-level `if:` from an identical one on a step elsewhere in the file.
require_job_condition() {
  local job="$1"
  local condition="$2"

  if ! awk -v job="  $job:" -v cond="$condition" '
    $0 == job { in_job = 1; next }
    in_job && index($0, cond) { found = 1 }
    in_job && /^  [a-z0-9-]+:$/ { exit }
    END { exit !found }
  ' "$workflow"; then
    echo "release workflow contract: job \"$job\" is missing its condition: $condition" >&2
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

require "if: \${{ !cancelled() && github.event_name == 'push' && !inputs.dry_run && needs.goreleaser.result == 'success' }}"
# Publishing is reserved for pushed tags: only a push can satisfy the
# verify-tag assertion that the commit passed CI Complete, and a v* tag can be
# created on any commit. Both halves are pinned -- the publish predicate and the
# snapshot predicate that must be its exact complement -- so a change that drops
# the event check from one of them cannot pass silently.
require_step_condition "Run goreleaser (publish)" \
  "if: \${{ github.event_name == 'push' && !inputs.dry_run }}"
require_step_condition "Capture artifact hashes for SLSA provenance" \
  "if: \${{ github.event_name == 'push' && !inputs.dry_run }}"
require_step_condition "Run goreleaser (snapshot/dry-run)" \
  "if: \${{ github.event_name != 'push' || inputs.dry_run }}"
require_step_condition "Upload dry-run artifact bundle for inspection" \
  "if: \${{ github.event_name != 'push' || inputs.dry_run }}"
require_step_condition "Refuse a manual dispatch that asks to publish" \
  "if: \${{ github.event_name == 'workflow_dispatch' && !inputs.dry_run }}"
require_job_condition "publish-release" \
  "if: \${{ github.event_name == 'push' && !inputs.dry_run }}"
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

# The three models below mirror the workflow's own predicates. They are what
# turns this file from a string check into a statement about behaviour: a change
# to a condition has to be reflected here, and the mode assertions then say
# whether the new behaviour is the intended one.

# publishes mirrors the "Run goreleaser (publish)" step.
publishes() {
  local event="$1"
  local dry_run="$2"

  [[ "$event" == push && "$dry_run" != true ]]
}

# refuses mirrors the verify-tag step that rejects a dispatch asking to publish,
# rather than silently downgrading it to a snapshot.
refuses() {
  local event="$1"
  local dry_run="$2"

  [[ "$event" == workflow_dispatch && "$dry_run" != true ]]
}

# attests mirrors the provenance job. Only a pushed, non-dry-run tag whose
# goreleaser job succeeded produces an attestation -- a snapshot has nothing
# published to attest against.
attests() {
  local event="$1"
  local dry_run="$2"
  local goreleaser="$3"

  [[ "$event" == push && "$dry_run" != true && "$goreleaser" == success ]]
}

assert_mode() {
  local model="$1"
  local name="$2"
  local expected="$3"
  shift 3

  if "$model" "$@"; then
    actual=true
  else
    actual=false
  fi
  if [[ "$actual" != "$expected" ]]; then
    echo "release workflow contract failed for $model/$name: got $actual, want $expected" >&2
    exit 1
  fi
}

# Only a pushed tag publishes. The dispatch rows are the regression this guard
# exists for: before #2228 a dispatch with dry_run=false published without the
# CI Complete assertion ever running.
assert_mode publishes "pushed tag" true push false
assert_mode publishes "dispatch asking to publish" false workflow_dispatch false
assert_mode publishes "dispatch dry-run" false workflow_dispatch true

assert_mode refuses "dispatch asking to publish" true workflow_dispatch false
assert_mode refuses "dispatch dry-run" false workflow_dispatch true
assert_mode refuses "pushed tag" false push false

assert_mode attests "normal release" true push false success
assert_mode attests "dispatch dry-run" false workflow_dispatch true success
assert_mode attests "dispatch asking to publish" false workflow_dispatch false success
assert_mode attests "pushed tag whose goreleaser failed" false push false failure

echo "release workflow contract: modes, checksums, pins, and permissions verified."
