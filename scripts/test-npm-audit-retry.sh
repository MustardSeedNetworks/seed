#!/usr/bin/env bash
#
# Tests for npm-audit-retry.sh, driven by a stub npm on PATH.
#
# The point of the wrapper is a distinction — an unreachable advisory endpoint
# is worth retrying, a real finding is not — and that distinction is made on
# `npm audit --json`'s structure (a top-level "error" key vs.
# "metadata.vulnerabilities"), never on npm's human-readable prose (#2416).
# These tests check that branching, the case #2416 filed (a completed audit
# whose report text contains the word "network" but has zero findings), and
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

check_not() {
    local name="$1" out="$2" avoid_text="$3"
    if grep -q "$avoid_text" <<<"$out"; then
        printf 'FAIL %s: output unexpectedly mentions %s\n%s\n' "$name" "$avoid_text" "$out"
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
OUT="$(run_case 'echo "{\"metadata\":{\"vulnerabilities\":{\"high\":0,\"critical\":0,\"total\":0}}}"; exit 0')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "clean audit passes" 0 "$STATUS" "$OUT" "0 vulnerabilities found"
[ "$CALLS" = "1" ] || { printf 'FAIL clean audit retried (%s calls)\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

# 2. A real finding fails immediately and is NOT retried — it will not heal.
OUT="$(run_case 'echo "{\"metadata\":{\"vulnerabilities\":{\"high\":3,\"critical\":0,\"total\":3}},\"vulnerabilities\":{\"foo\":{\"severity\":\"high\",\"range\":\"<1.0.0\"}}}"; exit 1')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "real finding fails" 1 "$STATUS" "$OUT" "3 vulnerabilities found (3 high"
[ "$CALLS" = "1" ] || { printf 'FAIL a real finding was retried (%s calls)\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

# 3. An endpoint outage is retried, and succeeds when the endpoint comes back.
#    The stub reads its own call log, whose path it only learns at run time, so
#    the body refers to $COUNTER — expanded inside the subshell, not here.
OUT="$(run_case 'if [ "$(wc -l <"$0.calls" 2>/dev/null || echo 0)" -lt 1 ]; then echo done >"$0.calls"; echo "{\"error\":{\"summary\":\"audit endpoint returned an error\"}}"; exit 1; fi; echo "{\"metadata\":{\"vulnerabilities\":{\"high\":0,\"critical\":0,\"total\":0}}}"; exit 0')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
check "outage then recovery passes" 0 "$STATUS" "$OUT" "0 vulnerabilities found"

# 4. A sustained outage still fails: the audit did not run, so we do not claim it did.
OUT="$(run_case 'echo "{\"error\":{\"summary\":\"audit endpoint returned an error\"}}"; exit 1')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "sustained outage fails" 1 "$STATUS" "$OUT" "could not reach the advisory endpoint"
[ "$CALLS" = "3" ] || { printf 'FAIL sustained outage made %s attempts, want 3\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

# 5. #2416: a completed audit (no "error" key) whose report text contains the
#    literal word "network", with zero vulnerabilities, must pass on the
#    first attempt with no outage message — the case the old grep-on-prose
#    check misclassified as a transport failure.
OUT="$(run_case 'echo "{\"metadata\":{\"vulnerabilities\":{\"high\":0,\"critical\":0,\"total\":0}},\"vulnerabilities\":{\"network-thing\":{}}}"; exit 0')"
STATUS="${OUT##*__STATUS__}"; STATUS="${STATUS%%$'\n'*}"
CALLS="${OUT##*__CALLS__}"
check "network-named finding with zero vulns passes" 0 "$STATUS" "$OUT" "0 vulnerabilities found"
check_not "network-named finding with zero vulns is not retried" "$OUT" "could not reach the advisory endpoint"
[ "$CALLS" = "1" ] || { printf 'FAIL #2416 case was retried (%s calls)\n' "$CALLS"; FAILURES=$((FAILURES + 1)); }

if [ "$FAILURES" -ne 0 ]; then
    printf '\n%d test(s) failed\n' "$FAILURES"
    exit 1
fi
printf '\nnpm-audit-retry: all tests passed\n'
