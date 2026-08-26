package system

import (
	"runtime"
	"sync"
	"testing"
	"time"
)

// TestCollectorReusesItsCaches is the regression this package exists to prevent.
//
// The previous code obtained its state through
//
//	sync.OnceValue(func() *processCacheState { ... })()
//
// which builds a fresh once-wrapper and calls it, so every caller got its own
// zero-valued state. The cache never hit, the single-flight flag guarded
// nothing, and each call launched its own process enumeration. Under sustained
// polling that exhausted OS threads and aborted the process with
// "pthread_create failed: Resource temporarily unavailable".
//
// Asserting on goroutine count rather than on cache contents is deliberate: the
// leak is the part that took the process down, and it is what the old suite —
// which asserted only that the constructor returned non-nil — could not see.
func TestCollectorReusesItsCaches(t *testing.T) {
	c := NewCollector()
	t.Cleanup(func() { _ = c.Close() })

	// Let the sampler take its first reading so it is not counted as growth.
	time.Sleep(cpuSampleInterval + 50*time.Millisecond)

	before := runtime.NumGoroutine()
	for range 200 {
		_, _ = c.Health()
	}

	// One background refresh may legitimately be in flight; more than that means
	// the single-flight guard is not guarding.
	if grew := runtime.NumGoroutine() - before; grew > 1 {
		t.Errorf("200 Health calls started %d goroutines, want at most 1 — "+
			"the caches are being rebuilt per call", grew)
	}
}

// TestCollectorSamplesCPU pins that the sampler actually writes a reading back.
// The old code started a sampler per call and read the fresh state immediately,
// so CPUPercent was always exactly 0 — and because GetHealth only collected top
// processes above a 75% threshold, that also made the CPU branch dead code.
func TestCollectorSamplesCPU(t *testing.T) {
	c := NewCollector()
	t.Cleanup(func() { _ = c.Close() })

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if c.cpu.cpuPercent() > 0 {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("CPU percent never rose above zero within 5s; the sampler is not " +
		"writing back into the cache the Collector reads")
}

// TestCollectorCloseStopsTheSampler pins that Close is not advisory. Without it
// a long-lived process accumulates one ticker goroutine per Collector, each
// waking every cpuTickerInterval to burn cpuSampleInterval measuring CPU.
func TestCollectorCloseStopsTheSampler(t *testing.T) {
	before := runtime.NumGoroutine()

	c := NewCollector()
	time.Sleep(cpuSampleInterval + 50*time.Millisecond)
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if after := runtime.NumGoroutine(); after > before {
		t.Errorf("goroutines %d -> %d after Close; the sampler outlived it",
			before, after)
	}
}

// TestCollectorHealthIsConcurrencySafe drives Health from many goroutines at
// once, which is how it is actually used — the handler runs per request.
func TestCollectorHealthIsConcurrencySafe(t *testing.T) {
	c := NewCollector()
	t.Cleanup(func() { _ = c.Close() })

	var wg sync.WaitGroup
	wg.Add(20)
	for range 20 {
		go func() {
			defer wg.Done()
			h, err := c.Health()
			if err != nil {
				t.Errorf("Health: %v", err)
				return
			}
			if h.CPUPercent < 0 || h.CPUPercent > 100 {
				t.Errorf("CPUPercent = %v, outside 0..100", h.CPUPercent)
			}
		}()
	}
	wg.Wait()
}
