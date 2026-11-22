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

# assert_config_rejected mutates a release config the workflow depends on and
# hands it to the checker through that config's path override ($2).
assert_config_rejected() {
  local name="$1"
  local var="$2"
  local source="$3"
  local fixture="$fixture_dir/$name.${source##*.}"

  OLD="$4" NEW="$5" python3 - "$source" "$fixture" <<'PY'
import os
import pathlib
import sys

source = pathlib.Path(sys.argv[1]).read_text()
old = os.environ["OLD"]
if source.count(old) != 1:
    raise SystemExit(f"mutation source occurs {source.count(old)} times, want 1: {old!r}")
pathlib.Path(sys.argv[2]).write_text(source.replace(old, os.environ["NEW"], 1))
PY

  if env "$var=$fixture" "$checker" >/dev/null 2>&1; then
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
# The pin assertions check shape rather than a literal value (#2622), so each
# needs a case proving the shape still refuses an un-pinning. A bump — the thing
# the literals used to reject — must pass, which the unmutated "$checker" run
# above already asserts.
#
# Each mutation reads its own source out of the workflow rather than naming a
# version. Writing "v1.27.1" here would put the value in two places again and
# break this test on any checkout whose workflow has moved on, which is the
# exact defect #2622 is about.
image_pin=$(grep -oE 'goreleaser/goreleaser-cross:[^[:space:]]+' "$source_workflow" | head -1)
assert_rejected \
  "goreleaser-cross-no-digest" \
  "$image_pin" \
  "${image_pin%%@*}"
assert_rejected \
  "goreleaser-cross-floating-tag" \
  "$image_pin" \
  "goreleaser/goreleaser-cross:latest@${image_pin#*@}"

cosign_sha=$(grep -oE 'COSIGN_SHA256: "[0-9a-f]{64}"' "$source_workflow" | head -1)
assert_rejected \
  "cosign-truncated-sha" \
  "$cosign_sha" \
  'COSIGN_SHA256: "4629c757"'

syft_version=$(grep -oE 'SYFT_VERSION: "[^"]+"' "$source_workflow" | head -1)
assert_rejected \
  "syft-version-unpinned" \
  "$syft_version" \
  'SYFT_VERSION: "latest"'

iperf_sha=$(grep -oE 'IPERF3_SHA256: "[0-9a-f]{64}"' "$source_workflow" | head -1)
assert_rejected \
  "iperf-sha-empty" \
  "$iperf_sha" \
  'IPERF3_SHA256: ""'
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
  "if: \${{ !cancelled() && github.event_name == 'push' && !inputs.dry_run && needs.goreleaser.result == 'success' && needs.macos-pkg.result == 'success' }}" \
  "if: \${{ !cancelled() && !inputs.dry_run && needs.goreleaser.result == 'success' && needs.macos-pkg.result == 'success' }}"
assert_rejected \
  "provenance-ignores-macos-pkg-result" \
  " && needs.macos-pkg.result == 'success' }}" \
  " }}"
assert_rejected \
  "provenance-does-not-wait-for-macos-pkg" \
  "needs: [goreleaser, macos-pkg]" \
  "needs: [goreleaser]"
assert_rejected \
  "pkg-publishes-on-dispatch" \
  $'      - name: SBOM, sign, and publish the .pkg\n        if: ${{ github.event_name == \'push\' && !inputs.dry_run }}' \
  $'      - name: SBOM, sign, and publish the .pkg\n        if: ${{ !inputs.dry_run }}'
assert_rejected \
  "pkg-subjects-without-push-event" \
  $'      - name: Add the .pkg to the provenance subjects\n        if: ${{ github.event_name == \'push\' && !inputs.dry_run }}' \
  $'      - name: Add the .pkg to the provenance subjects\n        if: ${{ !inputs.dry_run }}'
assert_rejected \
  "pkg-signing-does-not-fail-closed-on-push" \
  "elif [ \"\${EVENT}\" = \"push\" ]; then" \
  "elif [ \"\${EVENT}\" = \"never\" ]; then"
assert_rejected \
  "pkg-signing-ignores-p12-secret" \
  $'P12_BASE64: ${{ secrets.MACOS_SIGNING_P12_BASE64 }}\n          P12_PASSWORD: ${{ secrets.MACOS_SIGNING_P12_PASSWORD }}\n          NOTARY_KEY_BASE64: ${{ secrets.APPLE_NOTARY_KEY_P8_BASE64 }}' \
  $'P12_BASE64: ${{ secrets.MACOS_SIGNING_P12_BASE64_RENAMED }}\n          P12_PASSWORD: ${{ secrets.MACOS_SIGNING_P12_PASSWORD }}\n          NOTARY_KEY_BASE64: ${{ secrets.APPLE_NOTARY_KEY_P8_BASE64 }}'
assert_rejected \
  "pkg-notarize-step-gated-off-dispatch" \
  $'      - name: Notarize and staple the .pkg\n        if: ${{ env.SIGN == \'true\' }}' \
  $'      - name: Notarize and staple the .pkg\n        if: ${{ env.SIGN == \'true\' && github.event_name == \'push\' }}'
assert_rejected \
  "pkg-notarize-removed" \
  "xcrun notarytool submit" \
  "xcrun notarytool removed"
assert_rejected \
  "pkg-notarize-verdict-unchecked" \
  '!= "Accepted"' \
  '!= "Anything"'
assert_rejected \
  "pkg-staple-removed" \
  "xcrun stapler staple \"dist/\${PKG}\"" \
  'echo no staple'
assert_rejected \
  "pkg-staple-not-validated" \
  "xcrun stapler validate \"dist/\${PKG}\"" \
  'echo no validate'
assert_rejected \
  "pkg-gatekeeper-assessment-removed" \
  'spctl --assess --type install --verbose=2' \
  'echo no spctl'
assert_rejected \
  "pkg-installer-identity-dropped" \
  'export SEED_INSTALLER_IDENTITY="Developer ID Installer"' \
  'export SEED_INSTALLER_IDENTITY_UNUSED=1'
assert_rejected \
  "pkg-helper-presence-unchecked" \
  "Seed Wi-Fi Helper.app\"" \
  "Seed Wi-Fi Helper.appx\""
assert_rejected \
  "pkg-keychain-not-removed-always" \
  $'      - name: Remove the signing keychain\n        if: ${{ always() && env.SIGN == \'true\' }}' \
  $'      - name: Remove the signing keychain\n        if: ${{ env.SIGN == \'true\' }}'
assert_rejected \
  "pkg-job-removed-from-build" \
  "deploy/macos/build-pkg.sh" \
  "deploy/macos/build-pkg-removed.sh"
assert_rejected \
  "darwin-syft-checksum-skipped" \
  "echo \"\${SYFT_DARWIN_SHA256}  \${RUNNER_TEMP}/syft.tar.gz\" | shasum -a 256 -c -" \
  "echo \"\${SYFT_DARWIN_SHA256}  \${RUNNER_TEMP}/syft.tar.gz\""
assert_rejected \
  "darwin-cosign-checksum-skipped" \
  "echo \"\${COSIGN_DARWIN_SHA256}  \${bin}/cosign\" | shasum -a 256 -c -" \
  "echo \"\${COSIGN_DARWIN_SHA256}  \${bin}/cosign\""
assert_rejected \
  "darwin-cosign-sha-truncated" \
  "$(grep -oE 'COSIGN_DARWIN_SHA256: "[0-9a-f]{64}"' "$source_workflow" | head -1)" \
  'COSIGN_DARWIN_SHA256: "5cf948c2"'
assert_rejected \
  "darwin-syft-version-drifts-from-linux" \
  "$(grep -oE 'SYFT_DARWIN_VERSION: "[^"]+"' "$source_workflow" | head -1)" \
  'SYFT_DARWIN_VERSION: "0.0.1"'
assert_rejected \
  "darwin-cosign-version-drifts-from-linux" \
  "$(grep -oE 'COSIGN_DARWIN_VERSION: "[^"]+"' "$source_workflow" | head -1)" \
  'COSIGN_DARWIN_VERSION: "v0.0.1"'
assert_config_rejected \
  "release-artifacts-drops-pkg" RELEASE_ARTIFACTS_PATH \
  .github/release-artifacts.txt \
  $'seed-{version}-darwin-arm64.pkg\n' \
  ''
assert_rejected \
  "snapshot-no-longer-complements-publish" \
  $'      - name: Run goreleaser (snapshot/dry-run)\n        if: ${{ github.event_name != \'push\' || inputs.dry_run }}' \
  $'      - name: Run goreleaser (snapshot/dry-run)\n        if: inputs.dry_run'
assert_rejected \
  "dispatch-publish-refusal-removed" \
  "if: \${{ github.event_name == 'workflow_dispatch' && !inputs.dry_run }}" \
  "if: false"

# Immutable releases (#3157): the release must stay a draft until every asset
# is on it. Each case drops one hand-off that reproduced the empty release.
assert_rejected \
  "draft-never-published" \
  "run: gh release edit \"\$TARGET_TAG\" --draft=false --repo \"\${GITHUB_REPOSITORY}\"" \
  'run: echo "release left as a draft"'
assert_rejected \
  "publish-before-provenance" \
  "needs: [goreleaser, provenance]" \
  "needs: [goreleaser]"
assert_config_rejected \
  "release-please-publishes" RELEASE_PLEASE_CONFIG_PATH \
  .github/release-please-config.json \
  '      "draft": true,' \
  '      "draft": false,'
# Without a forced tag, a draft release creates no tag and release.yml never runs.
assert_config_rejected \
  "release-please-no-tag" RELEASE_PLEASE_CONFIG_PATH \
  .github/release-please-config.json \
  $'      "force-tag-creation": true,\n' \
  ''
assert_config_rejected \
  "goreleaser-ignores-draft" GORELEASER_CONFIG_PATH \
  .goreleaser.yml \
  '  use_existing_draft: true' \
  '  use_existing_draft: false'
assert_config_rejected \
  "windows-zip-ships-build-script" GORELEASER_CONFIG_PATH \
  .goreleaser.yml \
  'src: deploy/windows/install.ps1' \
  'src: deploy/windows/build.ps1'
assert_config_rejected \
  "goreleaser-publishes" GORELEASER_CONFIG_PATH \
  .goreleaser.yml \
  $'\n  draft: true\n' \
  $'\n  draft: false\n'

assert_rejected \
  "new-mutable-action" \
  $'    steps:\n      - name: Probe for build dependencies' \
  $'    steps:\n      - uses: example/untrusted-action@main\n      - name: Probe for build dependencies'

echo "release workflow contract mutation tests passed."
