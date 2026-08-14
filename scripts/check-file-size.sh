#!/bin/bash
# scripts/check-file-size.sh
# Checks Go source files don't exceed maximum line count
# Part of CI quality gates
#
# Files above the red-flag threshold fail unless their current debt is listed
# in scripts/file-size-baseline.txt. A baselined file may not grow.

set -euo pipefail

SCAN_ROOT=${SCAN_ROOT:-.}
MAX_GO_LINES=${MAX_GO_LINES:-600}
MAX_TEST_LINES=${MAX_TEST_LINES:-1000}
MAX_TS_LINES=${MAX_TS_LINES:-400}
RED_FLAG_GO=${RED_FLAG_GO:-1200}
RED_FLAG_TS=${RED_FLAG_TS:-800}
BASELINE_FILE=${BASELINE_FILE:-scripts/file-size-baseline.txt}
VIOLATIONS=0
WARNINGS=0

cd "$SCAN_ROOT"

baseline_limit() {
    local file=$1
    awk -v target="$file" '$1 == target { print $2; exit }' "$BASELINE_FILE"
}

record_red_flag() {
    local file=$1
    local lines=$2
    local threshold=$3
    local limit
    limit=$(baseline_limit "$file")
    if [ -n "$limit" ] && [ "$lines" -le "$limit" ]; then
        echo "⚠️  $file ($lines lines, baselined maximum: $limit)"
        WARNINGS=$((WARNINGS + 1))
        return
    fi
    echo "❌ $file ($lines lines, red flag: >${threshold})"
    VIOLATIONS=$((VIOLATIONS + 1))
}

echo "Checking file sizes:"
echo "  Go source: max ${MAX_GO_LINES}, red flag >${RED_FLAG_GO}"
echo "  Go tests:  max ${MAX_TEST_LINES}"
echo "  TS/TSX:    max ${MAX_TS_LINES}, red flag >${RED_FLAG_TS}"
echo "  Baseline:  ${BASELINE_FILE}"
echo "==========================================================================="

# Check Go non-test files
while IFS= read -r -d '' file; do
    file=${file#./}
    lines=$(wc -l < "$file" | tr -d ' ')
    if [ "$lines" -gt "$RED_FLAG_GO" ]; then
        record_red_flag "$file" "$lines" "$RED_FLAG_GO"
    elif [ "$lines" -gt "$MAX_GO_LINES" ]; then
        echo "⚠️  $file ($lines lines, max: $MAX_GO_LINES)"
        WARNINGS=$((WARNINGS + 1))
    fi
done < <(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -print0 2>/dev/null || true)

# Check Go test files (allow more lines)
while IFS= read -r -d '' file; do
    file=${file#./}
    lines=$(wc -l < "$file" | tr -d ' ')
    if [ "$lines" -gt "$MAX_TEST_LINES" ]; then
        echo "⚠️  $file ($lines lines, max: $MAX_TEST_LINES)"
        WARNINGS=$((WARNINGS + 1))
    fi
done < <(find . -name "*_test.go" -not -path "./vendor/*" -print0 2>/dev/null || true)

# Check TS/TSX files
if [ -d "ui/src" ]; then
    while IFS= read -r -d '' file; do
        file=${file#./}
        lines=$(wc -l < "$file" | tr -d ' ')
        if [ "$lines" -gt "$RED_FLAG_TS" ]; then
            record_red_flag "$file" "$lines" "$RED_FLAG_TS"
        elif [ "$lines" -gt "$MAX_TS_LINES" ]; then
            echo "⚠️  $file ($lines lines, max: $MAX_TS_LINES)"
            WARNINGS=$((WARNINGS + 1))
        fi
    done < <(find ui/src -name "*.ts" -o -name "*.tsx" | tr '\n' '\0' 2>/dev/null || true)
fi

echo "==========================================================================="

if [ "$VIOLATIONS" -eq 0 ] && [ "$WARNINGS" -eq 0 ]; then
    echo "✅ All files within size limits"
    exit 0
fi

echo "📊 Found $VIOLATIONS red flag(s), $WARNINGS warning(s)"

if [ "$VIOLATIONS" -gt 0 ]; then
    echo "❌ Failing due to new or increased red-flag files"
    echo "Split the file or update the reviewed baseline with its accepted maximum."
    exit 1
fi

echo "ℹ️  Warnings are existing size debt below the enforced red-flag boundary"
