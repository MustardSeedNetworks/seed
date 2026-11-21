#!/usr/bin/env bash
# check-release-workflow-contract.sh — release-mode and supply-chain regression gate.

set -euo pipefail

workflow="${RELEASE_WORKFLOW_PATH:-.github/workflows/release.yml}"
# Local composite refs (`uses: ./.github/actions/...`) resolve against the repo
# root, not the workflow's directory. Overridable so the self-test can stage a
# mutated composite without touching the tree.
repo_root="${RELEASE_REPO_ROOT:-.}"
goreleaser_config="${GORELEASER_CONFIG_PATH:-$repo_root/.goreleaser.yml}"
release_please_config="${RELEASE_PLEASE_CONFIG_PATH:-$repo_root/.github/release-please-config.json}"
release_artifacts="${RELEASE_ARTIFACTS_PATH:-$repo_root/.github/release-artifacts.txt}"

require() {
  local pattern="$1"
  if ! grep -Fq -- "$pattern" "$workflow"; then
    echo "release workflow contract missing: $pattern" >&2
    exit 1
  fi
}

# require_pin asserts the SHAPE of a Renovate-managed version pin rather than
# its literal value. The value lives in two places — the workflow and this
# gate — and only the workflow is what Renovate edits, so a literal `require`
# here made every toolchain bump red by construction and unfixable by a rebase
# (#2622: PR #2561 sat red from 2026-09-10 for exactly this reason).
#
# The gate's purpose survives intact: it exists so nobody silently UN-pins the
# release toolchain, and "pinned to an exact version and a full digest" is a
# shape a bump satisfies and an un-pinning does not.
require_pin() {
  local what="$1"
  local pattern="$2"
  if ! grep -Eq -- "$pattern" "$workflow"; then
    echo "release workflow contract: $what is missing or no longer pinned (want /$pattern/)" >&2
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

require "if: \${{ !cancelled() && github.event_name == 'push' && !inputs.dry_run && needs.goreleaser.result == 'success' && needs.macos-pkg.result == 'success' }}"
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

# The macOS .pkg is a required release artifact. It is built outside goreleaser,
# so three things keep it from being dropped: the job must exist and feed the
# provenance job (which publish-release waits for), its publishing steps carry
# the same push predicate as goreleaser's, and the artifact list names it.
require_job_condition "macos-pkg" "needs: [goreleaser]"
require_job_condition "macos-pkg" "runs-on: macos-latest"
require_job_condition "provenance" "needs: [goreleaser, macos-pkg]"
require "deploy/macos/build-pkg.sh"
require "PKG=seed-\${version}-darwin-arm64.pkg"
for step in "Install Syft and Cosign" "SBOM, sign, and publish the .pkg" "Add the .pkg to the provenance subjects"; do
  require_step_condition "$step" \
    "if: \${{ github.event_name == 'push' && !inputs.dry_run }}"
done
require_step_condition "Upload dry-run .pkg for inspection" \
  "if: \${{ github.event_name != 'push' || inputs.dry_run }}"
if ! grep -Fxq 'seed-{version}-darwin-arm64.pkg' "$release_artifacts"; then
  echo "release workflow contract: $release_artifacts must list seed-{version}-darwin-arm64.pkg" >&2
  exit 1
fi
# Renovate owns these four values; the gate owns their shape. See require_pin.
require_pin 'IPERF3_VERSION' '^ +IPERF3_VERSION: "[0-9]+\.[0-9]+(\.[0-9]+)?"$'
require_pin 'IPERF3_SHA256' '^ +IPERF3_SHA256: "[0-9a-f]{64}"$'
require_pin 'the goreleaser-cross image' \
  '^ +image: goreleaser/goreleaser-cross:v[0-9]+\.[0-9]+\.[0-9]+(-[0-9]+)?(-v[0-9.]+)?@sha256:[0-9a-f]{64}$'
require_pin 'SYFT_VERSION' '^ +SYFT_VERSION: "[0-9]+\.[0-9]+\.[0-9]+"$'
require_pin 'SYFT_SHA256' '^ +SYFT_SHA256: "[0-9a-f]{64}"$'
require_pin 'COSIGN_VERSION' '^ +COSIGN_VERSION: "v[0-9]+\.[0-9]+\.[0-9]+"$'
require_pin 'COSIGN_SHA256' '^ +COSIGN_SHA256: "[0-9a-f]{64}"$'
# The macOS job carries its own copy of the tool pins, with darwin checksums.
# Renovate does not manage them, so the contract keeps them in step with the
# linux pins above rather than letting one set drift.
require_pin 'SYFT_DARWIN_VERSION' '^ +SYFT_DARWIN_VERSION: "[0-9]+\.[0-9]+\.[0-9]+"$'
require_pin 'SYFT_DARWIN_SHA256' '^ +SYFT_DARWIN_SHA256: "[0-9a-f]{64}"$'
require_pin 'COSIGN_DARWIN_VERSION' '^ +COSIGN_DARWIN_VERSION: "v[0-9]+\.[0-9]+\.[0-9]+"$'
require_pin 'COSIGN_DARWIN_SHA256' '^ +COSIGN_DARWIN_SHA256: "[0-9a-f]{64}"$'
pin_value() {
  grep -oE "^ +$1: \"[^\"]+\"" "$workflow" | head -1 | cut -d'"' -f2
}
for tool in SYFT COSIGN; do
  if [[ "$(pin_value "${tool}_VERSION")" != "$(pin_value "${tool}_DARWIN_VERSION")" ]]; then
    echo "release workflow contract: ${tool}_DARWIN_VERSION must equal ${tool}_VERSION (rotate the darwin checksum with it)" >&2
    exit 1
  fi
done
require "syft_dir=\$(mktemp -d)"
require "trap 'rm -rf \"\$syft_dir\"' EXIT"
require "cosign_dir=\$(mktemp -d)"
require "trap 'rm -rf \"\$cosign_dir\"' EXIT"

# Releases are immutable once published, and an immutable release refuses
# every upload: v0.234.0 through v0.257.4 went out with zero assets because
# release-please published each one before goreleaser ran (#3157). The release
# must stay a draft from creation until the last asset is on it, so each
# hand-off is pinned: losing any one of them reproduces the empty release.
if ! jq -e '.packages["."] | .draft == true and ."force-tag-creation" == true' \
  "$release_please_config" >/dev/null; then
  echo "release workflow contract: release-please must create a draft and force the tag ($release_please_config)" >&2
  exit 1
fi
for setting in 'draft: true' 'use_existing_draft: true'; do
  if ! grep -Eq "^  ${setting}\$" "$goreleaser_config"; then
    echo "release workflow contract: goreleaser release block is missing '$setting' ($goreleaser_config)" >&2
    exit 1
  fi
done
require "run: gh release edit \"\$TARGET_TAG\" --draft=false"
require_job_condition "publish-release" "needs: [goreleaser, provenance]"

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
  /- name: Install Syft and Cosign$/ { in_step = 1; next }
  in_step && /^        shell: bash$/ { bash = 1 }
  in_step && /releases\/download\/v\$\{SYFT_DARWIN_VERSION\}\// { syft_download = 1 }
  in_step && /releases\/download\/\$\{COSIGN_DARWIN_VERSION\}\// { cosign_download = 1 }
  in_step && /echo "\$\{SYFT_DARWIN_SHA256\}  .*shasum -a 256 -c -$/ { syft_checksum = 1 }
  in_step && /echo "\$\{COSIGN_DARWIN_SHA256\}  .*shasum -a 256 -c -$/ { cosign_checksum = 1 }
  in_step && /^      - name:/ { exit }
  END { exit !(bash && syft_download && cosign_download && syft_checksum && cosign_checksum) }
' "$workflow"; then
  echo "release workflow contract missing checksum-bound Syft and Cosign installation in the macos-pkg job" >&2
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
# goreleaser and macos-pkg jobs succeeded produces an attestation -- a snapshot has nothing
# published to attest against.
attests() {
  local event="$1"
  local dry_run="$2"
  local goreleaser="$3"
  local macos_pkg="$4"

  [[ "$event" == push && "$dry_run" != true && "$goreleaser" == success && "$macos_pkg" == success ]]
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

assert_mode attests "normal release" true push false success success
assert_mode attests "dispatch dry-run" false workflow_dispatch true success success
assert_mode attests "dispatch asking to publish" false workflow_dispatch false success success
assert_mode attests "pushed tag whose goreleaser failed" false push false failure success
assert_mode attests "pushed tag whose macos-pkg failed" false push false success failure

echo "release workflow contract: modes, checksums, pins, and permissions verified."
