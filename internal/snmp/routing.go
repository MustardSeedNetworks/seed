// Package snmp provides SNMP query functionality for network device discovery.
// This file implements IP-FORWARD-MIB collection for L3 routing topology.
// Routing table information is essential for building network layer diagrams.
package snmp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"

	"github.com/krisarmstrong/seed/internal/config"
	"github.com/krisarmstrong/seed/internal/logging"
)

// IP-FORWARD-MIB OIDs (RFC 4292).
const (
	// inetCidrRouteTable - modern routing table (supports IPv4/IPv6).
	OIDInetCidrRouteDest    = "1.3.6.1.2.1.4.24.7.1.1"  // inetCidrRouteDest (index)
	OIDInetCidrRouteIfIndex = "1.3.6.1.2.1.4.24.7.1.7"  // inetCidrRouteIfIndex
	OIDInetCidrRouteType    = "1.3.6.1.2.1.4.24.7.1.8"  // inetCidrRouteType
	OIDInetCidrRouteProto   = "1.3.6.1.2.1.4.24.7.1.9"  // inetCidrRouteProto
	OIDInetCidrRouteNextHop = "1.3.6.1.2.1.4.24.7.1.4"  // inetCidrRouteNextHop (index)
	OIDInetCidrRouteMetric1 = "1.3.6.1.2.1.4.24.7.1.12" // inetCidrRouteMetric1

	// ipCidrRouteTable - legacy routing table (IPv4 only).
	OIDIpCidrRouteDest    = "1.3.6.1.2.1.4.24.4.1.1"  // ipCidrRouteDest
	OIDIpCidrRouteMask    = "1.3.6.1.2.1.4.24.4.1.2"  // ipCidrRouteMask
	OIDIpCidrRouteNextHop = "1.3.6.1.2.1.4.24.4.1.4"  // ipCidrRouteNextHop
	OIDIpCidrRouteIfIndex = "1.3.6.1.2.1.4.24.4.1.5"  // ipCidrRouteIfIndex
	OIDIpCidrRouteType    = "1.3.6.1.2.1.4.24.4.1.6"  // ipCidrRouteType
	OIDIpCidrRouteProto   = "1.3.6.1.2.1.4.24.4.1.7"  // ipCidrRouteProto
	OIDIpCidrRouteMetric1 = "1.3.6.1.2.1.4.24.4.1.11" // ipCidrRouteMetric1
)

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

// GetRoutes retrieves routing table from a device using IP-FORWARD-MIB.
// It tries the modern inetCidrRouteTable first, then falls back to legacy ipCidrRouteTable.
func GetRoutes(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]RouteEntry, error) {
	if cfg == nil {
		return nil, errors.New("SNMP config is nil")
	}

	// Try modern inetCidrRouteTable first.
	routes, err := getInetCidrRoutes(ctx, ip, cfg)
	if err == nil && len(routes) > 0 {
		return routes, nil
	}

	// Fall back to legacy ipCidrRouteTable.
	return getIPCidrRoutes(ctx, ip, cfg)
}

// getInetCidrRoutes retrieves routes from the modern inetCidrRouteTable.
func getInetCidrRoutes(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]RouteEntry, error) {
	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		routes, err := walkInetCidrRoutesV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return routes, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		routes, err := walkInetCidrRoutes(ctx, ip, community, cfg)
		if err == nil {
			return routes, nil
		}
	}

	return nil, errors.New("failed to query inetCidrRouteTable with all configured credentials")
}

// getIPCidrRoutes retrieves routes from the legacy ipCidrRouteTable.
func getIPCidrRoutes(ctx context.Context, ip string, cfg *config.SNMPConfig) ([]RouteEntry, error) {
	// Try SNMPv3 credentials first (more secure).
	for i := range cfg.V3Credentials {
		routes, err := walkIPCidrRoutesV3(ctx, ip, &cfg.V3Credentials[i], cfg)
		if err == nil {
			return routes, nil
		}
	}

	// Fall back to v2c community strings if v3 fails or not configured.
	for _, community := range cfg.Communities {
		routes, err := walkIPCidrRoutes(ctx, ip, community, cfg)
		if err == nil {
			return routes, nil
		}
	}

	return nil, errors.New("failed to query ipCidrRouteTable with all configured credentials")
}

// walkInetCidrRoutes walks the modern inetCidrRouteTable using SNMPv2c.
func walkInetCidrRoutes(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]RouteEntry, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkInetCidrRouteTable(params)
}

// walkInetCidrRoutesV3 walks the modern inetCidrRouteTable using SNMPv3.
func walkInetCidrRoutesV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]RouteEntry, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkInetCidrRouteTable(params)
}

// walkInetCidrRouteTable walks the modern inetCidrRouteTable.
func walkInetCidrRouteTable(params *gosnmp.GoSNMP) ([]RouteEntry, error) {
	routes := make(map[string]*RouteEntry)

	// Walk inetCidrRouteIfIndex to discover routes.
	err := params.BulkWalk(OIDInetCidrRouteIfIndex, func(pdu gosnmp.SnmpPDU) error {
		dest, prefix, nextHop := parseInetCidrRouteIndex(pdu.Name)
		if dest == "" {
			return nil
		}

		key := fmt.Sprintf("%s/%d-%s", dest, prefix, nextHop)
		ifIndex, _ := strconv.Atoi(formatSNMPValue(pdu))

		routes[key] = &RouteEntry{
			Destination: dest,
			Prefix:      prefix,
			NextHop:     nextHop,
			IfIndex:     ifIndex,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk inetCidrRouteIfIndex: %w", err)
	}

	// Walk route type.
	walkRouteAttribute(params, OIDInetCidrRouteType, routes, func(r *RouteEntry, value string) {
		r.Type = parseRouteType(value)
	})

	// Walk route protocol.
	walkRouteAttribute(params, OIDInetCidrRouteProto, routes, func(r *RouteEntry, value string) {
		r.Protocol = parseRouteProtocol(value)
	})

	// Walk route metric.
	walkRouteAttribute(params, OIDInetCidrRouteMetric1, routes, func(r *RouteEntry, value string) {
		r.Metric, _ = strconv.Atoi(value)
	})

	// Convert map to slice.
	result := make([]RouteEntry, 0, len(routes))
	for _, route := range routes {
		result = append(result, *route)
	}

	return result, nil
}

// walkIPCidrRoutes walks the legacy ipCidrRouteTable using SNMPv2c.
func walkIPCidrRoutes(ctx context.Context, ip, community string, cfg *config.SNMPConfig) ([]RouteEntry, error) {
	params, err := newV2cWalkClient(ctx, ip, community, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkIPCidrRouteTable(params)
}

// walkIPCidrRoutesV3 walks the legacy ipCidrRouteTable using SNMPv3.
func walkIPCidrRoutesV3(
	ctx context.Context,
	ip string,
	cred *config.SNMPv3Credential,
	cfg *config.SNMPConfig,
) ([]RouteEntry, error) {
	params, err := newV3WalkClient(ctx, ip, cred, cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = params.Conn.Close() }()

	return walkIPCidrRouteTable(params)
}

// walkIPCidrRouteTable walks the legacy ipCidrRouteTable.
func walkIPCidrRouteTable(params *gosnmp.GoSNMP) ([]RouteEntry, error) {
	routes := make(map[string]*RouteEntry)

	// Walk ipCidrRouteDest to discover routes.
	err := params.BulkWalk(OIDIpCidrRouteDest, func(pdu gosnmp.SnmpPDU) error {
		dest, mask, nextHop := parseIPCidrRouteIndex(pdu.Name)
		if dest == "" {
			return nil
		}

		key := fmt.Sprintf("%s/%s-%s", dest, mask, nextHop)
		prefix := netmaskToPrefix(mask)

		routes[key] = &RouteEntry{
			Destination: dest,
			Prefix:      prefix,
			NextHop:     nextHop,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to walk ipCidrRouteDest: %w", err)
	}

	// Walk ipCidrRouteIfIndex.
	walkIPCidrRouteAttribute(params, OIDIpCidrRouteIfIndex, routes, func(r *RouteEntry, value string) {
		r.IfIndex, _ = strconv.Atoi(value)
	})

	// Walk ipCidrRouteType.
	walkIPCidrRouteAttribute(params, OIDIpCidrRouteType, routes, func(r *RouteEntry, value string) {
		r.Type = parseRouteType(value)
	})

	// Walk ipCidrRouteProto.
	walkIPCidrRouteAttribute(params, OIDIpCidrRouteProto, routes, func(r *RouteEntry, value string) {
		r.Protocol = parseRouteProtocol(value)
	})

	// Walk ipCidrRouteMetric1.
	walkIPCidrRouteAttribute(params, OIDIpCidrRouteMetric1, routes, func(r *RouteEntry, value string) {
		r.Metric, _ = strconv.Atoi(value)
	})

	// Convert map to slice.
	result := make([]RouteEntry, 0, len(routes))
	for _, route := range routes {
		result = append(result, *route)
	}

	return result, nil
}

// walkRouteAttribute walks a routing table attribute (inetCidrRouteTable).
func walkRouteAttribute(
	params *gosnmp.GoSNMP,
	oid string,
	routes map[string]*RouteEntry,
	updateFunc func(*RouteEntry, string),
) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		dest, prefix, nextHop := parseInetCidrRouteIndex(pdu.Name)
		if dest == "" {
			return nil
		}

		key := fmt.Sprintf("%s/%d-%s", dest, prefix, nextHop)
		route, exists := routes[key]
		if !exists {
			return nil
		}

		updateFunc(route, formatSNMPValue(pdu))
		return nil
	})
	if err != nil {
		logging.GetLogger().Debug("Failed to walk route attribute", "oid", oid, "error", err)
	}
}

// walkIPCidrRouteAttribute walks a routing table attribute (ipCidrRouteTable).
func walkIPCidrRouteAttribute(
	params *gosnmp.GoSNMP,
	oid string,
	routes map[string]*RouteEntry,
	updateFunc func(*RouteEntry, string),
) {
	err := params.BulkWalk(oid, func(pdu gosnmp.SnmpPDU) error {
		dest, mask, nextHop := parseIPCidrRouteIndex(pdu.Name)
		if dest == "" {
			return nil
		}

		key := fmt.Sprintf("%s/%s-%s", dest, mask, nextHop)
		route, exists := routes[key]
		if !exists {
			return nil
		}

		updateFunc(route, formatSNMPValue(pdu))
		return nil
	})
	if err != nil {
		logging.GetLogger().Debug("Failed to walk IP CIDR route attribute", "oid", oid, "error", err)
	}
}

// parseInetCidrRouteIndex extracts destination, prefix, and next hop from OID.
// OID format: ...destType.destLen.dest.pfxLen.policy.nextHopType.nextHopLen.nextHop.
func parseInetCidrRouteIndex(oid string) (string, int, string) {
	parts := strings.Split(oid, ".")
	if len(parts) < 12 {
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
	if len(parts) < 14 {
		return "", "", ""
	}

	// Destination is last 13 parts starting at -13 to -10
	destStart := len(parts) - 13
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
