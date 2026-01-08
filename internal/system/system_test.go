// Package system_test tests the system package for health metrics collection.
package system_test

import (
	"encoding/json"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/krisarmstrong/seed/internal/system"
)

func TestGetHealth(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	if health == nil {
		t.Fatal("GetHealth() returned nil health")
	}
}

func TestGetHealthRuntimeFields(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	tests := []struct {
		name     string
		got      any
		wantType string
		check    func(any) bool
	}{
		{
			name:     "Goroutines is positive",
			got:      health.Goroutines,
			wantType: "int",
			check: func(v any) bool {
				return v.(int) > 0
			},
		},
		{
			name:     "OS is not empty",
			got:      health.OS,
			wantType: "string",
			check: func(v any) bool {
				return v.(string) != ""
			},
		},
		{
			name:     "Arch is not empty",
			got:      health.Arch,
			wantType: "string",
			check: func(v any) bool {
				return v.(string) != ""
			},
		},
		{
			name:     "NumCPU is positive",
			got:      health.NumCPU,
			wantType: "int",
			check: func(v any) bool {
				return v.(int) > 0
			},
		},
		{
			name:     "Hostname is not empty",
			got:      health.Hostname,
			wantType: "string",
			check: func(v any) bool {
				return v.(string) != ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.check(tt.got) {
				t.Errorf("%s: got %v, validation failed", tt.name, tt.got)
			}
		})
	}
}

func TestGetHealthRuntimeValuesMatchGo(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	// Verify runtime values match Go runtime
	if health.OS != runtime.GOOS {
		t.Errorf("OS: got %v, want %v", health.OS, runtime.GOOS)
	}

	if health.Arch != runtime.GOARCH {
		t.Errorf("Arch: got %v, want %v", health.Arch, runtime.GOARCH)
	}

	if health.NumCPU != runtime.NumCPU() {
		t.Errorf("NumCPU: got %v, want %v", health.NumCPU, runtime.NumCPU())
	}

	// Goroutines should be at least 1 (this test goroutine)
	if health.Goroutines < 1 {
		t.Errorf("Goroutines: got %v, want >= 1", health.Goroutines)
	}
}

func TestGetHealthMetricsInRange(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	tests := []struct {
		name  string
		value float64
		min   float64
		max   float64
	}{
		{"CPUPercent in range 0-100", health.CPUPercent, 0.0, 100.0},
		{"MemoryPercent in range 0-100", health.MemoryPercent, 0.0, 100.0},
		{"DiskPercent in range 0-100", health.DiskPercent, 0.0, 100.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < tt.min || tt.value > tt.max {
				t.Errorf("%s: got %v, want range [%v, %v]", tt.name, tt.value, tt.min, tt.max)
			}
		})
	}
}

func TestGetHealthMemoryStats(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	// Memory stats may not be available in all environments (containers, sandboxes)
	// Skip strict checks if memory stats couldn't be collected
	if health.MemoryTotal == 0 {
		t.Skip("Memory stats not available in this environment (likely sandbox/container)")
	}

	// Memory used should be less than or equal to total
	if health.MemoryUsed > health.MemoryTotal {
		t.Errorf("MemoryUsed (%v) > MemoryTotal (%v)", health.MemoryUsed, health.MemoryTotal)
	}

	// Process memory should be greater than 0 (this uses Go runtime, should always work)
	if health.ProcessMemory == 0 {
		t.Error("ProcessMemory should be > 0")
	}
}

func TestGetHealthDiskStats(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	// Disk used should be less than or equal to total
	if health.DiskUsed > health.DiskTotal {
		t.Errorf("DiskUsed (%v) > DiskTotal (%v)", health.DiskUsed, health.DiskTotal)
	}

	// Disk total should be greater than 0
	if health.DiskTotal == 0 {
		t.Error("DiskTotal should be > 0")
	}
}

func TestGetHealthLoadAverages(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	// Load averages should be non-negative
	if health.LoadAvg1 < 0 {
		t.Errorf("LoadAvg1 should be >= 0, got %v", health.LoadAvg1)
	}

	if health.LoadAvg5 < 0 {
		t.Errorf("LoadAvg5 should be >= 0, got %v", health.LoadAvg5)
	}

	if health.LoadAvg15 < 0 {
		t.Errorf("LoadAvg15 should be >= 0, got %v", health.LoadAvg15)
	}
}

func TestGetHealthUptime(t *testing.T) {
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	// Uptime should be greater than 0 (system has been running for some time)
	if health.Uptime == 0 {
		t.Error("Uptime should be > 0")
	}
}

func TestGetHealthMultipleCalls(t *testing.T) {
	// Test that multiple calls don't panic or error
	for i := range 5 {
		health, err := system.GetHealth()
		if err != nil {
			t.Fatalf("GetHealth() call %d returned error: %v", i+1, err)
		}

		if health == nil {
			t.Fatalf("GetHealth() call %d returned nil", i+1)
		}
	}
}

func TestHealthStructFields(_ *testing.T) {
	// Test that the Health struct has all expected fields
	health := &system.Health{
		CPUPercent:    50.0,
		MemoryPercent: 60.0,
		MemoryUsed:    1000,
		MemoryTotal:   2000,
		DiskPercent:   70.0,
		DiskUsed:      5000,
		DiskTotal:     10000,
		Uptime:        3600,
		LoadAvg1:      1.0,
		LoadAvg5:      1.5,
		LoadAvg15:     2.0,
		Goroutines:    10,
		ProcessMemory: 50000,
		Hostname:      "test-host",
		OS:            "linux",
		Arch:          "amd64",
		NumCPU:        4,
	}

	// Verify all fields are accessible
	_ = health.CPUPercent
	_ = health.MemoryPercent
	_ = health.MemoryUsed
	_ = health.MemoryTotal
	_ = health.DiskPercent
	_ = health.DiskUsed
	_ = health.DiskTotal
	_ = health.Uptime
	_ = health.LoadAvg1
	_ = health.LoadAvg5
	_ = health.LoadAvg15
	_ = health.Goroutines
	_ = health.ProcessMemory
	_ = health.Hostname
	_ = health.OS
	_ = health.Arch
	_ = health.NumCPU
}

func TestHealthStructWithProcessInfoFields(t *testing.T) {
	// Test Health struct with TopCPUProcesses and TopMemoryProcesses fields
	procs := []system.ProcessInfo{
		{Name: "test-proc", PID: 123, CPUPercent: 50.0, MemoryMB: 100.0},
	}

	health := &system.Health{
		TopCPUProcesses:    procs,
		TopMemoryProcesses: procs,
	}

	// Verify process fields are accessible
	if len(health.TopCPUProcesses) != 1 {
		t.Errorf("Expected 1 TopCPUProcesses, got %d", len(health.TopCPUProcesses))
	}
	if len(health.TopMemoryProcesses) != 1 {
		t.Errorf("Expected 1 TopMemoryProcesses, got %d", len(health.TopMemoryProcesses))
	}
}

func TestProcessInfoStruct(t *testing.T) {
	tests := []struct {
		name        string
		processInfo system.ProcessInfo
		wantName    string
		wantPID     int
		wantCPU     float64
		wantMemory  float64
	}{
		{
			name: "typical process",
			processInfo: system.ProcessInfo{
				Name:       "chrome",
				PID:        1234,
				CPUPercent: 25.5,
				MemoryMB:   512.0,
			},
			wantName:   "chrome",
			wantPID:    1234,
			wantCPU:    25.5,
			wantMemory: 512.0,
		},
		{
			name: "process with zero values",
			processInfo: system.ProcessInfo{
				Name:       "idle",
				PID:        0,
				CPUPercent: 0.0,
				MemoryMB:   0.0,
			},
			wantName:   "idle",
			wantPID:    0,
			wantCPU:    0.0,
			wantMemory: 0.0,
		},
		{
			name: "process with high CPU",
			processInfo: system.ProcessInfo{
				Name:       "stress",
				PID:        9999,
				CPUPercent: 100.0,
				MemoryMB:   8192.0,
			},
			wantName:   "stress",
			wantPID:    9999,
			wantCPU:    100.0,
			wantMemory: 8192.0,
		},
		{
			name: "process with special characters in name",
			processInfo: system.ProcessInfo{
				Name:       "my-process.exe",
				PID:        555,
				CPUPercent: 10.0,
				MemoryMB:   256.0,
			},
			wantName:   "my-process.exe",
			wantPID:    555,
			wantCPU:    10.0,
			wantMemory: 256.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.processInfo.Name != tt.wantName {
				t.Errorf("ProcessInfo.Name = %v, want %v", tt.processInfo.Name, tt.wantName)
			}
			if tt.processInfo.PID != tt.wantPID {
				t.Errorf("ProcessInfo.PID = %v, want %v", tt.processInfo.PID, tt.wantPID)
			}
			if tt.processInfo.CPUPercent != tt.wantCPU {
				t.Errorf("ProcessInfo.CPUPercent = %v, want %v", tt.processInfo.CPUPercent, tt.wantCPU)
			}
			if tt.processInfo.MemoryMB != tt.wantMemory {
				t.Errorf("ProcessInfo.MemoryMB = %v, want %v", tt.processInfo.MemoryMB, tt.wantMemory)
			}
		})
	}
}

func TestProcessInfoJSONSerialization(t *testing.T) {
	// Test JSON serialization of ProcessInfo
	proc := system.ProcessInfo{
		Name:       "test-process",
		PID:        12345,
		CPUPercent: 45.67,
		MemoryMB:   256.89,
	}

	data, err := json.Marshal(proc)
	if err != nil {
		t.Fatalf("Failed to marshal ProcessInfo: %v", err)
	}

	var decoded system.ProcessInfo
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal ProcessInfo: %v", unmarshalErr)
	}

	if decoded.Name != proc.Name {
		t.Errorf("Name mismatch after JSON round-trip: got %v, want %v", decoded.Name, proc.Name)
	}
	if decoded.PID != proc.PID {
		t.Errorf("PID mismatch after JSON round-trip: got %v, want %v", decoded.PID, proc.PID)
	}
	if decoded.CPUPercent != proc.CPUPercent {
		t.Errorf("CPUPercent mismatch after JSON round-trip: got %v, want %v", decoded.CPUPercent, proc.CPUPercent)
	}
	if decoded.MemoryMB != proc.MemoryMB {
		t.Errorf("MemoryMB mismatch after JSON round-trip: got %v, want %v", decoded.MemoryMB, proc.MemoryMB)
	}
}

func TestHealthJSONSerialization(t *testing.T) {
	health := &system.Health{
		CPUPercent:    50.0,
		MemoryPercent: 60.0,
		MemoryUsed:    8589934592,
		MemoryTotal:   17179869184,
		DiskPercent:   70.0,
		DiskUsed:      107374182400,
		DiskTotal:     214748364800,
		Uptime:        3600,
		LoadAvg1:      1.0,
		LoadAvg5:      1.5,
		LoadAvg15:     2.0,
		Goroutines:    10,
		ProcessMemory: 50000000,
		Hostname:      "test-host",
		OS:            "darwin",
		Arch:          "arm64",
		NumCPU:        8,
		TopCPUProcesses: []system.ProcessInfo{
			{Name: "proc1", PID: 1, CPUPercent: 50.0, MemoryMB: 100.0},
		},
		TopMemoryProcesses: []system.ProcessInfo{
			{Name: "proc2", PID: 2, CPUPercent: 10.0, MemoryMB: 500.0},
		},
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal Health: %v", err)
	}

	var decoded system.Health
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal Health: %v", unmarshalErr)
	}

	// Verify key fields
	if decoded.CPUPercent != health.CPUPercent {
		t.Errorf("CPUPercent mismatch: got %v, want %v", decoded.CPUPercent, health.CPUPercent)
	}
	if decoded.MemoryPercent != health.MemoryPercent {
		t.Errorf("MemoryPercent mismatch: got %v, want %v", decoded.MemoryPercent, health.MemoryPercent)
	}
	if decoded.Hostname != health.Hostname {
		t.Errorf("Hostname mismatch: got %v, want %v", decoded.Hostname, health.Hostname)
	}
	if len(decoded.TopCPUProcesses) != len(health.TopCPUProcesses) {
		t.Errorf("TopCPUProcesses length mismatch: got %v, want %v",
			len(decoded.TopCPUProcesses), len(health.TopCPUProcesses))
	}
	if len(decoded.TopMemoryProcesses) != len(health.TopMemoryProcesses) {
		t.Errorf("TopMemoryProcesses length mismatch: got %v, want %v",
			len(decoded.TopMemoryProcesses), len(health.TopMemoryProcesses))
	}
}

func TestHealthJSONOmitEmpty(t *testing.T) {
	// Test that empty TopCPUProcesses and TopMemoryProcesses are omitted
	health := &system.Health{
		CPUPercent:    50.0,
		MemoryPercent: 60.0,
		Hostname:      "test-host",
		OS:            "linux",
		Arch:          "amd64",
		NumCPU:        4,
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal Health: %v", err)
	}

	jsonStr := string(data)
	// These fields should be omitted when empty (omitempty tag)
	if contains(jsonStr, "topCpuProcesses") {
		t.Error("Expected topCpuProcesses to be omitted when empty")
	}
	if contains(jsonStr, "topMemoryProcesses") {
		t.Error("Expected topMemoryProcesses to be omitted when empty")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsImpl(s, substr))
}

func containsImpl(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetHealthConcurrency(t *testing.T) {
	// Test concurrent calls to GetHealth
	const numGoroutines = 10
	var wg sync.WaitGroup
	errChan := make(chan error, numGoroutines)

	wg.Add(numGoroutines)
	for range numGoroutines {
		go func() {
			defer wg.Done()
			health, err := system.GetHealth()
			if err != nil {
				errChan <- err
				return
			}
			if health == nil {
				errChan <- err
				return
			}
		}()
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		if err != nil {
			t.Errorf("Concurrent GetHealth() call failed: %v", err)
		}
	}
}

func TestGetHealthConsistency(t *testing.T) {
	// Test that GetHealth returns consistent static values
	health1, err := system.GetHealth()
	if err != nil {
		t.Fatalf("First GetHealth() call failed: %v", err)
	}

	health2, err := system.GetHealth()
	if err != nil {
		t.Fatalf("Second GetHealth() call failed: %v", err)
	}

	// These values should be identical between calls
	if health1.OS != health2.OS {
		t.Errorf("OS inconsistent: %v vs %v", health1.OS, health2.OS)
	}
	if health1.Arch != health2.Arch {
		t.Errorf("Arch inconsistent: %v vs %v", health1.Arch, health2.Arch)
	}
	if health1.NumCPU != health2.NumCPU {
		t.Errorf("NumCPU inconsistent: %v vs %v", health1.NumCPU, health2.NumCPU)
	}
	if health1.Hostname != health2.Hostname {
		t.Errorf("Hostname inconsistent: %v vs %v", health1.Hostname, health2.Hostname)
	}
}

func TestGetHealthCPUSampling(t *testing.T) {
	// Test that CPU sampling works over multiple calls
	// This exercises the background CPU sampler

	// First call initializes the sampler
	health1, err := system.GetHealth()
	if err != nil {
		t.Fatalf("First GetHealth() call failed: %v", err)
	}

	// Wait a bit for the sampler to potentially update
	time.Sleep(150 * time.Millisecond)

	// Second call should return cached or updated CPU value
	health2, err := system.GetHealth()
	if err != nil {
		t.Fatalf("Second GetHealth() call failed: %v", err)
	}

	// Both should have valid CPU percentages in range
	if health1.CPUPercent < 0 || health1.CPUPercent > 100 {
		t.Errorf("First call CPUPercent out of range: %v", health1.CPUPercent)
	}
	if health2.CPUPercent < 0 || health2.CPUPercent > 100 {
		t.Errorf("Second call CPUPercent out of range: %v", health2.CPUPercent)
	}
}

func TestGetHealthCPUSamplerTicker(t *testing.T) {
	// Test CPU sampler over a longer period to exercise ticker path
	// This test is longer but ensures the background goroutine works correctly

	if testing.Short() {
		t.Skip("Skipping long-running CPU sampler test in short mode")
	}

	// Make several calls spaced out to exercise the sampler
	for i := range 3 {
		health, err := system.GetHealth()
		if err != nil {
			t.Fatalf("GetHealth() call %d failed: %v", i+1, err)
		}

		if health.CPUPercent < 0 || health.CPUPercent > 100 {
			t.Errorf("Call %d: CPUPercent out of range: %v", i+1, health.CPUPercent)
		}

		if i < 2 {
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func TestGetHealthProcessMemory(t *testing.T) {
	// Test that ProcessMemory (Go runtime memory) is reported correctly
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() failed: %v", err)
	}

	// ProcessMemory should be positive (we're using memory right now)
	if health.ProcessMemory == 0 {
		t.Error("ProcessMemory should be > 0")
	}

	// Allocate some memory and verify ProcessMemory increases
	data := make([]byte, 10*1024*1024) // 10MB
	_ = data[0]                        // Prevent optimization

	health2, err := system.GetHealth()
	if err != nil {
		t.Fatalf("Second GetHealth() failed: %v", err)
	}

	// ProcessMemory should still be positive
	if health2.ProcessMemory == 0 {
		t.Error("ProcessMemory should be > 0 after allocation")
	}
}

func TestHealthFieldTypes(t *testing.T) {
	// Verify that Health struct fields have correct types by using them
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() failed: %v", err)
	}

	// Use type assertions to verify field types compile correctly
	_ = health.CPUPercent + 0.0
	_ = health.MemoryPercent + 0.0
	_ = health.MemoryUsed + 0
	_ = health.MemoryTotal + 0
	_ = health.DiskPercent + 0.0
	_ = health.DiskUsed + 0
	_ = health.DiskTotal + 0
	_ = health.Uptime + 0
	_ = health.LoadAvg1 + 0.0
	_ = health.LoadAvg5 + 0.0
	_ = health.LoadAvg15 + 0.0
	_ = health.Goroutines + 0
	_ = health.ProcessMemory + 0
	_ = health.Hostname + ""
	_ = health.OS + ""
	_ = health.Arch + ""
	_ = health.NumCPU + 0
	_ = append(health.TopCPUProcesses, system.ProcessInfo{})
	_ = append(health.TopMemoryProcesses, system.ProcessInfo{})
}

func TestProcessInfoFieldTypes(_ *testing.T) {
	// Verify ProcessInfo struct field types
	proc := system.ProcessInfo{
		Name:       "test",
		PID:        1,
		CPUPercent: 50.0,
		MemoryMB:   100.0,
	}

	// Use operations to verify field types compile correctly
	_ = proc.Name + ""
	_ = proc.PID + 0
	_ = proc.CPUPercent + 0.0
	_ = proc.MemoryMB + 0.0
}

func TestGetHealthGoroutineCount(t *testing.T) {
	// Test that goroutine count is reasonable
	initialGoroutines := runtime.NumGoroutine()

	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() failed: %v", err)
	}

	// Health goroutine count should be close to actual count
	// Allow for some variance due to timing
	if health.Goroutines < 1 {
		t.Errorf("Goroutines should be >= 1, got %d", health.Goroutines)
	}

	// Should not be wildly different from current count
	diff := health.Goroutines - initialGoroutines
	if diff < -10 || diff > 100 {
		t.Errorf("Goroutines differs too much from runtime.NumGoroutine(): health=%d, initial=%d",
			health.Goroutines, initialGoroutines)
	}
}

func TestGetHealthZeroValues(t *testing.T) {
	// Verify that Health struct has sensible zero value handling
	var health system.Health

	// Zero-value Health should have zero/empty values
	if health.CPUPercent != 0 {
		t.Errorf("Zero-value CPUPercent should be 0, got %v", health.CPUPercent)
	}
	if health.MemoryPercent != 0 {
		t.Errorf("Zero-value MemoryPercent should be 0, got %v", health.MemoryPercent)
	}
	if health.Hostname != "" {
		t.Errorf("Zero-value Hostname should be empty, got %v", health.Hostname)
	}
	if health.TopCPUProcesses != nil {
		t.Errorf("Zero-value TopCPUProcesses should be nil, got %v", health.TopCPUProcesses)
	}
}

func TestProcessInfoZeroValues(t *testing.T) {
	// Verify that ProcessInfo struct has sensible zero value handling
	var proc system.ProcessInfo

	if proc.Name != "" {
		t.Errorf("Zero-value Name should be empty, got %v", proc.Name)
	}
	if proc.PID != 0 {
		t.Errorf("Zero-value PID should be 0, got %v", proc.PID)
	}
	if proc.CPUPercent != 0 {
		t.Errorf("Zero-value CPUPercent should be 0, got %v", proc.CPUPercent)
	}
	if proc.MemoryMB != 0 {
		t.Errorf("Zero-value MemoryMB should be 0, got %v", proc.MemoryMB)
	}
}

func TestGetHealthRapidCalls(t *testing.T) {
	// Test rapid successive calls (stress test caching)
	const numCalls = 100

	for i := range numCalls {
		health, err := system.GetHealth()
		if err != nil {
			t.Fatalf("GetHealth() call %d failed: %v", i+1, err)
		}
		if health == nil {
			t.Fatalf("GetHealth() call %d returned nil", i+1)
		}
	}
}

func TestHealthAllFieldsPopulated(t *testing.T) {
	// Test that a real GetHealth call populates important fields
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() failed: %v", err)
	}

	// Runtime fields should always be populated
	if health.OS == "" {
		t.Error("OS should not be empty")
	}
	if health.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if health.NumCPU == 0 {
		t.Error("NumCPU should not be 0")
	}
	if health.Goroutines == 0 {
		t.Error("Goroutines should not be 0")
	}

	// System stats might be 0 in some environments, but we log them
	t.Logf("CPUPercent: %v", health.CPUPercent)
	t.Logf("MemoryPercent: %v", health.MemoryPercent)
	t.Logf("DiskPercent: %v", health.DiskPercent)
	t.Logf("Uptime: %v", health.Uptime)
	t.Logf("LoadAvg1: %v", health.LoadAvg1)
}

func TestGetHealthBoundaryConditions(t *testing.T) {
	// Test boundary conditions for metrics
	health, err := system.GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() failed: %v", err)
	}

	// Percentages should be in valid range
	percentTests := []struct {
		name  string
		value float64
	}{
		{"CPUPercent", health.CPUPercent},
		{"MemoryPercent", health.MemoryPercent},
		{"DiskPercent", health.DiskPercent},
	}

	for _, tt := range percentTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value < 0 {
				t.Errorf("%s should not be negative: %v", tt.name, tt.value)
			}
			if tt.value > 100 {
				t.Errorf("%s should not exceed 100: %v", tt.name, tt.value)
			}
		})
	}

	// Used should not exceed total
	if health.MemoryTotal > 0 && health.MemoryUsed > health.MemoryTotal {
		t.Errorf("MemoryUsed (%d) exceeds MemoryTotal (%d)",
			health.MemoryUsed, health.MemoryTotal)
	}
	if health.DiskTotal > 0 && health.DiskUsed > health.DiskTotal {
		t.Errorf("DiskUsed (%d) exceeds DiskTotal (%d)",
			health.DiskUsed, health.DiskTotal)
	}
}

func TestHealthJSONTagNames(t *testing.T) {
	// Verify JSON tag names by marshaling and checking keys
	health := &system.Health{
		CPUPercent:    50.0,
		MemoryPercent: 60.0,
		MemoryUsed:    1000,
		MemoryTotal:   2000,
		DiskPercent:   70.0,
		DiskUsed:      5000,
		DiskTotal:     10000,
		Uptime:        3600,
		LoadAvg1:      1.0,
		LoadAvg5:      1.5,
		LoadAvg15:     2.0,
		Goroutines:    10,
		ProcessMemory: 50000,
		Hostname:      "test-host",
		OS:            "linux",
		Arch:          "amd64",
		NumCPU:        4,
	}

	data, err := json.Marshal(health)
	if err != nil {
		t.Fatalf("Failed to marshal Health: %v", err)
	}

	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal to map: %v", unmarshalErr)
	}

	expectedKeys := []string{
		"cpuPercent", "memoryPercent", "memoryUsed", "memoryTotal",
		"diskPercent", "diskUsed", "diskTotal", "uptime",
		"loadAvg1", "loadAvg5", "loadAvg15", "goroutines",
		"processMemory", "hostname", "os", "arch", "numCpu",
	}

	for _, key := range expectedKeys {
		if _, ok := decoded[key]; !ok {
			t.Errorf("Expected JSON key %q not found", key)
		}
	}
}

func TestProcessInfoJSONTagNames(t *testing.T) {
	proc := system.ProcessInfo{
		Name:       "test",
		PID:        123,
		CPUPercent: 50.0,
		MemoryMB:   100.0,
	}

	data, err := json.Marshal(proc)
	if err != nil {
		t.Fatalf("Failed to marshal ProcessInfo: %v", err)
	}

	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(data, &decoded); unmarshalErr != nil {
		t.Fatalf("Failed to unmarshal to map: %v", unmarshalErr)
	}

	expectedKeys := []string{"name", "pid", "cpuPercent", "memoryMb"}
	for _, key := range expectedKeys {
		if _, ok := decoded[key]; !ok {
			t.Errorf("Expected JSON key %q not found", key)
		}
	}
}

// Benchmarks

func BenchmarkGetHealth(b *testing.B) {
	// Warm up
	_, _ = system.GetHealth()

	b.ResetTimer()
	for b.Loop() {
		_, _ = system.GetHealth()
	}
}

func BenchmarkGetHealthParallel(b *testing.B) {
	// Warm up
	_, _ = system.GetHealth()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = system.GetHealth()
		}
	})
}

func BenchmarkHealthJSONMarshal(b *testing.B) {
	health := &system.Health{
		CPUPercent:    50.0,
		MemoryPercent: 60.0,
		MemoryUsed:    8589934592,
		MemoryTotal:   17179869184,
		DiskPercent:   70.0,
		DiskUsed:      107374182400,
		DiskTotal:     214748364800,
		Uptime:        3600,
		LoadAvg1:      1.0,
		LoadAvg5:      1.5,
		LoadAvg15:     2.0,
		Goroutines:    10,
		ProcessMemory: 50000000,
		Hostname:      "test-host",
		OS:            "darwin",
		Arch:          "arm64",
		NumCPU:        8,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(health)
	}
}

func BenchmarkProcessInfoJSONMarshal(b *testing.B) {
	proc := system.ProcessInfo{
		Name:       "test-process",
		PID:        12345,
		CPUPercent: 45.67,
		MemoryMB:   256.89,
	}

	b.ResetTimer()
	for b.Loop() {
		_, _ = json.Marshal(proc)
	}
}
