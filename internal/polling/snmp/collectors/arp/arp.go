// Package arp implements the arp SNMP Collector. Walks
// IP-MIB::ipNetToMediaTable (1.3.6.1.2.1.4.22.1) and emits one
// Observation per poll listing every IP↔MAC binding the target has
// learned. ARP data feeds Stage A4 topology — it's how L2-only
// neighbor edges (e.g. attached hosts that don't speak LLDP/CDP)
// get an IP identity attached.
package arp

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "arp"

// ipNetToMediaTable column OIDs (RFC 1213 §6.7).
// Indexed by (ipNetToMediaIfIndex, ipNetToMediaNetAddress) where the
// IP is encoded as 4 trailing dotted octets in the OID suffix.
const (
	tablePrefix     = "1.3.6.1.2.1.4.22.1"
	colIfIndex      = "1"
	colPhysAddress  = "2"
	colNetAddress   = "3"
	colMediaType    = "4"
	indexFieldsARP  = 5 // ifIndex + 4 IPv4 octets
	macAddressBytes = 6
)

// MediaType enum values from RFC 1213 §6.7. Exported so the listener
// pipeline can distinguish static (configured) from dynamic (learned)
// ARP entries in alert text and the topology UI can flag static
// shadows over dynamic bindings.
const (
	MediaTypeOther   = 1
	MediaTypeInvalid = 2
	MediaTypeDynamic = 3
	MediaTypeStatic  = 4
)

// Entry is one ipNetToMediaTable row. Static entries (MediaType=4)
// often shadow dynamic entries — Stage A4 uses MediaType to flag
// configured-vs-learned bindings in the topology UI.
type Entry struct {
	IfIndex    uint32
	IPAddress  string
	MACAddress string
	MediaType  int
}

// Observation is the per-poll ARP cache snapshot.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	Entries    []Entry
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishARP(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns an ARP Collector. Pass nil now to use [time.Now] UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks ipNetToMediaTable and publishes the assembled cache.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("arp: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("arp: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("arp: dial: %w", err)
	}

	observedAt := c.now()
	vbs, err := client.Walk(ctx, tablePrefix)
	if err != nil {
		return fmt.Errorf("arp: walk ipNetToMediaTable: %w", err)
	}

	if pubErr := c.publisher.PublishARP(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Entries:    buildEntries(vbs),
	}); pubErr != nil {
		return fmt.Errorf("arp: publish: %w", pubErr)
	}
	return nil
}

// rowKey identifies one ipNetToMediaTable row.
type rowKey struct {
	ifIndex uint32
	ip      string // dotted-quad IPv4
}

// buildEntries folds the table walk into Entries keyed by
// (ifIndex, ip). Order is ascending by ifIndex then IP for
// deterministic downstream comparison.
func buildEntries(vbs []snmp.Varbind) []Entry {
	rows := make(map[rowKey]*Entry)

	for _, vb := range vbs {
		col, key, ok := parseRowOID(vb.OID)
		if !ok {
			continue
		}
		e := rows[key]
		if e == nil {
			e = &Entry{IfIndex: key.ifIndex, IPAddress: key.ip}
			rows[key] = e
		}
		applyColumn(e, col, vb.Value)
	}

	out := make([]Entry, 0, len(rows))
	for _, e := range rows {
		out = append(out, *e)
	}
	sortEntries(out)
	return out
}

// parseRowOID splits an OID under tablePrefix into the column number
// and the (ifIndex, ipv4) row key. The 4 IPv4 octets are joined back
// into dotted-quad form.
func parseRowOID(oid string) (string, rowKey, bool) {
	if !strings.HasPrefix(oid, tablePrefix+".") {
		return "", rowKey{}, false
	}
	rest := strings.TrimPrefix(oid, tablePrefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFieldsARP {
		return "", rowKey{}, false
	}
	ifIdx, err := parseUint32(parts[1])
	if err != nil {
		return "", rowKey{}, false
	}
	const ipv4OctetCount = 4
	octets := [ipv4OctetCount]byte{}
	for i := range ipv4OctetCount {
		v, perr := strconv.ParseUint(parts[2+i], 10, 8)
		if perr != nil {
			return "", rowKey{}, false
		}
		octets[i] = byte(v)
	}
	addr := netip.AddrFrom4(octets)
	return parts[0], rowKey{ifIndex: ifIdx, ip: addr.String()}, true
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

func applyColumn(e *Entry, col string, v any) {
	switch col {
	case colIfIndex:
		// already in row key, but record explicitly for completeness
		// when the column appears in a non-standard ordering
		if u := uint32Value(v); u != 0 {
			e.IfIndex = u
		}
	case colPhysAddress:
		e.MACAddress = macAddressString(v)
	case colNetAddress:
		// NetAddress is encoded in the row key already; the column
		// value is the same IP. Ignore.
	case colMediaType:
		e.MediaType = intValue(v)
	}
}

func macAddressString(v any) string {
	if b, ok := v.([]byte); ok && len(b) == macAddressBytes {
		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			b[0], b[1], b[2], b[3], b[4], b[5])
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	}
	return ""
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

func sortEntries(es []Entry) {
	less := func(i, j int) bool {
		if es[i].IfIndex != es[j].IfIndex {
			return es[i].IfIndex < es[j].IfIndex
		}
		return es[i].IPAddress < es[j].IPAddress
	}
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			es[j-1], es[j] = es[j], es[j-1]
		}
	}
}
