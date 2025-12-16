//go:build linux
// +build linux

// Package discovery implements multi-protocol network device discovery.
// NDP (Neighbor Discovery Protocol) support for Linux enables IPv6 neighbor discovery
// by reading from the kernel's neighbor table, allowing detection of IPv6-capable devices
// and routers on the local network segment.
package discovery

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vishvananda/netlink"
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

// scanNeighborTable scans the kernel's IPv6 neighbor table.
func (ns *NDPScanner) scanNeighborTable() error {
	// Get interface
	link, err := netlink.LinkByName(ns.interfaceName)
	if err != nil {
		return fmt.Errorf("failed to get interface: %w", err)
	}

	// Get neighbors for this interface (IPv6 only)
	neighs, err := netlink.NeighList(link.Attrs().Index, netlink.FAMILY_V6)
	if err != nil {
		return fmt.Errorf("failed to list neighbors: %w", err)
	}

	now := time.Now()

	ns.mu.Lock()
	defer ns.mu.Unlock()

	for _, neigh := range neighs {
		// Only process IPv6 addresses
		if neigh.IP.To4() != nil {
			continue
		}

		ipv6Addr := neigh.IP.String()
		mac := neigh.HardwareAddr.String()

		// Determine if this is a router
		// Routers typically have link-local fe80:: addresses and are in REACHABLE state
		isRouter := neigh.Flags&netlink.NTF_ROUTER != 0

		// Get NUD state as string
		state := nudStateString(neigh.State)

		neighbor, exists := ns.neighbors[ipv6Addr]
		if !exists {
			neighbor = &NDPNeighbor{
				IPv6:     ipv6Addr,
				MAC:      mac,
				IsRouter: isRouter,
				State:    state,
				LastSeen: now,
			}
			ns.neighbors[ipv6Addr] = neighbor
		} else {
			// Update existing neighbor
			if mac != "" {
				neighbor.MAC = mac
			}
			neighbor.IsRouter = isRouter
			neighbor.State = state
			neighbor.LastSeen = now
		}
	}

	return nil
}

// nudStateString converts netlink NUD state to string.
func nudStateString(state int) string {
	switch state {
	case netlink.NUD_INCOMPLETE:
		return "INCOMPLETE"
	case netlink.NUD_REACHABLE:
		return "REACHABLE"
	case netlink.NUD_STALE:
		return "STALE"
	case netlink.NUD_DELAY:
		return "DELAY"
	case netlink.NUD_PROBE:
		return "PROBE"
	case netlink.NUD_FAILED:
		return "FAILED"
	case netlink.NUD_NOARP:
		return "NOARP"
	case netlink.NUD_PERMANENT:
		return "PERMANENT"
	default:
		return "UNKNOWN"
	}
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
