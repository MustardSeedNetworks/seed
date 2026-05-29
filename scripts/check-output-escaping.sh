#!/usr/bin/env bash
# check-output-escaping.sh — output-escaping / XSS regression gate (#1225).
#
# Audited 2026-05-28: classified all 3 fmt.Fprintf(w, ...) sites — all in
# internal/api/handlers_sse.go:
#   - "data: %s\n\n", message  — SSE data frame; payload is a
#     JSON-marshaled message, Content-Type text/event-stream, consumed
#     by EventSource as data (not rendered as HTML).
#   - ": heartbeat\n\n"        — literal SSE comment.
# seed also uses html/template (auto-escaping) and has no
# text/template-for-HTML. No site renders user data as HTML.
#
# Two checks:
#   1. No raw innerHTML injection in ui/src (React XSS vector).
#   2. No NEW value-interpolating fmt.Fprintf(w, "...%s...") in
#      internal/api outside the audited allow-list (handlers_sse.go).
#      HTTP responses should use sendJSONResponse / json.Encode or
#      html/template, never raw format strings.
#
# Run locally: scripts/check-output-escaping.sh

set -uo pipefail

FAIL=0
INNER_HTML_RE='dangerously[S]etInnerHTML'

# handlers_sse.go: SSE data frames carry JSON payloads over
# text/event-stream, consumed by EventSource — not an HTML sink.
ALLOWLIST_RE='internal/api/handlers_sse\.go'

UI_DIR=""
if [ -d "ui/src" ]; then
  UI_DIR="ui/src"
elif [ -d "src" ] && [ -f "package.json" ]; then
  UI_DIR="src"
fi

if [ -n "$UI_DIR" ]; then
  DANGER=$(grep -rEn "$INNER_HTML_RE" "$UI_DIR" \
    --include='*.tsx' --include='*.ts' 2>/dev/null \
    | grep -vE '\.(test|spec|stories)\.(ts|tsx):' || true)
  if [ -n "$DANGER" ]; then
    echo "============================================================"
    echo "[XSS] raw innerHTML injection found in UI app code:"
    echo "$DANGER"
    echo "Use plain text/JSX, or sanitize with DOMPurify and justify in review."
    echo ""
    FAIL=1
  fi
fi

if [ -d "internal/api" ]; then
  HTTP_FMT=$(grep -rEn 'fmt\.Fprintf\(w,[^)]*%[svqd]' internal/api 2>/dev/null \
    | grep -v '_test.go' \
    | grep -vE "$ALLOWLIST_RE" || true)
  if [ -n "$HTTP_FMT" ]; then
    echo "============================================================"
    echo "[XSS] value-interpolating fmt.Fprintf(w, ...) in internal/api —"
    echo "HTTP responses must use sendJSONResponse / json.Encode or"
    echo "html/template, not raw format strings:"
    echo "$HTTP_FMT"
    echo ""
    echo "If a new site is deliberately safe (SSE/JSON wire format),"
    echo "add it to ALLOWLIST_RE with a justification note."
    echo ""
    FAIL=1
  fi
fi

if [ "$FAIL" -ne 0 ]; then
  echo "FAIL: output-escaping gate (#1225)."
  exit 1
fi

echo "OK: no raw innerHTML injection; internal/api Fprintf sites are within the audited allow-list."
