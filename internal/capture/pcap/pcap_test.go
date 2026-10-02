//go:build cgo || windows

package pcap_test

import (
	"errors"
	"io"
	"testing"

	gpcap "github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/capture/pcap"
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

// A timed-out read is the port's ErrTimeout, and nothing else is: a caller
// that reads past ErrTimeout must still stop on a closed or failed handle.
func TestPortErrorNamesOnlyTheTimeout(t *testing.T) {
	t.Parallel()

	if err := pcap.PortError(gpcap.NextErrorTimeoutExpired); !errors.Is(err, capture.ErrTimeout) {
		t.Fatalf("timeout = %v, want capture.ErrTimeout", err)
	}
	for _, err := range []error{io.EOF, gpcap.NextErrorReadError, nil} {
		if got := pcap.PortError(err); !errors.Is(got, err) || errors.Is(got, capture.ErrTimeout) {
			t.Errorf("PortError(%v) = %v, want it unchanged", err, got)
		}
	}
}
