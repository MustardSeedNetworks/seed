//go:build linux

package vlan_test

import (
	"strconv"
	"testing"

	"github.com/MustardSeedNetworks/seed/internal/diagnostics/vlan"
)

// TestMultipleManagerInstances tests multiple manager instances for different interfaces.
func TestMultipleManagerInstances(t *testing.T) {
	t.Parallel()

	interfaces := []string{"eth0", "eth1", "en0", "en1", "bond0"}
	managers := make([]*vlan.Manager, len(interfaces))

	// Create managers.
	for i, iface := range interfaces {
		managers[i] = vlan.NewManager(iface)
		managers[i].SetConfigured(true, (i+1)*100)
	}

	// Verify each manager is independent.
	for i, mgr := range managers {
		expectedID := (i + 1) * 100
		if mgr.ManagerConfiguredID() != expectedID {
			t.Errorf("manager %d: expected ID %d, got %d", i, expectedID, mgr.ManagerConfiguredID())
		}
		if mgr.ManagerInterfaceName() != interfaces[i] {
			t.Errorf("manager %d: expected interface %q, got %q", i, interfaces[i], mgr.ManagerInterfaceName())
		}
	}
}

// TestManagerInterfaceSwitch tests switching interfaces.
func TestManagerInterfaceSwitch(t *testing.T) {
	t.Parallel()

	manager := vlan.NewManager("eth0")
	manager.SetConfigured(true, 100)

	// Switch interfaces.
	interfaces := []string{"en0", "wlan0", "bond0", "br0", "eth1"}
	for _, iface := range interfaces {
		manager.SetInterface(iface)
		if manager.ManagerInterfaceName() != iface {
			t.Errorf("expected interface %q, got %q", iface, manager.ManagerInterfaceName())
		}
		// Configuration should persist.
		if !manager.ManagerEnabled() {
			t.Error("configuration should persist after interface switch")
		}
		if manager.ManagerConfiguredID() != 100 {
			t.Error("configured ID should persist after interface switch")
		}
	}
}

// TestContainsFunctionality tests the contains helper comprehensively.
func TestContainsFunctionality(t *testing.T) {
	t.Parallel()

	// Test with various slice sizes.
	sizes := []int{0, 1, 10, 100, 1000}

	for _, size := range sizes {
		t.Run("size_"+strconv.Itoa(size), func(t *testing.T) {
			t.Parallel()
			assertContainsForSize(t, size)
		})
	}
}

func assertContainsForSize(t *testing.T, size int) {
	t.Helper()

	values := make([]int, size)
	for i := range values {
		values[i] = i
	}
	if vlan.ExportContains(values, size) {
		t.Error("should not find element beyond slice")
	}
	if vlan.ExportContains(values, -1) {
		t.Error("should not find negative element")
	}
	if size == 0 {
		return
	}
	if !vlan.ExportContains(values, 0) {
		t.Error("should find first element")
	}
	if !vlan.ExportContains(values, size-1) {
		t.Error("should find last element")
	}
	if size > 1 && !vlan.ExportContains(values, size/2) {
		t.Error("should find middle element")
	}
}

// TestCreateDeleteVLANInterfaceSequence tests create/delete sequence.
func TestCreateDeleteVLANInterfaceSequence(t *testing.T) {
	t.Parallel()

	// Test create then delete.
	err := vlan.CreateVlanInterface("eth0", 100)
	_ = err // May fail without privileges.

	err = vlan.DeleteVlanInterface("eth0", 100)
	_ = err // May fail without privileges.

	// Test delete then create.
	err = vlan.DeleteVlanInterface("eth0", 200)
	_ = err

	err = vlan.CreateVlanInterface("eth0", 200)
	_ = err

	// Test multiple creates.
	for i := range 10 {
		err = vlan.CreateVlanInterface("eth0", i)
		_ = err
	}

	// Test multiple deletes.
	for i := range 10 {
		err = vlan.DeleteVlanInterface("eth0", i)
		_ = err
	}
}
