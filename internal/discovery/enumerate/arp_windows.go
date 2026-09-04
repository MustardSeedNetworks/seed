//go:build windows

package enumerate

// Windows-specific ARP table implementation using Windows IP Helper API.
// Uses GetIpNetTable to read the ARP cache entries.

import (
	"fmt"
	"math"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows IP Helper API structures and constants.
const (
	// MIB_IPNET_TYPE constants for ARP entry types.
	mibIPNetTypeOther   = 1
	mibIPNetTypeInvalid = 2
	mibIPNetTypeDynamic = 3
	mibIPNetTypeStatic  = 4
)

// macAddrLen is the length of an Ethernet hardware address in bytes.
const macAddrLen = 6

// mibIPNetRow represents a single ARP table entry from GetIpNetTable.
type mibIPNetRow struct {
	dwIndex       uint32
	dwPhysAddrLen uint32
	bPhysAddr     [8]byte // MAXLEN_PHYSADDR is typically 8
	dwAddr        uint32  // IPv4 address in network byte order
	dwType        uint32
}

// mibIPNetTable represents the ARP table structure from GetIpNetTable.
type mibIPNetTable struct {
	dwNumEntries uint32
	table        [1]mibIPNetRow // Variable-length array
}

// readARPTablePlatform reads the ARP table on Windows using GetIpNetTable.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	// NewLazySystemDLL/NewProc only resolve on first Call, and the OS loader
	// itself caches an already-loaded DLL by name, so building these locally
	// rather than caching them in a package-level var costs a lookup, not a
	// reload -- cheap next to the scan interval this runs on, and it keeps
	// the syscall handle out of package state between calls.
	procGetIPNetTable := windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetIpNetTable")

	// First call to get required buffer size
	var size uint32
	ret, _, _ := procGetIPNetTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)

	// ERROR_INSUFFICIENT_BUFFER (122) is expected on first call
	if ret != 0 && ret != 122 {
		return nil, fmt.Errorf("GetIpNetTable size query failed: %d", ret)
	}

	if size == 0 {
		// No ARP entries
		return []*ARPEntry{}, nil
	}

	// Allocate buffer and make second call
	buf := make([]byte, size)
	ret, _, _ = procGetIPNetTable.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
		0,
	)

	if ret != 0 {
		return nil, fmt.Errorf("GetIpNetTable failed: %d", ret)
	}

	// Parse the table
	table := (*mibIPNetTable)(unsafe.Pointer(&buf[0]))
	entries := make([]*ARPEntry, 0, table.dwNumEntries)

	// Get interface index to name mapping
	ifaceMap := buildInterfaceMap()

	for i := range table.dwNumEntries {
		// Calculate offset for each row
		rowPtr := unsafe.Add(unsafe.Pointer(&table.table[0]), uintptr(i)*unsafe.Sizeof(mibIPNetRow{}))
		row := (*mibIPNetRow)(rowPtr)

		// Skip invalid entries
		if row.dwType == mibIPNetTypeInvalid {
			continue
		}

		// Convert IP address from network byte order. Each conversion is
		// masked to a single byte so it can never overflow, regardless of
		// the width gosec assumes for a bare byte(uint32) truncation.
		ip := net.IPv4(
			byte(row.dwAddr&byteMask),
			byte((row.dwAddr>>byteShift8)&byteMask),
			byte((row.dwAddr>>byteShift16)&byteMask),
			byte((row.dwAddr>>byteShift24)&byteMask),
		)

		// Format MAC address
		mac := formatMAC(row.bPhysAddr[:row.dwPhysAddrLen])
		if mac == "" || mac == "00:00:00:00:00:00" {
			continue
		}

		// Get state string
		state := arpTypeToState(row.dwType)

		// Get interface name
		ifaceName := ifaceMap[row.dwIndex]

		entries = append(entries, &ARPEntry{
			IP:        ip.String(),
			MAC:       mac,
			Interface: ifaceName,
			State:     state,
			LastSeen:  time.Now(),
		})
	}

	return entries, nil
}

// arpTypeToState converts Windows ARP entry type to state string.
func arpTypeToState(arpType uint32) string {
	switch arpType {
	case mibIPNetTypeDynamic:
		return "REACHABLE"
	case mibIPNetTypeStatic:
		return "PERMANENT"
	case mibIPNetTypeOther:
		return "STALE"
	default:
		return "UNKNOWN"
	}
}

// formatMAC formats a byte slice as a MAC address string.
func formatMAC(mac []byte) string {
	if len(mac) < macAddrLen {
		return ""
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// buildInterfaceMap creates a mapping from interface index to interface name.
func buildInterfaceMap() map[uint32]string {
	m := make(map[uint32]string)

	interfaces, err := net.Interfaces()
	if err != nil {
		return m
	}

	for _, iface := range interfaces {
		// net.Interface.Index is platform-defined as a small positive
		// number; out of that range would mean the OS itself is broken, not
		// something to convert around, so it is skipped rather than wrapped.
		if iface.Index < 0 || iface.Index > math.MaxUint32 {
			continue
		}
		m[uint32(iface.Index)] = iface.Name
	}

	return m
}
