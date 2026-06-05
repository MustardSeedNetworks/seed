package api

import (
	"errors"
	"testing"
)

func TestValidateEngineScanConfig(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		req          EngineScanRequest
		acknowledged bool
		wantErr      bool
		wantIDSGate  bool // err is errIDsRiskUnacknowledged
	}{
		{
			name: "quick scan needs no acknowledgment",
			req:  EngineScanRequest{PortScanIntensity: "quick"},
		},
		{
			name: "comprehensive without acknowledgment is gated",
			req:  EngineScanRequest{PortScanIntensity: "comprehensive"},
			// acknowledged: false
			wantErr:     true,
			wantIDSGate: true,
		},
		{
			name:         "comprehensive with acknowledgment proceeds",
			req:          EngineScanRequest{PortScanIntensity: "comprehensive"},
			acknowledged: true,
		},
		{
			name:    "custom ports require custom intensity",
			req:     EngineScanRequest{PortScanIntensity: "quick", PortScanCustomPorts: []int{80, 443}},
			wantErr: true,
		},
		{
			name: "valid custom ports pass",
			req:  EngineScanRequest{PortScanIntensity: "custom", PortScanCustomPorts: []int{22, 80, 443}},
		},
		{
			name:    "out-of-range port rejected (zero)",
			req:     EngineScanRequest{PortScanIntensity: "custom", PortScanCustomPorts: []int{0}},
			wantErr: true,
		},
		{
			name:    "out-of-range port rejected (too high)",
			req:     EngineScanRequest{PortScanIntensity: "custom", PortScanCustomPorts: []int{70000}},
			wantErr: true,
		},
		{
			name: "too many custom ports rejected",
			req: EngineScanRequest{
				PortScanIntensity:   "custom",
				PortScanCustomPorts: makePorts(maxEngineCustomPorts + 1),
			},
			wantErr: true,
		},
		{
			name: "empty config is valid",
			req:  EngineScanRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateEngineScanConfig(tt.req, tt.acknowledged)
			if tt.wantErr != (err != nil) {
				t.Fatalf("validateEngineScanConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantIDSGate && !errors.Is(err, errIDsRiskUnacknowledged) {
				t.Fatalf("expected errIDsRiskUnacknowledged, got %v", err)
			}
			if !tt.wantIDSGate && errors.Is(err, errIDsRiskUnacknowledged) {
				t.Fatalf("unexpected IDS-risk gate error: %v", err)
			}
		})
	}
}

func TestDedupePorts(t *testing.T) {
	t.Parallel()

	got := dedupePorts([]int{80, 443, 80, 22, 443})
	want := []int{80, 443, 22}
	if len(got) != len(want) {
		t.Fatalf("dedupePorts length = %d, want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("dedupePorts[%d] = %d, want %d (order must be preserved)", i, got[i], want[i])
		}
	}
	if dedupePorts(nil) != nil {
		t.Fatalf("dedupePorts(nil) should return nil")
	}
}

func makePorts(n int) []int {
	ports := make([]int, n)
	for i := range ports {
		ports[i] = (i % 65535) + 1
	}
	return ports
}
