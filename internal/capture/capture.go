// Package capture defines the live packet-capture port used by the discovery,
// dhcp, and vlan features.
//
// The port exists to confine CGO. Live capture is backed by libpcap, pulled in
// transitively by github.com/gopacket/gopacket/pcap (its pcap_unix.go /
// pcap_darwin.go carry "#cgo LDFLAGS: -lpcap"). Importing that package taints a
// build with CGO and re-links libpcap into every dependent test binary. By
// routing capture through this interface, exactly one package imports
// gopacket/pcap — the libpcap-backed adapter in internal/capture/pcap — so it is
// the only CGO-tainted package. internal/capture/nullcapture is the CGO-free
// stub used for CGO_ENABLED=0 builds (e.g. Windows) and pure-Go test runs.
//
// Only github.com/gopacket/gopacket/pcap carries the CGO directive; the gopacket
// core and gopacket/layers are pure Go, so this port may name their types
// (PacketDataSource, CaptureInfo, layers.LinkType) without taking on the taint.
// See docs/architecture/CGO_BUILD_STRATEGY.md.
package capture

import (
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// BlockForever is the OpenLive timeout that makes packet reads block until a
// packet arrives instead of returning on a timeout. It mirrors the value of
// pcap.BlockForever so the libpcap adapter can pass it straight through; the
// pcap adapter's test asserts the two stay equal so a gopacket upgrade that
// changed the sentinel cannot silently break block-forever semantics here.
const BlockForever = -10 * time.Millisecond

// Handle is an open live-capture handle. It embeds gopacket.PacketDataSource so a
// handle can be passed straight to
// gopacket.NewPacketSource(handle, handle.LinkType()).
type Handle interface {
	gopacket.PacketDataSource

	// SetBPFFilter installs a Berkeley Packet Filter expression, restricting the
	// frames delivered by the handle.
	SetBPFFilter(filter string) error

	// LinkType reports the link-layer type of the handle, used as the gopacket
	// decoder when building a packet source.
	LinkType() layers.LinkType

	// Close releases the handle and any associated OS resources.
	Close()
}

// Opener opens live-capture handles on a named network interface. Implementations
// are the libpcap adapter (internal/capture/pcap) and the no-op stub
// (internal/capture/nullcapture).
type Opener interface {
	// OpenLive opens iface for live capture. snaplen is the per-packet capture
	// length in bytes; promiscuous toggles promiscuous mode; timeout is the read
	// timeout — pass BlockForever to block until a packet arrives.
	OpenLive(iface string, snaplen int32, promiscuous bool, timeout time.Duration) (Handle, error)
}
