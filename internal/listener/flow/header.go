package flow

import (
	"encoding/binary"
	"net/netip"
)

// packetInfo is the flow key and IP length read from one sampled packet.
type packetInfo struct {
	src, dst         netip.Addr
	srcPort, dstPort uint16
	protocol         uint8
	tcpFlags         uint8
	// length is the IP packet's length: the IPv4 total length, or the
	// IPv6 payload length plus its fixed header.
	length uint32
}

const (
	ethHeaderLen    = 14
	vlanTagLen      = 4
	etherTypeIPv4   = 0x0800
	etherTypeIPv6   = 0x86DD
	etherType8021Q  = 0x8100
	etherType8021AD = 0x88A8
	// etherTypeQinQ is the pre-standard outer tag some switches still send.
	etherTypeQinQ = 0x9100
	// maxVLANTags bounds the tag walk; QinQ stacks two.
	maxVLANTags = 4

	ipVersion4       = 4
	ipVersion6       = 6
	ipv4IHLMask      = 0x0F
	ipv4WordLen      = 4
	ipv4MinHeaderLen = 20
	ipv4FragOffset   = 0x1FFF
	ipv6HeaderLen    = 40
	ipv6ExtUnit      = 8
	// ipv6ExtMinLen reaches an extension header's length byte.
	ipv6ExtMinLen   = 2
	ipv6FragmentLen = 8
	// maxIPv6ExtHeaders bounds the extension-header walk.
	maxIPv6ExtHeaders = 8

	protoHopByHop  = 0
	protoTCP       = 6
	protoUDP       = 17
	protoRouting   = 43
	protoFragment  = 44
	protoDestOpts  = 60
	tcpFlagsOffset = 13
	portsLen       = 4
)

// parseEthernet skips an Ethernet header and any VLAN tags, then parses
// the IP packet behind them.
func parseEthernet(b []byte) (packetInfo, bool) {
	if len(b) < ethHeaderLen {
		return packetInfo{}, false
	}
	etherType := binary.BigEndian.Uint16(b[12:])
	b = b[ethHeaderLen:]
	for range maxVLANTags {
		if etherType != etherType8021Q && etherType != etherType8021AD && etherType != etherTypeQinQ {
			break
		}
		if len(b) < vlanTagLen {
			return packetInfo{}, false
		}
		etherType = binary.BigEndian.Uint16(b[2:])
		b = b[vlanTagLen:]
	}
	if etherType != etherTypeIPv4 && etherType != etherTypeIPv6 {
		return packetInfo{}, false
	}
	return parseIP(b)
}

// parseIP parses an IPv4 or IPv6 packet, picking by the version nibble.
// Ports and TCP flags are read when the header reaches them; a sampled
// header is often cut short, and a non-first fragment has none.
func parseIP(b []byte) (packetInfo, bool) {
	if len(b) == 0 {
		return packetInfo{}, false
	}
	switch b[0] >> 4 {
	case ipVersion4:
		return parseIPv4(b)
	case ipVersion6:
		return parseIPv6(b)
	}
	return packetInfo{}, false
}

func parseIPv4(b []byte) (packetInfo, bool) {
	if len(b) < ipv4MinHeaderLen {
		return packetInfo{}, false
	}
	be := binary.BigEndian
	p := packetInfo{
		length:   uint32(be.Uint16(b[2:])),
		protocol: b[9],
		src:      netip.AddrFrom4([4]byte(b[12:16])),
		dst:      netip.AddrFrom4([4]byte(b[16:20])),
	}
	ihl := int(b[0]&ipv4IHLMask) * ipv4WordLen
	if ihl < ipv4MinHeaderLen || be.Uint16(b[6:])&ipv4FragOffset != 0 || len(b) < ihl {
		return p, true
	}
	p.transport(b[ihl:])
	return p, true
}

func parseIPv6(b []byte) (packetInfo, bool) {
	if len(b) < ipv6HeaderLen {
		return packetInfo{}, false
	}
	p := packetInfo{
		length: ipv6HeaderLen + uint32(binary.BigEndian.Uint16(b[4:])),
		src:    netip.AddrFrom16([16]byte(b[8:24])),
		dst:    netip.AddrFrom16([16]byte(b[24:40])),
	}
	next, rest := b[6], b[ipv6HeaderLen:]
	for range maxIPv6ExtHeaders {
		var n int
		switch next {
		case protoHopByHop, protoRouting, protoDestOpts:
			if len(rest) < ipv6ExtMinLen {
				p.protocol = next
				return p, true
			}
			n = (int(rest[1]) + 1) * ipv6ExtUnit
		case protoFragment:
			if len(rest) < ipv6FragmentLen {
				p.protocol = next
				return p, true
			}
			if binary.BigEndian.Uint16(rest[2:])>>3 != 0 {
				p.protocol = rest[0]
				return p, true
			}
			n = ipv6FragmentLen
		default:
			p.protocol = next
			p.transport(rest)
			return p, true
		}
		if len(rest) < n {
			p.protocol = rest[0]
			return p, true
		}
		next, rest = rest[0], rest[n:]
	}
	p.protocol = next
	return p, true
}

// transport reads the ports, and for TCP the flags, from the start of
// the layer-4 header.
func (p *packetInfo) transport(b []byte) {
	if (p.protocol != protoTCP && p.protocol != protoUDP) || len(b) < portsLen {
		return
	}
	p.srcPort = binary.BigEndian.Uint16(b)
	p.dstPort = binary.BigEndian.Uint16(b[2:])
	if p.protocol == protoTCP && len(b) > tcpFlagsOffset {
		p.tcpFlags = b[tcpFlagsOffset]
	}
}
