package qos

import (
	"bytes"
	"net"
	"net/netip"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

const (
	// frameTTL is the injected probes' IPv4 TTL, the common host default.
	frameTTL = 64

	// ipv4Version, ethernetAddrLen and ipv4AddrLen are the header fields
	// gopacket does not fill in itself.
	ipv4Version     = 4
	ethernetAddrLen = 6
	ipv4AddrLen     = 4

	// frameSnaplen captures a whole probe frame: Ethernet, an optional VLAN
	// tag, IPv4 with options, UDP and probeSize.
	frameSnaplen = 256
)

// serializeOptions fills in every length and checksum the layers leave zero.
func serializeOptions() gopacket.SerializeOptions {
	return gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
}

// framePath is the link-layer route of a single-host check's probes: out of
// one interface, addressed to the other interface's IPv4 address, handed to
// the next hop on the first link.
type framePath struct {
	srcMAC, dstMAC net.HardwareAddr
	src, dst       netip.Addr
	port           uint16
}

// frame builds one probe as an Ethernet frame whose IPv4 header carries dscp.
// The ECN bits stay zero.
func (p framePath) frame(dscp uint8, payload []byte) ([]byte, error) {
	eth := &layers.Ethernet{SrcMAC: p.srcMAC, DstMAC: p.dstMAC, EthernetType: layers.EthernetTypeIPv4}
	ip := &layers.IPv4{
		Version:  ipv4Version,
		TOS:      dscp << ecnBits,
		Flags:    layers.IPv4DontFragment,
		TTL:      frameTTL,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    p.src.AsSlice(),
		DstIP:    p.dst.AsSlice(),
	}
	udp := &layers.UDP{SrcPort: layers.UDPPort(p.port), DstPort: layers.UDPPort(p.port)}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		return nil, err
	}
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, serializeOptions(), eth, ip, udp, gopacket.Payload(payload)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// decodeFrame reads a captured Ethernet frame as an IPv4 UDP datagram and
// returns its sender, the DSCP its header arrived with and its payload.
func decodeFrame(data []byte) (netip.Addr, int, []byte, bool) {
	pkt := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.NoCopy)
	ip, ok := pkt.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
	if !ok {
		return netip.Addr{}, 0, nil, false
	}
	udp, ok := pkt.Layer(layers.LayerTypeUDP).(*layers.UDP)
	if !ok {
		return netip.Addr{}, 0, nil, false
	}
	src, ok := netip.AddrFromSlice(ip.SrcIP)
	if !ok {
		return netip.Addr{}, 0, nil, false
	}
	return src.Unmap(), int(ip.TOS >> ecnBits), udp.Payload, true
}

// arpRequest asks, from iface, who has target.
func arpRequest(from hostInterface, target netip.Addr) ([]byte, error) {
	eth := &layers.Ethernet{SrcMAC: from.mac, DstMAC: layers.EthernetBroadcast, EthernetType: layers.EthernetTypeARP}
	arp := &layers.ARP{
		AddrType:          layers.LinkTypeEthernet,
		Protocol:          layers.EthernetTypeIPv4,
		HwAddressSize:     ethernetAddrLen,
		ProtAddressSize:   ipv4AddrLen,
		Operation:         layers.ARPRequest,
		SourceHwAddress:   from.mac,
		SourceProtAddress: from.prefix.Addr().AsSlice(),
		DstHwAddress:      make([]byte, ethernetAddrLen),
		DstProtAddress:    target.AsSlice(),
	}
	buf := gopacket.NewSerializeBuffer()
	if err := gopacket.SerializeLayers(buf, serializeOptions(), eth, arp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// arpReplyFrom returns the hardware address target answered with, when data
// is its ARP reply.
func arpReplyFrom(data []byte, target netip.Addr) (net.HardwareAddr, bool) {
	pkt := gopacket.NewPacket(data, layers.LayerTypeEthernet, gopacket.NoCopy)
	arp, ok := pkt.Layer(layers.LayerTypeARP).(*layers.ARP)
	if !ok || arp.Operation != layers.ARPReply || !bytes.Equal(arp.SourceProtAddress, target.AsSlice()) {
		return nil, false
	}
	return net.HardwareAddr(bytes.Clone(arp.SourceHwAddress)), true
}
