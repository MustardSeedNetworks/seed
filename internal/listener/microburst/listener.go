package microburst

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopacket/gopacket/layers"

	"github.com/MustardSeedNetworks/seed/internal/capture"
)

// Name is the listener key in the engine registry.
const Name = "microburst"

// SamplingMode is stored with every event so a reader knows how it was
// measured.
const SamplingMode = "capture-1ms"

const (
	// snaplen covers the Ethernet header; only the source MAC and the
	// frame's original length are read.
	snaplen     = 64
	readTimeout = 100 * time.Millisecond
	// flushLag is how long a bin stays open after it ends when the link
	// goes quiet, so frames libpcap delivers late still land in it.
	flushLag = time.Second

	queueSize     = 256
	flushInterval = time.Second
	statsInterval = time.Minute
)

// Store persists bursts.
type Store interface {
	InsertMicrobursts(ctx context.Context, events []Event) error
}

// Stats counts what the listener has seen since it started.
type Stats struct {
	Frames int64
	Bursts int64
	// Dropped counts bursts discarded because the queue was full.
	Dropped     int64
	StoreErrors int64
}

// Config configures a Listener.
type Config struct {
	// Interface is the capture interface, the probe's own link.
	Interface string
	// MAC is the interface's address; frames sourced from it are outbound.
	MAC net.HardwareAddr
	// LinkSpeedMbps is the negotiated link speed that utilization is
	// measured against.
	LinkSpeedMbps int64
	Opener        capture.Opener
	Store         Store
	Logger        *slog.Logger
	Now           func() time.Time
}

// Listener captures on one interface and records microbursts.
type Listener struct {
	cfg Config

	frames, bursts, dropped, failed atomic.Int64

	mu      sync.Mutex
	started bool
	cancel  context.CancelFunc
	wg      sync.WaitGroup
}

// New validates cfg and returns an unstarted Listener.
func New(cfg Config) (*Listener, error) {
	switch {
	case cfg.Interface == "":
		return nil, errors.New("microburst: Interface required")
	case len(cfg.MAC) != len(layers.EthernetBroadcast):
		return nil, fmt.Errorf("microburst: %s has no Ethernet address", cfg.Interface)
	case cfg.LinkSpeedMbps <= 0:
		return nil, fmt.Errorf("microburst: %s reports no link speed", cfg.Interface)
	case cfg.Opener == nil:
		return nil, errors.New("microburst: Opener required")
	case cfg.Store == nil:
		return nil, errors.New("microburst: Store required")
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
		Frames:      l.frames.Load(),
		Bursts:      l.bursts.Load(),
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
	handle, err := l.cfg.Opener.OpenLive(l.cfg.Interface, snaplen, false, readTimeout)
	if err != nil {
		return fmt.Errorf("microburst: open %s: %w", l.cfg.Interface, err)
	}
	if lt := handle.LinkType(); lt != layers.LinkTypeEthernet {
		handle.Close()
		return fmt.Errorf("microburst: %s link type %s is not Ethernet", l.cfg.Interface, lt)
	}
	loopCtx, cancel := context.WithCancel(ctx)
	l.cancel, l.started = cancel, true

	queue := make(chan Event, queueSize)
	l.wg.Go(func() {
		defer handle.Close()
		l.read(loopCtx, handle, queue)
	})
	l.wg.Go(func() { l.write(loopCtx, queue) })
	l.cfg.Logger.InfoContext(ctx, "microburst listener started",
		"interface", l.cfg.Interface, "link_speed_mbps", l.cfg.LinkSpeedMbps)
	return nil
}

// Stop ends the capture, stores the bursts already found and waits for
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
	l.cfg.Logger.InfoContext(ctx, "microburst listener stopped", "interface", l.cfg.Interface)
	return nil
}

// read owns the handle and the detector. It never blocks on the store: a
// burst that does not fit the queue is dropped and counted.
func (l *Listener) read(ctx context.Context, handle capture.Handle, queue chan<- Event) {
	defer close(queue)
	det := newDetector(l.cfg.Interface, l.cfg.LinkSpeedMbps)
	emit := func(events []Event) {
		for _, e := range events {
			select {
			case queue <- e:
				l.bursts.Add(1)
			default:
				l.dropped.Add(1)
			}
		}
	}
	for ctx.Err() == nil {
		data, ci, err := handle.ReadPacketData()
		switch {
		case err == nil:
			l.frames.Add(1)
			emit(det.add(l.direction(data), ci.Timestamp, ci.Length))
		case errors.Is(err, capture.ErrTimeout):
			// Only a quiet link needs flushing; on a busy one the next
			// frame closes the bins before it.
			emit(det.flush(l.cfg.Now().Add(-flushLag)))
		default:
			l.cfg.Logger.WarnContext(ctx, "microburst capture read failed",
				"interface", l.cfg.Interface, "error", err)
			emit(det.finish())
			return
		}
	}
	emit(det.finish())
}

func (l *Listener) direction(frame []byte) Direction {
	const srcStart, srcEnd = 6, 12
	if len(frame) >= srcEnd && bytes.Equal(frame[srcStart:srcEnd], l.cfg.MAC) {
		return DirectionOut
	}
	return DirectionIn
}

// write batches bursts into the store until the queue closes.
func (l *Listener) write(ctx context.Context, queue <-chan Event) {
	flush := time.NewTicker(flushInterval)
	defer flush.Stop()
	stats := time.NewTicker(statsInterval)
	defer stats.Stop()
	var batch []Event
	store := func() {
		if len(batch) == 0 {
			return
		}
		// The loop context is cancelled on Stop; the final batch still
		// has to land.
		if err := l.cfg.Store.InsertMicrobursts(context.WithoutCancel(ctx), batch); err != nil {
			l.failed.Add(1)
			l.cfg.Logger.WarnContext(ctx, "microburst store failed", "bursts", len(batch), "error", err)
		}
		batch = nil
	}
	for {
		select {
		case e, ok := <-queue:
			if !ok {
				store()
				return
			}
			batch = append(batch, e)
		case <-flush.C:
			store()
		case <-stats.C:
			s := l.Stats()
			l.cfg.Logger.DebugContext(ctx, "microburst listener stats",
				"frames", s.Frames, "bursts", s.Bursts, "dropped", s.Dropped, "store_errors", s.StoreErrors)
		}
	}
}
