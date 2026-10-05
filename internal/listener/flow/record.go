// Package flow implements a NetFlow v5, NetFlow v9, IPFIX (RFC 3954,
// RFC 7011) and sFlow v5 collector as a passive-ingress listener.
//
// One UDP socket accepts all four formats; the version field of each
// datagram picks the decoder. v9 and IPFIX data sets are decoded against
// templates learned from the same exporter, keyed by exporter address,
// observation domain and template ID. A data set whose template has not
// arrived yet, or has expired, is dropped and counted rather than held:
// exporters resend templates on a timer, so the flows after the next
// template decode normally and memory stays bounded. Byte and packet counts
// are scaled by the sampling rate the exporter announces, so they estimate
// the traffic observed rather than the packets sampled.
//
// Flows do not go through [listener.Sink]. One datagram carries up to
// dozens of flows and an exporter sends thousands of datagrams a minute,
// so the listener batches decoded records into a [Store] behind a bounded
// queue. A full queue drops and counts records instead of blocking the
// read loop or growing the heap. The exception is a flow to or from an
// address on the operator's threat indicator list: each batch's matches are
// published to the sink as [listener.FlowIndicatorKind] events, so the alert
// pipeline raises them.
package flow

import (
	"net/netip"
	"time"
)

// Export format versions, as carried in the first two bytes of every
// NetFlow and IPFIX datagram. sFlow carries its version in four bytes,
// so its datagrams open with two zero bytes.
const (
	VersionNetFlow5 = 5
	VersionNetFlow9 = 9
	VersionIPFIX    = 10
	VersionSFlow5   = 5
)

// Format is the export protocol a record arrived in. sFlow v5 and
// NetFlow v5 share a version number, so the version alone cannot say.
type Format string

// Formats, as stored in flow_records.format.
const (
	FormatNetFlow5 Format = "netflow5"
	FormatNetFlow9 Format = "netflow9"
	FormatIPFIX    Format = "ipfix"
	FormatSFlow5   Format = "sflow5"
)

// Record is one decoded flow, normalised across the three formats.
type Record struct {
	// Exporter is the address the datagram came from; for sFlow it is
	// the agent address the datagram names.
	Exporter netip.Addr
	Format   Format
	// ObservationDomain is the v9 source ID or IPFIX observation domain
	// ID; for NetFlow v5 it is the engine type and ID, and for sFlow the
	// sub-agent ID.
	ObservationDomain uint32

	// Start and End bound the flow. An sFlow record is one sampled
	// packet, so both are the time it was received.
	Start time.Time
	End   time.Time

	SrcAddr  netip.Addr
	DstAddr  netip.Addr
	SrcPort  uint16
	DstPort  uint16
	Protocol uint8
	TCPFlags uint8

	Bytes   uint64
	Packets uint64

	// InputIf and OutputIf are the exporter's ifIndex values.
	InputIf  uint32
	OutputIf uint32
}
