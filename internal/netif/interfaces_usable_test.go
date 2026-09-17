package netif_test

// The daemon can be bound to an interface its own API said did not exist.
//
// [netif.Manager.FindFirstAvailable] scores an InterfaceTypeOther interface
// with a routable address as an ethernet-class candidate (select.go,
// collectCandidates), so a name outside the ethernet/wifi prefix lists —
// feth0 on macOS, and equally bond0, ppp0 or usb0 — can become the active
// interface. The listing the API serves kept a narrower definition and
// dropped every type but ethernet and wifi, which is how seed#2692 saw
// GET /api/v1/interfaces omit feth0 while GET /api/v1/interface named it.
//
// These cases pin the one definition both sides now share: whatever the
// daemon is bound to is listed, and nothing else changes.

import (
	"slices"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/netif"
)

// usableFixture is the shape of the reported defect: an ethernet interface,
// a virtual one, and an InterfaceTypeOther one that selection can choose.
func usableFixture() map[string]*netif.InterfaceInfo {
	return map[string]*netif.InterfaceInfo{
		"eth0": {
			Name:      "eth0",
			Type:      netif.InterfaceTypeEthernet,
			Up:        true,
			Addresses: []string{"192.168.1.100/24"},
		},
		"feth0": {
			Name:      "feth0",
			Type:      netif.InterfaceTypeOther,
			Up:        true,
			Addresses: []string{"10.51.200.2/24"},
		},
		"docker0": {
			Name:      "docker0",
			Type:      netif.InterfaceTypeVirtual,
			Up:        true,
			Addresses: []string{"172.17.0.1/16"},
		},
	}
}

func usableNames(t *testing.T, mgr *netif.Manager) []string {
	t.Helper()
	got := make([]string, 0, 3)
	for _, iface := range mgr.GetUsableInterfaces() {
		got = append(got, iface.Name)
	}
	slices.Sort(got)
	return got
}

// TestGetUsableInterfacesListsTheSelectedInterface is seed#2692 itself: the
// active interface is feth0 and the listing must name it.
func TestGetUsableInterfacesListsTheSelectedInterface(t *testing.T) {
	mgr := netif.CreateManagerWithInterfaces(usableFixture())
	if err := mgr.SetCurrentInterface("feth0"); err != nil {
		t.Fatalf("SetCurrentInterface(feth0) = %v, want nil", err)
	}

	got := usableNames(t, mgr)
	want := []string{"eth0", "feth0"}
	if !slices.Equal(got, want) {
		t.Errorf("GetUsableInterfaces() = %v, want %v", got, want)
	}
}

// TestGetUsableInterfacesListsASelectedVirtualInterface covers the case the
// type filter argues hardest against. SetCurrentInterface accepts any name in
// the map, so an operator can bind the daemon to a bridge; denying that it
// exists is the same defect with a different name on it.
func TestGetUsableInterfacesListsASelectedVirtualInterface(t *testing.T) {
	mgr := netif.CreateManagerWithInterfaces(usableFixture())
	if err := mgr.SetCurrentInterface("docker0"); err != nil {
		t.Fatalf("SetCurrentInterface(docker0) = %v, want nil", err)
	}

	got := usableNames(t, mgr)
	want := []string{"docker0", "eth0"}
	if !slices.Equal(got, want) {
		t.Errorf("GetUsableInterfaces() = %v, want %v", got, want)
	}
}

// TestGetUsableInterfacesWithoutASelectionIsUnchanged keeps the filter doing
// its job: an unselected feth0 or docker0 is still noise, so the fix cannot
// be "list everything".
func TestGetUsableInterfacesWithoutASelectionIsUnchanged(t *testing.T) {
	mgr := netif.CreateManagerWithInterfaces(usableFixture())

	got := usableNames(t, mgr)
	want := []string{"eth0"}
	if !slices.Equal(got, want) {
		t.Errorf("GetUsableInterfaces() = %v, want %v", got, want)
	}
}

// TestGetUsableInterfacesSelectionNotDetected guards the stale-selection case:
// a configured interface that is no longer on the host must not appear as a
// nil entry or crash the listing.
func TestGetUsableInterfacesSelectionNotDetected(t *testing.T) {
	mgr := netif.CreateManagerWithInterfaces(usableFixture())
	helper := netif.NewManagerTestHelper(mgr)
	helper.SetCurrentInterfaceUnchecked("gone0")

	got := usableNames(t, mgr)
	want := []string{"eth0"}
	if !slices.Equal(got, want) {
		t.Errorf("GetUsableInterfaces() = %v, want %v", got, want)
	}
}
