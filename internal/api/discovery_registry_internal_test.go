// SPDX-License-Identifier: BUSL-1.1

package api

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/config"
	"github.com/MustardSeedNetworks/seed/internal/netif"
	"github.com/MustardSeedNetworks/seed/internal/testutil"
)

// seed#2831: the server built a second device registry inside the discovery
// service, so the 60 s rescan swept a registry no target network ever reached
// and GET /security/devices listed a registry the rescan never filled. There is
// one registry: the service sweeps the one the API lists and the settings feed.
func TestDiscoveryServiceSweepsTheRegistryTheAPIServes(t *testing.T) {
	dir := t.TempDir()
	cfg := testutil.NewConfigBuilder().WithPort(8080).Build()
	s := NewServer(cfg, filepath.Join(dir, "seed.json"), "",
		netif.NewMockManager(netif.DefaultMockConfig()), false, nil, nil, nil)
	t.Cleanup(s.Close)

	swept := s.discoveryService().DeviceDiscovery()
	if swept != s.deviceDiscovery() {
		t.Fatal("the discovery service sweeps a different registry from the one GET /security/devices lists")
	}

	const site = "10.51.200.0/24"
	if err := s.discoverySettings.AddSubnet(config.SubnetConfig{CIDR: site, Enabled: true}); err != nil {
		t.Fatalf("AddSubnet: %v", err)
	}
	if got := swept.GetTargetNetworks(); !slices.Contains(got, site) {
		t.Errorf("rescan target networks = %v, want the enabled %s", got, site)
	}
}
