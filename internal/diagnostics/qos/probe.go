package qos

import (
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"time"
)

const (
	// probeSize is every probe's length. The fields fit in 29 bytes; the
	// rest is zero so a probe is not mistaken for a runt by a middlebox.
	probeSize = 64

	// probeInterval spaces a send's datagrams so a burst does not overrun a
	// shallow queue and turn a marking check into a loss measurement.
	probeInterval = 5 * time.Millisecond

	// maxTrackedRuns bounds the listener's table: anyone can reach the port
	// while it is open, and a flood of forged runs must not grow it.
	maxTrackedRuns = 16

	// readBufferSize holds any datagram; only probeSize bytes are ever read.
	readBufferSize = 2048

	// probeMagic opens every probe, so a stray datagram on the port is
	// ignored rather than read as a class.
	probeMagic = "SEEDQOS1"
)

// probe is one datagram's fields. Each carries its run's whole class set and
// count, so the listener knows which classes to expect and how many of each.
type probe struct {
	run   uint64
	mask  uint64
	dscp  uint8
	count uint16
	seq   uint16
}

func (p probe) encode() []byte {
	b := make([]byte, probeSize)
	copy(b, probeMagic)
	binary.BigEndian.PutUint64(b[8:], p.run)
	binary.BigEndian.PutUint64(b[16:], p.mask)
	b[24] = p.dscp
	binary.BigEndian.PutUint16(b[25:], p.count)
	binary.BigEndian.PutUint16(b[27:], p.seq)
	return b
}

// decodeProbe reads a datagram as a probe, refusing one whose fields are
// inconsistent with each other.
func decodeProbe(b []byte) (probe, bool) {
	if len(b) < 29 || string(b[:8]) != probeMagic {
		return probe{}, false
	}
	p := probe{
		run:   binary.BigEndian.Uint64(b[8:]),
		mask:  binary.BigEndian.Uint64(b[16:]),
		dscp:  b[24],
		count: binary.BigEndian.Uint16(b[25:]),
		seq:   binary.BigEndian.Uint16(b[27:]),
	}
	if p.dscp > maxDSCP || p.mask&(1<<p.dscp) == 0 || p.count < 1 || p.count > MaxCount || p.seq >= p.count {
		return probe{}, false
	}
	return p, true
}

// probeConn is the sending socket, behind a seam so the send loop is
// testable without one.
type probeConn interface {
	setDSCP(dscp uint8) error
	write(b []byte) error
}

// send puts the spec's probes on conn, interleaving the classes so a burst
// that congests a queue costs every class alike. A cancel stops it and
// reports what went out.
func send(
	ctx context.Context,
	conn probeConn,
	s sendSpec,
	run uint64,
	wait func(context.Context) error,
) (*SendResult, error) {
	classes := s.classes()
	sent := make([]int, len(classes))
	var err error
sending:
	for seq := range s.count {
		for i, d := range classes {
			if err = conn.setDSCP(d); err != nil {
				err = fmt.Errorf("set DSCP %d: %w", d, err)
				break sending
			}
			p := probe{run: run, mask: s.mask, dscp: d, count: s.count, seq: seq}
			if err = conn.write(p.encode()); err != nil {
				err = fmt.Errorf("send DSCP %d to %s: %w", d, s.target, err)
				break sending
			}
			sent[i]++
			if err = wait(ctx); err != nil {
				break sending
			}
		}
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		return nil, err
	}

	res := &SendResult{
		Target:  s.target.Addr().String(),
		Port:    int(s.target.Port()),
		RunID:   fmt.Sprintf("%016x", run),
		Count:   int(s.count),
		Classes: make([]SentClass, len(classes)),
	}
	for i, d := range classes {
		res.Classes[i] = SentClass{DSCP: int(d), Name: Name(int(d)), Sent: sent[i]}
	}
	return res, nil
}

// packetSource is the listening socket, behind a seam so the tally is
// testable without one.
type packetSource interface {
	// read returns one datagram, its sender and the DSCP it arrived with;
	// dscp is -1 where the platform cannot report it.
	read(buf []byte) (n int, from netip.Addr, dscp int, err error)
	setDeadline(t time.Time)
	// interrupt ends a blocked read.
	interrupt()
	observesDSCP() bool
}

type runKey struct {
	sender netip.Addr
	run    uint64
}

// runTally accumulates one run. The first probe fixes the run's class set and
// count; a later probe that disagrees is forged or corrupt and is ignored.
type runTally struct {
	mask  uint64
	count uint16
	seen  map[uint8][]bool
	marks map[uint8]map[int]int
}

func newRunTally(p probe) *runTally {
	return &runTally{mask: p.mask, count: p.count, seen: map[uint8][]bool{}, marks: map[uint8]map[int]int{}}
}

func (t *runTally) add(p probe, dscp int) bool {
	if p.mask != t.mask || p.count != t.count {
		return false
	}
	seen := t.seen[p.dscp]
	if seen == nil {
		seen = make([]bool, t.count)
		t.seen[p.dscp] = seen
	}
	if seen[p.seq] {
		return true
	}
	seen[p.seq] = true
	if t.marks[p.dscp] == nil {
		t.marks[p.dscp] = map[int]int{}
	}
	t.marks[p.dscp][dscp]++
	return true
}

// collect reads probes until the window closes or ctx ends. Ending early is
// not a failure: what arrived is the answer the operator stopped for.
func collect(ctx context.Context, src packetSource, s listenSpec, now func() time.Time) (*ListenResult, error) {
	started := now()
	src.setDeadline(started.Add(s.window))
	stop := context.AfterFunc(ctx, src.interrupt)
	defer stop()

	res := &ListenResult{Port: s.port, Family: s.family(), DSCPObserved: src.observesDSCP()}
	runs := map[runKey]*runTally{}
	buf := make([]byte, readBufferSize)

	for ctx.Err() == nil {
		n, from, dscp, err := src.read(buf)
		if err != nil {
			// The window closing and interrupt both surface as a deadline.
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			return nil, err
		}
		p, ok := decodeProbe(buf[:n])
		if !ok {
			res.Ignored++
			continue
		}
		key := runKey{sender: from.Unmap(), run: p.run}
		tally, ok := runs[key]
		if !ok {
			if len(runs) >= maxTrackedRuns {
				res.RunsTruncated = true
				res.Ignored++
				continue
			}
			tally = newRunTally(p)
			runs[key] = tally
		}
		if !tally.add(p, dscp) {
			res.Ignored++
			continue
		}
		res.Probes++
	}

	res.ListenedMs = now().Sub(started).Milliseconds()
	res.Runs = summarize(runs)
	return res, nil
}

func summarize(runs map[runKey]*runTally) []RunResult {
	out := make([]RunResult, 0, len(runs))
	for key, t := range runs {
		r := RunResult{RunID: fmt.Sprintf("%016x", key.run), Sender: key.sender.String(), Preserved: true}
		for _, d := range (sendSpec{mask: t.mask}).classes() {
			c := classResult(d, t)
			r.Preserved = r.Preserved && c.Verdict == VerdictPreserved
			r.Classes = append(r.Classes, c)
		}
		out = append(out, r)
	}
	slices.SortFunc(out, func(a, b RunResult) int {
		return cmp.Or(cmp.Compare(a.Sender, b.Sender), cmp.Compare(a.RunID, b.RunID))
	})
	return out
}

func classResult(d uint8, t *runTally) ClassResult {
	c := ClassResult{SentDSCP: int(d), SentName: Name(int(d)), Expected: int(t.count), Observed: []ObservedDSCP{}}
	for mark, n := range t.marks[d] {
		c.Received += n
		if mark >= 0 {
			c.Observed = append(c.Observed, ObservedDSCP{DSCP: mark, Name: Name(mark), Count: n})
		}
	}
	slices.SortFunc(c.Observed, func(a, b ObservedDSCP) int {
		return cmp.Or(cmp.Compare(b.Count, a.Count), cmp.Compare(a.DSCP, b.DSCP))
	})
	switch {
	case c.Received == 0:
		c.Verdict = VerdictLost
	case len(c.Observed) == 0:
		c.Verdict = VerdictUnobserved
	case len(c.Observed) > 1:
		c.Verdict = VerdictMixed
	case c.Observed[0].DSCP == int(d):
		c.Verdict = VerdictPreserved
	default:
		c.Verdict = VerdictRemarked
	}
	return c
}
