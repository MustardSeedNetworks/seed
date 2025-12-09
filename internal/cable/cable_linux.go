//go:build linux

package cable

import (
	"net"
	"syscall"
	"unsafe"
)

// SIOCETHTOOL ioctl command
const siocethtool = 0x8946

// Ethtool command codes
const (
	ethtoolGDrvInfo   = 0x00000003 // Get driver info
	ethtoolGLinkState = 0x0000000a // Get link state
)

// ethtoolDrvInfo structure for driver information
type ethtoolDrvInfo struct {
	cmd         uint32
	driver      [32]byte
	version     [32]byte
	fwVersion   [32]byte
	busInfo     [32]byte
	eromVersion [32]byte
	reserved2   [12]byte
	nPrivFlags  uint32
	nStats      uint32
	testInfoLen uint32
	eedumpLen   uint32
	regdumpLen  uint32
}

// ethtoolValue structure for simple queries
type ethtoolValue struct {
	cmd  uint32
	data uint32
}

// ifreq structure for ioctl
type ifreq struct {
	name [16]byte
	data uintptr
}

// isSupportedPlatform checks if the NIC supports TDR on Linux.
// TDR support is driver-dependent and rare. Most NICs don't support it.
func isSupportedPlatform(iface string) bool {
	// Check if interface exists and get driver info
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return false
	}
	defer syscall.Close(fd)

	var req ifreq
	copy(req.name[:], iface)

	// Get driver info
	drvInfo := ethtoolDrvInfo{cmd: ethtoolGDrvInfo}
	req.data = uintptr(unsafe.Pointer(&drvInfo))

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(siocethtool),
		uintptr(unsafe.Pointer(&req)),
	)

	if errno != 0 {
		return false
	}

	// Convert driver name to string
	driverEnd := 0
	for i, b := range drvInfo.driver {
		if b == 0 {
			driverEnd = i
			break
		}
	}
	driver := string(drvInfo.driver[:driverEnd])

	// Only certain drivers support cable test
	// Common ones: e1000e, igb, ixgbe, marvell (some models)
	// Most virtual and common consumer NICs don't support it
	supportedDrivers := []string{
		"e1000e",
		"igb",
		"ixgbe",
		"i40e",
		"ice",
		"mlx5_core",
		"marvell",
	}

	for _, supported := range supportedDrivers {
		if driver == supported {
			return true
		}
	}

	return false
}

// testPlatform performs a cable test on Linux.
// Since the ethtool cable-test uses genetlink (not ioctl), and implementing
// the full genetlink protocol is complex, we use a simplified approach:
// - Check link state via ioctl (can indicate cable issues)
// - Check interface flags
// - Return basic connectivity info
func testPlatform(iface string) *TestResult {
	result := &TestResult{
		Supported: false,
		Status:    StatusUnknown,
		Faults:    make([]string, 0),
	}

	// Check if interface exists
	netIface, err := net.InterfaceByName(iface)
	if err != nil {
		result.Faults = append(result.Faults, "Interface not found")
		return result
	}

	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return result
	}
	defer syscall.Close(fd)

	var req ifreq
	copy(req.name[:], iface)

	// Get link state via ethtool
	linkState := ethtoolValue{cmd: ethtoolGLinkState}
	req.data = uintptr(unsafe.Pointer(&linkState))

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		uintptr(fd),
		uintptr(siocethtool),
		uintptr(unsafe.Pointer(&req)),
	)

	if errno != 0 {
		// Fallback to interface flags
		if netIface.Flags&net.FlagUp != 0 && netIface.Flags&net.FlagRunning != 0 {
			result.Status = StatusOK
		} else if netIface.Flags&net.FlagUp != 0 {
			result.Status = StatusOpen
			result.Faults = append(result.Faults, "No carrier detected")
		}
		return result
	}

	result.Supported = true

	if linkState.data != 0 {
		result.Status = StatusOK
	} else {
		// Link is down - could be cable issue
		result.Status = StatusOpen
		result.Faults = append(result.Faults, "No link detected - cable may be disconnected")
	}

	return result
}
