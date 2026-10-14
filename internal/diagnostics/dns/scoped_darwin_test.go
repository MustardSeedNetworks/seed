// SPDX-License-Identifier: BUSL-1.1

//go:build darwin

package dns_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

// The scoped table as the dynamic store yields it on a Mac with a wired link
// and a VPN tunnel up.
const scopedTable = "en0\t172.40.40.1\nen0\t172.40.40.2\nutun3\t10.9.0.1\n"

func TestScopedResolversForNamesOneInterface(t *testing.T) {
	got, _ := dns.ExportScopedResolversFor(scopedTable, "en0")
	if want := []string{"172.40.40.1", "172.40.40.2"}; !slices.Equal(got, want) {
		t.Errorf("en0 resolvers = %v, want %v", got, want)
	}
	tunnel, _ := dns.ExportScopedResolversFor(scopedTable, "utun3")
	if !slices.Equal(tunnel, []string{"10.9.0.1"}) {
		t.Errorf("utun3 resolvers = %v, want [10.9.0.1]", tunnel)
	}
}

// An interface with no scoped resolvers must come back empty and still
// attributable: falling back to the host's resolvers would name en0's under
// feth0, the way the gateway named the Wi-Fi router (#2690).
func TestScopedResolversForReportsAnInterfaceWithNone(t *testing.T) {
	servers, attributable := dns.ExportScopedResolversFor(scopedTable, "feth0")
	if !attributable {
		t.Error("a table with scoped resolvers is attributable even when it names no match")
	}
	if len(servers) != 0 {
		t.Errorf("feth0 resolvers = %v, want none", servers)
	}
}

// A store with no scoped resolver at all cannot attribute resolvers to an
// interface. Reporting that as "this interface has none" would put every
// interface, including the one carrying the route, behind the absence state.
func TestScopedResolversForSaysWhenItCannotTell(t *testing.T) {
	if _, attributable := dns.ExportScopedResolversFor("", "en0"); attributable {
		t.Error("an empty table must not be reported as attributable")
	}
}
