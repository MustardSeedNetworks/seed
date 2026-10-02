package snmp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/MustardSeedNetworks/seed/internal/logging"
)

// IP-FORWARD-MIB OIDs (RFC 4292).
const (
	// OIDInetCidrRouteDest is the IP-FORWARD-MIB OID for route destination (modern, IPv4/IPv6).
	OIDInetCidrRouteDest = "1.3.6.1.2.1.4.24.7.1.1"
	// OIDInetCidrRouteIfIndex is the IP-FORWARD-MIB OID for route interface index.
	OIDInetCidrRouteIfIndex = "1.3.6.1.2.1.4.24.7.1.7"
	// OIDInetCidrRouteType is the IP-FORWARD-MIB OID for route type.
	OIDInetCidrRouteType = "1.3.6.1.2.1.4.24.7.1.8"
	// OIDInetCidrRouteProto is the IP-FORWARD-MIB OID for route protocol.
	OIDInetCidrRouteProto = "1.3.6.1.2.1.4.24.7.1.9"
	// OIDInetCidrRouteNextHop is the IP-FORWARD-MIB OID for route next hop.
	OIDInetCidrRouteNextHop = "1.3.6.1.2.1.4.24.7.1.4"
	// OIDInetCidrRouteMetric1 is the IP-FORWARD-MIB OID for route metric.
	OIDInetCidrRouteMetric1 = "1.3.6.1.2.1.4.24.7.1.12"

	// OIDIpCidrRouteDest is the IP-FORWARD-MIB OID for route destination (legacy, IPv4 only).
	OIDIpCidrRouteDest = "1.3.6.1.2.1.4.24.4.1.1"
	// OIDIpCidrRouteMask is the IP-FORWARD-MIB OID for route mask.
	OIDIpCidrRouteMask = "1.3.6.1.2.1.4.24.4.1.2"
	// OIDIpCidrRouteNextHop is the IP-FORWARD-MIB OID for route next hop.
	OIDIpCidrRouteNextHop = "1.3.6.1.2.1.4.24.4.1.4"
	// OIDIpCidrRouteIfIndex is the IP-FORWARD-MIB OID for route interface index.
	OIDIpCidrRouteIfIndex = "1.3.6.1.2.1.4.24.4.1.5"
	// OIDIpCidrRouteType is the IP-FORWARD-MIB OID for route type.
	OIDIpCidrRouteType = "1.3.6.1.2.1.4.24.4.1.6"
	// OIDIpCidrRouteProto is the IP-FORWARD-MIB OID for route protocol.
	OIDIpCidrRouteProto = "1.3.6.1.2.1.4.24.4.1.7"
	// OIDIpCidrRouteMetric1 is the IP-FORWARD-MIB OID for route metric.
	OIDIpCidrRouteMetric1 = "1.3.6.1.2.1.4.24.4.1.11"
)

// RFC1213-MIB ipRouteTable OIDs. The table is deprecated, but plenty of
// devices serve only it, and an EtherScope reads it alongside the CIDR tables
// (seed#2587).
const (
	// OIDIpRouteDest is the ipRouteTable OID for route destination.
	OIDIpRouteDest = "1.3.6.1.2.1.4.21.1.1"
	// OIDIpRouteIfIndex is the ipRouteTable OID for route interface index.
	OIDIpRouteIfIndex = "1.3.6.1.2.1.4.21.1.2"
	// OIDIpRouteMetric1 is the ipRouteTable OID for route metric.
	OIDIpRouteMetric1 = "1.3.6.1.2.1.4.21.1.3"
	// OIDIpRouteNextHop is the ipRouteTable OID for route next hop.
	OIDIpRouteNextHop = "1.3.6.1.2.1.4.21.1.7"
	// OIDIpRouteType is the ipRouteTable OID for route type.
	OIDIpRouteType = "1.3.6.1.2.1.4.21.1.8"
	// OIDIpRouteProto is the ipRouteTable OID for route protocol.
	OIDIpRouteProto = "1.3.6.1.2.1.4.21.1.9"
	// OIDIpRouteMask is the ipRouteTable OID for route mask.
	OIDIpRouteMask = "1.3.6.1.2.1.4.21.1.11"
)

// ipRouteTypeInvalid is RFC 1213's ipRouteType invalid(2): an entry the agent
// has deleted but may still list.
const ipRouteTypeInvalid = "2"

// inetCidrRouteEntry is the IP-FORWARD-MIB row every inetCidrRouteTable
// column OID starts with; InetAddressType values are RFC 4001's.
const (
	inetCidrRouteEntry = "1.3.6.1.2.1.4.24.7.1"

	inetAddressUnknown = 0
	inetAddressIPv4    = 1
	inetAddressIPv6    = 2
)

// Routing table OID parsing constants.
const (
	// minOIDPartsIPCidrRoute is the minimum OID parts for legacy ipCidrRouteTable entries.
	// Format includes: OID base + 4 dest octets + 4 mask octets + 1 TOS + 4 nextHop octets = 14 parts minimum.
	minOIDPartsIPCidrRoute = 14
	// ipCidrRouteIndexOffset is the number of parts from the end of OID to the start of the route index.
	// (4 dest + 4 mask + 1 TOS + 4 nextHop = 13 parts from end).
	ipCidrRouteIndexOffset = 13
)

// MaxRouteRows bounds one route-table walk. An edge router carrying a full
// BGP table holds about a million routes, and the walk runs once per profiled
// device, so an unbounded walk puts the whole table in memory for one profile
// (seed#2833). Ten thousand rows is far more than any site's own networks;
// the rows past it are the internet's, which nothing here uses.
const MaxRouteRows = 10_000

// errRouteRowCap stops a BulkWalk once it has read MaxRouteRows rows. gosnmp
// ends a walk when the callback returns an error and hands that error back,
// so walkCapped recognises it as the cap rather than a failure.
var errRouteRowCap = errors.New("route row cap reached")

// bulkWalker is the part of a gosnmp session a table walk uses.
type bulkWalker interface {
	BulkWalk(rootOid string, walkFn gosnmp.WalkFunc) error
}

// RouteTable is a device's forwarding table as far as it was read. Truncated
// means the device held more than MaxRouteRows rows and the rest were not
// walked, so Routes is a prefix of the table rather than all of it.
type RouteTable struct {
	Routes    []RouteEntry
	Truncated bool
}

// RouteEntry contains routing table information from IP-FORWARD-MIB.
type RouteEntry struct {
	Destination string // Destination network
	Prefix      int    // Prefix length (CIDR notation)
	NextHop     string // Next hop address
	IfIndex     int    // Output interface index
	Type        string // local, remote, blackhole, other
	Protocol    string // static, ospf, bgp, rip, connected, etc.
	Metric      int    // Route metric
}

// GetRoutes retrieves routing table from a device using IP-FORWARD-MIB, at
// most MaxRouteRows rows of it.
// It reads the first of inetCidrRouteTable (RFC 4292), ipCidrRouteTable
// (RFC 2096) and ipRouteTable (RFC 1213) that holds any rows.
func GetRoutes(ctx context.Context, ip string, cfg *Session) (RouteTable, error) {
	if cfg == nil {
		return RouteTable{}, errors.New("SNMP config is nil")
	}

	return firstRouteTable(ctx, ip, cfg, getInetCidrRoutes, getIPCidrRoutes, getIPRoutes)
}

// routeReader reads one of the route tables.
type routeReader func(ctx context.Context, ip string, cfg *Session) (RouteTable, error)

// firstRouteTable returns the first table that holds rows, in the order the
// readers are given. When none does, the last reader's answer stands, so a
// device that serves no route table reports either nothing or the failure.
func firstRouteTable(ctx context.Context, ip string, cfg *Session, readers ...routeReader) (RouteTable, error) {
	last := len(readers) - 1
	for _, read := range readers[:last] {
		if table, err := read(ctx, ip, cfg); err == nil && len(table.Routes) > 0 {
			return table, nil
		}
	}
	return readers[last](ctx, ip, cfg)
}

// getInetCidrRoutes retrieves routes from the modern inetCidrRouteTable.
func getInetCidrRoutes(
	ctx context.Context,
	ip string,
	cfg *Session,
) (RouteTable, error) {
	return sweepCredentials(ctx, cfg, "failed to query inetCidrRouteTable with all configured credentials",
		func(cred *V3Credential) (RouteTable, error) {
			return walkInetCidrRoutesV3(ctx, ip, cred, cfg)
		},
		func(community string) (RouteTable, error) {
			return walkInetCidrRoutes(ctx, ip, community, cfg)
		},
	)
}

// getIPCidrRoutes retrieves routes from the legacy ipCidrRouteTable.
func getIPCidrRoutes(ctx context.Context, ip string, cfg *Session) (RouteTable, error) {
	return sweepCredentials(ctx, cfg, "failed to query ipCidrRouteTable with all configured credentials",
		func(cred *V3Credential) (RouteTable, error) {
			return walkIPCidrRoutesV3(ctx, ip, cred, cfg)
		},
		func(community string) (RouteTable, error) {
			return walkIPCidrRoutes(ctx, ip, community, cfg)
		},
	)
}

// walkInetCidrRoutes walks the modern inetCidrRouteTable using SNMPv2c.
func walkInetCidrRoutes(
	ctx context.Context,
	ip, community string,
	cfg *Session,
) (RouteTable, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return RouteTable{}, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkInetCidrRouteTable(params, MaxRouteRows)
}

// walkInetCidrRoutesV3 walks the modern inetCidrRouteTable using SNMPv3.
func walkInetCidrRoutesV3(
	ctx context.Context,
	ip string,
	cred *V3Credential,
	cfg *Session,
) (RouteTable, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return RouteTable{}, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkInetCidrRouteTable(params, MaxRouteRows)
}

// walkInetCidrRouteTable walks the modern inetCidrRouteTable, at most limit
// rows of it.
func walkInetCidrRouteTable(params bulkWalker, limit int) (RouteTable, error) {
	routes := make(map[string]*RouteEntry)

	// Walk inetCidrRouteIfIndex to discover routes.
	truncated, err := walkCapped(params, OIDInetCidrRouteIfIndex, limit, func(pdu gosnmp.SnmpPDU) {
		dest, prefix, nextHop := parseInetCidrRouteIndex(pdu.Name)
		if dest == "" {
			return
		}

		key := inetCidrRouteKey(pdu.Name)
		ifIndex, _ := strconv.Atoi(formatSNMPValue(pdu))

		routes[key] = &RouteEntry{
			Destination: dest,
			Prefix:      prefix,
			NextHop:     nextHop,
			IfIndex:     ifIndex,
		}
	})
	if err != nil {
		return RouteTable{}, fmt.Errorf("failed to walk inetCidrRouteIfIndex: %w", err)
	}

	// Walk route type.
	walkRouteColumn(params, OIDInetCidrRouteType, limit, routes, inetCidrRouteKey, func(r *RouteEntry, value string) {
		r.Type = parseRouteType(value)
	})

	// Walk route protocol.
	walkRouteColumn(params, OIDInetCidrRouteProto, limit, routes, inetCidrRouteKey, func(r *RouteEntry, value string) {
		r.Protocol = parseRouteProtocol(value)
	})

	// Walk route metric.
	walkRouteColumn(
		params,
		OIDInetCidrRouteMetric1,
		limit,
		routes,
		inetCidrRouteKey,
		func(r *RouteEntry, value string) {
			r.Metric, _ = strconv.Atoi(value)
		},
	)

	return routeTable(routes, truncated), nil
}

// walkIPCidrRoutes walks the legacy ipCidrRouteTable using SNMPv2c.
func walkIPCidrRoutes(
	ctx context.Context,
	ip, community string,
	cfg *Session,
) (RouteTable, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return RouteTable{}, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkIPCidrRouteTable(params, MaxRouteRows)
}

// walkIPCidrRoutesV3 walks the legacy ipCidrRouteTable using SNMPv3.
func walkIPCidrRoutesV3(
	ctx context.Context,
	ip string,
	cred *V3Credential,
	cfg *Session,
) (RouteTable, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return RouteTable{}, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkIPCidrRouteTable(params, MaxRouteRows)
}

// walkIPCidrRouteTable walks the legacy ipCidrRouteTable, at most limit rows
// of it.
func walkIPCidrRouteTable(params bulkWalker, limit int) (RouteTable, error) {
	routes := make(map[string]*RouteEntry)

	// Walk ipCidrRouteDest to discover routes.
	truncated, err := walkCapped(params, OIDIpCidrRouteDest, limit, func(pdu gosnmp.SnmpPDU) {
		dest, mask, nextHop := parseIPCidrRouteIndex(pdu.Name)
		if dest == "" {
			return
		}

		key := ipCidrRouteKey(pdu.Name)
		prefix := netmaskToPrefix(mask)

		routes[key] = &RouteEntry{
			Destination: dest,
			Prefix:      prefix,
			NextHop:     nextHop,
		}
	})
	if err != nil {
		return RouteTable{}, fmt.Errorf("failed to walk ipCidrRouteDest: %w", err)
	}

	// Walk ipCidrRouteIfIndex.
	walkRouteColumn(
		params,
		OIDIpCidrRouteIfIndex,
		limit,
		routes,
		ipCidrRouteKey,
		func(r *RouteEntry, value string) {
			r.IfIndex, _ = strconv.Atoi(value)
		},
	)

	// Walk ipCidrRouteType.
	walkRouteColumn(params, OIDIpCidrRouteType, limit, routes, ipCidrRouteKey, func(r *RouteEntry, value string) {
		r.Type = parseRouteType(value)
	})

	// Walk ipCidrRouteProto.
	walkRouteColumn(
		params,
		OIDIpCidrRouteProto,
		limit,
		routes,
		ipCidrRouteKey,
		func(r *RouteEntry, value string) {
			r.Protocol = parseRouteProtocol(value)
		},
	)

	// Walk ipCidrRouteMetric1.
	walkRouteColumn(
		params,
		OIDIpCidrRouteMetric1,
		limit,
		routes,
		ipCidrRouteKey,
		func(r *RouteEntry, value string) {
			r.Metric, _ = strconv.Atoi(value)
		},
	)

	return routeTable(routes, truncated), nil
}

// getIPRoutes retrieves routes from the RFC 1213 ipRouteTable.
func getIPRoutes(ctx context.Context, ip string, cfg *Session) (RouteTable, error) {
	return sweepCredentials(ctx, cfg, "failed to query ipRouteTable with all configured credentials",
		func(cred *V3Credential) (RouteTable, error) {
			params, err := newV3WalkClient(ctx, ip, cred, cfg)
			if err != nil {
				return RouteTable{}, err
			}
			defer func() { _ = params.Conn.Close() }()
			return walkIPRouteTable(params, MaxRouteRows)
		},
		func(community string) (RouteTable, error) {
			params, err := newV2cWalkClient(ctx, ip, community, cfg)
			if err != nil {
				return RouteTable{}, err
			}
			defer func() { _ = params.Conn.Close() }()
			return walkIPRouteTable(params, MaxRouteRows)
		},
	)
}

// walkIPRouteTable walks the RFC 1213 ipRouteTable, at most limit rows of it.
// The table is indexed by destination alone, so the mask and next hop are
// columns rather than index parts. A row is kept only once both have arrived
// as IPv4 values: netmaskToPrefix would read a missing mask as /0 and turn
// the row into a default route. Rows the agent marks invalid are dropped.
func walkIPRouteTable(params bulkWalker, limit int) (RouteTable, error) {
	routes := make(map[string]*RouteEntry)

	truncated, err := walkCapped(params, OIDIpRouteDest, limit, func(pdu gosnmp.SnmpPDU) {
		if dest := parseIPRouteIndex(pdu.Name); dest != "" {
			routes[dest] = &RouteEntry{Destination: dest, Prefix: -1}
		}
	})
	if err != nil {
		return RouteTable{}, fmt.Errorf("failed to walk ipRouteDest: %w", err)
	}

	invalid := make(map[string]bool)
	columns := []struct {
		oid    string
		update func(*RouteEntry, string)
	}{
		{OIDIpRouteMask, func(r *RouteEntry, value string) { r.Prefix = maskPrefix(value) }},
		{OIDIpRouteNextHop, func(r *RouteEntry, value string) {
			if addr, parseErr := netip.ParseAddr(value); parseErr == nil && addr.Is4() {
				r.NextHop = addr.String()
			}
		}},
		{OIDIpRouteIfIndex, func(r *RouteEntry, value string) { r.IfIndex, _ = strconv.Atoi(value) }},
		{OIDIpRouteType, func(r *RouteEntry, value string) {
			invalid[r.Destination] = value == ipRouteTypeInvalid
			r.Type = parseRouteType(value)
		}},
		{OIDIpRouteProto, func(r *RouteEntry, value string) { r.Protocol = parseRouteProtocol(value) }},
		{OIDIpRouteMetric1, func(r *RouteEntry, value string) { r.Metric, _ = strconv.Atoi(value) }},
	}
	for _, column := range columns {
		walkRouteColumn(params, column.oid, limit, routes, parseIPRouteIndex, column.update)
	}

	for dest, route := range routes {
		if route.Prefix < 0 || route.NextHop == "" || invalid[dest] {
			delete(routes, dest)
		}
	}
	return routeTable(routes, truncated), nil
}

// parseIPRouteIndex extracts the destination from an ipRouteTable OID, whose
// index is the four destination octets.
func parseIPRouteIndex(oid string) string {
	parts := strings.Split(oid, ".")
	if len(parts) < ipv4OctetCount {
		return ""
	}
	addr, err := netip.ParseAddr(strings.Join(parts[len(parts)-ipv4OctetCount:], "."))
	if err != nil || !addr.Is4() {
		return ""
	}
	return addr.String()
}

// maskPrefix returns the prefix length of a contiguous IPv4 netmask, or -1
// when value is not one.
func maskPrefix(value string) int {
	addr, err := netip.ParseAddr(value)
	if err != nil || !addr.Is4() {
		return -1
	}
	ones, bits := net.IPMask(addr.AsSlice()).Size()
	if bits == 0 {
		return -1
	}
	return ones
}

// walkCapped walks one column, handing visit at most limit rows. It reports
// truncated when the column holds more, which it knows only by being offered
// row limit+1, so a table of exactly limit rows is not called truncated.
//
// Every column of a table is capped the same way, not only the one that
// discovers the rows: the columns share the table's index order, so the rows
// past the cap in an attribute column belong to routes that were never kept,
// and walking them would cost the same million-row round trips for nothing.
func walkCapped(params bulkWalker, oid string, limit int, visit func(gosnmp.SnmpPDU)) (bool, error) {
	rows := 0
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		if rows == limit {
			return errRouteRowCap
		}
		rows++
		visit(pdu)
		return nil
	})
	if errors.Is(err, errRouteRowCap) {
		return true, nil
	}
	return false, err
}

// routeTable flattens the walked rows.
func routeTable(routes map[string]*RouteEntry, truncated bool) RouteTable {
	result := make([]RouteEntry, 0, len(routes))
	for _, route := range routes {
		result = append(result, *route)
	}
	return RouteTable{Routes: result, Truncated: truncated}
}

// walkRouteColumn walks one attribute column of a route table and applies
// each value to the route its index names, by the same key the table's
// discovery walk stored the routes under.
func walkRouteColumn(
	params bulkWalker,
	oid string,
	limit int,
	routes map[string]*RouteEntry,
	key func(oid string) string,
	updateFunc func(*RouteEntry, string),
) {
	_, err := walkCapped(params, oid, limit, func(pdu gosnmp.SnmpPDU) {
		if route, exists := routes[key(pdu.Name)]; exists {
			updateFunc(route, formatSNMPValue(pdu))
		}
	})
	if err != nil {
		logging.GetLogger().Debug("Failed to walk route column", "oid", oid, "error", err)
	}
}

// inetCidrRouteKey keys an inetCidrRouteTable row by its index.
func inetCidrRouteKey(oid string) string {
	dest, prefix, nextHop := parseInetCidrRouteIndex(oid)
	if dest == "" {
		return ""
	}
	return fmt.Sprintf("%s/%d-%s", dest, prefix, nextHop)
}

// ipCidrRouteKey keys an ipCidrRouteTable row by its index.
func ipCidrRouteKey(oid string) string {
	dest, mask, nextHop := parseIPCidrRouteIndex(oid)
	if dest == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s-%s", dest, mask, nextHop)
}

// parseInetCidrRouteIndex reads an inetCidrRouteEntry OID's index (RFC 4292):
// destType.destLen.dest….pfxLen.policyLen.policy….nextHopType.nextHopLen.nextHop….
// Both addresses and the policy are length-prefixed, so the index is read
// front to back. Only IPv4 destinations are returned; a next hop of type
// unknown(0), which a connected route carries, comes back empty.
func parseInetCidrRouteIndex(oid string) (string, int, string) {
	rest, ok := strings.CutPrefix(strings.TrimPrefix(oid, "."), inetCidrRouteEntry+".")
	if !ok {
		return "", 0, ""
	}
	// The first sub-identifier is the column; the index follows it.
	r := indexReader{ids: strings.Split(rest, "."), pos: 1}
	dest, ok := r.inetAddress()
	if !ok || !dest.Is4() {
		return "", 0, ""
	}
	prefix, ok := r.next()
	if !ok || prefix > net.IPv4len*8 {
		return "", 0, ""
	}
	if policyLen, read := r.next(); !read || !r.skip(policyLen) {
		return "", 0, ""
	}
	nextHop, ok := r.inetAddress()
	if !ok || r.pos != len(r.ids) {
		return "", 0, ""
	}
	if !nextHop.IsValid() {
		return dest.String(), prefix, ""
	}
	return dest.String(), prefix, nextHop.String()
}

// indexReader walks the sub-identifiers of a table index.
type indexReader struct {
	ids []string
	pos int
}

func (r *indexReader) next() (int, bool) {
	if r.pos >= len(r.ids) {
		return 0, false
	}
	r.pos++
	id, err := strconv.Atoi(r.ids[r.pos-1])
	return id, err == nil && id >= 0
}

func (r *indexReader) skip(n int) bool {
	if n > len(r.ids)-r.pos {
		return false
	}
	r.pos += n
	return true
}

// inetAddress reads an InetAddressType then a length-prefixed InetAddress.
// Type unknown(0) with no octets is the zero Addr; IPv4 and IPv6 must carry
// their own length; any other type is refused.
func (r *indexReader) inetAddress() (netip.Addr, bool) {
	addrType, ok := r.next()
	if !ok {
		return netip.Addr{}, false
	}
	length, ok := r.next()
	if !ok || length > len(r.ids)-r.pos {
		return netip.Addr{}, false
	}
	octets := make([]byte, 0, length)
	for _, id := range r.ids[r.pos : r.pos+length] {
		octet, err := strconv.ParseUint(id, 10, 8)
		if err != nil {
			return netip.Addr{}, false
		}
		octets = append(octets, byte(octet))
	}
	r.pos += length
	switch {
	case addrType == inetAddressUnknown && length == 0:
		return netip.Addr{}, true
	case addrType == inetAddressIPv4 && length == net.IPv4len,
		addrType == inetAddressIPv6 && length == net.IPv6len:
		addr, _ := netip.AddrFromSlice(octets)
		return addr, true
	default:
		return netip.Addr{}, false
	}
}

// parseIPCidrRouteIndex extracts destination, mask, and next hop from OID.
// OID format: ...Dest.Mask.TOS.NextHop.
func parseIPCidrRouteIndex(oid string) (string, string, string) {
	parts := strings.Split(oid, ".")
	// Need at least: base OID + 4 dest + 4 mask + 1 tos + 4 nexthop = 13 parts after base
	if len(parts) < minOIDPartsIPCidrRoute {
		return "", "", ""
	}

	// Destination is last 13 parts starting at -13 to -10
	destStart := len(parts) - ipCidrRouteIndexOffset
	dest := strings.Join(parts[destStart:destStart+4], ".")

	// Mask is next 4 parts
	mask := strings.Join(parts[destStart+4:destStart+8], ".")

	// TOS is at destStart+8, skip it

	// NextHop is last 4 parts
	nextHop := strings.Join(parts[destStart+9:destStart+13], ".")

	return dest, mask, nextHop
}

// parseRouteType converts route type value to string.
func parseRouteType(value string) string {
	switch value {
	case "1":
		return MACTypeOther
	case "2":
		return "reject"
	case "3":
		return IDSubtypeLocal
	case "4":
		return "remote"
	case "5":
		return "blackhole"
	default:
		return StatusUnknown
	}
}

// parseRouteProtocol converts route protocol value to string.
func parseRouteProtocol(value string) string {
	switch value {
	case "1":
		return MACTypeOther
	case "2":
		return IDSubtypeLocal
	case "3":
		return "netmgmt" // static
	case "4":
		return "icmp"
	case "5":
		return "egp"
	case "6":
		return "ggp"
	case "7":
		return "hello"
	case "8":
		return "rip"
	case "9":
		return "is-is"
	case "10":
		return "es-is"
	case "11":
		return "ciscoIgrp"
	case "12":
		return "bbnSpfIgp"
	case "13":
		return "ospf"
	case "14":
		return "bgp"
	case "15":
		return "idpr"
	case "16":
		return "ciscoEigrp"
	default:
		return StatusUnknown
	}
}
