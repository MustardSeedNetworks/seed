//go:build windows

// Windows-specific NDP (IPv6 Neighbor Discovery Protocol) implementation.
//
// This does NOT read the IPv6 neighbour table -- see #2174. The GetIpNetTable2
// scaffolding that used to sit here had no callers and carried two `go vet`
// unsafe.Pointer findings, so it was removed rather than left looking like a
// working implementation.
package enumerate

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

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

	// Pass stopChan as a parameter so the goroutine works on its captured
	// copy; matches the linux build.
	go ns.scanLoop(ns.stopChan)

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

// scanLoop periodically scans the IPv6 neighbor table. stopChan is passed as
// a parameter so the goroutine works on its captured copy; matches linux.
func (ns *NDPScanner) scanLoop(stopChan <-chan struct{}) {
	// Initial scan
	if err := ns.scanNeighborTable(); err != nil {
		slog.Error("IPv6 neighbor scan error", "error", err)
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
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
