//go:build darwin

package vlan

import (
	"net"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

// SIOCGIFVLAN ioctl to get VLAN info.
const siocgifvlan = 0xc020697f

// vlanreq is the structure for VLAN ioctl on macOS.
type vlanreq struct {
	name       [16]byte // IFNAMSIZ
	parentName [16]byte
	vlanTag    uint16
	_          [2]byte // padding
}

// detectVlanSubinterfacesPlatform detects VLAN subinterfaces on macOS.
func detectVlanSubinterfacesPlatform(iface string) []int {
	vlans := make([]int, 0)

	// Get all interfaces
	interfaces, err := net.Interfaces()
	if err != nil {
		return vlans
	}

	// Look for vlan interfaces
	for _, intf := range interfaces {
		// macOS VLAN interfaces start with "vlan"
		if !strings.HasPrefix(intf.Name, "vlan") {
			continue
		}

		// Try to get VLAN info via ioctl
		parent, vlanID := getVlanInfo(intf.Name)
		if parent == iface && vlanID > 0 {
			vlans = append(vlans, vlanID)
		}
	}

	return vlans
}

// getVlanInfo retrieves VLAN parent interface and tag for a VLAN interface.
func getVlanInfo(ifname string) (string, int) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return "", 0
	}
	defer func() { _ = syscall.Close(fd) }()

	var req vlanreq
	copy(req.name[:], ifname)

	// Perform ioctl to get VLAN info

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(siocgifvlan),
		uintptr(unsafe.Pointer(&req)),
	)

	if errno != 0 {
		// If ioctl fails, try to parse from interface name
		// Some VLANs might be named like "vlan100" where 100 is the VLAN ID
		if after, ok := strings.CutPrefix(ifname, "vlan"); ok {
			suffix := after
			if id, parseErr := strconv.Atoi(suffix); parseErr == nil {
				return "", id
			}
		}
		return "", 0
	}

	// Extract parent name (null-terminated string)
	parentBytes := req.parentName[:]
	parentEnd := 0
	for i, b := range parentBytes {
		if b == 0 {
			parentEnd = i
			break
		}
	}
	parent := string(parentBytes[:parentEnd])

	return parent, int(req.vlanTag)
}

// createSupportedPlatform reports that macOS cannot create VLAN subinterfaces.
//
// Doing so requires a Network preferences change that no API available to a
// daemon can make; it is not something we can perform on the operator's behalf.
func createSupportedPlatform() bool { return false }

// createVlanInterfacePlatform reports that macOS cannot create a VLAN
// subinterface.
//
// This previously returned nil, which reported success for work that was never
// performed: callers saw a created VLAN that did not exist.
func createVlanInterfacePlatform(_ string, _ int) error {
	return ErrUnsupported
}

// deleteVlanInterfacePlatform reports that macOS cannot remove a VLAN
// subinterface. It previously returned nil for the same reason, with the same
// consequence.
func deleteVlanInterfacePlatform(_ string, _ int) error {
	return ErrUnsupported
}
