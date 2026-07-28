#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
run_dir=$(mktemp -d "${TMPDIR:-/tmp}/seed-e2e.XXXXXX")
server_log=${E2E_SERVER_LOG:-$run_dir/server.log}
server_pid=
exit_status=0

cleanup() {
  exit_status=$?
  if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
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

port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
case $(uname -s) in
  Darwin) loopback=lo0 ;;
  *) loopback=lo ;;
esac
printf '%s\n' \
  "{\"server\":{\"port\":$port},\"interface\":{\"default\":\"$loopback\",\"fallbacks\":[],\"startup_retries\":0,\"startup_retry_wait\":0},\"networkDiscovery\":{\"enabled\":false,\"auto_scan\":false,\"options\":{\"passiveProtocols\":{\"lldp\":false,\"cdp\":false,\"edp\":false,\"ndp\":false},\"arpScan\":false,\"icmpScan\":false,\"portScan\":{\"enabled\":false},\"traceroute\":false,\"snmpQuery\":false},\"profiler\":{\"enabled\":false},\"ipv6_enabled\":false},\"healthChecks\":{\"ping_targets\":[],\"tcp_ports\":[],\"udp_ports\":[],\"http_endpoints\":[],\"rtsp_endpoints\":[],\"dicom_endpoints\":[],\"hl7_endpoints\":[],\"fhir_endpoints\":[],\"sql_endpoints\":[],\"fileshare_endpoints\":[],\"ldap_endpoints\":[],\"lti_endpoints\":[],\"opcua_endpoints\":[],\"modbus_endpoints\":[],\"run_performance\":false,\"run_speedtest\":false,\"run_iperf\":false,\"run_discovery\":false},\"iperf\":{\"enable_server\":false,\"auto_run_on_link\":false},\"fabOptions\":{\"run_health_checks\":false,\"run_network_discovery\":false,\"run_speedtest\":false,\"run_iperf\":false,\"run_performance\":false,\"auto_scan_on_link\":false},\"database\":{\"path\":\"$run_dir/seed.db\"},\"logging\":{\"file\":\"$run_dir/seed.log\"}}" \
  >"$run_dir/config.json"

(
  cd "$run_dir"
  SEED_LOGIN_MAX_ATTEMPTS=200 exec "$repo_dir/seed" --config "$run_dir/config.json"
) >"$server_log" 2>&1 &
server_pid=$!

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
