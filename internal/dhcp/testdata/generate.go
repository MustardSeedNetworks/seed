//go:build ignore

// generate.go builds the DHCP OFFER fixtures in this directory.
//
// Run with: go run ./internal/dhcp/testdata/generate.go
// See README.md for what each fixture is for.

package main

import (
	"fmt"
	"net"
	"os"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// buildOffer serialises a complete Ethernet/IPv4/UDP/DHCP OFFER frame.
func buildOffer(serverIP, serverMAC string, withServerID bool) ([]byte, error) {
	mac, err := net.ParseMAC(serverMAC)
	if err != nil {
		return nil, err
	}
	eth := &layers.Ethernet{
		SrcMAC:       mac,
		DstMAC:       net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, IHL: 5, TTL: 64,
		SrcIP: net.ParseIP(serverIP).To4(), DstIP: net.IPv4bcast,
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{SrcPort: 67, DstPort: 68}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		return nil, err
	}
	opts := layers.DHCPOptions{
		layers.NewDHCPOption(layers.DHCPOptMessageType, []byte{byte(layers.DHCPMsgTypeOffer)}),
	}
	if withServerID {
		opts = append(opts, layers.NewDHCPOption(layers.DHCPOptServerID, net.ParseIP(serverIP).To4()))
	}
	opts = append(opts, layers.NewDHCPOption(layers.DHCPOptEnd, nil))
	dhcp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  6,
		Xid:          0x0be7dead,
		YourClientIP: net.ParseIP("192.0.2.50").To4(),
		NextServerIP: net.ParseIP(serverIP).To4(),
		ClientHWAddr: net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x01},
		Options:      opts,
	}
	buf := gopacket.NewSerializeBuffer()
	err = gopacket.SerializeLayers(buf,
		gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true},
		eth, ip, udp, dhcp)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func main() {
	dir := "internal/dhcp/testdata"
	type fixture struct {
		name         string
		ip, mac      string
		withServerID bool
	}
	for _, f := range []fixture{
		{"dhcp_offer_authorized.bin", "192.0.2.1", "02:00:00:00:00:aa", true},
		{"dhcp_offer_rogue.bin", "192.0.2.66", "02:00:00:00:00:bb", true},
		{"dhcp_offer_no_serverid.bin", "192.0.2.77", "02:00:00:00:00:cc", false},
	} {
		b, err := buildOffer(f.ip, f.mac, f.withServerID)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(dir+"/"+f.name, b, 0o644); err != nil {
			panic(err)
		}
		fmt.Printf("wrote %s (%d bytes)\n", f.name, len(b))
	}
	// A truncated frame: the first 20 bytes only, so no DHCP layer parses.
	full, err := buildOffer("192.0.2.99", "02:00:00:00:00:dd", true)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(dir+"/dhcp_offer_truncated.bin", full[:20], 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote dhcp_offer_truncated.bin (20 bytes)\n")
}
