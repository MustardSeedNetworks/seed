//go:build linux || darwin

package api_test

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

// TestGetSystemDNSDelegatesToTheResolverReader covers the DNS fallback the IP
// config response uses when a DHCP lease carries no servers of its own (#93).
// The fallback was a stub returning an empty slice regardless of how the host
// was configured, so that path reported no DNS servers at all while a fully
// implemented platform reader sat unused in internal/diagnostics/dns.
//
// The reader itself is covered against fixtures in that package; what this
// asserts is the delegation, which is where the bug was.
func TestGetSystemDNSDelegatesToTheResolverReader(t *testing.T) {
	want := dns.GetSystemDNS()
	if len(want) == 0 {
		t.Skip("this host has no resolver configuration, so there is no delegation to observe")
	}

	got := api.ExportGetSystemDNS()

	if !slices.Equal(got, want) {
		t.Errorf("getSystemDNS() = %v, want %v (the platform reader's result)", got, want)
	}
}
