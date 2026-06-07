// Package cdp implements the cdp SNMP Collector for Cisco
// Discovery Protocol neighbors. Walks CDP-MIB::cdpCacheTable
// (1.3.6.1.4.1.9.9.23.1.2.1.1) and emits one Observation per poll.
//
// CDP coexists with LLDP on most Cisco devices and stays on the wire
// even when LLDP is disabled (default IOS-XE access-port behavior),
// so a topology built only from LLDP misses edges to IP phones, IP
// cameras, and ME-series gear that announce only via CDP. The lldp
// + cdp observations are merged at the Stage A4 topology reconciler.
//
// Foundry / Brocade FDP uses the same schema under a different OID
// prefix — the parser is parameterized via [WithTablePrefix] so an
// fdp collector instantiates this exact code with the foundry root.
package cdp

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

// Name is the default collector key used in polling_targets.collector_chain.
const Name = "cdp"

// DefaultTablePrefix is Cisco's cdpCacheTable root. The Foundry FDP
// table at 1.3.6.1.4.1.1991.1.1.3.2.2.1 is column-compatible; pass
// it via [WithTablePrefix] to reuse this collector for FDP.
const DefaultTablePrefix = "1.3.6.1.4.1.9.9.23.1.2.1.1"

// cdpCacheTable columns — Cisco's MIB definitions are stable since
// 1996 so these values are safe to hardcode.
const (
	colAddressType  = "3"
	colAddress      = "4"
	colVersion      = "5"
	colDeviceID     = "6"
	colDevicePort   = "7"
	colPlatform     = "8"
	colCapabilities = "9"
	colNativeVLAN   = "11"
	colDuplex       = "12"

	// indexFieldsCDP is the field count after the table prefix:
	// (cdpCacheIfIndex, cdpCacheDeviceIndex).
	indexFieldsCDP = 2

	addrTypeIPv4 = 1
	addrTypeIPv6 = 20

	macAddressBytes = 6
	ipv4AddrBytes   = 4
	ipv6AddrBytes   = 16
)

// Neighbor is one row of cdpCacheTable. LocalIfIndex is the index
// into ifTable on the local device through which this neighbor was
// learned — the bridge to Stage A4 topology edges.
type Neighbor struct {
	LocalIfIndex uint32
	DeviceIndex  uint32
	DeviceID     string
	DevicePort   string
	Platform     string
	Version      string
	Address      string
	Capabilities uint32
	NativeVLAN   int
	Duplex       int
}

// Observation is the per-poll set of CDP neighbors.
type Observation struct {
	ClientID    string
	TargetID    string
	ObservedAt  time.Time
	Neighbors   []Neighbor
	TablePrefix string // honest about which protocol (cdp vs fdp) emitted this
}

// Publisher is the consumer-defined seam. Stage A3.5 wires it to
// topology edge reconciliation alongside LLDP.
type Publisher interface {
	PublishCDP(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector. Default Name is "cdp"; for
// FDP, override the name via [WithName] and the table prefix via
// [WithTablePrefix].
type Collector struct {
	name        string
	tablePrefix string
	newClient   snmp.ClientFactory
	publisher   Publisher
	now         func() time.Time
}

// Option configures a Collector at construction time.
type Option func(*Collector)

// WithName overrides the collector's Name() (e.g. "fdp" for the
// foundry variant). Defaults to "cdp".
func WithName(n string) Option { return func(c *Collector) { c.name = n } }

// WithTablePrefix overrides the table root walked. Defaults to
// [DefaultTablePrefix] (CDP); set to the Foundry root for FDP.
func WithTablePrefix(p string) Option { return func(c *Collector) { c.tablePrefix = p } }

// New returns a CDP Collector bound to factory + publisher. Pass nil
// now to use [time.Now] in UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time, opts ...Option) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	c := &Collector{
		name:        Name,
		tablePrefix: DefaultTablePrefix,
		newClient:   factory,
		publisher:   publisher,
		now:         now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Name implements snmp.Collector. Returns the configured name.
func (c *Collector) Name() string { return c.name }

// Collect walks the configured cache table and publishes the result.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("cdp: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("cdp: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("cdp: dial: %w", err)
	}

	observedAt := c.now()
	vbs, err := client.Walk(ctx, c.tablePrefix)
	if err != nil {
		return fmt.Errorf("cdp: walk: %w", err)
	}

	if pubErr := c.publisher.PublishCDP(ctx, Observation{
		ClientID:    target.ClientID,
		TargetID:    target.ID,
		ObservedAt:  observedAt,
		Neighbors:   c.buildNeighbors(vbs),
		TablePrefix: c.tablePrefix,
	}); pubErr != nil {
		return fmt.Errorf("cdp: publish: %w", pubErr)
	}
	return nil
}

// rowKey identifies one cdpCacheTable row by (ifIndex, deviceIndex).
type rowKey struct {
	ifIndex     uint32
	deviceIndex uint32
}

func (c *Collector) buildNeighbors(vbs []snmp.Varbind) []Neighbor {
	rows := make(map[rowKey]*Neighbor)
	addrTypes := make(map[rowKey]int)

	for _, vb := range vbs {
		col, key, ok := c.parseRowOID(vb.OID)
		if !ok {
			continue
		}
		n := rows[key]
		if n == nil {
			n = &Neighbor{LocalIfIndex: key.ifIndex, DeviceIndex: key.deviceIndex}
			rows[key] = n
		}
		if col == colAddressType {
			addrTypes[key] = intValue(vb.Value)
			continue
		}
		applyColumn(n, col, vb.Value, addrTypes[key])
	}

	out := make([]Neighbor, 0, len(rows))
	for _, n := range rows {
		out = append(out, *n)
	}
	sortNeighbors(out)
	return out
}

// parseRowOID splits an OID under c.tablePrefix into the column
// number and the (ifIndex, deviceIndex) row key.
func (c *Collector) parseRowOID(oid string) (string, rowKey, bool) {
	if !strings.HasPrefix(oid, c.tablePrefix+".") {
		return "", rowKey{}, false
	}
	rest := strings.TrimPrefix(oid, c.tablePrefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFieldsCDP {
		return "", rowKey{}, false
	}
	ifIdx, err1 := parseUint32(parts[1])
	devIdx, err2 := parseUint32(parts[2])
	if err1 != nil || err2 != nil {
		return "", rowKey{}, false
	}
	return parts[0], rowKey{ifIndex: ifIdx, deviceIndex: devIdx}, true
}

func parseUint32(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// applyColumn writes one cdpCacheTable column onto n. addrType is
// the row's cdpCacheAddressType so we can decode cdpCacheAddress
// correctly (IPv4 = 4 bytes, IPv6 = 16).
func applyColumn(n *Neighbor, col string, v any, addrType int) {
	switch col {
	case colAddress:
		n.Address = addressString(v, addrType)
	case colVersion:
		n.Version = stringValue(v)
	case colDeviceID:
		n.DeviceID = stringValue(v)
	case colDevicePort:
		n.DevicePort = stringValue(v)
	case colPlatform:
		n.Platform = stringValue(v)
	case colCapabilities:
		n.Capabilities = capabilityValue(v)
	case colNativeVLAN:
		n.NativeVLAN = intValue(v)
	case colDuplex:
		n.Duplex = intValue(v)
	}
}

// addressString decodes cdpCacheAddress given its companion type.
// IPv4 → dotted-quad, IPv6 → canonical, MAC fallback for length 6,
// raw string fallback otherwise.
func addressString(v any, addrType int) string {
	b, ok := v.([]byte)
	if !ok {
		return stringValue(v)
	}
	switch addrType {
	case addrTypeIPv4:
		if len(b) == ipv4AddrBytes {
			addr, addrOk := netip.AddrFromSlice(b)
			if addrOk {
				return addr.String()
			}
		}
	case addrTypeIPv6:
		if len(b) == ipv6AddrBytes {
			addr, addrOk := netip.AddrFromSlice(b)
			if addrOk {
				return addr.String()
			}
		}
	}
	if len(b) == macAddressBytes {
		return formatMAC(b)
	}
	return string(b)
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

// capabilityValue decodes the CDP capability bit field. CDP returns
// the value as an INTEGER (4-byte big-endian) but some agents emit
// it as an OCTET STRING. Handle both.
func capabilityValue(v any) uint32 {
	const bitsPerByte = 8
	const bitsCapWidth = 32
	const maxUint32 uint64 = 1<<32 - 1
	switch t := v.(type) {
	case nil:
		return 0
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
	case uint32:
		return t
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

func sortNeighbors(ns []Neighbor) {
	less := func(i, j int) bool {
		if ns[i].LocalIfIndex != ns[j].LocalIfIndex {
			return ns[i].LocalIfIndex < ns[j].LocalIfIndex
		}
		return ns[i].DeviceIndex < ns[j].DeviceIndex
	}
	for i := 1; i < len(ns); i++ {
		for j := i; j > 0 && less(j, j-1); j-- {
			ns[j-1], ns[j] = ns[j], ns[j-1]
		}
	}
}
