package voip

import (
	"net/netip"
	"time"
)

const (
	// probationPackets is how many in-sequence packets with one SSRC and
	// payload type a UDP flow needs before it counts as RTP. Any UDP
	// payload whose first byte reads as version 2 starts a candidate; a
	// run of consecutive sequence numbers is what real RTP shows.
	probationPackets = 10

	// RFC 3550 A.1: a forward jump under maxDropout is loss, a backward
	// step under maxMisorder is reordering, and anything else is a
	// restart that a second packet must confirm.
	maxDropout  = 3000
	maxMisorder = 100
	seqMod      = 1 << 16

	// ReportInterval bounds one report, so a long call that degrades for
	// a minute is not averaged away.
	ReportInterval = time.Minute
	// minReportPackets is the shortest window scored: one second of 20 ms
	// audio. A shorter tail at the end of a call is dropped.
	minReportPackets = 50
	// defaultPacketizationMs is assumed until two consecutive packets
	// show the stream's own.
	defaultPacketizationMs = 20.0

	// jitterGain is RFC 3550's 1/16 smoothing of the jitter estimate.
	jitterGain = 16
	// jitterBufferFactor sizes the receiver's jitter buffer, which the
	// E-model counts as delay, at twice the mean jitter.
	jitterBufferFactor = 2
	msPerSecond        = 1000
	percent            = 100
)

// Report is one stream's quality over one window.
type Report struct {
	Interface string
	Src, Dst  netip.AddrPort
	SSRC      uint32
	Codec     string
	Start     time.Time
	End       time.Time
	Expected  int64
	Received  int64
	// LossPct is the RFC 3550 cumulative loss over the window, in percent.
	LossPct float64
	// JitterMs and MaxJitterMs are the mean and peak of the RFC 3550
	// interarrival jitter estimate across the window.
	JitterMs    float64
	MaxJitterMs float64
	// DelayMs is the delay the E-model was given: packetization, encoder
	// lookahead and a jitter buffer of twice the mean jitter. The
	// network's own one-way delay is not visible to a passive probe and
	// is not included.
	DelayMs float64
	RFactor float64
	MOS     float64
}

type streamKey struct {
	src, dst netip.AddrPort
	ssrc     uint32
}

// stream tracks one RTP stream. It is not safe for concurrent use.
type stream struct {
	key   streamKey
	pt    uint8
	codec Codec

	// probation counts down to zero before the stream is confirmed.
	probation int
	lastSeen  time.Time

	// RFC 3550 A.1 sequence state.
	maxSeq   uint16
	cycles   int64
	baseSeq  int64
	badSeq   int64
	received int64

	// RFC 3550 A.8 jitter state, in timestamp units.
	firstArrival    time.Time
	firstTS         uint32
	transit         int64
	haveTransit     bool
	jitter          float64
	lastTS          uint32
	packetizationMs float64

	// The open window.
	windowStart   time.Time
	expectedPrior int64
	receivedPrior int64
	jitterSumMs   float64
	jitterMaxMs   float64
	jitterSamples int64
}

func newStream(key streamKey, pt uint8, codec Codec, seq uint16, at time.Time) *stream {
	return &stream{
		key:             key,
		pt:              pt,
		codec:           codec,
		probation:       probationPackets - 1,
		maxSeq:          seq,
		lastSeen:        at,
		packetizationMs: defaultPacketizationMs,
	}
}

func (s *stream) confirmed() bool { return s.probation == 0 }

// add accounts one packet and returns the window it closed, if any.
func (s *stream) add(iface string, pt uint8, seq uint16, ts uint32, at time.Time) (Report, bool) {
	// Another payload type on the same SSRC is a comfort noise or DTMF
	// packet, not audio.
	if pt != s.pt {
		return Report{}, false
	}
	prev := s.lastSeen
	s.lastSeen = at
	if !s.confirmed() {
		s.probe(seq, ts, at)
		return Report{}, false
	}
	var report Report
	var closed bool
	if at.Sub(s.windowStart) >= ReportInterval {
		report, closed = s.close(iface, prev, at)
	}
	inOrder := seq == s.maxSeq+1
	if s.updateSeq(seq) {
		s.updateJitter(ts, at, inOrder)
	}
	return report, closed
}

// probe advances probation. A packet out of sequence restarts it.
func (s *stream) probe(seq uint16, ts uint32, at time.Time) {
	if seq != s.maxSeq+1 {
		s.probation, s.maxSeq = probationPackets-1, seq
		return
	}
	s.maxSeq = seq
	s.probation--
	if s.probation > 0 {
		return
	}
	s.initSeq(seq)
	s.received = 1
	s.firstArrival, s.firstTS, s.lastTS = at, ts, ts
	s.transit, s.haveTransit = 0, true
	s.windowStart = at
}

func (s *stream) initSeq(seq uint16) {
	s.baseSeq = int64(seq)
	s.maxSeq = seq
	s.badSeq = seqMod + 1
	s.cycles = 0
	s.received = 0
	s.expectedPrior, s.receivedPrior = 0, 0
}

// updateSeq is RFC 3550 A.1 after probation. It reports whether the
// packet is counted.
func (s *stream) updateSeq(seq uint16) bool {
	udelta := seq - s.maxSeq
	switch {
	case udelta < maxDropout:
		if seq < s.maxSeq {
			s.cycles += seqMod
		}
		s.maxSeq = seq
	case int(udelta) <= seqMod-maxMisorder:
		if int64(seq) != s.badSeq {
			s.badSeq = int64(seq + 1)
			return false
		}
		// Two sequential packets after a jump: the source restarted.
		s.initSeq(seq)
		s.haveTransit = false
	}
	s.received++
	return true
}

func (s *stream) updateJitter(ts uint32, at time.Time, inOrder bool) {
	rate := s.codec.ClockRate
	arrival := at.Sub(s.firstArrival).Nanoseconds() * int64(rate) / int64(time.Second)
	transit := arrival - int64(ts-s.firstTS)
	if s.haveTransit {
		d := transit - s.transit
		if d < 0 {
			d = -d
		}
		s.jitter += (float64(d) - s.jitter) / jitterGain
	}
	if step := ts - s.lastTS; inOrder && s.haveTransit && step > 0 && step < rate {
		s.packetizationMs = float64(step) * msPerSecond / float64(rate)
	}
	s.transit, s.haveTransit, s.lastTS = transit, true, ts

	ms := s.jitter * msPerSecond / float64(rate)
	s.jitterSumMs += ms
	s.jitterMaxMs = max(s.jitterMaxMs, ms)
	s.jitterSamples++
}

func (s *stream) expected() int64 {
	return s.cycles + int64(s.maxSeq) - s.baseSeq + 1
}

// close scores the window that ended at end and opens the next at next.
func (s *stream) close(iface string, end, next time.Time) (Report, bool) {
	expected, received := s.expected(), s.received
	expInterval := expected - s.expectedPrior
	recInterval := received - s.receivedPrior
	jitterMs := 0.0
	if s.jitterSamples > 0 {
		jitterMs = s.jitterSumMs / float64(s.jitterSamples)
	}
	report := Report{
		Interface:   iface,
		Src:         s.key.src,
		Dst:         s.key.dst,
		SSRC:        s.key.ssrc,
		Codec:       s.codec.Name,
		Start:       s.windowStart,
		End:         end,
		Expected:    expInterval,
		Received:    recInterval,
		JitterMs:    jitterMs,
		MaxJitterMs: s.jitterMaxMs,
	}
	s.windowStart, s.expectedPrior, s.receivedPrior = next, expected, received
	s.jitterSumMs, s.jitterMaxMs, s.jitterSamples = 0, 0, 0

	if expInterval < minReportPackets {
		return Report{}, false
	}
	// Duplicates can push received past expected; that is no loss.
	lost := max(expInterval-recInterval, 0)
	report.LossPct = percent * float64(lost) / float64(expInterval)
	report.DelayMs = s.packetizationMs + s.codec.LookaheadMs + jitterBufferFactor*jitterMs
	report.RFactor = RFactor(s.codec, report.DelayMs, report.LossPct)
	report.MOS = MOS(report.RFactor)
	return report, true
}
