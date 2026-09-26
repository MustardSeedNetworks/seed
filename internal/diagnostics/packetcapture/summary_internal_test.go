package packetcapture

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// addr is 10.0.0.n.
func addr(n byte) net.IP { return net.IPv4(10, 0, 0, n) }

func mac(n byte) net.HardwareAddr { return net.HardwareAddr{0x02, 0, 0, 0, 0, n} }

// serialize builds a frame from ls, computing lengths and checksums.
func serialize(t *testing.T, ls ...gopacket.SerializableLayer) []byte {
	t.Helper()
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	require.NoError(t, gopacket.SerializeLayers(buf, opts, ls...))
	return buf.Bytes()
}

func ethernet(ethType layers.EthernetType) *layers.Ethernet {
	return &layers.Ethernet{SrcMAC: mac(1), DstMAC: mac(2), EthernetType: ethType}
}

func ipv4(src, dst net.IP, proto layers.IPProtocol) *layers.IPv4 {
	return &layers.IPv4{Version: 4, TTL: 64, SrcIP: src, DstIP: dst, Protocol: proto}
}

// tcpFrame is an IPv4 TCP segment from src:sport to dst:dport.
func tcpFrame(t *testing.T, src net.IP, sport uint16, dst net.IP, dport uint16, tcp layers.TCP, payload string) []byte {
	t.Helper()
	ip := ipv4(src, dst, layers.IPProtocolTCP)
	tcp.SrcPort, tcp.DstPort = layers.TCPPort(sport), layers.TCPPort(dport)
	require.NoError(t, tcp.SetNetworkLayerForChecksum(ip))
	return serialize(t, ethernet(layers.EthernetTypeIPv4), ip, &tcp, gopacket.Payload(payload))
}

// udpFrame is an IPv4 UDP datagram from src:sport to dst:dport.
func udpFrame(
	t *testing.T,
	src net.IP,
	sport uint16,
	dst net.IP,
	dport uint16,
	payload ...gopacket.SerializableLayer,
) []byte {
	t.Helper()
	ip := ipv4(src, dst, layers.IPProtocolUDP)
	udp := &layers.UDP{SrcPort: layers.UDPPort(sport), DstPort: layers.UDPPort(dport)}
	require.NoError(t, udp.SetNetworkLayerForChecksum(ip))
	return serialize(t, append([]gopacket.SerializableLayer{ethernet(layers.EthernetTypeIPv4), ip, udp}, payload...)...)
}

func dnsMessage(response bool) *layers.DNS {
	return &layers.DNS{
		ID: 1,
		QR: response,
		Questions: []layers.DNSQuestion{
			{Name: []byte("seed.example"), Type: layers.DNSTypeA, Class: layers.DNSClassIN},
		},
	}
}

// summarize tallies frames one millisecond apart on an Ethernet link.
func summarize(frames ...[]byte) Summary {
	s := newSummarizer(layers.LinkTypeEthernet)
	at := time.Unix(1_700_000_000, 0)
	for i, f := range frames {
		s.add(gopacket.CaptureInfo{Timestamp: at.Add(time.Duration(i) * time.Millisecond), Length: len(f)}, f)
	}
	return s.summary()
}

func TestSummaryNamesTheInnermostProtocol(t *testing.T) {
	arp := serialize(t, ethernet(layers.EthernetTypeARP), &layers.ARP{
		AddrType: layers.LinkTypeEthernet, Protocol: layers.EthernetTypeIPv4,
		HwAddressSize: 6, ProtAddressSize: 4, Operation: layers.ARPRequest,
		SourceHwAddress: mac(1), SourceProtAddress: addr(1).To4(),
		DstHwAddress: make(net.HardwareAddr, 6), DstProtAddress: addr(2).To4(),
	})
	icmp := serialize(t, ethernet(layers.EthernetTypeIPv4), ipv4(addr(1), addr(2), layers.IPProtocolICMPv4),
		&layers.ICMPv4{TypeCode: layers.CreateICMPv4TypeCode(layers.ICMPv4TypeEchoRequest, 0)})
	lldp := serialize(t, ethernet(layers.EthernetTypeLinkLayerDiscovery), gopacket.Payload("chassis"))

	tests := []struct {
		name  string
		frame []byte
		want  string
	}{
		{"arp", arp, "ARP"},
		{"icmp", icmp, "ICMPv4"},
		{"dns", udpFrame(t, addr(1), 40000, addr(2), 53, dnsMessage(false)), "DNS"},
		{"tcp with a payload", tcpFrame(t, addr(1), 40000, addr(2), 80, layers.TCP{ACK: true}, "data"), "TCP"},
		{"undecoded application", tcpFrame(t, addr(1), 40000, addr(2), 443, layers.TCP{ACK: true}, "hello"), "TLS"},
		{"undecoded ethertype", lldp, "LinkLayerDiscovery"},
		{"runt frame", []byte{1, 2, 3}, "Ethernet"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sum := summarize(tc.frame)
			require.Len(t, sum.Protocols, 1)
			assert.Equal(t, ProtocolCount{Name: tc.want, Packets: 1, Bytes: int64(len(tc.frame))}, sum.Protocols[0])
		})
	}
}

func TestSummaryCountsEvents(t *testing.T) {
	get := "GET /index.html HTTP/1.1\r\nHost: seed.example\r\n\r\n"
	tests := []struct {
		name   string
		frame  []byte
		dns    int
		syn    int
		http   int
		reason string
	}{
		{name: "dns query", frame: udpFrame(t, addr(1), 40000, addr(2), 53, dnsMessage(false)), dns: 1},
		{
			name:   "dns response",
			frame:  udpFrame(t, addr(2), 53, addr(1), 40000, dnsMessage(true)),
			reason: "an answer is not a query",
		},
		{name: "syn", frame: tcpFrame(t, addr(1), 40000, addr(2), 80, layers.TCP{SYN: true}, ""), syn: 1},
		{
			name:   "syn-ack",
			frame:  tcpFrame(t, addr(2), 80, addr(1), 40000, layers.TCP{SYN: true, ACK: true}, ""),
			reason: "the answer to a SYN is not a new connection",
		},
		{
			name:  "http request",
			frame: tcpFrame(t, addr(1), 40000, addr(2), 8080, layers.TCP{ACK: true, PSH: true}, get),
			http:  1,
		},
		{
			name:  "http response",
			frame: tcpFrame(t, addr(2), 80, addr(1), 40000, layers.TCP{ACK: true}, "HTTP/1.1 200 OK\r\n\r\n"),
		},
		{
			name:  "method without a version",
			frame: tcpFrame(t, addr(1), 40000, addr(2), 80, layers.TCP{ACK: true}, "GET the milk\r\n"),
		},
		{
			name:   "request line over udp",
			frame:  udpFrame(t, addr(1), 40000, addr(2), 8080, gopacket.Payload(get)),
			reason: "HTTP/1.x is TCP",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sum := summarize(tc.frame)
			assert.Equal(t, tc.dns, sum.DNSQueries, tc.reason)
			assert.Equal(t, tc.syn, sum.TCPConnections, tc.reason)
			assert.Equal(t, tc.http, sum.HTTPRequests, tc.reason)
		})
	}
}

func TestSummaryJoinsBothDirectionsOfAFlow(t *testing.T) {
	syn := tcpFrame(t, addr(1), 40000, addr(2), 443, layers.TCP{SYN: true}, "")
	synAck := tcpFrame(t, addr(2), 443, addr(1), 40000, layers.TCP{SYN: true, ACK: true}, "")
	ack := tcpFrame(t, addr(1), 40000, addr(2), 443, layers.TCP{ACK: true}, "")
	query := udpFrame(t, addr(3), 5353, addr(1), 53, dnsMessage(false))

	sum := summarize(syn, synAck, ack, query)

	handshake := int64(len(syn) + len(synAck) + len(ack))
	assert.Equal(t, []Flow{
		{
			Transport:   "TCP",
			Source:      "10.0.0.1:40000",
			Destination: "10.0.0.2:443",
			Packets:     3,
			Bytes:       handshake,
			DurationMs:  2,
		},
		{Transport: "UDP", Source: "10.0.0.3:5353", Destination: "10.0.0.1:53", Packets: 1, Bytes: int64(len(query))},
	}, sum.TopFlows)
	assert.Equal(t, []Talker{
		{
			Address:       "10.0.0.1",
			Packets:       4,
			BytesSent:     int64(len(syn) + len(ack)),
			BytesReceived: int64(len(synAck) + len(query)),
		},
		{Address: "10.0.0.2", Packets: 3, BytesSent: int64(len(synAck)), BytesReceived: int64(len(syn) + len(ack))},
		{Address: "10.0.0.3", Packets: 1, BytesSent: int64(len(query))},
	}, sum.TopTalkers)
	assert.False(t, sum.Truncated)
}

func TestSummaryListsOnlyTheTop(t *testing.T) {
	var frames [][]byte
	// Flow n carries n+1 frames, so the largest are the last ones built.
	for n := range summaryTop + 5 {
		f := udpFrame(t, net.IPv4(10, 1, 0, byte(n+1)), 40000, addr(1), 9, gopacket.Payload("x"))
		for range n + 1 {
			frames = append(frames, f)
		}
	}
	sum := summarize(frames...)

	require.Len(t, sum.TopFlows, summaryTop)
	require.Len(t, sum.TopTalkers, summaryTop)
	assert.Equal(t, "10.1.0.15:40000", sum.TopFlows[0].Source)
	assert.Equal(t, "10.1.0.6:40000", sum.TopFlows[summaryTop-1].Source)
	// The sink receives every frame, so it outranks every sender.
	assert.Equal(t, "10.0.0.1", sum.TopTalkers[0].Address)
	assert.Equal(t, "10.1.0.15", sum.TopTalkers[1].Address)
}

func TestSummaryStopsTrackingNewFlowsAtTheBound(t *testing.T) {
	s := newSummarizer(layers.LinkTypeEthernet)
	f := udpFrame(t, addr(1), 1, addr(2), 9, gopacket.Payload("x"))
	ci := gopacket.CaptureInfo{Timestamp: time.Unix(0, 0), Length: len(f)}
	// Rewrite the source address and port in place, one frame per flow. The
	// ports stay clear of the well-known ones gopacket names a protocol by.
	const srcIPOffset, srcPortOffset = 14 + 12, 14 + 20
	for n := range maxTracked {
		f[srcIPOffset+3] = byte(n)
		port := 50000 + n>>8
		f[srcPortOffset], f[srcPortOffset+1] = byte(port>>8), byte(port)
		s.add(ci, f)
	}
	require.False(t, s.summary().Truncated, "exactly the bound is not over it")

	// A new source address: a flow past the bound.
	f[srcIPOffset+2] = 99
	s.add(ci, f)
	s.add(ci, f)
	sum := s.summary()
	assert.True(t, sum.Truncated)
	assert.Len(t, s.flows, maxTracked)
	require.Len(t, sum.Protocols, 1)
	assert.Equal(t, maxTracked+2, sum.Protocols[0].Packets, "an untracked flow's frames still count")
}

func TestSummaryOfAnUndecodedLinkTypeCountsFrames(t *testing.T) {
	tests := []struct {
		link layers.LinkType
		want string
	}{
		{layers.LinkTypeRaw, "Raw"},
		{layers.LinkTypeIEEE802_11, layers.LinkTypeIEEE802_11.String()},
	}
	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			s := newSummarizer(tc.link)
			s.add(gopacket.CaptureInfo{Length: 100}, make([]byte, 100))
			s.add(gopacket.CaptureInfo{Length: 50}, make([]byte, 50))

			sum := s.summary()
			assert.Equal(t, []ProtocolCount{{Name: tc.want, Packets: 2, Bytes: 150}}, sum.Protocols)
			assert.Empty(t, sum.TopTalkers)
			assert.Empty(t, sum.TopFlows)
		})
	}
}

func TestSummaryDecodesLoopbackFrames(t *testing.T) {
	ip := ipv4(addr(1), addr(1), layers.IPProtocolUDP)
	udp := &layers.UDP{SrcPort: 40000, DstPort: 53}
	require.NoError(t, udp.SetNetworkLayerForChecksum(ip))
	f := serialize(t, &layers.Loopback{Family: layers.ProtocolFamilyIPv4}, ip, udp, dnsMessage(false))

	s := newSummarizer(layers.LinkTypeNull)
	s.add(gopacket.CaptureInfo{Length: len(f)}, f)
	sum := s.summary()
	assert.Equal(t, "DNS", sum.Protocols[0].Name)
	assert.Equal(t, 1, sum.DNSQueries)
	assert.Equal(t, []Talker{
		{Address: "10.0.0.1", Packets: 1, BytesSent: int64(len(f)), BytesReceived: int64(len(f))},
	}, sum.TopTalkers, "a frame a host sends itself is one frame")
}

func TestSummaryListsAreEmptyNotNil(t *testing.T) {
	sum := newSummarizer(layers.LinkTypeEthernet).summary()
	assert.NotNil(t, sum.Protocols)
	assert.NotNil(t, sum.TopTalkers)
	assert.NotNil(t, sum.TopFlows)
}

func TestRunSummarizesWhatItWrote(t *testing.T) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	query := udpFrame(t, addr(1), 40000, addr(2), 53, dnsMessage(false))
	syn := tcpFrame(t, addr(1), 40001, addr(2), 80, layers.TCP{SYN: true}, "")
	h := &fakeHandle{clock: clock, frames: [][]byte{query, syn}}
	s, _ := newSession(t, h, MaxFileBytes)

	res, err := s.run(context.Background(), Request{Interface: "eth0", DurationSeconds: 1}, func(float64) {})
	require.NoError(t, err)

	assert.Equal(t, 2, res.Packets)
	assert.Equal(t, []ProtocolCount{
		{Name: "DNS", Packets: 1, Bytes: int64(len(query))},
		{Name: "TCP", Packets: 1, Bytes: int64(len(syn))},
	}, res.Summary.Protocols)
	assert.Equal(t, 1, res.Summary.DNSQueries)
	assert.Equal(t, 1, res.Summary.TCPConnections)
	assert.Len(t, res.Summary.TopFlows, 2)
}
