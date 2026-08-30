#!/usr/bin/env bash

set -euo pipefail

source_workflow=".github/workflows/release.yml"
checker="./scripts/check-release-workflow-contract.sh"
fixture_dir=$(mktemp -d)
trap 'rm -rf "$fixture_dir"' EXIT

assert_rejected() {
  local name="$1"
  local old="$2"
  local new="$3"
  local fixture="$fixture_dir/$name.yml"

  OLD="$old" NEW="$new" python3 - "$source_workflow" "$fixture" <<'PY'
import os
import pathlib
import sys

source = pathlib.Path(sys.argv[1]).read_text()
old = os.environ["OLD"]
if source.count(old) != 1:
    raise SystemExit(f"mutation source occurs {source.count(old)} times, want 1: {old!r}")
pathlib.Path(sys.argv[2]).write_text(source.replace(old, os.environ["NEW"], 1))
PY

  if RELEASE_WORKFLOW_PATH="$fixture" "$checker" >/dev/null 2>&1; then
    echo "release workflow contract accepted mutation: $name" >&2
    exit 1
  fi
}

"$checker"
assert_rejected \
  "iperf-wrong-file" \
  "echo \"\${IPERF3_SHA256}  iperf3.tar.gz\" | sha256sum -c -" \
  "echo \"\${IPERF3_SHA256}  different-file\" | sha256sum -c -"
assert_rejected \
  "cosign-wrong-file" \
  "echo \"\${COSIGN_SHA256}  \$cosign_dir/cosign-linux-amd64\" | sha256sum -c -" \
  "echo \"\${COSIGN_SHA256}  \$cosign_dir/different-file\" | sha256sum -c -"
assert_rejected \
  "syft-wrong-file" \
  "echo \"\${SYFT_SHA256}  \$syft_dir/syft.tar.gz\" | sha256sum -c -" \
  "echo \"\${SYFT_SHA256}  \$syft_dir/different-file\" | sha256sum -c -"
assert_rejected \
  "mutable-latest-url" \
  "releases/download/\${COSIGN_VERSION}/cosign-linux-amd64" \
  'releases/latest/download/cosign-linux-amd64'
assert_rejected \
  "workflow-wide-write" \
  $'permissions:\n  contents: read' \
  $'permissions:\n  contents: write\n  id-token: write'
assert_rejected \
  "workflow-extra-permission" \
  $'permissions:\n  contents: read' \
  $'permissions:\n  contents: read\n  id-token: write'
# setup-node moved into the local composite so release.yml and ci.yml cannot
# drift apart on the Node pin. The contract still has to reach it there, so
# this stages a mutated copy of the composite and checks the checker follows
# the `uses: ./...` ref into it instead of skipping local actions.
composite_dir=".github/actions/setup-node"
setup_node_ref=$(grep -oE 'actions/setup-node@[0-9a-f]{40}' "$composite_dir/action.yml" | sort -u)
if [ "$(printf '%s\n' "$setup_node_ref" | grep -c .)" -ne 1 ]; then
  echo "expected exactly one SHA-pinned actions/setup-node ref in $composite_dir/action.yml" >&2
  exit 1
fi

fixture_root="$fixture_dir/root"
mkdir -p "$fixture_root/$composite_dir"
sed "s|$setup_node_ref|actions/setup-node@v6|" \
  "$composite_dir/action.yml" >"$fixture_root/$composite_dir/action.yml"
if RELEASE_REPO_ROOT="$fixture_root" "$checker" >/dev/null 2>&1; then
  echo "release workflow contract accepted mutation: mutable-setup-node-action" >&2
  exit 1
fi

# A composite that does not exist must fail loudly rather than pass vacuously.
if RELEASE_REPO_ROOT="$fixture_dir/empty" "$checker" >/dev/null 2>&1; then
  echo "release workflow contract accepted a missing composite" >&2
  exit 1
fi
# The publish bypass fixed in #2228: dropping the event check from any of the
# publishing predicates must be rejected, or the guard would not have caught the
# defect it was extended for.
assert_rejected \
  "publish-without-push-event" \
  $'      - name: Run goreleaser (publish)\n        if: ${{ github.event_name == \'push\' && !inputs.dry_run }}' \
  $'      - name: Run goreleaser (publish)\n        if: ${{ !inputs.dry_run }}'
assert_rejected \
  "publish-release-job-without-push-event" \
  $'  publish-release:\n    name: Publish verified release\n    if: ${{ github.event_name == \'push\' && !inputs.dry_run }}' \
  $'  publish-release:\n    name: Publish verified release\n    if: ${{ !inputs.dry_run }}'
assert_rejected \
  "provenance-without-push-event" \
  "if: \${{ !cancelled() && github.event_name == 'push' && !inputs.dry_run && needs.goreleaser.result == 'success' }}" \
  "if: \${{ !cancelled() && !inputs.dry_run && needs.goreleaser.result == 'success' }}"
assert_rejected \
  "snapshot-no-longer-complements-publish" \
  $'      - name: Run goreleaser (snapshot/dry-run)\n        if: ${{ github.event_name != \'push\' || inputs.dry_run }}' \
  $'      - name: Run goreleaser (snapshot/dry-run)\n        if: inputs.dry_run'
assert_rejected \
  "dispatch-publish-refusal-removed" \
  "if: \${{ github.event_name == 'workflow_dispatch' && !inputs.dry_run }}" \
  "if: false"

assert_rejected \
  "new-mutable-action" \
  $'    steps:\n      - name: Probe for build dependencies' \
  $'    steps:\n      - uses: example/untrusted-action@main\n      - name: Probe for build dependencies'

echo "release workflow contract mutation tests passed."
