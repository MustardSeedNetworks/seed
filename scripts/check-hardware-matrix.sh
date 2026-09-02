#!/bin/sh
# Fails when HARDWARE.md's Platform Support Matrix disagrees with
# internal/capabilities.
#
# The matrix was hand-written and drifted in both directions — macOS ARP
# reading published as Full while it returned nothing, macOS Wi-Fi scanning
# published as Full while it shelled a binary Apple had removed. Neither could
# fail a build. This is that build failure.
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_dir"

if go run ./cmd/seed-hardware -file HARDWARE.md -check; then
  printf 'HARDWARE.md matrix is up to date.\n'
  exit 0
fi

printf '\nRun `make hardware-matrix` and commit the result.\n' >&2
exit 1
