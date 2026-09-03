package netif

// driverstats.go exposes the NIC driver's own statistics — the counters that
// say *why* a link is unhealthy rather than that it is (#416).
//
// A link can be up, negotiated at the right speed, and still be dropping
// frames. rx_crc_errors points at cabling; rx_missed_errors at a host that
// cannot keep up; rx_pause_frames at congestion downstream. None of that is
// visible from an interface listing.
//
// Linux only. ethtool is a Linux ioctl interface and neither macOS nor Windows
// has an equivalent, which is why this is a capability rather than a feature
// that silently returns nothing (#749, #750).

import "errors"

// ErrDriverStatsUnsupported is returned on platforms with no ethtool equivalent.
var ErrDriverStatsUnsupported = errors.New("driver statistics are not available on this platform")

// CuratedCounter is one counter worth putting in front of an operator, with the
// reason it matters.
//
// The driver exposes hundreds; a wall of them is not diagnosis. These are the
// ones that change what you do next.
type CuratedCounter struct {
	// Key is the driver's own name for it, kept so an operator can correlate
	// with ethtool -S output.
	Key string `json:"key"`
	// Label is what the UI shows.
	Label string `json:"label"`
	// Value is the count since the interface came up.
	Value uint64 `json:"value"`
	// Meaning says what a rising count indicates. Written here rather than in
	// the UI so the API, the card and any export agree.
	Meaning string `json:"meaning"`
}

// curated is the counter set and the reason each one is in it.
//
// Driver names vary, so each entry lists the spellings seen in the wild and the
// first match wins. The list spans both physical NICs (CRC, collisions, pause
// frames -- all meaningless without a wire) and paravirtualised ones, because a
// curation that only knows about physical hardware returns nothing at all on a
// VM, which is most of where Seed runs during development.
func curated() []struct {
	Names   []string
	Label   string
	Meaning string
} {
	return []struct {
		Names   []string
		Label   string
		Meaning string
	}{
		{
			Names:   []string{"rx_crc_errors", "rx_crc_error", "crc_errors"},
			Label:   "CRC errors",
			Meaning: "Frames arrived corrupted. Usually cabling, a bad port, or a duplex mismatch — not congestion.",
		},
		{
			Names:   []string{"rx_missed_errors", "rx_missed", "rx_no_buffer_count"},
			Label:   "Receive drops",
			Meaning: "Frames arrived and the host could not take them. Points at the receiver, not the network.",
		},
		{
			Names:   []string{"collisions", "tx_collisions"},
			Label:   "TX collisions",
			Meaning: "On a modern full-duplex link this should stay at zero. Any value suggests a duplex mismatch.",
		},
		{
			Names:   []string{"rx_pause_frames", "rx_pause", "rx_flow_control_xon"},
			Label:   "Pause frames received",
			Meaning: "Something downstream asked this link to slow down. Congestion beyond this host.",
		},
		{
			Names:   []string{"rx_length_errors", "rx_length_error"},
			Label:   "Length errors",
			Meaning: "Runts or giants. Often an MTU mismatch or a failing transceiver.",
		},
		{
			Names:   []string{"rx_over_errors", "rx_fifo_errors", "rx_fifo_error"},
			Label:   "Receive overruns",
			Meaning: "The NIC's own buffer overflowed before the host drained it.",
		},
		// Paravirtualised NICs report none of the counters above -- a virtio
		// interface has no notion of a CRC error or a collision, because there
		// is no wire. Probing one on dev-srv-ubuntu matched zero of the
		// physical set out of the twenty counters it does expose, which is
		// accurate and useless. These two are its equivalents.
		{
			Names:   []string{"rx_drops", "rx_dropped"},
			Label:   "Receive drops (virtual)",
			Meaning: "The virtual NIC discarded inbound frames, usually because the guest could not keep up with the host.",
		},
		{
			Names:   []string{"tx_tx_timeouts", "tx_timeouts", "tx_timeout_count"},
			Label:   "Transmit timeouts",
			Meaning: "The driver gave up waiting for a transmit to complete and reset the queue. On a virtual NIC this points at the hypervisor or a starved host.",
		},
	}
}

// Curate picks the counters worth showing from everything the driver reports.
//
// A counter the driver does not expose is omitted rather than reported as zero:
// "this NIC does not count that" and "it counted zero" are different answers,
// and showing 0 for the first is a lie an operator would act on.
func Curate(raw map[string]uint64) []CuratedCounter {
	out := make([]CuratedCounter, 0, len(curated()))
	for _, entry := range curated() {
		for _, name := range entry.Names {
			value, ok := raw[name]
			if !ok {
				continue
			}
			out = append(out, CuratedCounter{
				Key:     name,
				Label:   entry.Label,
				Value:   value,
				Meaning: entry.Meaning,
			})

			break
		}
	}

	return out
}
