package flow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"
)

// Name is the listener key in the engine registry.
const Name = "flow-collector"

const (
	defaultBindAddr = ":2055"
	// maxDatagram is the largest UDP payload; exporters keep under the
	// path MTU, but a jumbo-frame exporter may not.
	maxDatagram = 65535
	readTimeout = time.Second

	// queueSize bounds the records waiting for the store. At 30 flows a
	// datagram it holds about 270 full v5 datagrams; a burst beyond what
	// the store absorbs is dropped and counted.
	queueSize = 8192
	// batchSize and flushInterval trade write amplification against how
	// long a flow waits before it is queryable.
	batchSize     = 512
	flushInterval = time.Second
	// statsInterval paces the summary of drops and decode failures.
	statsInterval = time.Minute
)

// Store persists decoded flows. The database implementation writes one
// batch per call in one transaction.
type Store interface {
	InsertFlows(ctx context.Context, records []Record) error
}

// Stats counts what the listener has seen since it started.
type Stats struct {
	Datagrams int64
	Records   int64
	// Dropped counts records discarded because the queue was full.
	Dropped int64
	// MissingTemplate counts data sets that arrived before their
	// template, or after it expired.
	MissingTemplate int64
	// Malformed counts datagrams that failed to decode, whole or part.
	Malformed int64
	// StoreErrors counts batches the store rejected.
	StoreErrors int64
}

// Config configures a Listener.
type Config struct {
	// BindAddr defaults to ":2055", the conventional NetFlow port.
	BindAddr string
	Store    Store
	Logger   *slog.Logger
	Now      func() time.Time
	// QueueSize overrides the record queue bound; tests use it to force
	// a flood.
	QueueSize int
}

// Listener receives NetFlow v5, v9 and IPFIX on one UDP socket.
type Listener struct {
	bindAddr  string
	store     Store
	logger    *slog.Logger
	now       func() time.Time
	queueSize int

	datagrams, records, dropped        atomic.Int64
	missingTemplate, malformed, failed atomic.Int64

	mu      sync.Mutex
	conn    net.PacketConn
	addr    net.Addr
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New returns an unstarted Listener.
func New(cfg Config) (*Listener, error) {
	if cfg.Store == nil {
		return nil, errors.New("flow: Store required")
	}
	if cfg.BindAddr == "" {
		cfg.BindAddr = defaultBindAddr
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = queueSize
	}
	return &Listener{
		bindAddr:  cfg.BindAddr,
		store:     cfg.Store,
		logger:    cfg.Logger,
		now:       cfg.Now,
		queueSize: cfg.QueueSize,
	}, nil
}

// Name implements [listener.Listener].
func (*Listener) Name() string { return Name }

// Addr is the bound address, or nil before Start.
func (l *Listener) Addr() net.Addr {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.addr
}

// Stats returns the counters since Start.
func (l *Listener) Stats() Stats {
	return Stats{
		Datagrams:       l.datagrams.Load(),
		Records:         l.records.Load(),
		Dropped:         l.dropped.Load(),
		MissingTemplate: l.missingTemplate.Load(),
		Malformed:       l.malformed.Load(),
		StoreErrors:     l.failed.Load(),
	}
}

// Start binds the socket and starts the read and write loops. A second
// Start without a Stop is a no-op.
func (l *Listener) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.started {
		return nil
	}
	lc := &net.ListenConfig{}
	conn, err := lc.ListenPacket(ctx, "udp", l.bindAddr)
	if err != nil {
		return fmt.Errorf("flow: bind %s: %w", l.bindAddr, err)
	}
	loopCtx, cancel := context.WithCancel(ctx)
	l.conn, l.addr, l.cancel, l.started = conn, conn.LocalAddr(), cancel, true

	queue := make(chan Record, l.queueSize)
	l.wg.Go(func() { l.read(loopCtx, conn, queue) })
	l.wg.Go(func() { l.write(loopCtx, queue) })
	l.logger.InfoContext(ctx, "flow collector started", "addr", l.addr.String())
	return nil
}

// Stop closes the socket, flushes queued records and waits for both
// loops, up to the ctx deadline.
func (l *Listener) Stop(ctx context.Context) error {
	l.mu.Lock()
	if !l.started {
		l.mu.Unlock()
		return nil
	}
	l.started = false
	conn, cancel := l.conn, l.cancel
	l.conn = nil
	l.mu.Unlock()

	_ = conn.Close()
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
	l.logger.InfoContext(ctx, "flow collector stopped", "addr", l.bindAddr)
	return nil
}

// read owns the socket and the decoder. It never blocks on the store:
// a record that does not fit the queue is dropped and counted.
func (l *Listener) read(ctx context.Context, conn net.PacketConn, queue chan<- Record) {
	defer close(queue)
	dec := NewDecoder()
	buf := make([]byte, maxDatagram)
	for ctx.Err() == nil {
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
		n, from, err := conn.ReadFrom(buf)
		if err != nil {
			var nerr net.Error
			if errors.As(err, &nerr) && nerr.Timeout() {
				continue
			}
			if ctx.Err() == nil && !errors.Is(err, net.ErrClosed) {
				l.logger.WarnContext(ctx, "flow: read failed", "error", err)
				continue
			}
			return
		}
		l.handle(ctx, dec, from, buf[:n], queue)
	}
}

func (l *Listener) handle(ctx context.Context, dec *Decoder, from net.Addr, pkt []byte, queue chan<- Record) {
	l.datagrams.Add(1)
	exporter := exporterAddr(from)
	res, err := dec.Decode(exporter, pkt, l.now())
	if err != nil {
		l.malformed.Add(1)
		l.logger.DebugContext(ctx, "flow: decode failed", "exporter", exporter.String(), "error", err)
	}
	l.missingTemplate.Add(int64(res.MissingTemplate))
	for i := range res.Records {
		select {
		case queue <- res.Records[i]:
			l.records.Add(1)
		default:
			l.dropped.Add(1)
		}
	}
}

// write drains the queue into the store in batches. It runs until read
// closes the queue, so records queued before Stop are still written.
func (l *Listener) write(ctx context.Context, queue <-chan Record) {
	flush := time.NewTicker(flushInterval)
	defer flush.Stop()
	report := time.NewTicker(statsInterval)
	defer report.Stop()

	// The store write outlives the loop context so the final flush on
	// Stop is not cancelled before it runs.
	storeCtx := context.WithoutCancel(ctx)
	batch := make([]Record, 0, batchSize)
	var last Stats
	for {
		select {
		case rec, ok := <-queue:
			if !ok {
				l.flush(storeCtx, batch)
				return
			}
			batch = append(batch, rec)
			if len(batch) == batchSize {
				l.flush(storeCtx, batch)
				batch = batch[:0]
			}
		case <-flush.C:
			l.flush(storeCtx, batch)
			batch = batch[:0]
		case <-report.C:
			last = l.report(ctx, last)
		}
	}
}

func (l *Listener) flush(ctx context.Context, batch []Record) {
	if len(batch) == 0 {
		return
	}
	if err := l.store.InsertFlows(ctx, batch); err != nil {
		l.failed.Add(1)
		l.logger.WarnContext(ctx, "flow: store failed", "records", len(batch), "error", err)
	}
}

// report logs what was lost since the previous report, once a minute
// and only when something was.
func (l *Listener) report(ctx context.Context, last Stats) Stats {
	now := l.Stats()
	if now.Dropped == last.Dropped && now.MissingTemplate == last.MissingTemplate &&
		now.Malformed == last.Malformed && now.StoreErrors == last.StoreErrors {
		return now
	}
	l.logger.WarnContext(ctx, "flow collector lost records",
		"dropped_queue_full", now.Dropped-last.Dropped,
		"sets_missing_template", now.MissingTemplate-last.MissingTemplate,
		"malformed_datagrams", now.Malformed-last.Malformed,
		"store_errors", now.StoreErrors-last.StoreErrors,
	)
	return now
}

func exporterAddr(a net.Addr) netip.Addr {
	if u, ok := a.(*net.UDPAddr); ok {
		return u.AddrPort().Addr().Unmap()
	}
	return netip.Addr{}
}
