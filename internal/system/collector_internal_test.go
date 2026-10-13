package system

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
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
//
// On darwin it also caught a sampler that measured over 100 ms windows while
// the OS advances its tick counters about once a second, so most readings saw
// no change and stored 0 (seed#2773).
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

// TestBusyPercent pins the arithmetic over two cumulative readings, and that a
// window in which the counters did not move reports no reading rather than 0%:
// that is the normal case on darwin for any window under a second (seed#2773).
func TestBusyPercent(t *testing.T) {
	base := cpu.TimesStat{User: 100, System: 50, Idle: 800, Iowait: 50}
	tests := []struct {
		name   string
		cur    cpu.TimesStat
		want   float64
		wantOK bool
	}{
		{"counters unchanged", base, 0, false},
		{"all idle", cpu.TimesStat{User: 100, System: 50, Idle: 900, Iowait: 50}, 0, true},
		{"all busy", cpu.TimesStat{User: 150, System: 100, Idle: 800, Iowait: 50}, 100, true},
		{"quarter busy", cpu.TimesStat{User: 110, System: 60, Idle: 860, Iowait: 50}, 25, true},
		{"iowait counts as idle", cpu.TimesStat{User: 110, System: 50, Idle: 820, Iowait: 120}, 10, true},
		{"counters went backwards", cpu.TimesStat{User: 10, System: 5, Idle: 80, Iowait: 5}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := busyPercent(base, tt.cur)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("busyPercent = (%v, %v), want (%v, %v)", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

// TestCollectorCloseStopsTheSampler pins that Close is not advisory. Without it
// a long-lived process accumulates one ticker goroutine per Collector, each
// waking every cpuTickerInterval to read the CPU counters.
//
// It used to compare [runtime.NumGoroutine] either side of Close. That number is
// process-global, so any goroutine another test happened to be running counted
// against this one — it failed on healthy code under -race (seed#2361), and it
// could equally pass while this collector leaked, as long as something else
// finished at the same moment.
//
// sampleCPU closes done as it returns, so that channel says precisely what this
// test means: the sampler goroutine is gone. Reading it after Close also proves
// Close waited, rather than merely signalling and returning.
func TestCollectorCloseStopsTheSampler(t *testing.T) {
	c := NewCollector()
	if err := c.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	select {
	case <-c.done:
	default:
		t.Error("Close returned while the sampler goroutine was still running")
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
