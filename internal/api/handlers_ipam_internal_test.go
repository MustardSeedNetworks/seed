package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/MustardSeedNetworks/seed/internal/alerts"
	"github.com/MustardSeedNetworks/seed/internal/app"
	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/discovery"
	discoverysettings "github.com/MustardSeedNetworks/seed/internal/discovery/settings"
	"github.com/MustardSeedNetworks/seed/internal/ipam"
	"github.com/MustardSeedNetworks/seed/internal/logging"
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
	s, db := newTopologyTestServer(t)
	_ = seedTopology(t, db)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, b := range []*topology.ARPBinding{
		{IPAddress: "10.44.20.1", MACAddress: "aa:bb:cc:00:00:01", LastSeen: now},
		{IPAddress: "10.44.20.7", MACAddress: "aa:bb:cc:00:00:07", LastSeen: now.Add(-time.Hour)},
		{IPAddress: "10.44.20.9", MACAddress: "aa:bb:cc:00:00:09", LastSeen: now.Add(-arpInUseWindow - time.Hour)},
		{IPAddress: "10.44.99.5", MACAddress: "aa:bb:cc:00:00:05", LastSeen: now},
	} {
		b.ClientID, b.SourceNodeID, b.IfIndex = "default", "node-test-1", 1
		require.NoError(t, db.Topology().UpsertARPBinding(ctx, b))
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

// The acceptance of P-D5's second half: an operator enters a DHCP scope, its
// count is a hand-count of the addresses seen inside it, and running out
// raises an alert through the alert write chain, once per climb.
func TestDHCPScopeExhaustionAlerts(t *testing.T) {
	s, db := newTopologyTestServer(t)
	_ = seedTopology(t, db)
	s.discoverySettings = discoverysettings.NewService(&fakeDiscoveryStore{}, noopSink{}, noopApplier{})
	s.useAlertDelivery(app.NewAlertDelivery(s.persist, nil, logging.GetLogger()))

	scopes := func(method, query, body string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		s.handleIPAMScopes(
			w,
			httptest.NewRequest(method, APIVersionPrefix+"/ipam/scopes"+query, strings.NewReader(body)),
		)
		return w
	}
	require.Equal(t, http.StatusBadRequest,
		scopes(http.MethodPost, "", `{"start":"10.44.20.109","end":"10.44.20.100"}`).Code)
	require.Equal(t, http.StatusOK,
		scopes(http.MethodPost, "", `{"name":"Lobby","start":"10.44.20.100","end":"10.44.20.109"}`).Code)
	require.Equal(t, http.StatusConflict,
		scopes(http.MethodPost, "", `{"start":"10.44.20.109","end":"10.44.20.120"}`).Code)

	devices := make([]*discovery.DiscoveredDevice, 0, 10)
	for host := 100; host <= 108; host++ {
		devices = append(devices, &discovery.DiscoveredDevice{IP: fmt.Sprintf("10.44.20.%d", host)})
	}
	devices = append(devices, &discovery.DiscoveredDevice{IP: "10.44.20.99"})

	s.afterSweep(t.Context(), devices)
	s.afterSweep(t.Context(), devices)
	require.NoError(t, db.Topology().UpsertARPBinding(t.Context(), &topology.ARPBinding{
		ClientID: "default", SourceNodeID: "node-test-1", IfIndex: 1,
		IPAddress: "10.44.20.109", MACAddress: "aa:bb:cc:00:01:09", LastSeen: time.Now().UTC(),
	}))
	s.afterSweep(t.Context(), devices)

	list, err := db.Alerts().List(t.Context(), alerts.ListOptions{Limit: 100})
	require.NoError(t, err)
	var raised []*alerts.Alert
	for _, a := range list {
		if a.Rule == alerts.RuleDHCPScopeExhaustion {
			raised = append(raised, a)
		}
	}
	slices.SortFunc(raised, func(a, b *alerts.Alert) int { return strings.Compare(a.Severity, b.Severity) })
	require.Len(t, raised, 2, "near once, then full once")
	require.Equal(t, alerts.SeverityCritical, raised[0].Severity)
	require.Equal(t, "DHCP scope Lobby (10.44.20.100-10.44.20.109) has no address left", raised[0].Title)
	require.Equal(t, alerts.SeverityWarning, raised[1].Severity)
	require.Equal(t, "DHCP scope Lobby (10.44.20.100-10.44.20.109) is 90% in use", raised[1].Title)
	var evidence alerts.DHCPScopeEvidence
	require.NoError(t, json.Unmarshal([]byte(raised[0].Metadata), &evidence))
	require.Equal(t, alerts.DHCPScopeEvidence{
		Start: "10.44.20.100", End: "10.44.20.109", Size: 10, Used: 10, UsedPercent: 100, Exhaustion: "full",
	}, evidence)

	w := httptest.NewRecorder()
	s.handleIPAMSubnets(w, httptest.NewRequest(http.MethodGet, APIVersionPrefix+"/ipam/subnets", http.NoBody))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp IPAMSubnetsResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, []IPAMScope{{
		Name: "Lobby", Start: "10.44.20.100", End: "10.44.20.109",
		Size: 10, Used: 1, Free: 9, UtilizationPct: 10, Exhaustion: ipam.ExhaustionNone,
		InUse: []string{"10.44.20.109"},
	}}, resp.Scopes, "the handler reads the registry, which holds no device here; the ARP entry still counts")

	require.Equal(t, http.StatusNotFound, scopes(http.MethodDelete, "?start=10.44.30.1", "").Code)
	require.Equal(t, http.StatusOK, scopes(http.MethodDelete, "?start=10.44.20.100", "").Code)
	require.Empty(t, s.discoverySettings.Scopes())
}
