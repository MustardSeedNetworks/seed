// Package hostresources implements the host_resources SNMP Collector.
// Walks the three HOST-RESOURCES-MIB (RFC 2790) subtrees the listener
// pipeline + topology Node UI need:
//
//   - hrSystemUptime scalar — sanity-check against sysUpTime (sysUpTime
//     resets on agent restart; hrSystemUptime tracks the host).
//   - hrStorageTable — per-filesystem size/used in canonical bytes.
//   - hrProcessorTable — per-processor load percentage.
//
// Storage and processor data drive the "running hot" alert: stage
// a3.5 listener tracks the trend, fires when an FS crosses 85% or a
// CPU averages >80% for >5min.
package hostresources

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
const Name = "host_resources"

const (
	oidHrSystemUptime = "1.3.6.1.2.1.25.1.1.0"

	storageTablePrefix   = "1.3.6.1.2.1.25.2.3.1"
	colStorageType       = "2"
	colStorageDescr      = "3"
	colStorageAllocUnits = "4"
	colStorageSize       = "5"
	colStorageUsed       = "6"
	indexFieldsStorage   = 1

	processorTablePrefix = "1.3.6.1.2.1.25.3.3.1"
	colProcessorLoad     = "2"
	indexFieldsProcessor = 1
)

// Well-known hrStorageType OID values (RFC 2790 §3.1). Compared as
// strings (gosnmp emits ObjectIdentifier as either string or []byte).
const (
	StorageTypeOther         = "1.3.6.1.2.1.25.2.1.1"
	StorageTypeRAM           = "1.3.6.1.2.1.25.2.1.2"
	StorageTypeVirtualMemory = "1.3.6.1.2.1.25.2.1.3"
	StorageTypeFixedDisk     = "1.3.6.1.2.1.25.2.1.4"
	StorageTypeRemovableDisk = "1.3.6.1.2.1.25.2.1.5"
	StorageTypeFloppyDisk    = "1.3.6.1.2.1.25.2.1.6"
	StorageTypeCompactDisc   = "1.3.6.1.2.1.25.2.1.7"
	StorageTypeRAMDisk       = "1.3.6.1.2.1.25.2.1.8"
	StorageTypeFlashMemory   = "1.3.6.1.2.1.25.2.1.9"
	StorageTypeNetworkDisk   = "1.3.6.1.2.1.25.2.1.10"
)

// Storage is one hrStorageTable row, with allocation units already
// multiplied through so SizeBytes / UsedBytes are absolute.
type Storage struct {
	Index           uint32
	TypeOID         string // canonical OID string of hrStorageType
	Description     string
	AllocationUnits uint32
	SizeBytes       uint64
	UsedBytes       uint64
}

// Processor is one hrProcessorTable row.
type Processor struct {
	Index   uint32
	LoadPct int // 0..100, "average load over last minute"
}

// Observation is the per-poll host-resources snapshot.
type Observation struct {
	ClientID     string
	TargetID     string
	ObservedAt   time.Time
	SystemUptime uint32 // TimeTicks (hundredths of a second since boot)
	Storage      []Storage
	Processors   []Processor
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishHostResources(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns a host-resources Collector. Pass nil now to use
// [time.Now] in UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect performs one Get + two Walks and publishes the result.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("hostresources: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("hostresources: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("hostresources: dial: %w", err)
	}

	observedAt := c.now()

	scalar, err := client.Get(ctx, []string{oidHrSystemUptime})
	if err != nil {
		return fmt.Errorf("hostresources: get hrSystemUptime: %w", err)
	}
	storageVbs, err := client.Walk(ctx, storageTablePrefix)
	if err != nil {
		return fmt.Errorf("hostresources: walk hrStorageTable: %w", err)
	}
	procVbs, err := client.Walk(ctx, processorTablePrefix)
	if err != nil {
		return fmt.Errorf("hostresources: walk hrProcessorTable: %w", err)
	}

	obs := Observation{
		ClientID:     target.ClientID,
		TargetID:     target.ID,
		ObservedAt:   observedAt,
		SystemUptime: extractUptime(scalar),
		Storage:      buildStorage(storageVbs),
		Processors:   buildProcessors(procVbs),
	}
	if pubErr := c.publisher.PublishHostResources(ctx, obs); pubErr != nil {
		return fmt.Errorf("hostresources: publish: %w", pubErr)
	}
	return nil
}

// extractUptime pulls the TimeTicks scalar from a single-element Get
// response. Missing or wrong-typed values yield 0.
func extractUptime(vbs []snmp.Varbind) uint32 {
	for _, vb := range vbs {
		if vb.OID == oidHrSystemUptime {
			return uint32Value(vb.Value)
		}
	}
	return 0
}

// storageRow holds raw + computed fields during the build pass — the
// allocation-unit multiply happens after the walk completes so column
// order in the varbind stream doesn't matter.
type storageRow struct {
	row           Storage
	hasAllocUnits bool
	rawSizeUnits  uint64
	rawUsedUnits  uint64
}

func buildStorage(vbs []snmp.Varbind) []Storage {
	rows := make(map[uint32]*storageRow)
	for _, vb := range vbs {
		col, idx, ok := parseTableOID(vb.OID, storageTablePrefix, indexFieldsStorage)
		if !ok {
			continue
		}
		r := rows[idx]
		if r == nil {
			r = &storageRow{row: Storage{Index: idx}}
			rows[idx] = r
		}
		applyStorageColumn(r, col, vb.Value)
	}

	out := make([]Storage, 0, len(rows))
	for _, r := range rows {
		if r.hasAllocUnits {
			r.row.SizeBytes = r.rawSizeUnits * uint64(r.row.AllocationUnits)
			r.row.UsedBytes = r.rawUsedUnits * uint64(r.row.AllocationUnits)
		}
		out = append(out, r.row)
	}
	sortStorage(out)
	return out
}

func applyStorageColumn(r *storageRow, col string, v any) {
	switch col {
	case colStorageType:
		r.row.TypeOID = oidString(v)
	case colStorageDescr:
		r.row.Description = stringValue(v)
	case colStorageAllocUnits:
		r.row.AllocationUnits = uint32Value(v)
		r.hasAllocUnits = r.row.AllocationUnits > 0
	case colStorageSize:
		r.rawSizeUnits = uint64Value(v)
	case colStorageUsed:
		r.rawUsedUnits = uint64Value(v)
	}
}

func buildProcessors(vbs []snmp.Varbind) []Processor {
	rows := make(map[uint32]*Processor)
	for _, vb := range vbs {
		col, idx, ok := parseTableOID(vb.OID, processorTablePrefix, indexFieldsProcessor)
		if !ok {
			continue
		}
		p := rows[idx]
		if p == nil {
			p = &Processor{Index: idx}
			rows[idx] = p
		}
		if col == colProcessorLoad {
			p.LoadPct = intValue(vb.Value)
		}
	}

	out := make([]Processor, 0, len(rows))
	for _, p := range rows {
		out = append(out, *p)
	}
	sortProcessors(out)
	return out
}

// parseTableOID splits an OID under prefix into (column, index). The
// expected suffix is <column>.<index> when indexFields==1.
func parseTableOID(oid, prefix string, indexFields int) (string, uint32, bool) {
	if !strings.HasPrefix(oid, prefix+".") {
		return "", 0, false
	}
	rest := strings.TrimPrefix(oid, prefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFields {
		return "", 0, false
	}
	idx, err := parseUint32(parts[1])
	if err != nil {
		return "", 0, false
	}
	return parts[0], idx, true
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// oidString accepts both the gosnmp OID variants: string ("1.2.3.4")
// or []byte (BER-encoded).
func oidString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	}
	return ""
}

func stringValue(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
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

// uint64Value extracts a 64-bit unsigned (hrStorageSize/Used are
// INTEGER in the MIB but real values exceed 2^31 — many agents emit
// uint64 or wrap via Counter64).
func uint64Value(v any) uint64 {
	switch t := v.(type) {
	case nil:
		return 0
	case uint:
		return uint64(t)
	case uint32:
		return uint64(t)
	case uint64:
		return t
	case int:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case int32:
		if t < 0 {
			return 0
		}
		return uint64(t)
	case int64:
		if t < 0 {
			return 0
		}
		return uint64(t)
	}
	return 0
}

func sortStorage(ss []Storage) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j-1].Index > ss[j].Index; j-- {
			ss[j-1], ss[j] = ss[j], ss[j-1]
		}
	}
}

func sortProcessors(ps []Processor) {
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0 && ps[j-1].Index > ps[j].Index; j-- {
			ps[j-1], ps[j] = ps[j], ps[j-1]
		}
	}
}
