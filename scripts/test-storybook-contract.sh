#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root/ui"

if ! npm run | rg -q 'test:storybook:run'; then
  echo 'Storybook runner is missing' >&2
  exit 1
fi

verify_defect() {
  local defect=$1
  local expected=$2
  local log_file
  log_file=$(mktemp)

  if VITE_STORYBOOK_INJECT_DEFECT="$defect" npm run test:storybook:run >"$log_file" 2>&1; then
    echo "Injected $defect defect did not fail Storybook tests" >&2
    rm -f "$log_file"
    return 1
  fi

  if ! rg -qi "$expected" "$log_file"; then
    echo "Injected $defect defect failed for the wrong reason" >&2
    cat "$log_file" >&2
    rm -f "$log_file"
    return 1
  fi

  rm -f "$log_file"
}

verify_defect interaction 'SharedComponentInteraction|Shared Component Interaction'
verify_defect accessibility 'button-name|discernible text'
