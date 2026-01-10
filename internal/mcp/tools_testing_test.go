package mcp_test

import (
	"testing"

	"github.com/krisarmstrong/seed/internal/iperf"
	"github.com/krisarmstrong/seed/internal/mcp"
)

func TestParseIperfConfig(t *testing.T) {
	tests := []struct {
		name           string
		serverAddr     string
		args           map[string]any
		wantServer     string
		wantPort       int
		wantDuration   int
		wantProtocol   string
		wantDirection  string
		wantErr        bool
		wantErrContain string
	}{
		{
			name:         "minimal config",
			serverAddr:   "192.168.1.100",
			args:         map[string]any{},
			wantServer:   "192.168.1.100",
			wantPort:     mcp.ExportDefaultIperfPort,
			wantDuration: mcp.ExportDefaultIperfDurationSeconds,
			wantProtocol: "tcp",
			wantErr:      false,
		},
		{
			name:       "with custom port",
			serverAddr: "10.0.0.1",
			args: map[string]any{
				"port": float64(5202),
			},
			wantServer:   "10.0.0.1",
			wantPort:     5202,
			wantDuration: mcp.ExportDefaultIperfDurationSeconds,
			wantProtocol: "tcp",
			wantErr:      false,
		},
		{
			name:       "with custom duration",
			serverAddr: "localhost",
			args: map[string]any{
				"duration": float64(30),
			},
			wantServer:   "localhost",
			wantPort:     mcp.ExportDefaultIperfPort,
			wantDuration: 30,
			wantProtocol: "tcp",
			wantErr:      false,
		},
		{
			name:       "duration exceeds max",
			serverAddr: "localhost",
			args: map[string]any{
				"duration": float64(120),
			},
			wantServer:   "localhost",
			wantPort:     mcp.ExportDefaultIperfPort,
			wantDuration: mcp.ExportMaxIperfDurationSeconds,
			wantProtocol: "tcp",
			wantErr:      false,
		},
		{
			name:       "with udp protocol",
			serverAddr: "server.example.com",
			args: map[string]any{
				"protocol": "udp",
			},
			wantServer:   "server.example.com",
			wantPort:     mcp.ExportDefaultIperfPort,
			wantDuration: mcp.ExportDefaultIperfDurationSeconds,
			wantProtocol: "udp",
			wantErr:      false,
		},
		{
			name:       "with tcp protocol explicit",
			serverAddr: "server.example.com",
			args: map[string]any{
				"protocol": "tcp",
			},
			wantServer:   "server.example.com",
			wantProtocol: "tcp",
			wantErr:      false,
		},
		{
			name:       "invalid protocol",
			serverAddr: "localhost",
			args: map[string]any{
				"protocol": "sctp",
			},
			wantErr:        true,
			wantErrContain: "protocol",
		},
		{
			name:       "with download direction",
			serverAddr: "localhost",
			args: map[string]any{
				"direction": "download",
			},
			wantServer:    "localhost",
			wantDirection: "download",
			wantErr:       false,
		},
		{
			name:       "with upload direction",
			serverAddr: "localhost",
			args: map[string]any{
				"direction": "upload",
			},
			wantServer:    "localhost",
			wantDirection: "upload",
			wantErr:       false,
		},
		{
			name:       "with bidirectional direction",
			serverAddr: "localhost",
			args: map[string]any{
				"direction": "bidirectional",
			},
			wantServer:    "localhost",
			wantDirection: "bidirectional",
			wantErr:       false,
		},
		{
			name:       "invalid direction",
			serverAddr: "localhost",
			args: map[string]any{
				"direction": "sideways",
			},
			wantErr:        true,
			wantErrContain: "direction",
		},
		{
			name:       "full config",
			serverAddr: "192.168.1.1",
			args: map[string]any{
				"port":      float64(5210),
				"duration":  float64(20),
				"protocol":  "udp",
				"direction": "upload",
			},
			wantServer:    "192.168.1.1",
			wantPort:      5210,
			wantDuration:  20,
			wantProtocol:  "udp",
			wantDirection: "upload",
			wantErr:       false,
		},
		{
			name:       "zero port uses default",
			serverAddr: "localhost",
			args: map[string]any{
				"port": float64(0),
			},
			wantServer: "localhost",
			wantPort:   mcp.ExportDefaultIperfPort,
			wantErr:    false,
		},
		{
			name:       "negative port uses default",
			serverAddr: "localhost",
			args: map[string]any{
				"port": float64(-1),
			},
			wantServer: "localhost",
			wantPort:   mcp.ExportDefaultIperfPort,
			wantErr:    false,
		},
		{
			name:       "zero duration uses default",
			serverAddr: "localhost",
			args: map[string]any{
				"duration": float64(0),
			},
			wantServer:   "localhost",
			wantDuration: mcp.ExportDefaultIperfDurationSeconds,
			wantErr:      false,
		},
		{
			name:       "empty protocol uses default",
			serverAddr: "localhost",
			args: map[string]any{
				"protocol": "",
			},
			wantServer:   "localhost",
			wantProtocol: "tcp",
			wantErr:      false,
		},
		{
			name:       "empty direction is ok",
			serverAddr: "localhost",
			args: map[string]any{
				"direction": "",
			},
			wantServer:    "localhost",
			wantDirection: "",
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runParseIperfConfigCase(t, tt)
		})
	}
}

func runParseIperfConfigCase(
	t *testing.T,
	tt struct {
		name           string
		serverAddr     string
		args           map[string]any
		wantServer     string
		wantPort       int
		wantDuration   int
		wantProtocol   string
		wantDirection  string
		wantErr        bool
		wantErrContain string
	},
) {
	t.Helper()

	result, err := mcp.ExportParseIperfConfig(tt.serverAddr, tt.args)

	if tt.wantErr {
		if err == nil {
			t.Errorf("ExportParseIperfConfig() expected error, got nil")
			return
		}
		if tt.wantErrContain != "" {
			errStr := err.Error()
			if !containsString(errStr, tt.wantErrContain) {
				t.Errorf(
					"ExportParseIperfConfig() error = %q, want to contain %q",
					errStr,
					tt.wantErrContain,
				)
			}
		}
		return
	}

	if err != nil {
		t.Errorf("ExportParseIperfConfig() unexpected error: %v", err)
		return
	}

	config, ok := result.(*iperf.ClientConfig)
	if !ok {
		t.Fatalf("ExportParseIperfConfig() returned wrong type: %T", result)
	}

	if config.Server != tt.wantServer {
		t.Errorf(
			"ExportParseIperfConfig() server = %q, want %q",
			config.Server,
			tt.wantServer,
		)
	}

	if tt.wantPort != 0 && config.Port != tt.wantPort {
		t.Errorf("ExportParseIperfConfig() port = %d, want %d", config.Port, tt.wantPort)
	}

	if tt.wantDuration != 0 && config.Duration != tt.wantDuration {
		t.Errorf(
			"ExportParseIperfConfig() duration = %d, want %d",
			config.Duration,
			tt.wantDuration,
		)
	}

	if tt.wantProtocol != "" && config.Protocol != tt.wantProtocol {
		t.Errorf(
			"ExportParseIperfConfig() protocol = %q, want %q",
			config.Protocol,
			tt.wantProtocol,
		)
	}

	if config.Direction != tt.wantDirection {
		t.Errorf(
			"ExportParseIperfConfig() direction = %q, want %q",
			config.Direction,
			tt.wantDirection,
		)
	}
}

// containsString checks if s contains substr.
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTestingConstants(t *testing.T) {
	tests := []struct {
		name     string
		value    int
		expected int
	}{
		{
			name:     "SpeedtestTimeoutMinutes",
			value:    mcp.ExportSpeedtestTimeoutMinutes,
			expected: 2,
		},
		{
			name:     "DefaultIperfPort",
			value:    mcp.ExportDefaultIperfPort,
			expected: 5201,
		},
		{
			name:     "DefaultIperfDurationSeconds",
			value:    mcp.ExportDefaultIperfDurationSeconds,
			expected: 10,
		},
		{
			name:     "MaxIperfDurationSeconds",
			value:    mcp.ExportMaxIperfDurationSeconds,
			expected: 60,
		},
		{
			name:     "IperfTimeoutBufferSeconds",
			value:    mcp.ExportIperfTimeoutBufferSeconds,
			expected: 30,
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

func TestIperfConfigDefaultValues(t *testing.T) {
	// Verify that default port matches the standard iPerf3 port
	if mcp.ExportDefaultIperfPort != 5201 {
		t.Errorf("DefaultIperfPort = %d, want 5201 (standard iPerf3 port)", mcp.ExportDefaultIperfPort)
	}

	// Verify max duration is reasonable
	if mcp.ExportMaxIperfDurationSeconds < 30 {
		t.Errorf(
			"MaxIperfDurationSeconds = %d, should be at least 30 for meaningful tests",
			mcp.ExportMaxIperfDurationSeconds,
		)
	}

	// Verify buffer is reasonable
	if mcp.ExportIperfTimeoutBufferSeconds < 10 {
		t.Errorf(
			"IperfTimeoutBufferSeconds = %d, should be at least 10 for setup/teardown",
			mcp.ExportIperfTimeoutBufferSeconds,
		)
	}
}
