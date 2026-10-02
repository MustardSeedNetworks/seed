package checkers

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// Go's dialer starts one goroutine per address family when a host has both A
// and AAAA records, and each calls ConnectStart and ConnectDone (#2630). Run
// under -race, this fails if the hooks write the trace without a lock.
func TestHTTPTraceHooksTolerateParallelDials(t *testing.T) {
	t.Parallel()
	h := &httpTrace{}
	trace := h.clientTrace()

	var wg sync.WaitGroup
	for _, network := range []string{"tcp4", "tcp6"} {
		wg.Go(func() {
			trace.ConnectStart(network, "addr")
			trace.ConnectDone(network, "addr", nil)
		})
	}
	wg.Go(func() { _ = h.timings(time.Now()) })
	wg.Wait()
}

// The losing family's dial is cancelled and reports an error after the winner
// connected; the tcp phase must still be the winner's, not the loser's
// failure, and a connect that never succeeded must not report a tcp time.
func TestHTTPTraceTCPPhaseIsTheSuccessfulDial(t *testing.T) {
	t.Parallel()
	h := &httpTrace{}
	trace := h.clientTrace()
	start := time.Now()

	trace.ConnectStart("tcp6", "[::1]:443")
	trace.ConnectStart("tcp4", "127.0.0.1:443")
	time.Sleep(5 * time.Millisecond)
	trace.ConnectDone("tcp4", "127.0.0.1:443", nil)
	won := time.Since(start)
	time.Sleep(50 * time.Millisecond)
	trace.ConnectDone("tcp6", "[::1]:443", errors.New("operation was canceled"))

	tcp, ok := h.timings(start)["tcp"]
	if !ok {
		t.Fatal("no tcp phase recorded for a connect that succeeded")
	}
	if got := time.Duration(tcp * float64(time.Millisecond)); got > won+time.Millisecond {
		t.Fatalf("tcp phase %v includes the cancelled dial; the winner connected after %v", got, won)
	}

	failed := &httpTrace{}
	failedTrace := failed.clientTrace()
	failedTrace.ConnectStart("tcp4", "127.0.0.1:443")
	failedTrace.ConnectDone("tcp4", "127.0.0.1:443", errors.New("connection refused"))
	if _, measured := failed.timings(start)["tcp"]; measured {
		t.Fatal("a connect that failed reported a tcp time")
	}
}
