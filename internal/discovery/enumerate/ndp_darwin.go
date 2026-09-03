//go:build darwin

package enumerate

// NDP (Neighbor Discovery Protocol) support for macOS is a stub implementation
// as IPv6 neighbor discovery on macOS is complex and the primary production target is Linux.

import (
	"context"
	"errors"
	"net"
	"os/exec"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// NDPScanner is a stub for macOS (production target is Linux).
type NDPScanner struct {
	interfaceName string
	neighbors     map[string]*NDPNeighbor
}

// NDPNeighbor represents an IPv6 neighbor.
type NDPNeighbor struct {
	IPv6     string
	MAC      string
	IsRouter bool
	State    string
	LastSeen time.Time
}

// NewNDPScanner creates a new IPv6 NDP scanner.
func NewNDPScanner(interfaceName string) *NDPScanner {
	return &NDPScanner{
		interfaceName: interfaceName,
		neighbors:     make(map[string]*NDPNeighbor),
	}
}

// Start is a stub on macOS.
func (ns *NDPScanner) Start() error {
	return errors.New("IPv6 NDP scanning not implemented on macOS (production target is Linux)")
}

// Stop is a stub on macOS.
func (ns *NDPScanner) Stop() error {
	return nil
}

// IsRunning returns false on macOS.
func (ns *NDPScanner) IsRunning() bool {
	return false
}

// ndpCommandTimeout bounds the ndp call so a hung command cannot stall the
// scan loop, which runs on a ticker.
const ndpCommandTimeout = 15 * time.Second

const (
	// ndpMinFields is the column count of a row carrying no flags:
	// Neighbor, Linklayer, Netif, Expire, St.
	ndpMinFields = 5
	// ndpFlagsField is the index of the Flgs column, present only on rows
	// that have flags at all.
	ndpFlagsField = 5
	// macOctets is how many octets a link-layer address must split into.
	macOctets = 6
)

// GetNeighbors returns the IPv6 neighbours read from the kernel's table.
//
// This used to return an empty map (#2089). It shells out where the ARP scanner
// reads the routing socket, and that asymmetry is no longer justified: the
// measurement it rested on was taken from a process the Go tool spawned, which
// macOS answers with placeholder link-layer addresses (#2272). Closing the gap
// means reconciling what the RIB reports against what `ndp -an` does, which is
// #2336 rather than a change to make in passing.
func (ns *NDPScanner) GetNeighbors() map[string]*NDPNeighbor {
	ctx, cancel := context.WithTimeout(context.Background(), ndpCommandTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "ndp", "-an").Output()
	if err != nil {
		logging.GetLogger().Error("IPv6 neighbour read failed", "error", err)

		return map[string]*NDPNeighbor{}
	}

	return parseNDPTable(string(out), ns.interfaceName)
}

// parseNDPTable turns `ndp -an` output into neighbours for one interface.
//
// Split from the command so it is testable without a neighbour table in front
// of it. An empty ifaceName returns every interface's entries.
func parseNDPTable(out, ifaceName string) map[string]*NDPNeighbor {
	neighbors := make(map[string]*NDPNeighbor)
	now := time.Now()

	for i, line := range strings.Split(out, "\n") {
		if i == 0 {
			continue // header
		}
		fields := strings.Fields(line)
		// Neighbor, Linklayer, Netif, Expire, St -- and optionally Flgs, Prbs.
		if len(fields) < ndpMinFields {
			continue
		}

		// The address carries its zone as fe80::1%lo0; the zone is the
		// interface, which is already its own column.
		addr, _, _ := strings.Cut(fields[0], "%")
		ip := net.ParseIP(addr)
		if ip == nil || ip.To4() != nil {
			continue
		}

		netif := fields[2]
		if ifaceName != "" && netif != ifaceName {
			continue
		}

		neighbors[ip.String()] = &NDPNeighbor{
			IPv6: ip.String(),
			MAC:  parseNDPLinklayer(fields[1]),
			// The Flgs column carries R for a router; it is absent on rows
			// that have no flags at all, which is why it is read by index
			// rather than assumed present.
			IsRouter: len(fields) > ndpFlagsField && strings.Contains(fields[ndpFlagsField], "R"),
			State:    ndpStateFromFlag(fields[4]),
			LastSeen: now,
		}
	}

	return neighbors
}

// parseNDPLinklayer normalises ndp's link-layer column. It prints
// "(incomplete)" when unresolved and does not zero-pad octets, so
// 3a:f0:5a:8b:c5:4 has to be widened before [net.ParseMAC] will take it.
func parseNDPLinklayer(field string) string {
	if strings.HasPrefix(field, "(") {
		return ""
	}

	octets := strings.Split(field, ":")
	if len(octets) != macOctets {
		return ""
	}
	for i, o := range octets {
		if len(o) == 1 {
			octets[i] = "0" + o
		}
	}
	hw, err := net.ParseMAC(strings.Join(octets, ":"))
	if err != nil {
		return ""
	}

	return hw.String()
}

// ndpStateFromFlag maps ndp's single-letter state onto the NUD vocabulary the
// Linux scanner reports, so a caller reads one set of names whichever platform
// answered.
func ndpStateFromFlag(state string) string {
	switch state {
	case "R":
		return "REACHABLE"
	case "S":
		return "STALE"
	case "D":
		return "DELAY"
	case "P":
		return "PROBE"
	case "I":
		return "INCOMPLETE"
	case "N":
		return "FAILED"
	default:
		return "UNKNOWN"
	}
}

// CleanupStale is a no-op on macOS.
func (ns *NDPScanner) CleanupStale(_ time.Duration) {
	// No-op
}
