#!/bin/sh
#
# Run the seed daemon the E2E suite and the phone-width gate test against:
# a fresh config in RUN_DIR with discovery, health checks and every active
# probe off, listening on PORT. Stays in the foreground (it execs seed), so a
# caller backgrounds it and owns its lifetime.
#
#   scripts/e2e-daemon.sh RUN_DIR PORT
#
# scripts/run-e2e.sh starts it on a free port and reaps it. The phone-width
# job in ci.yml starts it on a fixed one, since the gate's base-url is a
# workflow input.
set -eu

if [ "$#" -ne 2 ]; then
  printf 'usage: %s RUN_DIR PORT\n' "$0" >&2
  exit 2
fi
run_dir=$1
port=$2
repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)

case $(uname -s) in
  Darwin) loopback=lo0 ;;
  *) loopback=lo ;;
esac
mkdir -p "$run_dir"
printf '%s\n' \
  "{\"server\":{\"port\":$port},\"interface\":{\"default\":\"$loopback\",\"fallbacks\":[],\"startup_retries\":0,\"startup_retry_wait\":0},\"networkDiscovery\":{\"enabled\":false,\"auto_scan\":false,\"options\":{\"passiveProtocols\":{\"lldp\":false,\"cdp\":false,\"edp\":false,\"ndp\":false},\"arpScan\":false,\"icmpScan\":false,\"portScan\":{\"enabled\":false},\"traceroute\":false,\"snmpQuery\":false},\"profiler\":{\"enabled\":false},\"ipv6_enabled\":false},\"healthChecks\":{\"ping_targets\":[],\"tcp_ports\":[],\"udp_ports\":[],\"http_endpoints\":[],\"rtsp_endpoints\":[],\"dicom_endpoints\":[],\"hl7_endpoints\":[],\"fhir_endpoints\":[],\"sql_endpoints\":[],\"fileshare_endpoints\":[],\"ldap_endpoints\":[],\"lti_endpoints\":[],\"opcua_endpoints\":[],\"modbus_endpoints\":[],\"run_performance\":false,\"run_speedtest\":false,\"run_iperf\":false,\"run_discovery\":false},\"iperf\":{\"enable_server\":false,\"auto_run_on_link\":false},\"fabOptions\":{\"run_health_checks\":false,\"run_network_discovery\":false,\"run_speedtest\":false,\"run_iperf\":false,\"run_performance\":false,\"auto_scan_on_link\":false},\"database\":{\"path\":\"$run_dir/seed.db\"},\"logging\":{\"file\":\"$run_dir/seed.log\"}}" \
  >"$run_dir/config.json"

# HOME is repointed at the run directory so the daemon cannot read the
# developer's own ~/.config/seed/.license (seed#2688). The licence manager
# resolves its state from os.UserHomeDir() and has no other override, so a
# machine that has ever run `seed license trial` was serving the suite an
# activated Pro licence — every RequireFeature page unlocked and the free-tier
# gate assertions in reports-page.spec.ts passed or failed by accident of the
# host. CI was green only because its runners have no licence file.
cd "$run_dir"
HOME="$run_dir" SEED_LOGIN_MAX_ATTEMPTS=200 exec "$repo_dir/seed" --config "$run_dir/config.json"
