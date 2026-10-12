package enumerate

import (
	"context"
	"net"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
)

// seed#2686: a /22 swept under the default one-/24 budget was a quarter
// covered, and only an Info log said so.
func TestPlanCoverageCountsTheHostsTheSweepProbes(t *testing.T) {
	t.Parallel()
	target := netip.MustParsePrefix("192.168.20.0/22")
	tests := []struct {
		name     string
		evidence []netip.Addr
		maxHosts int
		want     SweepCoverage
	}{
		{
			name:     "no evidence probes the first hosts of each /24",
			maxHosts: DefaultMaxHostsPerSubnet,
			want:     SweepCoverage{Subnet: "192.168.20.0/22", Swept: 4 * probeHostsPerBlock, Total: 1016},
		},
		{
			name:     "one evidenced /24 under a one-/24 budget",
			evidence: addrs(t, "192.168.22.7"),
			maxHosts: DefaultMaxHostsPerSubnet,
			want:     SweepCoverage{Subnet: "192.168.20.0/22", Swept: 254, Total: 1016},
		},
		{
			name:     "a budget covering every /24",
			maxHosts: 1016,
			want:     SweepCoverage{Subnet: "192.168.20.0/22", Swept: 1016, Total: 1016},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			plan := (&rotation{}).planSweep(target, tt.evidence, tt.maxHosts)
			require.Equal(t, tt.want, plan.coverage(target))
		})
	}
}

func TestPingSweepReportsPartialAndCompleteCoverage(t *testing.T) {
	requireAPingSweepIsPossible(t)
	tests := []struct {
		subnet string
		want   SweepCoverage
	}{
		{"127.0.4.0/22", SweepCoverage{Subnet: "127.0.4.0/22", Swept: probeHostsPerBlock, Total: 1016}},
		{"127.0.9.0/24", SweepCoverage{Subnet: "127.0.9.0/24", Swept: 254, Total: 254}},
	}
	for _, tt := range tests {
		t.Run(tt.subnet, func(t *testing.T) {
			s := NewARPScanner("lo", nil)
			// A budget of one probe block keeps the /22 sweep to four hosts.
			s.SetMaxHostsPerSubnet(probeHostsPerBlock)
			_, subnet, err := net.ParseCIDR(tt.subnet)
			require.NoError(t, err)

			_, got, err := s.pingSweep(context.Background(), subnet)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestServiceStatusCarriesTheLastSweepCoverage(t *testing.T) {
	t.Parallel()
	dd := NewDeviceDiscovery("lo")
	svc := NewService(&config.Config{}, dd, nil)

	require.Equal(t, []SweepCoverage{}, svc.GetStatus().SweepCoverage, "before the first sweep")

	swept := []SweepCoverage{
		{Subnet: "192.168.20.0/22", Swept: 254, Total: 1016},
		{Subnet: "10.0.0.0/24", Swept: 254, Total: 254},
	}
	dd.arpScanner.mu.Lock()
	dd.arpScanner.coverage = swept
	dd.arpScanner.mu.Unlock()

	require.Equal(t, swept, svc.GetStatus().SweepCoverage)
}
