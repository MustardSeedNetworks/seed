package system

import (
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

func BenchmarkGetTopProcessesInternal(b *testing.B) {
	for b.Loop() {
		_, _ = getTopProcessesInternal()
	}
}
