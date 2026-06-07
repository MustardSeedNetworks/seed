// Package routing implements the routing SNMP Collector. Walks
// IP-FORWARD-MIB::ipCidrRouteTable (1.3.6.1.2.1.4.24.4.1) and emits
// one Observation per poll listing every IPv4 route entry. Used by
// Stage A4 topology to draw L3 next-hop edges between routers and
// by the listener pipeline to alert on flapping/withdrawn routes.
//
// V1.0 uses ipCidrRouteTable (RFC 2096) which is IPv4-only but
// universally implemented. The newer inetCidrRouteTable (RFC 4292,
// dual-stack) lands when an IPv6 customer asks for it.
package routing

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "routing"

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

// Collect walks ipCidrRouteTable and publishes the assembled routes.
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
	vbs, err := client.Walk(ctx, tablePrefix)
	if err != nil {
		return fmt.Errorf("routing: walk ipCidrRouteTable: %w", err)
	}

	if pubErr := c.publisher.PublishRouting(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Routes:     buildRoutes(vbs),
	}); pubErr != nil {
		return fmt.Errorf("routing: publish: %w", pubErr)
	}
	return nil
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
	less := func(i, j int) bool {
		if rs[i].Destination != rs[j].Destination {
			return rs[i].Destination < rs[j].Destination
		}
		if rs[i].Mask != rs[j].Mask {
			return rs[i].Mask < rs[j].Mask
		}
		return rs[i].NextHop < rs[j].NextHop
	}
	for i := 1; i < len(rs); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			rs[j-1], rs[j] = rs[j], rs[j-1]
		}
	}
}
