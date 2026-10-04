// Package capturetest provides a capture.Opener that behaves like libpcap on a
// quiet Linux interface, for testing that a capture can always be stopped.
package capturetest

import (
	"io"
	"sync"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// QuietOpener opens handles on an interface that carries only Frames. Each
// handle first times out once, then yields Frames in order, then stays silent.
//
// A read holds the handle the way libpcap on Linux does, and Close waits for
// it. With a positive timeout the read returns capture.ErrTimeout when it
// expires, so Close gets its turn. With a non-positive timeout a read on the
// silent interface never returns, and neither does Close: the hang a capture
// opened that way has on real hardware.
type QuietOpener struct {
	LinkType layers.LinkType
	Frames   [][]byte

	mu      sync.Mutex
	timeout time.Duration
	opened  bool
}

// OpenLive records timeout and returns a new handle.
func (o *QuietOpener) OpenLive(_ string, _ int32, _ bool, timeout time.Duration) (capture.Handle, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.timeout = timeout
	o.opened = true
	return &quietHandle{linkType: o.LinkType, frames: o.Frames, timeout: timeout}, nil
}

// Timeout reports the read timeout of the last handle opened, and whether one
// was opened at all.
func (o *QuietOpener) Timeout() (time.Duration, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.timeout, o.opened
}

type quietHandle struct {
	mu       sync.Mutex
	linkType layers.LinkType
	frames   [][]byte
	timeout  time.Duration
	timedOut bool
	closed   bool
}

func (h *quietHandle) ReadPacketData() ([]byte, gopacket.CaptureInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return nil, gopacket.CaptureInfo{}, io.EOF
	}
	if h.timedOut && len(h.frames) > 0 {
		frame := h.frames[0]
		h.frames = h.frames[1:]
		return frame, gopacket.CaptureInfo{Timestamp: time.Now()}, nil
	}
	if h.timeout <= 0 {
		select {} // libpcap waits for a frame that never comes
	}
	time.Sleep(h.timeout)
	h.timedOut = true
	return nil, gopacket.CaptureInfo{}, capture.ErrTimeout
}

func (h *quietHandle) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.closed = true
}

func (h *quietHandle) SetBPFFilter(string) error    { return nil }
func (h *quietHandle) LinkType() layers.LinkType    { return h.linkType }
func (h *quietHandle) WritePacketData([]byte) error { return nil }
