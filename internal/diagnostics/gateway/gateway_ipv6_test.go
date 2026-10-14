package gateway_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/gateway"
)

// seed#3251: every link has its own fe80::/10, so a link-local gateway names a
// router only together with the interface its route leaves by.
func TestWithZone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, gateway, iface, want string
	}{
		{"a link-local IPv6 gateway takes the interface as its zone", "fe80::1", "eth0", "fe80::1%eth0"},
		{"a global IPv6 gateway needs no zone", "2001:db8::1", "eth0", "2001:db8::1"},
		{"an IPv4 link-local gateway has no zones", "169.254.0.1", "eth0", "169.254.0.1"},
		{"a zone already present is kept", "fe80::1%eth1", "eth0", "fe80::1%eth1"},
		{"no interface leaves the address alone", "fe80::1", "", "fe80::1"},
		{"a non-address is returned as given", "not-an-ip", "eth0", "not-an-ip"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := gateway.WithZone(tt.gateway, tt.iface); got != tt.want {
				t.Errorf("WithZone(%q, %q) = %q, want %q", tt.gateway, tt.iface, got, tt.want)
			}
		})
	}
}

// seed#3251: the tester pinged every gateway with an ICMPv4 socket, so the
// IPv6 card reported 100 % loss for a router that answers.
func TestTesterPingsAnIPv6Gateway(t *testing.T) {
	tester := gateway.NewTester(gateway.DefaultThresholds())
	defer tester.Close()
	if tester.TesterGetPinger6() == nil {
		t.Skip("this host allows no ICMPv6 socket to this process")
	}
	tester.SetGateway("::1")

	if result := tester.Ping(); !result.Success {
		t.Fatalf("Ping(::1) failed: %s", result.Error)
	}
}
