// SPDX-License-Identifier: BUSL-1.1

//go:build linux

package dns_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

func TestParseResolvedLinkDNSReadsBothSpellings(t *testing.T) {
	const content = `# This is private data. Do not parse.
LLMNR=yes
DNS=192.168.1.1
DNS=192.168.1.2 192.168.1.3
DOMAINS=lan
`
	got := dns.ExportParseResolvedLinkDNS(content)
	want := []string{"192.168.1.1", "192.168.1.2", "192.168.1.3"}
	if !slices.Equal(got, want) {
		t.Errorf("link resolvers = %v, want %v", got, want)
	}
}

// systemd 259 dropped the per-link files, so on a current host the resolvers
// come from resolved's DumpDNSConfiguration varlink reply. The reply below is
// trimmed from dev-srv-ubuntu (systemd 259.5): ens18 holds the lab resolver
// and docker0 none.
func TestParseDNSConfiguration(t *testing.T) {
	const reply = `{"parameters":{"configuration":[` +
		`{"dnssec":"no","scopes":[{"protocol":"dns"}]},` +
		`{"ifname":"ens18","ifindex":2,"defaultRoute":true,"servers":[` +
		`{"addressString":"10.44.40.10","address":[10,44,40,10],"family":2,"port":53,"ifindex":2}]},` +
		`{"ifname":"docker0","ifindex":5,"defaultRoute":false}]}}`

	tests := []struct {
		iface string
		want  []string
	}{
		{"ens18", []string{"10.44.40.10"}},
		{"docker0", nil},
		{"absent0", nil},
	}
	for _, tc := range tests {
		got, err := dns.ExportParseDNSConfiguration([]byte(reply), tc.iface)
		if err != nil || !slices.Equal(got, tc.want) {
			t.Errorf("%s: resolvers = %v, %v; want %v", tc.iface, got, err, tc.want)
		}
	}
}

// An older resolved answers the call with a varlink error; that is not an
// interface without resolvers, so the caller must go on to the per-link files.
func TestParseDNSConfigurationReportsAVarlinkError(t *testing.T) {
	const reply = `{"error":"org.varlink.service.MethodNotFound","parameters":{"method":"io.systemd.Resolve.DumpDNSConfiguration"}}`
	if _, err := dns.ExportParseDNSConfiguration([]byte(reply), "ens18"); err == nil {
		t.Fatal("a varlink error reply parsed as a configuration")
	}
}
