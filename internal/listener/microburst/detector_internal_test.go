package microburst

import (
	"testing"
	"time"
)

// t0() is bin-aligned so expected starts are exact.
func t0() time.Time { return time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC) }

// fullFrame is a 1500-byte payload frame: 1514 captured bytes, 1538 on
// the wire, 12.304 µs at 1 Gb/s.
const (
	fullFrame = 1514
	speed1G   = 1000
)

type frame struct {
	dir Direction
	ts  time.Time
	len int
}

// paced returns frames of fullFrame bytes that keep dir at util of
// 1 Gb/s from start for d.
func paced(dir Direction, start time.Time, d time.Duration, util float64) []frame {
	wire := time.Duration(float64((fullFrame+wireOverheadBytes)*bitsPerByte) / util)
	var out []frame
	for at := time.Duration(0); at < d; at += wire {
		out = append(out, frame{dir, start.Add(at), fullFrame})
	}
	return out
}

func concat(parts ...[]frame) []frame {
	var out []frame
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func detect(frames []frame) []Event {
	d := newDetector("eth0", speed1G)
	var events []Event
	for _, f := range frames {
		events = append(events, d.add(f.dir, f.ts, f.len)...)
	}
	return append(events, d.finish()...)
}

func TestDetector(t *testing.T) {
	t.Parallel()

	batch := make([]frame, 1000)
	for i := range batch {
		batch[i] = frame{DirectionIn, t0(), fullFrame}
	}

	tests := []struct {
		name   string
		frames []frame
		want   []Event
	}{
		{
			name:   "idle link",
			frames: paced(DirectionIn, t0(), 50*time.Millisecond, 0.10),
		},
		{
			name:   "sustained 60 percent is no burst",
			frames: paced(DirectionIn, t0(), 50*time.Millisecond, 0.60),
		},
		{
			name:   "89 percent stays under the threshold",
			frames: paced(DirectionIn, t0(), 20*time.Millisecond, 0.89),
		},
		{
			name:   "92 percent for 20 ms is one burst",
			frames: paced(DirectionIn, t0(), 20*time.Millisecond, 0.92),
			want:   []Event{{Start: t0(), Direction: DirectionIn, Duration: 20 * time.Millisecond}},
		},
		{
			name:   "line rate for 5 ms",
			frames: paced(DirectionOut, t0(), 5*time.Millisecond, 1),
			want:   []Event{{Start: t0(), Direction: DirectionOut, Duration: 5 * time.Millisecond}},
		},
		{
			name:   "one saturated bin is under the floor",
			frames: paced(DirectionIn, t0(), time.Millisecond, 1),
		},
		{
			name: "two bursts split by an idle gap",
			frames: concat(
				paced(DirectionIn, t0(), 3*time.Millisecond, 1),
				paced(DirectionIn, t0().Add(10*time.Millisecond), 4*time.Millisecond, 1),
			),
			want: []Event{
				{Start: t0(), Direction: DirectionIn, Duration: 3 * time.Millisecond},
				{Start: t0().Add(10 * time.Millisecond), Direction: DirectionIn, Duration: 4 * time.Millisecond},
			},
		},
		{
			name: "two bursts split by a quiet bin",
			frames: concat(
				paced(DirectionIn, t0(), 3*time.Millisecond, 1),
				paced(DirectionIn, t0().Add(3*time.Millisecond), time.Millisecond, 0.5),
				paced(DirectionIn, t0().Add(4*time.Millisecond), 2*time.Millisecond, 1),
			),
			want: []Event{
				{Start: t0(), Direction: DirectionIn, Duration: 3 * time.Millisecond},
				{Start: t0().Add(4 * time.Millisecond), Direction: DirectionIn, Duration: 2 * time.Millisecond},
			},
		},
		{
			name: "directions are measured apart",
			frames: concat(
				paced(DirectionIn, t0(), 10*time.Millisecond, 0.6),
				paced(DirectionOut, t0(), 10*time.Millisecond, 0.6),
			),
		},
		{
			// 1000 frames the kernel stamped together take 12.3 ms of
			// wire: twelve full bins, the thirteenth 30 % busy.
			name:   "a same-timestamp batch is spread at line rate",
			frames: batch,
			want:   []Event{{Start: t0(), Direction: DirectionIn, Duration: 12 * time.Millisecond}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assertEvents(t, detect(tt.frames), tt.want)
		})
	}
}

func assertEvents(t *testing.T, got, want []Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d events %+v, want %d", len(got), got, len(want))
	}
	for i, w := range want {
		g := got[i]
		if !g.Start.Equal(w.Start) || g.Direction != w.Direction || g.Duration != w.Duration {
			t.Errorf("event %d = %s %s %s, want %s %s %s",
				i, g.Start, g.Direction, g.Duration, w.Start, w.Direction, w.Duration)
		}
		if g.PeakUtilization < Threshold*100 || g.PeakUtilization > 100 {
			t.Errorf("event %d peak = %.2f%%, want within [%.0f, 100]", i, g.PeakUtilization, Threshold*100)
		}
		if g.Interface != "eth0" || g.LinkSpeedMbps != speed1G {
			t.Errorf("event %d link = %s %d, want eth0 %d", i, g.Interface, g.LinkSpeedMbps, speed1G)
		}
	}
}

// A burst followed by silence is reported once the flush lag passes,
// without waiting for another frame.
func TestDetectorFlushReportsBurstOnQuietLink(t *testing.T) {
	t.Parallel()
	d := newDetector("eth0", speed1G)
	for _, f := range paced(DirectionIn, t0(), 4*time.Millisecond, 1) {
		if ev := d.add(f.dir, f.ts, f.len); len(ev) != 0 {
			t.Fatalf("burst reported before it ended: %+v", ev)
		}
	}
	if ev := d.flush(t0().Add(3 * time.Millisecond)); len(ev) != 0 {
		t.Fatalf("flush inside the open bin reported %+v", ev)
	}
	ev := d.flush(t0().Add(10 * time.Millisecond))
	if len(ev) != 1 || ev[0].Duration != 4*time.Millisecond {
		t.Fatalf("flush after the burst = %+v, want one 4 ms burst", ev)
	}
	// A late frame stamped inside the flushed bins cannot reopen them.
	if late := d.add(DirectionIn, t0().Add(2*time.Millisecond), fullFrame); len(late) != 0 {
		t.Fatalf("late frame reported %+v", late)
	}
	if rest := d.finish(); len(rest) != 0 {
		t.Fatalf("finish after a lone late frame = %+v, want none", rest)
	}
}

// Small frames are padded to the Ethernet minimum on the wire.
func TestDetectorPadsRuntFrames(t *testing.T) {
	t.Parallel()
	// 64-byte minimum + 20 preamble/IFG = 84 bytes = 672 ns at 1 Gb/s.
	const wire = 672 * time.Nanosecond
	var frames []frame
	for at := time.Duration(0); at < 3*time.Millisecond; at += wire {
		frames = append(frames, frame{DirectionIn, t0().Add(at), 42})
	}
	got := detect(frames)
	if len(got) != 1 || got[0].Duration != 3*time.Millisecond {
		t.Fatalf("minimum-size frames at line rate = %+v, want one 3 ms burst", got)
	}
}
