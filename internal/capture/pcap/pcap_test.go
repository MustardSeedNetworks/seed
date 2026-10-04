//go:build cgo || windows

package pcap_test

import (
	"errors"
	"io"
	"testing"
	"time"

	gpcap "github.com/gopacket/gopacket/pcap"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/capture/pcap"
)

// A handle without a read timeout cannot be closed on a quiet interface
// (#2862), so the adapter refuses to open one, gopacket's BlockForever included.
func TestOpenLiveRefusesNoReadTimeout(t *testing.T) {
	t.Parallel()

	for _, timeout := range []time.Duration{0, gpcap.BlockForever, -time.Second} {
		handle, err := pcap.New().OpenLive("lo", 65535, false, timeout)
		if handle != nil || !errors.Is(err, capture.ErrNoReadTimeout) {
			t.Errorf("OpenLive(timeout %v) = %v, %v; want nil, ErrNoReadTimeout", timeout, handle, err)
		}
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
