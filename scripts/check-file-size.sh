#!/bin/bash
# scripts/check-file-size.sh
# Checks Go source files don't exceed maximum line count
# Part of CI quality gates
#
# Modes:
#   STRICT=1 - Fail on violations (default: disabled for existing codebases)
#   STRICT=0 - Warn only, don't fail CI (default)
#
# Usage:
#   ./scripts/check-file-size.sh           # Warn mode
#   STRICT=1 ./scripts/check-file-size.sh  # Strict mode (fail CI)

set -euo pipefail

MAX_GO_LINES=${MAX_GO_LINES:-600}
MAX_TEST_LINES=${MAX_TEST_LINES:-1000}
MAX_TS_LINES=${MAX_TS_LINES:-400}
RED_FLAG_GO=${RED_FLAG_GO:-1200}
RED_FLAG_TS=${RED_FLAG_TS:-800}
STRICT=${STRICT:-0}
VIOLATIONS=0
WARNINGS=0

echo "Checking file sizes:"
echo "  Go source: max ${MAX_GO_LINES}, red flag >${RED_FLAG_GO}"
echo "  Go tests:  max ${MAX_TEST_LINES}"
echo "  TS/TSX:    max ${MAX_TS_LINES}, red flag >${RED_FLAG_TS}"
echo "Mode: $([ "$STRICT" = "1" ] && echo "STRICT (will fail CI)" || echo "WARN ONLY")"
echo "==========================================================================="

# Check Go non-test files
while IFS= read -r -d '' file; do
    lines=$(wc -l < "$file" | tr -d ' ')
    if [ "$lines" -gt "$RED_FLAG_GO" ]; then
        echo "❌ $file ($lines lines, RED FLAG: >${RED_FLAG_GO})"
        VIOLATIONS=$((VIOLATIONS + 1))
    elif [ "$lines" -gt "$MAX_GO_LINES" ]; then
        echo "⚠️  $file ($lines lines, max: $MAX_GO_LINES)"
        WARNINGS=$((WARNINGS + 1))
    fi
done < <(find . -name "*.go" -not -name "*_test.go" -not -path "./vendor/*" -print0 2>/dev/null || true)

# Check Go test files (allow more lines)
while IFS= read -r -d '' file; do
    lines=$(wc -l < "$file" | tr -d ' ')
    if [ "$lines" -gt "$MAX_TEST_LINES" ]; then
        echo "⚠️  $file ($lines lines, max: $MAX_TEST_LINES)"
        WARNINGS=$((WARNINGS + 1))
    fi
done < <(find . -name "*_test.go" -not -path "./vendor/*" -print0 2>/dev/null || true)

# Check TS/TSX files
if [ -d "ui/src" ]; then
    while IFS= read -r -d '' file; do
        lines=$(wc -l < "$file" | tr -d ' ')
        if [ "$lines" -gt "$RED_FLAG_TS" ]; then
            echo "❌ $file ($lines lines, RED FLAG: >${RED_FLAG_TS})"
            VIOLATIONS=$((VIOLATIONS + 1))
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

if [ "$VIOLATIONS" -gt 0 ] && [ "$STRICT" = "1" ]; then
    echo "❌ STRICT mode: Failing CI due to red flag file size violations"
    echo ""
    echo "To fix: Split large files into smaller, focused modules"
    echo "To temporarily bypass: Set STRICT=0 in CI workflow"
    exit 1
elif [ "$VIOLATIONS" -gt 0 ]; then
    echo "⚠️  Red flag violations found but not failing CI (STRICT=0)"
    echo "To enable strict enforcement: Set STRICT=1 in CI workflow"
    exit 0
else
    echo "ℹ️  Warnings only - files exceed ideal limits but are under red flag threshold"
    exit 0
fi
