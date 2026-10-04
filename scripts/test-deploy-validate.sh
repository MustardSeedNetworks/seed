#!/usr/bin/env bash
#
# Tests for deploy-validate.sh, driven by a stub curl on PATH that serves a
# canned /__version body.
#
# Release tags carry a leading "v" and the daemon reports the bare version, so
# validating a host against its own tag must pass (#3069). A wrong version or
# commit, or a build without its UI embedded, must fail.
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$HERE/deploy-validate.sh"
FAILURES=0

STUB_DIR="$(mktemp -d)"
trap 'rm -rf "$STUB_DIR"' EXIT
cat >"$STUB_DIR/curl" <<'STUB'
#!/usr/bin/env bash
for arg in "$@"; do
    [ "$arg" = "-o" ] && exit 0
done
printf '%s' "$STUB_BODY"
STUB
chmod +x "$STUB_DIR/curl"
export PATH="$STUB_DIR:$PATH"

COMMIT=6b569bc497541ea5eb48ca1db17614b6396a3fcb

body() {
    printf '{"version":"%s","commit":"%s","buildTime":"2026-10-04T00:19:13Z","uiBuildHash":"%s"}' "$1" "$2" "$3"
}

run_case() {
    local name="$1" want_status="$2" want_text="$3" served="$4"
    shift 4
    local out status
    out="$(STUB_BODY="$served" "$SCRIPT" "$@" 2>&1)"
    status=$?
    if [ "$status" -ne "$want_status" ]; then
        printf 'FAIL %s: exit %s, want %s\n%s\n' "$name" "$status" "$want_status" "$out"
        FAILURES=$((FAILURES + 1))
        return
    fi
    if ! grep -q "$want_text" <<<"$out"; then
        printf 'FAIL %s: output does not mention %s\n%s\n' "$name" "$want_text" "$out"
        FAILURES=$((FAILURES + 1))
        return
    fi
    printf 'ok   %s\n' "$name"
}

run_case "release tag matches bare daemon version" 0 "Version matches" \
    "$(body 0.221.3 "$COMMIT" d029ab7b)" v0.221.3 "$COMMIT" host
run_case "wrong release fails" 1 "Version MISMATCH" \
    "$(body 0.221.3 "$COMMIT" d029ab7b)" v0.227.1 "$COMMIT" host
run_case "wrong commit fails" 1 "Commit MISMATCH" \
    "$(body 0.221.3 "$COMMIT" d029ab7b)" v0.221.3 a81311a9 host
run_case "empty uiBuildHash fails" 1 "UI build hash MISSING" \
    "$(body 0.221.3 "$COMMIT" "")" v0.221.3 "$COMMIT" host
run_case "unknown uiBuildHash fails" 1 "UI build hash MISSING" \
    "$(body 0.221.3 "$COMMIT" unknown)" v0.221.3 "$COMMIT" host

if [ "$FAILURES" -ne 0 ]; then
    printf '%d case(s) failed\n' "$FAILURES"
    exit 1
fi
