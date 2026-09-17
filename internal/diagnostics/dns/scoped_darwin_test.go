// SPDX-License-Identifier: BUSL-1.1

//go:build darwin

package dns_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

// scutil --dns as this Mac prints it: the first block is the unscoped
// configuration, which is exactly the answer that must NOT be given for an
// interface that has no resolvers of its own.
const scutilFixture = `DNS configuration

resolver #1
  search domain[0] : example.test
  nameserver[0] : 172.40.40.1
  if_index : 12 (en0)
  flags    : Request A records
  reach    : 0x00020002 (Reachable,Directly Reachable Address)

resolver #2
  domain   : local
  options  : mdns
  timeout  : 5

DNS configuration (for scoped queries)

resolver #1
  search domain[0] : example.test
  nameserver[0] : 172.40.40.1
  nameserver[1] : 172.40.40.2
  if_index : 12 (en0)
  flags    : Scoped, Request A records

resolver #2
  nameserver[0] : 10.9.0.1
  if_index : 15 (utun3)
  flags    : Scoped, Request A records
`

func TestParseScutilScopedResolversNamesOneInterface(t *testing.T) {
	got := dns.ExportParseScutilScopedResolvers(scutilFixture, "en0")
	want := []string{"172.40.40.1", "172.40.40.2"}
	if !slices.Equal(got, want) {
		t.Errorf("en0 resolvers = %v, want %v", got, want)
	}
	tunnel := dns.ExportParseScutilScopedResolvers(scutilFixture, "utun3")
	if !slices.Equal(tunnel, []string{"10.9.0.1"}) {
		t.Errorf("utun3 resolvers = %v, want [10.9.0.1]", tunnel)
	}
}

// An interface with no scoped resolvers must come back empty. Falling through
// to the unscoped block is the defect in a new coat: it would name en0's
// resolvers under feth0, the way the gateway named the Wi-Fi router (#2690).
func TestParseScutilScopedResolversDoesNotFallBackToTheUnscopedBlock(t *testing.T) {
	if got := dns.ExportParseScutilScopedResolvers(scutilFixture, "feth0"); len(got) != 0 {
		t.Errorf("feth0 resolvers = %v, want none", got)
	}
}
