package wificapture_test

import (
	"io"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/krisarmstrong/seed/internal/capture"
	wificapture "github.com/krisarmstrong/seed/internal/wifi/capture"
	"github.com/krisarmstrong/seed/internal/wifi/dot11"
)

// --- fakes ----------------------------------------------------------------

type fakeHandle struct {
	mu       sync.Mutex
	frames   [][]byte
	idx      int
	linkType layers.LinkType
	closed   bool
}

func (h *fakeHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || h.idx >= len(h.frames) {
		return nil, gopacket.CaptureInfo{}, io.EOF
	}
	d := h.frames[h.idx]
	h.idx++
	return d, gopacket.CaptureInfo{}, nil
}

func (h *fakeHandle) SetBPFFilter(string) error { return nil }
func (h *fakeHandle) LinkType() layers.LinkType { return h.linkType }
func (h *fakeHandle) Close()                    { h.mu.Lock(); h.closed = true; h.mu.Unlock() }

type fakeOpener struct {
	handle capture.Handle
	err    error
	calls  int
}

func (o *fakeOpener) OpenLive(string, int32, bool, time.Duration) (capture.Handle, error) {
	o.calls++
	return o.handle, o.err
}

type fakeSink struct {
	mu       sync.Mutex
	frames   []*dot11.Frame
	setCalls int
	source   string
	cleared  bool
}

func (s *fakeSink) Ingest(f *dot11.Frame, _ time.Time) {
	s.mu.Lock()
	s.frames = append(s.frames, f)
	s.mu.Unlock()
}
func (s *fakeSink) SetSource(n string) { s.mu.Lock(); s.setCalls++; s.source = n; s.mu.Unlock() }
func (s *fakeSink) ClearSource()       { s.mu.Lock(); s.cleared = true; s.source = ""; s.mu.Unlock() }

func (s *fakeSink) frameCount() int { s.mu.Lock(); defer s.mu.Unlock(); return len(s.frames) }

// beaconBytes builds radiotap + an 802.11 beacon advertising ssid (mirrors the
// frame format the dot11 decoder is tested against).
func beaconBytes(ssid string) []byte {
	bssid := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	frame := []byte{0x80, 0x00, 0x00, 0x00} // FC mgmt/beacon + duration
	frame = append(frame, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)
	frame = append(frame, bssid...)
	frame = append(frame, bssid...)
	frame = append(frame, 0x00, 0x00)         // sequence control
	frame = append(frame, make([]byte, 8)...) // timestamp
	frame = append(frame, 0x64, 0x00)         // beacon interval
	frame = append(frame, 0x01, 0x00)         // capability (ESS)
	frame = append(frame, 0x00, byte(len(ssid)))
	frame = append(frame, []byte(ssid)...) // SSID IE
	frame = append(frame, 0xde, 0xad, 0xbe, 0xef)
	radiotap := []byte{0x00, 0x00, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00}
	return append(radiotap, frame...)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("condition not met within deadline")
}

// --- tests ----------------------------------------------------------------

func TestCaptureIngestsDecodedFramesAndSkipsGarbage(t *testing.T) {
	h := &fakeHandle{
		frames:   [][]byte{beaconBytes("corp"), {0x01, 0x02, 0x03}, beaconBytes("guest")},
		linkType: layers.LinkTypeIEEE80211Radio,
	}
	sink := &fakeSink{}
	c := wificapture.New(&fakeOpener{handle: h}, sink, "mon0")

	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitFor(t, func() bool { return sink.frameCount() == 2 })
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if sink.frameCount() != 2 {
		t.Errorf("ingested %d frames, want 2 (2 beacons, 1 garbage skipped)", sink.frameCount())
	}
	if sink.setCalls == 0 || !sink.cleared {
		t.Errorf("expected SetSource on start and ClearSource on stop: set=%d cleared=%v", sink.setCalls, sink.cleared)
	}
}

func TestCaptureRejectsNonMonitorInterface(t *testing.T) {
	h := &fakeHandle{frames: [][]byte{beaconBytes("corp")}, linkType: layers.LinkTypeEthernet}
	sink := &fakeSink{}
	c := wificapture.New(&fakeOpener{handle: h}, sink, "eth0")

	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = c.Stop() })

	if sink.frameCount() != 0 {
		t.Error("non-monitor interface must not ingest frames")
	}
	if sink.setCalls != 0 {
		t.Error("non-monitor interface must not set a capture source")
	}
	h.mu.Lock()
	closed := h.closed
	h.mu.Unlock()
	if !closed {
		t.Error("the non-monitor handle should be closed")
	}
}

func TestCaptureGracefulWhenOpenFails(t *testing.T) {
	sink := &fakeSink{}
	c := wificapture.New(&fakeOpener{err: io.ErrUnexpectedEOF}, sink, "mon0")
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("Start should degrade gracefully, got error: %v", err)
	}
	if sink.setCalls != 0 || sink.frameCount() != 0 {
		t.Error("a failed open must not set a source or ingest frames")
	}
}

func TestCaptureDisabledWithoutInterface(t *testing.T) {
	op := &fakeOpener{}
	c := wificapture.New(op, &fakeSink{}, "")
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if op.calls != 0 {
		t.Error("an empty interface must not attempt to open a handle")
	}
}

func TestOptionsApplied(t *testing.T) {
	h := &fakeHandle{frames: [][]byte{beaconBytes("corp")}, linkType: layers.LinkTypeIEEE80211Radio}
	sink := &fakeSink{}
	fixed := time.Unix(1700000000, 0).UTC()
	c := wificapture.New(&fakeOpener{handle: h}, sink, "mon0",
		wificapture.WithClock(func() time.Time { return fixed }),
		wificapture.WithLogger(slog.New(slog.DiscardHandler)),
		wificapture.WithSnapLen(2048),
	)
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	waitFor(t, func() bool { return sink.frameCount() == 1 })
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestStopIdempotent(t *testing.T) {
	h := &fakeHandle{frames: [][]byte{beaconBytes("corp")}, linkType: layers.LinkTypeIEEE80211Radio}
	c := wificapture.New(&fakeOpener{handle: h}, &fakeSink{}, "mon0")
	if err := c.Start(t.Context()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := c.Stop(); err != nil {
		t.Fatalf("second Stop: %v", err)
	}
}
