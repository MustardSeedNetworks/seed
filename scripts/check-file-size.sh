#!/bin/bash
# scripts/check-file-size.sh
# One size limit per file type, enforced. Part of CI quality gates.
#
# A file over its limit fails the build unless scripts/file-size-baseline.txt
# grants it an accepted maximum, and a granted file may not grow past it. That
# is the whole rule: debt is explicit, reviewed, and shrink-only.
#
# There used to be two tiers -- an advisory warning at 600 Go / 400 TS lines
# and a failing "red flag" at 1200 / 800. The warning tier changed nothing
# between January and September 2026: it printed on every run, blocked nothing,
# and trained everyone to scroll past it. A gate that cannot fail is not a
# gate, so the advisory tier is gone and the enforced limit moved to where the
# warning used to be pointing (900 Go / 600 TS) rather than staying at a
# number nobody would ever hit deliberately.

set -euo pipefail

SCAN_ROOT=${SCAN_ROOT:-.}
MAX_GO_LINES=${MAX_GO_LINES:-900}
MAX_TEST_LINES=${MAX_TEST_LINES:-1000}
MAX_TS_LINES=${MAX_TS_LINES:-600}
BASELINE_FILE=${BASELINE_FILE:-scripts/file-size-baseline.txt}
VIOLATIONS=0
BASELINED=0

cd "$SCAN_ROOT"

baseline_limit() {
    local file=$1
    # No baseline file is a valid state -- it means nothing has been granted
    # size debt. Reading it unconditionally made the first violation die in
    # awk instead of being reported.
    [ -f "$BASELINE_FILE" ] || return 0
    awk -v target="$file" '$1 == target { print $2; exit }' "$BASELINE_FILE"
}

check() {
    local file=$1 lines=$2 limit=$3 granted
    [ "$lines" -le "$limit" ] && return 0
    granted=$(baseline_limit "$file")
    if [ -n "$granted" ] && [ "$lines" -le "$granted" ]; then
        echo "📎 $file ($lines lines, baselined at $granted — may not grow)"
        BASELINED=$((BASELINED + 1))
        return 0
    fi
    if [ -n "$granted" ]; then
        echo "❌ $file ($lines lines) grew past its baselined maximum of $granted"
    else
        echo "❌ $file ($lines lines, limit ${limit})"
    fi
    VIOLATIONS=$((VIOLATIONS + 1))
}

echo "Checking file sizes:"
echo "  Go source: max ${MAX_GO_LINES}"
echo "  Go tests:  max ${MAX_TEST_LINES}"
echo "  TS/TSX:    max ${MAX_TS_LINES}"
echo "  Baseline:  ${BASELINE_FILE} (shrink-only)"
echo "==========================================================================="

while IFS= read -r -d '' file; do
    file=${file#./}
    check "$file" "$(wc -l < "$file" | tr -d ' ')" "$MAX_GO_LINES"
done < <(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -print0 2>/dev/null || true)

while IFS= read -r -d '' file; do
    file=${file#./}
    check "$file" "$(wc -l < "$file" | tr -d ' ')" "$MAX_TEST_LINES"
done < <(find . -name "*_test.go" -not -path "./vendor/*" -print0 2>/dev/null || true)

if [ -d "ui/src" ]; then
    while IFS= read -r -d '' file; do
        file=${file#./}
        check "$file" "$(wc -l < "$file" | tr -d ' ')" "$MAX_TS_LINES"
    done < <(find ui/src \( -name "*.ts" -o -name "*.tsx" \) -print0 2>/dev/null || true)
fi

echo "==========================================================================="

if [ "$VIOLATIONS" -gt 0 ]; then
    echo "❌ $VIOLATIONS file(s) failed the size gate"
    echo "Split the file, or record its accepted maximum in ${BASELINE_FILE}."
    echo "A file already listed there may shrink but never grow."
    exit 1
fi

if [ "$BASELINED" -gt 0 ]; then
    echo "✅ Within limits — ${BASELINED} baselined file(s), none grown"
else
    echo "✅ All files within size limits"
fi
