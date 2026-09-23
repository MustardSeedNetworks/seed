#!/bin/sh
#
# First-run discovery against a NIAC scenario over a virtual link (seed#2674).
#
#   scripts/e2e-niac-link.sh [TEMPLATE [SEED_ADDRESS/PREFIX]]
#
# Defaults: the `home-network` template (nine devices on 192.168.1.0/24) and
# 192.168.1.250/24 for seed. Linux only; needs `niac` on PATH, sudo for the
# network namespace, the veth pair and the two daemons, and the UI's
# node_modules for Playwright.
#
# NIAC runs alone in a network namespace on one end of a veth pair. Seed runs
# on the other end with a config that names only its interface, port and file
# paths: every discovery setting is the compiled-in default, which is the
# thing under test. ui/e2e/first-run-niac-link.spec.ts then asserts that
# Network and Security list every scenario device within 90 s of seed
# starting, and this script asserts the service log names the default
# discovery methods.
set -eu

template=${1:-home-network}
seed_cidr=${2:-192.168.1.250/24}
repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)

if [ "$(uname -s)" != Linux ]; then
  printf '%s\n' 'e2e-niac-link needs Linux network namespaces' >&2
  exit 2
fi
command -v niac >/dev/null || {
  printf '%s\n' 'niac is not on PATH; install a NIAC release first' >&2
  exit 2
}

run_dir=$(mktemp -d "${TMPDIR:-/tmp}/seed-niac-link.XXXXXX")
netns=seed-e2e-$$
seed_link=sdl$$
niac_link=nil$$
niac_pid=
seed_pid=
exit_status=0

# Each daemon starts under setsid, so it leads its own process group and
# cleanup can reach everything it forked (seed#2420). Not `set -m`: dash
# turns job control off when there is no terminal, which is every CI runner
# and every ssh session, and the daemons then share this script's group.
kill_group() {
  sudo kill -TERM -- "-$1" 2>/dev/null || return 0
  sleep 1
  sudo kill -KILL -- "-$1" 2>/dev/null || true
}

cleanup() {
  exit_status=$?
  for pid in $seed_pid $niac_pid; do
    kill_group "$pid"
  done
  for pid in $seed_pid $niac_pid; do
    wait "$pid" 2>/dev/null || true
  done
  sudo ip netns del "$netns" 2>/dev/null || true
  sudo ip link del "$seed_link" 2>/dev/null || true
  if [ "$exit_status" -ne 0 ]; then
    for log in niac.log seed.log; do
      if [ -f "$run_dir/$log" ]; then
        printf '\n%s (last 60 lines):\n' "$log" >&2
        tail -60 "$run_dir/$log" >&2
      fi
    done
  fi
  sudo rm -rf "$run_dir"
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
[ -x ./seed ] || {
  printf '%s\n' 'seed binary is missing; run without E2E_SKIP_BUILD or build it first' >&2
  exit 1
}

niac template use "$template" "$run_dir/scenario.yaml" >/dev/null

# The addresses the spec expects: each device's first address inside seed's
# subnet. A device with none there (a second interface on another network)
# is not on this link and is not expected.
expected_ips=$(python3 - "$run_dir/scenario.yaml" "$seed_cidr" <<'EOF'
import ipaddress
import re
import sys

path, seed_cidr = sys.argv[1], sys.argv[2]
subnet = ipaddress.ip_interface(seed_cidr).network
devices = []
for line in open(path, encoding="utf-8"):
    if re.match(r"^  - name:", line):
        devices.append([])
    match = re.match(r'^\s+- "?(\d+\.\d+\.\d+\.\d+)"?\s*$', line)
    if match and devices:
        devices[-1].append(match.group(1))
found = []
for addresses in devices:
    local = [a for a in addresses if ipaddress.ip_address(a) in subnet]
    if local:
        found.append(local[0])
if not found:
    sys.exit(f"no device in {path} has an address in {subnet}")
print(",".join(found))
EOF
)

sudo ip netns add "$netns"
sudo ip link add "$seed_link" type veth peer name "$niac_link"
sudo ip link set "$niac_link" netns "$netns"
sudo ip netns exec "$netns" ip link set lo up
sudo ip netns exec "$netns" ip link set "$niac_link" up
sudo ip addr add "$seed_cidr" dev "$seed_link"
sudo ip link set "$seed_link" up

setsid sudo ip netns exec "$netns" niac daemon --once "$niac_link" "$run_dir/scenario.yaml" \
  --storage disabled --cert-dir "$run_dir/niac-certs" >"$run_dir/niac.log" 2>&1 &
niac_pid=$!

attempt=0
until grep -q 'Simulation started' "$run_dir/niac.log" 2>/dev/null; do
  attempt=$((attempt + 1))
  if [ "$attempt" -gt 60 ] || ! kill -0 "$niac_pid" 2>/dev/null; then
    printf '%s\n' 'NIAC did not start its simulation within 30 seconds' >&2
    exit 1
  fi
  sleep 0.5
done

port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
printf '%s\n' \
  "{\"server\":{\"port\":$port},\"interface\":{\"default\":\"$seed_link\",\"fallbacks\":[],\"startup_retries\":0,\"startup_retry_wait\":0},\"database\":{\"path\":\"$run_dir/seed.db\"},\"logging\":{\"file\":\"$run_dir/seed.log\"}}" \
  >"$run_dir/config.json"

# Root for the raw sockets the sweep uses; HOME repointed so the host's own
# licence file cannot change the tier under test (seed#2688).
started_at=$(python3 -c 'import time; print(int(time.time() * 1000))')
setsid sudo env HOME="$run_dir" SEED_LOGIN_MAX_ATTEMPTS=200 \
  "$repo_dir/seed" --config "$run_dir/config.json" >"$run_dir/seed.out" 2>&1 &
seed_pid=$!

base_url=
attempt=0
while [ "$attempt" -lt 120 ]; do
  offset=0
  while [ "$offset" -le 9 ]; do
    if curl -skf "https://127.0.0.1:$((port + offset))/__version" >/dev/null 2>&1; then
      base_url="https://127.0.0.1:$((port + offset))"
      break 2
    fi
    offset=$((offset + 1))
  done
  kill -0 "$seed_pid" 2>/dev/null || {
    printf '%s\n' 'Seed exited before becoming ready' >&2
    exit 1
  }
  attempt=$((attempt + 1))
  sleep 0.25
done
[ -n "$base_url" ] || {
  printf '%s\n' 'Seed did not become ready within 30 seconds' >&2
  exit 1
}

cd "$repo_dir/ui"
SEED_E2E_NIAC_LINK=1 \
SEED_E2E_STARTED_AT="$started_at" \
SEED_E2E_EXPECTED_IPS="$expected_ips" \
E2E_BASE_URL="$base_url" \
PLAYWRIGHT_IGNORE_HTTPS_ERRORS=true \
  node --disable-warning=DEP0205 ./node_modules/playwright/cli.js test \
  e2e/first-run-niac-link.spec.ts --project=chromium

methods=$(sudo grep -m1 'Starting discovery service' "$run_dir/seed.log" || true)
for method in lldp cdp arp icmp; do
  case $methods in
    *"$method"*) ;;
    *)
      printf 'the service log does not list the default method %s:\n  %s\n' "$method" "$methods" >&2
      exit 1
      ;;
  esac
done
printf 'service log: %s\n' "$methods"
