#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
run_dir=$(mktemp -d "${TMPDIR:-/tmp}/seed-e2e.XXXXXX")
server_log=${E2E_SERVER_LOG:-$run_dir/server.log}
server_pid=
exit_status=0

# Kill the whole process group the server leads, not just the server.
#
# seed#2420: the daemon forks system helpers (networksetup, ifconfig,
# system_profiler on darwin) while serving, and a child stuck before execve
# carries the parent's argv — so `pgrep -f 'seed --config'` counts it as another
# daemon. Killing only $server_pid leaves those children reparented to init,
# spinning; one bad afternoon left 2 056 of them and a load average of 971.
#
# The server is started under `set -m`, which makes it a process-group leader
# with pgid == $server_pid, so `kill -- -$server_pid` reaches it and everything
# it forked. TERM first, then KILL for anything that ignored it.
kill_group() {
  kill -TERM -- "-$1" 2>/dev/null || kill -TERM "$1" 2>/dev/null || return 0
  # Give the group a moment to go down cleanly before insisting.
  sleep 1
  kill -KILL -- "-$1" 2>/dev/null || true
}

cleanup() {
  exit_status=$?
  if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
    kill_group "$server_pid"
    wait "$server_pid" 2>/dev/null || true
  fi
  if [ "$exit_status" -ne 0 ] && [ -f "$server_log" ]; then
    printf '\nSeed E2E server log:\n' >&2
    tail -200 "$server_log" >&2
  fi
  rm -rf "$run_dir"
  trap - EXIT HUP INT TERM
  exit "$exit_status"
}
trap cleanup EXIT
trap 'exit 129' HUP
trap 'exit 130' INT
trap 'exit 143' TERM

cd "$repo_dir"

if [ "${E2E_SKIP_BUILD:-0}" != 1 ]; then
  make --no-print-directory build-frontend-quiet
  make --no-print-directory build-backend-quiet
fi

if [ ! -x ./seed ]; then
  printf '%s\n' 'seed binary is missing; run without E2E_SKIP_BUILD or build it first' >&2
  exit 1
fi

# Pre-flight orphan report (seed#2420). Deliberately a warning, not a refusal:
# CI runs four shards at once and a developer may legitimately have a sibling
# session's daemon up, so refusing here would break both. It is also why the
# remedy printed below is scoped to a run directory rather than
# `pkill -f 'seed --config'` — a blanket pkill has already killed another
# session's server mid-investigation.
stray=$(pgrep -f 'seed --config' 2>/dev/null | wc -l | tr -d ' ')
if [ "${stray:-0}" -gt 0 ]; then
  printf 'note: %s process(es) already match "seed --config".\n' "$stray" >&2
  printf '      Some may be another session. To clear only a known run:\n' >&2
  printf '        pkill -f "seed --config /tmp/seed-e2e.<id>/config.json"\n' >&2
fi

port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
case $(uname -s) in
  Darwin) loopback=lo0 ;;
  *) loopback=lo ;;
esac
printf '%s\n' \
  "{\"server\":{\"port\":$port},\"interface\":{\"default\":\"$loopback\",\"fallbacks\":[],\"startup_retries\":0,\"startup_retry_wait\":0},\"networkDiscovery\":{\"enabled\":false,\"auto_scan\":false,\"options\":{\"passiveProtocols\":{\"lldp\":false,\"cdp\":false,\"edp\":false,\"ndp\":false},\"arpScan\":false,\"icmpScan\":false,\"portScan\":{\"enabled\":false},\"traceroute\":false,\"snmpQuery\":false},\"profiler\":{\"enabled\":false},\"ipv6_enabled\":false},\"healthChecks\":{\"ping_targets\":[],\"tcp_ports\":[],\"udp_ports\":[],\"http_endpoints\":[],\"rtsp_endpoints\":[],\"dicom_endpoints\":[],\"hl7_endpoints\":[],\"fhir_endpoints\":[],\"sql_endpoints\":[],\"fileshare_endpoints\":[],\"ldap_endpoints\":[],\"lti_endpoints\":[],\"opcua_endpoints\":[],\"modbus_endpoints\":[],\"run_performance\":false,\"run_speedtest\":false,\"run_iperf\":false,\"run_discovery\":false},\"iperf\":{\"enable_server\":false,\"auto_run_on_link\":false},\"fabOptions\":{\"run_health_checks\":false,\"run_network_discovery\":false,\"run_speedtest\":false,\"run_iperf\":false,\"run_performance\":false,\"auto_scan_on_link\":false},\"database\":{\"path\":\"$run_dir/seed.db\"},\"logging\":{\"file\":\"$run_dir/seed.log\"}}" \
  >"$run_dir/config.json"

# Job control, so the subshell below becomes a process-group leader and cleanup
# can reap its children as well as itself. Turned off again immediately: with
# `set -m` left on, this script's own foreground commands get their own groups
# and stop inheriting the terminal's signals as expected.
set -m
(
  cd "$run_dir"
  SEED_LOGIN_MAX_ATTEMPTS=200 exec "$repo_dir/seed" --config "$run_dir/config.json"
) >"$server_log" 2>&1 &
server_pid=$!
set +m

base_url=
attempt=0
while [ "$attempt" -lt 120 ]; do
  offset=0
  while [ "$offset" -le 9 ]; do
    candidate_port=$((port + offset))
    if curl -skf "https://127.0.0.1:$candidate_port/__version" >/dev/null 2>&1; then
      base_url="https://127.0.0.1:$candidate_port"
      break 2
    fi
    offset=$((offset + 1))
  done
  if ! kill -0 "$server_pid" 2>/dev/null; then
    wait "$server_pid" || true
    printf '%s\n' 'Seed exited before becoming ready' >&2
    exit 1
  fi
  attempt=$((attempt + 1))
  sleep 0.25
done

if [ -z "$base_url" ]; then
	printf '%s\n' 'Seed did not become ready within 30 seconds' >&2
	exit 1
fi

plain_url="http://${base_url#https://}"
if curl -sf --max-time 2 "$plain_url/__version" >/dev/null 2>&1; then
	printf '%s\n' "Seed served application content over plaintext HTTP at $plain_url" >&2
	exit 1
fi

cd "$repo_dir/ui"
E2E_BASE_URL="$base_url" \
PLAYWRIGHT_IGNORE_HTTPS_ERRORS=true \
  node --disable-warning=DEP0205 ./node_modules/playwright/cli.js test "$@"
