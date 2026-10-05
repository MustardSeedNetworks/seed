//go:build darwin

package enumerate

// macOS has no passive NDP listener; the scanner reads the kernel's neighbour
// cache on demand instead.

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/net/route"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// NDPScanner reads the IPv6 neighbour cache on macOS.
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

// Start reports that macOS has no passive NDP listener; GetNeighbors reads the
// table directly.
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

const (
	// macOctets is the length of an Ethernet link-layer address.
	macOctets = 6
	// locallyAdministeredBit is the low-order bit of a MAC's first octet.
	// macOS sets it alone to stand in for an address it never resolved.
	locallyAdministeredBit = 0x02
)

// in6NbrInfo mirrors struct in6_nbrinfo from <netinet6/nd6.h> on LP64.
type in6NbrInfo struct {
	ifname   [syscall.IFNAMSIZ]byte
	addr     [16]byte
	asked    int64
	isRouter int32
	state    int32
	expire   int32
	_        int32 // tail padding to the struct's 8-byte alignment
}

// siocgnbrinfoIn6 is SIOCGNBRINFO_IN6, _IOWR('i', 78, struct in6_nbrinfo).
// Neither syscall nor x/sys/unix exports it. Built from the struct's size so
// the request code cannot drift from the layout it describes.
const siocgnbrinfoIn6 = 0xc0000000 | (unsafe.Sizeof(in6NbrInfo{})&0x1fff)<<16 | 'i'<<8 | 78

// ND6_LLINFO_* from <netinet6/nd6.h>: the neighbour states the kernel reports.
const (
	nd6LLInfoIncomplete = 0
	nd6LLInfoReachable  = 1
	nd6LLInfoStale      = 2
	nd6LLInfoDelay      = 3
	nd6LLInfoProbe      = 4
)

// GetNeighbors returns the IPv6 neighbours read from the kernel's table.
func (ns *NDPScanner) GetNeighbors() map[string]*NDPNeighbor {
	neighbors, err := readNDPTable(ns.interfaceName)
	if err != nil {
		logging.GetLogger().Error("IPv6 neighbour read failed", "error", err)

		return map[string]*NDPNeighbor{}
	}

	return neighbors
}

// readNDPTable reads the IPv6 neighbour cache the way `ndp -an` does, without
// running it (#2336, #2530): the routing socket lists the entries, and
// SIOCGNBRINFO_IN6 supplies each one's state and router flag, which the
// routing message does not carry. An empty ifaceName returns every
// interface's entries.
//
// Reconciled against `ndp -an` on Darwin 27.2, 2026-10-05: 23 routing
// messages, 22 `ndp -an` rows. The extra message is ::1, whose gateway is not
// a link address. Of the 22 rows, 14 are the host's own addresses (RTF_LOCAL,
// printed "permanent"), which the Linux reader never sees either, and 8 are
// neighbours, with the same link addresses, states and router flag as ndp.
func readNDPTable(ifaceName string) (map[string]*NDPNeighbor, error) {
	rib, err := route.FetchRIB(syscall.AF_INET6, ribTypeFlags, syscall.RTF_LLINFO)
	if err != nil {
		return nil, fmt.Errorf("fetch routing table: %w", err)
	}

	msgs, err := route.ParseRIB(ribTypeFlags, rib)
	if err != nil {
		return nil, fmt.Errorf("parse routing table: %w", err)
	}

	fd, err := syscall.Socket(syscall.AF_INET6, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("open neighbour query socket: %w", err)
	}
	defer func() { _ = syscall.Close(fd) }()

	now := time.Now()
	neighbors := make(map[string]*NDPNeighbor)
	for _, msg := range msgs {
		rm, ok := msg.(*route.RouteMessage)
		if !ok {
			continue
		}
		row, ok := ndpRowFromRoute(rm)
		if !ok || (ifaceName != "" && row.iface != ifaceName) {
			continue
		}

		n := &NDPNeighbor{
			IPv6:     row.ip.String(),
			MAC:      row.mac,
			State:    "UNKNOWN",
			LastSeen: now,
		}
		// The entry can expire between the two reads; ndp prints such a row
		// without a state, and so does this.
		if info, infoErr := neighbourInfo(fd, row); infoErr == nil {
			n.State = nd6StateName(info.state)
			n.IsRouter = info.isRouter != 0
		}
		neighbors[n.IPv6] = n
	}

	return neighbors, nil
}

// ndpRow is one neighbour-cache entry as the routing socket reports it.
type ndpRow struct {
	ip    net.IP
	zone  int
	iface string
	mac   string
}

// ndpRowFromRoute turns one link-layer route into a neighbour-cache row, or
// reports false when the route is not a neighbour.
//
// An unresolved entry is kept with no MAC, as ndp prints "(incomplete)" for
// it: unlike the ARP reader, an IPv6 neighbour that has not answered is still
// one the kernel is resolving.
func ndpRowFromRoute(rm *route.RouteMessage) (ndpRow, bool) {
	// The host's own addresses are in this table too, routed over lo0.
	if rm.Flags&syscall.RTF_LOCAL != 0 || len(rm.Addrs) <= syscall.RTAX_GATEWAY {
		return ndpRow{}, false
	}

	dst, ok := rm.Addrs[syscall.RTAX_DST].(*route.Inet6Addr)
	if !ok {
		return ndpRow{}, false
	}
	ip := net.IP(dst.IP[:])
	if ip.IsMulticast() || ip.IsUnspecified() {
		return ndpRow{}, false
	}

	link, ok := rm.Addrs[syscall.RTAX_GATEWAY].(*route.LinkAddr)
	if !ok {
		return ndpRow{}, false
	}

	// The link address's index is the neighbour's interface; the message's
	// own index is lo0 for some entries, which is why ndp reads this one.
	iface := link.Name
	if iface == "" && link.Index > 0 {
		if byIndex, err := net.InterfaceByIndex(link.Index); err == nil {
			iface = byIndex.Name
		}
	}

	return ndpRow{ip: ip, zone: dst.ZoneID, iface: iface, mac: ndpLinkLayer(link.Addr)}, true
}

// ndpLinkLayer returns a neighbour's MAC, or "" when the kernel holds none.
//
// 02:00:00:00:00:00 is what macOS hands a filtered reader for an entry whose
// link layer it will not disclose (ADR-0030) -- the locally-administered bit
// over an otherwise empty address. It parses as a MAC, so it is rejected by
// value, the same way the Windows ARP reader rejects the all-zero address
// (#2337).
func ndpLinkLayer(addr []byte) string {
	if len(addr) != macOctets {
		return ""
	}
	for i, octet := range addr {
		if i == 0 {
			octet &^= locallyAdministeredBit
		}
		if octet != 0 {
			return net.HardwareAddr(addr).String()
		}
	}

	return ""
}

// neighbourInfo asks the kernel for one entry's state, as ndp's getnbrinfo
// does. A link-local address goes in with its scope embedded in the second
// 16-bit word, which is the KAME form the kernel matches on.
func neighbourInfo(fd int, row ndpRow) (in6NbrInfo, error) {
	var req in6NbrInfo
	copy(req.ifname[:], row.iface)
	copy(req.addr[:], row.ip.To16())
	if row.ip.IsLinkLocalUnicast() {
		if row.zone <= 0 || row.zone > math.MaxUint16 {
			return in6NbrInfo{}, fmt.Errorf("link-local %s has no usable scope %d", row.ip, row.zone)
		}
		binary.BigEndian.PutUint16(req.addr[2:4], uint16(row.zone))
	}

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), siocgnbrinfoIn6, uintptr(unsafe.Pointer(&req)))
	if errno != 0 {
		return in6NbrInfo{}, errno
	}

	return req, nil
}

// nd6StateName maps the kernel's ND6_LLINFO state onto the NUD vocabulary the
// Linux scanner reports, so a caller reads one set of names whichever platform
// answered.
func nd6StateName(state int32) string {
	switch state {
	case nd6LLInfoIncomplete:
		return "INCOMPLETE"
	case nd6LLInfoReachable:
		return "REACHABLE"
	case nd6LLInfoStale:
		return "STALE"
	case nd6LLInfoDelay:
		return "DELAY"
	case nd6LLInfoProbe:
		return "PROBE"
	default:
		return "UNKNOWN"
	}
}

// CleanupStale is a no-op on macOS.
func (ns *NDPScanner) CleanupStale(_ time.Duration) {
	// No-op
}
