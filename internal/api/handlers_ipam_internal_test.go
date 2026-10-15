package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/ipam"
	"github.com/MustardSeedNetworks/seed/internal/topology"
)

func TestIPAMNetworks(t *testing.T) {
	for name, tc := range map[string]struct {
		local   string
		targets []config.SubnetConfig
		want    []ipam.Network
	}{
		"the local subnet comes first and a dismissed learned network is left out": {
			local: "192.168.1.0/24",
			targets: []config.SubnetConfig{
				{CIDR: "10.44.10.0/24", Name: "Servers", Enabled: true},
				{CIDR: "10.44.20.0/24", Learned: true},
				{CIDR: "10.44.30.0/24", Learned: true, Decision: config.LearnedDismissed},
			},
			want: []ipam.Network{
				{Prefix: netip.MustParsePrefix("192.168.1.0/24"), Source: ipam.SourceLocal},
				{Prefix: netip.MustParsePrefix("10.44.10.0/24"), Name: "Servers", Source: ipam.SourceConfigured},
				{Prefix: netip.MustParsePrefix("10.44.20.0/24"), Source: ipam.SourceLearned},
			},
		},
		"no active interface and an unparsable target leave nothing": {
			targets: []config.SubnetConfig{{CIDR: "not-a-cidr"}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, ipamNetworks(tc.local, tc.targets))
		})
	}
}

func TestAddressesInUse(t *testing.T) {
	devices := []*discovery.DiscoveredDevice{
		{IP: "10.44.10.1", SNMPData: &discovery.SNMPFullData{
			IPAddresses: []discovery.SNMPIPAddress{{Address: "10.44.20.1", Prefix: 24}},
		}},
		{IP: ""},
	}
	bindings := []*topology.ARPBinding{{IPAddress: "10.44.20.7"}, {IPAddress: "garbage"}}

	require.Equal(t,
		[]netip.Addr{
			netip.MustParseAddr("10.44.10.1"),
			netip.MustParseAddr("10.44.20.1"),
			netip.MustParseAddr("10.44.20.7"),
		},
		addressesInUse(devices, bindings))
}

// The acceptance of P-D5: a learned /24 behind a router reports what a
// hand-count of the router's recent ARP table gives, and an entry older
// than the window is not counted.
func TestHandleIPAMSubnetsCountsRecentARPEntries(t *testing.T) {
	s := newTopologyTestServer(t)
	_ = seedTopology(t, s.db())
	ctx := context.Background()
	now := time.Now().UTC()
	for _, b := range []*topology.ARPBinding{
		{IPAddress: "10.44.20.1", MACAddress: "aa:bb:cc:00:00:01", LastSeen: now},
		{IPAddress: "10.44.20.7", MACAddress: "aa:bb:cc:00:00:07", LastSeen: now.Add(-time.Hour)},
		{IPAddress: "10.44.20.9", MACAddress: "aa:bb:cc:00:00:09", LastSeen: now.Add(-arpInUseWindow - time.Hour)},
		{IPAddress: "10.44.99.5", MACAddress: "aa:bb:cc:00:00:05", LastSeen: now},
	} {
		b.ClientID, b.SourceNodeID, b.IfIndex = "default", "node-test-1", 1
		require.NoError(t, s.db().Topology().UpsertARPBinding(ctx, b))
	}
	s.discoverySettings = discoverysettings.NewService(&fakeDiscoveryStore{cfg: config.NetworkDiscoveryConfig{
		TargetNetworks: []config.SubnetConfig{{CIDR: "10.44.20.0/24", Learned: true}},
	}}, noopSink{}, noopApplier{})

	w := httptest.NewRecorder()
	s.handleIPAMSubnets(w, httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/ipam/subnets", http.NoBody))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp IPAMSubnetsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, []IPAMSubnet{{
		CIDR:           "10.44.20.0/24",
		Source:         ipam.SourceLearned,
		Size:           254,
		Used:           2,
		Free:           252,
		UtilizationPct: float64(2) * 100 / 254,
		InUse:          []string{"10.44.20.1", "10.44.20.7"},
	}}, resp.Subnets)
}
