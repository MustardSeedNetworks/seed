package voip

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"math"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture/capturetest"
	"github.com/MustardSeedNetworks/seed/internal/listener"
)

type memStore struct {
	mu      sync.Mutex
	reports []Report
}

func (s *memStore) InsertVoIPReports(_ context.Context, reports []Report) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reports = append(s.reports, reports...)
	return nil
}

func (s *memStore) all() []Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Report(nil), s.reports...)
}

type memSink struct {
	mu     sync.Mutex
	events []listener.Event
}

func (s *memSink) Publish(_ context.Context, evt listener.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evt)
	return nil
}

func (s *memSink) all() []listener.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]listener.Event(nil), s.events...)
}

func t0() time.Time { return time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC) }

// rtpFrame builds an Ethernet/IPv4/UDP frame carrying an RTP header.
func rtpFrame(t *testing.T, srcPort uint16, pt uint8, ssrc uint32, seq uint16, ts uint32, vlan bool) []byte {
	t.Helper()
	rtp := make([]byte, rtpHeaderLen+160)
	rtp[0] = rtpVersion << 6
	rtp[1] = pt
	binary.BigEndian.PutUint16(rtp[2:4], seq)
	binary.BigEndian.PutUint32(rtp[4:8], ts)
	binary.BigEndian.PutUint32(rtp[8:12], ssrc)
	eth := &layers.Ethernet{
		SrcMAC:       net.HardwareAddr{0x02, 0, 0, 0, 0, 0x01},
		DstMAC:       net.HardwareAddr{0x02, 0, 0, 0, 0, 0x02},
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version: 4, TTL: 64, Protocol: layers.IPProtocolUDP,
		SrcIP: net.IPv4(192, 0, 2, 10), DstIP: net.IPv4(192, 0, 2, 20),
	}
	udp := &layers.UDP{SrcPort: layers.UDPPort(srcPort), DstPort: 30000}
	if err := udp.SetNetworkLayerForChecksum(ip); err != nil {
		t.Fatal(err)
	}
	stack := []gopacket.SerializableLayer{eth}
	if vlan {
		eth.EthernetType = layers.EthernetTypeDot1Q
		stack = append(stack, &layers.Dot1Q{VLANIdentifier: 200, Type: layers.EthernetTypeIPv4})
	}
	stack = append(stack, ip, udp, gopacket.Payload(rtp))
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, stack...); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// call describes the replayed stream the Python reference also models: n
// packets of 20 ms audio, sequence numbers from 65000 and timestamps from
// 4294960000 so both wrap, every dropEvery-th packet from the eleventh on
// lost at phase dropPhase, and arrivals alternating jitterMs early and late.
type call struct {
	pt                   uint8
	n                    int
	dropEvery, dropPhase int
	jitterMs             float64
	vlan                 bool
}

func (c call) frames(t *testing.T) []capturetest.Frame {
	t.Helper()
	var out []capturetest.Frame
	for i := range c.n {
		if i >= probationPackets && i%c.dropEvery == c.dropPhase {
			continue
		}
		offset := time.Duration(c.jitterMs * float64(time.Millisecond))
		if i%2 == 1 {
			offset = -offset
		}
		data := rtpFrame(t, 40000, c.pt, 0x1234abcd, uint16(65000+i), uint32(4294960000+160*i), c.vlan)
		out = append(out, capturetest.Frame{
			Data: data,
			Info: gopacket.CaptureInfo{
				Timestamp:     t0().Add(time.Duration(i)*20*time.Millisecond + offset),
				CaptureLength: len(data), Length: len(data),
			},
		})
	}
	return out
}

func run(t *testing.T, frames []capturetest.Frame) ([]Report, []listener.Event) {
	t.Helper()
	store, sink := &memStore{}, &memSink{}
	l, err := New(Config{
		Interface: "eth0",
		Opener:    &capturetest.ReplayOpener{LinkType: layers.LinkTypeEthernet, Frames: frames},
		Store:     store,
		Sink:      sink,
		// The replayed capture is long past, so every stream is idle.
		Now: func() time.Time { return t0().Add(time.Hour) },
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if startErr := l.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	deadline := time.Now().Add(5 * time.Second)
	for l.Stats().Packets < int64(len(frames)) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	// One read timeout sweeps the idle streams.
	time.Sleep(3 * readTimeout)
	if stopErr := l.Stop(ctx); stopErr != nil {
		t.Fatal(stopErr)
	}
	return store.all(), sink.all()
}

func near(got, want, tol float64) bool { return math.Abs(got-want) <= tol }

// reference is a stream's expected score from the Python reference.
type reference struct {
	expected, received int64
	lossPct, jitterMs  float64
	mos                float64
}

func checkReport(t *testing.T, r Report, want reference) {
	t.Helper()
	if r.Expected != want.expected || r.Received != want.received {
		t.Errorf("expected/received = %d/%d, want %d/%d", r.Expected, r.Received, want.expected, want.received)
	}
	if !near(r.LossPct, want.lossPct, 0.001) || !near(r.JitterMs, want.jitterMs, 0.01) {
		t.Errorf("loss %.3f %%, jitter %.3f ms; want %.3f %%, %.3f ms",
			r.LossPct, r.JitterMs, want.lossPct, want.jitterMs)
	}
	if !near(r.MOS, want.mos, 0.05) {
		t.Errorf("MOS %.3f, want %.3f ± 0.05", r.MOS, want.mos)
	}
	if r.Src.String() != "192.0.2.10:40000" || r.Dst.String() != "192.0.2.20:30000" || r.SSRC != 0x1234abcd {
		t.Errorf("stream %s -> %s ssrc %x", r.Src, r.Dst, r.SSRC)
	}
}

func checkEvents(t *testing.T, events []listener.Event, r Report, want int) {
	t.Helper()
	if len(events) != want {
		t.Fatalf("got %d events, want %d", len(events), want)
	}
	for _, evt := range events {
		var q listener.VoIPQuality
		if err := json.Unmarshal(evt.Payload, &q); err != nil {
			t.Fatal(err)
		}
		if evt.Kind != listener.VoIPQualityKind || evt.SourceAddr != "192.0.2.10" || !near(q.MOS, r.MOS, 1e-9) {
			t.Errorf("event %+v payload %+v", evt, q)
		}
	}
}

// TestKnownStreamScoresWithinToleranceOfReference is the row's acceptance:
// a known RTP stream with injected loss and jitter scores within 0.05 MOS
// of the reference, and only the poor one alerts. The want values come
// from an independent Python simulation of the same streams.
func TestKnownStreamScoresWithinToleranceOfReference(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		call   call
		want   reference
		alerts int
	}{
		{
			name: "G.711 2 percent loss 2 ms jitter",
			call: call{pt: ptPCMU, n: 3000, dropEvery: 50, dropPhase: 25, jitterMs: 2},
			want: reference{expected: 2991, received: 2931, lossPct: 2.006, jitterMs: 3.898, mos: 4.207},
		},
		{
			name:   "G.729 10 percent loss 8 ms jitter on a VLAN",
			call:   call{pt: ptG729, n: 3000, dropEvery: 10, dropPhase: 5, jitterMs: 8, vlan: true},
			want:   reference{expected: 2991, received: 2692, lossPct: 9.997, jitterMs: 14.143, mos: 2.665},
			alerts: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			reports, events := run(t, tc.call.frames(t))
			if len(reports) != 1 {
				t.Fatalf("got %d reports, want 1: %+v", len(reports), reports)
			}
			checkReport(t, reports[0], tc.want)
			checkEvents(t, events, reports[0], tc.alerts)
		})
	}
}

func TestLongCallReportsEachInterval(t *testing.T) {
	t.Parallel()
	// 150 s of clean audio: two full windows and a 30 s tail.
	reports, events := run(t, call{pt: ptPCMA, n: 7500, dropEvery: 7500, dropPhase: 1}.frames(t))
	if len(reports) != 3 {
		t.Fatalf("got %d reports, want 3", len(reports))
	}
	var expected int64
	for _, r := range reports {
		expected += r.Expected
		if r.LossPct != 0 || r.MOS < 4.3 {
			t.Errorf("clean window scored loss %.2f MOS %.2f", r.LossPct, r.MOS)
		}
	}
	if want := int64(7500 - probationPackets + 1); expected != want {
		t.Errorf("windows expected %d packets, want %d", expected, want)
	}
	if len(events) != 0 {
		t.Errorf("clean call published %d events", len(events))
	}
}

func TestNonRTPTrafficIsIgnored(t *testing.T) {
	t.Parallel()
	var frames []capturetest.Frame
	for i := range 500 {
		at := t0().Add(time.Duration(i) * 20 * time.Millisecond)
		// A dynamic payload type is RTP but cannot be scored.
		dynamic := rtpFrame(t, 40002, 111, 7, uint16(i), uint32(960*i), false)
		// Version 2 in the first byte, but the sequence numbers wander.
		noise := rtpFrame(t, 40004, 0, 9, uint16(i*7919), uint32(i), false)
		for _, d := range [][]byte{dynamic, noise} {
			frames = append(frames, capturetest.Frame{
				Data: d, Info: gopacket.CaptureInfo{Timestamp: at, CaptureLength: len(d), Length: len(d)},
			})
		}
	}
	if reports, _ := run(t, frames); len(reports) != 0 {
		t.Fatalf("got %d reports from non-RTP traffic: %+v", len(reports), reports)
	}
}

func TestStreamSequenceHandling(t *testing.T) {
	t.Parallel()
	codec, _ := codecFor(ptPCMU)
	start := t0()
	confirmed := func() *stream {
		s := newStream(streamKey{ssrc: 1}, 0, codec, 100, start)
		for i := 1; i < probationPackets; i++ {
			s.add("eth0", 0, uint16(100+i), uint32(160*i), start.Add(time.Duration(i)*20*time.Millisecond))
		}
		if !s.confirmed() {
			t.Fatal("stream not confirmed after probation")
		}
		return s
	}
	at := func(i int) time.Time { return start.Add(time.Duration(i) * 20 * time.Millisecond) }

	t.Run("reordered packet is counted once", func(t *testing.T) {
		t.Parallel()
		s := confirmed()
		s.add("eth0", 0, 111, 160*11, at(11))
		s.add("eth0", 0, 110, 160*10, at(12))
		if s.expected() != 3 || s.received != 3 {
			t.Errorf("expected/received %d/%d, want 3/3", s.expected(), s.received)
		}
	})
	t.Run("restart needs two sequential packets", func(t *testing.T) {
		t.Parallel()
		s := confirmed()
		s.add("eth0", 0, 30000, 0, at(10))
		if s.received != 1 {
			t.Fatalf("a lone jump was counted: received %d", s.received)
		}
		s.add("eth0", 0, 30001, 160, at(11))
		if s.expected() != 1 || s.received != 1 {
			t.Errorf("after restart expected/received %d/%d, want 1/1", s.expected(), s.received)
		}
	})
	t.Run("other payload type on the SSRC is skipped", func(t *testing.T) {
		t.Parallel()
		s := confirmed()
		s.add("eth0", 13, 110, 160*10, at(10))
		if s.received != 1 || s.maxSeq != 109 {
			t.Errorf("comfort noise counted: received %d max %d", s.received, s.maxSeq)
		}
	})
	t.Run("out-of-sequence packet restarts probation", func(t *testing.T) {
		t.Parallel()
		s := newStream(streamKey{ssrc: 2}, 0, codec, 1, start)
		s.add("eth0", 0, 2, 160, at(1))
		s.add("eth0", 0, 9, 320, at(2))
		if s.probation != probationPackets-1 {
			t.Errorf("probation %d, want %d", s.probation, probationPackets-1)
		}
	})
}
