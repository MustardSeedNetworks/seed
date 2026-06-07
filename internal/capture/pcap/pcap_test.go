//go:build cgo || windows

package pcap_test

import (
	"testing"

	gpcap "github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// TestBlockForeverMatchesPcap guards against drift: capture.BlockForever is
// defined CGO-free (the port may not import gopacket/pcap) by mirroring
// pcap.BlockForever's value. The adapter passes the timeout straight through, so
// if a gopacket upgrade changed the sentinel, callers using capture.BlockForever
// would silently lose block-forever semantics. This test is the only place that
// can compare the two values.
func TestBlockForeverMatchesPcap(t *testing.T) {
	t.Parallel()

	if capture.BlockForever != gpcap.BlockForever {
		t.Fatalf(
			"capture.BlockForever (%v) != pcap.BlockForever (%v): update the constant in internal/capture",
			capture.BlockForever, gpcap.BlockForever,
		)
	}
}
