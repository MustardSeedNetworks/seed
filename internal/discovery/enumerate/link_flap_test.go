package enumerate_test

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/discovery/enumerate"
)

// countingOpener records how many capture handles were opened and how many were
// closed, which is what a restart costs: every reopen is a window during which
// the interface is not being watched.
type countingOpener struct {
	mu      sync.Mutex
	opened  int
	handles []*countingHandle
}

func (o *countingOpener) OpenLive(string, int32, bool, time.Duration) (capture.Handle, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	handle := &countingHandle{}
	o.opened++
	o.handles = append(o.handles, handle)
	return handle, nil
}

func (o *countingOpener) counts() (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()

	closed := 0
	for _, handle := range o.handles {
		if handle.isClosed() {
			closed++
		}
	}
	return o.opened, closed
}

// countingHandle is a capture handle that never yields a packet -- these tests
// are about the handle's lifecycle, not about decoding.
type countingHandle struct {
	mu     sync.Mutex
	closed bool
}

func (h *countingHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	// Yield rather than spin; the capture loop exits on its own context.
	time.Sleep(10 * time.Millisecond)
	return nil, gopacket.CaptureInfo{}, io.EOF
}

func (h *countingHandle) SetBPFFilter(string) error { return nil }
func (h *countingHandle) LinkType() layers.LinkType { return layers.LinkTypeEthernet }

func (h *countingHandle) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
}

func (h *countingHandle) isClosed() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.closed
}

// TestRestartingProtocolCaptureReopensEveryHandle is the measurement behind
// #323. A switch advertises LLDP the instant a port comes up; restarting the
// captures on a link-up event closes the handles at exactly that moment, and a
// missed advertisement is not resent for a full transmit interval -- 30 seconds
// for LLDP, 60 for CDP.
//
// The test does not assert that restarting is wrong. It asserts what restarting
// does, so that the reason onLinkStateChange stopped doing it is written down
// somewhere that fails if the behaviour changes.
func TestRestartingProtocolCaptureReopensEveryHandle(t *testing.T) {
	t.Parallel()

	opener := &countingOpener{}
	manager := enumerate.NewManager("eth0", enumerate.WithCapture(opener))

	if err := manager.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	firstOpen, firstClosed := opener.counts()
	if firstOpen == 0 {
		t.Fatal("starting the manager opened no capture handles")
	}
	if firstClosed != 0 {
		t.Errorf("%d handles were closed by a plain Start", firstClosed)
	}

	manager.Stop()
	_, afterStop := opener.counts()
	if afterStop != firstOpen {
		t.Errorf("Stop closed %d of %d handles; a partial stop leaks a capture", afterStop, firstOpen)
	}

	if err := manager.Start(); err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(manager.Stop)

	secondOpen, _ := opener.counts()
	if secondOpen != firstOpen*2 {
		t.Errorf("restart opened %d handles in total, want %d -- every protocol reopens", secondOpen, firstOpen*2)
	}
}

// A manager that is left alone keeps the handles it opened. This is what the
// link-up path now relies on: a pcap handle stays valid across a carrier
// transition and simply delivers nothing while the link is down, so there is
// nothing to restart.
func TestUndisturbedProtocolCaptureKeepsItsHandles(t *testing.T) {
	t.Parallel()

	opener := &countingOpener{}
	manager := enumerate.NewManager("eth0", enumerate.WithCapture(opener))

	if err := manager.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(manager.Stop)

	opened, _ := opener.counts()
	time.Sleep(100 * time.Millisecond)

	stillOpened, closed := opener.counts()
	if stillOpened != opened {
		t.Errorf("handles grew from %d to %d with no restart", opened, stillOpened)
	}
	if closed != 0 {
		t.Errorf("%d handles closed themselves without a Stop", closed)
	}
}

// Start is documented as idempotent, and the link-up path depends on that:
// anything that calls Start on an already-running manager must not silently
// reopen the handles and reintroduce the gap.
func TestStartingAnAlreadyRunningManagerOpensNothing(t *testing.T) {
	t.Parallel()

	opener := &countingOpener{}
	manager := enumerate.NewManager("eth0", enumerate.WithCapture(opener))

	if err := manager.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(manager.Stop)
	opened, _ := opener.counts()

	if err := manager.Start(); err != nil {
		t.Fatalf("second Start: %v", err)
	}

	if again, _ := opener.counts(); again != opened {
		t.Errorf("a redundant Start opened %d more handles", again-opened)
	}
}
