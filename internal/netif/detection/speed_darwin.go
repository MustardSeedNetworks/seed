//go:build darwin

package detection

// macOS reads link speed from the kernel's interface table rather than from
// networksetup or ifconfig. Forking helpers here was the source of seed#2420 and
// seed#2520: a fork() out of this cgo/ObjC process can wedge the child before
// execve, and DetectAll forks once per interface at once.

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include <stdlib.h>
#include "pci_darwin.h"
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/net/route"
)

// Offsets into struct if_msghdr2 (the NET_RT_IFLIST2 record) for the fields the
// speed reader needs: ifm_msglen, ifm_type, ifm_index and ifm_data.ifi_baudrate.
// The layout is the same on arm64 and amd64; TestIfMsghdr2LayoutMatchesKernelABI
// holds these against x/sys's generated struct.
const (
	offMsglen       = 0
	offType         = 3
	offIndex        = 12
	offBaudrate     = 48
	sizeofIfMsghdr2 = 160
)

// getInterfaceSpeed returns the interface speed in bits per second, as the
// kernel reports it in ifi_baudrate. Ports with no link report 0; Wi-Fi reports
// its current PHY rate.
func getInterfaceSpeed(_ commandRunner, name string) int64 {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return 0
	}
	rib, err := route.FetchRIB(syscall.AF_UNSPEC, syscall.NET_RT_IFLIST2, 0)
	if err != nil {
		return 0
	}
	baud := baudrates(rib)[iface.Index]
	if baud > math.MaxInt64 {
		return 0
	}
	return int64(baud)
}

// baudrates walks a NET_RT_IFLIST2 dump and returns ifi_baudrate keyed by
// interface index. Address records that follow each RTM_IFINFO2 are stepped
// over by their length; a zero length or a record running past the buffer
// ends the walk.
func baudrates(rib []byte) map[int]uint64 {
	speeds := make(map[int]uint64)
	for off := 0; off+offType < len(rib); {
		msglen := int(binary.NativeEndian.Uint16(rib[off+offMsglen:]))
		if msglen == 0 || off+msglen > len(rib) {
			break
		}
		if rib[off+offType] == syscall.RTM_IFINFO2 && msglen >= sizeofIfMsghdr2 {
			index := int(binary.NativeEndian.Uint16(rib[off+offIndex:]))
			speeds[index] = binary.NativeEndian.Uint64(rib[off+offBaudrate:])
		}
		off += msglen
	}
	return speeds
}

// identifyByPlatformUncached matches the PCI IDs of the device behind the
// interface against the database, the same IDs Linux reads from sysfs. They come
// from the IORegistry in process: the system_profiler keyword scan this replaced
// forked (seed#2530) and matched "ice" inside "device", naming every interface
// an Intel E810 (seed#2528).
func (db *ChipsetDatabase) identifyByPlatformUncached(name string) *ChipsetInfo {
	vendor, device, ok := pciIDs(name)
	if !ok {
		return nil
	}
	return db.IdentifyByPCI(fmt.Sprintf("%04x", vendor), fmt.Sprintf("%04x", device))
}

// pciIDs returns the vendor and device IDs of the PCI device behind the
// interface, or false when there is none.
func pciIDs(name string) (uint16, uint16, bool) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	var v, d C.uint16_t
	if C.netif_pci_ids(cname, &v, &d) == 0 {
		return 0, 0, false
	}
	return uint16(v), uint16(d), true
}

// hasTDRCapability checks if the interface supports Time Domain Reflectometry.
// macOS generally doesn't expose TDR capabilities directly.
func hasTDRCapability(_ string) bool {
	// TDR is not typically accessible on macOS
	// Would need specialized drivers or hardware tools
	return false
}

// hasDOMCapability checks if the interface supports Digital Optical Monitoring.
// macOS generally doesn't expose DOM capabilities directly.
func hasDOMCapability(_ string) bool {
	// DOM is not typically accessible on macOS consumer hardware
	return false
}
