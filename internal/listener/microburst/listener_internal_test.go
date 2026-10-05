package microburst

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture/capturetest"
)

func ownMAC() net.HardwareAddr  { return net.HardwareAddr{0x02, 0, 0, 0, 0, 0x01} }
func peerMAC() net.HardwareAddr { return net.HardwareAddr{0x02, 0, 0, 0, 0, 0x02} }

type memStore struct {
	mu     sync.Mutex
	events []Event
}

func (s *memStore) InsertMicrobursts(_ context.Context, events []Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, events...)
	return nil
}

func (s *memStore) all() []Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Event(nil), s.events...)
}

// ethernet returns a frame header with src as the source address.
func ethernet(src net.HardwareAddr) []byte {
	b := make([]byte, 14)
	copy(b[0:6], peerMAC())
	copy(b[6:12], src)
	return b
}

func replay(frames []frame) []capturetest.Frame {
	out := make([]capturetest.Frame, len(frames))
	for i, f := range frames {
		src := peerMAC()
		if f.dir == DirectionOut {
			src = ownMAC()
		}
		out[i] = capturetest.Frame{
			Data: ethernet(src),
			Info: gopacket.CaptureInfo{Timestamp: f.ts, CaptureLength: 14, Length: f.len},
		}
	}
	return out
}

func TestListenerRecordsBurstsByDirection(t *testing.T) {
	t.Parallel()
	opener := &capturetest.ReplayOpener{
		LinkType: layers.LinkTypeEthernet,
		Frames: replay(concat(
			paced(DirectionIn, t0(), 3*time.Millisecond, 1),
			paced(DirectionOut, t0().Add(5*time.Millisecond), 4*time.Millisecond, 1),
		)),
	}
	store := &memStore{}
	l, err := New(Config{
		Interface: "eth0", MAC: ownMAC(), LinkSpeedMbps: speed1G,
		Opener: opener, Store: store,
		// The replayed capture is long past, so every bin is flushable.
		Now: func() time.Time { return t0().Add(time.Minute) },
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := t.Context()
	if startErr := l.Start(ctx); startErr != nil {
		t.Fatal(startErr)
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(store.all()) < 2 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if stopErr := l.Stop(ctx); stopErr != nil {
		t.Fatal(stopErr)
	}

	got := store.all()
	if len(got) != 2 {
		t.Fatalf("stored %+v, want two bursts (stats %+v)", got, l.Stats())
	}
	want := []struct {
		start time.Time
		dir   Direction
		dur   time.Duration
	}{
		{t0(), DirectionIn, 3 * time.Millisecond},
		{t0().Add(5 * time.Millisecond), DirectionOut, 4 * time.Millisecond},
	}
	for i, w := range want {
		if !got[i].Start.Equal(w.start) || got[i].Direction != w.dir || got[i].Duration != w.dur {
			t.Errorf("burst %d = %s %s %s, want %s %s %s",
				i, got[i].Start, got[i].Direction, got[i].Duration, w.start, w.dir, w.dur)
		}
	}
	if s := l.Stats(); s.Bursts != 2 || s.Dropped != 0 || s.StoreErrors != 0 {
		t.Errorf("stats = %+v", s)
	}
}

func TestNewRejectsUnmeasurableInterface(t *testing.T) {
	t.Parallel()
	ok := Config{
		Interface: "eth0", MAC: ownMAC(), LinkSpeedMbps: speed1G,
		Opener: &capturetest.ReplayOpener{}, Store: &memStore{},
	}
	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{"no interface", func(c *Config) { c.Interface = "" }},
		{"no MAC", func(c *Config) { c.MAC = nil }},
		{"no link speed", func(c *Config) { c.LinkSpeedMbps = 0 }},
		{"no opener", func(c *Config) { c.Opener = nil }},
		{"no store", func(c *Config) { c.Store = nil }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ok
			tt.mutate(&cfg)
			if _, err := New(cfg); err == nil {
				t.Error("New accepted the config")
			}
		})
	}
}

func TestStartRejectsNonEthernetLink(t *testing.T) {
	t.Parallel()
	l, err := New(Config{
		Interface: "any", MAC: ownMAC(), LinkSpeedMbps: speed1G,
		Opener: &capturetest.ReplayOpener{LinkType: layers.LinkTypeLinuxSLL},
		Store:  &memStore{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if startErr := l.Start(t.Context()); startErr == nil {
		_ = l.Stop(t.Context())
		t.Fatal("Start accepted a Linux cooked capture")
	}
}
