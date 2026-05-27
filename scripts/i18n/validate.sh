#!/usr/bin/env bash
# i18n-validate.sh — enforce msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md.
#
# Run from repo root. Exits 0 if all checks pass, 1 on any failure.
# Designed to be identical across seed / stem / niac for cross-repo
# consistency. Path overrides via env vars; defaults assume the standard
# Go-embedded layout.
#
# Required tools: bash 4+, jq, grep (GNU or BSD), find.
#
# Usage:
#   ./scripts/i18n-validate.sh
#   ./scripts/i18n-validate.sh --quick    # skip slow checks (dead-key)
#   ./scripts/i18n-validate.sh --check key-parity   # run single check
#
# Per-check exit hint format:
#   ::error file=path,line=N::message    (GitHub Actions annotation)

set -u

# -----------------------------------------------------------------------------
# Config (override via env)
# -----------------------------------------------------------------------------
LOCALES_DIR="${LOCALES_DIR:-internal/i18n/locales}"
UI_SRC_DIR="${UI_SRC_DIR:-ui/src}"
GLOSSARY_FILE="${GLOSSARY_FILE:-scripts/i18n/glossary.txt}"
BANNED_FILE="${BANNED_FILE:-scripts/i18n/banned-vocab.txt}"
# Per-key allow-list for glossary-term false positives (e.g., key uses
# "Pro" in the sense of "professional", not the license tier brand).
# One `namespace.key.path` per line, comments with #.
GLOSSARY_EXCEPTIONS="${GLOSSARY_EXCEPTIONS:-scripts/i18n/glossary-exceptions.txt}"
PRIMARY="en"
SECONDARY_LOCALES=(es)

# -----------------------------------------------------------------------------
# Output helpers
# -----------------------------------------------------------------------------
if [ -t 1 ]; then
  R='\033[0;31m'; G='\033[0;32m'; Y='\033[0;33m'; B='\033[0;34m'; N='\033[0m'
else
  R=''; G=''; Y=''; B=''; N=''
fi

FAILED=0
WARNED=0

fail() {
  printf "${R}✘ %s${N}\n" "$1" >&2
  FAILED=$((FAILED + 1))
}

warn() {
  printf "${Y}⚠ %s${N}\n" "$1" >&2
  WARNED=$((WARNED + 1))
}

ok() {
  printf "${G}✓ %s${N}\n" "$1"
}

section() {
  printf "\n${B}=== %s ===${N}\n" "$1"
}

# Annotation for GitHub Actions
annotate() {
  local file="$1" msg="$2"
  printf "::error file=%s::%s\n" "$file" "$msg"
}

# -----------------------------------------------------------------------------
# Preflight
# -----------------------------------------------------------------------------
if [ ! -d "$LOCALES_DIR" ]; then
  fail "LOCALES_DIR=$LOCALES_DIR does not exist (run from repo root or set LOCALES_DIR)"
  exit 1
fi
if ! command -v jq >/dev/null 2>&1; then
  fail "jq is required but not installed"
  exit 1
fi
if [ ! -d "$LOCALES_DIR/$PRIMARY" ]; then
  fail "Primary locale dir missing: $LOCALES_DIR/$PRIMARY"
  exit 1
fi

# -----------------------------------------------------------------------------
# Check: key parity
# -----------------------------------------------------------------------------
check_key_parity() {
  section "Key parity ($PRIMARY vs each secondary)"
  local issues=0
  for file in "$LOCALES_DIR/$PRIMARY"/*.json; do
    [ -f "$file" ] || continue
    local ns
    ns=$(basename "$file")
    for loc in "${SECONDARY_LOCALES[@]}"; do
      local other="$LOCALES_DIR/$loc/$ns"
      if [ ! -f "$other" ]; then
        fail "$loc is missing namespace $ns"
        annotate "$LOCALES_DIR/$loc/$ns" "missing namespace file"
        issues=$((issues + 1))
        continue
      fi
      local en_keys other_keys
      en_keys=$(jq -r 'paths(scalars) | join(".")' "$file" 2>/dev/null | sort -u)
      other_keys=$(jq -r 'paths(scalars) | join(".")' "$other" 2>/dev/null | sort -u)

      local missing extra
      missing=$(comm -23 <(echo "$en_keys") <(echo "$other_keys"))
      extra=$(comm -13 <(echo "$en_keys") <(echo "$other_keys"))

      if [ -n "$missing" ]; then
        fail "$loc/$ns missing keys present in $PRIMARY:"
        echo "$missing" | sed 's/^/      /'
        annotate "$LOCALES_DIR/$loc/$ns" "missing keys: $(echo "$missing" | tr '\n' ',' | sed 's/,$//')"
        issues=$((issues + 1))
      fi
      if [ -n "$extra" ]; then
        fail "$loc/$ns has keys not in $PRIMARY (stale):"
        echo "$extra" | sed 's/^/      /'
        annotate "$LOCALES_DIR/$loc/$ns" "stale keys not in $PRIMARY: $(echo "$extra" | tr '\n' ',' | sed 's/,$//')"
        issues=$((issues + 1))
      fi
    done
  done
  [ "$issues" -eq 0 ] && ok "all keys present in all locales"
}

# -----------------------------------------------------------------------------
# Check: no empty values
# -----------------------------------------------------------------------------
check_no_empty_values() {
  section "No empty values"
  local issues=0
  for file in "$LOCALES_DIR"/*/*.json; do
    [ -f "$file" ] || continue
    local empty
    empty=$(jq -r 'paths(. == "") | join(".")' "$file" 2>/dev/null)
    if [ -n "$empty" ]; then
      fail "$file has empty value at keys:"
      echo "$empty" | sed 's/^/      /'
      annotate "$file" "empty value(s): $(echo "$empty" | tr '\n' ',' | sed 's/,$//')"
      issues=$((issues + 1))
    fi
  done
  [ "$issues" -eq 0 ] && ok "no empty values"
}

# -----------------------------------------------------------------------------
# Check: no fallback patterns (t('key', 'English fallback'))
# -----------------------------------------------------------------------------
check_no_fallback_patterns() {
  section "No t('key', 'fallback') patterns in $UI_SRC_DIR"
  [ ! -d "$UI_SRC_DIR" ] && { warn "UI_SRC_DIR=$UI_SRC_DIR does not exist; skipping"; return; }

  # Match t('key', '...something...') with single OR double quotes for both
  # arguments. Ignore: t(key) [single arg], t('key', { interpolation }) [object].
  # The interpolation form starts with { not a quote.
  local hits
  hits=$(grep -rnE "t\(\s*['\"][^'\"]+['\"]\s*,\s*['\"]" "$UI_SRC_DIR" \
    --include='*.ts' --include='*.tsx' 2>/dev/null \
    | grep -v "// allow-fallback") || true

  if [ -n "$hits" ]; then
    local count
    count=$(echo "$hits" | wc -l | tr -d ' ')
    ratchet_fail "$count fallback pattern(s) t('key', 'string') — banned per I18N_CONVENTIONS:"
    echo "$hits" | sed 's/^/      /'
    while IFS= read -r line; do
      local file
      file=$(echo "$line" | cut -d: -f1)
      local lineno
      lineno=$(echo "$line" | cut -d: -f2)
      annotate "$file" "line $lineno: fallback pattern banned — add key to locale file instead"
    done <<<"$hits"
  else
    ok "no fallback patterns"
  fi
}

# -----------------------------------------------------------------------------
# Check: banned vocabulary in locale files
# -----------------------------------------------------------------------------
check_banned_vocab() {
  section "Banned vocabulary in locale files"
  if [ ! -f "$BANNED_FILE" ]; then
    warn "BANNED_FILE=$BANNED_FILE missing; skipping banned-vocab check"
    return
  fi
  local issues=0
  while IFS= read -r term; do
    # Skip empty lines and comments
    [ -z "$term" ] && continue
    case "$term" in '#'*) continue ;; esac
    # Case-insensitive search; word-boundary-ish via -E and surrounding non-letter
    local hits
    hits=$(grep -rinE "(^|[^a-zA-Z])$term([^a-zA-Z]|\$)" "$LOCALES_DIR" --include='*.json' 2>/dev/null) || true
    if [ -n "$hits" ]; then
      fail "banned term '$term' found in locale files:"
      echo "$hits" | sed 's/^/      /'
      while IFS= read -r hit; do
        local file
        file=$(echo "$hit" | cut -d: -f1)
        annotate "$file" "banned vocabulary: '$term' — see CLAUDE.md banned list"
      done <<<"$hits"
      issues=$((issues + 1))
    fi
  done <"$BANNED_FILE"
  [ "$issues" -eq 0 ] && ok "no banned vocabulary"
}

# -----------------------------------------------------------------------------
# Check: glossary preservation (terms appear verbatim in es when in en)
# -----------------------------------------------------------------------------
check_glossary_preservation() {
  section "Glossary preservation (technical terms verbatim in secondary locales)"
  if [ ! -f "$GLOSSARY_FILE" ]; then
    warn "GLOSSARY_FILE=$GLOSSARY_FILE missing; skipping glossary check"
    return
  fi
  local issues=0
  for ns_file in "$LOCALES_DIR/$PRIMARY"/*.json; do
    [ -f "$ns_file" ] || continue
    local ns
    ns=$(basename "$ns_file")
    for loc in "${SECONDARY_LOCALES[@]}"; do
      local other="$LOCALES_DIR/$loc/$ns"
      [ -f "$other" ] || continue
      while IFS= read -r term; do
        [ -z "$term" ] && continue
        case "$term" in '#'*) continue ;; esac
        # Find keys in EN that contain the term as a whole word
        local en_keys_with_term
        en_keys_with_term=$(jq -r --arg term "$term" \
          'paths(strings | test("(^|[^a-zA-Z0-9_])" + $term + "([^a-zA-Z0-9_]|$)")) | join(".")' \
          "$ns_file" 2>/dev/null | sort -u)
        [ -z "$en_keys_with_term" ] && continue
        # For each such key, check the same key in the other locale ALSO contains the term
        while IFS= read -r key; do
          [ -z "$key" ] && continue
          # Allow-list: skip if `<ns_basename>.<key>` is in exceptions file
          local ns_base="${ns%.json}"
          local fq_key="${ns_base}.${key}"
          if [ -f "$GLOSSARY_EXCEPTIONS" ] && grep -qxF "$fq_key" "$GLOSSARY_EXCEPTIONS" 2>/dev/null; then
            continue
          fi
          local jq_arr
          jq_arr=$(printf '%s' "$key" | jq -R 'split(".")')
          local en_val other_val
          en_val=$(jq -r --argjson p "$jq_arr" 'getpath($p) // empty' "$ns_file" 2>/dev/null)
          other_val=$(jq -r --argjson p "$jq_arr" 'getpath($p) // empty' "$other" 2>/dev/null)
          [ -z "$other_val" ] && continue
          if ! printf '%s' "$other_val" | grep -qE "(^|[^a-zA-Z0-9_])$term([^a-zA-Z0-9_]|$)"; then
            fail "$loc/$ns key '$key' missing glossary term '$term'"
            printf "      EN: %s\n" "$en_val"
            printf "      %s: %s\n" "$loc" "$other_val"
            printf "      (if intentional, add '%s' to %s)\n" "$fq_key" "$GLOSSARY_EXCEPTIONS"
            annotate "$other" "key '$key' must contain glossary term '$term' verbatim, or allow-list '$fq_key' in $GLOSSARY_EXCEPTIONS"
            issues=$((issues + 1))
          fi
        done <<<"$en_keys_with_term"
      done <"$GLOSSARY_FILE"
    done
  done
  [ "$issues" -eq 0 ] && ok "all glossary terms preserved in secondary locales"
}

# -----------------------------------------------------------------------------
# Check: interpolation parity ({{var}} tokens match across locales)
# -----------------------------------------------------------------------------
check_interpolation_parity() {
  section "Interpolation parity ({{var}} tokens match)"
  local issues=0
  for ns_file in "$LOCALES_DIR/$PRIMARY"/*.json; do
    [ -f "$ns_file" ] || continue
    local ns
    ns=$(basename "$ns_file")
    for loc in "${SECONDARY_LOCALES[@]}"; do
      local other="$LOCALES_DIR/$loc/$ns"
      [ -f "$other" ] || continue
      # For every key in EN with at least one {{var}}, extract the sorted var set
      # and compare to the same key in the other locale.
      local en_pairs
      # Set-based parity: deduplicate vars on both sides. We care that the
      # same SET of variables is used, not how many times each one appears.
      # i18next happily resolves a single value to multiple occurrences.
      en_pairs=$(jq -r '
        paths(strings) as $p
        | getpath($p) as $v
        | select($v | test("\\{\\{[^}]+\\}\\}"))
        | [($p | join(".")),
           ($v | [scan("\\{\\{[^}]+\\}\\}")] | unique | sort | join(","))]
        | @tsv
      ' "$ns_file" 2>/dev/null)
      while IFS=$'\t' read -r key en_vars; do
        [ -z "$key" ] && continue
        local jq_arr
        jq_arr=$(printf '%s' "$key" | jq -R 'split(".")')
        local other_val other_vars
        other_val=$(jq -r --argjson p "$jq_arr" 'getpath($p) // empty' "$other" 2>/dev/null)
        [ -z "$other_val" ] && continue
        other_vars=$(printf '%s' "$other_val" | grep -oE '\{\{[^}]+\}\}' | sort -u | tr '\n' ',' | sed 's/,$//')
        if [ "$en_vars" != "$other_vars" ]; then
          fail "$loc/$ns key '$key' interpolation drift"
          printf "      EN vars: %s\n" "$en_vars"
          printf "      %s vars: %s\n" "$loc" "$other_vars"
          annotate "$other" "key '$key' interpolation vars mismatch: EN [$en_vars] vs $loc [$other_vars]"
          issues=$((issues + 1))
        fi
      done <<<"$en_pairs"
    done
  done
  [ "$issues" -eq 0 ] && ok "interpolation vars match across locales"
}

# -----------------------------------------------------------------------------
# Check: plural completeness (_one paired with _other; vice versa)
# -----------------------------------------------------------------------------
check_plural_completeness() {
  section "Plural completeness (_one + _other)"
  local issues=0
  for file in "$LOCALES_DIR"/*/*.json; do
    [ -f "$file" ] || continue
    local plural_keys
    plural_keys=$(jq -r 'paths(scalars) | join(".")' "$file" 2>/dev/null \
      | grep -E "_(one|other|zero|two|few|many)$" || true)
    [ -z "$plural_keys" ] && continue
    # For each base key, ensure both _one and _other exist
    local bases
    bases=$(echo "$plural_keys" | sed -E 's/_(one|other|zero|two|few|many)$//' | sort -u)
    while IFS= read -r base; do
      [ -z "$base" ] && continue
      local has_one has_other
      echo "$plural_keys" | grep -qE "^${base}_one$" && has_one=1 || has_one=0
      echo "$plural_keys" | grep -qE "^${base}_other$" && has_other=1 || has_other=0
      if [ "$has_one" -eq 1 ] && [ "$has_other" -eq 0 ]; then
        fail "$file has ${base}_one without ${base}_other"
        annotate "$file" "plural incomplete: ${base}_one without ${base}_other"
        issues=$((issues + 1))
      fi
      if [ "$has_other" -eq 1 ] && [ "$has_one" -eq 0 ]; then
        fail "$file has ${base}_other without ${base}_one"
        annotate "$file" "plural incomplete: ${base}_other without ${base}_one"
        issues=$((issues + 1))
      fi
    done <<<"$bases"
  done
  [ "$issues" -eq 0 ] && ok "all plural keys complete"
}

# -----------------------------------------------------------------------------
# Check: hardcoded English text in JSX (warn-only — regex is fuzzy)
# -----------------------------------------------------------------------------
check_hardcoded_jsx() {
  section "Hardcoded English JSX text (warn-only)"
  [ ! -d "$UI_SRC_DIR" ] && { warn "UI_SRC_DIR missing; skipping"; return; }

  # Heuristic: JSX text nodes starting with an uppercase letter followed by
  # lowercase letters and a space — typical English sentence pattern.
  # Allowlist common technical strings (single capitalized word, glossary terms).
  local hits
  hits=$(grep -rnE ">[A-Z][a-z]+ [a-zA-Z]" "$UI_SRC_DIR" \
    --include='*.tsx' 2>/dev/null \
    | grep -v "// allow-hardcoded" \
    | grep -v ".stories.tsx" \
    | grep -v "/test/" \
    | grep -v ".test.tsx") || true

  if [ -n "$hits" ]; then
    local count
    count=$(echo "$hits" | wc -l | tr -d ' ')
    warn "$count possible hardcoded JSX strings (heuristic; manual review needed):"
    echo "$hits" | head -10 | sed 's/^/      /'
    [ "$count" -gt 10 ] && printf "      ... and %d more\n" $((count - 10))
  else
    ok "no hardcoded JSX text detected"
  fi
}

# -----------------------------------------------------------------------------
# Check: locked package versions (matches I18N_CONVENTIONS.md)
# -----------------------------------------------------------------------------
check_locked_versions() {
  section "i18n package versions (pinned exact, per CLAUDE.md)"
  local pkg="${UI_SRC_DIR%/src}/package.json"
  [ ! -f "$pkg" ] && { warn "package.json not found at $pkg; skipping"; return; }
  local issues=0
  # POSIX-compatible parallel arrays (no `declare -A` — macOS bash 3.x).
  local names=(i18next react-i18next i18next-browser-languagedetector)
  local wants=(26.3.0 17.0.8 8.2.1)
  local i=0
  while [ "$i" -lt "${#names[@]}" ]; do
    local name="${names[$i]}"
    local want="${wants[$i]}"
    local actual
    actual=$(jq -r --arg n "$name" '.dependencies[$n] // .devDependencies[$n] // empty' "$pkg")
    if [ -z "$actual" ]; then
      warn "$name not installed in $pkg"
    elif [ "$actual" != "$want" ]; then
      ratchet_fail "$name pinned to $actual but lockstep target is $want"
      annotate "$pkg" "$name should be pinned to $want (CLAUDE.md always-latest + I18N_CONVENTIONS.md)"
      issues=$((issues + 1))
    fi
    i=$((i + 1))
  done
  [ "$issues" -eq 0 ] && ok "all i18n packages on locked versions"
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------
QUICK=0
RATCHET=0
ONLY_CHECK=""
while [ $# -gt 0 ]; do
  case "$1" in
    --quick)   QUICK=1; shift ;;
    --ratchet) RATCHET=1; shift ;;
    --check)   ONLY_CHECK="$2"; shift 2 ;;
    --help|-h)
      sed -n '2,30p' "$0"
      exit 0
      ;;
    *) echo "Unknown arg: $1"; exit 2 ;;
  esac
done

# Helper used by the two ratchet-eligible checks to downgrade fail → warn.
ratchet_fail() {
  if [ "$RATCHET" -eq 1 ]; then
    warn "$1"
  else
    fail "$1"
  fi
}

run_check() {
  local fn="$1"
  [ -n "$ONLY_CHECK" ] && [ "$ONLY_CHECK" != "${fn#check_}" ] && return
  $fn
}

run_check check_key_parity
run_check check_no_empty_values
run_check check_no_fallback_patterns
run_check check_banned_vocab
run_check check_glossary_preservation
run_check check_interpolation_parity
run_check check_plural_completeness
run_check check_locked_versions
[ "$QUICK" -eq 0 ] && run_check check_hardcoded_jsx

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------
echo ""
printf "${B}=== Summary ===${N}\n"
if [ "$FAILED" -gt 0 ]; then
  printf "${R}FAILED: %d issue(s)${N}\n" "$FAILED"
  [ "$WARNED" -gt 0 ] && printf "${Y}WARNINGS: %d${N}\n" "$WARNED"
  exit 1
fi
if [ "$WARNED" -gt 0 ]; then
  printf "${Y}OK with %d warning(s)${N}\n" "$WARNED"
else
  printf "${G}OK — all checks passed${N}\n"
fi
exit 0
