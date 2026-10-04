// Package iftable implements the if_table SNMP Collector: walks
// IF-MIB ifTable (RFC 2233 §6) plus ifXTable extensions and emits
// one Observation containing every interface row indexed by ifIndex.
// Used by Stage A4 topology to attach physical/logical interfaces
// to their parent Node.
package iftable

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Name is the collector key written into polling_targets.collector_chain.
const Name = "if_table"

// Column OIDs under ifTable (1.3.6.1.2.1.2.2.1.*) — the subset that
// V1.0 topology + counters need. The full ifTable carries 22 columns;
// the remainder land when their consumers materialize.
const (
	ifTablePrefix    = "1.3.6.1.2.1.2.2.1"
	colIfDescr       = "2"
	colIfType        = "3"
	colIfSpeed       = "5"
	colIfPhysAddress = "6"
	colIfAdminStatus = "7"
	colIfOperStatus  = "8"
	colIfInOctets    = "10"
	colIfInDiscards  = "13"
	colIfInErrors    = "14"
	colIfOutOctets   = "16"
	colIfOutDiscards = "19"
	colIfOutErrors   = "20"
)

// Column OIDs under ifXTable (1.3.6.1.2.1.31.1.1.1.*).
const (
	ifXTablePrefix                = "1.3.6.1.2.1.31.1.1.1"
	colIfName                     = "1"
	colIfHCInOctets               = "6"
	colIfHCOutOctets              = "10"
	colIfHighSpeed                = "15"
	colIfAlias                    = "18"
	colIfCounterDiscontinuityTime = "19"
)

// oidSysUpTime is read with the tables so counter deltas can tell an agent
// restart from a counter wrap (RFC 2863 §3.1.6).
const oidSysUpTime = "1.3.6.1.2.1.1.3.0"

// AdminStatus / OperStatus values from RFC 2233 — exported because
// downstream listeners decode them by name in alert messages.
const (
	StatusUp             = 1
	StatusDown           = 2
	StatusTesting        = 3
	StatusUnknown        = 4 // ifOperStatus only
	StatusDormant        = 5 // ifOperStatus only
	StatusNotPresent     = 6 // ifOperStatus only
	StatusLowerLayerDown = 7 // ifOperStatus only
)

// Row is one ifTable/ifXTable row keyed by IfIndex. SpeedBps prefers
// ifHighSpeed*1e6 when available (multi-gigabit links); falls back
// to ifSpeed (32-bit bps) for older agents.
type Row struct {
	IfIndex    uint32
	IfDescr    string
	IfName     string
	IfAlias    string
	IfType     uint32
	IfAdmin    int
	IfOper     int
	IfPhysAddr string
	SpeedBps   uint64
	Counters   Counters

	rawIfSpeed    uint32
	rawHighMbps   uint32
	rawInOctets   uint64
	rawOutOctets  uint64
	rawHCIn       uint64
	rawHCOut      uint64
	hasHCInOctet  bool
	hasHCOutOctet bool
}

// Counters are the row's raw cumulative counters. Octets come from the
// 64-bit ifHC columns when the agent serves both, otherwise from the 32-bit
// ifTable columns, and HCOctets records which: the two wrap at different
// widths. Discontinuity is ifCounterDiscontinuityTime, which an agent bumps
// when the counters restart without a reboot (a "clear counters").
type Counters struct {
	InOctets      uint64
	OutOctets     uint64
	HCOctets      bool
	InErrors      uint64
	OutErrors     uint64
	InDiscards    uint64
	OutDiscards   uint64
	Discontinuity uint32
}

// Observation is the per-target ifTable snapshot. Rows is sorted by
// ascending IfIndex so downstream comparisons (alerting on
// new/missing interfaces) are deterministic. SysUpTime is nil when the
// agent did not serve sysUpTime.0, and the counters then cannot be rated.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	SysUpTime  *uint32
	Rows       []Row
}

// Publisher is the consumer-defined seam for iftable observations.
// Stage A3.5 wires it to topology reconciliation.
type Publisher interface {
	PublishIfTable(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector for the if_table chain step.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns a Collector bound to factory + publisher. Pass nil now
// to use [time.Now].UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks ifTable + ifXTable subtrees, merges by ifIndex, and
// publishes the resulting Observation.
func (c *Collector) Collect(
	ctx context.Context,
	target snmp.Target,
	creds snmp.ResolvedCredentials,
) error {
	if c.newClient == nil {
		return errors.New("iftable: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("iftable: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("iftable: dial: %w", err)
	}

	observedAt := c.now()

	upTime, err := client.Get(ctx, []string{oidSysUpTime})
	if err != nil {
		return fmt.Errorf("iftable: get sysUpTime: %w", err)
	}

	ifVarbinds, err := client.Walk(ctx, ifTablePrefix)
	if err != nil {
		return fmt.Errorf("iftable: walk ifTable: %w", err)
	}
	ifXVarbinds, err := client.Walk(ctx, ifXTablePrefix)
	if err != nil {
		return fmt.Errorf("iftable: walk ifXTable: %w", err)
	}

	rows := mergeRows(ifVarbinds, ifXVarbinds)

	if pubErr := c.publisher.PublishIfTable(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		SysUpTime:  sysUpTime(upTime),
		Rows:       rows,
	}); pubErr != nil {
		return fmt.Errorf("iftable: publish: %w", pubErr)
	}
	return nil
}

// mergeRows folds ifTable + ifXTable varbinds into Rows keyed by
// ifIndex. Order is ascending ifIndex for deterministic downstream
// comparisons.
func mergeRows(ifVarbinds, ifXVarbinds []snmp.Varbind) []Row {
	byIndex := make(map[uint32]*Row)

	for _, vb := range ifVarbinds {
		col, idx, ok := parseColumnIndex(vb.OID, ifTablePrefix)
		if !ok {
			continue
		}
		row := getOrCreate(byIndex, idx)
		applyIfTableColumn(row, col, vb.Value)
	}
	for _, vb := range ifXVarbinds {
		col, idx, ok := parseColumnIndex(vb.OID, ifXTablePrefix)
		if !ok {
			continue
		}
		row := getOrCreate(byIndex, idx)
		applyIfXTableColumn(row, col, vb.Value)
	}

	out := make([]Row, 0, len(byIndex))
	for _, row := range byIndex {
		row.SpeedBps = pickSpeedBps(row.rawHighMbps, row.rawIfSpeed)
		row.Counters.HCOctets = row.hasHCInOctet && row.hasHCOutOctet
		if row.Counters.HCOctets {
			row.Counters.InOctets, row.Counters.OutOctets = row.rawHCIn, row.rawHCOut
		} else {
			row.Counters.InOctets, row.Counters.OutOctets = row.rawInOctets, row.rawOutOctets
		}
		out = append(out, *row)
	}
	sortByIfIndex(out)
	return out
}

// columnIndexParts is the number of dot-separated fields a column-
// indexed OID has under its table prefix: <column>.<ifIndex>.
const columnIndexParts = 2

// parseColumnIndex splits an OID under prefix into its column and
// ifIndex parts. Returns ok=false when the OID is malformed or not
// rooted at prefix.
func parseColumnIndex(oid, prefix string) (string, uint32, bool) {
	if !strings.HasPrefix(oid, prefix+".") {
		return "", 0, false
	}
	rest := strings.TrimPrefix(oid, prefix+".")
	parts := strings.SplitN(rest, ".", columnIndexParts)
	if len(parts) != columnIndexParts {
		return "", 0, false
	}
	idx64, err := strconv.ParseUint(parts[1], 10, 32)
	if err != nil {
		return "", 0, false
	}
	return parts[0], uint32(idx64), true
}

func getOrCreate(m map[uint32]*Row, idx uint32) *Row {
	if r, ok := m[idx]; ok {
		return r
	}
	r := &Row{IfIndex: idx}
	m[idx] = r
	return r
}

// applyIfTableColumn sets one ifTable column on row by column number.
func applyIfTableColumn(row *Row, col string, v any) {
	switch col {
	case colIfDescr:
		row.IfDescr = stringValue(v)
	case colIfType:
		row.IfType = uint32Value(v)
	case colIfSpeed:
		row.rawIfSpeed = uint32Value(v)
	case colIfPhysAddress:
		row.IfPhysAddr = macAddressString(v)
	case colIfAdminStatus:
		row.IfAdmin = intValue(v)
	case colIfOperStatus:
		row.IfOper = intValue(v)
	case colIfInOctets:
		row.rawInOctets = uint64Value(v)
	case colIfInDiscards:
		row.Counters.InDiscards = uint64Value(v)
	case colIfInErrors:
		row.Counters.InErrors = uint64Value(v)
	case colIfOutOctets:
		row.rawOutOctets = uint64Value(v)
	case colIfOutDiscards:
		row.Counters.OutDiscards = uint64Value(v)
	case colIfOutErrors:
		row.Counters.OutErrors = uint64Value(v)
	}
}

// applyIfXTableColumn sets one ifXTable column on row by column number.
func applyIfXTableColumn(row *Row, col string, v any) {
	switch col {
	case colIfName:
		row.IfName = stringValue(v)
	case colIfHighSpeed:
		row.rawHighMbps = uint32Value(v)
	case colIfAlias:
		row.IfAlias = stringValue(v)
	case colIfHCInOctets:
		row.rawHCIn, row.hasHCInOctet = uint64Value(v), v != nil
	case colIfHCOutOctets:
		row.rawHCOut, row.hasHCOutOctet = uint64Value(v), v != nil
	case colIfCounterDiscontinuityTime:
		row.Counters.Discontinuity = uint32Value(v)
	}
}

// mbpsToBps converts ifHighSpeed (megabits per second) to bps. Kept
// as a named constant to avoid an opaque "1_000_000" at the call
// site.
const mbpsToBps uint64 = 1_000_000

// pickSpeedBps prefers ifHighSpeed (Mbps) when present — it covers
// multi-gigabit links that overflow ifSpeed's 32-bit bps. Falls back
// to ifSpeed for older agents.
func pickSpeedBps(highMbps, ifSpeedBps uint32) uint64 {
	if highMbps > 0 {
		return uint64(highMbps) * mbpsToBps
	}
	return uint64(ifSpeedBps)
}

// stringValue extracts a string from a varbind value. SNMP OCTET
// STRINGs arrive as []byte; gosnmp DisplayStrings arrive as string.
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

// macAddressString formats a 6-byte MAC as "aa:bb:cc:dd:ee:ff".
// Non-6-byte inputs fall back to the raw stringValue.
func macAddressString(v any) string {
	if b, ok := v.([]byte); ok && len(b) == 6 {
		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
			b[0], b[1], b[2], b[3], b[4], b[5])
	}
	return stringValue(v)
}

// intValue extracts a small signed int (admin/oper status, etc.).
// Status values are 1..7, so wider scalars are clamped to the int
// range; out-of-range (negative or huge) falls back to 0, which
// downstream code treats as "unknown".
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

// uint32Value mirrors sysinfo's tolerant decoder. Negative values
// clamp to 0; oversized clamp to [math.MaxUint32].
func uint32Value(v any) uint32 {
	const maxUint32 uint64 = 1<<32 - 1
	switch t := v.(type) {
	case nil:
		return 0
	case uint32:
		return t
	case uint:
		if uint64(t) > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
	case uint64:
		if t > maxUint32 {
			return uint32(maxUint32)
		}
		return uint32(t)
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
	default:
		return 0
	}
}

// sysUpTime returns the TimeTicks value of a sysUpTime.0 Get, or nil when
// the agent did not serve it.
func sysUpTime(vbs []snmp.Varbind) *uint32 {
	if len(vbs) != 1 || vbs[0].Value == nil {
		return nil
	}
	ticks := uint32Value(vbs[0].Value)
	return &ticks
}

// uint64Value decodes a Counter32 or Counter64. Negative values clamp to 0.
func uint64Value(v any) uint64 {
	switch t := v.(type) {
	case uint64:
		return t
	case uint:
		return uint64(t)
	case uint32:
		return uint64(t)
	case int:
		return uint64(max(t, 0))
	case int32:
		return uint64(max(t, 0))
	case int64:
		return uint64(max(t, 0))
	default:
		return 0
	}
}

// sortByIfIndex sorts rows by ascending IfIndex in-place.
// Uses an insertion sort because real targets have O(10-1000)
// interfaces and the simpler implementation reads cleaner than
// pulling in [sort.Slice] for this hot path.
func sortByIfIndex(rows []Row) {
	for i := 1; i < len(rows); i++ {
		for j := i; j > 0 && rows[j-1].IfIndex > rows[j].IfIndex; j-- {
			rows[j-1], rows[j] = rows[j], rows[j-1]
		}
	}
}
