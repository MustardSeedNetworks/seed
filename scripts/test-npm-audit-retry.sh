#!/usr/bin/env bash
#
# Tests for npm-audit-retry.sh, driven by a stub npm on PATH.
#
# The point of the wrapper is a distinction — an unreachable advisory endpoint is
# worth retrying, a real finding is not — so that is what these check, along with
# the rule that a sustained outage still fails the build (#2387).
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$HERE/npm-audit-retry.sh"
FAILURES=0

# Fast retries: the tests are about the branching, not the backoff.
export NPM_AUDIT_RETRY_DELAY=0

check() {
    local name="$1" want_status="$2" got_status="$3" out="$4" want_text="${5:-}"
    if [ "$got_status" -ne "$want_status" ]; then
        printf 'FAIL %s: exit %s, want %s\n%s\n' "$name" "$got_status" "$want_status" "$out"
        FAILURES=$((FAILURES + 1))
        return
    fi
    if [ -n "$want_text" ] && ! grep -q "$want_text" <<<"$out"; then
        printf 'FAIL %s: output does not mention %s\n%s\n' "$name" "$want_text" "$out"
        FAILURES=$((FAILURES + 1))
        return
    fi
    printf 'ok   %s\n' "$name"
}

# stub_npm <script-body> builds a fake npm and puts it first on PATH.
stub_npm() {
    STUB_DIR="$(mktemp -d)"
    COUNTER="$STUB_DIR/calls"
    : >"$COUNTER"
    cat >"$STUB_DIR/npm" <<STUB
#!/usr/bin/env bash
echo call >>"$COUNTER"
$1
STUB
    chmod +x "$STUB_DIR/npm"
    export PATH="$STUB_DIR:$PATH"
}

run_case() {
    ( stub_npm "$1"; "$SCRIPT" high 2>&1; echo "__STATUS__$?"; echo "__CALLS__$(wc -l <"$COUNTER" | tr -d ' ')" )
}

# 1. A clean audit passes on the first attempt.
OUT="$(run_case 'echo "found 0 vulnerabilities"; exit 0')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "clean audit passes" 0 "$STATUS" "$OUT" "found 0 vulnerabilities"
[ "$CALLS" = "1" ] || { printf 'FAIL clean audit retried (%s calls)\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

# 2. A real finding fails immediately and is NOT retried — it will not heal.
OUT="$(run_case 'echo "found 3 high severity vulnerabilities"; exit 1')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "real finding fails" 1 "$STATUS" "$OUT" "high severity"
[ "$CALLS" = "1" ] || { printf 'FAIL a real finding was retried (%s calls)\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

# 3. An endpoint outage is retried, and succeeds when the endpoint comes back.
#    The stub reads its own call log, whose path it only learns at run time, so
#    the body refers to $COUNTER — expanded inside the subshell, not here.
OUT="$(run_case 'if [ "$(wc -l <"$0.calls" 2>/dev/null || echo 0)" -lt 1 ]; then echo done >"$0.calls"; echo "npm error audit endpoint returned an error"; exit 1; fi; echo "found 0 vulnerabilities"; exit 0')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
check "outage then recovery passes" 0 "$STATUS" "$OUT" "found 0 vulnerabilities"

# 4. A sustained outage still fails: the audit did not run, so we do not claim it did.
OUT="$(run_case 'echo "npm warn audit 503 Service Unavailable"; echo "npm error audit endpoint returned an error"; exit 1')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "sustained outage fails" 1 "$STATUS" "$OUT" "could not reach the advisory endpoint"
[ "$CALLS" = "3" ] || { printf 'FAIL sustained outage made %s attempts, want 3\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

if [ "$FAILURES" -ne 0 ]; then
    printf '\n%d test(s) failed\n' "$FAILURES"
    exit 1
fi
printf '\nnpm-audit-retry: all tests passed\n'
