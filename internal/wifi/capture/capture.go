// Package wificapture is the monitor-mode 802.11 capture producer for the Wi-Fi
// visibility feature. It opens a live-capture handle on a monitor-capable
// interface (via the CGO-confined internal/capture port), reads radiotap frames,
// decodes them with internal/wifi/dot11, and feeds the decoded frames to a sink
// (the internal/wifi/visibility service).
//
// The package is CGO-free: the libpcap binding is injected as a capture.Opener
// (DefaultOpener selects the real adapter on cgo/windows builds and a no-op stub
// otherwise), so the read→decode→ingest loop is exercised with a fake handle in
// tests. Capture degrades gracefully — a missing interface, an open failure, or a
// non-monitor (non-radiotap) link type disables capture with a log line rather
// than an error, leaving the visibility endpoints to serve an empty airspace.
package wificapture

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/wifi/dot11"
)

// defaultSnapLen is the per-packet capture length. Radiotap + a full management
// frame with its information elements fits comfortably; data frames are captured
// only far enough to attribute a client to its BSSID.
const defaultSnapLen = 65535

// Sink receives decoded frames and capture-source lifecycle signals. The
// visibility service satisfies it.
type Sink interface {
	Ingest(f *dot11.Frame, at time.Time)
	SetSource(name string)
	ClearSource()
}

// Capture owns a monitor-mode read loop feeding a Sink. Lifecycle (Start/Stop)
// mirrors the other background components; all methods are concurrency-safe.
type Capture struct {
	opener  capture.Opener
	sink    Sink
	iface   string
	snapLen int32
	enabler Enabler
	now     func() time.Time
	log     *slog.Logger

	mu      sync.Mutex
	handle  capture.Handle
	cancel  context.CancelFunc
	restore func() error
	wg      sync.WaitGroup
}

// Option configures a Capture.
type Option func(*Capture)

// WithClock overrides the timestamp source (for tests).
func WithClock(now func() time.Time) Option {
	return func(c *Capture) {
		if now != nil {
			c.now = now
		}
	}
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) Option {
	return func(c *Capture) {
		if l != nil {
			c.log = l
		}
	}
}

// WithSnapLen overrides the per-packet capture length.
func WithSnapLen(n int32) Option {
	return func(c *Capture) {
		if n > 0 {
			c.snapLen = n
		}
	}
}

// WithEnabler installs an Enabler that switches the interface into monitor mode
// before capture and restores it afterwards. Without one, capture is
// bring-your-own monitor (the interface must already be in monitor mode).
func WithEnabler(e Enabler) Option {
	return func(c *Capture) {
		if e != nil {
			c.enabler = e
		}
	}
}

// New builds a capture bound to opener, feeding sink from iface (the monitor
// interface name; empty disables capture).
func New(opener capture.Opener, sink Sink, iface string, opts ...Option) *Capture {
	c := &Capture{
		opener:  opener,
		sink:    sink,
		iface:   iface,
		snapLen: defaultSnapLen,
		now:     time.Now,
		log:     slog.Default(),
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Start opens the capture handle and launches the read loop. It returns nil and
// disables capture (with a log line) when no interface is configured, the handle
// cannot be opened, or the interface is not in monitor mode — these are normal
// degraded states, not startup failures. Idempotent: a second call while running
// is a no-op.
func (c *Capture) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return nil
	}
	if c.iface == "" {
		c.log.InfoContext(ctx, "wifi capture disabled: no monitor interface configured")
		return nil
	}

	// Optionally switch the interface into monitor mode. A failure is not fatal —
	// the interface may already be in monitor mode (bring-your-own), and the
	// link-type guard below catches the case where it is not.
	var restore func() error
	if c.enabler != nil {
		r, enErr := c.enabler.Enable(ctx, c.iface)
		if enErr != nil {
			c.log.WarnContext(ctx, "monitor-mode enable failed; trying interface as-is",
				"iface", c.iface, "error", enErr)
		} else {
			restore = r
		}
	}

	handle, err := c.opener.OpenLive(c.iface, c.snapLen, true, capture.BlockForever)
	if err != nil {
		runRestore(restore)
		c.log.WarnContext(ctx, "wifi capture unavailable: cannot open interface",
			"iface", c.iface, "error", err)
		return nil
	}
	if lt := handle.LinkType(); lt != layers.LinkTypeIEEE80211Radio {
		handle.Close()
		runRestore(restore)
		c.log.WarnContext(ctx, "wifi capture unavailable: interface is not in monitor mode (radiotap required)",
			"iface", c.iface, "linkType", lt.String())
		return nil
	}

	loopCtx, cancel := context.WithCancel(ctx)
	c.cancel = cancel
	c.handle = handle
	c.restore = restore
	c.sink.SetSource(c.iface)
	c.wg.Add(1)
	go c.loop(loopCtx, handle)
	c.log.InfoContext(ctx, "wifi monitor capture started", "iface", c.iface)
	return nil
}

// runRestore invokes a restore function if present, ignoring its error (best
// effort during a degraded/abort path).
func runRestore(restore func() error) {
	if restore != nil {
		_ = restore()
	}
}

// loop reads frames until the handle is closed (by Stop) or a read error occurs.
// Undecodable frames (non-802.11 / malformed) are skipped, never fatal.
func (c *Capture) loop(ctx context.Context, handle capture.Handle) {
	defer c.wg.Done()
	for {
		if ctx.Err() != nil {
			return
		}
		data, _, err := handle.ReadPacketData()
		if err != nil {
			// Closed handle (Stop) or a fatal read error ends the loop.
			return
		}
		frame, derr := dot11.Decode(data)
		if derr != nil {
			continue
		}
		c.sink.Ingest(frame, c.now())
	}
}

// Stop ends the read loop, closes the handle, and clears the capture source.
// Idempotent.
func (c *Capture) Stop() error {
	c.mu.Lock()
	cancel := c.cancel
	handle := c.handle
	restore := c.restore
	c.cancel = nil
	c.handle = nil
	c.restore = nil
	c.mu.Unlock()

	if cancel == nil {
		return nil // not running
	}
	cancel()
	if handle != nil {
		handle.Close() // unblocks a blocked ReadPacketData
	}
	c.wg.Wait()
	c.sink.ClearSource()
	runRestore(restore) // revert the interface to its prior mode
	return nil
}
