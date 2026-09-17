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
