package discovery

import (
	"fmt"

	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/seed/internal/capture"
)

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
