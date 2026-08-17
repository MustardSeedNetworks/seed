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

// vlanStripper removes a single 802.1Q tag from each Ethernet frame before the
// decoder sees it.
//
// gopacket's Dot1Q layer passes its inner type field straight to
// EthernetType.LayerType(). The Ethernet layer treats a value below 0x0600 as an
// 802.3 length and continues into LLC, but Dot1Q has no such branch, so a tagged
// frame carrying LLC/SNAP — which is how CDP and EDP are framed — stops at Dot1Q
// with its payload undecoded. Removing the tag restores the untagged layout the
// Ethernet decoder already handles correctly, and keeps the source address that
// the neighbour records report.
//
// Only one tag is removed. QinQ (0x88a8, 0x9100) is out of scope: no product
// surface consumes a second tag, and the lab trunk carries single tags only.
type vlanStripper struct {
	capture.Handle
}

func (s vlanStripper) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	data, info, err := s.Handle.ReadPacketData()
	if err != nil {
		return data, info, err
	}

	if !isDot1QTagged(data) {
		return data, info, nil
	}

	stripped := make([]byte, 0, len(data)-dot1qTagLength)
	stripped = append(stripped, data[:ethernetTypeOffset]...)
	stripped = append(stripped, data[ethernetTypeOffset+dot1qTagLength:]...)

	info.CaptureLength -= dot1qTagLength
	info.Length -= dot1qTagLength
	return stripped, info, nil
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

	linkType := handle.LinkType()
	if linkType == layers.LinkTypeEthernet {
		return vlanStripper{Handle: handle}, linkType, nil
	}
	return handle, linkType, nil
}
