package ipam_test

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/ipam"
)

func addrs(in ...string) []netip.Addr {
	out := make([]netip.Addr, 0, len(in))
	for _, s := range in {
		out = append(out, netip.MustParseAddr(s))
	}
	return out
}

func network(cidr string, source ipam.Source) ipam.Network {
	return ipam.Network{Prefix: netip.MustParsePrefix(cidr), Source: source}
}

func TestUtilization(t *testing.T) {
	for name, tc := range map[string]struct {
		networks []ipam.Network
		inUse    []netip.Addr
		want     []ipam.Usage
	}{
		"a /24 counts hosts and skips network and broadcast": {
			networks: []ipam.Network{network("10.44.10.0/24", ipam.SourceLocal)},
			inUse:    addrs("10.44.10.0", "10.44.10.1", "10.44.10.20", "10.44.10.255", "10.44.11.5"),
			want: []ipam.Usage{{
				Network: network("10.44.10.0/24", ipam.SourceLocal),
				Size:    254,
				InUse:   addrs("10.44.10.1", "10.44.10.20"),
			}},
		},
		"an address seen twice is counted once, in address order": {
			networks: []ipam.Network{network("192.168.1.0/28", ipam.SourceConfigured)},
			inUse:    addrs("192.168.1.9", "192.168.1.2", "192.168.1.9"),
			want: []ipam.Usage{{
				Network: network("192.168.1.0/28", ipam.SourceConfigured),
				Size:    14,
				InUse:   addrs("192.168.1.2", "192.168.1.9"),
			}},
		},
		"a /31 and a /32 have no network or broadcast address": {
			networks: []ipam.Network{
				network("10.0.0.0/31", ipam.SourceLearned),
				network("10.0.0.9/32", ipam.SourceLearned),
			},
			inUse: addrs("10.0.0.0", "10.0.0.1", "10.0.0.9"),
			want: []ipam.Usage{
				{Network: network("10.0.0.0/31", ipam.SourceLearned), Size: 2, InUse: addrs("10.0.0.0", "10.0.0.1")},
				{Network: network("10.0.0.9/32", ipam.SourceLearned), Size: 1, InUse: addrs("10.0.0.9")},
			},
		},
		"a network listed twice keeps its first source": {
			networks: []ipam.Network{
				network("10.1.0.0/24", ipam.SourceLocal),
				network("10.1.0.7/24", ipam.SourceConfigured),
			},
			want: []ipam.Usage{{Network: network("10.1.0.0/24", ipam.SourceLocal), Size: 254}},
		},
		"overlapping networks are each counted in full": {
			networks: []ipam.Network{
				network("172.16.0.0/16", ipam.SourceConfigured),
				network("172.16.5.0/24", ipam.SourceLearned),
			},
			inUse: addrs("172.16.5.10", "172.16.9.1"),
			want: []ipam.Usage{
				{Network: network("172.16.0.0/16", ipam.SourceConfigured), Size: 65534, InUse: addrs("172.16.5.10", "172.16.9.1")},
				{Network: network("172.16.5.0/24", ipam.SourceLearned), Size: 254, InUse: addrs("172.16.5.10")},
			},
		},
		"IPv6 networks are skipped": {
			networks: []ipam.Network{network("2001:db8::/64", ipam.SourceConfigured)},
			inUse:    addrs("2001:db8::1"),
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, ipam.Utilization(tc.networks, tc.inUse))
		})
	}
}

func TestUsageFreeAndPercent(t *testing.T) {
	usage := ipam.Usage{Size: 254, InUse: addrs("10.0.0.1", "10.0.0.2")}
	require.Equal(t, uint64(252), usage.Free())
	require.InDelta(t, 0.787, usage.Percent(), 0.001)
}
