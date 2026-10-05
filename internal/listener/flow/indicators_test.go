package flow_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/indicators"
	"github.com/MustardSeedNetworks/seed/internal/listener"
	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

type fakeIndicators struct{ list *indicators.List }

func (f fakeIndicators) FlowIndicators(context.Context) (*indicators.List, error) {
	return f.list, nil
}

type failingStore struct{}

func (failingStore) InsertFlows(context.Context, []flow.Record) error {
	return errors.New("database is locked")
}

type recordingSink struct {
	mu     sync.Mutex
	events []listener.Event
}

func (s *recordingSink) Publish(_ context.Context, evt listener.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, evt)
	return nil
}

func (s *recordingSink) snapshot() []listener.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.events)
}

// v5Flow is one flow of a v5 export: addresses, bytes and packets.
type v5Flow struct {
	src, dst       string
	bytes, packets uint32
}

// v5Export builds a v5 datagram whose flows start 100 ms before and end at
// exportAt.
func v5Export(flows ...v5Flow) pkt {
	p := pkt{}.u16(5).u16(uint16(len(flows))).u32(1000).u32(uint32(exportAt().Unix())).u32(0).u32(1).u8(0).u8(0).u16(0)
	for _, f := range flows {
		p = p.addr(f.src).addr(f.dst).addr("0.0.0.0").u16(1).u16(2).
			u32(f.packets).u32(f.bytes).u32(900).u32(1000).u16(50000).u16(443).u8(0).u8(0).u8(6).u8(0).
			u16(0).u16(0).u8(0).u8(0).u16(0)
	}
	return p
}

func startIndicatorListener(
	t *testing.T,
	store flow.Store,
	entries ...string,
) (*flow.Listener, *recordingSink, net.Conn) {
	t.Helper()
	list, err := indicators.New(entries)
	if err != nil {
		t.Fatal(err)
	}
	sink := &recordingSink{}
	l, err := flow.New(flow.Config{
		BindAddr:   "127.0.0.1:0",
		Store:      store,
		Indicators: fakeIndicators{list: list},
		Sink:       sink,
		Logger:     slog.New(slog.DiscardHandler),
	})
	if err != nil {
		t.Fatal(err)
	}
	if startErr := l.Start(t.Context()); startErr != nil {
		t.Fatal(startErr)
	}
	conn, err := net.Dial("udp", l.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return l, sink, conn
}

func decodeHits(t *testing.T, events []listener.Event) []listener.FlowIndicatorHit {
	t.Helper()
	hits := make([]listener.FlowIndicatorHit, 0, len(events))
	for _, evt := range events {
		if evt.Kind != listener.FlowIndicatorKind {
			t.Fatalf("event kind = %q, want %q", evt.Kind, listener.FlowIndicatorKind)
		}
		var h listener.FlowIndicatorHit
		if err := json.Unmarshal(evt.Payload, &h); err != nil {
			t.Fatal(err)
		}
		if evt.SourceAddr != h.Host {
			t.Errorf("event source %q, want the host %q", evt.SourceAddr, h.Host)
		}
		hits = append(hits, h)
	}
	return hits
}

// One batch holds two flows each way between 10.0.0.5 and a listed address,
// a flow from another host to an address inside a listed prefix, and
// traffic that matches nothing. The expected events are worked by hand:
// one per host and listed address, totals summed over both directions.
func TestListenerPublishesIndicatorHits(t *testing.T) {
	t.Parallel()
	l, sink, conn := startIndicatorListener(t, &fakeStore{}, "198.51.100.7", "203.0.113.0/24")

	d := v5Export(
		v5Flow{"10.0.0.5", "198.51.100.7", 1000, 10},
		v5Flow{"198.51.100.7", "10.0.0.5", 4000, 8},
		v5Flow{"192.168.1.20", "203.0.113.99", 300, 3},
		v5Flow{"10.0.0.5", "198.51.100.8", 500, 5},
		v5Flow{"10.0.0.5", "10.0.0.6", 700, 7},
	)
	if _, err := conn.Write(d); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "two indicator events", func() bool { return len(sink.snapshot()) == 2 })
	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}

	start, end := exportAt().Add(-100*time.Millisecond), exportAt()
	want := []listener.FlowIndicatorHit{
		{
			Host: "10.0.0.5", Listed: "198.51.100.7", Indicator: "198.51.100.7/32", Exporter: "127.0.0.1",
			Flows: 2, Bytes: 5000, Packets: 18, FirstSeen: start, LastSeen: end,
		},
		{
			Host: "192.168.1.20", Listed: "203.0.113.99", Indicator: "203.0.113.0/24", Exporter: "127.0.0.1",
			Flows: 1, Bytes: 300, Packets: 3, FirstSeen: start, LastSeen: end,
		},
	}
	got := decodeHits(t, sink.snapshot())
	for i := range got {
		got[i].FirstSeen, got[i].LastSeen = got[i].FirstSeen.UTC(), got[i].LastSeen.UTC()
	}
	if !slices.Equal(got, want) {
		t.Errorf("hits =\n%+v\nwant\n%+v", got, want)
	}
}

// A store that refuses every batch must not hide traffic to a listed
// address.
func TestListenerPublishesIndicatorHitsWhenTheStoreFails(t *testing.T) {
	t.Parallel()
	l, sink, conn := startIndicatorListener(t, failingStore{}, "198.51.100.7")
	if _, err := conn.Write(v5Export(v5Flow{"10.0.0.5", "198.51.100.7", 1000, 10})); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "an indicator event", func() bool { return len(sink.snapshot()) == 1 })
	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if l.Stats().StoreErrors == 0 {
		t.Error("the store failure was not counted")
	}
}

func TestListenerWithoutIndicatorsPublishesNothing(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	l, sink, conn := startIndicatorListener(t, store)
	if _, err := conn.Write(v5Export(v5Flow{"10.0.0.5", "198.51.100.7", 1000, 10})); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "the stored flow", func() bool { return store.count() == 1 })
	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := len(sink.snapshot()); n != 0 {
		t.Errorf("published %d events with an empty list", n)
	}
}

func TestNewRequiresIndicatorsAndSinkTogether(t *testing.T) {
	t.Parallel()
	_, err := flow.New(flow.Config{Store: &fakeStore{}, Sink: &recordingSink{}})
	if err == nil {
		t.Error("a Sink without Indicators was accepted")
	}
	_, err = flow.New(flow.Config{Store: &fakeStore{}, Indicators: fakeIndicators{}})
	if err == nil {
		t.Error("Indicators without a Sink was accepted")
	}
}
