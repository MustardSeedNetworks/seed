#!/usr/bin/env bash
# sync-shell.sh — pull canonical UI shell files from stem.
#
# Stem owns: ui/src/ui/Sidebar.tsx, ui/src/ui/PageHeader.tsx
# (see stem/ui/SHELL.md for the contract).
#
# This script:
#   1. Copies canonical files from ../stem/ui/src/ui/ into this repo
#   2. Prepends a banner with the source commit SHA
#   3. Runs Biome format
#   4. Writes a checksums lock to ui/src/ui/.shell-sync.lock
#
# Run from repo root:        scripts/sync-shell.sh
# Override stem path:        STEM_DIR=../path/to/stem scripts/sync-shell.sh
# Verify lock matches files: scripts/sync-shell.sh --verify

set -euo pipefail

# Resolve to repo root so paths work whether invoked from repo root or ui/.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

STEM_DIR="${STEM_DIR:-../stem}"
SHELL_FILES=(
  "ui/src/ui/Sidebar.tsx"
  "ui/src/ui/PageHeader.tsx"
)
LOCKFILE="ui/src/ui/.shell-sync.lock"
MODE="${1:-sync}"

if [ "$MODE" = "--verify" ]; then
  if [ ! -f "$LOCKFILE" ]; then
    echo "ERROR: $LOCKFILE not found. Run scripts/sync-shell.sh first."
    exit 1
  fi
  # Verify: re-compute current checksums and diff against lock
  CURRENT=$(mktemp)
  trap 'rm -f "$CURRENT"' EXIT
  for f in "${SHELL_FILES[@]}"; do
    shasum -a 256 "$f" >> "$CURRENT"
  done
  # Strip the SOURCE_SHA line from the lock for the comparison
  LOCK_BODY=$(mktemp)
  trap 'rm -f "$CURRENT" "$LOCK_BODY"' EXIT
  grep -v '^# SOURCE_SHA:' "$LOCKFILE" > "$LOCK_BODY"
  if ! diff -u "$LOCK_BODY" "$CURRENT" >/dev/null; then
    echo "ERROR: canonical shell files differ from $LOCKFILE."
    echo "Either re-sync from upstream stem (scripts/sync-shell.sh) or"
    echo "push your changes upstream and re-sync."
    diff -u "$LOCK_BODY" "$CURRENT"
    exit 1
  fi
  echo "OK: shell files match $LOCKFILE."
  exit 0
fi

if [ ! -d "$STEM_DIR" ]; then
  echo "ERROR: stem source directory not found at $STEM_DIR"
  echo "Set STEM_DIR or clone stem alongside this repo:"
  echo "  git -C .. clone git@github.com:krisarmstrong/stem.git"
  exit 1
fi

# Capture the source SHA so the banner records exactly which stem commit was synced.
SOURCE_SHA=$(git -C "$STEM_DIR" rev-parse HEAD)
SOURCE_BRANCH=$(git -C "$STEM_DIR" rev-parse --abbrev-ref HEAD)
SOURCE_REPO=$(git -C "$STEM_DIR" config --get remote.origin.url || echo "(local stem checkout)")

echo "Syncing shell files from stem"
echo "  source: $SOURCE_REPO"
echo "  ref:    $SOURCE_BRANCH @ ${SOURCE_SHA:0:12}"
echo ""

# Write the lock header
{
  echo "# SOURCE_SHA: $SOURCE_SHA  ($SOURCE_BRANCH)"
} > "$LOCKFILE"

for f in "${SHELL_FILES[@]}"; do
  if [ ! -f "$STEM_DIR/$f" ]; then
    echo "ERROR: $STEM_DIR/$f not found in stem"
    exit 1
  fi
  echo "  syncing $f"
  mkdir -p "$(dirname "$f")"
  # Prepend the synced-from banner, then the source file body.
  {
    echo "// SYNCED FROM stem@${SOURCE_SHA:0:12} — DO NOT EDIT."
    echo "// Edits in this repo will be overwritten on next \`make sync-shell\`."
    echo "// To change this file, send a PR to stem then re-sync."
    cat "$STEM_DIR/$f"
  } > "$f"
done

# Run Biome format on the synced files so they conform to local code style.
if [ -d "ui/node_modules" ]; then
  echo ""
  echo "Running Biome format on synced files..."
  (cd ui && npx @biomejs/biome format --write src/ui/Sidebar.tsx src/ui/PageHeader.tsx 2>/dev/null) || true
fi

# Record post-format checksums
for f in "${SHELL_FILES[@]}"; do
  shasum -a 256 "$f" >> "$LOCKFILE"
done

echo ""
echo "OK: synced ${#SHELL_FILES[@]} file(s). Lockfile: $LOCKFILE"
echo ""
echo "Next: review the diff and commit. CI will run \`sync-shell.sh --verify\` on every PR."
