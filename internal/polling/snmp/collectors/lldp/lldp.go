// Package lldp implements the lldp SNMP Collector: walks
// LLDP-MIB::lldpRemTable (1.0.8802.1.1.2.1.4.1.1) and emits one
// Observation per poll listing every LLDP neighbor the target's
// agent has learned. The neighbor relationships form the edges of
// Stage A4's topology graph.
package lldp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "lldp"

// lldpRemTable column OIDs. See RFC 8628 / IEEE 802.1AB.
// Indexed by (lldpRemTimeMark, lldpRemLocalPortNum, lldpRemIndex).
const (
	remTablePrefix      = "1.0.8802.1.1.2.1.4.1.1"
	colChassisIDSubtype = "4"
	colChassisID        = "5"
	colPortIDSubtype    = "6"
	colPortID           = "7"
	colPortDesc         = "8"
	colSysName          = "9"
	colSysDesc          = "10"
	colSysCapSupported  = "11"
	colSysCapEnabled    = "12"
	indexFieldsRemTable = 3 // (timeMark, localPortNum, remIndex)
)

// Neighbor is one row of lldpRemTable identifying a remote device
// + remote port reachable through the local port LocalPortNum.
type Neighbor struct {
	LocalPortNum     uint32
	ChassisIDSubtype int
	ChassisID        string
	PortIDSubtype    int
	PortID           string
	PortDescription  string
	SysName          string
	SysDescription   string
	SysCapSupported  uint32
	SysCapEnabled    uint32
	LldpRemIndex     uint32
	LldpRemTimeMark  uint32
}

// Observation is the per-poll set of LLDP neighbors.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	Neighbors  []Neighbor
}

// Publisher is the consumer-defined seam. Stage A3.5 wires it to
// the topology edge reconciler.
type Publisher interface {
	PublishLLDP(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns a Collector. Pass nil now to use [time.Now] in UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks lldpRemTable and publishes the assembled neighbor list.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("lldp: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("lldp: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("lldp: dial: %w", err)
	}

	observedAt := c.now()
	vbs, err := client.Walk(ctx, remTablePrefix)
	if err != nil {
		return fmt.Errorf("lldp: walk lldpRemTable: %w", err)
	}

	neighbors := buildNeighbors(vbs)

	if pubErr := c.publisher.PublishLLDP(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Neighbors:  neighbors,
	}); pubErr != nil {
		return fmt.Errorf("lldp: publish: %w", pubErr)
	}
	return nil
}

// rowKey identifies one lldpRemTable row by its index triple.
type rowKey struct {
	timeMark     uint32
	localPortNum uint32
	remIndex     uint32
}

// buildNeighbors folds the lldpRemTable walk into one Neighbor per
// unique (timeMark, localPortNum, remIndex). Order is ascending by
// LocalPortNum then RemIndex for deterministic downstream comparison.
func buildNeighbors(vbs []snmp.Varbind) []Neighbor {
	rows := make(map[rowKey]*Neighbor)
	for _, vb := range vbs {
		col, key, ok := parseRemTableOID(vb.OID)
		if !ok {
			continue
		}
		n := rows[key]
		if n == nil {
			n = &Neighbor{
				LocalPortNum:    key.localPortNum,
				LldpRemTimeMark: key.timeMark,
				LldpRemIndex:    key.remIndex,
			}
			rows[key] = n
		}
		applyColumn(n, col, vb.Value)
	}

	out := make([]Neighbor, 0, len(rows))
	for _, n := range rows {
		out = append(out, *n)
	}
	sortNeighbors(out)
	return out
}

// parseRemTableOID splits an OID under remTablePrefix into the
// column number and the (timeMark, localPortNum, remIndex) index
// triple.
func parseRemTableOID(oid string) (string, rowKey, bool) {
	if !strings.HasPrefix(oid, remTablePrefix+".") {
		return "", rowKey{}, false
	}
	rest := strings.TrimPrefix(oid, remTablePrefix+".")
	parts := strings.Split(rest, ".")
	// expect: <column>.<timeMark>.<localPortNum>.<remIndex>
	if len(parts) != 1+indexFieldsRemTable {
		return "", rowKey{}, false
	}
	tm, err1 := parseUint32(parts[1])
	lp, err2 := parseUint32(parts[2])
	ri, err3 := parseUint32(parts[3])
	if err1 != nil || err2 != nil || err3 != nil {
		return "", rowKey{}, false
	}
	return parts[0], rowKey{timeMark: tm, localPortNum: lp, remIndex: ri}, true
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// applyColumn writes one lldpRemTable column onto n.
func applyColumn(n *Neighbor, col string, v any) {
	switch col {
	case colChassisIDSubtype:
		n.ChassisIDSubtype = intValue(v)
	case colChassisID:
		n.ChassisID = chassisIDString(v)
	case colPortIDSubtype:
		n.PortIDSubtype = intValue(v)
	case colPortID:
		n.PortID = chassisIDString(v)
	case colPortDesc:
		n.PortDescription = stringValue(v)
	case colSysName:
		n.SysName = stringValue(v)
	case colSysDesc:
		n.SysDescription = stringValue(v)
	case colSysCapSupported:
		n.SysCapSupported = capabilityValue(v)
	case colSysCapEnabled:
		n.SysCapEnabled = capabilityValue(v)
	}
}

// macAddressBytes is the byte width of an Ethernet MAC chassis ID —
// the most common LLDP chassis subtype.
const macAddressBytes = 6

// chassisIDString prefers the colon-hex form for MAC chassis IDs (the
// common case) and falls back to the raw string otherwise. Six-byte
// inputs match the MAC-address chassis subtype most agents emit.
func chassisIDString(v any) string {
	if b, ok := v.([]byte); ok {
		switch len(b) {
		case macAddressBytes:
			return formatMAC(b)
		case 0:
			return ""
		default:
			return string(b)
		}
	}
	return stringValue(v)
}

func formatMAC(b []byte) string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		b[0], b[1], b[2], b[3], b[4], b[5])
}

func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return ""
	}
}

// intValue extracts a small signed integer scalar. Out-of-int range
// or negative falls back to 0 — downstream code treats 0 as
// "unspecified subtype".
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

// capabilityValue decodes lldpSysCapMap (an LLDP BITS field). The
// MIB exposes it as a short OCTET STRING (2 bytes) with each bit a
// capability flag — we pack it back into a uint32 for compact
// storage, MSB first.
func capabilityValue(v any) uint32 {
	const bitsPerByte = 8
	const bitsCapWidth = 32
	switch t := v.(type) {
	case nil:
		return 0
	case []byte:
		var out uint32
		for i, b := range t {
			shift := bitsCapWidth - bitsPerByte*(i+1)
			if shift < 0 {
				break
			}
			out |= uint32(b) << uint(shift)
		}
		return out
	}
	return 0
}

// sortNeighbors orders the slice ascending by LocalPortNum, then by
// LldpRemIndex within each port. Insertion sort is fine — a target
// rarely has more than ~10 neighbors.
func sortNeighbors(ns []Neighbor) {
	less := func(i, j int) bool {
		if ns[i].LocalPortNum != ns[j].LocalPortNum {
			return ns[i].LocalPortNum < ns[j].LocalPortNum
		}
		return ns[i].LldpRemIndex < ns[j].LldpRemIndex
	}
	for i := 1; i < len(ns); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			ns[j-1], ns[j] = ns[j], ns[j-1]
		}
	}
}
