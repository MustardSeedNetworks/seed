// Package system provides system health metrics collection.
// Test suite validates CPU, memory, disk, and uptime metrics collection.
package system

import (
	"runtime"
	"testing"
)

func TestGetHealth(t *testing.T) {
	health, err := GetHealth()
	if err != nil {
		t.Fatalf("GetHealth() returned error: %v", err)
	}

	if health == nil {
		t.Fatal("GetHealth() returned nil health")
	}
}

func TestGetHealthRuntimeFields(t *testing.T) {
	health, err := GetHealth()
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
	health, err := GetHealth()
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
	health, err := GetHealth()
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
	health, err := GetHealth()
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
	health, err := GetHealth()
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
	health, err := GetHealth()
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
	health, err := GetHealth()
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
		health, err := GetHealth()
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
	health := &Health{
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
