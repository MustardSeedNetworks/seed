package snmp

import (
	"context"
	"errors"
	"fmt"
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

// Routing table OID parsing constants.
const (
	// minOIDPartsInetCidrRoute is the minimum OID parts for modern inetCidrRouteTable entries.
	// Format includes: OID base + destType + destLen + dest(4-16) + pfxLen + policy + nextHopType + nextHopLen + nextHop.
	minOIDPartsInetCidrRoute = 12
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
// It tries the modern inetCidrRouteTable first, then falls back to legacy ipCidrRouteTable.
func GetRoutes(ctx context.Context, ip string, cfg *Session) (RouteTable, error) {
	if cfg == nil {
		return RouteTable{}, errors.New("SNMP config is nil")
	}

	// Try modern inetCidrRouteTable first.
	table, err := getInetCidrRoutes(ctx, ip, cfg)
	if err == nil && len(table.Routes) > 0 {
		return table, nil
	}

	// Fall back to legacy ipCidrRouteTable.
	return getIPCidrRoutes(ctx, ip, cfg)
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

		key := fmt.Sprintf("%s/%d-%s", dest, prefix, nextHop)
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
	walkRouteAttribute(params, OIDInetCidrRouteType, limit, routes, func(r *RouteEntry, value string) {
		r.Type = parseRouteType(value)
	})

	// Walk route protocol.
	walkRouteAttribute(params, OIDInetCidrRouteProto, limit, routes, func(r *RouteEntry, value string) {
		r.Protocol = parseRouteProtocol(value)
	})

	// Walk route metric.
	walkRouteAttribute(params, OIDInetCidrRouteMetric1, limit, routes, func(r *RouteEntry, value string) {
		r.Metric, _ = strconv.Atoi(value)
	})

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

		key := fmt.Sprintf("%s/%s-%s", dest, mask, nextHop)
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
	walkIPCidrRouteAttribute(
		params,
		OIDIpCidrRouteIfIndex,
		limit,
		routes,
		func(r *RouteEntry, value string) {
			r.IfIndex, _ = strconv.Atoi(value)
		},
	)

	// Walk ipCidrRouteType.
	walkIPCidrRouteAttribute(params, OIDIpCidrRouteType, limit, routes, func(r *RouteEntry, value string) {
		r.Type = parseRouteType(value)
	})

	// Walk ipCidrRouteProto.
	walkIPCidrRouteAttribute(
		params,
		OIDIpCidrRouteProto,
		limit,
		routes,
		func(r *RouteEntry, value string) {
			r.Protocol = parseRouteProtocol(value)
		},
	)

	// Walk ipCidrRouteMetric1.
	walkIPCidrRouteAttribute(
		params,
		OIDIpCidrRouteMetric1,
		limit,
		routes,
		func(r *RouteEntry, value string) {
			r.Metric, _ = strconv.Atoi(value)
		},
	)

	return routeTable(routes, truncated), nil
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

// walkRouteAttribute walks a routing table attribute (inetCidrRouteTable).
func walkRouteAttribute(
	params bulkWalker,
	oid string,
	limit int,
	routes map[string]*RouteEntry,
	updateFunc func(*RouteEntry, string),
) {
	_, err := walkCapped(params, oid, limit, func(pdu gosnmp.SnmpPDU) {
		dest, prefix, nextHop := parseInetCidrRouteIndex(pdu.Name)
		if dest == "" {
			return
		}

		key := fmt.Sprintf("%s/%d-%s", dest, prefix, nextHop)
		route, exists := routes[key]
		if !exists {
			return
		}

		updateFunc(route, formatSNMPValue(pdu))
	})
	if err != nil {
		logging.GetLogger().Debug("Failed to walk route attribute", "oid", oid, "error", err)
	}
}

// walkIPCidrRouteAttribute walks a routing table attribute (ipCidrRouteTable).
func walkIPCidrRouteAttribute(
	params bulkWalker,
	oid string,
	limit int,
	routes map[string]*RouteEntry,
	updateFunc func(*RouteEntry, string),
) {
	_, err := walkCapped(params, oid, limit, func(pdu gosnmp.SnmpPDU) {
		dest, mask, nextHop := parseIPCidrRouteIndex(pdu.Name)
		if dest == "" {
			return
		}

		key := fmt.Sprintf("%s/%s-%s", dest, mask, nextHop)
		route, exists := routes[key]
		if !exists {
			return
		}

		updateFunc(route, formatSNMPValue(pdu))
	})
	if err != nil {
		logging.GetLogger().
			Debug("Failed to walk IP CIDR route attribute", "oid", oid, "error", err)
	}
}

// parseInetCidrRouteIndex extracts destination, prefix, and next hop from OID.
// OID format: ...destType.destLen.dest.pfxLen.policy.nextHopType.nextHopLen.nextHop.
func parseInetCidrRouteIndex(oid string) (string, int, string) {
	parts := strings.Split(oid, ".")
	if len(parts) < minOIDPartsInetCidrRoute {
		return "", 0, ""
	}

	// Find the starting point - look for address type (1=ipv4, 2=ipv6)
	// This is complex because the OID embeds variable-length addresses
	// For simplicity, we'll try to parse IPv4 addresses which have predictable format

	for i := len(parts) - 1; i >= 10; i-- {
		destType, err := strconv.Atoi(parts[i-10])
		if err != nil || destType != 1 {
			continue
		}

		destLen, err := strconv.Atoi(parts[i-9])
		if err != nil || destLen != 4 {
			continue
		}

		// Extract destination (4 octets)
		if i-8+4 > len(parts) {
			continue
		}
		dest := strings.Join(parts[i-8:i-4], ".")

		// Extract prefix length
		if i-4 >= len(parts) {
			continue
		}
		prefix, _ := strconv.Atoi(parts[i-4])

		// For simplicity, use destination as next hop placeholder
		// Real implementation would parse the full next hop from OID
		nextHop := "0.0.0.0"

		return dest, prefix, nextHop
	}

	return "", 0, ""
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
