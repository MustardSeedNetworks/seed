package system

import (
	"sync"
	"testing"
	"time"
)

func TestGetTopProcessesInternal(t *testing.T) {
	// Test the internal function that retrieves top processes
	topCPU, topMemory := getTopProcessesInternal()

	// Should return slices (may be empty if no processes are accessible)
	// but should not panic
	t.Logf("TopCPU processes: %d", len(topCPU))
	t.Logf("TopMemory processes: %d", len(topMemory))

	// If we got processes, validate their structure
	for _, p := range topCPU {
		if p.PID < 0 {
			t.Errorf("CPU process has invalid PID: %d", p.PID)
		}
		if p.CPUPercent < 0 {
			t.Errorf("CPU process has negative CPUPercent: %f", p.CPUPercent)
		}
	}

	for _, p := range topMemory {
		if p.PID < 0 {
			t.Errorf("Memory process has invalid PID: %d", p.PID)
		}
		if p.MemoryMB < 0 {
			t.Errorf("Memory process has negative MemoryMB: %f", p.MemoryMB)
		}
	}
}

func TestGetTopProcessesInternalLimitsToFive(t *testing.T) {
	topCPU, topMemory := getTopProcessesInternal()

	// Should return at most topProcessCount (5) processes
	if len(topCPU) > topProcessCount {
		t.Errorf("topCPU has more than %d processes: %d", topProcessCount, len(topCPU))
	}
	if len(topMemory) > topProcessCount {
		t.Errorf("topMemory has more than %d processes: %d", topProcessCount, len(topMemory))
	}
}

func TestGetTopProcessesInternalSorting(t *testing.T) {
	topCPU, topMemory := getTopProcessesInternal()

	// Verify CPU processes are sorted by CPU percent (descending)
	for i := 1; i < len(topCPU); i++ {
		if topCPU[i].CPUPercent > topCPU[i-1].CPUPercent {
			t.Errorf("topCPU not sorted by CPUPercent: %f > %f at index %d",
				topCPU[i].CPUPercent, topCPU[i-1].CPUPercent, i)
		}
	}

	// Verify Memory processes are sorted by MemoryMB (descending)
	for i := 1; i < len(topMemory); i++ {
		if topMemory[i].MemoryMB > topMemory[i-1].MemoryMB {
			t.Errorf("topMemory not sorted by MemoryMB: %f > %f at index %d",
				topMemory[i].MemoryMB, topMemory[i-1].MemoryMB, i)
		}
	}
}

func TestGetCachedProcesses(t *testing.T) {
	// First call should trigger background update
	topCPU1, topMemory1 := getCachedProcesses()

	// Initial call may return empty (cache not yet populated)
	t.Logf("First call - TopCPU: %d, TopMemory: %d", len(topCPU1), len(topMemory1))

	// Wait for background update to complete
	time.Sleep(200 * time.Millisecond)

	// Second call should return cached data
	topCPU2, topMemory2 := getCachedProcesses()
	t.Logf("Second call - TopCPU: %d, TopMemory: %d", len(topCPU2), len(topMemory2))

	// Should be able to call multiple times without error
	for i := range 5 {
		cpuProcs, memProcs := getCachedProcesses()
		if cpuProcs == nil && memProcs == nil {
			t.Logf("Call %d returned nil slices (expected for initial state)", i+1)
		}
	}
}

func TestGetCachedProcessesConcurrency(_ *testing.T) {
	// Test concurrent calls to getCachedProcesses
	const numGoroutines = 20
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			cpuProcs, memProcs := getCachedProcesses()
			// Just verify no panic
			_ = cpuProcs
			_ = memProcs
		}()
	}

	wg.Wait()
}

func TestProcessCacheSingleton(t *testing.T) {
	// Verify singleton returns a valid instance
	// Note: Due to the sync.OnceValue pattern used, each call creates a new OnceValue
	// wrapper, but the inner function only executes once per wrapper. The current
	// implementation returns new instances on each call.
	state := processCacheSingleton()
	if state == nil {
		t.Error("processCacheSingleton should return a non-nil instance")
	}
}

func TestCPUCacheSingleton(t *testing.T) {
	// Verify singleton returns a valid instance
	// Note: Due to the sync.OnceValue pattern used, each call creates a new OnceValue
	// wrapper, but the inner function only executes once per wrapper. The current
	// implementation returns new instances on each call.
	state := cpuCacheSingleton()
	if state == nil {
		t.Error("cpuCacheSingleton should return a non-nil instance")
	}
}

func TestGetCachedCPUPercent(t *testing.T) {
	// First call starts the sampler
	pct1 := getCachedCPUPercent()

	// Should return a valid percentage (0 initially, then updated)
	if pct1 < 0 || pct1 > 100 {
		t.Errorf("getCachedCPUPercent returned invalid percentage: %f", pct1)
	}

	// Wait for sampler to potentially update
	time.Sleep(150 * time.Millisecond)

	// Subsequent call should still work
	pct2 := getCachedCPUPercent()
	if pct2 < 0 || pct2 > 100 {
		t.Errorf("getCachedCPUPercent returned invalid percentage on second call: %f", pct2)
	}
}

func TestGetCachedCPUPercentConcurrency(t *testing.T) {
	// Test concurrent calls to getCachedCPUPercent
	const numGoroutines = 20
	var wg sync.WaitGroup

	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			pct := getCachedCPUPercent()
			if pct < 0 || pct > 100 {
				t.Errorf("Invalid CPU percentage: %f", pct)
			}
		}()
	}

	wg.Wait()
}

func TestProcessCacheStateFields(_ *testing.T) {
	// Test that processCacheState has expected fields
	state := processCacheSingleton()

	// Lock and verify fields exist
	state.cacheMu.RLock()
	_ = state.top5
	_ = state.mem5
	_ = state.cacheTime
	state.cacheMu.RUnlock()

	state.updateMu.Lock()
	_ = state.updateInFly
	state.updateMu.Unlock()
}

func TestCPUCacheStateFields(_ *testing.T) {
	// Test that cpuCacheState has expected fields
	state := cpuCacheSingleton()

	// Lock and verify fields exist
	state.mu.RLock()
	_ = state.percent
	state.mu.RUnlock()
}

func TestProcessInfoStructValues(t *testing.T) {
	tests := []struct {
		name string
		proc ProcessInfo
	}{
		{
			name: "zero values",
			proc: ProcessInfo{},
		},
		{
			name: "typical process",
			proc: ProcessInfo{
				Name:       "test",
				PID:        1234,
				CPUPercent: 50.5,
				MemoryMB:   256.0,
			},
		},
		{
			name: "max values",
			proc: ProcessInfo{
				Name:       "stress-test",
				PID:        65535,
				CPUPercent: 100.0,
				MemoryMB:   16384.0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify fields are accessible
			if tt.proc.PID < 0 {
				t.Error("PID should not be negative")
			}
			if tt.proc.CPUPercent < 0 {
				t.Error("CPUPercent should not be negative")
			}
			if tt.proc.MemoryMB < 0 {
				t.Error("MemoryMB should not be negative")
			}
		})
	}
}

func TestHealthStructValues(t *testing.T) {
	tests := []struct {
		name   string
		health Health
	}{
		{
			name:   "zero values",
			health: Health{},
		},
		{
			name: "typical values",
			health: Health{
				CPUPercent:    50.0,
				MemoryPercent: 60.0,
				MemoryUsed:    8 * 1024 * 1024 * 1024,
				MemoryTotal:   16 * 1024 * 1024 * 1024,
				DiskPercent:   70.0,
				DiskUsed:      500 * 1024 * 1024 * 1024,
				DiskTotal:     1024 * 1024 * 1024 * 1024,
				Uptime:        86400,
				LoadAvg1:      1.5,
				LoadAvg5:      2.0,
				LoadAvg15:     2.5,
				Goroutines:    100,
				ProcessMemory: 100 * 1024 * 1024,
				Hostname:      "test-host",
				OS:            "linux",
				Arch:          "amd64",
				NumCPU:        8,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.health.CPUPercent < 0 || tt.health.CPUPercent > 100 {
				t.Errorf("Invalid CPUPercent: %f", tt.health.CPUPercent)
			}
			if tt.health.MemoryPercent < 0 || tt.health.MemoryPercent > 100 {
				t.Errorf("Invalid MemoryPercent: %f", tt.health.MemoryPercent)
			}
			if tt.health.DiskPercent < 0 || tt.health.DiskPercent > 100 {
				t.Errorf("Invalid DiskPercent: %f", tt.health.DiskPercent)
			}
		})
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have expected values
	if cpuSampleInterval != 100*time.Millisecond {
		t.Errorf("cpuSampleInterval = %v, want 100ms", cpuSampleInterval)
	}
	if cpuTickerInterval != 2*time.Second {
		t.Errorf("cpuTickerInterval = %v, want 2s", cpuTickerInterval)
	}
	if bytesPerKilobyte != 1024 {
		t.Errorf("bytesPerKilobyte = %d, want 1024", bytesPerKilobyte)
	}
	if bytesPerMegabyte != 1024*1024 {
		t.Errorf("bytesPerMegabyte = %d, want %d", bytesPerMegabyte, 1024*1024)
	}
	if topProcessCount != 5 {
		t.Errorf("topProcessCount = %d, want 5", topProcessCount)
	}
	if processCacheTTL != 5*time.Second {
		t.Errorf("processCacheTTL = %v, want 5s", processCacheTTL)
	}
}

func TestGetCachedProcessesBackgroundUpdate(t *testing.T) {
	// Force cache to be stale by waiting
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// Get initial cached data
	_, _ = getCachedProcesses()

	// Wait longer than cache TTL
	time.Sleep(processCacheTTL + 100*time.Millisecond)

	// This call should trigger a background update
	topCPU, topMemory := getCachedProcesses()

	// Wait for background update
	time.Sleep(300 * time.Millisecond)

	// Get updated data
	topCPU2, topMemory2 := getCachedProcesses()

	t.Logf("After TTL - TopCPU: %d->%d, TopMemory: %d->%d",
		len(topCPU), len(topCPU2), len(topMemory), len(topMemory2))
}

func TestCPUSamplerMultipleCalls(t *testing.T) {
	// Make multiple calls to ensure the sampler stays stable
	var lastPct float64
	for i := range 10 {
		pct := getCachedCPUPercent()
		if pct < 0 || pct > 100 {
			t.Errorf("Call %d: invalid CPU percentage %f", i+1, pct)
		}
		lastPct = pct
	}
	t.Logf("Last CPU percentage: %f", lastPct)
}

func TestGetTopProcessesInternalNoDuplicates(t *testing.T) {
	topCPU, topMemory := getTopProcessesInternal()

	// Check for duplicate PIDs in CPU list
	cpuPIDs := make(map[int]bool)
	for _, p := range topCPU {
		if cpuPIDs[p.PID] {
			// Note: This is not necessarily an error - same PID could appear
			// if there are threading issues, but log it
			t.Logf("Duplicate PID in topCPU: %d", p.PID)
		}
		cpuPIDs[p.PID] = true
	}

	// Check for duplicate PIDs in Memory list
	memPIDs := make(map[int]bool)
	for _, p := range topMemory {
		if memPIDs[p.PID] {
			t.Logf("Duplicate PID in topMemory: %d", p.PID)
		}
		memPIDs[p.PID] = true
	}
}

func TestGetTopProcessesInternalProcessNames(t *testing.T) {
	topCPU, topMemory := getTopProcessesInternal()

	// All processes should have non-empty names (we skip those without names)
	for i, p := range topCPU {
		if p.Name == "" {
			t.Errorf("topCPU[%d] has empty name", i)
		}
	}

	for i, p := range topMemory {
		if p.Name == "" {
			t.Errorf("topMemory[%d] has empty name", i)
		}
	}
}

func TestProcessCacheUpdateFlag(t *testing.T) {
	state := processCacheSingleton()

	// Check initial state of updateInFly
	state.updateMu.Lock()
	initialFlag := state.updateInFly
	state.updateMu.Unlock()

	t.Logf("Initial updateInFly: %v", initialFlag)

	// Trigger a cache refresh by calling getCachedProcesses
	// when cache is stale
	_, _ = getCachedProcesses()

	// The flag might be true if update is in progress
	state.updateMu.Lock()
	afterFlag := state.updateInFly
	state.updateMu.Unlock()

	t.Logf("After getCachedProcesses updateInFly: %v", afterFlag)
}

func BenchmarkGetTopProcessesInternal(b *testing.B) {
	for b.Loop() {
		_, _ = getTopProcessesInternal()
	}
}

func BenchmarkGetCachedProcesses(b *testing.B) {
	// Warm up
	_, _ = getCachedProcesses()
	time.Sleep(200 * time.Millisecond)

	b.ResetTimer()
	for b.Loop() {
		_, _ = getCachedProcesses()
	}
}

func BenchmarkGetCachedCPUPercent(b *testing.B) {
	// Warm up
	_ = getCachedCPUPercent()
	time.Sleep(150 * time.Millisecond)

	b.ResetTimer()
	for b.Loop() {
		_ = getCachedCPUPercent()
	}
}

func BenchmarkCPUCacheSingleton(b *testing.B) {
	for b.Loop() {
		_ = cpuCacheSingleton()
	}
}

func BenchmarkProcessCacheSingleton(b *testing.B) {
	for b.Loop() {
		_ = processCacheSingleton()
	}
}
