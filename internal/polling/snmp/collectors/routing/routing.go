// Package routing implements the routing SNMP Collector. Walks
// IP-FORWARD-MIB::ipCidrRouteTable (1.3.6.1.2.1.4.24.4.1) and emits
// one Observation per poll listing the IPv4 route entries, at most
// [MaxRows] of them. Used by
// Stage A4 topology to draw L3 next-hop edges between routers and
// by the listener pipeline to alert on flapping/withdrawn routes.
//
// V1.0 uses ipCidrRouteTable (RFC 2096), IPv4-only, and falls back to
// the RFC 1213 ipRouteTable when it is empty: plenty of devices serve
// only the older table (seed#2587). The newer inetCidrRouteTable
// (RFC 4292, dual-stack) lands when an IPv6 customer asks for it.
package routing

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "routing"

// MaxRows bounds one route-table read, the same bound discovery's walk has
// (seed#2833, seed#2857). An edge router carrying a full BGP table holds
// about a million routes, and this read runs on every poll; ten thousand rows
// is far more than any site's own networks, and the rows past it are the
// internet's.
const MaxRows = 10_000

const (
	tablePrefix = "1.3.6.1.2.1.4.24.4.1"

	// Columns we care about. The key columns (Dest, Mask, Tos,
	// NextHop) are redundantly present as both index suffix and
	// column 1-4 — we read them from the index for canonicalization.
	colIfIndex = "5"
	colType    = "6"
	colProto   = "7"
	colAge     = "8"
	colMetric1 = "11"

	// 4 octets dest + 4 octets mask + 1 octet tos + 4 octets nextHop.
	indexFieldsRouting = 13
	ipv4OctetCount     = 4

	// legacyTablePrefix is RFC1213-MIB::ipRouteEntry, indexed by the
	// destination alone; the mask and next hop are columns.
	legacyTablePrefix = "1.3.6.1.2.1.4.21.1"

	legacyColIfIndex = "2"
	legacyColMetric1 = "3"
	legacyColNextHop = "7"
	legacyColType    = "8"
	legacyColProto   = "9"
	legacyColAge     = "10"
	legacyColMask    = "11"

	// legacyTypeInvalid is ipRouteType invalid(2), an entry the agent
	// has deleted but may still list. Its other values, other(1),
	// direct(3) and indirect(4), share TypeOther, TypeLocal and
	// TypeRemote's numbers and meaning.
	legacyTypeInvalid = 2
)

// RouteType values (RFC 2096).
const (
	TypeOther  = 1
	TypeReject = 2
	TypeLocal  = 3
	TypeRemote = 4
)

// RouteProto values (RFC 2096). Stage A4 alerting filters on these:
// connected/local/bgp/ospf get different alert severities than
// learned via RIP.
const (
	ProtoOther     = 1
	ProtoLocal     = 2
	ProtoNetmgmt   = 3
	ProtoICMP      = 4
	ProtoEGP       = 5
	ProtoGGP       = 6
	ProtoHello     = 7
	ProtoRIP       = 8
	ProtoISIS      = 9
	ProtoESIS      = 10
	ProtoCiscoIGRP = 11
	ProtoBBNSpfIGP = 12
	ProtoOSPF      = 13
	ProtoBGP       = 14
)

// Route is one ipCidrRouteTable row.
type Route struct {
	Destination string // dotted-quad IPv4
	Mask        string // dotted-quad IPv4 mask
	Tos         uint32 // type-of-service, usually 0
	NextHop     string // dotted-quad IPv4
	IfIndex     uint32
	Type        int
	Proto       int
	AgeSeconds  uint32
	Metric1     int
}

// Observation is the per-poll route table snapshot.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	Routes     []Route
	// Truncated is true when the device held more than MaxRows routes
	// and Routes is only the first of them.
	Truncated bool
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishRouting(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns a routing Collector. Pass nil now to use [time.Now] UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks ipCidrRouteTable, or ipRouteTable when that is empty, and
// publishes the assembled routes.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("routing: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("routing: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("routing: dial: %w", err)
	}

	observedAt := c.now()
	vbs, truncated, err := walkColumns(ctx, client, tablePrefix,
		colIfIndex, colType, colProto, colAge, colMetric1)
	if err != nil {
		return fmt.Errorf("routing: walk ipCidrRouteTable: %w", err)
	}

	routes := buildRoutes(vbs)
	if len(routes) == 0 {
		legacy, legacyTruncated, legacyErr := walkColumns(ctx, client, legacyTablePrefix,
			legacyColIfIndex, legacyColMetric1, legacyColNextHop, legacyColType,
			legacyColProto, legacyColAge, legacyColMask)
		if legacyErr != nil {
			return fmt.Errorf("routing: walk ipRouteTable: %w", legacyErr)
		}
		routes, truncated = buildLegacyRoutes(legacy), legacyTruncated
	}

	if pubErr := c.publisher.PublishRouting(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Routes:     routes,
		Truncated:  truncated,
	}); pubErr != nil {
		return fmt.Errorf("routing: publish: %w", pubErr)
	}
	return nil
}

// walkColumns reads the given columns of table, at most MaxRows rows of
// each, and reports whether any held more. A table walk is column-major, so
// a bound on the whole table would end inside its first columns and never
// reach the ones read here; each column is walked, and bounded, on its own.
func walkColumns(
	ctx context.Context,
	client snmp.Client,
	table string,
	columns ...string,
) ([]snmp.Varbind, bool, error) {
	var vbs []snmp.Varbind
	truncated := false
	for _, col := range columns {
		got, err := client.WalkLimit(ctx, table+"."+col, MaxRows+1)
		if err != nil {
			return nil, false, err
		}
		if len(got) > MaxRows {
			got, truncated = got[:MaxRows], true
		}
		vbs = append(vbs, got...)
	}
	return vbs, truncated, nil
}

// routeKey is one ipCidrRouteTable row keyed by the 4-tuple
// (dest, mask, tos, nextHop) — those four make a route unique.
type routeKey struct {
	dest    string
	mask    string
	tos     uint32
	nextHop string
}

func buildRoutes(vbs []snmp.Varbind) []Route {
	rows := make(map[routeKey]*Route)
	for _, vb := range vbs {
		col, key, ok := parseRouteOID(vb.OID)
		if !ok {
			continue
		}
		r := rows[key]
		if r == nil {
			r = &Route{
				Destination: key.dest,
				Mask:        key.mask,
				Tos:         key.tos,
				NextHop:     key.nextHop,
			}
			rows[key] = r
		}
		applyColumn(r, col, vb.Value)
	}

	out := make([]Route, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sortRoutes(out)
	return out
}

// parseRouteOID expects tablePrefix.col.<4 dest>.<4 mask>.<tos>.<4 nextHop>.
func parseRouteOID(oid string) (string, routeKey, bool) {
	if !strings.HasPrefix(oid, tablePrefix+".") {
		return "", routeKey{}, false
	}
	rest := strings.TrimPrefix(oid, tablePrefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFieldsRouting {
		return "", routeKey{}, false
	}

	dest, ok := parseIPv4(parts[1:5])
	if !ok {
		return "", routeKey{}, false
	}
	mask, ok := parseIPv4(parts[5:9])
	if !ok {
		return "", routeKey{}, false
	}
	tos, err := parseUint32(parts[9])
	if err != nil {
		return "", routeKey{}, false
	}
	nextHop, ok := parseIPv4(parts[10:14])
	if !ok {
		return "", routeKey{}, false
	}
	return parts[0], routeKey{
		dest: dest, mask: mask, tos: tos, nextHop: nextHop,
	}, true
}

// buildLegacyRoutes assembles ipRouteTable rows. A row is kept only
// once its mask and next hop have arrived as IPv4 values, and never
// when the agent marks it invalid: a row without its mask cannot be
// told apart from a default route.
func buildLegacyRoutes(vbs []snmp.Varbind) []Route {
	rows := make(map[string]*Route)
	for _, vb := range vbs {
		col, dest, ok := parseLegacyRouteOID(vb.OID)
		if !ok {
			continue
		}
		r := rows[dest]
		if r == nil {
			r = &Route{Destination: dest}
			rows[dest] = r
		}
		applyLegacyColumn(r, col, vb.Value)
	}

	out := make([]Route, 0, len(rows))
	for _, r := range rows {
		if r.Type == legacyTypeInvalid || !contiguousMask(r.Mask) || r.NextHop == "" {
			continue
		}
		out = append(out, *r)
	}
	sortRoutes(out)
	return out
}

// parseLegacyRouteOID expects legacyTablePrefix.col.<4 dest>.
func parseLegacyRouteOID(oid string) (string, string, bool) {
	rest, ok := strings.CutPrefix(oid, legacyTablePrefix+".")
	if !ok {
		return "", "", false
	}
	parts := strings.Split(rest, ".")
	if len(parts) != 1+ipv4OctetCount {
		return "", "", false
	}
	dest, ok := parseIPv4(parts[1:])
	if !ok {
		return "", "", false
	}
	return parts[0], dest, true
}

func applyLegacyColumn(r *Route, col string, v any) {
	switch col {
	case legacyColIfIndex:
		r.IfIndex = uint32Value(v)
	case legacyColMetric1:
		r.Metric1 = intValue(v)
	case legacyColNextHop:
		r.NextHop = ipv4Value(v)
	case legacyColType:
		r.Type = intValue(v)
	case legacyColProto:
		r.Proto = intValue(v)
	case legacyColAge:
		r.AgeSeconds = uint32Value(v)
	case legacyColMask:
		r.Mask = ipv4Value(v)
	}
}

// ipv4Value renders an IpAddress varbind, which gosnmp decodes to a
// dotted-quad string, as a canonical dotted quad; anything else is "".
func ipv4Value(v any) string {
	s, ok := v.(string)
	if !ok {
		return ""
	}
	addr, err := netip.ParseAddr(s)
	if err != nil || !addr.Is4() {
		return ""
	}
	return addr.String()
}

// contiguousMask reports whether mask is a dotted-quad netmask with its
// one bits leading.
func contiguousMask(mask string) bool {
	addr, err := netip.ParseAddr(mask)
	if err != nil || !addr.Is4() {
		return false
	}
	_, bits := net.IPMask(addr.AsSlice()).Size()
	return bits != 0
}

// parseIPv4 reads four decimal octets from an OID suffix slice and
// formats them as a canonical dotted quad via [netip.AddrFrom4].
func parseIPv4(octetParts []string) (string, bool) {
	if len(octetParts) != ipv4OctetCount {
		return "", false
	}
	var b [ipv4OctetCount]byte
	for i, s := range octetParts {
		v, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			return "", false
		}
		b[i] = byte(v)
	}
	return netip.AddrFrom4(b).String(), true
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

func applyColumn(r *Route, col string, v any) {
	switch col {
	case colIfIndex:
		r.IfIndex = uint32Value(v)
	case colType:
		r.Type = intValue(v)
	case colProto:
		r.Proto = intValue(v)
	case colAge:
		r.AgeSeconds = uint32Value(v)
	case colMetric1:
		r.Metric1 = intValue(v)
	}
}

func intValue(v any) int {
	const maxIntAsUint64 = uint64(^uint(0) >> 1)
	switch t := v.(type) {
	case nil:
		return 0
	case int:
		return t
	case int32:
		return int(t)
	case int64:
		if t < 0 || uint64(t) > maxIntAsUint64 {
			return 0
		}
		return int(t)
	case uint:
		if uint64(t) > maxIntAsUint64 {
			return 0
		}
		return int(t)
	case uint32:
		return int(t)
	case uint64:
		if t > maxIntAsUint64 {
			return 0
		}
		return int(t)
	default:
		return 0
	}
}

func uint32Value(v any) uint32 {
	const maxUint32 uint64 = 1<<32 - 1
	switch t := v.(type) {
	case nil:
		return 0
	case uint32:
		return t
	case int:
		if t < 0 {
			return 0
		}
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case int32:
		if t < 0 {
			return 0
		}
		return uint32(t)
	case int64:
		if t < 0 {
			return 0
		}
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case uint64:
		if t > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	}
	return 0
}

func sortRoutes(rs []Route) {
	slices.SortFunc(rs, func(a, b Route) int {
		return cmp.Or(
			strings.Compare(a.Destination, b.Destination),
			strings.Compare(a.Mask, b.Mask),
			strings.Compare(a.NextHop, b.NextHop),
		)
	})
}
