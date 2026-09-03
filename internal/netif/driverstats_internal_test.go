package netif

import "testing"

// Curate is the difference between a wall of counters and a diagnosis, so what
// it picks and what it omits both matter.
func TestCurateOmitsCountersTheDriverDoesNotExpose(t *testing.T) {
	t.Parallel()

	// A driver that reports only CRC errors. Everything else must be absent,
	// not present as zero: "this NIC does not count that" and "it counted
	// zero" are different answers, and showing 0 for the first is a lie an
	// operator would act on.
	got := Curate(map[string]uint64{"rx_crc_errors": 7})

	if len(got) != 1 {
		t.Fatalf("Curate returned %d counters, want 1: %+v", len(got), got)
	}
	if got[0].Key != "rx_crc_errors" || got[0].Value != 7 {
		t.Errorf("Curate returned %+v, want the CRC counter with value 7", got[0])
	}
	if got[0].Meaning == "" {
		t.Error("a counter with no meaning is a number the operator has to look up")
	}
}

// Driver names vary, so each counter lists the spellings seen in the wild.
func TestCurateMatchesAlternateDriverSpellings(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"rx_missed_errors", "rx_missed", "rx_no_buffer_count"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := Curate(map[string]uint64{name: 3})
			if len(got) != 1 {
				t.Fatalf("Curate(%q) returned %d counters, want 1", name, len(got))
			}
			if got[0].Label != "Receive drops" {
				t.Errorf("Curate(%q) labelled it %q, want %q", name, got[0].Label, "Receive drops")
			}
		})
	}
}

// A driver exposing two spellings of the same counter must yield one row, not
// two rows saying the same thing with different names.
func TestCurateTakesOneSpellingPerCounter(t *testing.T) {
	t.Parallel()

	got := Curate(map[string]uint64{"rx_missed_errors": 3, "rx_missed": 3})

	if len(got) != 1 {
		t.Errorf("Curate returned %d rows for one counter: %+v", len(got), got)
	}
}

// An empty map is a driver that exposes nothing, which must not become a row
// set full of zeroes.
func TestCurateOnADriverThatExposesNothing(t *testing.T) {
	t.Parallel()

	if got := Curate(map[string]uint64{}); len(got) != 0 {
		t.Errorf("Curate({}) returned %d counters, want none: %+v", len(got), got)
	}
}

// A paravirtualised NIC exposes none of the physical-layer counters -- no CRC
// errors, no collisions, no pause frames, because there is no wire. Probing a
// virtio interface on dev-srv-ubuntu matched zero of the original set out of
// the twenty counters it does expose. This pins the virtio names so the card
// says something useful there rather than "this driver exposes none of the
// counters worth watching".
func TestCurateHandlesAParavirtualisedNIC(t *testing.T) {
	t.Parallel()

	// The exact counter names `ethtool -S ens18` reports on virtio-net.
	virtio := map[string]uint64{
		"rx_drops": 0, "rx_xdp_packets": 0, "rx_xdp_tx": 0, "rx_xdp_redirects": 0,
		"rx_xdp_drops": 0, "rx_kicks": 105, "tx_xdp_tx": 0, "tx_xdp_tx_drops": 0,
		"tx_kicks": 443049, "tx_tx_timeouts": 0,
	}

	got := Curate(virtio)

	if len(got) == 0 {
		t.Fatal("Curate found nothing on a virtio NIC; the card would say the driver exposes nothing useful")
	}
	labels := map[string]bool{}
	for _, counter := range got {
		labels[counter.Label] = true
		if counter.Meaning == "" {
			t.Errorf("%q has no meaning", counter.Label)
		}
	}
	for _, want := range []string{"Receive drops (virtual)", "Transmit timeouts"} {
		if !labels[want] {
			t.Errorf("Curate did not surface %q from a virtio counter set", want)
		}
	}
}
