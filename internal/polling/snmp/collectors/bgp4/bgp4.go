// Package bgp4 implements the bgp4_mib SNMP Collector. Walks
// BGP4-MIB::bgpPeerTable (1.3.6.1.2.1.15.3.1) and emits one
// Observation per poll listing every BGP peer the target has
// configured.
//
// BGP peer state + the EstablishedTransitions counter are the bread
// and butter of network-wide alerting: a peer flapping between
// established and not-established is one of the most expensive
// silent outages SP/ISP/large-enterprise customers face.
package bgp4

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
const Name = "bgp4_mib"

const (
	tablePrefix = "1.3.6.1.2.1.15.3.1"

	colIdentifier          = "1"
	colState               = "2"
	colAdminStatus         = "3"
	colNegotiatedVersion   = "4"
	colLocalAddr           = "5"
	colLocalPort           = "6"
	colRemotePort          = "8"
	colRemoteAs            = "9"
	colInUpdates           = "10"
	colOutUpdates          = "11"
	colInTotalMessages     = "12"
	colOutTotalMessages    = "13"
	colFsmEstablishedTrans = "15"
	colFsmEstablishedTime  = "16"

	indexFieldsBGP = 4 // 4-octet IPv4 peer-remote-addr
	ipv4OctetCount = 4
)

// PeerState values (RFC 4273).
const (
	StateIdle        = 1
	StateConnect     = 2
	StateActive      = 3
	StateOpenSent    = 4
	StateOpenConfirm = 5
	StateEstablished = 6
)

// PeerAdminStatus values (RFC 4273).
const (
	AdminStop  = 1
	AdminStart = 2
)

// Peer is one bgpPeerTable row. Counter columns are surfaced as raw
// uint64 (Counter32 widened) — Stage A3.5 listener tracks deltas
// across consecutive observations to compute rate.
type Peer struct {
	RemoteAddr             string // dotted-quad IPv4 (the row index)
	Identifier             string // dotted-quad IPv4 BGP ID
	State                  int
	AdminStatus            int
	NegotiatedVersion      int
	LocalAddr              string
	LocalPort              int
	RemotePort             int
	RemoteAS               uint32
	InUpdates              uint64
	OutUpdates             uint64
	InTotalMessages        uint64
	OutTotalMessages       uint64
	EstablishedTransitions uint32
	EstablishedTimeSeconds uint32
}

// Observation is the per-poll BGP peer snapshot.
type Observation struct {
	ClientID   string
	TargetID   string
	ObservedAt time.Time
	Peers      []Peer
}

// Publisher is the consumer-defined seam.
type Publisher interface {
	PublishBGP4(ctx context.Context, obs Observation) error
}

// Collector implements snmp.Collector.
type Collector struct {
	newClient snmp.ClientFactory
	publisher Publisher
	now       func() time.Time
}

// New returns a BGP4 Collector. Pass nil now to use [time.Now] UTC.
func New(factory snmp.ClientFactory, publisher Publisher, now func() time.Time) *Collector {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &Collector{newClient: factory, publisher: publisher, now: now}
}

// Name implements snmp.Collector.
func (*Collector) Name() string { return Name }

// Collect walks bgpPeerTable and publishes the assembled Peer list.
func (c *Collector) Collect(ctx context.Context, target snmp.Target, creds snmp.ResolvedCredentials) error {
	if c.newClient == nil {
		return errors.New("bgp4: client factory not configured")
	}
	if c.publisher == nil {
		return errors.New("bgp4: publisher not configured")
	}

	client, err := c.newClient(target, creds)
	if err != nil {
		return fmt.Errorf("bgp4: dial: %w", err)
	}

	observedAt := c.now()
	vbs, err := client.Walk(ctx, tablePrefix)
	if err != nil {
		return fmt.Errorf("bgp4: walk bgpPeerTable: %w", err)
	}

	if pubErr := c.publisher.PublishBGP4(ctx, Observation{
		ClientID:   target.ClientID,
		TargetID:   target.ID,
		ObservedAt: observedAt,
		Peers:      buildPeers(vbs),
	}); pubErr != nil {
		return fmt.Errorf("bgp4: publish: %w", pubErr)
	}
	return nil
}

func buildPeers(vbs []snmp.Varbind) []Peer {
	rows := make(map[string]*Peer)
	for _, vb := range vbs {
		col, remoteAddr, ok := parsePeerOID(vb.OID)
		if !ok {
			continue
		}
		p := rows[remoteAddr]
		if p == nil {
			p = &Peer{RemoteAddr: remoteAddr}
			rows[remoteAddr] = p
		}
		applyColumn(p, col, vb.Value)
	}

	out := make([]Peer, 0, len(rows))
	for _, p := range rows {
		out = append(out, *p)
	}
	sortPeers(out)
	return out
}

// parsePeerOID expects tablePrefix.col.<4 octet peer IPv4>.
func parsePeerOID(oid string) (string, string, bool) {
	if !strings.HasPrefix(oid, tablePrefix+".") {
		return "", "", false
	}
	rest := strings.TrimPrefix(oid, tablePrefix+".")
	parts := strings.Split(rest, ".")
	if len(parts) != 1+indexFieldsBGP {
		return "", "", false
	}
	var octets [ipv4OctetCount]byte
	for i := range ipv4OctetCount {
		v, err := strconv.ParseUint(parts[1+i], 10, 8)
		if err != nil {
			return "", "", false
		}
		octets[i] = byte(v)
	}
	return parts[0], netip.AddrFrom4(octets).String(), true
}

func applyColumn(p *Peer, col string, v any) {
	switch col {
	case colIdentifier:
		p.Identifier = ipv4OrString(v)
	case colState:
		p.State = intValue(v)
	case colAdminStatus:
		p.AdminStatus = intValue(v)
	case colNegotiatedVersion:
		p.NegotiatedVersion = intValue(v)
	case colLocalAddr:
		p.LocalAddr = ipv4OrString(v)
	case colLocalPort:
		p.LocalPort = intValue(v)
	case colRemotePort:
		p.RemotePort = intValue(v)
	case colRemoteAs:
		p.RemoteAS = uint32Value(v)
	case colInUpdates:
		p.InUpdates = uint64Value(v)
	case colOutUpdates:
		p.OutUpdates = uint64Value(v)
	case colInTotalMessages:
		p.InTotalMessages = uint64Value(v)
	case colOutTotalMessages:
		p.OutTotalMessages = uint64Value(v)
	case colFsmEstablishedTrans:
		p.EstablishedTransitions = uint32Value(v)
	case colFsmEstablishedTime:
		p.EstablishedTimeSeconds = uint32Value(v)
	}
}

// ipv4OrString decodes a BGP IP-address column. RFC 4273 emits these
// as 4-byte OCTET STRINGs; some agents emit them as already-formatted
// strings.
func ipv4OrString(v any) string {
	if b, ok := v.([]byte); ok && len(b) == ipv4OctetCount {
		var octets [ipv4OctetCount]byte
		copy(octets[:], b)
		return netip.AddrFrom4(octets).String()
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

func sortPeers(ps []Peer) {
	for i := 1; i < len(ps); i++ {
		for j := i; j > 0 && ps[j-1].RemoteAddr > ps[j].RemoteAddr; j-- {
			ps[j-1], ps[j] = ps[j], ps[j-1]
		}
	}
}
