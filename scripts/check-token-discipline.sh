#!/usr/bin/env bash
# check-token-discipline.sh — full design-token discipline gate (Phase 0).
#
# Enforces that UI application code uses semantic theme tokens instead of
# raw Tailwind utilities. Run by CI on every PR.
#
# Scope: *.tsx and *.ts files under ui/src/ EXCEPT styles/, constants/, and
# test files (.test.tsx, .spec.tsx, .stories.tsx, .mock.tsx).
#
# What's banned:
#   - Raw Tailwind palette colors (gray-N, slate-N, red-N, blue-N, etc.)
#   - Bare bg-white / bg-black / text-white / text-black / etc.
#   - Raw spacing utilities that have semantic replacements
#       (space-y-N, gap-N (numeric), p-N (standalone), mb-N, mt-N, etc.)
#   - text-[var(--color-X)] arbitrary-value indirection — use text-X directly
#
# What's allowed:
#   - Tokens defined in ui/src/index.css (@theme + @layer components)
#   - Tokens defined in ui/src/styles/ (theme.ts, themeSpacing.ts, etc.)
#   - Raw Tailwind for layout/sizing that has no semantic replacement
#       (w-full, h-screen, fixed, relative, absolute, flex, grid, etc.)
#   - Raw text-N / font-N (these ARE the canonical tokens per
#       typography.size.* / typography.weight.* in stem/seed/niac)
#
# Run locally: scripts/check-token-discipline.sh
# To see what would be flagged, pass --report (continues past first failure).

set -uo pipefail

REPORT_MODE=0
if [ "${1:-}" = "--report" ]; then
  REPORT_MODE=1
fi

if [ -d "ui/src" ]; then
  TARGET="ui/src"
elif [ -d "src" ] && [ -f "package.json" ]; then
  TARGET="src"
else
  echo "ERROR: cannot find ui/src — run from repo root or ui/ directory" >&2
  exit 2
fi

# Files we never check (definition sites, tests).
EXCLUDE_RE='\.(test|spec|stories|mock)\.(ts|tsx):|/styles/|/constants/'

# Pattern groups. Each group has a label + regex + remediation hint.
declare -a GROUPS=(
  # Colors — never use raw palette in app code.
  "RAW_PALETTE|(-(gray|slate|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-[0-9]+)|Use brand-*/surface-*/text-*/status-*/log-* tokens instead"
  "BARE_WHITE_BLACK|(\\b(bg|text|border|from|via|to|ring|fill|stroke|placeholder|shadow|outline|divide|accent|caret|decoration)-(white|black)\\b)|Use bg-knob (constant white) / bg-scrim/N (constant black) / text-text-inverse / etc."

  # Arbitrary-value color indirection — text-[var(--color-status-error)] is the same as text-status-error.
  "ARBITRARY_COLOR_VAR|(text|bg|border|ring|fill|stroke|placeholder|shadow|from|via|to|outline|divide|accent|caret|decoration)-\\[var\\(--color-[a-z0-9-]+\\)\\]|Drop the var() indirection: text-[var(--color-status-error)] → text-status-error"

  # Spacing — raw must be replaced by semantic. Word boundaries prevent matching px-/py-/pt- etc.
  "RAW_SPACE_Y|(?<![-\\w])space-y-(1|2|3|4|6)(?![-\\w])|Use stack-xs/sm/[default]/lg/xl"
  "RAW_GAP|(?<![-\\w])gap-(1|2|3|4|6)(?![-\\w])|Use gap-tight/compact/default/comfortable/spacious"
  "RAW_P|(?<![-\\w])p-(2|3|4|6|8)(?![-\\w])|Use pad-xs/sm/[default]/lg/xl"
  "RAW_MB|(?<![-\\w])mb-(1|3|4|6|8)(?![-\\w])|Use mb-tight/heading/content/section/section-lg"
  "RAW_MT|(?<![-\\w])mt-(1|2|3|4|8)(?![-\\w])|Use mt-tight/inline/heading/content/section"
  "RAW_ML|(?<![-\\w])ml-(1|2|4|6)(?![-\\w])|Use ml-tight/inline/content/spacious"
  "RAW_PT|(?<![-\\w])pt-(1|3|4)(?![-\\w])|Use pt-tight/heading/section"
  "RAW_PB|(?<![-\\w])pb-(1|2)(?![-\\w])|Use pb-tight/inline"
  "RAW_PX_2|(?<![-\\w])px-2(?![-\\w])|Use px-cell"
  "RAW_PR|(?<![-\\w])pr-(8|10)(?![-\\w])|Use pr-tight/icon"
  "RAW_PL_5|(?<![-\\w])pl-5(?![-\\w])|Use pl-indent"
  "RAW_PY_12|(?<![-\\w])py-12(?![-\\w])|Use py-centered"

  # Typography compounds — paired text+font should use heading-N or body-N etc.
  "RAW_HEADING_PAIR|(text-2xl[^\"\\\`]*font-bold|font-bold[^\"\\\`]*text-2xl|text-xl[^\"\\\`]*font-semibold|font-semibold[^\"\\\`]*text-xl|text-lg[^\"\\\`]*font-semibold|font-semibold[^\"\\\`]*text-lg)|Use heading-1/2/3 (or heading-4 for text-base font-medium)"

  # Flex shortcuts
  "RAW_FLEX_BETWEEN|flex items-center justify-between|Use flex-between"
  "RAW_FLEX_CENTER|flex items-center justify-center|Use flex-center"
)

TOTAL_VIOLATIONS=0
FAILED_GROUPS=()

for GROUP in "${GROUPS[@]}"; do
  LABEL="${GROUP%%|*}"
  REST="${GROUP#*|}"
  REGEX="${REST%%|*}"
  HINT="${REST#*|}"

  HITS=$(grep -rEn --include='*.tsx' --include='*.ts' -P -- "$REGEX" "$TARGET" 2>/dev/null \
    | grep -vE "$EXCLUDE_RE" || true)

  if [ -n "$HITS" ]; then
    COUNT=$(echo "$HITS" | wc -l | tr -d ' ')
    TOTAL_VIOLATIONS=$((TOTAL_VIOLATIONS + COUNT))
    FAILED_GROUPS+=("$LABEL ($COUNT)")
    echo "============================================================"
    echo "[$LABEL] $COUNT violation(s)"
    echo "Hint: $HINT"
    echo "------------------------------------------------------------"
    echo "$HITS" | head -10
    HIDDEN=$((COUNT - 10))
    if [ "$HIDDEN" -gt 0 ]; then
      echo "... and $HIDDEN more"
    fi
    echo ""
    if [ "$REPORT_MODE" -eq 0 ]; then
      echo "FAIL: token discipline violated (see above). Run scripts/migrate-tokens.py to auto-fix many cases."
      exit 1
    fi
  fi
done

if [ "$TOTAL_VIOLATIONS" -gt 0 ]; then
  echo "============================================================"
  echo "TOTAL: $TOTAL_VIOLATIONS violation(s) across groups: ${FAILED_GROUPS[*]}"
  exit 1
fi

echo "OK: ui/src is token-clean across all categories."
