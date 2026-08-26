#!/bin/bash
# scripts/check-package-reachability.sh
# Fails when an internal package is reachable from no binary.
#
# staticcheck's unused checks (U1000 and friends) cannot answer this. They find
# unused identifiers *within* a package that is being compiled; a package
# nobody imports still looks fully used from inside its own tests. That gap is
# not theoretical — stem carried 2,800 lines across two packages that no binary
# reached, past a weekly dead-code job that reported success every time.
#
# `go list -deps ./cmd/...` answers it directly and in about a second.
#
# Packages already unreachable when this gate was adopted are recorded in
# scripts/package-reachability-baseline.txt with a note. They are debt, not
# permission: the list may shrink and must not grow.

set -euo pipefail

BASELINE_FILE=${BASELINE_FILE:-scripts/package-reachability-baseline.txt}
MODULE=$(go list -m)

# Packages reached from any binary's import graph, on every platform seed
# ships to. A single-platform check would report a package wired only under
# //go:build linux as dead when run on a developer's Mac, which is the fastest
# way to make a gate untrustworthy.
reachable=$(
    for goos in linux darwin windows; do
        GOOS=$goos go list -deps ./cmd/... 2>/dev/null
    done | sort -u
)

# Every internal package in the module.
all_internal=$(go list ./... 2>/dev/null | grep "/internal/" | sort -u)

baselined() {
    local pkg=$1
    # Comment and blank lines are ignored; the package path is the first field.
    awk -v target="$pkg" '
        /^[[:space:]]*(#|$)/ { next }
        $1 == target { found = 1; exit }
        END { exit !found }
    ' "$BASELINE_FILE"
}

new_unreachable=()
still_unreachable=0

while read -r pkg; do
    [ -z "$pkg" ] && continue
    if grep -qxF "$pkg" <<<"$reachable"; then
        continue
    fi
    short=${pkg#"$MODULE/"}
    if baselined "$short"; then
        still_unreachable=$((still_unreachable + 1))
        continue
    fi
    new_unreachable+=("$short")
done <<<"$all_internal"

if [ ${#new_unreachable[@]} -eq 0 ]; then
    echo "✓ no new unreachable packages ($still_unreachable baselined)"
    exit 0
fi

echo "::error::${#new_unreachable[@]} internal package(s) are reachable from no binary"
echo
for pkg in "${new_unreachable[@]}"; do
    echo "  $pkg"
done
echo
echo "Nothing in ./cmd/... can reach these, so they are not in any shipped"
echo "binary. Either wire them up, delete them, or — if a package is"
echo "deliberately test-only or platform-gated — add it to $BASELINE_FILE"
echo "with a line saying which."
echo
echo "Before deleting: check for consumers Go tooling cannot see. A frontend"
echo "importing a directory inside a Go package is invisible to go build, and"
echo "stem's locales sat inside a dead package for exactly that reason."
exit 1
