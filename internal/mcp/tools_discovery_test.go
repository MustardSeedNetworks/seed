package mcp_test

import (
	"slices"
	"testing"

	"github.com/krisarmstrong/seed/internal/mcp"
)

type parsePortsCase struct {
	name     string
	spec     string
	expected []int
	wantErr  bool
}

func runParsePortsCase(t *testing.T, tt parsePortsCase) {
	t.Helper()

	result, err := mcp.ExportParsePorts(tt.spec)

	if tt.wantErr {
		if err == nil {
			t.Errorf("ExportParsePorts(%q) expected error, got nil", tt.spec)
		}
		return
	}

	if err != nil {
		t.Errorf("ExportParsePorts(%q) unexpected error: %v", tt.spec, err)
		return
	}

	// For max range size test, just check length
	if tt.name == "max valid range size" {
		if len(result) != 1000 {
			t.Errorf("ExportParsePorts(%q) expected 1000 ports, got %d", tt.spec, len(result))
		}
		return
	}

	if !slices.Equal(result, tt.expected) {
		t.Errorf("ExportParsePorts(%q) = %v, want %v", tt.spec, result, tt.expected)
	}
}

func TestParsePorts(t *testing.T) {
	tests := []parsePortsCase{
		{
			name:     "single port",
			spec:     "80",
			expected: []int{80},
			wantErr:  false,
		},
		{
			name:     "multiple ports comma separated",
			spec:     "22,80,443",
			expected: []int{22, 80, 443},
			wantErr:  false,
		},
		{
			name:     "port range",
			spec:     "20-25",
			expected: []int{20, 21, 22, 23, 24, 25},
			wantErr:  false,
		},
		{
			name:     "mixed ports and range",
			spec:     "22,80,100-102,443",
			expected: []int{22, 80, 100, 101, 102, 443},
			wantErr:  false,
		},
		{
			name:     "ports with spaces",
			spec:     " 22 , 80 , 443 ",
			expected: []int{22, 80, 443},
			wantErr:  false,
		},
		{
			name:     "duplicate ports",
			spec:     "80,80,443,443",
			expected: []int{80, 443},
			wantErr:  false,
		},
		{
			name:     "overlapping range and port",
			spec:     "22,20-25",
			expected: []int{22, 20, 21, 23, 24, 25},
			wantErr:  false,
		},
		{
			name:     "empty spec",
			spec:     "",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "only commas",
			spec:     ",,,",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid port string",
			spec:     "abc",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "port out of range high",
			spec:     "65536",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "port out of range zero",
			spec:     "0",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "invalid range reversed",
			spec:     "100-50",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "range too large",
			spec:     "1-2000",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "range with invalid start",
			spec:     "abc-100",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "range with invalid end",
			spec:     "100-abc",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "negative port in range",
			spec:     "-5-10",
			expected: nil,
			wantErr:  true,
		},
		{
			name:     "max valid range size",
			spec:     "1-1000",
			expected: nil, // not checking exact values due to length
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runParsePortsCase(t, tt)
		})
	}
}

func TestParsePortRange(t *testing.T) {
	tests := []struct {
		name      string
		part      string
		wantStart int
		wantEnd   int
		wantErr   bool
	}{
		{
			name:      "valid range",
			part:      "20-25",
			wantStart: 20,
			wantEnd:   25,
			wantErr:   false,
		},
		{
			name:      "single port range",
			part:      "80-80",
			wantStart: 80,
			wantEnd:   80,
			wantErr:   false,
		},
		{
			name:      "max valid range",
			part:      "1-1000",
			wantStart: 1,
			wantEnd:   1000,
			wantErr:   false,
		},
		{
			name:      "range with spaces",
			part:      " 100 - 200 ",
			wantStart: 100,
			wantEnd:   200,
			wantErr:   false,
		},
		{
			name:      "high valid ports",
			part:      "65000-65535",
			wantStart: 65000,
			wantEnd:   65535,
			wantErr:   false,
		},
		{
			name:    "missing hyphen",
			part:    "2025",
			wantErr: true,
		},
		{
			name:    "reversed range",
			part:    "100-50",
			wantErr: true,
		},
		{
			name:    "range too large",
			part:    "1-1002",
			wantErr: true,
		},
		{
			name:    "invalid start port",
			part:    "abc-100",
			wantErr: true,
		},
		{
			name:    "invalid end port",
			part:    "100-xyz",
			wantErr: true,
		},
		{
			name:    "port zero start",
			part:    "0-100",
			wantErr: true,
		},
		{
			name:    "port exceeds max",
			part:    "100-65536",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := mcp.ExportParsePortRange(tt.part)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportParsePortRange(%q) expected error, got nil", tt.part)
				}
				return
			}

			if err != nil {
				t.Errorf("ExportParsePortRange(%q) unexpected error: %v", tt.part, err)
				return
			}

			if start != tt.wantStart {
				t.Errorf("ExportParsePortRange(%q) start = %d, want %d", tt.part, start, tt.wantStart)
			}
			if end != tt.wantEnd {
				t.Errorf("ExportParsePortRange(%q) end = %d, want %d", tt.part, end, tt.wantEnd)
			}
		})
	}
}

func TestParseSinglePort(t *testing.T) {
	tests := []struct {
		name     string
		part     string
		expected int
		wantErr  bool
	}{
		{
			name:     "valid port 22",
			part:     "22",
			expected: 22,
			wantErr:  false,
		},
		{
			name:     "valid port 80",
			part:     "80",
			expected: 80,
			wantErr:  false,
		},
		{
			name:     "valid port 443",
			part:     "443",
			expected: 443,
			wantErr:  false,
		},
		{
			name:     "valid port 1",
			part:     "1",
			expected: 1,
			wantErr:  false,
		},
		{
			name:     "valid port max",
			part:     "65535",
			expected: 65535,
			wantErr:  false,
		},
		{
			name:    "port zero",
			part:    "0",
			wantErr: true,
		},
		{
			name:    "port exceeds max",
			part:    "65536",
			wantErr: true,
		},
		{
			name:    "negative port",
			part:    "-1",
			wantErr: true,
		},
		{
			name:    "non-numeric",
			part:    "abc",
			wantErr: true,
		},
		{
			name:    "float",
			part:    "80.5",
			wantErr: true,
		},
		{
			name:    "empty string",
			part:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcp.ExportParseSinglePort(tt.part)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportParseSinglePort(%q) expected error, got nil", tt.part)
				}
				return
			}

			if err != nil {
				t.Errorf("ExportParseSinglePort(%q) unexpected error: %v", tt.part, err)
				return
			}

			if result != tt.expected {
				t.Errorf("ExportParseSinglePort(%q) = %d, want %d", tt.part, result, tt.expected)
			}
		})
	}
}

func TestAddPortIfUnique(t *testing.T) {
	tests := []struct {
		name           string
		initialPorts   []int
		initialSeen    map[int]bool
		port           int
		expectedPorts  []int
		expectedInSeen bool
	}{
		{
			name:           "add new port to empty slice",
			initialPorts:   []int{},
			initialSeen:    map[int]bool{},
			port:           80,
			expectedPorts:  []int{80},
			expectedInSeen: true,
		},
		{
			name:           "add new port to existing slice",
			initialPorts:   []int{22, 443},
			initialSeen:    map[int]bool{22: true, 443: true},
			port:           80,
			expectedPorts:  []int{22, 443, 80},
			expectedInSeen: true,
		},
		{
			name:           "add duplicate port",
			initialPorts:   []int{22, 80, 443},
			initialSeen:    map[int]bool{22: true, 80: true, 443: true},
			port:           80,
			expectedPorts:  []int{22, 80, 443},
			expectedInSeen: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mcp.ExportAddPortIfUnique(tt.initialPorts, tt.initialSeen, tt.port)

			if !slices.Equal(result, tt.expectedPorts) {
				t.Errorf(
					"ExportAddPortIfUnique() = %v, want %v",
					result,
					tt.expectedPorts,
				)
			}

			if tt.initialSeen[tt.port] != tt.expectedInSeen {
				t.Errorf(
					"ExportAddPortIfUnique() seen[%d] = %v, want %v",
					tt.port,
					tt.initialSeen[tt.port],
					tt.expectedInSeen,
				)
			}
		})
	}
}

func TestAddPortRange(t *testing.T) {
	tests := []struct {
		name          string
		initialPorts  []int
		initialSeen   map[int]bool
		start         int
		end           int
		expectedCount int
	}{
		{
			name:          "add range to empty slice",
			initialPorts:  []int{},
			initialSeen:   map[int]bool{},
			start:         20,
			end:           22,
			expectedCount: 3,
		},
		{
			name:          "add range with some overlap",
			initialPorts:  []int{21},
			initialSeen:   map[int]bool{21: true},
			start:         20,
			end:           22,
			expectedCount: 3,
		},
		{
			name:          "add single port range",
			initialPorts:  []int{},
			initialSeen:   map[int]bool{},
			start:         80,
			end:           80,
			expectedCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mcp.ExportAddPortRange(tt.initialPorts, tt.initialSeen, tt.start, tt.end)

			if len(result) != tt.expectedCount {
				t.Errorf(
					"ExportAddPortRange() returned %d ports, want %d",
					len(result),
					tt.expectedCount,
				)
			}

			// Verify all ports in range are in the result
			for p := tt.start; p <= tt.end; p++ {
				if !slices.Contains(result, p) {
					t.Errorf("ExportAddPortRange() missing port %d", p)
				}
			}
		})
	}
}

func TestParsePortPart(t *testing.T) {
	tests := []struct {
		name          string
		initialPorts  []int
		initialSeen   map[int]bool
		part          string
		expectedCount int
		wantErr       bool
	}{
		{
			name:          "parse single port",
			initialPorts:  []int{},
			initialSeen:   map[int]bool{},
			part:          "80",
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name:          "parse port range",
			initialPorts:  []int{},
			initialSeen:   map[int]bool{},
			part:          "20-25",
			expectedCount: 6,
			wantErr:       false,
		},
		{
			name:          "parse with existing ports",
			initialPorts:  []int{22, 443},
			initialSeen:   map[int]bool{22: true, 443: true},
			part:          "80",
			expectedCount: 3,
			wantErr:       false,
		},
		{
			name:         "invalid port",
			initialPorts: []int{},
			initialSeen:  map[int]bool{},
			part:         "abc",
			wantErr:      true,
		},
		{
			name:         "invalid range",
			initialPorts: []int{},
			initialSeen:  map[int]bool{},
			part:         "100-50",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mcp.ExportParsePortPart(tt.initialPorts, tt.initialSeen, tt.part)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ExportParsePortPart(%q) expected error, got nil", tt.part)
				}
				return
			}

			if err != nil {
				t.Errorf("ExportParsePortPart(%q) unexpected error: %v", tt.part, err)
				return
			}

			if len(result) != tt.expectedCount {
				t.Errorf(
					"ExportParsePortPart(%q) returned %d ports, want %d",
					tt.part,
					len(result),
					tt.expectedCount,
				)
			}
		})
	}
}

func TestDiscoveryConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{"defaultScanTimeoutSeconds", mcp.ExportDefaultScanTimeoutSeconds, 30},
		{"maxScanTimeoutSeconds", mcp.ExportMaxScanTimeoutSeconds, 300},
		{"maxTracerouteHops", mcp.ExportMaxTracerouteHops, 64},
		{"defaultTracerouteTimeoutSeconds", mcp.ExportDefaultTracerouteTimeoutSeconds, 3},
		{"defaultTCPProbeTimeoutSeconds", mcp.ExportDefaultTCPProbeTimeoutSeconds, 5},
		{"defaultPortScanTimeoutSeconds", mcp.ExportDefaultPortScanTimeoutSeconds, 5},
		{"portScanConcurrency", mcp.ExportPortScanConcurrency, 10},
		{"portRangeSplitParts", mcp.ExportPortRangeSplitParts, 2},
		{"maxPortRangeSize", mcp.ExportMaxPortRangeSize, 1000},
		{"defaultTracerouteMaxHops", mcp.ExportDefaultTracerouteMaxHops, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.expected {
				t.Errorf("constant %s = %d, want %d", tt.name, tt.value, tt.expected)
			}
		})
	}
}
