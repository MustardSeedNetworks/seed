package packetcapture

import (
	"bytes"
	"cmp"
	"errors"
	"net/netip"
	"slices"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

const (
	// summaryTop is how many talkers and flows a summary lists.
	summaryTop = 10
	// maxTracked bounds the distinct hosts and the distinct flows a summary
	// tallies, so a scan's one-frame flows cannot grow it without limit.
	maxTracked = 1 << 16
)

// Summary is what a finished capture held, tallied as its frames were
// written (#239), so an operator sees the shape of the traffic before
// opening the file in Wireshark. Byte counts are wire lengths.
type Summary struct {
	// Protocols counts frames by the innermost protocol they were decoded
	// to, such as "DNS", "TCP" or "ARP"; a protocol this summary does not
	// decode is named but not looked into, such as "TLS" or "NTP".
	Protocols  []ProtocolCount `json:"protocols"`
	TopTalkers []Talker        `json:"topTalkers"`
	TopFlows   []Flow          `json:"topFlows"`
	// DNSQueries counts DNS and mDNS questions.
	DNSQueries int `json:"dnsQueries"`
	// TCPConnections counts connection attempts: SYNs without an ACK.
	TCPConnections int `json:"tcpConnections"`
	// HTTPRequests counts TCP segments that open with an HTTP/1.x request
	// line. Requests inside TLS or HTTP/2 are not visible.
	HTTPRequests int `json:"httpRequests"`
	// Truncated reports that more distinct hosts or flows crossed the wire
	// than a summary tracks; the lists cover the first ones seen.
	Truncated bool `json:"truncated"`
}

// ProtocolCount is the frames of one protocol.
type ProtocolCount struct {
	Name    string `json:"name"`
	Packets int    `json:"packets"`
	Bytes   int64  `json:"bytes"`
}

// Talker is one IP address and the traffic it sent and received.
type Talker struct {
	Address       string `json:"address"`
	Packets       int    `json:"packets"`
	BytesSent     int64  `json:"bytesSent"`
	BytesReceived int64  `json:"bytesReceived"`
}

// Flow is one TCP or UDP conversation, both directions together. Source is
// the endpoint that sent its first frame.
type Flow struct {
	Transport   string `json:"transport"   jsonschema:"enum=TCP,enum=UDP"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Packets     int    `json:"packets"`
	Bytes       int64  `json:"bytes"`
	DurationMs  int64  `json:"durationMs"`
}

// flowKey names a conversation whichever way a frame travels: lo and hi are
// its endpoints in address order.
type flowKey struct {
	transport string
	lo, hi    netip.AddrPort
}

type flowTally struct {
	Flow

	first, last time.Time
}

// summarizer tallies frames into a Summary. The layers are decoded in place
// by one DecodingLayerParser, so a frame costs no allocation unless it names
// a protocol, host or flow not seen before.
type summarizer struct {
	link    layers.LinkType
	parser  *gopacket.DecodingLayerParser
	decoded []gopacket.LayerType

	eth     layers.Ethernet
	loop    layers.Loopback
	sll     layers.LinuxSLL
	dot1q   layers.Dot1Q
	arp     layers.ARP
	ip4     layers.IPv4
	ip6     layers.IPv6
	icmp4   layers.ICMPv4
	icmp6   layers.ICMPv6
	tcp     layers.TCP
	udp     layers.UDP
	dns     layers.DNS
	dhcp    layers.DHCPv4
	payload gopacket.Payload

	protocols map[string]*ProtocolCount
	hosts     map[netip.Addr]*Talker
	flows     map[flowKey]*flowTally
	sum       Summary
}

// newSummarizer returns a summarizer for frames of the given link type. A
// link type it has no decoder for still counts every frame, under the link
// type's name.
func newSummarizer(link layers.LinkType) *summarizer {
	s := &summarizer{
		link:      link,
		protocols: make(map[string]*ProtocolCount),
		hosts:     make(map[netip.Addr]*Talker),
		flows:     make(map[flowKey]*flowTally),
	}
	if first, ok := firstLayer(link); ok {
		s.parser = gopacket.NewDecodingLayerParser(first,
			&s.eth, &s.loop, &s.sll, &s.dot1q, &s.arp, &s.ip4, &s.ip6, &s.icmp4, &s.icmp6,
			&s.tcp, &s.udp, &s.dns, &s.dhcp, &s.payload)
	}
	return s
}

// firstLayer returns the layer a frame of the link type starts with, for the
// link types capture interfaces hand over: Ethernet (every NIC, and Linux
// loopback), BSD loopback (macOS lo0) and Linux cooked (the "any" device).
// gopacket's LinkType.LayerType leaves Ethernet unset, so it cannot say.
func firstLayer(link layers.LinkType) (gopacket.LayerType, bool) {
	if link == layers.LinkTypeEthernet {
		return layers.LayerTypeEthernet, true
	}
	if link == layers.LinkTypeNull || link == layers.LinkTypeLoop {
		return layers.LayerTypeLoopback, true
	}
	if link == layers.LinkTypeLinuxSLL {
		return layers.LayerTypeLinuxSLL, true
	}
	return 0, false
}

// add tallies one frame.
func (s *summarizer) add(ci gopacket.CaptureInfo, data []byte) {
	size := int64(ci.Length)
	if s.parser == nil {
		s.count(s.link.String(), size)
		return
	}
	err := s.parser.DecodeLayers(data, &s.decoded)
	s.count(s.protocolName(err), size)

	src, dst, ok := s.addresses()
	if !ok {
		return
	}
	s.talk(src, dst, size)
	for _, lt := range s.decoded {
		switch lt {
		case layers.LayerTypeTCP:
			if s.tcp.SYN && !s.tcp.ACK {
				s.sum.TCPConnections++
			}
			s.flow("TCP", netip.AddrPortFrom(src, uint16(s.tcp.SrcPort)),
				netip.AddrPortFrom(dst, uint16(s.tcp.DstPort)), ci, size)
		case layers.LayerTypeUDP:
			s.flow("UDP", netip.AddrPortFrom(src, uint16(s.udp.SrcPort)),
				netip.AddrPortFrom(dst, uint16(s.udp.DstPort)), ci, size)
		case layers.LayerTypeDNS:
			if !s.dns.QR {
				s.sum.DNSQueries++
			}
		case gopacket.LayerTypePayload:
			if slices.Contains(s.decoded, layers.LayerTypeTCP) && isHTTPRequest(s.payload) {
				s.sum.HTTPRequests++
			}
		}
	}
}

// addresses returns the frame's IP source and destination, if it has an IP
// layer.
func (s *summarizer) addresses() (netip.Addr, netip.Addr, bool) {
	for _, lt := range s.decoded {
		switch lt {
		case layers.LayerTypeIPv4:
			src, _ := netip.AddrFromSlice(s.ip4.SrcIP)
			dst, _ := netip.AddrFromSlice(s.ip4.DstIP)
			return src, dst, true
		case layers.LayerTypeIPv6:
			src, _ := netip.AddrFromSlice(s.ip6.SrcIP)
			dst, _ := netip.AddrFromSlice(s.ip6.DstIP)
			return src, dst, true
		}
	}
	return netip.Addr{}, netip.Addr{}, false
}

// talk tallies a frame against the hosts that sent and received it.
func (s *summarizer) talk(src, dst netip.Addr, size int64) {
	if t := s.host(src); t != nil {
		t.Packets++
		t.BytesSent += size
	}
	if t := s.host(dst); t != nil {
		if dst != src { // a frame to itself is still one frame
			t.Packets++
		}
		t.BytesReceived += size
	}
}

// protocolName names a decoded frame by its innermost protocol: the layer
// the parser had no decoder for, or else the last one it decoded. A frame
// that failed to decode at all is named after its link type.
func (s *summarizer) protocolName(err error) string {
	if unsupported, ok := errors.AsType[gopacket.UnsupportedLayerType](err); ok {
		return gopacket.LayerType(unsupported).String()
	}
	for _, lt := range slices.Backward(s.decoded) {
		if lt != gopacket.LayerTypePayload {
			return lt.String()
		}
	}
	return s.link.String()
}

func (s *summarizer) count(name string, size int64) {
	p, ok := s.protocols[name]
	if !ok {
		p = &ProtocolCount{Name: name}
		s.protocols[name] = p
	}
	p.Packets++
	p.Bytes += size
}

// host returns the tally for addr, or nil once maxTracked hosts are tallied
// and addr is not one of them.
func (s *summarizer) host(addr netip.Addr) *Talker {
	if t, ok := s.hosts[addr]; ok {
		return t
	}
	if len(s.hosts) >= maxTracked {
		s.sum.Truncated = true
		return nil
	}
	t := &Talker{Address: addr.String()}
	s.hosts[addr] = t
	return t
}

func (s *summarizer) flow(transport string, src, dst netip.AddrPort, ci gopacket.CaptureInfo, size int64) {
	key := flowKey{transport: transport, lo: src, hi: dst}
	if src.Compare(dst) > 0 {
		key.lo, key.hi = dst, src
	}
	f, ok := s.flows[key]
	if !ok {
		if len(s.flows) >= maxTracked {
			s.sum.Truncated = true
			return
		}
		f = &flowTally{Transport: transport, Source: src.String(), Destination: dst.String(), first: ci.Timestamp}
		s.flows[key] = f
	}
	f.Packets++
	f.Bytes += size
	f.last = ci.Timestamp
}

// summary returns the tallies, protocols by frame count and the top
// talkers and flows by bytes. Every list is empty rather than nil.
func (s *summarizer) summary() Summary {
	sum := s.sum
	sum.Protocols = make([]ProtocolCount, 0, len(s.protocols))
	for _, p := range s.protocols {
		sum.Protocols = append(sum.Protocols, *p)
	}
	slices.SortFunc(sum.Protocols, func(a, b ProtocolCount) int {
		return cmp.Or(cmp.Compare(b.Packets, a.Packets), cmp.Compare(a.Name, b.Name))
	})

	sum.TopTalkers = make([]Talker, 0, len(s.hosts))
	for _, t := range s.hosts {
		sum.TopTalkers = append(sum.TopTalkers, *t)
	}
	slices.SortFunc(sum.TopTalkers, func(a, b Talker) int {
		return cmp.Or(
			cmp.Compare(b.BytesSent+b.BytesReceived, a.BytesSent+a.BytesReceived),
			cmp.Compare(a.Address, b.Address))
	})
	sum.TopTalkers = sum.TopTalkers[:min(len(sum.TopTalkers), summaryTop)]

	sum.TopFlows = make([]Flow, 0, len(s.flows))
	for _, f := range s.flows {
		flow := f.Flow
		flow.DurationMs = f.last.Sub(f.first).Milliseconds()
		sum.TopFlows = append(sum.TopFlows, flow)
	}
	slices.SortFunc(sum.TopFlows, func(a, b Flow) int {
		return cmp.Or(cmp.Compare(b.Bytes, a.Bytes), cmp.Compare(a.Source, b.Source),
			cmp.Compare(a.Destination, b.Destination))
	})
	sum.TopFlows = sum.TopFlows[:min(len(sum.TopFlows), summaryTop)]
	return sum
}

// isHTTPRequest reports whether a TCP payload opens with an HTTP/1.x request
// line: a method, a target, and the version.
func isHTTPRequest(payload []byte) bool {
	line, _, _ := bytes.Cut(payload, []byte("\r\n"))
	method, rest, _ := bytes.Cut(line, []byte(" "))
	switch string(method) {
	case "GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH", "CONNECT", "TRACE":
		return bytes.Contains(rest, []byte(" HTTP/1."))
	}
	return false
}
