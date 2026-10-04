// Package flow implements a NetFlow v5, NetFlow v9 and IPFIX collector
// (RFC 3954, RFC 7011) as a passive-ingress listener.
//
// One UDP socket accepts all three formats; the version field of each
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
// read loop or growing the heap.
package flow

import (
	"net/netip"
	"time"
)

// Export format versions, as carried in the first two bytes of every
// datagram.
const (
	VersionNetFlow5 = 5
	VersionNetFlow9 = 9
	VersionIPFIX    = 10
)

// Record is one decoded flow, normalised across the three formats.
type Record struct {
	// Exporter is the address the datagram came from.
	Exporter netip.Addr
	// Version is the export format: 5, 9 or 10 (IPFIX).
	Version uint16
	// ObservationDomain is the v9 source ID or IPFIX observation domain
	// ID; for v5 it is the engine type and ID.
	ObservationDomain uint32

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
