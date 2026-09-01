//go:build darwin

package dns_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/dns"
)

func TestGetDNSFromInterfaces(t *testing.T) {
	// This is mostly a smoke test to ensure the function doesn't panic.
	servers := dns.ExportGetDNSFromInterfaces()
	if servers == nil {
		t.Error("expected non-nil slice from GetDNSFromInterfaces")
	}
	// The function currently returns an empty slice as it's a placeholder.
	// This test verifies it runs without error.
}

func TestGetSystemDNSPlatformDarwin(t *testing.T) {
	servers := dns.GetSystemDNS()
	if servers == nil {
		t.Error("expected non-nil slice from GetSystemDNS")
	}
	// On darwin, we should get at least the servers from /etc/resolv.conf
	// if it exists. This test just ensures no panic.
}

func TestGetSystemDNSPlatformDarwinWithResolverDir(t *testing.T) {
	// Test reading from /etc/resolver directory if it exists.
	// This is a smoke test as we can't easily mock the filesystem.
	servers := dns.ExportGetSystemDNSPlatform()
	if servers == nil {
		t.Error("expected non-nil slice")
	}
}
