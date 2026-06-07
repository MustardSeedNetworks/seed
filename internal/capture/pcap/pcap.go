//go:build cgo || windows

// Package pcap is the libpcap-backed adapter for the capture port.
//
// The build constraint (cgo || windows) keeps this package out of CGO_ENABLED=0
// builds on non-Windows platforms, where libpcap cannot be linked — there the
// composition root selects the CGO-free no-op adapter (nullcapture) instead, and
// a CGO_ENABLED=0 `go build ./...` skips this package cleanly. On Windows
// gopacket/pcap is CGO-free (wpcap), so live capture is retained there.
//
// It is the ONLY package in seed that imports github.com/gopacket/gopacket/pcap,
// and therefore the only CGO-tainted (libpcap-linked) package. A depguard rule
// confines gopacket/pcap to this package so the taint cannot leak back into the
// domain. See docs/architecture/CGO_BUILD_STRATEGY.md.
package pcap

import (
	"time"

	"github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// Compile-time guarantees: this adapter implements the port, and *pcap.Handle
// satisfies capture.Handle directly (ReadPacketData/SetBPFFilter/LinkType/Close),
// so OpenLive returns it without a wrapper type.
var (
	_ capture.Opener = Opener{}
	_ capture.Handle = (*pcap.Handle)(nil)
)

// Opener is the libpcap-backed capture.Opener.
type Opener struct{}

// New returns a libpcap-backed capture.Opener.
func New() Opener { return Opener{} }

// OpenLive opens a live libpcap capture handle.
//
// *pcap.Handle already satisfies capture.Handle — it has ReadPacketData,
// SetBPFFilter, LinkType, and Close with matching signatures — so the returned
// handle needs no wrapping.
func (Opener) OpenLive(
	iface string,
	snaplen int32,
	promiscuous bool,
	timeout time.Duration,
) (capture.Handle, error) {
	handle, err := pcap.OpenLive(iface, snaplen, promiscuous, timeout)
	if err != nil {
		// Return an explicit nil interface (not a typed-nil *pcap.Handle) so
		// callers' `handle == nil` checks behave. Callers add feature context.
		return nil, err
	}
	return handle, nil
}
