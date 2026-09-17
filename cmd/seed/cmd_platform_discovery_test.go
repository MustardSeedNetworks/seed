package main

import (
	"strings"
	"testing"
)

// seed#2674: an operator who reads `seed platform` must learn what the daemon
// does to their network before they visit Settings — which sweeps run by
// default, which stay opt-in, and that ICMP needs a privilege.
func TestPlatformDescribesTheDiscoveryDefaults(t *testing.T) {
	var out strings.Builder
	runPlatform(&out)
	got := out.String()

	for _, want := range []string{
		"DISCOVERY DEFAULTS",
		"LLDP, CDP and EDP",
		"ARP sweep",
		"ICMP sweep",
		"every 60s",
		"Off until you turn them on: full port scan, traceroute, SNMP",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("seed platform output is missing %q\n---\n%s", want, got)
		}
	}
}
