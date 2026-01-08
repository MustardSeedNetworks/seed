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
		name    string
		input   any
		wantErr bool
	}{
		{name: "simple map", input: map[string]string{"key": "value"}, wantErr: false},
		{name: "nested map", input: map[string]any{"outer": map[string]string{"inner": "value"}}, wantErr: false},
		{name: "slice of strings", input: []string{"a", "b", "c"}, wantErr: false},
		{name: "slice of ints", input: []int{1, 2, 3, 4, 5}, wantErr: false},
		{name: "empty map", input: map[string]string{}, wantErr: false},
		{name: "empty slice", input: []string{}, wantErr: false},
		{name: "nil value", input: nil, wantErr: false},
		{
			name: "complex struct",
			input: struct {
				Name    string `json:"name"`
				Count   int    `json:"count"`
				Enabled bool   `json:"enabled"`
			}{Name: "test", Count: 42, Enabled: true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcp.ExportFormatJSONResult(tt.input)
			checkFormatJSONResultError(t, tt.name, tt.wantErr, result, err)
		})
	}
}

// checkFormatJSONResultError validates the result of ExportFormatJSONResult.
func checkFormatJSONResultError(t *testing.T, name string, wantErr bool, result any, err error) {
	t.Helper()
	if wantErr {
		if err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
		return
	}
	if err != nil {
		t.Errorf("%s: unexpected error: %v", name, err)
		return
	}
	if result == nil {
		t.Errorf("%s: returned nil", name)
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
