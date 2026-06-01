package traceroute

import (
	"net"
	"testing"
)

// Tests for internal functions using export_test.go

func TestExportAnalyzeHops(t *testing.T) {
	tests := []struct {
		name              string
		hops              []Hop
		wantTotalRTT      float64
		wantLostHops      int
		wantBottleneckCnt int
	}{
		{
			name:              "empty hops",
			hops:              []Hop{},
			wantTotalRTT:      0,
			wantLostHops:      0,
			wantBottleneckCnt: 0,
		},
		{
			name: "all responding hops",
			hops: []Hop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, RTTMs: 10.0, Lost: false},
				{Number: 3, RTTMs: 15.0, Lost: false},
			},
			wantTotalRTT:      30.0,
			wantLostHops:      0,
			wantBottleneckCnt: 0,
		},
		{
			name: "mixed responding and lost hops no bottleneck",
			hops: []Hop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, RTTMs: 8.0, Lost: false}, // 1.6x ratio < 2x threshold, 3ms increase < 50ms
			},
			wantTotalRTT:      13.0, // 5 + 8, skip lost hop
			wantLostHops:      1,
			wantBottleneckCnt: 0,
		},
		{
			name: "mixed responding and lost hops with bottleneck",
			hops: []Hop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, Lost: true},
				{Number: 3, RTTMs: 15.0, Lost: false}, // 3x ratio > 2x threshold (compares to last responding hop)
			},
			wantTotalRTT:      20.0, // 5 + 15, skip lost hop
			wantLostHops:      1,
			wantBottleneckCnt: 1, // Bottleneck detected due to 3x ratio
		},
		{
			name: "with bottleneck",
			hops: []Hop{
				{Number: 1, RTTMs: 5.0, Lost: false},
				{Number: 2, RTTMs: 60.0, Lost: false}, // 55ms increase
				{Number: 3, RTTMs: 65.0, Lost: false},
			},
			wantTotalRTT:      130.0,
			wantLostHops:      0,
			wantBottleneckCnt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bottlenecks := make([]Bottleneck, 0)
			totalRTT, lostHops := ExportAnalyzeHops(tt.hops, &bottlenecks)

			if totalRTT != tt.wantTotalRTT {
				t.Errorf("totalRTT = %f, want %f", totalRTT, tt.wantTotalRTT)
			}
			if lostHops != tt.wantLostHops {
				t.Errorf("lostHops = %d, want %d", lostHops, tt.wantLostHops)
			}
			if len(bottlenecks) != tt.wantBottleneckCnt {
				t.Errorf("bottleneck count = %d, want %d", len(bottlenecks), tt.wantBottleneckCnt)
			}
		})
	}
}

func TestExportDetectBottleneck(t *testing.T) {
	tests := []struct {
		name        string
		hopIndex    int
		hop         Hop
		previousRTT float64
		currentRTT  float64
		wantNil     bool
	}{
		{
			name:        "first hop (index 0) never bottleneck",
			hopIndex:    0,
			hop:         Hop{Number: 1, RTTMs: 100.0},
			previousRTT: 0,
			currentRTT:  100.0,
			wantNil:     true,
		},
		{
			name:        "zero previous RTT",
			hopIndex:    1,
			hop:         Hop{Number: 2, RTTMs: 100.0},
			previousRTT: 0,
			currentRTT:  100.0,
			wantNil:     true,
		},
		{
			name:        "zero current RTT",
			hopIndex:    1,
			hop:         Hop{Number: 2, RTTMs: 0},
			previousRTT: 10.0,
			currentRTT:  0,
			wantNil:     true,
		},
		{
			name:        "small increase not bottleneck",
			hopIndex:    1,
			hop:         Hop{Number: 2, RTTMs: 20.0},
			previousRTT: 10.0,
			currentRTT:  20.0,
			wantNil:     true, // 10ms increase < 50ms threshold, 2x ratio == threshold
		},
		{
			name:        "significant absolute increase",
			hopIndex:    1,
			hop:         Hop{Number: 2, RTTMs: 60.0},
			previousRTT: 5.0,
			currentRTT:  60.0, // 55ms increase > 50ms threshold
			wantNil:     false,
		},
		{
			name:        "significant ratio increase",
			hopIndex:    1,
			hop:         Hop{Number: 2, RTTMs: 25.0},
			previousRTT: 10.0,
			currentRTT:  25.0, // 2.5x ratio > 2x threshold
			wantNil:     false,
		},
		{
			name:        "bottleneck with address",
			hopIndex:    2,
			hop:         Hop{Number: 3, Address: net.ParseIP("192.168.1.1"), RTTMs: 70.0},
			previousRTT: 10.0,
			currentRTT:  70.0,
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExportDetectBottleneck(tt.hopIndex, tt.hop, tt.previousRTT, tt.currentRTT)

			if tt.wantNil {
				if result != nil {
					t.Errorf("expected nil bottleneck, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Error("expected bottleneck, got nil")
				return
			}

			if result.HopNumber != tt.hop.Number {
				t.Errorf("HopNumber = %d, want %d", result.HopNumber, tt.hop.Number)
			}

			expectedIncrease := tt.currentRTT - tt.previousRTT
			if result.RTTIncrease != expectedIncrease {
				t.Errorf("RTTIncrease = %f, want %f", result.RTTIncrease, expectedIncrease)
			}

			if result.Reason == "" {
				t.Error("Reason should not be empty")
			}
		})
	}
}

func TestExportCalculateScore(t *testing.T) {
	tests := []struct {
		name     string
		analysis *PathAnalysis
		want     int
	}{
		{
			name: "perfect score",
			analysis: &PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  10.0,
				Bottlenecks: []Bottleneck{},
			},
			want: 100,
		},
		{
			name: "50% packet loss",
			analysis: &PathAnalysis{
				PacketLoss:  50.0,
				AverageRTT:  10.0,
				Bottlenecks: []Bottleneck{},
			},
			want: 50,
		},
		{
			name: "100% packet loss",
			analysis: &PathAnalysis{
				PacketLoss:  100.0,
				AverageRTT:  0,
				Bottlenecks: []Bottleneck{},
			},
			want: 0,
		},
		{
			name: "high RTT penalty",
			analysis: &PathAnalysis{
				PacketLoss:  0,
				AverageRTT:  200.0, // 100ms above threshold
				Bottlenecks: []Bottleneck{},
			},
			want: 90, // 100 - (200-100)/10 = 90
		},
		{
			name: "bottleneck penalty",
			analysis: &PathAnalysis{
				PacketLoss: 0,
				AverageRTT: 10.0,
				Bottlenecks: []Bottleneck{
					{HopNumber: 1},
					{HopNumber: 2},
				},
			},
			want: 90, // 100 - 2*5 = 90
		},
		{
			name: "combined penalties",
			analysis: &PathAnalysis{
				PacketLoss:  20.0,
				AverageRTT:  150.0, // 50ms above threshold
				Bottlenecks: []Bottleneck{{HopNumber: 1}},
			},
			want: 70, // 100 - 20 - 5 - 5 = 70
		},
		{
			name: "score clamped at 0",
			analysis: &PathAnalysis{
				PacketLoss: 80.0,
				AverageRTT: 500.0, // 400ms above threshold
				Bottlenecks: []Bottleneck{
					{HopNumber: 1},
					{HopNumber: 2},
					{HopNumber: 3},
				},
			},
			want: 0, // Would be negative but clamped to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExportCalculateScore(tt.analysis)
			if got != tt.want {
				t.Errorf("calculateScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestBottleneckRTTRatioBoundary(t *testing.T) {
	// Test exactly at the 2x ratio boundary
	hop := Hop{Number: 2, RTTMs: 20.0}
	result := ExportDetectBottleneck(1, hop, 10.0, 20.0)

	// Exactly 2x ratio should NOT be a bottleneck (need > 2x)
	if result != nil {
		t.Error("exactly 2x ratio should not trigger bottleneck")
	}

	// Just above 2x should be a bottleneck
	hop = Hop{Number: 2, RTTMs: 20.1}
	result = ExportDetectBottleneck(1, hop, 10.0, 20.1)

	if result == nil {
		t.Error("just above 2x ratio should trigger bottleneck")
	}
}

func TestBottleneckAbsoluteThresholdBoundary(t *testing.T) {
	// Test exactly at the 50ms absolute threshold
	hop := Hop{Number: 2, RTTMs: 55.0}
	result := ExportDetectBottleneck(1, hop, 5.0, 55.0)

	// Exactly 50ms increase should NOT be a bottleneck (need > 50ms)
	// But the ratio check (55/5 = 11x) will trigger it
	if result == nil {
		t.Error("high ratio should still trigger bottleneck even at exact threshold")
	}

	// Test case where neither threshold is exceeded
	hop = Hop{Number: 2, RTTMs: 70.0}
	result = ExportDetectBottleneck(1, hop, 50.0, 70.0)

	// 20ms increase < 50ms threshold, 1.4x ratio < 2x threshold
	if result != nil {
		t.Error("neither threshold exceeded should not be bottleneck")
	}
}

func TestHopWithNilAddress(t *testing.T) {
	hop := Hop{Number: 2, RTTMs: 60.0, Address: nil}
	result := ExportDetectBottleneck(1, hop, 5.0, 60.0)

	if result == nil {
		t.Fatal("expected bottleneck")
	}

	if result.Address != "" {
		t.Errorf("Address = %q, want empty string for nil Address", result.Address)
	}
}
