#!/usr/bin/env bash
# run-storybook-gated.sh — run the Storybook interaction/a11y suite, then hold
# its console output to the contract in check-storybook-console.sh.
#
# This lives in a script rather than inline in package.json because npm runs
# scripts with `sh`, which is dash on Ubuntu and has no `set -o pipefail`. The
# inline version worked on macOS, where /bin/sh is bash, and failed in CI with
# `sh: 1: set: Illegal option -o pipefail`.
#
# Arguments are passed through to vitest, so a developer can narrow the run.

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
log="${STORYBOOK_RUN_LOG:-$(mktemp -t storybook-run.XXXXXX)}"

cd "$repo_root/ui"

# The runner's own failure takes precedence: a suite that fell over says more
# than whatever it managed to log before doing so.
status=0
npm run test:storybook:run -- "$@" 2>&1 | tee "$log" || status=$?
if [ "$status" -ne 0 ]; then
  exit "$status"
fi

"$repo_root/scripts/check-storybook-console.sh" "$log"
