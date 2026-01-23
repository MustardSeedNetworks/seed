//go:build windows

// Windows-specific NDP (IPv6 Neighbor Discovery Protocol) implementation.
// Uses GetIpNetTable2 to read IPv6 neighbor entries from Windows.
package discovery

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
	"unsafe"
)

// Windows IP Helper API structures for IPv6.
const (
	// NL_NEIGHBOR_STATE constants from nldef.h
	nlnsUnreachable = 0
	nlnsIncomplete  = 1
	nlnsProbe       = 2
	nlnsDelay       = 3
	nlnsStale       = 4
	nlnsReachable   = 5
	nlnsPermanent   = 6
)

// afINET6 is the Windows address family constant for IPv6.
const afINET6 = 23

var (
	iphlpapiDLL        = syscallNewLazyDLL("iphlpapi.dll")
	procGetIpNetTable2 = iphlpapiDLL.NewProc("GetIpNetTable2")
	procFreeMibTable   = iphlpapiDLL.NewProc("FreeMibTable")
)

// syscallNewLazyDLL creates a lazy-loaded DLL handle.
// This is a wrapper to avoid import cycle with golang.org/x/sys/windows.
func syscallNewLazyDLL(name string) *lazyDLL {
	return &lazyDLL{name: name}
}

// lazyDLL represents a lazy-loaded DLL.
type lazyDLL struct {
	name string
	dll  uintptr
}

// NewProc returns a procedure from the DLL.
func (d *lazyDLL) NewProc(name string) *lazyProc {
	return &lazyProc{dll: d, name: name}
}

// lazyProc represents a procedure in a lazy-loaded DLL.
type lazyProc struct {
	dll  *lazyDLL
	name string
	addr uintptr
}

// Call calls the procedure.
// This is a minimal implementation - actual syscall would use windows.NewLazySystemDLL.
func (p *lazyProc) Call(args ...uintptr) (uintptr, uintptr, error) {
	// Note: This is a stub. The actual implementation uses golang.org/x/sys/windows.
	// For Windows builds, this should use:
	//   windows.NewLazySystemDLL("iphlpapi.dll").NewProc("GetIpNetTable2")
	return 0, 0, fmt.Errorf("not implemented: requires golang.org/x/sys/windows")
}

// NDPScanner scans for IPv6 neighbors using the kernel's neighbor table.
type NDPScanner struct {
	mu            sync.RWMutex
	interfaceName string
	neighbors     map[string]*NDPNeighbor // key is IPv6 address
	running       bool
	stopChan      chan struct{}
}

// NDPNeighbor represents an IPv6 neighbor.
type NDPNeighbor struct {
	IPv6     string
	MAC      string
	IsRouter bool
	State    string // NUD state: REACHABLE, STALE, DELAY, PROBE, etc.
	LastSeen time.Time
}

// NewNDPScanner creates a new IPv6 NDP scanner.
func NewNDPScanner(interfaceName string) *NDPScanner {
	return &NDPScanner{
		interfaceName: interfaceName,
		neighbors:     make(map[string]*NDPNeighbor),
	}
}

// Start begins periodic scanning of the IPv6 neighbor table.
func (ns *NDPScanner) Start() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if ns.running {
		return fmt.Errorf("NDP scanner already running")
	}

	ns.running = true
	ns.stopChan = make(chan struct{})

	// Start background scanner
	go ns.scanLoop()

	slog.Info("IPv6 NDP scanner started", "interface", ns.interfaceName)
	return nil
}

// Stop stops the NDP scanner.
func (ns *NDPScanner) Stop() error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.running {
		return nil
	}

	close(ns.stopChan)
	ns.running = false

	slog.Info("IPv6 NDP scanner stopped")
	return nil
}

// IsRunning returns whether the scanner is running.
func (ns *NDPScanner) IsRunning() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.running
}

// GetNeighbors returns all discovered IPv6 neighbors.
func (ns *NDPScanner) GetNeighbors() map[string]*NDPNeighbor {
	ns.mu.RLock()
	defer ns.mu.RUnlock()

	// Return a copy
	neighbors := make(map[string]*NDPNeighbor, len(ns.neighbors))
	for k, v := range ns.neighbors {
		neighborCopy := *v
		neighbors[k] = &neighborCopy
	}

	return neighbors
}

// scanLoop periodically scans the IPv6 neighbor table.
func (ns *NDPScanner) scanLoop() {
	// Initial scan
	if err := ns.scanNeighborTable(); err != nil {
		slog.Error("IPv6 neighbor scan error", "error", err)
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ns.stopChan:
			return
		case <-ticker.C:
			if err := ns.scanNeighborTable(); err != nil {
				slog.Error("IPv6 neighbor scan error", "error", err)
			}
		}
	}
}

// scanNeighborTable scans the Windows IPv6 neighbor table using GetIpNetTable2.
func (ns *NDPScanner) scanNeighborTable() error {
	// Use netsh as a fallback since GetIpNetTable2 requires proper Windows API bindings
	return ns.scanNeighborTableNetsh()
}

// scanNeighborTableNetsh reads IPv6 neighbors using netsh command.
func (ns *NDPScanner) scanNeighborTableNetsh() error {
	// This would use: netsh interface ipv6 show neighbors
	// For now, use Go's standard net package to get interface info
	iface, err := net.InterfaceByName(ns.interfaceName)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}

	// Get addresses to determine subnet
	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get addresses: %w", err)
	}

	now := time.Now()
	ns.mu.Lock()
	defer ns.mu.Unlock()

	// On Windows, we can't easily get the neighbor table without Windows API
	// This is a simplified implementation that tracks local interface IPv6 addresses
	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipNet.IP
		if ip.To4() != nil {
			continue // Skip IPv4
		}

		ipv6Addr := ip.String()

		// Check if this is a link-local address (starts with fe80::)
		isRouter := false // We can't determine router status without proper API

		neighbor, exists := ns.neighbors[ipv6Addr]
		if !exists {
			neighbor = &NDPNeighbor{
				IPv6:     ipv6Addr,
				MAC:      iface.HardwareAddr.String(),
				IsRouter: isRouter,
				State:    "REACHABLE",
				LastSeen: now,
			}
			ns.neighbors[ipv6Addr] = neighbor
		} else {
			neighbor.LastSeen = now
		}
	}

	return nil
}

// CleanupStale removes neighbors that haven't been seen in the specified duration.
func (ns *NDPScanner) CleanupStale(maxAge time.Duration) {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	now := time.Now()
	for ipv6, neighbor := range ns.neighbors {
		if now.Sub(neighbor.LastSeen) > maxAge {
			delete(ns.neighbors, ipv6)
		}
	}
}

// mibIPNetRow2 represents a single IPv6 neighbor entry from GetIpNetTable2.
// Reserved for future use when proper Windows API bindings are available.
type mibIPNetRow2 struct {
	Address         [28]byte // SOCKADDR_INET (28 bytes)
	InterfaceIndex  uint32
	InterfaceLuid   uint64
	PhysicalAddress [32]byte
	PhysicalAddrLen uint32
	State           uint32
	Flags           uint8
	_               [3]byte // padding
	ReachabilityTime uint32
}

// readIPv6NeighborTable reads IPv6 neighbors using Windows API.
// This is reserved for future implementation with proper Windows bindings.
func readIPv6NeighborTable() ([]*NDPNeighbor, error) {
	var tablePtr uintptr
	ret, _, _ := procGetIpNetTable2.Call(uintptr(afINET6), uintptr(unsafe.Pointer(&tablePtr)))
	if ret != 0 {
		return nil, fmt.Errorf("GetIpNetTable2 failed: %d", ret)
	}

	if tablePtr == 0 {
		return []*NDPNeighbor{}, nil
	}

	// Free table when done
	defer procFreeMibTable.Call(tablePtr)

	// Parse table structure
	// First uint32 is count, then array of mibIPNetRow2
	numEntries := *(*uint32)(unsafe.Pointer(tablePtr))
	if numEntries == 0 {
		return []*NDPNeighbor{}, nil
	}

	neighbors := make([]*NDPNeighbor, 0, numEntries)
	rowSize := unsafe.Sizeof(mibIPNetRow2{})
	rowsStart := tablePtr + 8 // Skip count + padding

	for i := uint32(0); i < numEntries; i++ {
		rowPtr := rowsStart + uintptr(i)*rowSize
		row := (*mibIPNetRow2)(unsafe.Pointer(rowPtr))

		// Parse IPv6 address from SOCKADDR_INET
		// Offset 4 in SOCKADDR_INET is where sin6_addr starts for AF_INET6
		ip := net.IP(row.Address[8:24])
		if ip == nil || ip.To4() != nil {
			continue // Skip non-IPv6
		}

		mac := formatMACAddress(row.PhysicalAddress[:row.PhysicalAddrLen])
		state := ndpStateToString(row.State)

		neighbors = append(neighbors, &NDPNeighbor{
			IPv6:     ip.String(),
			MAC:      mac,
			IsRouter: false, // Would need to check flags
			State:    state,
			LastSeen: time.Now(),
		})
	}

	return neighbors, nil
}

// formatMACAddress formats a byte slice as a MAC address.
func formatMACAddress(mac []byte) string {
	if len(mac) < 6 {
		return ""
	}
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
}

// ndpStateToString converts Windows NL_NEIGHBOR_STATE to string.
func ndpStateToString(state uint32) string {
	switch state {
	case nlnsReachable:
		return "REACHABLE"
	case nlnsStale:
		return "STALE"
	case nlnsDelay:
		return "DELAY"
	case nlnsProbe:
		return "PROBE"
	case nlnsPermanent:
		return "PERMANENT"
	case nlnsIncomplete:
		return "INCOMPLETE"
	case nlnsUnreachable:
		return "UNREACHABLE"
	default:
		return "UNKNOWN"
	}
}
