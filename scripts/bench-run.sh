#!/usr/bin/env bash
# Run the benchmark suite over whatever tree this script is pointed at.
#
# Package discovery is done here rather than pinned in the Makefile so the
# script works against an arbitrary checkout. The CI gate benchmarks the merge
# base as well as the head, and the merge base generally predates whatever the
# head added — a hard-coded package list, or a `make` target, would simply not
# exist there. Copy this file somewhere outside the worktree, check out the
# base, and run it: it discovers that tree's own benchmarks.
set -euo pipefail

# 200ms rather than the 1s default: the full six-count sweep drops from about
# five minutes to one, and the metric the gate reads (allocs/op) does not care
# how long each iteration ran. -count=6 gives benchstat enough samples to say
# something about spread.
BENCHTIME="${BENCHTIME:-200ms}"
COUNT="${COUNT:-6}"

# Directories holding at least one benchmark. Listing them beats ./..., which
# still builds and runs a test binary for every package and spends minutes
# producing nothing.
# git ls-files, not a recursive grep of the working directory. CI sets
# GOMODCACHE to $GITHUB_WORKSPACE/.cache/go/pkg/mod — inside the checkout — so a
# `grep -r .` there walks the entire module cache and finds every dependency's
# benchmarks: 486 packages instead of 4, and then `go test` on packages outside
# this module fails with "updates to go.mod needed". Tracked files only.
#
# Deliberately not mapfile: macOS ships bash 3.2, and a script the CI gate
# depends on has to be runnable on the machine where it is being changed.
pkgs=()
while IFS= read -r dir; do
  [ -n "$dir" ] && pkgs+=("./$dir")
done < <(
  git ls-files '*_test.go' \
    | xargs grep -l '^func Benchmark' 2>/dev/null \
    | sed 's|/[^/]*$||' \
    | sort -u
)

if [ ${#pkgs[@]} -eq 0 ]; then
  echo "no benchmarks found in $(pwd)" >&2
  exit 1
fi

echo "benchmarking ${#pkgs[@]} packages from $(pwd)" >&2
exec go test -bench=. -benchmem -run='^$' \
  -benchtime="$BENCHTIME" -count="$COUNT" "${pkgs[@]}"
