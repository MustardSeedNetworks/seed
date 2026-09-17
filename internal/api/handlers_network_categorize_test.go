package api_test

// The UI reads interfaces only through GET /api/v1/interfaces?categorized=true
// (ui/src/hooks/useNetworkFetchers.ts) and renders the ethernet and wifi lists
// it returns. An interface whose type is neither — feth0 on macOS, and equally
// bond0, ppp0 or usb0, all of which netif's selection will happily choose —
// therefore reached no list at all, so seed#2692's active interface was absent
// from the only surface an operator can select one on, while currentInterface
// named it.

import (
	"slices"
	"testing"

	api "github.com/MustardSeedNetworks/seed/internal/api"
	"github.com/MustardSeedNetworks/seed/internal/netif"
)

func categorizeFixture() []*netif.InterfaceInfo {
	return []*netif.InterfaceInfo{
		{Name: "eth0", Type: netif.InterfaceTypeEthernet, Up: true, Score: 10},
		{Name: "wlan0", Type: netif.InterfaceTypeWiFi, Up: true, Score: 5},
		{Name: "feth0", Type: netif.InterfaceTypeOther, Up: true, Score: 1},
	}
}

func ifaceNames(ifaces []api.InterfaceInfo) []string {
	out := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		out = append(out, iface.Name)
	}
	slices.Sort(out)
	return out
}

// TestCategorizeInterfacesPlacesTheSelectedOtherInterface is seed#2692 on the
// surface the UI actually reads: feth0 is the active interface and has to be
// selectable. It joins the ethernet list because that is where netif's own
// selection puts it — collectCandidates scores InterfaceTypeOther with a
// routable address as an ethernet-class candidate.
func TestCategorizeInterfacesPlacesTheSelectedOtherInterface(t *testing.T) {
	resp := api.ExportCategorizeInterfaces(categorizeFixture(), "feth0")

	if got, want := ifaceNames(resp.Ethernet), []string{"eth0", "feth0"}; !slices.Equal(got, want) {
		t.Errorf("ethernet = %v, want %v", got, want)
	}
	if got, want := ifaceNames(resp.WiFi), []string{"wlan0"}; !slices.Equal(got, want) {
		t.Errorf("wifi = %v, want %v", got, want)
	}
	if resp.CurrentInterface != "feth0" {
		t.Errorf("currentInterface = %q, want feth0", resp.CurrentInterface)
	}
	if resp.CurrentType != string(netif.InterfaceTypeOther) {
		t.Errorf("currentType = %q, want %q", resp.CurrentType, netif.InterfaceTypeOther)
	}
}

// TestCategorizeInterfacesDropsAnUnselectedOtherInterface keeps the filter
// honest: the fix is "list what we are bound to", not "list everything".
func TestCategorizeInterfacesDropsAnUnselectedOtherInterface(t *testing.T) {
	resp := api.ExportCategorizeInterfaces(categorizeFixture(), "eth0")

	if got, want := ifaceNames(resp.Ethernet), []string{"eth0"}; !slices.Equal(got, want) {
		t.Errorf("ethernet = %v, want %v", got, want)
	}
	if got, want := ifaceNames(resp.WiFi), []string{"wlan0"}; !slices.Equal(got, want) {
		t.Errorf("wifi = %v, want %v", got, want)
	}
	if resp.CurrentType != string(netif.InterfaceTypeEthernet) {
		t.Errorf("currentType = %q, want %q", resp.CurrentType, netif.InterfaceTypeEthernet)
	}
}

// TestCategorizeInterfacesRecommendsByScore pins the #756 recommendation, and
// that a selected non-ethernet interface does not displace a better real one.
func TestCategorizeInterfacesRecommendsByScore(t *testing.T) {
	ifaces := append(categorizeFixture(),
		&netif.InterfaceInfo{Name: "eth1", Type: netif.InterfaceTypeEthernet, Up: true, Score: 42},
		&netif.InterfaceInfo{Name: "eth2", Type: netif.InterfaceTypeEthernet, Up: false, Score: 99},
	)

	resp := api.ExportCategorizeInterfaces(ifaces, "feth0")

	if resp.RecommendedEthernet != "eth1" {
		t.Errorf("recommendedEthernet = %q, want eth1 (highest score that is up)",
			resp.RecommendedEthernet)
	}
	if resp.RecommendedWiFi != "wlan0" {
		t.Errorf("recommendedWifi = %q, want wlan0", resp.RecommendedWiFi)
	}
}

// TestCategorizeInterfacesNoSelection covers a daemon that has not resolved an
// interface yet: currentType is empty rather than guessed, and no list gains a
// member from the empty name.
func TestCategorizeInterfacesNoSelection(t *testing.T) {
	resp := api.ExportCategorizeInterfaces(categorizeFixture(), "")

	if resp.CurrentType != "" {
		t.Errorf("currentType = %q, want empty", resp.CurrentType)
	}
	if got, want := ifaceNames(resp.Ethernet), []string{"eth0"}; !slices.Equal(got, want) {
		t.Errorf("ethernet = %v, want %v", got, want)
	}
	if got, want := ifaceNames(resp.WiFi), []string{"wlan0"}; !slices.Equal(got, want) {
		t.Errorf("wifi = %v, want %v", got, want)
	}
}

// TestCategorizeInterfacesSkipsNilEntries guards the loop against a nil the
// listing should never contain but toInterfaceInfos already tolerates.
func TestCategorizeInterfacesSkipsNilEntries(t *testing.T) {
	resp := api.ExportCategorizeInterfaces(
		[]*netif.InterfaceInfo{nil, {Name: "eth0", Type: netif.InterfaceTypeEthernet, Up: true}},
		"eth0",
	)

	if got, want := ifaceNames(resp.Ethernet), []string{"eth0"}; !slices.Equal(got, want) {
		t.Errorf("ethernet = %v, want %v", got, want)
	}
}
