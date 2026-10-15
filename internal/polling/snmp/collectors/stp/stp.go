// Package stp implements the stp SNMP Collector: spanning-tree state from
// BRIDGE-MIB (RFC 4188). One pass reads
//
//   - the dot1dStp scalars (1.3.6.1.2.1.17.2): the root bridge this switch
//     believes in, its root port and cost, and dot1dStpTopChanges;
//   - dot1dStpPortTable (1.3.6.1.2.1.17.2.15.1): each port's state, its
//     designated bridge and its forward-transition counter;
//   - dot1dBasePortIfIndex (1.3.6.1.2.1.17.1.4.1.2), so a bridge port can be
//     named by the interface it is.
//
// BRIDGE-MIB has no port-role column, so each port's role is derived the way
// 802.1D defines it: the root port is the one dot1dStpRootPort names, a port
// whose designated bridge is this bridge is designated (backup when it is
// blocking), and any other blocking port is an alternate.
package stp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/polling/snmp"
)

// Name is the collector key used in polling_targets.collector_chain.
const Name = "stp"

const (
	oidBaseBridgeAddress = "1.3.6.1.2.1.17.1.1.0"
	basePortIfIndexCol   = "1.3.6.1.2.1.17.1.4.1.2"

	oidStpProtocol    = "1.3.6.1.2.1.17.2.1.0"
	oidStpPriority    = "1.3.6.1.2.1.17.2.2.0"
	oidStpSinceChange = "1.3.6.1.2.1.17.2.3.0"
	oidStpTopChanges  = "1.3.6.1.2.1.17.2.4.0"
	oidStpRoot        = "1.3.6.1.2.1.17.2.5.0"
	oidStpRootCost    = "1.3.6.1.2.1.17.2.6.0"
	oidStpRootPort    = "1.3.6.1.2.1.17.2.7.0"

	portTablePrefix = "1.3.6.1.2.1.17.2.15.1"

	colPortState              = "3"
	colPortEnable             = "4"
	colPortPathCost           = "5"
	colPortDesignatedBridge   = "8"
	colPortForwardTransitions = "10"

	bridgeIDBytes   = 8 // 2-octet priority + 6-octet MAC
	macAddressBytes = 6
	ticksPerSecond  = 100
)

// Port states (dot1dStpPortState).
const (
	StateDisabled   = 1
	StateBlocking   = 2
	StateListening  = 3
	StateLearning   = 4
	StateForwarding = 5
	StateBroken     = 6
)

// portDisabled is dot1dStpPortEnable's disabled(2).
const portDisabled = 2

// Port roles, derived per 802.1D (BRIDGE-MIB does not carry them).
const (
	RoleRoot       = "root"
	RoleDesignated = "designated"
	RoleAlternate  = "alternate"
	RoleBackup     = "backup"
	RoleDisabled   = "disabled"
)

// BridgeID is an 802.1D bridge identifier: the 16-bit priority (including
// any 802.1t system-ID extension) and the bridge MAC address.
type BridgeID struct {
	Priority uint16
	Address  string
}

// String renders the identifier as priority/MAC, the form alert text uses.
func (b BridgeID) String() string {
	return fmt.Sprintf("%d/%s", b.Priority, b.Address)
}

// Port is one dot1dStpPortTable row with its derived role.
type Port struct {
	Port               int
	IfIndex            int
	State              int
	Role               string
	PathCost           int
	DesignatedBridge   BridgeID
	ForwardTransitions uint32
}

// Observation is the per-poll spanning-tree snapshot. Root is the zero
// BridgeID when the device runs no spanning tree or does not implement
// BRIDGE-MIB's dot1dStp group.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time

	Protocol int
	Bridge   BridgeID
	Root     BridgeID
	RootCost int
	// RootPort is the bridge port toward the root, 0 on the root bridge.
	RootPort int
	// TopologyChanges is dot1dStpTopChanges, a Counter32.
	TopologyChanges uint32
	// SecondsSinceTopologyChange is dot1dStpTimeSinceTopologyChange.
	SecondsSinceTopologyChange uint32
	Ports                      []Port
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishSTP(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns an STP Collector. Pass nil now to use [time.Now] UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect reads the dot1dStp group and publishes one Observation.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("stp: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("stp: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("stp: dial: %w", err)
	}

	observedAt := c.now()
	scalars, err := client.Get(ctx, []string{
		oidBaseBridgeAddress, oidStpProtocol, oidStpPriority, oidStpSinceChange,
		oidStpTopChanges, oidStpRoot, oidStpRootCost, oidStpRootPort,
	})
	if err != nil {
		return fmt.Errorf("stp: get dot1dStp scalars: %w", err)
	}
	portVbs, err := client.Walk(ctx, portTablePrefix)
	if err != nil {
		return fmt.Errorf("stp: walk dot1dStpPortTable: %w", err)
	}
	ifIndexVbs, err := client.Walk(ctx, basePortIfIndexCol)
	if err != nil {
		return fmt.Errorf("stp: walk dot1dBasePortIfIndex: %w", err)
	}

	obs := Observation{ClientID: target.ClientID, TargetID: target.ID, ObservedAt: observedAt}
	applyScalars(&obs, scalars)
	obs.Ports = buildPorts(portVbs, ifIndexVbs, obs.Bridge, obs.RootPort)

	if pubErr := c.publisher.PublishSTP(ctx, obs); pubErr != nil {
		return fmt.Errorf("stp: publish: %w", pubErr)
	}
	return nil
}

func applyScalars(obs *Observation, vbs []snmp.Varbind) {
	var mac string
	var priority uint16
	for _, vb := range vbs {
		switch strings.TrimPrefix(vb.OID, ".") {
		case oidBaseBridgeAddress:
			if b, ok := vb.Value.([]byte); ok && len(b) == macAddressBytes {
				mac = net.HardwareAddr(b).String()
			}
		case oidStpProtocol:
			obs.Protocol = integer(vb.Value)
		case oidStpPriority:
			if p := integer(vb.Value); p >= 0 && p <= math.MaxUint16 {
				priority = uint16(p)
			}
		case oidStpSinceChange:
			obs.SecondsSinceTopologyChange = counter(vb.Value) / ticksPerSecond
		case oidStpTopChanges:
			obs.TopologyChanges = counter(vb.Value)
		case oidStpRoot:
			obs.Root = bridgeID(vb.Value)
		case oidStpRootCost:
			obs.RootCost = integer(vb.Value)
		case oidStpRootPort:
			obs.RootPort = integer(vb.Value)
		}
	}
	if mac != "" {
		obs.Bridge = BridgeID{Priority: priority, Address: mac}
	}
}

func buildPorts(portVbs, ifIndexVbs []snmp.Varbind, bridge BridgeID, rootPort int) []Port {
	ifIndex := make(map[int]int, len(ifIndexVbs))
	for _, vb := range ifIndexVbs {
		if port, ok := portIndex(vb.OID, basePortIfIndexCol); ok {
			ifIndex[port] = integer(vb.Value)
		}
	}

	type row struct {
		Port

		disabled bool
	}
	rows := make(map[int]*row)
	for _, vb := range portVbs {
		col, port, ok := portColumn(vb.OID)
		if !ok {
			continue
		}
		r := rows[port]
		if r == nil {
			r = &row{Port: Port{Port: port, IfIndex: ifIndex[port]}}
			rows[port] = r
		}
		switch col {
		case colPortState:
			r.State = integer(vb.Value)
		case colPortEnable:
			r.disabled = integer(vb.Value) == portDisabled
		case colPortPathCost:
			r.PathCost = integer(vb.Value)
		case colPortDesignatedBridge:
			r.DesignatedBridge = bridgeID(vb.Value)
		case colPortForwardTransitions:
			r.ForwardTransitions = counter(vb.Value)
		}
	}

	out := make([]Port, 0, len(rows))
	for _, r := range rows {
		r.Role = role(r.Port, r.disabled, bridge, rootPort)
		out = append(out, r.Port)
	}
	slices.SortFunc(out, func(a, b Port) int { return a.Port - b.Port })
	return out
}

// role derives a port's 802.1D role. It is empty when the bridge's own
// identifier is unknown, because designated and alternate cannot then be
// told apart.
func role(p Port, disabled bool, bridge BridgeID, rootPort int) string {
	switch {
	case disabled || p.State == StateDisabled:
		return RoleDisabled
	case p.Port == rootPort:
		return RoleRoot
	case bridge.Address == "":
		return ""
	case p.DesignatedBridge.Address == bridge.Address:
		if p.State == StateBlocking {
			return RoleBackup
		}
		return RoleDesignated
	default:
		return RoleAlternate
	}
}

// portColumn splits portTablePrefix.<col>.<port>.
func portColumn(oid string) (string, int, bool) {
	rest, ok := strings.CutPrefix(strings.TrimPrefix(oid, "."), portTablePrefix+".")
	if !ok {
		return "", 0, false
	}
	col, idx, ok := strings.Cut(rest, ".")
	if !ok {
		return "", 0, false
	}
	port, err := strconv.Atoi(idx)
	if err != nil {
		return "", 0, false
	}
	return col, port, true
}

// portIndex reads the single-arc port index after a column prefix.
func portIndex(oid, prefix string) (int, bool) {
	idx, ok := strings.CutPrefix(strings.TrimPrefix(oid, "."), prefix+".")
	if !ok {
		return 0, false
	}
	port, err := strconv.Atoi(idx)
	return port, err == nil
}

// bridgeID decodes an 8-octet BridgeId; anything else is the zero value.
func bridgeID(v any) BridgeID {
	b, ok := v.([]byte)
	if !ok || len(b) != bridgeIDBytes {
		return BridgeID{}
	}
	return BridgeID{
		Priority: uint16(b[0])<<8 | uint16(b[1]),
		Address:  net.HardwareAddr(b[2:]).String(),
	}
}

// integer reads an INTEGER column. gosnmp decodes INTEGER as int; anything
// else, or a value outside Integer32, reads as 0.
func integer(v any) int {
	if t, ok := v.(int); ok && t >= math.MinInt32 && t <= math.MaxInt32 {
		return t
	}
	return 0
}

// counter reads a Counter32 or TimeTicks column, which gosnmp decodes as
// uint and uint32 respectively.
func counter(v any) uint32 {
	switch t := v.(type) {
	case uint32:
		return t
	case uint:
		if t <= math.MaxUint32 {
			return uint32(t)
		}
	}
	return 0
}
