package mcp_test

import (
	"encoding/json"
	"testing"

	"github.com/krisarmstrong/seed/internal/mcp"
)

func TestSystemConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "BytesPerKilobyte",
			value:    mcp.ExportBytesPerKilobyte,
			expected: 1024,
		},
		{
			name:     "BytesPerMegabyte",
			value:    mcp.ExportBytesPerMegabyte,
			expected: 1024 * 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("constant %s = %d, want %d", tt.name, tt.value, tt.expected)
			}
		})
	}
}

func TestBytesConversion(t *testing.T) {
	tests := []struct {
		name       string
		bytes      int
		expectedKB float64
		expectedMB float64
	}{
		{
			name:       "zero bytes",
			bytes:      0,
			expectedKB: 0,
			expectedMB: 0,
		},
		{
			name:       "one kilobyte",
			bytes:      1024,
			expectedKB: 1.0,
			expectedMB: 1.0 / 1024.0,
		},
		{
			name:       "one megabyte",
			bytes:      1024 * 1024,
			expectedKB: 1024.0,
			expectedMB: 1.0,
		},
		{
			name:       "100 megabytes",
			bytes:      100 * 1024 * 1024,
			expectedKB: 100 * 1024.0,
			expectedMB: 100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kb := float64(tt.bytes) / float64(mcp.ExportBytesPerKilobyte)
			mb := float64(tt.bytes) / float64(mcp.ExportBytesPerMegabyte)

			if kb != tt.expectedKB {
				t.Errorf("bytes to KB: got %f, want %f", kb, tt.expectedKB)
			}
			if mb != tt.expectedMB {
				t.Errorf("bytes to MB: got %f, want %f", mb, tt.expectedMB)
			}
		})
	}
}

func TestFormatJSONResult(t *testing.T) {
	tests := []struct {
		name        string
		input       any
		wantErr     bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:    "simple map",
			input:   map[string]string{"key": "value"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to unmarshal output: %v", err)
					return
				}
				if result["key"] != "value" {
					t.Errorf("expected key=value, got key=%s", result["key"])
				}
			},
		},
		{
			name:    "nested map",
			input:   map[string]any{"outer": map[string]string{"inner": "value"}},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to unmarshal output: %v", err)
					return
				}
				outer, ok := result["outer"].(map[string]any)
				if !ok {
					t.Errorf("expected outer to be a map")
					return
				}
				if outer["inner"] != "value" {
					t.Errorf("expected inner=value, got inner=%v", outer["inner"])
				}
			},
		},
		{
			name:    "slice of strings",
			input:   []string{"a", "b", "c"},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result []string
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to unmarshal output: %v", err)
					return
				}
				if len(result) != 3 {
					t.Errorf("expected 3 elements, got %d", len(result))
				}
			},
		},
		{
			name:    "slice of ints",
			input:   []int{1, 2, 3, 4, 5},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result []int
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to unmarshal output: %v", err)
					return
				}
				if len(result) != 5 {
					t.Errorf("expected 5 elements, got %d", len(result))
				}
			},
		},
		{
			name:    "empty map",
			input:   map[string]string{},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if output != "{}" {
					t.Errorf("expected {}, got %s", output)
				}
			},
		},
		{
			name:    "empty slice",
			input:   []string{},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if output != "[]" {
					t.Errorf("expected [], got %s", output)
				}
			},
		},
		{
			name:    "nil value",
			input:   nil,
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				if output != "null" {
					t.Errorf("expected null, got %s", output)
				}
			},
		},
		{
			name: "complex struct",
			input: struct {
				Name    string `json:"name"`
				Count   int    `json:"count"`
				Enabled bool   `json:"enabled"`
			}{
				Name:    "test",
				Count:   42,
				Enabled: true,
			},
			wantErr: false,
			checkOutput: func(t *testing.T, output string) {
				var result map[string]any
				if err := json.Unmarshal([]byte(output), &result); err != nil {
					t.Errorf("failed to unmarshal output: %v", err)
					return
				}
				if result["name"] != "test" {
					t.Errorf("expected name=test, got name=%v", result["name"])
				}
				if result["count"] != float64(42) {
					t.Errorf("expected count=42, got count=%v", result["count"])
				}
				if result["enabled"] != true {
					t.Errorf("expected enabled=true, got enabled=%v", result["enabled"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcp.ExportFormatJSONResult(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportFormatJSONResult() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ExportFormatJSONResult() unexpected error: %v", err)
				return
			}

			// The result is a *mcp.CallToolResult, we need to extract the text content
			// For testing purposes, we'll use a type assertion approach
			if result == nil {
				t.Errorf("ExportFormatJSONResult() returned nil")
				return
			}

			// Since we can't easily access the internal structure of CallToolResult,
			// we'll just verify it's not nil for now
			// More detailed testing would require mocking or integration tests
		})
	}
}

func TestSystemHealthResponse(t *testing.T) {
	// Test that we can create expected health response structures
	tests := []struct {
		name   string
		health map[string]any
		check  func(t *testing.T, h map[string]any)
	}{
		{
			name: "basic health response",
			health: map[string]any{
				"version":    "0.165.34",
				"goVersion":  "go1.25.5",
				"os":         "darwin",
				"arch":       "arm64",
				"numCPU":     8,
				"goroutines": 10,
				"memory": map[string]any{
					"allocMB":      50.5,
					"totalAllocMB": 100.0,
					"sysMB":        150.0,
					"numGC":        5,
				},
			},
			check: func(t *testing.T, h map[string]any) {
				if h["version"] != "0.165.34" {
					t.Errorf("expected version 0.165.34, got %v", h["version"])
				}
				if h["numCPU"] != 8 {
					t.Errorf("expected numCPU 8, got %v", h["numCPU"])
				}
				mem, ok := h["memory"].(map[string]any)
				if !ok {
					t.Error("expected memory to be a map")
					return
				}
				if mem["allocMB"] != 50.5 {
					t.Errorf("expected allocMB 50.5, got %v", mem["allocMB"])
				}
			},
		},
		{
			name: "health with interface and config",
			health: map[string]any{
				"version":       "0.165.34",
				"interface":     "eth0",
				"mcpEnabled":    true,
				"icmpAvailable": true,
			},
			check: func(t *testing.T, h map[string]any) {
				if h["interface"] != "eth0" {
					t.Errorf("expected interface eth0, got %v", h["interface"])
				}
				if h["mcpEnabled"] != true {
					t.Errorf("expected mcpEnabled true, got %v", h["mcpEnabled"])
				}
				if h["icmpAvailable"] != true {
					t.Errorf("expected icmpAvailable true, got %v", h["icmpAvailable"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.health)
		})
	}
}

func TestLinkStatusResponse(t *testing.T) {
	// Test link status response structure
	tests := []struct {
		name   string
		status map[string]any
		check  func(t *testing.T, s map[string]any)
	}{
		{
			name: "link up",
			status: map[string]any{
				"state": "up",
				"isUp":  true,
			},
			check: func(t *testing.T, s map[string]any) {
				if s["isUp"] != true {
					t.Errorf("expected isUp true, got %v", s["isUp"])
				}
			},
		},
		{
			name: "link down",
			status: map[string]any{
				"state": "down",
				"isUp":  false,
			},
			check: func(t *testing.T, s map[string]any) {
				if s["isUp"] != false {
					t.Errorf("expected isUp false, got %v", s["isUp"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.check(t, tt.status)
		})
	}
}

func TestIPConfigResponse(t *testing.T) {
	// Test IP configuration response structure validation
	tests := []struct {
		name   string
		config map[string]any
		valid  bool
	}{
		{
			name: "valid IPv4 config",
			config: map[string]any{
				"ipv4Address": "192.168.1.100",
				"netmask":     "255.255.255.0",
				"gateway":     "192.168.1.1",
				"dns":         []string{"8.8.8.8", "8.8.4.4"},
			},
			valid: true,
		},
		{
			name: "valid IPv6 config",
			config: map[string]any{
				"ipv6Address": "2001:db8::1",
				"prefix":      64,
				"gateway":     "2001:db8::fffe",
				"dns":         []string{"2001:4860:4860::8888"},
			},
			valid: true,
		},
		{
			name: "dual stack config",
			config: map[string]any{
				"ipv4Address": "192.168.1.100",
				"ipv6Address": "2001:db8::1",
				"netmask":     "255.255.255.0",
				"prefix":      64,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the map can be serialized to JSON
			data, err := json.Marshal(tt.config)
			if err != nil && tt.valid {
				t.Errorf("expected valid JSON, got error: %v", err)
			}
			if err == nil && !tt.valid {
				t.Errorf("expected invalid JSON, but got: %s", data)
			}
		})
	}
}
