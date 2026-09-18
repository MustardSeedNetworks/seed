// SPDX-License-Identifier: BUSL-1.1

//go:build windows

package dns_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

const ipconfigFixture = `Windows IP Configuration

Ethernet adapter Ethernet 2:

   Description . . . . . . . . . . . : Intel(R) Ethernet
   DNS Servers . . . . . . . . . . . : 10.1.1.1
                                       10.1.1.2

Wireless LAN adapter Wi-Fi:

   Description . . . . . . . . . . . : Wireless-AC
   DNS Servers . . . . . . . . . . . : 192.168.0.1
`

func TestParseIPConfigAdapterDNSScopesToOneAdapter(t *testing.T) {
	got, ok := dns.ExportParseIPConfigAdapterDNS(ipconfigFixture, "Ethernet 2")
	if !ok {
		t.Fatalf("adapter Ethernet 2 not found in the fixture")
	}
	if want := []string{"10.1.1.1", "10.1.1.2"}; !slices.Equal(got, want) {
		t.Errorf("resolvers = %v, want %v", got, want)
	}
}

func TestParseIPConfigAdapterDNSDoesNotAttributeAnUnnamedAdapter(t *testing.T) {
	if _, ok := dns.ExportParseIPConfigAdapterDNS(ipconfigFixture, "Ethernet 9"); ok {
		t.Error("an adapter the output does not name must not be reported as attributable")
	}
}
