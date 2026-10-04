package flow_test

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/listener/flow"
)

// fakeStore records batches. While gate is non-nil, InsertFlows blocks
// until it is closed, standing in for a database that has fallen behind.
type fakeStore struct {
	mu      sync.Mutex
	records []flow.Record
	gate    chan struct{}
}

func (s *fakeStore) InsertFlows(_ context.Context, batch []flow.Record) error {
	if s.gate != nil {
		<-s.gate
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, batch...)
	return nil
}

func (s *fakeStore) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

func startListener(t *testing.T, store *fakeStore, queueSize int) (*flow.Listener, net.Conn) {
	t.Helper()
	l, err := flow.New(flow.Config{
		BindAddr:  "127.0.0.1:0",
		Store:     store,
		Logger:    slog.New(slog.DiscardHandler),
		QueueSize: queueSize,
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
	return l, conn
}

// v5Datagram is a full v5 export of 30 flows.
func v5Datagram() pkt {
	p := pkt{}.u16(5).u16(30).u32(1000).u32(uint32(exportAt().Unix())).u32(0).u32(1).u8(0).u8(0).u16(0)
	for range 30 {
		p = p.addr("10.0.0.1").addr("10.0.0.2").addr("0.0.0.0").u16(1).u16(2).
			u32(1).u32(64).u32(900).u32(1000).u16(1).u16(2).u8(0).u8(0).u8(17).u8(0).
			u16(0).u16(0).u8(0).u8(0).u16(0)
	}
	return p
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestListenerStoresDecodedFlows(t *testing.T) {
	t.Parallel()
	store := &fakeStore{}
	l, conn := startListener(t, store, 0)

	template := v9(3, 1000, v9Template())
	data := v9(3, 1000, set(256, v9Data("10.9.9.1", "10.9.9.2", 5000, 80, 700, 7, 900, 950)))
	for _, d := range []pkt{data, template, data, v5Datagram(), pkt{}.u16(99)} {
		if _, err := conn.Write(d); err != nil {
			t.Fatal(err)
		}
	}
	waitFor(t, "31 stored flows", func() bool { return store.count() == 31 })

	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	got := l.Stats()
	want := flow.Stats{Datagrams: 5, Records: 31, MissingTemplate: 1, Malformed: 1}
	if got != want {
		t.Errorf("stats = %+v, want %+v", got, want)
	}
}

// TestListenerFloodIsBounded holds the store while an exporter floods the
// socket. The read loop must keep reading and drop what does not fit the
// queue rather than block or buffer without limit; after the store
// recovers, exactly the queued records are written, Stop included.
func TestListenerFloodIsBounded(t *testing.T) {
	t.Parallel()
	const queueSize, datagrams = 64, 200
	store := &fakeStore{gate: make(chan struct{})}
	l, conn := startListener(t, store, queueSize)

	d := v5Datagram()
	for range datagrams {
		if _, err := conn.Write(d); err != nil {
			t.Fatal(err)
		}
	}
	// The reader keeps reading with nothing being written. Loopback UDP
	// may itself drop part of the flood, so wait for the drops to start
	// and the reader to go quiet rather than for every datagram.
	waitFor(t, "queue-full drops", func() bool { return l.Stats().Dropped > 0 })
	waitFor(t, "the reader to go quiet", func() bool {
		before := l.Stats().Datagrams
		time.Sleep(100 * time.Millisecond)
		return l.Stats().Datagrams == before
	})
	stats := l.Stats()
	// The writer holds at most one batch it took before blocking, so the
	// records accepted are bounded by the queue plus that batch.
	if stats.Records > queueSize+512 {
		t.Errorf("accepted %d records with the store blocked, bound is %d", stats.Records, queueSize+512)
	}

	close(store.gate)
	if err := l.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	stats = l.Stats()
	if int64(store.count()) != stats.Records {
		t.Errorf("stored %d records, accepted %d", store.count(), stats.Records)
	}
	if stats.Records+stats.Dropped != 30*stats.Datagrams {
		t.Errorf("records %d + dropped %d != 30 x %d datagrams", stats.Records, stats.Dropped, stats.Datagrams)
	}
}
