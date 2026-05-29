#!/usr/bin/env bash
# check-token-discipline.sh — design-token discipline gate.
#
# Enforces that UI application code uses the semantic theme tokens defined in
# ui/src/index.css instead of raw values. Run by CI on every PR.
#
# TWO TIERS:
#   BLOCKING (fails CI) — COLOR discipline. The codebase is fully migrated, so
#     these must stay at zero:
#       - raw Tailwind palette colors (gray-N, blue-N, amber-N, …)
#       - bare bg/text/border-white|black
#       - text-[var(--color-X)] indirection (use text-X directly)
#       - raw 6/8-digit hex colors in .ts/.tsx
#       - undefined tokens: bg-/text-/… utilities whose --color-* is not in
#         index.css (compiles to nothing, renders no color) — see
#         resolve-token-refs.py
#   ADVISORY (warns, never fails) — spacing / typography / flex shortcuts that
#     have semantic replacements. Pre-existing debt; surfaced for burn-down.
#
# Scope: *.tsx / *.ts under ui/src/, EXCEPT:
#   - test files (.test/.spec/.stories/.mock)
#   - token definition sites (styles/, constants/)
#   - documented fixed-palette exceptions (intentional, not theme colors):
#       components/cards/cableWire.ts          physical T568B wire colors
#       utils/reportRenderer.ts                self-contained light print CSS
#       components/survey/FloorPlanCanvas.tsx  canvas drawing over imagery
#       lib/logger.ts                          devtools console %c colors
#
# Portability: BLOCKING rules use POSIX ERE (GNU + BSD/macOS grep). ADVISORY
# rules use PCRE lookbehind (-P, GNU grep); on BSD grep they report nothing,
# which is fine because they never fail the build.
#
# Run locally: scripts/check-token-discipline.sh   (--report = never fail)

set -uo pipefail

REPORT_MODE=0
[ "${1:-}" = "--report" ] && REPORT_MODE=1

if [ -d "ui/src" ]; then
  TARGET="ui/src"
elif [ -d "src" ] && [ -f "package.json" ]; then
  TARGET="src"
else
  echo "ERROR: cannot find ui/src — run from repo root or ui/ directory" >&2
  exit 2
fi

# Definition sites, tests, and the documented fixed-palette exceptions.
EXCLUDE_RE='\.(test|spec|stories|mock)\.(ts|tsx):|/styles/|/constants/|/cableWire\.ts:|/reportRenderer\.ts:|/FloorPlanCanvas\.tsx:|/HeatmapLegend\.tsx:|/logger\.ts:'

FAIL_COUNT=0

# block LABEL REGEX HINT — POSIX ERE, fails the build on any hit.
block() {
  local label="$1" rx="$2" hint="$3" hits count
  hits=$(grep -rEn --include='*.tsx' --include='*.ts' -- "$rx" "$TARGET" 2>/dev/null \
    | grep -vE "$EXCLUDE_RE" || true)
  [ -z "$hits" ] && return 0
  count=$(printf '%s\n' "$hits" | grep -c .)
  FAIL_COUNT=$((FAIL_COUNT + count))
  echo "============================================================"
  echo "[BLOCK: $label] $count violation(s)"
  echo "  fix: $hint"
  echo "------------------------------------------------------------"
  printf '%s\n' "$hits" | head -10
  local hidden=$((count - 10))
  [ "$hidden" -gt 0 ] && echo "  … and $hidden more"
  echo ""
}

# advise LABEL REGEX HINT — PCRE; warns only, never fails.
advise() {
  local label="$1" rx="$2" hint="$3" hits count
  hits=$(grep -rn --include='*.tsx' --include='*.ts' -P -- "$rx" "$TARGET" 2>/dev/null \
    | grep -vE "$EXCLUDE_RE" || true)
  [ -z "$hits" ] && return 0
  count=$(printf '%s\n' "$hits" | grep -c .)
  echo "[warn: $label] $count occurrence(s) — $hint"
}

# ── BLOCKING: color discipline ──────────────────────────────────────────────
block RAW_PALETTE \
  '-(gray|slate|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-[0-9]+' \
  'Use brand-*/surface-*/text-*/status-*/log-*/cat-*/severity-* tokens'
block BARE_WHITE_BLACK \
  '\b(bg|text|border|from|via|to|ring|fill|stroke|placeholder|shadow|outline|divide|accent|caret|decoration)-(white|black)\b' \
  'Use bg-knob (white) / bg-scrim (black) / text-on-* / text-text-inverse'
block ARBITRARY_COLOR_VAR \
  '(text|bg|border|ring|fill|stroke|placeholder|shadow|from|via|to|outline|divide|accent|caret|decoration)-\[var\(--color-[a-z0-9-]+\)\]' \
  'Drop the var() indirection: text-[var(--color-status-error)] -> text-status-error'
block RAW_HEX_COLOR \
  '#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?\b' \
  'Use a CSS theme variable; for <canvas>/PDF read it via styles/tokens.ts'
block RGB_HSL_COLOR \
  '\b(rgba?|hsla?)\([0-9]' \
  'Use a CSS theme variable; for white/black overlays use color-mix(var(--color-knob|scrim))'

# UNDEFINED_TOKEN — every referenced color token must resolve to a --color-*
# defined in index.css. Catches typos / renamed tokens that compile to nothing
# and silently render no color (invisible to the palette/hex rules above).
if command -v python3 >/dev/null 2>&1; then
  resolve_out=$(python3 "$(dirname "$0")/resolve-token-refs.py" "$TARGET" 2>/dev/null)
  if [ -n "$resolve_out" ]; then
    rc_count=$(printf '%s\n' "$resolve_out" | grep -c 'undefined token:')
    FAIL_COUNT=$((FAIL_COUNT + rc_count))
    echo "============================================================"
    echo "[BLOCK: UNDEFINED_TOKEN] $rc_count token(s) referenced but not defined in index.css"
    echo "  fix: define the --color-* in index.css, or use the correct token name"
    echo "------------------------------------------------------------"
    printf '%s\n' "$resolve_out"
    echo ""
  fi
else
  echo "[warn: UNDEFINED_TOKEN] python3 not found — skipping undefined-token resolve check"
fi

# ── ADVISORY: spacing / typography / flex (warn-only) ───────────────────────
advise RAW_SPACE_Y '(?<![-\w])space-y-(1|2|3|4|6)(?![-\w])' 'Use stack-xs/sm/[default]/lg/xl'
advise RAW_GAP '(?<![-\w])gap-(1|2|3|4|6)(?![-\w])' 'Use gap-tight/compact/default/comfortable/spacious'
advise RAW_P '(?<![-\w])p-(2|3|4|6|8)(?![-\w])' 'Use pad-xs/sm/[default]/lg/xl'
advise RAW_MB '(?<![-\w])mb-(1|3|4|6|8)(?![-\w])' 'Use mb-tight/heading/content/section/section-lg'
advise RAW_MT '(?<![-\w])mt-(1|2|3|4|8)(?![-\w])' 'Use mt-tight/inline/heading/content/section'
advise RAW_ML '(?<![-\w])ml-(1|2|4|6)(?![-\w])' 'Use ml-tight/inline/content/spacious'
advise RAW_PT '(?<![-\w])pt-(1|3|4)(?![-\w])' 'Use pt-tight/heading/section'
advise RAW_PB '(?<![-\w])pb-(1|2)(?![-\w])' 'Use pb-tight/inline'
advise RAW_PX_2 '(?<![-\w])px-2(?![-\w])' 'Use px-cell'
advise RAW_PR '(?<![-\w])pr-(8|10)(?![-\w])' 'Use pr-tight/icon'
advise RAW_PL_5 '(?<![-\w])pl-5(?![-\w])' 'Use pl-indent'
advise RAW_PY_12 '(?<![-\w])py-12(?![-\w])' 'Use py-centered'
advise RAW_HEADING_PAIR \
  '(text-2xl[^"`]*font-bold|font-bold[^"`]*text-2xl|text-xl[^"`]*font-semibold|font-semibold[^"`]*text-xl|text-lg[^"`]*font-semibold|font-semibold[^"`]*text-lg)' \
  'Use heading-1/2/3 (or heading-4 for text-base font-medium)'
advise RAW_FLEX_BETWEEN 'flex items-center justify-between' 'Use flex-between'
advise RAW_FLEX_CENTER 'flex items-center justify-center' 'Use flex-center'

if [ "$FAIL_COUNT" -gt 0 ] && [ "$REPORT_MODE" -eq 0 ]; then
  echo "============================================================"
  echo "FAIL: $FAIL_COUNT blocking color-token violation(s) above."
  exit 1
fi

echo "OK: blocking color-token discipline is clean (advisory warnings, if any, are non-blocking)."
