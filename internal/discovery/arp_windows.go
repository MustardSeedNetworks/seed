//go:build windows

// Windows-specific ARP table implementation using Windows IP Helper API.
// Uses GetIpNetTable to read the ARP cache entries.
package discovery

import (
	"fmt"
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

var (
	iphlpapi            = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetIpNetTable   = iphlpapi.NewProc("GetIpNetTable")
	procGetAdaptersInfo = iphlpapi.NewProc("GetAdaptersInfo")
)

// readARPTablePlatform reads the ARP table on Windows using GetIpNetTable.
func (s *ARPScanner) readARPTablePlatform() ([]*ARPEntry, error) {
	// First call to get required buffer size
	var size uint32
	ret, _, _ := procGetIpNetTable.Call(0, uintptr(unsafe.Pointer(&size)), 0)

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
	ret, _, _ = procGetIpNetTable.Call(
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

	for i := uint32(0); i < table.dwNumEntries; i++ {
		// Calculate offset for each row
		rowPtr := unsafe.Pointer(uintptr(unsafe.Pointer(&table.table[0])) + uintptr(i)*unsafe.Sizeof(mibIPNetRow{}))
		row := (*mibIPNetRow)(rowPtr)

		// Skip invalid entries
		if row.dwType == mibIPNetTypeInvalid {
			continue
		}

		// Convert IP address from network byte order
		ip := net.IPv4(
			byte(row.dwAddr),
			byte(row.dwAddr>>8),
			byte(row.dwAddr>>16),
			byte(row.dwAddr>>24),
		)

		// Check if this IP is in our target subnets
		if !s.isInSubnet(ip.String()) {
			continue
		}

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
	if len(mac) < 6 {
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
		m[uint32(iface.Index)] = iface.Name
	}

	return m
}
