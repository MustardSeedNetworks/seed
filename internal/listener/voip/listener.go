// Package voip scores the voice quality of RTP streams seen on a capture
// interface (P-A7).
//
// Each RTP stream (one direction of a call, keyed by its addresses and
// SSRC) is tracked with the RFC 3550 receiver algorithms: cumulative loss
// from extended sequence numbers (A.1, A.3) and the interarrival jitter
// estimate (A.8). Every [ReportInterval], and when the stream ends, the
// window's loss and jitter go through the ITU-T G.107 E-model to an R
// factor and a MOS. A window at or below [AlertMOS] is also published as a
// [listener.VoIPQualityKind] event for the alert pipeline.
//
// Streams are found without signalling: a UDP flow counts as RTP once ten
// packets in a row carry version 2, one SSRC, one scored payload type and
// consecutive sequence numbers. Without the call's SDP only the static
// narrowband payload types are scored (PCMU, PCMA, G.729).
//
// The interface can be the probe's own link or a SPAN port: the analysis
// is per stream and does not depend on direction.
package voip

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

// Name is the listener key in the engine registry.
const Name = "voip"

// AlertMOS is the score at or below which a window raises an alert: R 70,
// the G.109 boundary below which many users are dissatisfied.
const AlertMOS = 3.6

const (
	// snaplen covers Ethernet, one VLAN tag, IPv6 with an extension
	// header, UDP and the fixed RTP header.
	snaplen     = 192
	bpfFilter   = "udp or (vlan and udp)"
	readTimeout = 100 * time.Millisecond

	rtpHeaderLen = 12
	rtpVersion   = 2
	// rtpPTMask strips the marker bit from the second header byte.
	rtpPTMask = 0x7f

	// IdleTimeout ends a stream that has sent nothing for this long.
	IdleTimeout = 5 * time.Second
	// maxStreams bounds the tracked streams and candidates; a new flow
	// beyond it is ignored until a sweep frees room.
	maxStreams    = 4096
	sweepInterval = time.Second

	queueSize     = 256
	flushInterval = time.Second
	statsInterval = time.Minute
)

// Store persists reports.
type Store interface {
	InsertVoIPReports(ctx context.Context, reports []Report) error
}

// Stats counts what the listener has seen since it started.
type Stats struct {
	Packets int64
	Streams int64
	Reports int64
	// Untracked counts packets of new flows ignored at maxStreams.
	Untracked int64
	// Dropped counts reports discarded because the queue was full.
	Dropped     int64
	StoreErrors int64
}

// Config configures a Listener.
type Config struct {
	Interface string
	Opener    capture.Opener
	Store     Store
	// Sink receives a [listener.VoIPQualityKind] event per window at or
	// below AlertMOS.
	Sink   listener.Sink
	Logger *slog.Logger
	Now    func() time.Time
}

// Listener captures on one interface and scores its RTP streams.
type Listener struct {
	cfg Config

	packets, streams, reports, untracked, dropped, failed atomic.Int64

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New validates cfg and returns an unstarted Listener.
func New(cfg Config) (*Listener, error) {
	switch {
	case cfg.Interface == "":
		return nil, errors.New("voip: Interface required")
	case cfg.Opener == nil:
		return nil, errors.New("voip: Opener required")
	case cfg.Store == nil:
		return nil, errors.New("voip: Store required")
	case cfg.Sink == nil:
		return nil, errors.New("voip: Sink required")
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &Listener{cfg: cfg}, nil
}

// Name implements [listener.Listener].
func (*Listener) Name() string { return Name }

// Stats returns the counters since Start.
func (l *Listener) Stats() Stats {
	return Stats{
		Packets:     l.packets.Load(),
		Streams:     l.streams.Load(),
		Reports:     l.reports.Load(),
		Untracked:   l.untracked.Load(),
		Dropped:     l.dropped.Load(),
		StoreErrors: l.failed.Load(),
	}
}

// Start opens the capture and starts the read and write loops. A second
// Start without a Stop is a no-op.
func (l *Listener) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		return nil
	}
	// Promiscuous, so a SPAN port's mirrored calls are seen.
	handle, err := l.cfg.Opener.OpenLive(l.cfg.Interface, snaplen, true, readTimeout)
	if err != nil {
		return fmt.Errorf("voip: open %s: %w", l.cfg.Interface, err)
	}
	if lt := handle.LinkType(); lt != layers.LinkTypeEthernet {
		handle.Close()
		return fmt.Errorf("voip: %s link type %s is not Ethernet", l.cfg.Interface, lt)
	}
	if filterErr := handle.SetBPFFilter(bpfFilter); filterErr != nil {
		handle.Close()
		return fmt.Errorf("voip: filter %s: %w", l.cfg.Interface, filterErr)
	}
	loopCtx, cancel := context.WithCancel(ctx)
	l.cancel, l.started = cancel, true

	queue := make(chan Report, queueSize)
	l.wg.Go(func() {
		defer handle.Close()
		l.read(loopCtx, handle, queue)
	})
	l.wg.Go(func() { l.write(loopCtx, queue) })
	l.cfg.Logger.InfoContext(ctx, "voip listener started", "interface", l.cfg.Interface)
	return nil
}

// Stop ends the capture, stores the windows already scored and waits for
// both loops, up to the ctx deadline.
func (l *Listener) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	l.started = false
	cancel := l.cancel
	l.mu.Unlock()

	cancel()
	done := make(chan struct{})
	go func() {
		l.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	l.cfg.Logger.InfoContext(ctx, "voip listener stopped", "interface", l.cfg.Interface)
	return nil
}

// tracker owns the stream table. It is not safe for concurrent use.
type tracker struct {
	iface     string
	streams   map[streamKey]*stream
	lastSweep time.Time
}

// read owns the handle and the stream table. It never blocks on the
// store: a report that does not fit the queue is dropped and counted.
func (l *Listener) read(ctx context.Context, handle capture.Handle, queue chan<- Report) {
	defer close(queue)
	tr := &tracker{iface: l.cfg.Interface, streams: make(map[streamKey]*stream)}
	emit := func(reports []Report) {
		for _, r := range reports {
			select {
			case queue <- r:
				l.reports.Add(1)
			default:
				l.dropped.Add(1)
			}
		}
	}
	dec := newDecoder()
	for ctx.Err() == nil {
		data, ci, err := handle.ReadPacketData()
		switch {
		case err == nil:
			l.packets.Add(1)
			if pkt, ok := dec.decode(data); ok {
				emit(l.observe(tr, pkt, ci.Timestamp))
			}
			if ci.Timestamp.Sub(tr.lastSweep) >= sweepInterval {
				emit(tr.sweep(ci.Timestamp))
			}
		case errors.Is(err, capture.ErrTimeout):
			emit(tr.sweep(l.cfg.Now()))
		default:
			l.cfg.Logger.WarnContext(ctx, "voip capture read failed",
				"interface", l.cfg.Interface, "error", err)
			emit(tr.finish())
			return
		}
	}
	emit(tr.finish())
}

func (l *Listener) observe(tr *tracker, pkt rtpPacket, at time.Time) []Report {
	s, ok := tr.streams[pkt.key]
	if !ok {
		codec, scored := codecFor(pkt.pt)
		if !scored {
			return nil
		}
		if len(tr.streams) >= maxStreams {
			l.untracked.Add(1)
			return nil
		}
		tr.streams[pkt.key] = newStream(pkt.key, pkt.pt, codec, pkt.seq, at)
		return nil
	}
	wasConfirmed := s.confirmed()
	report, closed := s.add(tr.iface, pkt.pt, pkt.seq, pkt.ts, at)
	if !wasConfirmed && s.confirmed() {
		l.streams.Add(1)
	}
	if closed {
		return []Report{report}
	}
	return nil
}

// sweep ends every stream idle for IdleTimeout before now, reporting its
// last window, and forgets idle candidates.
func (tr *tracker) sweep(now time.Time) []Report {
	tr.lastSweep = now
	var reports []Report
	for k, s := range tr.streams {
		if now.Sub(s.lastSeen) < IdleTimeout {
			continue
		}
		if s.confirmed() {
			if r, ok := s.close(tr.iface, s.lastSeen, now); ok {
				reports = append(reports, r)
			}
		}
		delete(tr.streams, k)
	}
	return reports
}

// finish reports every open window, the last call before the capture ends.
func (tr *tracker) finish() []Report {
	var reports []Report
	for _, s := range tr.streams {
		if !s.confirmed() {
			continue
		}
		if r, ok := s.close(tr.iface, s.lastSeen, s.lastSeen); ok {
			reports = append(reports, r)
		}
	}
	clear(tr.streams)
	return reports
}

// rtpPacket is the part of an RTP packet the analyser reads.
type rtpPacket struct {
	key streamKey
	pt  uint8
	seq uint16
	ts  uint32
}

type decoder struct {
	eth    layers.Ethernet
	dot1q  layers.Dot1Q
	ip4    layers.IPv4
	ip6    layers.IPv6
	udp    layers.UDP
	parser *gopacket.DecodingLayerParser
	found  []gopacket.LayerType
}

func newDecoder() *decoder {
	d := &decoder{}
	d.parser = gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet,
		&d.eth, &d.dot1q, &d.ip4, &d.ip6, &d.udp)
	// The UDP payload is read directly; nothing decodes past it.
	d.parser.IgnoreUnsupported = true
	return d
}

// decode returns frame's RTP header when it is a UDP datagram whose
// payload reads as RTP version 2.
func (d *decoder) decode(frame []byte) (rtpPacket, bool) {
	if err := d.parser.DecodeLayers(frame, &d.found); err != nil {
		return rtpPacket{}, false
	}
	var src, dst netip.Addr
	var udp bool
	for _, t := range d.found {
		switch t {
		case layers.LayerTypeIPv4:
			src, _ = netip.AddrFromSlice(d.ip4.SrcIP.To4())
			dst, _ = netip.AddrFromSlice(d.ip4.DstIP.To4())
		case layers.LayerTypeIPv6:
			src, _ = netip.AddrFromSlice(d.ip6.SrcIP)
			dst, _ = netip.AddrFromSlice(d.ip6.DstIP)
		case layers.LayerTypeUDP:
			udp = true
		}
	}
	payload := d.udp.Payload
	if !udp || !src.IsValid() || len(payload) < rtpHeaderLen || payload[0]>>6 != rtpVersion {
		return rtpPacket{}, false
	}
	return rtpPacket{
		key: streamKey{
			src:  netip.AddrPortFrom(src, uint16(d.udp.SrcPort)),
			dst:  netip.AddrPortFrom(dst, uint16(d.udp.DstPort)),
			ssrc: binary.BigEndian.Uint32(payload[8:12]),
		},
		pt:  payload[1] & rtpPTMask,
		seq: binary.BigEndian.Uint16(payload[2:4]),
		ts:  binary.BigEndian.Uint32(payload[4:8]),
	}, true
}

// write stores reports in batches and publishes the poor ones until the
// queue closes.
func (l *Listener) write(ctx context.Context, queue <-chan Report) {
	flush := time.NewTicker(flushInterval)
	defer flush.Stop()
	stats := time.NewTicker(statsInterval)
	defer stats.Stop()
	var batch []Report
	store := func() {
		if len(batch) == 0 {
			return
		}
		// The loop context is cancelled on Stop; the final batch still
		// has to land.
		storeCtx := context.WithoutCancel(ctx)
		if err := l.cfg.Store.InsertVoIPReports(storeCtx, batch); err != nil {
			l.failed.Add(1)
			l.cfg.Logger.WarnContext(ctx, "voip store failed", "reports", len(batch), "error", err)
		}
		for i := range batch {
			if batch[i].MOS <= AlertMOS {
				l.publish(storeCtx, &batch[i])
			}
		}
		batch = nil
	}
	for {
		select {
		case r, ok := <-queue:
			if !ok {
				store()
				return
			}
			batch = append(batch, r)
		case <-flush.C:
			store()
		case <-stats.C:
			s := l.Stats()
			l.cfg.Logger.DebugContext(ctx, "voip listener stats",
				"packets", s.Packets, "streams", s.Streams, "reports", s.Reports,
				"untracked", s.Untracked, "dropped", s.Dropped, "store_errors", s.StoreErrors)
		}
	}
}

func (l *Listener) publish(ctx context.Context, r *Report) {
	payload, err := json.Marshal(listener.VoIPQuality{
		Interface:   r.Interface,
		Src:         r.Src.String(),
		Dst:         r.Dst.String(),
		SSRC:        r.SSRC,
		Codec:       r.Codec,
		Start:       r.Start,
		End:         r.End,
		LossPct:     r.LossPct,
		JitterMs:    r.JitterMs,
		MaxJitterMs: r.MaxJitterMs,
		RFactor:     r.RFactor,
		MOS:         r.MOS,
	})
	if err != nil {
		l.cfg.Logger.WarnContext(ctx, "voip: encode quality event", "error", err)
		return
	}
	if pubErr := l.cfg.Sink.Publish(ctx, listener.Event{
		Kind:       listener.VoIPQualityKind,
		SourceAddr: r.Src.Addr().String(),
		Severity:   "warning",
		Timestamp:  l.cfg.Now(),
		Payload:    payload,
	}); pubErr != nil {
		l.cfg.Logger.WarnContext(ctx, "voip: publish quality event failed",
			"src", r.Src, "dst", r.Dst, "error", pubErr)
	}
}
