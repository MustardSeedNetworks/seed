package netif_test

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif"
)

// TestMockManagerWritesNeverReachTheHost pins #2530: every handler test builds
// its server on the mock manager, which used to fall through to the platform
// writes. On a Mac that ran `networksetup -setdhcp` against the developer's own
// network service and forked a child that could wedge; as an unprivileged user
// on Linux the same calls fail, which is what makes this test RED there.
func TestMockManagerWritesNeverReachTheHost(t *testing.T) {
	m := netif.NewMockManager(netif.DefaultMockConfig())

	writes := []struct {
		name  string
		write func() error
	}{
		{"dhcp", func() error { return m.ConfigureDHCP("eth0") }},
		{"mtu", func() error { return m.SetMTU("lo", 1500) }},
		{"static", func() error {
			return m.ConfigureStaticIP("eth0", &netif.StaticIPConfig{
				Address: "192.0.2.10",
				Netmask: "255.255.255.0",
				Gateway: "192.0.2.1",
				DNS:     []string{"192.0.2.1"},
			})
		}},
	}
	for _, w := range writes {
		t.Run(w.name, func(t *testing.T) {
			if err := w.write(); err != nil {
				t.Fatalf("mock %s write reached the platform: %v", w.name, err)
			}
		})
	}
}
