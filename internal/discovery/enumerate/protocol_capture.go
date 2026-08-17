package enumerate

import (
	"fmt"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// dot1qTagLength is the size of a single 802.1Q tag: 2 bytes of TPID plus 2 of
// TCI, inserted between the Ethernet source address and the type field.
const dot1qTagLength = 4

// ethernetTypeOffset is where the type field sits in an untagged Ethernet frame.
const ethernetTypeOffset = 12

// dot1qVLANMask selects the 12-bit VLAN ID out of the TCI, discarding the
// 3-bit priority and the drop-eligible flag above it.
const dot1qVLANMask = 0x0FFF

// VLANUntagged is the ObservedVLAN of a neighbour heard on an untagged frame.
// 0 is not a usable VLAN ID (802.1Q reserves it for priority tagging), so it
// doubles as "no tag" without needing a second field.
const VLANUntagged = 0

// taggedPacket is a decoded frame plus the VLAN it arrived on.
type taggedPacket struct {
	Packet gopacket.Packet
	VLAN   uint16
}

// stripDot1Q removes a single 802.1Q tag and reports the VLAN it carried.
//
// gopacket's Dot1Q layer passes its inner type field straight to
// EthernetType.LayerType(). The Ethernet layer treats a value below 0x0600 as an
// 802.3 length and continues into LLC, but Dot1Q has no such branch, so a tagged
// frame carrying LLC/SNAP — which is how CDP and EDP are framed — stops at Dot1Q
// with its payload undecoded. Removing the tag restores the untagged layout the
// Ethernet decoder already handles, and keeps the source address the neighbour
// records report.
//
// Only one tag is removed. QinQ (0x88a8, 0x9100) is out of scope: no product
// surface consumes a second tag.
func stripDot1Q(data []byte) ([]byte, uint16) {
	if !isDot1QTagged(data) {
		return data, VLANUntagged
	}

	vlan := (uint16(data[ethernetTypeOffset+2])<<8 | uint16(data[ethernetTypeOffset+3])) & dot1qVLANMask

	stripped := make([]byte, 0, len(data)-dot1qTagLength)
	stripped = append(stripped, data[:ethernetTypeOffset]...)
	stripped = append(stripped, data[ethernetTypeOffset+dot1qTagLength:]...)
	return stripped, vlan
}

// readTaggedPackets decodes frames off handle until it errors, forwarding each
// with the VLAN it arrived on.
//
// This replaces gopacket.PacketSource because the tag has to be read and removed
// in the same step that decodes the frame — a PacketSource would hand back a
// packet with the VLAN already discarded, and pairing them afterwards would race.
// The channel closes when the handle does, which is what Stop relies on.
func readTaggedPackets(handle capture.Handle, linkType layers.LinkType) <-chan taggedPacket {
	out := make(chan taggedPacket)
	go func() {
		defer close(out)
		for {
			data, _, err := handle.ReadPacketData()
			if err != nil {
				return
			}
			frame, vlan := data, uint16(VLANUntagged)
			if linkType == layers.LinkTypeEthernet {
				frame, vlan = stripDot1Q(data)
			}
			out <- taggedPacket{
				Packet: gopacket.NewPacket(frame, linkType, gopacket.Default),
				VLAN:   vlan,
			}
		}
	}()
	return out
}

// isDot1QTagged reports whether data is an Ethernet frame long enough to carry a
// single 802.1Q tag and an inner type field.
func isDot1QTagged(data []byte) bool {
	if len(data) < ethernetTypeOffset+dot1qTagLength+2 {
		return false
	}
	etherType := layers.EthernetType(uint16(data[ethernetTypeOffset])<<8 | uint16(data[ethernetTypeOffset+1]))
	return etherType == layers.EthernetTypeDot1Q
}

// openProtocolCapture opens a live capture handle on iface in promiscuous mode,
// installs the given BPF filter, and returns the handle with its link type.
//
// It is shared by the LLDP, CDP, and EDP captures, whose open sequences are
// identical apart from the snapshot length and BPF expression. Centralizing it
// keeps the three Start methods small and free of duplicated open/filter logic.
func openProtocolCapture(
	opener capture.Opener,
	iface string,
	snaplen int32,
	bpfFilter string,
) (capture.Handle, layers.LinkType, error) {
	handle, err := opener.OpenLive(iface, snaplen, true, capture.BlockForever)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open capture: %w", err)
	}
	if filterErr := handle.SetBPFFilter(bpfFilter); filterErr != nil {
		handle.Close()
		return nil, 0, fmt.Errorf("failed to set BPF filter: %w", filterErr)
	}

	return handle, handle.LinkType(), nil
}
