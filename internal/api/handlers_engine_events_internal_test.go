package api

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MustardSeedNetworks/seed/internal/discovery"
	"github.com/MustardSeedNetworks/seed/internal/discovery/devices"
)

// eventsEngine is a discovery engine that hands its single subscriber to the
// test, which then delivers events the way the discovery bus does: one
// goroutine per event.
type eventsEngine struct {
	devices.Engine

	subscribed chan func(*discovery.Event)
}

func (e *eventsEngine) Available() bool { return true }

func (e *eventsEngine) SubscribeAll(handler func(*discovery.Event)) string {
	e.subscribed <- handler
	return "sub-1"
}

func (e *eventsEngine) Unsubscribe(string) {}

// overlapWriter records a Write that starts while another is in progress, or
// after the handler has returned. Either corrupts the stream: frames
// interleave, or a finished response is written to.
type overlapWriter struct {
	*httptest.ResponseRecorder

	mu       sync.Mutex
	inWrite  atomic.Bool
	returned atomic.Bool
	overlaps atomic.Int32
	late     atomic.Int32
}

func (w *overlapWriter) Write(p []byte) (int, error) {
	if w.returned.Load() {
		w.late.Add(1)
	}
	if !w.inWrite.CompareAndSwap(false, true) {
		w.overlaps.Add(1)
	}
	defer w.inWrite.Store(false)
	time.Sleep(time.Millisecond)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ResponseRecorder.Write(p)
}

func (w *overlapWriter) body() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.Body.String()
}

// The discovery bus calls a subscriber on a fresh goroutine per event, so the
// stream handler must not write from the subscriber: every frame is written by
// the handler alone, whole, and nothing is written once it returns.
func TestEngineEventsSerializesFrames(t *testing.T) {
	t.Parallel()
	const events = 20

	engine := &eventsEngine{subscribed: make(chan func(*discovery.Event), 1)}
	s := &Server{discoveryDevices: devices.NewService(engine)}
	w := &overlapWriter{ResponseRecorder: httptest.NewRecorder()}
	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet,
		APIVersionPrefix+"/discovery/engine/events", http.NoBody)

	done := make(chan struct{})
	go func() {
		defer close(done)
		s.handleEngineEvents(w, req)
		w.returned.Store(true)
	}()

	deliver := <-engine.subscribed
	var wg sync.WaitGroup
	for range events {
		wg.Go(func() {
			deliver(discovery.NewEvent(discovery.EvtDeviceDiscovered, discovery.SourceWired))
		})
	}
	wg.Wait()

	deadline := time.Now().Add(5 * time.Second)
	for strings.Count(w.body(), "event: ") < events && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	cancel()
	<-done
	deliver(discovery.NewEvent(discovery.EvtDeviceLost, discovery.SourceWired))
	time.Sleep(20 * time.Millisecond)

	if n := w.overlaps.Load(); n != 0 {
		t.Errorf("%d writes overlapped another write", n)
	}
	if n := w.late.Load(); n != 0 {
		t.Errorf("%d writes after the handler returned", n)
	}
	if got := w.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Errorf("Content-Type = %q, want text/event-stream", got)
	}

	frames := 0
	sc := bufio.NewScanner(strings.NewReader(w.body()))
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, "event: ") {
			continue
		}
		if line != "event: "+string(discovery.EvtDeviceDiscovered) {
			t.Fatalf("frame event line = %q", line)
		}
		if !sc.Scan() || !strings.HasPrefix(sc.Text(), "data: ") {
			t.Fatalf("event line not followed by its data line: %q", sc.Text())
		}
		var ev discovery.Event
		if err := json.Unmarshal([]byte(strings.TrimPrefix(sc.Text(), "data: ")), &ev); err != nil {
			t.Fatalf("frame data is not one event: %v", err)
		}
		frames++
	}
	if frames != events {
		t.Errorf("frames = %d, want %d", frames, events)
	}
}
