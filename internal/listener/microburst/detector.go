// Package microburst measures sub-second link saturation on the probe's
// own interface from captured frames (P-A6, OA-11 #3029).
//
// SNMP cannot see a microburst: agents refresh their interface counters
// every few seconds (net-snmp every ~3 s), so a short poll interval reads
// zero and then one jump far above line rate. The detector instead sums
// each captured frame's time on the wire into 1 ms bins per direction and
// reports a run of bins at or above 90 % of line rate.
//
// Each frame occupies the wire for its serialization time at link speed,
// starting at its capture timestamp or when the previous frame in that
// direction finished, whichever is later. A frame cannot overlap the one
// before it on a real wire, so a batch of frames the kernel stamped
// together, or one TSO/GRO super-frame, is spread at line rate instead of
// landing in one bin as an impossible spike. A bin can therefore never
// read above 100 %.
//
// Scope: the probe's own link, where a frame sourced from the interface's
// MAC is outbound and every other frame is inbound. A SPAN port mirrors
// both directions of another port as inbound and is not measured correctly.
package microburst

import (
	"time"
)

const (
	// BinWidth is the measurement resolution.
	BinWidth = time.Millisecond
	// Threshold is the fraction of line rate a bin must reach to count
	// as saturated.
	Threshold = 0.90
	// FloorBins is the shortest run of saturated bins reported as a
	// burst. One bin alone can be a timestamping artefact.
	FloorBins = 2

	// wireOverheadBytes is what the wire carries per frame beyond the
	// captured length: FCS (4), preamble and SFD (8), inter-frame gap (12).
	wireOverheadBytes = 24
	// minFrameBytes is the shortest Ethernet frame without FCS; shorter
	// captured frames are padded on the wire.
	minFrameBytes  = 60
	bitsPerByte    = 8
	bitsPerMegabit = 1_000_000
)

// Direction is the side of the link a frame travelled.
type Direction string

// The two directions, as stored in microburst_events.direction.
const (
	DirectionIn  Direction = "in"
	DirectionOut Direction = "out"
)

// Event is one microburst: consecutive bins in one direction at or above
// Threshold.
type Event struct {
	Start     time.Time
	Interface string
	Direction Direction
	Duration  time.Duration
	// PeakUtilization is the busiest bin's share of line rate, in percent.
	PeakUtilization float64
	LinkSpeedMbps   int64
}

// detector bins one interface's frames. It is not safe for concurrent use.
type detector struct {
	iface     string
	speedMbps int64
	speedBps  int64
	dirs      map[Direction]*binner
}

func newDetector(iface string, speedMbps int64) *detector {
	return &detector{
		iface:     iface,
		speedMbps: speedMbps,
		speedBps:  speedMbps * bitsPerMegabit,
		dirs: map[Direction]*binner{
			DirectionIn:  {},
			DirectionOut: {},
		},
	}
}

// add accounts one frame of capturedLen bytes stamped at ts and returns
// any burst it closed.
func (d *detector) add(dir Direction, ts time.Time, capturedLen int) []Event {
	length := max(capturedLen, minFrameBytes) + wireOverheadBytes
	bits := int64(length) * bitsPerByte
	// Rounded to the nearest nanosecond: the error is unbiased and at
	// most 0.5 ns a frame.
	wire := (bits*int64(time.Second) + d.speedBps/2) / d.speedBps
	return d.wrap(dir, d.dirs[dir].add(ts.UnixNano(), wire))
}

// flush closes every bin that ended before now and returns the bursts
// that finished. It runs while the link is quiet, so a burst followed by
// silence is reported without waiting for the next frame.
func (d *detector) flush(now time.Time) []Event {
	var events []Event
	for _, dir := range []Direction{DirectionIn, DirectionOut} {
		events = append(events, d.wrap(dir, d.dirs[dir].flush(now.UnixNano()))...)
	}
	return events
}

// finish closes every open bin, the last call before the capture ends.
func (d *detector) finish() []Event {
	var events []Event
	for _, dir := range []Direction{DirectionIn, DirectionOut} {
		events = append(events, d.wrap(dir, d.dirs[dir].finish())...)
	}
	return events
}

func (d *detector) wrap(dir Direction, runs []run) []Event {
	if len(runs) == 0 {
		return nil
	}
	events := make([]Event, len(runs))
	for i, r := range runs {
		events[i] = Event{
			Start:           time.Unix(0, r.start).UTC(),
			Interface:       d.iface,
			Direction:       dir,
			Duration:        time.Duration(r.bins) * BinWidth,
			PeakUtilization: 100 * float64(r.peakBusy) / float64(BinWidth),
			LinkSpeedMbps:   d.speedMbps,
		}
	}
	return events
}

// run is a finished burst in nanoseconds since the epoch.
type run struct {
	start    int64
	bins     int64
	peakBusy int64
}

// binner holds one direction's wire cursor, open bin and open run.
type binner struct {
	// busyUntil is when the last frame finished on the wire.
	busyUntil int64
	// bin is the open bin's start; busy is its wire time so far.
	bin, busy int64
	open      bool

	current run
}

const binNs = int64(BinWidth)

func (b *binner) add(ts, wire int64) []run {
	start := max(ts, b.busyUntil)
	end := start + wire
	b.busyUntil = end
	var done []run
	for start < end {
		bin := start - start%binNs
		if !b.open || bin != b.bin {
			done = append(done, b.advance(bin)...)
		}
		seg := min(end, bin+binNs) - start
		b.busy += seg
		start += seg
	}
	return done
}

// advance closes the open bin and opens next. Idle bins in between end
// any run.
func (b *binner) advance(next int64) []run {
	var done []run
	if b.open {
		done = b.close()
		if next != b.bin+binNs {
			done = append(done, b.endRun()...)
		}
	}
	b.bin, b.busy, b.open = next, 0, true
	return done
}

func (b *binner) flush(now int64) []run {
	if !b.open || now < b.bin+binNs || now < b.busyUntil {
		return nil
	}
	return b.finish()
}

func (b *binner) finish() []run {
	if !b.open {
		return nil
	}
	done := append(b.close(), b.endRun()...)
	// A frame stamped inside the closed bin but delivered after the
	// flush is placed after it, never back into it.
	b.busyUntil = max(b.busyUntil, b.bin+binNs)
	b.open = false
	return done
}

// close scores the open bin against the threshold.
func (b *binner) close() []run {
	if float64(b.busy) >= Threshold*float64(binNs) {
		if b.current.bins == 0 {
			b.current.start = b.bin
		}
		b.current.bins++
		b.current.peakBusy = max(b.current.peakBusy, b.busy)
		return nil
	}
	return b.endRun()
}

func (b *binner) endRun() []run {
	r := b.current
	b.current = run{}
	if r.bins < FloorBins {
		return nil
	}
	return []run{r}
}
