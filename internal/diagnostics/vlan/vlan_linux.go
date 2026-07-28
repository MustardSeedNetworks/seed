//go:build linux

package vlan

import (
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/vishvananda/netlink"
)

const vlanConfigFieldCount = 3

// detectVlanSubinterfacesPlatform detects VLAN subinterfaces on Linux using netlink.
func detectVlanSubinterfacesPlatform(iface string) []int {
	vlans := make([]int, 0)

	// Get all links
	links, err := netlink.LinkList()
	if err != nil {
		return vlans
	}

	for _, link := range links {
		// Check if this is a VLAN interface
		vlan, ok := link.(*netlink.Vlan)
		if !ok {
			continue
		}

		// Get the parent link
		parentLink, parentErr := netlink.LinkByIndex(vlan.Attrs().ParentIndex)
		if parentErr != nil {
			continue
		}

		// Check if parent matches our interface
		if parentLink.Attrs().Name == iface {
			vlans = append(vlans, vlan.VlanId)
		}
	}

	// Also check /proc/net/vlan/config for legacy VLAN module
	procVlans := detectVlansFromProc(iface)
	for _, v := range procVlans {
		if !contains(vlans, v) {
			vlans = append(vlans, v)
		}
	}

	return vlans
}

// detectVlansFromProc reads VLAN config from /proc/net/vlan/config.
func detectVlansFromProc(iface string) []int {
	vlans := make([]int, 0)

	data, err := os.ReadFile("/proc/net/vlan/config")
	if err != nil {
		return vlans
	}

	// Format: "eth0.100 | 100 | eth0"
	for line := range strings.SplitSeq(string(data), "\n") {
		fields := strings.Split(line, "|")
		if len(fields) >= vlanConfigFieldCount {
			parentIface := strings.TrimSpace(fields[2])
			if parentIface == iface {
				if vlanID, parseErr := strconv.Atoi(strings.TrimSpace(fields[1])); parseErr == nil {
					vlans = append(vlans, vlanID)
				}
			}
		}
	}

	return vlans
}

// createVlanInterfacePlatform creates a VLAN interface on Linux using netlink.
func createVlanInterfacePlatform(parentIface string, vlanID int) error {
	// Get parent link
	parent, err := netlink.LinkByName(parentIface)
	if err != nil {
		return err
	}

	vlanIface := parentIface + "." + strconv.Itoa(vlanID)

	// Create VLAN link
	vlan := &netlink.Vlan{
		LinkAttrs: netlink.LinkAttrs{
			Name:        vlanIface,
			ParentIndex: parent.Attrs().Index,
		},
		VlanId: vlanID,
	}

	if addErr := netlink.LinkAdd(vlan); addErr != nil {
		return addErr
	}

	// Bring interface up
	return netlink.LinkSetUp(vlan)
}

// contains checks if a slice contains a value.
func contains(slice []int, val int) bool {
	return slices.Contains(slice, val)
}

// deleteVlanInterfacePlatform removes a VLAN interface on Linux using netlink.
func deleteVlanInterfacePlatform(parentIface string, vlanID int) error {
	vlanIface := parentIface + "." + strconv.Itoa(vlanID)

	link, err := netlink.LinkByName(vlanIface)
	if err != nil {
		return err
	}

	return netlink.LinkDel(link)
}
