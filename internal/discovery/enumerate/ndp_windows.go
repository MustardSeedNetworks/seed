//go:build windows

package enumerate

// Windows-specific NDP (IPv6 Neighbor Discovery Protocol) implementation.
//
// Reads the IPv6 neighbour table through Get-NetNeighbor. The GetIpNetTable2
// scaffolding that used to sit here had no callers and carried two `go vet`
// unsafe.Pointer findings, so it was removed rather than left looking like a
// working implementation.

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// neighborQueryTimeout bounds each PowerShell call. A hung query must not
// stall the scan loop, which runs on a ticker.
const neighborQueryTimeout = 20 * time.Second

// neighborScanInterval is how often the neighbour table is re-read.
const neighborScanInterval = 30 * time.Second

// neighborCSVFieldCount is the number of columns Get-NetNeighbor's CSV
// carries: InterfaceAlias, IPAddress, LinkLayerAddress, State.
const neighborCSVFieldCount = 4

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
		return errors.New("NDP scanner already running")
	}

	ns.running = true
	ns.stopChan = make(chan struct{})

	// Pass stopChan as a parameter so the goroutine works on its captured
	// copy; matches the linux build.
	go ns.scanLoop(ns.stopChan)

	logging.GetLogger().Info("IPv6 NDP scanner started", "interface", ns.interfaceName)
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

	logging.GetLogger().Info("IPv6 NDP scanner stopped")
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
		logging.GetLogger().Error("IPv6 neighbor scan error", "error", err)
	}

	ticker := time.NewTicker(neighborScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopChan:
			return
		case <-ticker.C:
			if err := ns.scanNeighborTable(); err != nil {
				logging.GetLogger().Error("IPv6 neighbor scan error", "error", err)
			}
		}
	}
}

// windowsNeighborQuery is the PowerShell used to read the neighbour table.
// Get-NetNeighbor rather than `netsh interface ipv6 show neighbors`: netsh
// output is localised, so parsing it breaks on a non-English Windows, while
// ConvertTo-Csv is stable. This mirrors how vlan_windows.go reads adapter VLANs.
const windowsNeighborQuery = `Get-NetNeighbor -AddressFamily IPv6 -ErrorAction SilentlyContinue | ` +
	`Select-Object InterfaceAlias,IPAddress,LinkLayerAddress,State | ConvertTo-Csv -NoTypeInformation`

// windowsRouterQuery lists the next hops of IPv6 default routes. Get-NetNeighbor
// carries no router flag, so the Linux NTF_ROUTER equivalent is derived by
// matching neighbours against these.
const windowsRouterQuery = `Get-NetRoute -AddressFamily IPv6 -DestinationPrefix ::/0 -ErrorAction SilentlyContinue | ` +
	`Select-Object -ExpandProperty NextHop`

// scanNeighborTable reads the kernel's IPv6 neighbour table.
//
// This used to enumerate the *local interface's own addresses* and record each
// as a neighbour, every one carrying the local MAC and a hardcoded REACHABLE
// (#2174). That is worse than returning an error: callers received a populated,
// plausible-looking list that was entirely the host itself, with no signal it
// was wrong.
func (ns *NDPScanner) scanNeighborTable() error {
	ctx, cancel := context.WithTimeout(context.Background(), neighborQueryTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
		windowsNeighborQuery).Output()
	if err != nil {
		return fmt.Errorf("failed to read IPv6 neighbor table: %w", err)
	}

	routers := ns.readDefaultRouters(ctx)
	found := parseNetNeighbors(string(out), ns.interfaceName, routers)

	now := time.Now()
	ns.mu.Lock()
	defer ns.mu.Unlock()

	for i := range found {
		n := &found[i]
		if existing, ok := ns.neighbors[n.IPv6]; ok {
			if n.MAC != "" {
				existing.MAC = n.MAC
			}
			existing.IsRouter = n.IsRouter
			existing.State = n.State
			existing.LastSeen = now
			continue
		}
		n.LastSeen = now
		ns.neighbors[n.IPv6] = n
	}

	return nil
}

// readDefaultRouters returns the next hops of IPv6 default routes. A failure
// here is not fatal: it costs the router flag, not the neighbour list.
func (ns *NDPScanner) readDefaultRouters(ctx context.Context) map[string]bool {
	routers := make(map[string]bool)

	out, err := exec.CommandContext(ctx, "powershell", "-NoProfile", "-Command",
		windowsRouterQuery).Output()
	if err != nil {
		return routers
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if hop := strings.TrimSpace(line); hop != "" && hop != "::" {
			routers[strings.ToLower(hop)] = true
		}
	}

	return routers
}

// parseNetNeighbors turns Get-NetNeighbor CSV into neighbours for one interface.
//
// Split out from the command so it is testable without a neighbour table to
// stand in front of.
func parseNetNeighbors(csvOut, ifaceName string, routers map[string]bool) []NDPNeighbor {
	var out []NDPNeighbor

	for i, line := range strings.Split(csvOut, "\n") {
		if i == 0 {
			continue // header
		}
		fields := splitCSVRow(strings.TrimRight(line, "\r"))
		if len(fields) < neighborCSVFieldCount {
			continue
		}
		alias, addr, mac, state := fields[0], fields[1], fields[2], fields[3]
		if alias != ifaceName {
			continue
		}
		ip := net.ParseIP(addr)
		// Multicast entries are group memberships rather than hosts, and their
		// link-layer address is derived from the group. They are not neighbours.
		if ip == nil || ip.To4() != nil || ip.IsMulticast() {
			continue
		}
		out = append(out, NDPNeighbor{
			IPv6:     ip.String(),
			MAC:      normalizeWindowsMAC(mac),
			IsRouter: routers[strings.ToLower(ip.String())],
			State:    windowsNeighborState(state),
		})
	}

	return out
}

// splitCSVRow splits one ConvertTo-Csv row. The fields are quoted and none of
// the four contain a comma, so a full CSV reader would be ceremony.
func splitCSVRow(row string) []string {
	parts := strings.Split(row, ",")
	for i, p := range parts {
		parts[i] = strings.Trim(strings.TrimSpace(p), `"`)
	}

	return parts
}

// normalizeWindowsMAC converts Windows' 74-AC-B9-3B-AF-40 to Go's colon form,
// so a neighbour reads the same whichever platform reported it. The all-zero
// address Windows reports for an unresolved entry becomes empty, which is what
// the Linux path yields for the same state.
func normalizeWindowsMAC(mac string) string {
	hw, err := net.ParseMAC(strings.ReplaceAll(mac, "-", ":"))
	if err != nil {
		return ""
	}
	for _, b := range hw {
		if b != 0 {
			return hw.String()
		}
	}

	return ""
}

// windowsNeighborState maps Windows neighbour states onto the NUD vocabulary
// the Linux scanner reports, so a caller reads one set of names.
func windowsNeighborState(state string) string {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "incomplete":
		return "INCOMPLETE"
	case "reachable":
		return "REACHABLE"
	case "stale":
		return "STALE"
	case "delay":
		return "DELAY"
	case "probe":
		return "PROBE"
	case "unreachable":
		return "FAILED"
	case "permanent":
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
