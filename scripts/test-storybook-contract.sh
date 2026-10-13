#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root/ui"

if [ -z "$(node -p 'require("./package.json").scripts["test:storybook:run"] ?? ""')" ]; then
  echo 'Storybook runner is missing' >&2
  exit 1
fi

# The injected defects live only in this story file, so only it runs. Running the
# whole suite here proved nothing more and ran it twice more per job, where three
# of the four recorded Storybook hangs happened (#3283).
gate_stories=src/test/storybook/StorybookGate.stories.tsx

verify_defect() {
  local defect=$1
  local expected=$2
  local log_file
  log_file=$(mktemp)

  # Streamed as well as kept: a run that hangs is killed by the job timeout
  # before anything could print the file.
  if VITE_STORYBOOK_INJECT_DEFECT="$defect" npm run test:storybook:run -- "$gate_stories" 2>&1 | tee "$log_file"; then
    echo "Injected $defect defect did not fail Storybook tests" >&2
    rm -f "$log_file"
    return 1
  fi

  if ! grep -Eqi "$expected" "$log_file"; then
    echo "Injected $defect defect failed for the wrong reason" >&2
    rm -f "$log_file"
    return 1
  fi

  rm -f "$log_file"
}

verify_defect interaction 'SharedComponentInteraction|Shared Component Interaction'
verify_defect accessibility 'button-name|discernible text'
