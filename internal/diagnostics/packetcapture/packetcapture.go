// Package packetcapture records live traffic on one interface to a pcap file
// an operator downloads and opens in Wireshark (#326). A capture is bounded
// three ways — its duration, a file-size ceiling, and the caller's context,
// which is how an operator stops it early — and a stopped capture keeps what
// it recorded. Frames come through the capture port (internal/capture), so
// this package stays CGO-free; the file is written with gopacket/pcapgo,
// which is pure Go.
package packetcapture

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/gopacket/gopacket/pcapgo"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

const (
	// DefaultDuration is how long a capture runs when the request names no
	// duration.
	DefaultDuration = time.Minute
	// MaxDuration is the longest capture a request may ask for.
	MaxDuration = time.Hour

	// MaxFileBytes is the size a capture file stops growing at. With
	// keepFinished it bounds what captures can take of the data directory.
	MaxFileBytes = 64 << 20

	// snaplen keeps whole frames, jumbo ones included: tcpdump's default.
	snaplen = 262144

	// readTimeout bounds each read so a quiet interface still sees the stop
	// and the deadline promptly, and so closing the handle never waits on a
	// read (see capture.ErrTimeout).
	readTimeout = 100 * time.Millisecond

	// drainWindow bounds reading what the kernel buffered before a stop:
	// two read timeouts, since a frame is handed over within one.
	drainWindow = 2 * readTimeout

	// progressInterval throttles progress reports on a busy link.
	progressInterval = 250 * time.Millisecond

	// pcap framing: the file header, then a header before every record.
	fileHeaderBytes   = 24
	recordHeaderBytes = 16
)

// ErrInterface rejects a request that names no interface.
var ErrInterface = errors.New("interface is required")

// StopReason says why a capture ended.
type StopReason string

// The ways a capture ends with a file.
const (
	StopDuration StopReason = "duration" // it ran for the requested duration
	StopSize     StopReason = "size"     // the next frame would pass MaxFileBytes
	StopStopped  StopReason = "stopped"  // the operator stopped it
)

// Request is one capture as a caller asks for it.
type Request struct {
	Interface string `json:"interface"`
	// Filter is a libpcap filter expression, such as "host 10.0.0.1 and
	// port 443"; every frame when empty.
	Filter string `json:"filter,omitempty"`
	// DurationSeconds is how long to capture, at most an hour; a minute when
	// zero.
	DurationSeconds int `json:"durationSeconds,omitempty"`
}

// Result describes a finished capture file.
type Result struct {
	// ID names the file: GET /api/v1/captures/{id} downloads it.
	ID         string     `json:"id"`
	Interface  string     `json:"interface"`
	Filter     string     `json:"filter,omitempty"`
	Packets    int        `json:"packets"`
	Bytes      int64      `json:"bytes"`
	DurationMs int64      `json:"durationMs"`
	StopReason StopReason `json:"stopReason"       jsonschema:"enum=duration,enum=size,enum=stopped"`
}

// Run captures on req.Interface through opener into a new file in store.
// report receives the fraction of the duration elapsed.
func Run(
	ctx context.Context,
	opener capture.Opener,
	store *Store,
	req Request,
	report func(float64),
) (*Result, error) {
	s := session{opener: opener, store: store, now: time.Now, maxBytes: MaxFileBytes}
	return s.run(ctx, req, report)
}

// session is a capture's dependencies. now and maxBytes are seams so the
// duration and size bounds are testable without waiting out an hour or
// writing 64 MiB.
type session struct {
	opener   capture.Opener
	store    *Store
	now      func() time.Time
	maxBytes int64
}

func (s session) run(ctx context.Context, req Request, report func(float64)) (*Result, error) {
	duration, err := validate(req)
	if err != nil {
		return nil, err
	}
	handle, err := s.opener.OpenLive(req.Interface, snaplen, true, readTimeout)
	if err != nil {
		return nil, fmt.Errorf("open %s for capture: %w", req.Interface, err)
	}
	defer handle.Close()
	if req.Filter != "" {
		if filterErr := handle.SetBPFFilter(req.Filter); filterErr != nil {
			return nil, fmt.Errorf("invalid capture filter %q: %w", req.Filter, filterErr)
		}
	}

	id, f, err := s.store.create()
	if err != nil {
		return nil, err
	}
	res, err := s.record(ctx, handle, f, duration, report)
	closeErr := f.Close()
	if err == nil && closeErr != nil {
		err = fmt.Errorf("close capture file: %w", closeErr)
	}
	if err != nil {
		s.store.remove(id)
		return nil, err
	}
	s.store.release(id)
	res.ID = id
	res.Interface = req.Interface
	res.Filter = req.Filter
	return res, nil
}

// record copies frames from handle into f until the duration passes, the
// next frame would pass the size ceiling, or ctx ends.
func (s session) record(
	ctx context.Context,
	handle capture.Handle,
	f io.Writer,
	duration time.Duration,
	report func(float64),
) (*Result, error) {
	buf := bufio.NewWriter(f)
	w := pcapgo.NewWriter(buf)
	if err := w.WriteFileHeader(snaplen, handle.LinkType()); err != nil {
		return nil, fmt.Errorf("write capture header: %w", err)
	}
	res := &Result{Bytes: fileHeaderBytes}
	start := s.now()
	lastReport := start
	// Once a stop or the deadline is seen, frames the kernel already holds
	// are still read: libpcap on Linux hands frames over a buffer block at a
	// time, so the last ones before the stop arrive up to a read timeout
	// later. The drain ends at the first quiet read, or after drainWindow
	// on a link too busy to go quiet.
	var stopping StopReason
	var stoppedAt time.Time
	for res.StopReason == "" {
		now := s.now()
		if stopping == "" {
			stopping = stopCondition(ctx, now.Sub(start), duration)
			stoppedAt = now
		}
		switch {
		case stopping != "" && now.Sub(stoppedAt) >= drainWindow:
			res.StopReason = stopping
			continue
		case stopping == "" && now.Sub(lastReport) >= progressInterval:
			report(float64(now.Sub(start)) / float64(duration))
			lastReport = now
		}

		data, ci, err := handle.ReadPacketData()
		if errors.Is(err, capture.ErrTimeout) {
			res.StopReason = stopping // empty until a stop: keep reading
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read frame: %w", err)
		}
		size := int64(recordHeaderBytes + len(data))
		if res.Bytes+size > s.maxBytes {
			res.StopReason = StopSize
			continue
		}
		if err = w.WritePacket(ci, data); err != nil {
			return nil, fmt.Errorf("write frame: %w", err)
		}
		res.Packets++
		res.Bytes += size
	}
	if err := buf.Flush(); err != nil {
		return nil, fmt.Errorf("write capture file: %w", err)
	}
	if stopping == "" {
		stoppedAt = s.now()
	}
	res.DurationMs = stoppedAt.Sub(start).Milliseconds()
	return res, nil
}

// stopCondition reports whether a capture should stop: the operator stopped
// it, or it has run for its duration.
func stopCondition(ctx context.Context, elapsed, duration time.Duration) StopReason {
	switch {
	case ctx.Err() != nil:
		return StopStopped
	case elapsed >= duration:
		return StopDuration
	}
	return ""
}

// validate checks req and returns the duration it asks for.
func validate(req Request) (time.Duration, error) {
	if req.Interface == "" {
		return 0, ErrInterface
	}
	if req.DurationSeconds == 0 {
		return DefaultDuration, nil
	}
	d := time.Duration(req.DurationSeconds) * time.Second
	if d < 0 || d > MaxDuration {
		return 0, fmt.Errorf("durationSeconds must be between 1 and %d", int(MaxDuration.Seconds()))
	}
	return d, nil
}
