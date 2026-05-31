// Package fdb implements the fdb SNMP Collector for switch-side L2
// forwarding-database visibility. Walks two tables:
//
//   - dot1dBasePortTable (1.3.6.1.2.1.17.1.4.1) — bridge port to
//     ifIndex translation, needed to resolve fdb rows back to the
//     parent interface row in ifTable.
//   - dot1qTpFdbTable (1.3.6.1.2.1.17.7.1.2.2.1) — VLAN-aware MAC
//     learning table; rows are (vlan, mac) → bridge_port.
//
// Together they let Stage A4 topology stitch L2 edges to host MACs
// behind access ports (printers, IP phones, security cameras, etc.).
// FDB tables are large (10k+ rows on enterprise switches), so the
// collector streams via Walk rather than per-row Get.
package fdb

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/krisarmstrong/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "fdb"

const (
	// dot1dBasePortTable: bridge port number → ifIndex.
	basePortTablePrefix = "1.3.6.1.2.1.17.1.4.1"
	colBasePortIfIndex  = "2"

	// dot1qTpFdbTable: per-VLAN MAC learning table.
	tpFdbTablePrefix = "1.3.6.1.2.1.17.7.1.2.2.1"
	colTpFdbPort     = "2"
	colTpFdbStatus   = "3"

	macAddressBytes = 6
	indexFieldsFdb  = 7 // vlanID + 6 MAC octets
	indexFieldsBase = 1 // bridge port number
)

// FdbStatus values from RFC 4188.
const (
	StatusOther   = 1
	StatusInvalid = 2
	StatusLearned = 3
	StatusSelf    = 4
	StatusMgmt    = 5
)

// Entry is one dot1qTpFdbTable row joined with its bridge-port→
// ifIndex translation. IfIndex is 0 when the bridge port has no
// corresponding ifTable row (rare but valid on some agents).
type Entry struct {
	VLANID     uint32
	MACAddress string
	BridgePort uint32
	IfIndex    uint32
	Status     int
}

// Observation is the per-poll forwarding-database snapshot.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	Entries    []Entry
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishFDB(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns an FDB Collector. Pass nil now to use [time.Now] UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks dot1dBasePortTable + dot1qTpFdbTable, joins them by
// bridge-port number, and publishes the resulting Entry list.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("fdb: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("fdb: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("fdb: dial: %w", err)
	}

	observedAt := c.now()

	basePortVbs, err := client.Walk(ctx, basePortTablePrefix)
	if err != nil {
		return fmt.Errorf("fdb: walk dot1dBasePortTable: %w", err)
	}
	fdbVbs, err := client.Walk(ctx, tpFdbTablePrefix)
	if err != nil {
		return fmt.Errorf("fdb: walk dot1qTpFdbTable: %w", err)
	}

	portToIfIndex := buildBasePortMap(basePortVbs)
	entries := buildEntries(fdbVbs, portToIfIndex)

	if pubErr := c.publisher.PublishFDB(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Entries:    entries,
	}); pubErr != nil {
		return fmt.Errorf("fdb: publish: %w", pubErr)
	}
	return nil
}

// buildBasePortMap turns a dot1dBasePortTable walk into a
// bridgePort → ifIndex map.
func buildBasePortMap(vbs []snmp.Varbind) map[uint32]uint32 {
	m := make(map[uint32]uint32)
	for _, vb := range vbs {
		col, port, ok := parseBasePortOID(vb.OID)
		if !ok || col != colBasePortIfIndex {
			continue
		}
		m[port] = uint32Value(vb.Value)
	}
	return m
}

// parseBasePortOID expects basePortTablePrefix.col.bridgePort.
func parseBasePortOID(oid string) (string, uint32, bool) {
	if !strings.HasPrefix(oid, basePortTablePrefix+".") {
		return "", 0, false
	}
	rest := strings.TrimPrefix(oid, basePortTablePrefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFieldsBase {
		return "", 0, false
	}
	port, err := parseUint32(parts[1])
	if err != nil {
		return "", 0, false
	}
	return parts[0], port, true
}

// fdbRowKey identifies one dot1qTpFdbTable row by (vlanID, mac).
type fdbRowKey struct {
	vlanID uint32
	mac    string // canonical aa:bb:cc:dd:ee:ff
}

// buildEntries folds the fdb walk into one Entry per (vlan, mac),
// joined with the basePort→ifIndex map.
func buildEntries(vbs []snmp.Varbind, portToIfIndex map[uint32]uint32) []Entry {
	rows := make(map[fdbRowKey]*Entry)
	for _, vb := range vbs {
		col, key, ok := parseFdbOID(vb.OID)
		if !ok {
			continue
		}
		e := rows[key]
		if e == nil {
			e = &Entry{VLANID: key.vlanID, MACAddress: key.mac}
			rows[key] = e
		}
		switch col {
		case colTpFdbPort:
			port := uint32Value(vb.Value)
			e.BridgePort = port
			if ifIdx, found := portToIfIndex[port]; found {
				e.IfIndex = ifIdx
			}
		case colTpFdbStatus:
			e.Status = intValue(vb.Value)
		}
	}

	out := make([]Entry, 0, len(rows))
	for _, e := range rows {
		out = append(out, *e)
	}
	sortEntries(out)
	return out
}

// parseFdbOID expects tpFdbTablePrefix.col.vlanID.<6 MAC octets>.
func parseFdbOID(oid string) (string, fdbRowKey, bool) {
	if !strings.HasPrefix(oid, tpFdbTablePrefix+".") {
		return "", fdbRowKey{}, false
	}
	rest := strings.TrimPrefix(oid, tpFdbTablePrefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFieldsFdb {
		return "", fdbRowKey{}, false
	}
	vlan, err := parseUint32(parts[1])
	if err != nil {
		return "", fdbRowKey{}, false
	}
	macBytes := [macAddressBytes]byte{}
	for i := range macAddressBytes {
		v, perr := strconv.ParseUint(parts[2+i], 10, 8)
		if perr != nil {
			return "", fdbRowKey{}, false
		}
		macBytes[i] = byte(v)
	}
	mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		macBytes[0], macBytes[1], macBytes[2],
		macBytes[3], macBytes[4], macBytes[5])
	return parts[0], fdbRowKey{vlanID: vlan, mac: mac}, true
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
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
		if es[i].VLANID != es[j].VLANID {
			return es[i].VLANID < es[j].VLANID
		}
		return es[i].MACAddress < es[j].MACAddress
	}
	for i := 1; i < len(es); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			es[j-1], es[j] = es[j], es[j-1]
		}
	}
}
