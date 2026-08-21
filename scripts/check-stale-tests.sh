#!/usr/bin/env bash
# check-stale-tests.sh — refuse to start a test run while orphaned test binaries
# are still holding the machine.
#
# Go's `-test.timeout` cannot always kill a stuck test binary: if the process is
# blocked in a cgo call or a syscall, the timeout panic never lands and the
# binary keeps running after `go test` has given up on it. They accumulate
# silently. On 2026-08-21 this machine was carrying eleven of them, the oldest
# three days old, holding the load average at 317 — a suite that runs in 72s was
# timing out at 660s, and the obvious reading ("flaky test", "slow machine") was
# wrong in a way no amount of re-running would have revealed.
#
# The threshold is deliberately far above any real run. Every suite here
# finishes in minutes and CI caps at -test.timeout=10m, so a test binary older
# than STALE_AFTER_MINUTES has definitively outlived its own timeout and is
# stuck, not busy. That is what makes failing safe rather than annoying.

set -euo pipefail

STALE_AFTER_MINUTES="${STALE_AFTER_MINUTES:-30}"

# elapsed_seconds converts ps's [[dd-]hh:]mm:ss into seconds. macOS ps has no
# `etimes` keyword, so the parsing is done here rather than asked for.
elapsed_seconds() {
	local etime="$1"
	local days=0 hours=0 minutes seconds
	local rest="$etime"
	if [[ "$rest" == *-* ]]; then
		days="${rest%%-*}"
		rest="${rest#*-}"
	fi
	local IFS=:
	read -ra parts <<<"$rest"
	case "${#parts[@]}" in
		3) hours="${parts[0]}"; minutes="${parts[1]}"; seconds="${parts[2]}" ;;
		2) minutes="${parts[0]}"; seconds="${parts[1]}" ;;
		*) return 1 ;;
	esac
	echo $(( (10#$days * 86400) + (10#$hours * 3600) + (10#$minutes * 60) + 10#$seconds ))
}

threshold=$((STALE_AFTER_MINUTES * 60))
stale_pids=()
stale_rows=()

# `read pid etime command` rather than parameter expansion: ps pads its pid
# column with leading spaces, which %% would otherwise read as an empty field.
while read -r pid etime command; do
	[ -n "$pid" ] || continue
	seconds="$(elapsed_seconds "$etime")" || continue
	if [ "$seconds" -gt "$threshold" ]; then
		stale_pids+=("$pid")
		stale_rows+=("  $pid  $etime  ${command##*/}")
	fi
# shellcheck disable=SC2009  # pgrep cannot report elapsed time, which is the
# whole signal here; ps is the only source for it.
done < <(ps -axo pid=,etime=,command= | grep -E '/[a-z0-9_.-]+\.test( |$)' | grep -v grep || true)

if [ "${#stale_pids[@]}" -eq 0 ]; then
	exit 0
fi

cat >&2 <<EOF

Refusing to run: ${#stale_pids[@]} orphaned test binar$( [ "${#stale_pids[@]}" -eq 1 ] && echo y || echo ies ) still running.

Each has outlived its own -test.timeout, so it is stuck rather than busy, and
it is competing with the run you are about to start. Timings taken now would be
meaningless and a suite may appear to hang.

  PID   ELAPSED  PROCESS
$(printf '%s\n' "${stale_rows[@]}")

  kill -9 ${stale_pids[*]}

Set STALE_AFTER_MINUTES to change the ${STALE_AFTER_MINUTES}-minute threshold.
EOF
exit 1
