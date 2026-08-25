#!/usr/bin/env bash
# Template for each product repo's scripts/i18n/validate.sh.
#
# The gate itself lives once in MustardSeedNetworks/.github. This shim is what
# keeps `./scripts/i18n/validate.sh` working from a repo checkout, because a
# CI-only gate is a slower loop and the pre-push run is how these get caught
# early.
#
# The pin is read from this repo's own ci.yml `uses:` line, so there is exactly
# one pinned SHA per repo and the shim cannot drift from what CI runs. Renovate
# bumps that line; this follows automatically.
#
# Offline: once a SHA is cached the shim never reaches the network again.

set -euo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
cd "$REPO_ROOT"

SHARED_REPO="${MSN_SHARED_REPO:-https://github.com/MustardSeedNetworks/.github.git}"
WORKFLOW=".github/workflows/ci.yml"

# The one pin: whatever SHA CI calls the shared action at.
PIN="$(grep -oE 'MustardSeedNetworks/\.github/\.github/actions/i18n-validate@[0-9a-f]{40}' "$WORKFLOW" \
        | head -1 | cut -d@ -f2 || true)"
if [ -z "$PIN" ]; then
  echo "✘ no pinned i18n-validate SHA found in $WORKFLOW" >&2
  echo "  CI and this script share that pin; without it they could disagree." >&2
  exit 1
fi

CACHE="${MSN_SHARED_CACHE:-$HOME/.cache/msn-shared}/$PIN"
if [ ! -d "$CACHE/scripts/i18n" ]; then
  echo "→ fetching shared i18n gate @ ${PIN:0:12} (first run for this pin)" >&2
  rm -rf "$CACHE"
  mkdir -p "$CACHE"
  git -C "$CACHE" init -q
  git -C "$CACHE" remote add origin "$SHARED_REPO"
  git -C "$CACHE" fetch -q --depth 1 origin "$PIN"
  git -C "$CACHE" checkout -q FETCH_HEAD
fi

exec bash "$CACHE/scripts/i18n/validate.sh" "$@"
