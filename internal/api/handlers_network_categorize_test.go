package api

// The UI reads interfaces only through GET /api/v1/interfaces?categorized=true
// (ui/src/hooks/useNetworkFetchers.ts) and renders the ethernet and wifi lists
// it returns. An interface whose type is neither — feth0 on macOS, and equally
// bond0, ppp0 or usb0, all of which netif's selection will happily choose —
// therefore reached no list at all, so seed#2692's active interface was absent
// from the only surface an operator can select one on, while currentInterface
// named it.

import (
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif"
)

func categorizeFixture() []*netif.InterfaceInfo {
	return []*netif.InterfaceInfo{
		{Name: "eth0", Type: netif.InterfaceTypeEthernet, Up: true, Score: 10},
		{Name: "wlan0", Type: netif.InterfaceTypeWiFi, Up: true, Score: 5},
		{Name: "feth0", Type: netif.InterfaceTypeOther, Up: true, Score: 1},
	}
}

func names(ifaces []InterfaceInfo) []string {
	out := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		out = append(out, iface.Name)
	}
	return out
}

func contains(ifaces []InterfaceInfo, name string) bool {
	for _, iface := range ifaces {
		if iface.Name == name {
			return true
		}
	}
	return false
}

// TestCategorizeInterfacesPlacesTheSelectedOtherInterface is seed#2692 on the
// surface the UI actually reads: feth0 is the active interface and has to be
// selectable. It joins the ethernet list because that is where netif's own
// selection puts it — collectCandidates scores InterfaceTypeOther with a
// routable address as an ethernet-class candidate.
func TestCategorizeInterfacesPlacesTheSelectedOtherInterface(t *testing.T) {
	resp := categorizeInterfaces(categorizeFixture(), "feth0")

	if !contains(resp.Ethernet, "feth0") {
		t.Errorf("ethernet = %v, want it to contain feth0", names(resp.Ethernet))
	}
	if contains(resp.WiFi, "feth0") {
		t.Errorf("wifi = %v, want feth0 absent", names(resp.WiFi))
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
	resp := categorizeInterfaces(categorizeFixture(), "eth0")

	if contains(resp.Ethernet, "feth0") || contains(resp.WiFi, "feth0") {
		t.Errorf("ethernet = %v, wifi = %v, want feth0 in neither",
			names(resp.Ethernet), names(resp.WiFi))
	}
	if resp.CurrentType != string(netif.InterfaceTypeEthernet) {
		t.Errorf("currentType = %q, want %q", resp.CurrentType, netif.InterfaceTypeEthernet)
	}
}

// TestCategorizeInterfacesRecommendsByScore pins the #756 recommendation the
// extraction has to carry over unchanged.
func TestCategorizeInterfacesRecommendsByScore(t *testing.T) {
	ifaces := append(categorizeFixture(),
		&netif.InterfaceInfo{Name: "eth1", Type: netif.InterfaceTypeEthernet, Up: true, Score: 42},
		&netif.InterfaceInfo{Name: "eth2", Type: netif.InterfaceTypeEthernet, Up: false, Score: 99},
	)

	resp := categorizeInterfaces(ifaces, "eth0")

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
	resp := categorizeInterfaces(categorizeFixture(), "")

	if resp.CurrentType != "" {
		t.Errorf("currentType = %q, want empty", resp.CurrentType)
	}
	if len(resp.Ethernet) != 1 || len(resp.WiFi) != 1 {
		t.Errorf("ethernet = %v, wifi = %v, want one each",
			names(resp.Ethernet), names(resp.WiFi))
	}
}
