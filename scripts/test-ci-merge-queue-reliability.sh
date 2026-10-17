#!/usr/bin/env bash
# The two guards this covers do not fire on a pull_request run.
#
# SARIF upload fails only after the merge queue deletes
# refs/heads/gh-readonly-queue/... . release-please's enqueue goes red only
# when `gh pr checks --required` passes before a check named "CI Complete"
# exists. Both are reproduced here against the workflow source.
set -uo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
failures=0

fail() {
  echo "FAIL: $*" >&2
  failures=$((failures + 1))
}

ok() {
  echo "ok: $*"
}

python3 - "$root" <<'PY' || failures=$((failures + 1))
import sys
from pathlib import Path

root = Path(sys.argv[1])
workflows = sorted((root / ".github" / "workflows").glob("*.yml"))
uploads = []
for path in workflows:
    lines = path.read_text(encoding="utf-8").splitlines()
    i = 0
    while i < len(lines):
        if lines[i].startswith("      - name:") or lines[i].startswith("      - uses:"):
            start = i
            i += 1
            while i < len(lines) and not lines[i].startswith("      - "):
                i += 1
            step = "\n".join(lines[start:i])
            if "github/codeql-action/upload-sarif@" in step:
                uploads.append((path.name, step))
        else:
            i += 1

if len(uploads) < 4:
    print(f"FAIL: expected gosec, trivy, codeql, and scorecard uploads, found {len(uploads)}", file=sys.stderr)
    sys.exit(1)

for name, step in uploads:
    if "github.event_name != 'merge_group'" not in step:
        print(f"FAIL: {name} SARIF upload does not skip merge_group:\n{step}", file=sys.stderr)
        sys.exit(1)
    print(f"ok: {name} upload skips merge_group")

ci = (root / ".github" / "workflows" / "ci.yml").read_text(encoding="utf-8")
codeql = (root / ".github" / "workflows" / "codeql.yml").read_text(encoding="utf-8")

def step_named(text, title):
    marker = f"- name: {title}"
    start = text.index(marker)
    rest = text[start + len(marker):]
    end = rest.find("\n      - name:")
    return rest[:end if end != -1 else len(rest)]

for title in ("Upload gosec results", "Upload Trivy results"):
    body = step_named(ci, title)
    if "github.event_name != 'merge_group'" not in body or "gh-readonly-queue" not in body:
        print(f"FAIL: inline {title} does not skip merge-queue SARIF upload", file=sys.stderr)
        sys.exit(1)
    if "upload-sarif@" not in body:
        print(f"FAIL: {title} is not an upload-sarif step", file=sys.stderr)
        sys.exit(1)
    print(f"ok: inline {title} skips merge_group and gh-readonly-queue refs")

for title in (
    "Run govulncheck",
    "Run gosec (SARIF visibility only — gate is golangci-lint)",
    "Run Trivy vulnerability scanner",
    "Fail on CRITICAL/HIGH findings",
):
    body = step_named(ci, title)
    if "github.event_name != 'merge_group'" in body:
        print(f"FAIL: scan gate {title!r} is skipped on merge_group", file=sys.stderr)
        sys.exit(1)
    print(f"ok: {title} still gates merge_group")

if "upload: ${{ github.event_name == 'merge_group' && 'never' || 'failure-only' }}" not in codeql:
    print("FAIL: CodeQL analyze still uploads SARIF on merge_group when analysis fails", file=sys.stderr)
    sys.exit(1)
print("ok: CodeQL analyze does not upload on merge_group")

semgrep_pin = "MustardSeedNetworks/.github/.github/workflows/semgrep.yml@8f012be52b47d0be45104b0ed8d3eb72f20840f5"
if semgrep_pin not in ci:
    print("FAIL: semgrep is not pinned to .github#174, which skips the merge-queue SARIF upload", file=sys.stderr)
    sys.exit(1)
if "workflows/semgrep.yml@048d173dd72ca0f9e5fa832fb5b9fd2a6447e5e9" in ci:
    print("FAIL: semgrep is still pinned to the workflow that uploads on merge_group", file=sys.stderr)
    sys.exit(1)
print("ok: semgrep is pinned to the shared workflow that skips merge-queue upload")
PY

workflow="${root}/.github/workflows/release-please.yml"
script="$(mktemp)"
python3 - "$workflow" "$script" <<'PY' || failures=$((failures + 1))
import sys
from pathlib import Path

text = Path(sys.argv[1]).read_text(encoding="utf-8")
start = text.index("- name: Enqueue the release PR at the front of the merge queue")
run_at = text.index("\n        run: |\n", start)
rest = text[run_at + len("\n        run: |\n"):]
end = rest.find("\n      - name:")
if end == -1:
    print("FAIL: enqueue run block has no following step", file=sys.stderr)
    sys.exit(1)
body = rest[:end]
lines = []
for line in body.splitlines():
    if line.startswith("          "):
        lines.append(line[10:])
    elif line.strip() == "":
        lines.append("")
    else:
        print(f"FAIL: unexpected indent in enqueue script: {line!r}", file=sys.stderr)
        sys.exit(1)
extracted = "\n".join(lines) + "\n"
if 'name == "CI Complete"' not in extracted and "CI Complete" not in extracted:
    print("FAIL: enqueue script does not name CI Complete", file=sys.stderr)
    sys.exit(1)
if "gh pr merge" in extracted:
    print("FAIL: enqueue script merged the PR instead of queueing it", file=sys.stderr)
    sys.exit(1)
Path(sys.argv[2]).write_text(extracted, encoding="utf-8")
print("ok: extracted enqueue script")
PY

if ! bash -n "$script"; then
  fail "enqueue script failed bash -n"
else
  ok "enqueue script passes bash -n"
fi

fake_root="$(mktemp -d)"
cat > "${fake_root}/gh" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "${GH_LOG}"
args="$*"
case "${args}" in
  *'releases/latest'*)
    printf '%s\n' 'v0.1.0'
    ;;
  *'/compare/'*)
    printf '%s\n' '{"ahead":2,"newest":"2026-10-16T00:00:00Z"}'
    ;;
  *'pr list'*)
    printf '%s\n' '42'
    ;;
  *'commits/release-please--branches--main'*)
    printf '%s\n' 'abc123'
    ;;
  *'commits/main'*)
    printf '%s\n' 'abc123'
    ;;
  *'pr checks'*'--json name,bucket'*)
    poll=0
    if [ -n "${GH_POLL_COUNT_FILE:-}" ]; then
      poll="$(cat "${GH_POLL_COUNT_FILE}")"
      printf '%s\n' "$((poll + 1))" > "${GH_POLL_COUNT_FILE}"
    fi
    if [ "${poll}" -eq 0 ] && [ -n "${GH_CHECKS_JSON_FIRST:-}" ]; then
      printf '%s\n' "${GH_CHECKS_JSON_FIRST}"
    else
      printf '%s\n' "${GH_CHECKS_JSON}"
    fi
    exit "${GH_CHECKS_EXIT:-0}"
    ;;
  *'pr checks'*'--required'*)
    exit "${GH_REQUIRED_EXIT:-0}"
    ;;
  *'pr view'*)
    printf '%s\n' 'PR_NODE'
    ;;
  *'-F jump=true'*)
    if [ "${GH_ENQUEUE_TRUE_MODE:-ok}" = "expected" ]; then
      printf '%s\n' 'Pull request Required status check "CI Complete" is expected.' >&2
      exit 1
    fi
    if [ "${GH_ENQUEUE_TRUE_MODE:-ok}" = "denied" ]; then
      printf '%s\n' 'graphql error: jump not permitted' >&2
      exit 1
    fi
    printf '%s\n' 'queued at position 1'
    ;;
  *'mergeQueue'*)
    printf '%s\n' "${GH_QUEUE_DEPTH:-0}"
    ;;
  *'-F jump=false'*)
    if [ "${GH_ENQUEUE_FALSE_MODE:-ok}" = "fail" ]; then
      printf '%s\n' 'jump false failed' >&2
      exit 1
    fi
    printf '%s\n' 'queued at position 2'
    ;;
  *)
    printf 'unexpected gh: %s\n' "$*" >&2
    exit 99
    ;;
esac
EOF
chmod +x "${fake_root}/gh"

run_case() {
  local name="$1" want="$2"
  shift 2
  local dir
  dir="$(mktemp -d)"
  local log="${dir}/gh.log"
  : > "${log}"
  local out="${dir}/out"
  local err="${dir}/err"
  set +e
  env \
    PATH="${fake_root}:${PATH}" \
    GH_LOG="${log}" \
    GITHUB_REPOSITORY="MustardSeedNetworks/seed" \
    GITHUB_REPOSITORY_OWNER="MustardSeedNetworks" \
    GITHUB_EVENT_NAME="workflow_run" \
    ENQUEUE='mutation { enqueuePullRequest(input: {pullRequestId: $id, jump: $jump}) { mergeQueueEntry { position } } }' \
    QUEUE_DEPTH='query { repository { mergeQueue(branch: "main") { entries { totalCount } } } }' \
    CI_COMPLETE_INITIAL_DELAY=0 \
    CI_COMPLETE_MAX_DELAY=0 \
    "$@" \
    bash "$script" >"${out}" 2>"${err}"
  local got=$?
  set -e
  CASE_OUT="${out}"
  CASE_ERR="${err}"
  CASE_LOG="${log}"
  if [ "${got}" -ne "${want}" ]; then
    fail "${name} exited ${got}, want ${want}"
    echo "--- stdout ---" >&2
    cat "${out}" >&2
    echo "--- stderr ---" >&2
    cat "${err}" >&2
    echo "--- gh ---" >&2
    cat "${log}" >&2
    return
  fi
  ok "${name} exited ${want}"
}

count_jumps() {
  local flag="$1" file="$2"
  grep -c -- "-F ${flag}" "${file}" || true
}

# Required checks that have reported are green, but CI Complete does not
# exist yet. The run must stay green and must not call the enqueue API.
run_case "missing CI Complete does not fail" 0 \
  CI_COMPLETE_WAIT_SECONDS=0 \
  GH_CHECKS_JSON='[{"name":"Lint PR Title","bucket":"pass"},{"name":"Lint PR Body","bucket":"pass"}]'
if [ "$(count_jumps jump=true "${CASE_LOG}")" -ne 0 ]; then
  fail "missing CI Complete still enqueued"
else
  ok "missing CI Complete did not enqueue"
fi
if ! grep -q 'Not failing this run' "${CASE_OUT}"; then
  fail "missing CI Complete did not say the run is not failing"
fi

# The check appears on the second poll. Wait, then jump the queue.
poll_file="$(mktemp)"
printf '0\n' > "${poll_file}"
run_case "waits until CI Complete passes" 0 \
  CI_COMPLETE_WAIT_SECONDS=30 \
  GH_POLL_COUNT_FILE="${poll_file}" \
  GH_CHECKS_JSON_FIRST='[{"name":"CI Complete","bucket":"pending"}]' \
  GH_CHECKS_JSON='[{"name":"CI Complete","bucket":"pass"}]'
if [ "$(count_jumps jump=true "${CASE_LOG}")" -ne 1 ]; then
  fail "passing CI Complete did not enqueue once with jump"
else
  ok "passing CI Complete enqueued with jump"
fi
if [ "$(count_jumps jump=false "${CASE_LOG}")" -ne 0 ]; then
  fail "passing CI Complete also enqueued without jump"
fi

# A red CI Complete is not something to enqueue, and not a failed run.
run_case "failed CI Complete does not enqueue" 0 \
  CI_COMPLETE_WAIT_SECONDS=30 \
  GH_CHECKS_JSON='[{"name":"CI Complete","bucket":"fail"}]'
if [ "$(count_jumps jump=true "${CASE_LOG}")" -ne 0 ]; then
  fail "failed CI Complete was enqueued"
else
  ok "failed CI Complete was not enqueued"
fi

# Other required checks still pending after CI Complete passed.
run_case "other required checks still pending" 0 \
  CI_COMPLETE_WAIT_SECONDS=30 \
  GH_REQUIRED_EXIT=8 \
  GH_CHECKS_JSON='[{"name":"CI Complete","bucket":"pass"}]'
if [ "$(count_jumps jump=true "${CASE_LOG}")" -ne 0 ]; then
  fail "enqueued while other required checks were pending"
else
  ok "did not enqueue while other required checks were pending"
fi

# The queue says CI Complete is expected even though the check list shows
# a pass. Retry, then leave the run green. Do not fall through to jump=false,
# which is what exited 1 on 2026-10-16.
run_case "expected CI Complete retries then exits 0" 0 \
  CI_COMPLETE_WAIT_SECONDS=30 \
  ENQUEUE_EXPECTED_RETRIES=2 \
  GH_ENQUEUE_TRUE_MODE=expected \
  GH_ENQUEUE_FALSE_MODE=fail \
  GH_CHECKS_JSON='[{"name":"CI Complete","bucket":"pass"}]'
jumps="$(count_jumps jump=true "${CASE_LOG}")"
if [ "${jumps}" -ne 2 ]; then
  fail "expected-check retry count was ${jumps}, want 2"
else
  ok "expected-check refusal retried"
fi
if [ "$(count_jumps jump=false "${CASE_LOG}")" -ne 0 ]; then
  fail "expected-check refusal fell through to jump=false"
else
  ok "expected-check refusal did not fail via jump=false"
fi

# A jump permission refusal is a different error. An empty queue may enqueue
# at the back; a non-empty queue must wait.
run_case "jump refused on an empty queue falls back" 0 \
  CI_COMPLETE_WAIT_SECONDS=30 \
  GH_ENQUEUE_TRUE_MODE=denied \
  GH_QUEUE_DEPTH=0 \
  GH_CHECKS_JSON='[{"name":"CI Complete","bucket":"pass"}]'
if [ "$(count_jumps jump=false "${CASE_LOG}")" -ne 1 ]; then
  fail "empty queue did not fall back to jump=false"
else
  ok "empty queue fell back to jump=false"
fi

run_case "jump refused on a busy queue waits" 0 \
  CI_COMPLETE_WAIT_SECONDS=30 \
  GH_ENQUEUE_TRUE_MODE=denied \
  GH_QUEUE_DEPTH=3 \
  GH_ENQUEUE_FALSE_MODE=fail \
  GH_CHECKS_JSON='[{"name":"CI Complete","bucket":"pass"}]'
if [ "$(count_jumps jump=false "${CASE_LOG}")" -ne 0 ]; then
  fail "busy queue enqueued at the back"
else
  ok "busy queue did not enqueue at the back"
fi

rm -rf "${fake_root}" "${script}" "${poll_file}"

if [ "${failures}" -ne 0 ]; then
  echo "${failures} merge-queue reliability failure(s)" >&2
  exit 1
fi
echo "merge-queue reliability checks passed"
