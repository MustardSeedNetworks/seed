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
assert_rejected \
  "mutable-setup-node-action" \
  'actions/setup-node@48b55a011bda9f5d6aeb4c2d9c7362e8dae4041e' \
  'actions/setup-node@v6'
assert_rejected \
  "new-mutable-action" \
  $'    steps:\n      - name: Install build dependencies' \
  $'    steps:\n      - uses: example/untrusted-action@main\n      - name: Install build dependencies'

echo "release workflow contract mutation tests passed."
